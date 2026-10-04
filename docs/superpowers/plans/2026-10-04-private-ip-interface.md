# Explicit Private-IP Interface Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans or superpowers:subagent-driven-development to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking. For this small sequential change, use executing-plans unless the user requests delegation.

**Goal:** Read the private IPv4 address only from a configured interface, and skip DNS writes safely while that interface has no usable address when `privateip` is the only fetcher.

**Architecture:** Extend the existing settings chain with an interface name and filter the existing interface retriever by that exact name. Keep discovery fresh on every call and reuse the update loop's existing invalid-address skip. Make private fetching explicit so public defaults need no new configuration.

**Tech Stack:** Go 1.26 (merged `go.mod`/Dockerfile) standard library, existing `gosettings` reader, `testify`, the repository Dockerfile targets for lint/mocks/test/build.

**Spec:** [Explicit interface selection for private DDNS addresses](../specs/2026-10-04-private-ip-interface.md). Read it in full; its behavior, file map and acceptance matrix are normative. Each acceptance row is one named test case.

## Global Constraints

- New setting `PRIVATEIP_INTERFACE`, read with `reader.ForceLowercase(false)`; case sensitive; the reader already trims, add no trimming code in config.
- The private fetcher never falls back to another interface.
- Zero updater writes on private discovery failure is guaranteed **only when `privateip` is the only fetcher**. Mixed lists stay valid, keep cycling, and may publish a public address; do not reject them.
- `IP`/`IP4` return one distinct RFC1918 IPv4 (`Is4() && IsPrivate()` after `AddrFromSlice` and `Unmap()`); otherwise an invalid address and an error. `IP6` is unsupported.
- `all` and the default mean HTTP and DNS only; private mode requires explicit `privateip`.
- No new dependencies, polling changes, or watchers.
- Do not modify `internal/update/service.go`, `pkg/publicip/subfetcher.go`, `pkg/publicip/privateip/options.go`, or `options_test.go`.
- Keep the exported `PubIP.ToPrivateIPSettings()` and carry the new field in it.
- **Deployment is excluded from this plan**: no access or changes to the deployment server, router or systemd. Git remote access for the preparation below is allowed. The spec's deployment checklist is an inactive reference.
- Never create a commit just to get clean build metadata; commit only if the user's implementation request asks for commits.
- Before feature work, merge the upstream project's current default branch into `private-ip`. The required merge commit is authorized by this synchronization step; the conditional rule above applies to feature commits.
- Lint rules that shape the code (merged `.golangci.yml`, golangci-lint v2): `err113` (sentinel errors; tests exempt), `paralleltest` (`t.Parallel()` in every test and subtest; no `testCase := testCase` copy, Go 1.22+ semantics and `copyloopvar` forbid it), `thelper`, `prealloc`, `gofumpt`, `godot`, `lll` (120 columns).

## Prerequisites and commands

Go is absent on Windows and in WSL; Docker runs inside WSL `Ubuntu-24.04` (docker is `/snap/bin/docker`, on PATH only in a login shell, hence `bash -lc`). Run from Git Bash and start every command with these two lines (shell state does not persist between tool calls). The Windows checkout has `core.autocrlf=true`, so working files are CRLF and `gofumpt`/`gci`/`mockgen` checks fail on them; run the Docker targets on an LF snapshot of the working tree (see "Merged-baseline reconciliation"):

```bash
export MSYS_NO_PATHCONV=1
PREP=/d/Hobbies/mylabb/ddns-updater-prep
"$PREP/mkctx.sh"   # LF snapshot of the working tree -> /d/Hobbies/mylabb/ddns-updater-ctx (rerun after every source edit)
dkc() { "$PREP/dk.sh" ddns-updater-ctx "$@"; }   # docker in WSL, argv preserved (wsl.exe --exec bash -lc 'exec docker "$@"' _ args)
```

`dk.sh` is `wsl.exe -d Ubuntu-24.04 --cd /mnt/d/Hobbies/mylabb/<dir> --exec bash -lc 'exec docker "$@"' _ "$@"`; do not use `bash -lc "docker $*"`, which flattens and re-splits arguments. Right after the WSL VM starts the snap `dockerd` socket `/var/run/docker.sock` may be missing; `dk.sh` waits up to 60 s for it before running docker. No service is restarted or changed.

