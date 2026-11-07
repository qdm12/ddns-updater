# Explicit interface selection for private DDNS addresses

Date: 2026-10-04 (revised after review round 1)
Repository: `D:\Hobbies\mylabb\ddns-updater`
Inspected baseline: branch `private-ip`, commit `9daae4d` (historical); merged baseline `b186506` (upstream `7d6fd78` merged 2026-10-04; see the plan's reconciliation section; Go 1.26, golangci-lint v2). Before feature work, follow the plan's mandatory preparation to fetch and merge the upstream project's current default branch into `private-ip`, record the merged revision, and reconcile this spec against that source. The upstream project is `qdm12/ddns-updater` (`upstream`); its default branch is currently `master`, not `main` or the fork's `origin/master`.
Companion implementation plan: [2026-10-04-private-ip-interface.md](../plans/2026-10-04-private-ip-interface.md).

## Goal and scope

When private-IP fetching is enabled, obtain the address from one configured network interface. Never fall back to another interface when the configured interface is missing, down, or has no usable address.

When `privateip` is the **only** enabled fetcher, a failed discovery must also leave DNS records unchanged (zero provider writes) and recover on a later update cycle. This guarantee does not hold for mixed fetcher lists; see [Mixed fetcher lists](#mixed-fetcher-lists).

The deployment target uses `enp2s0` and currently has DHCP IPv4 address `192.168.50.159`. Configure the interface name, not that IP address: DHCP address changes must work without restarting the updater.

Scope of the future implementation this spec describes: local source changes, tests, README, and local verification. **Historical note:** the original request was plan verification only; implementation has since been authorized and the scope above is the current implementation scope. **Deployment is excluded.** The deployment checklist at the end is an inactive reference kept for a possible later, separately authorized request.

## Evidence and limits

- Pre-implementation state (historical, baseline `9daae4d`; since replaced): the fetcher called `net.Interfaces()`, skips down and loopback interfaces, and immediately returns the first RFC1918 address (old `privateip.go:63-85`). Docker bridges qualify.
- Historical server logs show attempts to update `sebk.top` and `*.sebk.top` to `10.128.11.1`, which belongs to a Docker bridge. Some attempts failed while resolving the Cloudflare API hostname. The inspected logs do not prove that those writes succeeded or that every client outage had this cause.
- During inspection, Ethernet was up, the names resolved to `192.168.50.159`, and the updater was running in the host network namespace. Ethernet precedes the bridges in the current interface list. Ethernet temporarily lacking its address is a plausible explanation for the historical fallback, not a confirmed timeline.
- The server binary reports unknown build metadata. Do not assume its exact source revision has been established.
- Startup readiness has already been addressed on the server by a systemd drop-in that waits for `enp2s0` to be routable. This spec does not touch it.

## Configuration contract

Add environment variable `PRIVATEIP_INTERFACE`, read with the existing configuration reader.

| Setting | Required behavior |
| --- | --- |
| `PUBLICIP_FETCHERS=privateip` | Enable private fetching; require a nonempty `PRIVATEIP_INTERFACE`. |
| `PRIVATEIP_INTERFACE=enp2s0` | Select that exact interface name, case sensitive. |
| Interface name case | Preserved. The reader lowercases values by default, so the read must pass `reader.ForceLowercase(false)` (same idiom as `internal/config/shoutrrr.go:53`). |
| Interface name surrounding whitespace | Trimmed by the existing reader; add no extra trimming code. Whitespace-only is missing. |
| Private fetching enabled, interface omitted | Fail configuration validation with a message naming `PRIVATEIP_INTERFACE`; do not start fetching or write DNS records. |
| Private fetching disabled | No interface setting is required. |
| `PUBLICIP_FETCHERS` omitted | Preserve the public HTTP/DNS default; private fetching is disabled. |
| `PUBLICIP_FETCHERS=all` | Enable HTTP and DNS only, restoring the documented upstream meaning. Private fetching must be explicit. |
| Explicit mixed list containing `privateip` | Allowed. Require an interface. Existing fetcher cycling is preserved; see below. |

Compatibility decision: before this change the fork included private fetching in `all`, while README and the default Docker configuration describe/use `all` for public fetching. Making private mode explicit prevents the new interface requirement from breaking those defaults. Document that users previously relying on `all` for private fetching must switch to `privateip` and supply an interface.

Defaults (pre-implementation state, now aligned): `PubIP.setDefaults` defaulted private fetching to enabled while `PubIP.read` defaulted it to disabled. Both are now disabled.

Pass the interface name through every settings layer:

1. `config.PubIP.PrivateIPInterface string`.
2. `publicip.PrivateIPSettings.Interface string`.
3. `privateip.Settings.Interface string`.

Update the live constructor path in `cmd/ddns-updater/main.go` and `publicip.NewFetcher`; these are where the value would otherwise be dropped (`main.go:191-196`, `publicip.go:64-69`). Also carry the field in the exported helper `config.PubIP.ToPrivateIPSettings()`. The helper has no callers today; it is kept and updated rather than deleted because removing exported API is outside this change.

Startup summary: keep the `Private IP enabled` line; add a `Private IP interface: <name>` line only when private fetching is enabled. Do not add secrets or provider credentials to logs.

### Mixed fetcher lists

A list such as `http,privateip` or `dns,privateip` is valid configuration and is not rejected. Its behavior is the existing cycling behavior and must be documented, not changed:

- The top-level fetcher advances to the next sub-fetcher on every call (`pkg/publicip/subfetcher.go`).
- The updater retries a failed fetch up to three times in the same cycle (`internal/update/getip.go`), and each retry lands on the next sub-fetcher.
- Consequently, successive cycles alternate between public and private sources, and a failed private discovery is retried against a public fetcher. **A public address may be published, including while the private interface is failing.**
- Mixed lists are therefore not suitable for LAN-only publication. LAN-only publication requires `PUBLICIP_FETCHERS=privateip` alone.

Within a mixed list the private fetcher itself still never selects another interface; only the top-level cycling moves to another source.

## Address selection contract

Use the existing `InterfaceRetriever` dependency and Go standard library; add no dependencies or new discovery framework. Leave `pkg/publicip/privateip/options.go` and its test untouched (unused today, not extended).

For `IP4(ctx)` and `IP(ctx)`:

1. Retrieve the current interface list on each call. Find the interface whose name equals `Settings.Interface`.
2. Do not query addresses on unrelated interfaces. Docker, VPN, and other interfaces must not influence the selection, even when they appear earlier in the list or produce errors.
3. If the target is missing, administratively down (`net.FlagUp` absent), or loopback, return an invalid `netip.Addr` and an error containing the configured name.
4. Retrieve the target's current addresses. On retrieval failure, return an invalid address and the wrapped error with interface context.
5. Accept both `*net.IPNet` and `*net.IPAddr` entries; skip any other `net.Addr` type, a nil entry, and a nil or incorrect-length `net.IP`, without failing. Convert with `netip.AddrFromSlice` then `Unmap()`; a candidate is an address for which `Is4() && IsPrivate()` holds. That is exactly `10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`, and it excludes invalid, unspecified, loopback, multicast, public, link-local, and all IPv6 addresses. Delete the existing per-call CIDR helper `isPrivateIP`.
6. Return the address if exactly one distinct candidate exists. Duplicate representations of the same address count once.
7. With zero candidates, return an error. With multiple distinct private IPv4 candidates, return an ambiguity error rather than choosing by address order. This includes a secondary address or a brief overlap of two leases; the record is simply skipped until one address remains.

Do not cache the interface's addresses or index. Interfaces may be recreated and DHCP may replace addresses during process lifetime. Do not return the previous address as a substitute when current discovery fails.

This fix targets IPv4. Preserve the `IP6` method for interface compatibility, but return an explicit unsupported error and an invalid address without consulting any interface.

Errors: semantic failures (disabled, interface not set, not found, down, loopback, no candidate, ambiguous, IPv6 unsupported) are package-level sentinel errors wrapped with `%w` and interface context, so callers and tests use `errors.Is`; the repository linter (`err113`) requires this. Retrieval failures from the interface list or address lookup wrap the original error with interface context and get no new sentinel. The constructor rejects an empty or whitespace-only interface name with the not-set sentinel, even though trimming of configuration is owned by the reader, and does not require the physical interface to exist at construction time.

Errors are returned to the existing updater logger. No new notification or logging subsystem is needed.

## Failure and recovery contract

An empty interface setting is a static configuration error. An interface that disappears, goes down, or lacks a usable address is a runtime discovery failure, not a reason to terminate the application.

Keep the existing update loop and retries. The following holds **when `privateip` is the only enabled fetcher**:

- A failed discovery yields an invalid address on all three immediate retries of that cycle, the affected record updates are skipped, and the updater's `Update` method is called zero times.
- Each record's `History` is unchanged. No History changes are appended or persisted (`db.Update` may persist `Status`/`Message`/`Time`).
- A later cycle discovers the current address afresh and resumes the existing update decision flow. If it differs from DNS and the cooldown has elapsed, the record is updated to the new address.

Not covered by "unchanged": on the first cycle after start, a record whose status is still unset is marked failed by the existing code (`internal/update/service.go:334-342` on the merged baseline). Its `Status`, `Message` and `Time` change and `db.Update` is called. Only `History` is guaranteed unchanged.

### Known limitation: status stays failed after recovery

If that first-cycle failure is followed by a successful discovery whose address already matches DNS, no update is needed and the existing code never resets the status (`service.go:334`). The record keeps status `failure` with message "public IP address not found" until its address next changes, so the web UI shows a failure and the health check (`internal/health/check.go:41`) reports unhealthy.

Decision: address safety and functional recovery are in scope; repairing the health/UI status is **deferred**. Do not modify `internal/update/service.go` for this limitation. A test records the current behavior so a later fix is a visible, deliberate change.

The current server uses `PERIOD=10m`; recovery may therefore wait until the next cycle, plus existing cooldown rules and DNS caching. Do not add event watchers, a new retry scheduler, forced updates on startup, or a shorter production polling period in this change.

## File map

All paths below are relative to the repository root.

| File | Responsibility |
| --- | --- |
| `pkg/publicip/privateip/privateip.go` | Add interface setting and sentinel errors; validate construction; select only the target's current private IPv4. |
| `pkg/publicip/privateip/privateip_test.go` | Replace exact-string assertions with `errors.Is`; add selection/failure/recovery tests on a name-keyed fake. |
| `pkg/publicip/settings.go` | Carry the interface through public fetcher settings. |
| `pkg/publicip/publicip.go` | Pass the interface into the private fetcher constructor. |
| `pkg/publicip/publicip_test.go` (new) | Verify constructor propagation (read-only probe) and pin the cycling behavior of mixed lists, without network requests. |
| `internal/config/pubip.go` | Read (case-preserving) and validate the setting, summarize it, propagate it, align defaults, fix `all`. |
| `internal/config/pubip_test.go` (new) | Exercise reader/defaults/validation/migration/mixed-list cases. |
| `internal/config/settings_test.go` | Update the exact settings-summary expectation: private disabled by default. |
| `cmd/ddns-updater/main.go` | Carry the configured interface along the live startup path. |
| `internal/update/service_test.go` (new) | Verify discovery errors skip updater writes, recovery updates, and the documented status limitation. |
| `README.md` | Document private mode, interface requirement, migration, mixed-list behavior, failure behavior, and IPv4 scope. |

Not modified by the feature diff relative to the merged baseline: `internal/update/service.go`, `pkg/publicip/subfetcher.go`, `pkg/publicip/privateip/options.go`, `options_test.go`. Upstream merge changes are allowed. Recheck the documented status limitation after merging; if upstream fixed it, update the spec and regression expectations to the corrected behavior rather than preserving the old bug. Reconcile toolchain, CI commands and source paths against the merged checkout before implementation.

## Acceptance tests

Use deterministic fakes; unit tests must not contact any provider or public-IP service, change the host's interfaces, require SSH, or use credentials. All selection tests are fake-based. The one exception is the constructor propagation test, a read-only constructor/runtime probe: it reads the host's real interface list through `net.Interfaces` to look up a name that cannot exist.

| Case | Assertion |
| --- | --- |
| Docker bridge before Ethernet | With `br-test=10.128.11.1` and `enp2s0=192.168.50.159`, choosing `enp2s0` returns `192.168.50.159` for `IP` and `IP4`. |
| Interface list reordered | Same result regardless of relative positions. |
| Target missing, Docker present | Invalid address + not-found error containing the name; never `10.128.11.1`. |
| Target down, Docker present | Invalid address + down error. |
| Target loopback | Invalid address + error even if a private alias is present. |
| Target has only IPv6/link-local/public addresses | Invalid address + no-candidate error. |
| Target has one private IPv4 plus public/IPv6 addresses | Return only the private IPv4. |
| Two distinct private IPv4 addresses | Invalid address + ambiguity error, in both address orders. |
| Duplicate / mapped / `*net.IPAddr` representation | Deduplicate and normalize; one address is selected. |
| Case-sensitive name | Configured `Enp2s0` does not match interface `enp2s0`: not-found error and no `Addrs` call. |
| Malformed address entries | A nil `net.Addr`, an unsupported `net.Addr` type, a nil `net.IP`, and an incorrect-length `net.IP` (3 or 5 bytes) are skipped; alone they give the no-candidate error, beside one private IPv4 that address is returned. |
| DHCP changes without process restart | Same fetcher instance returns `.159`, then `.160` after fake address replacement. |
| Interface disappears and is recreated | Same fetcher reports failure, then succeeds for the new index with the configured name. |
| Interface list/address retrieval fails | Invalid address + wrapped error; unrelated interfaces' addresses are never queried. |
| Direct private fetcher construction without interface | Constructor rejects an empty and a whitespace-only name with the not-set sentinel. |
| Private IPv6 request | Invalid address + unsupported sentinel; retriever not called. |
| Config private mode with/without interface | Success with a mixed-case name preserved exactly; validation error naming `PRIVATEIP_INTERFACE` without it or with whitespace only. |
| Config public defaults and `all` | HTTP/DNS enabled, private disabled, no interface requirement. |
| Config mixed list | `http,privateip` with an interface validates. |
| Config helper | `ToPrivateIPSettings()` carries the interface. |
| Constructor propagation | `publicip.NewFetcher` with private enabled and no interface is rejected; with an impossible interface name construction succeeds, and `IP4` returns an invalid address and an error containing that name. |
| Mixed-list cycling | With fake public and failing private sub-fetchers, consecutive calls alternate, so the call after a private failure returns the public address. |
| Failed discovery in updater service | For `ipv4 or ipv6` and `ipv4` records: fetcher called three times, updater spy receives zero calls, `History` unchanged, status is `failure`. |
| Recovery in updater service | A later valid discovery `192.168.50.160` differing from stubbed DNS `192.168.50.159`, beyond cooldown, calls the updater once with `.160`. |
| Recovery when DNS already matches | A later valid discovery equal to stubbed DNS: zero updater calls, `History` unchanged, status still `failure` (known limitation). |
| IPv6 record with private fetcher | Unsupported error on all three tries; zero updater calls. |

## Verification and build

Go is not installed on the implementation workstation (neither Windows nor WSL). Docker is available inside WSL (`Ubuntu-24.04`). Verification therefore uses the repository's own Docker targets, which are also what CI runs (`.github/workflows/build.yml`, `Dockerfile`): `lint`, `mocks`, and `test` (`go test -race ./...`). Targeted package tests run with `go test` inside the `base` stage image. If a Go 1.26 toolchain is present, the equivalent direct commands are acceptable.

Build a Linux amd64 binary with the Dockerfile `build` stage, which is the repository's build convention (`CGO_ENABLED=0`, `-trimpath`, `-ldflags` setting `main.version`, `main.date`, `main.commit`). Reconcile these conventions against the merged checkout. Build metadata must be truthful: use the current `HEAD` (the merged HEAD if feature changes remain uncommitted, or the latest feature commit if committed), and when the build-context sources (`cmd`, `internal`, `pkg`, `go.mod`, `go.sum`, `Dockerfile`) differ from `HEAD`, append a dirty marker and a hash covering all staged and unstaged tracked changes relative to `HEAD` plus the content of untracked source files. Documentation-only changes do not make the source dirty. Record the artifact SHA-256 and keep a manifest (changed-file list and the hash) sufficient to identify the snapshot. Never create a commit merely to obtain clean metadata.

Report failures caused by existing external integration tests separately, rather than concealing them or treating them as a reason to broaden this fix.

## Deployment checklist (INACTIVE REFERENCE — not part of this implementation)

This section is historical preparation only. It is excluded from the implementation plan and must not be executed, verified, or reported on as part of it. It applies only if the user later gives a separate, explicit deployment instruction.

Target connection: `prime@proxy50.rt3.io`, SSH port `32719`. Obtain credentials from the user or the session's authorized credential mechanism; do not store them in this repository or documentation.

- Reinspect the current service, executable, ownership, launcher, interface name, and data paths before changing the server.
- Build and stage a verified Linux amd64 executable. Record its source revision and checksum; make a recoverable copy of the existing executable and service configuration with ownership/modes.
- Add a separate systemd drop-in, for example `/etc/systemd/system/ddns-updater.service.d/30-private-ip-interface.conf`, with `[Service]` and `Environment=PRIVATEIP_INTERFACE=enp2s0`.
- Keep `PUBLICIP_FETCHERS=privateip`, `PERIOD=10m`, the existing data directory/configuration/provider tokens, and `20-network-online.conf` in place.
- Stop the updater, replace the executable using the staged file, reload systemd, and start it. Do not restart the network manager or reboot to deploy this fix.
- Verify the child executable is running, the HTTP listener on port 8000 is available, the interface setting appears in the startup summary, and the chosen IPv4 is the current address on `enp2s0`.
- The launcher currently backgrounds the executable and keeps sleeping. `systemctl is-active` alone is insufficient: the wrapper can remain alive after the executable exits. Verify the child process and logs. Launcher lifecycle repair is a separate scope.
- Observe a normal update cycle, or use an existing supported force-update control if authorized. Verify apex and wildcard DNS records against an authoritative source and the client/LAN resolver; account for TTL and caches. A record already correct need not trigger a provider write.
- If startup or functional validation fails, restore the prior executable and added interface drop-in state, reload systemd, restart, and verify recovery. Preserve the network readiness drop-in.
- Do not deliberately unplug the server, change its IP, remove interfaces, or publish a Docker address to test production behavior. Exercise those failure cases with mocks or an isolated environment.

## Handoff instructions for the implementing agent

Read this document and the companion plan before editing. Use these installed skills as applicable:

- `superpowers:executing-plans`: execute the small, sequential plan directly. Subagents are unnecessary for this change unless the user requests them.
- `superpowers:test-driven-development`: write the deterministic regression tests before changing behavior.
- `ponytail:ponytail`: use the current abstractions and standard library, with the smallest fix covering all callers.
- `superpowers:verification-before-completion`: show passing applicable checks and build evidence before claiming success.
- `superpowers:requesting-code-review`: obtain a focused review of configuration propagation, failure behavior, and compatibility when required by the execution workflow.

Implement and verify locally; deliver the diff, test results, and artifact location/checksum/metadata if built. Do not deploy, connect to the server, or modify the router.
