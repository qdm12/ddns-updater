# DDNS Updater v3 — Interactive WebUI

A lightweight, universal Dynamic DNS updater for 50+ DNS providers, now with a fully interactive web UI for managing your DDNS entries directly in the browser.

This is a fork of [qdm12/ddns-updater](https://github.com/qdm12/ddns-updater). The original update logic is unchanged — v3 adds a complete management UI on top.

[![Latest release](https://img.shields.io/github/v/release/reptil1990/ddns-updater?label=Latest%20release)](https://github.com/reptil1990/ddns-updater/releases)
[![License](https://img.shields.io/github/license/reptil1990/ddns-updater)](LICENSE)

---

## What's new in v3.0

The v3.0 WebUI was developed by [@reptil1990](https://github.com/reptil1990).

- **Dashboard tab** — modernized table view of all DNS records with status indicators, current/previous IPs, and one-click force update
- **Configuration tab** — full CRUD management for DDNS entries directly in the browser; no more editing `config.json` by hand
- **Dynamic provider forms** — fields are generated automatically per provider; you only see what you actually need
- **Multi-auth support** — providers with multiple authentication methods (Cloudflare, OVH, Spdyn, Gandi) get a radio selector
- **Hot-reload** — configuration changes are picked up immediately; no application restart required
- **Sensitive field masking** — tokens, passwords and keys are masked in API responses to keep secrets safe
- **Dark / light theme** — auto-detects your system preference, fully responsive on mobile

The original update logic from upstream is **unchanged**. Only the management surface and persistence wrapper around `config.json` were rewritten.

### Screenshots

| Dashboard | Configuration |
| --- | --- |
| Modernized table view of all records with status dots, status timestamps, and the original column layout. | List of all configured entries with edit/delete actions and an "Add Entry" modal. |

### REST API

Everything the WebUI does is also available as a clean REST API on the same port:

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/api/status` | Current status of all records |
| `GET` | `/api/config` | All settings (sensitive fields masked) |
| `POST` | `/api/config` | Add a new entry |
| `PUT` | `/api/config/{index}` | Update an entry by array index |
| `DELETE` | `/api/config/{index}` | Delete an entry by array index |
| `GET` | `/api/providers` | Provider field definitions used by the WebUI form generator |

Sensitive fields (`password`, `token`, `key`, `secret`, `api_key`, `secret_api_key`, `access_key_id`, `access_secret`, `consumer_key`, `app_key`, `app_secret`, `client_key`, `user_service_key`, `credentials`, `personal_access_token`, `apikey`, `customer_number`) are returned as `"***"` in `GET` responses. On `PUT`, sending `""` or `"***"` for a sensitive field preserves the existing value.

---

## Features

- 🆕 Interactive WebUI with full CRUD config management (v3.0)
- 🆕 REST API for programmatic config and status access (v3.0)
- 🆕 Hot-reload — config changes apply without restart (v3.0)
- Available as zero-dependency binaries for **Linux (amd64, arm64, armv7), Windows, macOS (Intel & Apple Silicon)** — see the [releases page](https://github.com/reptil1990/ddns-updater/releases)
- Periodically updates A and AAAA records on 50+ DNS providers
- Persistent IP history in `data/updates.json`
- Optional notifications via [Shoutrrr](https://containrrr.dev/shoutrrr/v0.8/services/overview/)

### Supported providers

Aliyun · AllInkl · ChangeIP · Cloudflare · Custom · DD24 · DDNSS.de · deSEC · DigitalOcean · DNSOMatic · DNSPod · Domeneshop · DonDominio · Dreamhost · DuckDNS · DynDNS · Dynu · DynV6 · EasyDNS · FreeDNS · Gandi · GCP · GoDaddy · GoIP.de · He.net · Hetzner · Infomaniak · INWX · Ionos · Linode · Loopia · LuaDNS · Myaddr · Name.com · Namecheap · NameSilo · Netcup · Njalla · NoIP · Now-DNS · OpenDNS · OVH · Porkbun · Route53 · Selfhost.de · Servercow · Spdyn · Strato · Variomedia · Vultr · Zoneedit

Per-provider field documentation lives in [`docs/`](docs/).

---

## Quick start

### 1. Download

Grab the binary for your platform from the [releases page](https://github.com/reptil1990/ddns-updater/releases) and make it executable (Linux/macOS):

```sh
chmod +x ddns-updater-linux-amd64
```

### 2. First run (empty config)

Create a `data/` directory next to the binary and start it:

```sh
mkdir -p data
echo '{"settings":[]}' > data/config.json
LISTENING_ADDRESS=:8000 ./ddns-updater-linux-amd64
```

The application starts up with no records and an empty config.

### 3. Open the WebUI

Navigate to **<http://localhost:8000>** in your browser.

- **Dashboard** tab — shows current record status (empty on first launch)
- **Configuration** tab — click **+ Add Entry** to create your first DDNS record

Pick a provider from the dropdown — the form fields appropriate for that provider appear automatically. Fill them in, save, and the record is active immediately. No restart needed.

### Docker

```sh
mkdir -p data && echo '{"settings":[]}' > data/config.json
docker run -d \
  --name ddns-updater \
  -p 8000:8000/tcp \
  -v "$(pwd)/data:/updater/data" \
  ghcr.io/reptil1990/ddns-updater:latest
```

> **Note:** A Docker image for this fork is built on every push to `master`. Until then, you can build the image locally with `docker build -t ddns-updater .` from the repo root.

---

## Configuration

You can manage entries entirely through the WebUI. If you prefer editing `data/config.json` directly, both work — the WebUI hot-reloads on file changes and direct edits are supported.

Minimal `config.json`:

```json
{
  "settings": [
    {
      "provider": "cloudflare",
      "domain": "sub.example.com",
      "zone_identifier": "abc123def456",
      "token": "your-api-token",
      "ttl": 1
    }
  ]
}
```

For provider-specific field reference, see [`docs/`](docs/) — each provider has its own markdown file with the exact fields it accepts.

### Environment variables

Selected variables. The full list is in upstream's documentation; the variables here are the ones most relevant to running v3.

| Variable | Default | Description |
| --- | --- | --- |
| `LISTENING_ADDRESS` | `:8000` | Address the WebUI listens on |
| `ROOT_URL` | `/` | URL prefix when running behind a reverse proxy |
| `PERIOD` | `5m` | How often to check for IP changes |
| `UPDATE_COOLDOWN_PERIOD` | `5m` | Minimum time between updates per record |
| `CONFIG_FILEPATH` | `/updater/data/config.json` | Path to the JSON config file |
| `DATADIR` | `/updater/data` | Directory for `updates.json` and config |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warning`, or `error` |
| `SERVER_ENABLED` | `yes` | Set to `no` to disable the WebUI entirely |
| `SHOUTRRR_ADDRESSES` | | Comma-separated [Shoutrrr](https://containrrr.dev/shoutrrr/v0.8/services/overview/) notification URLs |
| `TZ` | | Timezone for log timestamps, e.g. `Europe/Berlin` |

---

## Architecture

```text
┌─────────────┐    HTTP/JSON    ┌──────────────────┐    file I/O    ┌──────────────┐
│   Browser   │ ──────────────▶ │  REST API layer  │ ─────────────▶ │ config.json  │
│  (SPA, JS)  │ ◀────────────── │  internal/server │ ◀───────────── │              │
└─────────────┘                 └────────┬─────────┘                └──────────────┘
                                         │ hot-reload
                                         ▼
                                ┌──────────────────┐
                                │  Update logic    │
                                │ (unchanged from  │
                                │   upstream)      │
                                └──────────────────┘
```

- **Frontend** (`internal/server/ui/`): vanilla JS SPA, no build step, embedded into the binary via Go's `embed` package
- **API** (`internal/server/api.go`): chi router with handlers for status, config CRUD, and provider definitions
- **Provider field definitions** (`internal/provider/fielddefs.go`): static map describing the fields each provider accepts; drives dynamic form generation
- **Update logic** (`internal/update/`, `internal/provider/providers/*`): unchanged from upstream

For each record, every period:

1. Fetch your public IP address
2. DNS resolve the record to its current IP
3. If they differ, call the provider API to update

For Cloudflare records with `proxied: true`, DNS resolution is skipped and the last known IP from `updates.json` is used instead.

---

## Building from source

Requires Go 1.25+.

```sh
git clone https://github.com/reptil1990/ddns-updater.git
cd ddns-updater
go build -o ddns-updater ./cmd/ddns-updater/
```

Cross-compile for other platforms:

```sh
GOOS=linux  GOARCH=arm64       go build -o ddns-updater-linux-arm64       ./cmd/ddns-updater/
GOOS=linux  GOARCH=arm   GOARM=7 go build -o ddns-updater-linux-armv7    ./cmd/ddns-updater/
GOOS=darwin GOARCH=arm64       go build -o ddns-updater-darwin-arm64     ./cmd/ddns-updater/
```

CI builds and publishes binaries for all platforms automatically on every push to `master`.

---

## Development

Run the test suite:

```sh
go test ./...
```

Run the linter (uses the same `.golangci.yml` as upstream):

```sh
golangci-lint run --timeout=10m
```

Project layout:

| Path | Purpose |
| --- | --- |
| `cmd/ddns-updater/` | Application entry point |
| `internal/server/api.go` | REST API handlers (v3) |
| `internal/server/ui/` | Embedded SPA (HTML, CSS, JS) (v3) |
| `internal/provider/fielddefs.go` | Provider form field definitions (v3) |
| `internal/provider/providers/` | Per-provider update implementations (upstream) |
| `internal/update/` | Update scheduler (upstream) |
| `internal/params/json.go` | Config parsing (upstream + `ParseProviders` export for hot-reload) |
| `docs/` | Per-provider configuration reference |

---

## Credits

- **Original ddns-updater** by [Quentin McGaw (qdm12)](https://github.com/qdm12) — the entire update engine, all 50+ provider integrations, and the project foundation
- **Original WebUI rework** by [Gottfried Mayer (fuse314)](https://github.com/fuse314)
- **v3.0 Interactive WebUI** by [@reptil1990](https://github.com/reptil1990)

---

## License

[MIT](LICENSE) — same as upstream.