- Targeted tests (the `base` image copies the source, so rebuild after edits): `dkc build --target base -t ddns-base . && dkc run --rm ddns-base go test <packages>`
- CI checks, same targets as `.github/workflows/build.yml`: `dkc build --target lint .`, `dkc build --target mocks .`, `dkc build --target test -t test-container . && dkc run --rm test-container` (runs `go test -race ./...`).
- With a Go 1.26 toolchain on PATH, the direct `go` commands are equivalent.

These commands were derived from the Dockerfile and not executed during planning. If one fails for an environment reason, fix the invocation, not the source.

## Review Focus

- Upstream changes can invalidate the inspected baseline, file paths, toolchain and status assumptions: synchronize first, record the merged revision, then reconcile the plan and spec before implementation.
- A mixed-case interface name is lowercased by the default reader and then never matches: test that `Enp2S0` survives the read.
- The setting can be dropped between config and the real constructor: test through `publicip.NewFetcher` with an impossible name.
- The default Docker configuration (`PUBLICIP_FETCHERS=all`) must not start requiring an interface: test omitted fetchers and `all`.
- A user combining `privateip` with `http` or `dns` expects LAN-only publication and does not get it: pin the cycling in a test and document it in the README.
- After a failed first discovery, a record whose DNS is already correct stays in `failure` status: pin with a test and document as a known limitation.

## Review decisions (round 1)

| # | Decision |
| --- | --- |
| F1 | Accepted. `reader.ForceLowercase(false)`; rely on reader trimming; mixed-case and whitespace tests. |
| F2 | Mixed lists stay allowed. The zero-write guarantee is scoped to `privateip` alone; cycling is documented and pinned by a test. Reviewer's proposal to reject mixed lists was declined. |
| F3 | Stuck `failure` status after recovery is a documented, deferred limitation. `service.go` is not modified; a test records it. "Unchanged" means `History` only. |
| F4 | Accepted. Docker targets for verification; build metadata carries a dirty marker and source hash. |
| F5 | Accepted. Sentinel errors for semantic failures, `errors.Is`, parallel tests; exact-string assertions rewritten. Retrieval errors wrap the original cause. |
| F6 | Accepted. `*net.IPNet` and `*net.IPAddr`; malformed entries skipped; `isPrivateIP` deleted. |
| F7 | `ToPrivateIPSettings` is kept and updated. Reviewer's proposal to delete it was declined. Propagation is tested through `NewFetcher`. |
| F8 | Accepted. Defaults aligned to disabled; summary shows the interface only when enabled. |
| F9 | Accepted. In-memory database fake and `UpdaterInterface` spy; `ipv4 or ipv6` and `ipv4` table; three failed tries; IPv6 unsupported. |

---

## Merged-baseline reconciliation (executed 2026-10-04)

- Starting HEAD `9daae4d805b1d99e1ffc6ca57c0a392c6d37f63e` (backup branch `backup/private-ip-pre-upstream-sync`); fetched `qdm12/ddns-updater` default `master` `7d6fd785f78317fc8bc53e6b245a1d33e43f3ad2` over HTTPS (SSH host-key lookup failed; remotes unchanged); merge commit `b186506f02c0c40dc679663d87f1493eb85f22f0` (`merge --no-ff --no-commit`, no conflicts, 270 upstream files). Upstream is an ancestor of the merged HEAD. Only untracked `docs/superpowers/` existed, so it was left in place and checkpointed by an outside copy plus SHA-256 list (no stash created; pre-existing `stash@{0}` = `e756b0c...` untouched).
- Merged toolchain: Go 1.26 (`go.mod`, Dockerfile `GO_VERSION=1.26`, container `go1.26.8`), golangci-lint v2.11.4 (`err113`, `copyloopvar`, `thelper`, `prealloc`, `gofumpt`, `godot`, `lll`), Alpine 3.23. Test idioms: no `testCase := testCase`, `t.Context()` allowed.
- Upstream did not touch the private-IP code paths: file map, interfaces and constraints stand. Merged line references: `main.go:191-196`, `publicip.go:64-69`, README `PUBLICIP_FETCHERS` row 298, status limitation `service.go:334-342` (unchanged logic) and health `check.go:41`.
- The FAIL-status limitation is **still present** upstream (`record.Status != constants.UNSET` skips the status of recovered records); it stays deferred and pinned by test.
- Environment findings: the checkout is CRLF (`core.autocrlf=true`), which makes `gofumpt`/`gci` and the mocks stage fail; the Docker targets were run on an LF snapshot of the working tree. Merged baseline on the LF snapshot: `lint` fails with 11 issues, all in the fork's private-IP code (`privateip.go`, `options.go`, `publicip.go` formatting); `mocks` and `test` pass. After the feature, `lint` keeps 3 pre-existing issues in the do-not-modify `options.go` (`godot` x2, `gofumpt`).
- `VersionString()` (`internal/models/build.go`) returns the version alone whenever it is not `latest`; the commit is appended only for `latest` with a 7-character commit. A dirty marker in `COMMIT` therefore never appears in `version` output, so the artifact carries the marker in the `VERSION` build argument as well (`private-ip-<commit>-dirty-<hash>`); the commit and date are also embedded through `-ldflags`.
- Superseded: the task steps below still cite pre-merge line numbers and the original `dk`/build recipe text where noted; the merged line numbers above and the scripts in `ddns-updater-prep/` (`dk.sh`, `mkctx.sh`, `source-hash.sh`, `build-binary.sh`) are authoritative.

### Preparation: Merge the upstream project before Task 1

This is a mandatory first step during implementation. This planning request does not execute the merge. The main project's remote is `upstream` (`qdm12/ddns-updater`); `origin` is the fork (`primez/ddns-updater`). Both default branches are named `master`, not `main`. Read-only inspection on 2026-10-04 found upstream HEAD `7d6fd785f78317fc8bc53e6b245a1d33e43f3ad2` and fork HEAD `94e008dc195d80697c215f2d4e8bbc8fd1964b74`; fetch again at execution time instead of pinning these observations.

- [x] Confirm the active branch is `private-ip`, inspect `git status --short` and any merge/rebase already in progress, and record the starting full HEAD SHA. Preserve every local tracked/untracked change, including the currently untracked `docs/superpowers/` documents; copy the plan/spec outside the checkout so they remain readable while stashed. Create a named backup ref for the starting HEAD. If the checkout is dirty, save a named stash with `--include-untracked` or an equivalent recoverable checkpoint, record its SHA, and verify it before merging. Leave all pre-existing stash entries untouched; do not discard changes or overwrite untracked files.
- [x] Confirm `upstream` identifies `qdm12/ddns-updater` and recheck its default branch with `git ls-remote --symref upstream HEAD`, falling back to `git ls-remote --symref https://github.com/qdm12/ddns-updater.git HEAD` if SSH is unavailable. Fetch that branch explicitly: currently `git fetch --no-tags upstream refs/heads/master:refs/remotes/upstream/master`. This workstation's SSH host-key verification failed during planning; HTTPS read-only inspection succeeded. If SSH remains unavailable, use `git fetch --no-tags https://github.com/qdm12/ddns-updater.git refs/heads/master:refs/remotes/upstream/master`, preserving the configured remotes and host-key checks. Do not substitute the stale fork branch. If the default branch was renamed, use the advertised default branch and its corresponding tracking ref instead.
- [x] Record the fetched full upstream SHA, inspect its changes against the starting branch, then merge the fetched revision into the current `private-ip` branch with `git merge --no-ff --no-commit upstream/master` (or the verified renamed default ref). Resolve conflicts individually, preserving existing private-fetcher functionality and upstream fixes; do not use blanket ours/theirs selection, reset, rebase, or force-push. Inspect the merged toolchain and run compilation plus targeted baseline tests before concluding the merge; correct merge-caused failures and record unrelated existing failures separately. Stage only merge resolutions and finish the required merge with `git commit --no-edit`, or record that the upstream revision was already included (no merge commit needed).
- [x] Restore the preserved local changes with `git stash apply <recorded-checkpoint-SHA>` (or the equivalent checkpoint restore), keeping the checkpoint until all files are verified; do not use an unqualified `stash pop` or alter pre-existing stashes. Resolve restoration conflicts explicitly and confirm that the spec/plan and unrelated user changes remain intact. Record the post-merge HEAD SHA and prove inclusion with `git merge-base --is-ancestor <fetched-upstream-SHA> HEAD` (exit 0); `git diff --name-only --diff-filter=U` must be empty and no merge may remain in progress. Report the upstream, starting, and merged SHAs separately.
- [x] Re-read repository instructions, `go.mod`, Dockerfile/CI, linter configuration and the settings-to-fetcher-to-updater flow on the merged source. Update stale paths, signatures, line references, toolchain versions, lint rules/test idioms, test expectations and build commands in this plan and its spec. This includes the historical `Go 1.22`, `goerr113` and loop-variable-copy guidance above. The original `9daae4d` is historical evidence, not a required implementation HEAD. The do-not-modify constraints describe the feature diff relative to the merged baseline; they do not exclude upstream changes to those files. Recheck whether upstream already repaired the documented FAIL-status limitation; if so, record that and test the corrected behavior rather than reintroducing or asserting the bug. Obtain focused adversarial agreement on any material implementation changes revealed by the merge.
- [x] Before feature edits, run the applicable targeted tests/build and repository checks using the merged checkout's own toolchain and CI conventions. Record existing failures separately for comparison with final checks. Proceed only after conflicts are resolved, upstream inclusion and local-change preservation are verified, and the revised implementation plan matches the merged code. No push or deployment is part of this step.

### Task 1: Implement explicit private-interface selection and configuration

**Files:** Modify `pkg/publicip/privateip/privateip.go`, `pkg/publicip/privateip/privateip_test.go`, `pkg/publicip/settings.go`, `pkg/publicip/publicip.go`, `internal/config/pubip.go`, `internal/config/settings_test.go`, `cmd/ddns-updater/main.go`, `README.md`. Create `internal/config/pubip_test.go` and `pkg/publicip/publicip_test.go`.

**Interfaces:** Keep `privateip.New(settings Settings, retriever InterfaceRetriever) (*Fetcher, error)`, `publicip.NewFetcher(...)`, `(*PubIP).ToPrivateIPSettings() privateip.Settings` and the `IP`/`IP4`/`IP6` signatures unchanged. Add `Interface string` to `privateip.Settings` and `publicip.PrivateIPSettings`; add `PrivateIPInterface string` to `config.PubIP`. Add exported sentinels in `privateip`: `ErrDisabled`, `ErrInterfaceNotSet`, `ErrRetrieverNil`, `ErrInterfaceNotFound`, `ErrInterfaceDown`, `ErrInterfaceLoopback`, `ErrNoPrivateIPv4`, `ErrAmbiguousPrivateIPv4`, `ErrIPv6NotSupported`; and `config.ErrPrivateIPInterfaceNotSet`, whose message names `PRIVATEIP_INTERFACE`.

- [x] Confirm preparation is complete, the branch is still `private-ip` at the recorded merged HEAD, and the reconciled Docker/toolchain commands work. Use that merged baseline for feature-diff review and artifact metadata.
- [x] **Selection tests** in `privateip_test.go`: replace the function-field mock with a fake retriever that serves interfaces and addresses keyed by interface name and records which names had `Addrs` called. Rewrite the six exact-string assertions (pre-rewrite lines 107, 136, 170, 259, 281, 297 of the old test file) to `errors.Is`, and give every fixture an interface name. Add one table case per spec row from "Docker bridge before Ethernet" through "Interface list/address retrieval fails", each run against both `IP` and `IP4`, asserting:
  - success: `got == netip.MustParseAddr("192.168.50.159")` and `Addrs` called for `enp2s0` only;
  - failure: `!got.IsValid()`, `errors.Is(err, <sentinel or injected cause>)`, error text contains the configured name;
  - missing, down, loopback and case-mismatch (`Enp2s0` vs `enp2s0`) cases: zero `Addrs` calls;
  - malformed entries: nil `net.Addr`, an unsupported `net.Addr` type, nil `net.IP`, 3-byte and 5-byte `net.IP` are skipped (the existing `TestFetcher_InvalidIP` uses `255.255.255.255`, which is a valid address, so replace it with these);
  - duplicates: the same address as 16-byte `IPNet`, 4-byte `IPNet` and `IPAddr` yields one address; two distinct private addresses give `ErrAmbiguousPrivateIPv4` in both orders.
- [x] **Change-tracking test**: one fetcher instance over a mutated fake returns `.159`, then `.160`, then `ErrInterfaceNotFound` when only `br-test=10.128.11.1` remains, then `.160` again when `enp2s0` returns with a new index.
- [x] **Constructor and IPv6 tests**: `New` returns `ErrDisabled`, `ErrInterfaceNotSet` for `""` and for `"   "`, `ErrRetrieverNil`, and succeeds with a name that does not exist. `IP6` returns `ErrIPv6NotSupported` and an invalid address, and makes no retriever call.
- [x] **`pkg/publicip/publicip_test.go`** (in-package, no network requests):
  - propagation, a read-only constructor/runtime probe: `NewFetcher` with only private enabled and no interface returns `privateip.ErrInterfaceNotSet`; with the impossible name `ddns-test-absent0` construction succeeds, and `IP4` returns an invalid address and an error containing that name;
  - cycling: a `Fetcher` built from a fake public sub-fetcher (`203.0.113.7`) followed by a fake failing private one returns, on consecutive calls, private error, public address, private error, public address. Comment that this pins the mixed-list behavior and that `subfetcher.go` is unchanged.
- [x] **`internal/config/pubip_test.go`**: build a reader over `env.New(env.Settings{Environ: ...})` (always include one unrelated entry, because an empty `Environ` falls back to the process environment), then run `read`, `setDefaults`, `Validate`. Table cases:
  - fetchers omitted, and `all`: HTTP and DNS enabled, private disabled, valid, summary has no interface line;
  - `privateip` with `PRIVATEIP_INTERFACE="  Enp2S0  "`: valid, value is exactly `Enp2S0`, summary contains `Private IP interface: Enp2S0`, `ToPrivateIPSettings()` equals `{Enabled: true, Interface: "Enp2S0"}`;
  - `privateip` without the variable, and with a whitespace-only value: `errors.Is(err, ErrPrivateIPInterfaceNotSet)` and the text contains `PRIVATEIP_INTERFACE`;
  - `http,privateip` with an interface: valid, HTTP and private enabled, DNS disabled.
- [x] In `settings_test.go`, change the expected summary line to `Private IP enabled: no`.
- [x] Run `go test ./pkg/publicip/privateip ./pkg/publicip ./internal/config` in the base image and confirm the new tests fail for the intended reason (unknown `Interface` field, undefined sentinels) before implementing.
- [x] **Implement `privateip.go`**: declare the sentinels; `New` checks disabled, then `strings.TrimSpace(settings.Interface) == ""`, then nil retriever. `IP` lists interfaces on every call, matches `iface.Name == settings.Interface` before any `Addrs` call, rejects down then loopback, reads addresses of that interface only, converts `*net.IPNet`/`*net.IPAddr` with `netip.AddrFromSlice` + `Unmap()`, keeps `Is4() && IsPrivate()`, and returns the single distinct candidate. `IP4` delegates to `IP`; `IP6` returns `ErrIPv6NotSupported` without touching the retriever. Sentinels are wrapped with `%w` plus the interface name; list and address retrieval errors wrap the original error with the interface name. Delete `isPrivateIP`. No caching.
- [x] **Propagate**: add `Interface` to `publicip.PrivateIPSettings`; in `publicip.go:64-69` (merged baseline) pass `settings.privateIP.Interface` to `privateip.New`; in `main.go:191-196` set `Interface: config.PubIP.PrivateIPInterface`.
- [x] **Config in `pubip.go`**: add the field; in `read` set it with `r.String("PRIVATEIP_INTERFACE", reader.ForceLowercase(false))`; change the `setDefaults` private default to `false` (`setDefaults`, merged line 44); in `Validate` return `ErrPrivateIPInterfaceNotSet` when private is enabled and the interface is empty; in `toLinesNode` append `Private IP interface: <name>` only when enabled; carry the field in `ToPrivateIPSettings`; remove `*privateIP = true` from the `all` case of `getFetchers` (merged line ~340).
- [x] **README**: change the `PUBLICIP_FETCHERS` row (merged line 298) to list `http`, `dns`, `privateip` and state that `all` means `http` and `dns`; add a `PRIVATEIP_INTERFACE` row; add a "Private IP" subsection before `### Host firewall` covering: the `PUBLICIP_FETCHERS=privateip` + `PRIVATEIP_INTERFACE=enp2s0` example; no fallback to other interfaces; IPv4 RFC1918 only; more than one private address means none is chosen; address re-read every period; migration from the fork's old `all`; mixed lists alternate sources and retry on public fetchers, so a public address can be published even while the private interface fails, and LAN-only publication requires `privateip` alone; the known status limitation.
- [x] Rerun the targeted command plus `go build ./...` in the base image. Expected: all pass and `main.go` compiles. Check `gofmt -l cmd internal pkg` prints nothing. Review the diff for unnecessary abstractions and unrelated edits.
- [x] No feature commits requested: implementation, tests and README remain uncommitted.

### Task 2: Verify skip/recovery behavior and prepare the handoff

**Files:** Create `internal/update/service_test.go`. `internal/update/service.go` is not modified; if one of these tests fails, stop and report instead of editing it.

**Interfaces:** Consume the existing `update.PublicIPFetcher`, `update.UpdaterInterface`, `update.Database`, `update.LookupIPer`, `update.Logger` and `update.HealthchecksIOClient` contracts and `NewService`. Use small local fakes: a fetcher returning one configurable result and counting calls, an in-memory `Database`, an `UpdaterInterface` spy recording the addresses it is given, a resolver returning `192.168.50.159`, a no-op logger and healthchecks client, and a minimal `provider.Provider` whose `IPVersion()` is configurable and `Proxied()` is false. Call `Service.updateNecessary(ctx)` in-package; no timers, no network. The fetcher fake stands in for a `privateip`-only configuration.

- [x] Add `Test_Service_updateNecessary_privateIPFailureThenRecovery`, a table over record version (`ipv4 or ipv6`, `ipv4`) and recovered address (`192.168.50.160`, `192.168.50.159`). Each case starts from `records.New` (status unset) with history `[192.168.50.159]` and a clock beyond the cooldown:
  - cycle 1, fetcher fails: returned errors non-empty; fetcher called exactly 3 times; updater spy has zero calls; `History` equals the original; `Status` is `failure` and `Message` is `public IP address not found`;
  - cycle 2 with `.160`: fetcher called once; spy called exactly once with `.160`; no errors;
  - cycle 2 with `.159` (DNS already matches): spy has zero calls; `History` unchanged; `Status` still `failure`, with a comment that this is the documented deferred limitation and that the spec must change if status repair is implemented.
- [x] Add `Test_Service_updateNecessary_privateIPv6Unsupported`: an `ipv6` record with the fetcher's `IP6` returning `privateip.ErrIPv6NotSupported`; assert 3 calls, zero spy calls, `History` unchanged.
- [x] Run `go test ./internal/update` in the base image. These tests pin existing behavior and are expected to pass without a source change; confirm they guard it by temporarily inverting one expectation.
- [x] Run the three CI targets from "Prerequisites and commands". Fix lint findings in files this plan touched. Report failures in untouched packages or external integration tests separately.
- [x] Build the Linux amd64 binary with the Dockerfile `build` stage (repository convention: `CGO_ENABLED=0`, `-trimpath`, `-ldflags` setting `main.version`, `main.date`, `main.commit`):

  ```bash
  /d/Hobbies/mylabb/ddns-updater-prep/build-binary.sh /d/Hobbies/mylabb/ddns-updater-linux-amd64
  ```

  The script snapshots the sources with LF endings, computes `COMMIT=<short HEAD>-dirty-<first 12 hex of the source hash>` (clean if `cmd internal pkg go.mod go.sum Dockerfile` equal `HEAD`) and `VERSION=private-ip-<COMMIT>`, writes the manifest `../ddns-updater-build.manifest` (merged HEAD, hash, build args, context, file list), builds the Dockerfile `build` stage for `linux/amd64` with quoted `--build-arg VERSION CREATED COMMIT`, extracts `/tmp/gobuild/app` with `docker create`+`docker cp`, then prints the SHA-256 and `version`. The hash is `sha256(git diff HEAD -- SRC` followed by the `sha256sum` lines of the sorted untracked source files`)` (`source-hash.sh`). Docs-only changes do not require a rebuild.

  `git diff HEAD` covers staged and unstaged tracked changes; untracked source files are hashed by content; documentation is outside `$SRC`, so doc-only changes leave the metadata clean. Expected: the `version` output is `private-ip-<COMMIT>`. Record the build command, `COMMIT`, artifact path, SHA-256 and the manifest. The artifact and manifest sit beside the repository, not in it.
- [x] Review the diff against every acceptance row and constraint in the spec. Verify `main.go` and `publicip.go` both forward the field, no failure path returns a stale address or another interface's address, and the four do-not-modify files are unchanged.
- [x] Deliver the diff summary, check results, unrelated failures listed separately, and the artifact metadata. State that deployment was not performed and is not part of this plan. Stop.
