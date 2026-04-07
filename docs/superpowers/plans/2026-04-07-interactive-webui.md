# Interactive WebUI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Transform the static HTML table UI into an interactive SPA with Dashboard + Configuration tabs, backed by new REST API endpoints that read/write config.json.

**Architecture:** New REST API endpoints on the existing chi router handle config CRUD operations on config.json. A vanilla JS SPA replaces the Go-template HTML. Provider field definitions drive dynamic form generation. The existing update logic is untouched.

**Tech Stack:** Go (chi router, embed), Vanilla JS, CSS Custom Properties

**Spec:** `docs/superpowers/specs/2026-04-07-interactive-webui-design.md`

---

## File Structure

### New files

| File | Responsibility |
|------|---------------|
| `internal/provider/fielddefs.go` | Static map of provider field definitions for form generation |
| `internal/server/api.go` | REST API handlers: config CRUD, status, providers list |
| `internal/server/api_test.go` | Tests for API handlers |
| `internal/server/ui/static/app.js` | Client-side SPA logic (tabs, forms, modals, API calls) |

### Modified files

| File | Change |
|------|--------|
| `internal/server/handler.go` | Add API routes, accept configPath parameter |
| `internal/server/server.go` | Pass configPath to newHandler |
| `internal/server/interfaces.go` | Add StatusRecord interface for JSON status API |
| `internal/server/ui/index.html` | Replace Go template with SPA shell (no more `{{range}}`) |
| `internal/server/ui/static/styles.css` | Complete rewrite with modern card-based dark theme |
| `cmd/ddns-updater/main.go` | Pass config path to server.New |

---

## Task 1: Provider Field Definitions

**Files:**
- Create: `internal/provider/fielddefs.go`

This is a pure data file. No tests needed — it's a static map consumed by the API handler.

- [ ] **Step 1: Create the field definitions file**

Create `internal/provider/fielddefs.go` with all 50+ provider definitions:

```go
package provider

// FieldDefinition describes a single form field for a provider.
type FieldDefinition struct {
	Name        string   `json:"name"`
	Label       string   `json:"label"`
	Type        string   `json:"type"` // "text", "password", "number", "boolean", "select"
	Required    bool     `json:"required"`
	Placeholder string   `json:"placeholder,omitempty"`
	Help        string   `json:"help,omitempty"`
	Options     []string `json:"options,omitempty"`
}

// AuthGroup represents a set of fields for one authentication method.
type AuthGroup struct {
	Name   string            `json:"name"`
	Fields []FieldDefinition `json:"fields"`
}

// ProviderDefinition describes a provider's form fields for the WebUI.
type ProviderDefinition struct {
	Name       string            `json:"name"`
	URL        string            `json:"url"`
	Fields     []FieldDefinition `json:"fields"`
	AuthGroups []AuthGroup       `json:"auth_groups,omitempty"`
}

// ProviderDefinitions maps provider IDs to their form field definitions.
var ProviderDefinitions = map[string]ProviderDefinition{
	"aliyun": {
		Name: "Aliyun",
		URL:  "https://www.aliyun.com",
		Fields: []FieldDefinition{
			{Name: "access_key_id", Label: "Access Key ID", Type: "password", Required: true, Placeholder: "Your Aliyun access key ID"},
			{Name: "access_secret", Label: "Access Secret", Type: "password", Required: true, Placeholder: "Your Aliyun access secret"},
			{Name: "region", Label: "Region", Type: "text", Required: false, Placeholder: "e.g. cn-hangzhou"},
		},
	},
	"allinkl": {
		Name: "All-Inkl",
		URL:  "https://all-inkl.com",
		Fields: []FieldDefinition{
			{Name: "username", Label: "Username", Type: "text", Required: true, Placeholder: "dynXXXXXXX"},
			{Name: "password", Label: "Password", Type: "password", Required: true},
		},
	},
	"changeip": {
		Name: "ChangeIP",
		URL:  "https://changeip.com",
		Fields: []FieldDefinition{
			{Name: "username", Label: "Username", Type: "text", Required: true},
			{Name: "password", Label: "Password", Type: "password", Required: true},
		},
	},
	"cloudflare": {
		Name: "Cloudflare",
		URL:  "https://www.cloudflare.com",
		Fields: []FieldDefinition{
			{Name: "zone_identifier", Label: "Zone Identifier", Type: "text", Required: true, Placeholder: "e.g. abc123def456", Help: "Found in your Cloudflare dashboard under Overview"},
			{Name: "ttl", Label: "TTL", Type: "number", Required: true, Placeholder: "1", Help: "Set to 1 for automatic"},
			{Name: "proxied", Label: "Proxied", Type: "boolean", Required: false, Help: "Enable Cloudflare proxy"},
		},
		AuthGroups: []AuthGroup{
			{
				Name: "API Token (recommended)",
				Fields: []FieldDefinition{
					{Name: "token", Label: "API Token", Type: "password", Required: true, Placeholder: "Your Cloudflare API token"},
				},
			},
			{
				Name: "Global API Key",
				Fields: []FieldDefinition{
					{Name: "email", Label: "Email", Type: "text", Required: true},
					{Name: "key", Label: "Global API Key", Type: "password", Required: true},
				},
			},
			{
				Name: "User Service Key",
				Fields: []FieldDefinition{
					{Name: "user_service_key", Label: "User Service Key", Type: "password", Required: true},
				},
			},
		},
	},
	"custom": {
		Name: "Custom",
		URL:  "",
		Fields: []FieldDefinition{
			{Name: "url", Label: "URL", Type: "text", Required: true, Placeholder: "https://example.com/update?ip=%s", Help: "URL template for updating DNS"},
			{Name: "ipv4key", Label: "IPv4 Key", Type: "text", Required: false, Placeholder: "Query parameter name for IPv4"},
			{Name: "ipv6key", Label: "IPv6 Key", Type: "text", Required: false, Placeholder: "Query parameter name for IPv6"},
			{Name: "success_regex", Label: "Success Regex", Type: "text", Required: true, Placeholder: "e.g. ^(ok|good)", Help: "Regex pattern to match a successful response body"},
		},
	},
	"dd24": {
		Name: "DD24",
		URL:  "https://www.domaindiscount24.com",
		Fields: []FieldDefinition{
			{Name: "password", Label: "Password", Type: "password", Required: true},
		},
	},
	"ddnss": {
		Name: "DDNSS.de",
		URL:  "https://ddnss.de",
		Fields: []FieldDefinition{
			{Name: "username", Label: "Username", Type: "text", Required: true},
			{Name: "password", Label: "Password", Type: "password", Required: true},
			{Name: "dual_stack", Label: "Dual Stack", Type: "boolean", Required: false, Help: "Update both IPv4 and IPv6 simultaneously"},
		},
	},
	"desec": {
		Name: "deSEC",
		URL:  "https://desec.io",
		Fields: []FieldDefinition{
			{Name: "token", Label: "Token", Type: "password", Required: true},
		},
	},
	"digitalocean": {
		Name: "DigitalOcean",
		URL:  "https://www.digitalocean.com",
		Fields: []FieldDefinition{
			{Name: "token", Label: "API Token", Type: "password", Required: true},
		},
	},
	"dnsomatic": {
		Name: "DNS-O-Matic",
		URL:  "https://www.dnsomatic.com",
		Fields: []FieldDefinition{
			{Name: "username", Label: "Username", Type: "text", Required: true},
			{Name: "password", Label: "Password", Type: "password", Required: true},
		},
	},
	"dnspod": {
		Name: "DNSPod",
		URL:  "https://www.dnspod.cn",
		Fields: []FieldDefinition{
			{Name: "token", Label: "Token", Type: "password", Required: true},
		},
	},
	"domeneshop": {
		Name: "Domeneshop",
		URL:  "https://www.domeneshop.no",
		Fields: []FieldDefinition{
			{Name: "token", Label: "Token", Type: "password", Required: true},
			{Name: "secret", Label: "Secret", Type: "password", Required: true},
		},
	},
	"dondominio": {
		Name: "Don Dominio",
		URL:  "https://www.dondominio.com",
		Fields: []FieldDefinition{
			{Name: "username", Label: "Username", Type: "text", Required: true},
			{Name: "password", Label: "Password", Type: "password", Required: false, Help: "Deprecated, use key instead"},
			{Name: "key", Label: "Key", Type: "password", Required: true},
		},
	},
	"dreamhost": {
		Name: "Dreamhost",
		URL:  "https://www.dreamhost.com",
		Fields: []FieldDefinition{
			{Name: "key", Label: "API Key", Type: "password", Required: true},
		},
	},
	"duckdns": {
		Name: "DuckDNS",
		URL:  "https://www.duckdns.org",
		Fields: []FieldDefinition{
			{Name: "token", Label: "Token", Type: "password", Required: true, Placeholder: "UUID format", Help: "Get your token from duckdns.org"},
		},
	},
	"dyn": {
		Name: "DynDNS",
		URL:  "https://dyn.com",
		Fields: []FieldDefinition{
			{Name: "username", Label: "Username", Type: "text", Required: true},
			{Name: "password", Label: "Password", Type: "password", Required: false, Help: "Deprecated, use client_key instead"},
			{Name: "client_key", Label: "Client Key", Type: "password", Required: true},
		},
	},
	"dynu": {
		Name: "Dynu",
		URL:  "https://www.dynu.com",
		Fields: []FieldDefinition{
			{Name: "username", Label: "Username", Type: "text", Required: true},
			{Name: "password", Label: "Password", Type: "password", Required: true, Help: "Can be plain text, MD5, or SHA256"},
			{Name: "group", Label: "Group", Type: "text", Required: false},
		},
	},
	"dynv6": {
		Name: "DynV6",
		URL:  "https://dynv6.com",
		Fields: []FieldDefinition{
			{Name: "token", Label: "Token", Type: "password", Required: true},
		},
	},
	"easydns": {
		Name: "EasyDNS",
		URL:  "https://www.easydns.com",
		Fields: []FieldDefinition{
			{Name: "username", Label: "Username", Type: "text", Required: true},
			{Name: "token", Label: "Token", Type: "password", Required: true},
		},
	},
	"example": {
		Name: "Example",
		URL:  "",
		Fields: []FieldDefinition{
			{Name: "username", Label: "Username", Type: "text", Required: true},
			{Name: "password", Label: "Password", Type: "password", Required: true},
		},
	},
	"freedns": {
		Name: "FreeDNS",
		URL:  "https://freedns.afraid.org",
		Fields: []FieldDefinition{
			{Name: "token", Label: "Token", Type: "password", Required: true, Help: "Enable v2 dynamic DNS at freedns.afraid.org/dynamic/v2/"},
		},
	},
	"gandi": {
		Name: "Gandi",
		URL:  "https://www.gandi.net",
		Fields: []FieldDefinition{
			{Name: "ttl", Label: "TTL", Type: "number", Required: false, Placeholder: "3600", Help: "Default: 3600"},
		},
		AuthGroups: []AuthGroup{
			{
				Name: "Personal Access Token (recommended)",
				Fields: []FieldDefinition{
					{Name: "personal_access_token", Label: "Personal Access Token", Type: "password", Required: true},
				},
			},
			{
				Name: "API Key (deprecated)",
				Fields: []FieldDefinition{
					{Name: "key", Label: "API Key", Type: "password", Required: true, Help: "Deprecated, use Personal Access Token"},
				},
			},
		},
	},
	"gcp": {
		Name: "Google Cloud Platform",
		URL:  "https://cloud.google.com",
		Fields: []FieldDefinition{
			{Name: "project", Label: "Project", Type: "text", Required: true, Placeholder: "GCP project ID"},
			{Name: "zone", Label: "Zone", Type: "text", Required: true, Placeholder: "DNS zone name"},
			{Name: "credentials", Label: "Credentials JSON", Type: "password", Required: true, Help: "Full service account JSON credentials object"},
		},
	},
	"godaddy": {
		Name: "GoDaddy",
		URL:  "https://www.godaddy.com",
		Fields: []FieldDefinition{
			{Name: "key", Label: "API Key", Type: "password", Required: true, Placeholder: "Production API key"},
			{Name: "secret", Label: "API Secret", Type: "password", Required: true},
		},
	},
	"goip": {
		Name: "GoIP.de",
		URL:  "https://www.goip.de",
		Fields: []FieldDefinition{
			{Name: "username", Label: "Username", Type: "text", Required: true},
			{Name: "password", Label: "Password", Type: "password", Required: true},
		},
	},
	"he": {
		Name: "Hurricane Electric",
		URL:  "https://dns.he.net",
		Fields: []FieldDefinition{
			{Name: "password", Label: "Password", Type: "password", Required: true},
		},
	},
	"hetzner": {
		Name: "Hetzner",
		URL:  "https://www.hetzner.com",
		Fields: []FieldDefinition{
			{Name: "zone_identifier", Label: "Zone Identifier", Type: "text", Required: true},
			{Name: "token", Label: "API Token", Type: "password", Required: true},
			{Name: "ttl", Label: "TTL", Type: "number", Required: false, Placeholder: "1", Help: "Default: 1"},
		},
	},
	"infomaniak": {
		Name: "Infomaniak",
		URL:  "https://www.infomaniak.com",
		Fields: []FieldDefinition{
			{Name: "username", Label: "DynDNS Username", Type: "text", Required: true, Help: "Use DynDNS credentials, not admin"},
			{Name: "password", Label: "DynDNS Password", Type: "password", Required: true, Help: "Use DynDNS credentials, not admin"},
		},
	},
	"inwx": {
		Name: "INWX",
		URL:  "https://www.inwx.de",
		Fields: []FieldDefinition{
			{Name: "username", Label: "Username", Type: "text", Required: true},
			{Name: "password", Label: "Password", Type: "password", Required: true},
		},
	},
	"ionos": {
		Name: "Ionos",
		URL:  "https://www.ionos.com",
		Fields: []FieldDefinition{
			{Name: "api_key", Label: "API Key", Type: "password", Required: true, Placeholder: "prefix.key", Help: "Format: prefix.key"},
		},
	},
	"linode": {
		Name: "Linode",
		URL:  "https://www.linode.com",
		Fields: []FieldDefinition{
			{Name: "token", Label: "API Token", Type: "password", Required: true},
		},
	},
	"loopia": {
		Name: "Loopia",
		URL:  "https://www.loopia.se",
		Fields: []FieldDefinition{
			{Name: "username", Label: "Username", Type: "text", Required: true},
			{Name: "password", Label: "Password", Type: "password", Required: true},
		},
	},
	"luadns": {
		Name: "LuaDNS",
		URL:  "https://www.luadns.com",
		Fields: []FieldDefinition{
			{Name: "email", Label: "Email", Type: "text", Required: true},
			{Name: "token", Label: "Token", Type: "password", Required: true},
		},
	},
	"myaddr": {
		Name: "Myaddr.tools",
		URL:  "https://myaddr.tools",
		Fields: []FieldDefinition{
			{Name: "key", Label: "Key", Type: "password", Required: true},
		},
	},
	"namecheap": {
		Name: "Namecheap",
		URL:  "https://www.namecheap.com",
		Fields: []FieldDefinition{
			{Name: "password", Label: "Dynamic DNS Password", Type: "password", Required: true, Placeholder: "32-character hex", Help: "IPv4 only"},
		},
	},
	"name.com": {
		Name: "Name.com",
		URL:  "https://www.name.com",
		Fields: []FieldDefinition{
			{Name: "username", Label: "Username", Type: "text", Required: true},
			{Name: "token", Label: "Token", Type: "password", Required: true},
			{Name: "ttl", Label: "TTL", Type: "number", Required: false, Placeholder: "300", Help: "Minimum: 300"},
		},
	},
	"namesilo": {
		Name: "NameSilo",
		URL:  "https://www.namesilo.com",
		Fields: []FieldDefinition{
			{Name: "key", Label: "API Key", Type: "password", Required: true},
			{Name: "ttl", Label: "TTL", Type: "number", Required: false, Placeholder: "7207", Help: "Range: 3600-2592000"},
		},
	},
	"netcup": {
		Name: "Netcup",
		URL:  "https://www.netcup.de",
		Fields: []FieldDefinition{
			{Name: "api_key", Label: "API Key", Type: "password", Required: true},
			{Name: "password", Label: "API Password", Type: "password", Required: true, Help: "API password, not account password"},
			{Name: "customer_number", Label: "Customer Number", Type: "text", Required: true},
		},
	},
	"njalla": {
		Name: "Njalla",
		URL:  "https://njal.la",
		Fields: []FieldDefinition{
			{Name: "key", Label: "Key", Type: "password", Required: true},
		},
	},
	"noip": {
		Name: "No-IP",
		URL:  "https://www.noip.com",
		Fields: []FieldDefinition{
			{Name: "username", Label: "Username", Type: "text", Required: true},
			{Name: "password", Label: "Password", Type: "password", Required: true},
		},
	},
	"nowdns": {
		Name: "Now-DNS",
		URL:  "https://now-dns.com",
		Fields: []FieldDefinition{
			{Name: "username", Label: "Email", Type: "text", Required: true},
			{Name: "password", Label: "Password", Type: "password", Required: true},
		},
	},
	"opendns": {
		Name: "OpenDNS",
		URL:  "https://www.opendns.com",
		Fields: []FieldDefinition{
			{Name: "username", Label: "Username", Type: "text", Required: true},
			{Name: "password", Label: "Password", Type: "password", Required: true},
		},
	},
	"ovh": {
		Name: "OVH",
		URL:  "https://www.ovh.com",
		Fields: []FieldDefinition{
			{Name: "mode", Label: "Mode", Type: "select", Required: false, Options: []string{"dynamic", "api"}, Help: "DynHost (dynamic) or ZoneDNS API (api)"},
		},
		AuthGroups: []AuthGroup{
			{
				Name: "DynHost (dynamic)",
				Fields: []FieldDefinition{
					{Name: "username", Label: "Username", Type: "text", Required: true},
					{Name: "password", Label: "Password", Type: "password", Required: true},
				},
			},
			{
				Name: "ZoneDNS API",
				Fields: []FieldDefinition{
					{Name: "api_endpoint", Label: "API Endpoint", Type: "select", Required: false, Options: []string{"ovh-eu", "ovh-ca", "ovh-us", "soyoustart-eu", "soyoustart-ca", "kimsufi-eu", "kimsufi-ca"}, Help: "Default: ovh-eu"},
					{Name: "app_key", Label: "App Key", Type: "password", Required: true},
					{Name: "app_secret", Label: "App Secret", Type: "password", Required: true},
					{Name: "consumer_key", Label: "Consumer Key", Type: "password", Required: true},
				},
			},
		},
	},
	"porkbun": {
		Name: "Porkbun",
		URL:  "https://porkbun.com",
		Fields: []FieldDefinition{
			{Name: "api_key", Label: "API Key", Type: "password", Required: true},
			{Name: "secret_api_key", Label: "Secret API Key", Type: "password", Required: true},
			{Name: "ttl", Label: "TTL", Type: "number", Required: false},
		},
	},
	"route53": {
		Name: "Route53 (AWS)",
		URL:  "https://aws.amazon.com/route53",
		Fields: []FieldDefinition{
			{Name: "access_key", Label: "Access Key", Type: "password", Required: true},
			{Name: "secret_key", Label: "Secret Key", Type: "password", Required: true},
			{Name: "zone_id", Label: "Zone ID", Type: "text", Required: true},
			{Name: "ttl", Label: "TTL", Type: "number", Required: false, Placeholder: "300", Help: "Default: 300"},
		},
	},
	"selfhost.de": {
		Name: "Selfhost.de",
		URL:  "https://www.selfhost.de",
		Fields: []FieldDefinition{
			{Name: "username", Label: "DynDNS Username", Type: "text", Required: true},
			{Name: "password", Label: "DynDNS Password", Type: "password", Required: true},
		},
	},
	"servercow": {
		Name: "Servercow",
		URL:  "https://www.servercow.de",
		Fields: []FieldDefinition{
			{Name: "username", Label: "Username", Type: "text", Required: true},
			{Name: "password", Label: "Password", Type: "password", Required: true},
			{Name: "ttl", Label: "TTL", Type: "number", Required: false, Placeholder: "120", Help: "Default: 120"},
		},
	},
	"spdyn": {
		Name: "Spdyn",
		URL:  "https://www.spdyn.de",
		Fields: []FieldDefinition{},
		AuthGroups: []AuthGroup{
			{
				Name: "Token",
				Fields: []FieldDefinition{
					{Name: "token", Label: "Token", Type: "password", Required: true},
				},
			},
			{
				Name: "User & Password",
				Fields: []FieldDefinition{
					{Name: "user", Label: "User", Type: "text", Required: true},
					{Name: "password", Label: "Password", Type: "password", Required: true},
				},
			},
		},
	},
	"strato": {
		Name: "Strato",
		URL:  "https://www.strato.de",
		Fields: []FieldDefinition{
			{Name: "password", Label: "DynDNS Password", Type: "password", Required: true},
		},
	},
	"variomedia": {
		Name: "Variomedia",
		URL:  "https://www.variomedia.de",
		Fields: []FieldDefinition{
			{Name: "email", Label: "Email", Type: "text", Required: true},
			{Name: "password", Label: "DNS Password", Type: "password", Required: true, Help: "DNS settings password, not account password"},
		},
	},
	"vultr": {
		Name: "Vultr",
		URL:  "https://www.vultr.com",
		Fields: []FieldDefinition{
			{Name: "apikey", Label: "API Key", Type: "password", Required: true},
			{Name: "ttl", Label: "TTL", Type: "number", Required: false, Placeholder: "900", Help: "Default: 900"},
		},
	},
	"zoneedit": {
		Name: "Zoneedit",
		URL:  "https://www.zoneedit.com",
		Fields: []FieldDefinition{
			{Name: "username", Label: "Username", Type: "text", Required: true},
			{Name: "token", Label: "Token", Type: "password", Required: true},
		},
	},
}
```

- [ ] **Step 2: Verify it compiles**

Run: `cd /c/Users/repti/OneDrive/Dokumente/Programming/DDNS-updater-v3 && go build ./internal/provider/...`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add internal/provider/fielddefs.go
git commit -m "feat: add provider field definitions for WebUI form generation"
```

---

## Task 2: Backend API Handlers

**Files:**
- Create: `internal/server/api.go`
- Create: `internal/server/api_test.go`
- Modify: `internal/server/interfaces.go`

- [ ] **Step 1: Add StatusRecord to interfaces.go**

Add a new interface method to expose record status as JSON-friendly data. Add after the existing `Database` interface in `internal/server/interfaces.go`:

```go
// Add these imports to the existing import block:
// "net/netip"
// "time"

// StatusRecord holds JSON-serializable record status for the API.
type StatusRecord struct {
	Domain      string   `json:"domain"`
	Owner       string   `json:"owner"`
	Provider    string   `json:"provider"`
	IPVersion   string   `json:"ip_version"`
	Status      string   `json:"status"`
	Message     string   `json:"message"`
	CurrentIP   string   `json:"current_ip"`
	PreviousIPs []string `json:"previous_ips"`
	LastUpdated string   `json:"last_updated"`
}
```

The full file should look like:

```go
package server

import (
	"context"

	"github.com/qdm12/ddns-updater/internal/records"
)

type Database interface {
	SelectAll() (records []records.Record)
}

type UpdateForcer interface {
	ForceUpdate(ctx context.Context) (errors []error)
}

type Logger interface {
	Info(s string)
	Warn(s string)
	Error(s string)
}

// StatusRecord holds JSON-serializable record status for the API.
type StatusRecord struct {
	Domain      string   `json:"domain"`
	Owner       string   `json:"owner"`
	Provider    string   `json:"provider"`
	IPVersion   string   `json:"ip_version"`
	Status      string   `json:"status"`
	Message     string   `json:"message"`
	CurrentIP   string   `json:"current_ip"`
	PreviousIPs []string `json:"previous_ips"`
	LastUpdated string   `json:"last_updated"`
}
```

- [ ] **Step 2: Create api.go with all API handlers**

Create `internal/server/api.go`:

```go
package server

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/qdm12/ddns-updater/internal/provider"
)

// sensitiveFields are masked in GET responses.
var sensitiveFields = map[string]bool{
	"password":           true,
	"token":              true,
	"key":                true,
	"secret":             true,
	"api_key":            true,
	"secret_api_key":     true,
	"access_key":         true,
	"secret_key":         true,
	"access_key_id":      true,
	"access_secret":      true,
	"consumer_key":       true,
	"app_key":            true,
	"app_secret":         true,
	"client_key":         true,
	"user_service_key":   true,
	"credentials":           true,
	"personal_access_token": true,
	"apikey":                true,
	"customer_number":       true,
}

type apiHandlers struct {
	configPath string
	configMu   sync.Mutex
	db         Database
}

func newAPIHandlers(configPath string, db Database) *apiHandlers {
	return &apiHandlers{
		configPath: configPath,
		db:         db,
	}
}

// GET /api/status
func (a *apiHandlers) getStatus(w http.ResponseWriter, _ *http.Request) {
	allRecords := a.db.SelectAll()
	now := time.Now()
	statusRecords := make([]StatusRecord, len(allRecords))
	for i, rec := range allRecords {
		row := rec.Provider.HTML()
		currentIP := rec.History.GetCurrentIP()
		currentIPStr := ""
		if currentIP.IsValid() {
			currentIPStr = currentIP.String()
		}
		previousIPs := rec.History.GetPreviousIPs()
		prevIPStrs := make([]string, len(previousIPs))
		for j, ip := range previousIPs {
			prevIPStrs[j] = ip.String()
		}
		lastUpdated := ""
		if !rec.Time.IsZero() {
			lastUpdated = rec.Time.Format(time.RFC3339)
		}
		_ = now // available for future use
		statusRecords[i] = StatusRecord{
			Domain:      row.Domain,
			Owner:       row.Owner,
			Provider:    row.Provider,
			IPVersion:   row.IPVersion,
			Status:      string(rec.Status),
			Message:     rec.Message,
			CurrentIP:   currentIPStr,
			PreviousIPs: prevIPStrs,
			LastUpdated: lastUpdated,
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"records": statusRecords,
	})
}

// readConfig reads and parses the config.json file.
func (a *apiHandlers) readConfig() (map[string]interface{}, error) {
	data, err := os.ReadFile(a.configPath)
	if err != nil {
		return nil, err
	}
	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return config, nil
}

// getSettings extracts the settings array from config.
func getSettings(config map[string]interface{}) []interface{} {
	settings, ok := config["settings"]
	if !ok {
		return nil
	}
	arr, ok := settings.([]interface{})
	if !ok {
		return nil
	}
	return arr
}

// writeConfig writes config back to the file atomically.
func (a *apiHandlers) writeConfig(config map[string]interface{}) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	dir := filepath.Dir(a.configPath)
	tmpFile, err := os.CreateTemp(dir, "config-*.json.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()
	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, a.configPath); err != nil {
		os.Remove(tmpPath)
		// Fallback: direct write (rename fails across volumes on Windows)
		return os.WriteFile(a.configPath, data, fs.FileMode(0o666))
	}
	return nil
}

// maskSensitive replaces sensitive field values with "***".
func maskSensitive(entry map[string]interface{}) map[string]interface{} {
	masked := make(map[string]interface{}, len(entry))
	for k, v := range entry {
		if sensitiveFields[k] {
			if str, ok := v.(string); ok && str != "" {
				masked[k] = "***"
			} else {
				masked[k] = v
			}
		} else {
			masked[k] = v
		}
	}
	return masked
}

// GET /api/config
func (a *apiHandlers) getConfig(w http.ResponseWriter, _ *http.Request) {
	a.configMu.Lock()
	defer a.configMu.Unlock()

	config, err := a.readConfig()
	if err != nil {
		httpError(w, http.StatusInternalServerError, "failed to read config: "+err.Error())
		return
	}
	settings := getSettings(config)
	maskedSettings := make([]map[string]interface{}, len(settings))
	for i, s := range settings {
		entry, ok := s.(map[string]interface{})
		if !ok {
			maskedSettings[i] = map[string]interface{}{}
			continue
		}
		maskedSettings[i] = maskSensitive(entry)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"settings": maskedSettings,
	})
}

// POST /api/config
func (a *apiHandlers) postConfig(w http.ResponseWriter, r *http.Request) {
	a.configMu.Lock()
	defer a.configMu.Unlock()

	var newEntry map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&newEntry); err != nil {
		httpError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if _, ok := newEntry["provider"]; !ok {
		httpError(w, http.StatusBadRequest, "provider field is required")
		return
	}
	if _, ok := newEntry["domain"]; !ok {
		httpError(w, http.StatusBadRequest, "domain field is required")
		return
	}

	config, err := a.readConfig()
	if err != nil {
		httpError(w, http.StatusInternalServerError, "failed to read config: "+err.Error())
		return
	}
	settings := getSettings(config)
	settings = append(settings, newEntry)
	config["settings"] = settings

	if err := a.writeConfig(config); err != nil {
		httpError(w, http.StatusInternalServerError, "failed to write config: "+err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(maskSensitive(newEntry))
}

// PUT /api/config/{index}
func (a *apiHandlers) putConfig(w http.ResponseWriter, r *http.Request) {
	a.configMu.Lock()
	defer a.configMu.Unlock()

	indexStr := chi.URLParam(r, "index")
	index, err := strconv.Atoi(indexStr)
	if err != nil {
		httpError(w, http.StatusBadRequest, "invalid index")
		return
	}

	var updatedEntry map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updatedEntry); err != nil {
		httpError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	config, err := a.readConfig()
	if err != nil {
		httpError(w, http.StatusInternalServerError, "failed to read config: "+err.Error())
		return
	}
	settings := getSettings(config)
	if index < 0 || index >= len(settings) {
		httpError(w, http.StatusNotFound, "index out of range")
		return
	}

	existing, ok := settings[index].(map[string]interface{})
	if !ok {
		existing = map[string]interface{}{}
	}

	// Preserve sensitive fields if the new value is empty or "***"
	for k, v := range updatedEntry {
		if sensitiveFields[k] {
			str, isStr := v.(string)
			if isStr && (str == "" || str == "***") {
				if oldVal, exists := existing[k]; exists {
					updatedEntry[k] = oldVal
				}
			}
		}
	}
	// Also preserve sensitive fields not present in the update
	for k, v := range existing {
		if sensitiveFields[k] {
			if _, exists := updatedEntry[k]; !exists {
				updatedEntry[k] = v
			}
		}
	}

	settings[index] = updatedEntry
	config["settings"] = settings

	if err := a.writeConfig(config); err != nil {
		httpError(w, http.StatusInternalServerError, "failed to write config: "+err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(maskSensitive(updatedEntry))
}

// DELETE /api/config/{index}
func (a *apiHandlers) deleteConfig(w http.ResponseWriter, r *http.Request) {
	a.configMu.Lock()
	defer a.configMu.Unlock()

	indexStr := chi.URLParam(r, "index")
	index, err := strconv.Atoi(indexStr)
	if err != nil {
		httpError(w, http.StatusBadRequest, "invalid index")
		return
	}

	config, err := a.readConfig()
	if err != nil {
		httpError(w, http.StatusInternalServerError, "failed to read config: "+err.Error())
		return
	}
	settings := getSettings(config)
	if index < 0 || index >= len(settings) {
		httpError(w, http.StatusNotFound, "index out of range")
		return
	}

	settings = append(settings[:index], settings[index+1:]...)
	config["settings"] = settings

	if err := a.writeConfig(config); err != nil {
		httpError(w, http.StatusInternalServerError, "failed to write config: "+err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GET /api/providers
func (a *apiHandlers) getProviders(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"providers": provider.ProviderDefinitions,
	})
}
```

- [ ] **Step 3: Write API handler tests**

Create `internal/server/api_test.go`:

```go
package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"
)

func setupTestConfig(t *testing.T, content string) (string, *apiHandlers) {
	t.Helper()
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	err := os.WriteFile(configPath, []byte(content), 0o666)
	if err != nil {
		t.Fatal(err)
	}
	api := newAPIHandlers(configPath, nil)
	return configPath, api
}

func TestGetConfig(t *testing.T) {
	t.Parallel()
	_, api := setupTestConfig(t, `{"settings":[{"provider":"duckdns","domain":"test.duckdns.org","token":"secret123"}]}`)

	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	w := httptest.NewRecorder()
	api.getConfig(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	settings := result["settings"].([]interface{})
	if len(settings) != 1 {
		t.Fatalf("expected 1 setting, got %d", len(settings))
	}
	entry := settings[0].(map[string]interface{})
	if entry["token"] != "***" {
		t.Fatalf("expected token to be masked, got %v", entry["token"])
	}
	if entry["domain"] != "test.duckdns.org" {
		t.Fatalf("expected domain test.duckdns.org, got %v", entry["domain"])
	}
}

func TestPostConfig(t *testing.T) {
	t.Parallel()
	configPath, api := setupTestConfig(t, `{"settings":[]}`)

	body := `{"provider":"duckdns","domain":"new.duckdns.org","token":"abc123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/config", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	api.postConfig(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// Verify file was written
	data, _ := os.ReadFile(configPath)
	var config map[string]interface{}
	json.Unmarshal(data, &config)
	settings := config["settings"].([]interface{})
	if len(settings) != 1 {
		t.Fatalf("expected 1 setting in file, got %d", len(settings))
	}
}

func TestPutConfig(t *testing.T) {
	t.Parallel()
	configPath, api := setupTestConfig(t, `{"settings":[{"provider":"duckdns","domain":"old.duckdns.org","token":"secret"}]}`)

	router := chi.NewRouter()
	router.Put("/api/config/{index}", api.putConfig)

	body := `{"provider":"duckdns","domain":"updated.duckdns.org","token":"***"}`
	req := httptest.NewRequest(http.MethodPut, "/api/config/0", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Token should be preserved from existing
	data, _ := os.ReadFile(configPath)
	var config map[string]interface{}
	json.Unmarshal(data, &config)
	settings := config["settings"].([]interface{})
	entry := settings[0].(map[string]interface{})
	if entry["token"] != "secret" {
		t.Fatalf("expected token to be preserved as 'secret', got %v", entry["token"])
	}
	if entry["domain"] != "updated.duckdns.org" {
		t.Fatalf("expected domain updated.duckdns.org, got %v", entry["domain"])
	}
}

func TestDeleteConfig(t *testing.T) {
	t.Parallel()
	configPath, api := setupTestConfig(t, `{"settings":[{"provider":"a"},{"provider":"b"}]}`)

	router := chi.NewRouter()
	router.Delete("/api/config/{index}", api.deleteConfig)

	req := httptest.NewRequest(http.MethodDelete, "/api/config/0", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}

	data, _ := os.ReadFile(configPath)
	var config map[string]interface{}
	json.Unmarshal(data, &config)
	settings := config["settings"].([]interface{})
	if len(settings) != 1 {
		t.Fatalf("expected 1 setting, got %d", len(settings))
	}
	remaining := settings[0].(map[string]interface{})
	if remaining["provider"] != "b" {
		t.Fatalf("expected provider 'b' to remain, got %v", remaining["provider"])
	}
}

func TestDeleteConfigOutOfRange(t *testing.T) {
	t.Parallel()
	_, api := setupTestConfig(t, `{"settings":[{"provider":"a"}]}`)

	router := chi.NewRouter()
	router.Delete("/api/config/{index}", api.deleteConfig)

	req := httptest.NewRequest(http.MethodDelete, "/api/config/5", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestPostConfigMissingProvider(t *testing.T) {
	t.Parallel()
	_, api := setupTestConfig(t, `{"settings":[]}`)

	body := `{"domain":"test.com"}`
	req := httptest.NewRequest(http.MethodPost, "/api/config", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	api.postConfig(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestMaskSensitive(t *testing.T) {
	t.Parallel()
	entry := map[string]interface{}{
		"provider": "cloudflare",
		"domain":   "example.com",
		"token":    "my-secret-token",
		"ttl":      float64(1),
	}
	masked := maskSensitive(entry)
	if masked["token"] != "***" {
		t.Fatalf("expected token masked, got %v", masked["token"])
	}
	if masked["provider"] != "cloudflare" {
		t.Fatalf("expected provider unchanged, got %v", masked["provider"])
	}
	if masked["domain"] != "example.com" {
		t.Fatalf("expected domain unchanged, got %v", masked["domain"])
	}
}

func TestGetProviders(t *testing.T) {
	t.Parallel()
	api := newAPIHandlers("", nil)

	req := httptest.NewRequest(http.MethodGet, "/api/providers", nil)
	w := httptest.NewRecorder()
	api.getProviders(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	providers, ok := result["providers"].(map[string]interface{})
	if !ok {
		t.Fatal("expected providers map")
	}
	if _, ok := providers["cloudflare"]; !ok {
		t.Fatal("expected cloudflare in providers")
	}
	if _, ok := providers["duckdns"]; !ok {
		t.Fatal("expected duckdns in providers")
	}
}
```

- [ ] **Step 4: Run the tests**

Run: `cd /c/Users/repti/OneDrive/Dokumente/Programming/DDNS-updater-v3 && go test ./internal/server/ -run "TestGetConfig|TestPostConfig|TestPutConfig|TestDeleteConfig|TestMaskSensitive|TestGetProviders" -v`
Expected: All PASS

- [ ] **Step 5: Commit**

```bash
git add internal/server/api.go internal/server/api_test.go internal/server/interfaces.go
git commit -m "feat: add REST API handlers for config CRUD and status"
```

---

## Task 3: Wire Up Routes

**Files:**
- Modify: `internal/server/handler.go`
- Modify: `internal/server/server.go`
- Modify: `cmd/ddns-updater/main.go`

- [ ] **Step 1: Update handler.go to accept configPath and register API routes**

Replace `internal/server/handler.go` with:

```go
package server

import (
	"context"
	"embed"
	"io/fs"
	"net/http"
	"strings"
	"text/template"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type handlers struct {
	ctx context.Context //nolint:containedctx
	// Objects
	db            Database
	runner        UpdateForcer
	indexTemplate *template.Template
	// Mockable functions
	timeNow func() time.Time
}

//go:embed ui/*
var uiFS embed.FS

func newHandler(ctx context.Context, rootURL string,
	db Database, runner UpdateForcer, configPath string,
) http.Handler {
	indexTemplate := template.Must(template.ParseFS(uiFS, "ui/index.html"))

	staticFolder, err := fs.Sub(uiFS, "ui/static")
	if err != nil {
		panic(err)
	}

	handlers := &handlers{
		ctx:           ctx,
		db:            db,
		indexTemplate: indexTemplate,
		timeNow:       time.Now,
		runner:        runner,
	}

	api := newAPIHandlers(configPath, db)

	router := chi.NewRouter()

	router.Use(middleware.RealIP)
	router.Use(middleware.Logger)
	rootURL = strings.TrimSuffix(rootURL, "/")

	if rootURL != "" {
		router.Handle(rootURL, http.RedirectHandler(rootURL+"/", http.StatusPermanentRedirect))
	}
	router.Get(rootURL+"/", handlers.index)
	router.Get(rootURL+"/update", handlers.update)

	// API routes
	router.Get(rootURL+"/api/status", api.getStatus)
	router.Get(rootURL+"/api/config", api.getConfig)
	router.Post(rootURL+"/api/config", api.postConfig)
	router.Put(rootURL+"/api/config/{index}", api.putConfig)
	router.Delete(rootURL+"/api/config/{index}", api.deleteConfig)
	router.Get(rootURL+"/api/providers", api.getProviders)

	router.Handle(rootURL+"/static/*", http.StripPrefix(rootURL+"/static/", http.FileServerFS(staticFolder)))

	return router
}
```

- [ ] **Step 2: Update server.go to pass configPath**

Replace `internal/server/server.go` with:

```go
package server

import (
	"context"

	"github.com/qdm12/goservices/httpserver"
)

func New(ctx context.Context, address, rootURL string, db Database,
	logger Logger, runner UpdateForcer, configPath string,
) (server *httpserver.Server, err error) {
	return httpserver.New(httpserver.Settings{
		Handler: newHandler(ctx, rootURL, db, runner, configPath),
		Address: &address,
		Logger:  logger,
	})
}
```

- [ ] **Step 3: Update main.go to pass config path to server**

In `cmd/ddns-updater/main.go`, change the `createServer` function signature and call. Find the function `createServer` (around line 369) and replace it:

```go
//nolint:ireturn
func createServer(ctx context.Context, config config.Server,
	logger log.LoggerInterface, db server.Database,
	updaterService server.UpdateForcer, configPath string) (
	service goservices.Service, err error,
) {
	if !*config.Enabled {
		return noop.New("server"), nil
	}
	serverLogger := logger.New(log.SetComponent("http server"))
	return server.New(ctx, config.ListeningAddress, config.RootURL,
		db, serverLogger, updaterService, configPath)
}
```

Then find the call to `createServer` in `_main` (around line 218) and add the config path argument:

Change:
```go
server, err := createServer(ctx, config.Server, logger, db, updaterService)
```
To:
```go
server, err := createServer(ctx, config.Server, logger, db, updaterService, *config.Paths.Config)
```

- [ ] **Step 4: Verify it compiles**

Run: `cd /c/Users/repti/OneDrive/Dokumente/Programming/DDNS-updater-v3 && go build ./...`
Expected: No errors

- [ ] **Step 5: Run all existing tests**

Run: `cd /c/Users/repti/OneDrive/Dokumente/Programming/DDNS-updater-v3 && go test ./internal/server/ -v`
Expected: All PASS

- [ ] **Step 6: Commit**

```bash
git add internal/server/handler.go internal/server/server.go cmd/ddns-updater/main.go
git commit -m "feat: wire up API routes and pass config path through server"
```

---

## Task 4: Frontend HTML Shell

**Files:**
- Modify: `internal/server/ui/index.html`

- [ ] **Step 1: Replace index.html with SPA shell**

Replace `internal/server/ui/index.html` entirely. Note: the Go template `{{range .Rows}}` is removed. The `index` handler in `index.go` still works — it renders this HTML (the template just has no dynamic parts now, which is fine — the SPA fetches data via JS).

```html
<!DOCTYPE html>
<html lang="en">

<head>
  <title>DDNS Updater</title>
  <meta http-equiv="Content-Type" content="text/html; charset=UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <link rel="icon" href="static/favicon.svg" sizes="any" type="image/svg+xml">
  <link rel="icon" href="static/favicon.ico" type="image/x-icon">
  <link rel="stylesheet" href="static/styles.css" type="text/css">
</head>

<body>
  <nav class="tabs">
    <button class="tab active" data-tab="dashboard">Dashboard</button>
    <button class="tab" data-tab="configuration">Configuration</button>
  </nav>

  <main>
    <section id="dashboard" class="tab-content active">
      <div class="toolbar">
        <h2>DNS Records</h2>
        <button id="force-update-btn" class="btn btn-primary">Force Update All</button>
      </div>
      <div id="records-grid" class="cards-grid">
        <p class="loading">Loading...</p>
      </div>
    </section>

    <section id="configuration" class="tab-content">
      <div class="toolbar">
        <h2>Configuration</h2>
        <button id="add-entry-btn" class="btn btn-primary">+ Add Entry</button>
      </div>
      <div id="config-list" class="cards-grid">
        <p class="loading">Loading...</p>
      </div>
      <p class="config-note" id="restart-banner" style="display:none;">
        Configuration changed. Restart the application to apply changes.
      </p>
    </section>
  </main>

  <!-- Modal -->
  <div id="modal-overlay" class="modal-overlay" style="display:none;">
    <div class="modal">
      <div class="modal-header">
        <h3 id="modal-title">Add Entry</h3>
        <button id="modal-close" class="modal-close-btn">&times;</button>
      </div>
      <form id="entry-form" class="modal-body">
        <div class="form-group">
          <label for="provider-select">Provider</label>
          <select id="provider-select" required>
            <option value="">Select a provider...</option>
          </select>
        </div>
        <div class="form-group">
          <label for="domain-input">Domain</label>
          <input type="text" id="domain-input" required placeholder="e.g. sub.example.com">
        </div>
        <div class="form-row">
          <div class="form-group">
            <label for="ip-version-select">IP Version</label>
            <select id="ip-version-select">
              <option value="ipv4 or ipv6">IPv4 or IPv6</option>
              <option value="ipv4">IPv4</option>
              <option value="ipv6">IPv6</option>
            </select>
          </div>
          <div class="form-group" id="ipv6-suffix-group" style="display:none;">
            <label for="ipv6-suffix-input">IPv6 Suffix</label>
            <input type="text" id="ipv6-suffix-input" placeholder="e.g. ::1/64">
          </div>
        </div>
        <div id="auth-groups-container"></div>
        <div id="provider-fields-container"></div>
        <div class="modal-footer">
          <button type="button" id="modal-cancel" class="btn btn-secondary">Cancel</button>
          <button type="submit" class="btn btn-primary">Save</button>
        </div>
      </form>
    </div>
  </div>

  <!-- Delete Confirmation -->
  <div id="delete-overlay" class="modal-overlay" style="display:none;">
    <div class="modal modal-sm">
      <div class="modal-header">
        <h3>Confirm Delete</h3>
      </div>
      <div class="modal-body">
        <p id="delete-message">Are you sure?</p>
      </div>
      <div class="modal-footer">
        <button id="delete-cancel" class="btn btn-secondary">Cancel</button>
        <button id="delete-confirm" class="btn btn-danger">Delete</button>
      </div>
    </div>
  </div>

  <!-- Toast -->
  <div id="toast" class="toast" style="display:none;"></div>

  <footer>
    <div>
      <a href="https://github.com/qdm12/ddns-updater" class="text-big">
        <svg class="github-icon" height="1em" aria-hidden="true" viewBox="0 0 16 16" version="1.1">
          <path
            d="M8 0c4.42 0 8 3.58 8 8a8.013 8.013 0 0 1-5.45 7.59c-.4.08-.55-.17-.55-.38 0-.27.01-1.13.01-2.2 0-.75-.25-1.23-.54-1.48 1.78-.2 3.65-.88 3.65-3.95 0-.88-.31-1.59-.82-2.15.08-.2.36-1.02-.08-2.12 0 0-.67-.22-2.2.82-.64-.18-1.32-.27-2-.27-.68 0-1.36.09-2 .27-1.53-1.03-2.2-.82-2.2-.82-.44 1.1-.16 1.92-.08 2.12-.51.56-.82 1.28-.82 2.15 0 3.06 1.86 3.75 3.64 3.95-.23.2-.44.55-.51 1.07-.46.21-1.61.55-2.33-.66-.15-.24-.6-.83-1.23-.82-.67.01-.27.38.01.53.34.19.73.9.82 1.13.16.45.68 1.31 2.69.94 0 .67.01 1.3.01 1.49 0 .21-.15.45-.55.38A7.995 7.995 0 0 1 0 8c0-4.42 3.58-8 8-8Z">
          </path>
        </svg>
      </a>
    </div>
    <div>by <a href="https://github.com/qdm12">Quentin McGaw</a></div>
  </footer>
  <script src="static/app.js"></script>
</body>

</html>
```

- [ ] **Step 2: Update index.go to not fail on missing template variables**

The `index.html` no longer uses `{{range .Rows}}`, so the template just renders static HTML. The handler still works because `template.ExecuteTemplate` with a static template just outputs the HTML unchanged regardless of the data passed. No code change needed in `index.go` — verify by reading the file and confirming `ExecuteTemplate` is used (it is, line 15).

- [ ] **Step 3: Commit**

```bash
git add internal/server/ui/index.html
git commit -m "feat: replace Go template with SPA shell for interactive WebUI"
```

---

## Task 5: Frontend CSS

**Files:**
- Modify: `internal/server/ui/static/styles.css`

- [ ] **Step 1: Replace styles.css with modern dark-theme design**

Replace `internal/server/ui/static/styles.css` entirely:

```css
:root {
  --bg-primary: #0d1117;
  --bg-secondary: #161b22;
  --bg-card: #1c2128;
  --bg-card-hover: #252d38;
  --bg-input: #0d1117;
  --bg-modal: #1c2128;
  --text-primary: #e6edf3;
  --text-secondary: #8b949e;
  --text-muted: #6e7681;
  --border: #30363d;
  --border-hover: #484f58;
  --accent: #58a6ff;
  --accent-hover: #79c0ff;
  --accent-bg: rgba(56, 139, 253, 0.15);
  --danger: #f85149;
  --danger-hover: #ff7b72;
  --danger-bg: rgba(248, 81, 73, 0.15);
  --success: #3fb950;
  --warning: #d29922;
  --updating: #bc8cff;
  --radius: 10px;
  --radius-sm: 6px;
  --shadow: 0 3px 6px rgba(0,0,0,0.4);
  --transition: 0.15s ease;
  --font-mono: 'SF Mono', 'Cascadia Code', 'Consolas', monospace;
}

@media (prefers-color-scheme: light) {
  :root {
    --bg-primary: #f6f8fa;
    --bg-secondary: #ffffff;
    --bg-card: #ffffff;
    --bg-card-hover: #f3f4f6;
    --bg-input: #f6f8fa;
    --bg-modal: #ffffff;
    --text-primary: #1f2328;
    --text-secondary: #656d76;
    --text-muted: #8b949e;
    --border: #d0d7de;
    --border-hover: #afb8c1;
    --accent: #0969da;
    --accent-hover: #0550ae;
    --accent-bg: rgba(9, 105, 218, 0.1);
    --danger: #cf222e;
    --danger-hover: #a40e26;
    --danger-bg: rgba(207, 34, 46, 0.1);
    --success: #1a7f37;
    --warning: #9a6700;
    --updating: #8250df;
    --shadow: 0 3px 6px rgba(0,0,0,0.1);
  }
}

*, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }

html, body {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'Noto Sans', Helvetica, Arial, sans-serif;
  font-size: 14px;
  background: var(--bg-primary);
  color: var(--text-primary);
  min-height: 100vh;
}

a { color: var(--accent); text-decoration: none; }
a:hover { color: var(--accent-hover); text-decoration: underline; }

/* Tabs */
.tabs {
  display: flex;
  gap: 2px;
  background: var(--bg-secondary);
  padding: 8px 16px 0;
  border-bottom: 1px solid var(--border);
  position: sticky;
  top: 0;
  z-index: 100;
}

.tab {
  padding: 10px 20px;
  border: none;
  background: none;
  color: var(--text-secondary);
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  border-bottom: 2px solid transparent;
  transition: all var(--transition);
}
.tab:hover { color: var(--text-primary); }
.tab.active {
  color: var(--accent);
  border-bottom-color: var(--accent);
}

/* Main content */
main {
  max-width: 1400px;
  margin: 0 auto;
  padding: 20px 16px 80px;
}

.tab-content { display: none; }
.tab-content.active { display: block; }

.toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}
.toolbar h2 {
  font-size: 20px;
  font-weight: 600;
}

/* Buttons */
.btn {
  padding: 8px 16px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: all var(--transition);
  background: var(--bg-card);
  color: var(--text-primary);
}
.btn:hover { border-color: var(--border-hover); background: var(--bg-card-hover); }
.btn-primary {
  background: var(--accent);
  color: #fff;
  border-color: var(--accent);
}
.btn-primary:hover { background: var(--accent-hover); border-color: var(--accent-hover); }
.btn-danger { background: var(--danger-bg); color: var(--danger); border-color: var(--danger); }
.btn-danger:hover { background: var(--danger); color: #fff; }
.btn-secondary { background: var(--bg-card); color: var(--text-secondary); }
.btn-icon {
  padding: 6px 10px;
  background: none;
  border: 1px solid transparent;
  border-radius: var(--radius-sm);
  cursor: pointer;
  color: var(--text-secondary);
  font-size: 16px;
  transition: all var(--transition);
}
.btn-icon:hover { background: var(--bg-card-hover); color: var(--text-primary); border-color: var(--border); }
.btn-icon.danger:hover { color: var(--danger); }

/* Cards grid */
.cards-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(380px, 1fr));
  gap: 12px;
}

.card {
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  padding: 16px;
  transition: all var(--transition);
}
.card:hover { border-color: var(--border-hover); box-shadow: var(--shadow); }

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 12px;
}
.card-domain {
  font-size: 16px;
  font-weight: 600;
  word-break: break-all;
}
.card-actions { display: flex; gap: 4px; flex-shrink: 0; }

.badge {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 20px;
  font-size: 11px;
  font-weight: 500;
  border: 1px solid var(--border);
  background: var(--bg-secondary);
  color: var(--text-secondary);
}
.badge-provider { background: var(--accent-bg); color: var(--accent); border-color: transparent; }

.card-body { display: flex; flex-direction: column; gap: 8px; }
.card-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 13px;
}
.card-label { color: var(--text-secondary); }
.card-value { color: var(--text-primary); font-family: var(--font-mono); font-size: 12px; }

.card-footer {
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid var(--border);
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}
.status-dot.success { background: var(--success); }
.status-dot.fail, .status-dot.failure { background: var(--danger); }
.status-dot.uptodate { background: var(--success); }
.status-dot.updating { background: var(--updating); }
.status-dot.unset { background: var(--warning); }

.status-text { color: var(--text-secondary); }

/* Modal */
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0,0,0,0.6);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  backdrop-filter: blur(2px);
}
.modal {
  background: var(--bg-modal);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  width: 90%;
  max-width: 600px;
  max-height: 85vh;
  overflow-y: auto;
  box-shadow: 0 8px 24px rgba(0,0,0,0.4);
}
.modal-sm { max-width: 400px; }
.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border);
}
.modal-header h3 { font-size: 16px; font-weight: 600; }
.modal-close-btn {
  background: none;
  border: none;
  color: var(--text-secondary);
  font-size: 22px;
  cursor: pointer;
  padding: 0 4px;
  line-height: 1;
}
.modal-close-btn:hover { color: var(--text-primary); }
.modal-body { padding: 20px; }
.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  padding: 16px 20px;
  border-top: 1px solid var(--border);
}

/* Forms */
.form-group { margin-bottom: 16px; }
.form-group label {
  display: block;
  margin-bottom: 6px;
  font-size: 13px;
  font-weight: 500;
  color: var(--text-secondary);
}
.form-group input,
.form-group select,
.form-group textarea {
  width: 100%;
  padding: 8px 12px;
  background: var(--bg-input);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  color: var(--text-primary);
  font-size: 13px;
  font-family: inherit;
  transition: border-color var(--transition);
}
.form-group input:focus,
.form-group select:focus {
  outline: none;
  border-color: var(--accent);
  box-shadow: 0 0 0 3px var(--accent-bg);
}
.form-group .help-text {
  margin-top: 4px;
  font-size: 11px;
  color: var(--text-muted);
}
.form-row { display: flex; gap: 16px; }
.form-row .form-group { flex: 1; }

.form-group input[type="checkbox"] {
  width: auto;
  margin-right: 8px;
}
.checkbox-label {
  display: flex !important;
  align-items: center;
  flex-direction: row !important;
}

/* Auth group radio */
.auth-group-selector {
  margin-bottom: 16px;
  padding: 12px;
  background: var(--bg-secondary);
  border-radius: var(--radius-sm);
  border: 1px solid var(--border);
}
.auth-group-selector legend {
  font-size: 13px;
  font-weight: 500;
  color: var(--text-secondary);
  margin-bottom: 8px;
}
.auth-radio-group { display: flex; gap: 12px; flex-wrap: wrap; }
.auth-radio-group label {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  cursor: pointer;
  color: var(--text-primary);
}
.auth-radio-group input[type="radio"] { accent-color: var(--accent); }

.auth-fields { margin-top: 12px; }

/* Config note */
.config-note {
  margin-top: 16px;
  padding: 12px 16px;
  background: var(--warning);
  color: #000;
  border-radius: var(--radius-sm);
  font-size: 13px;
  font-weight: 500;
  text-align: center;
}

/* Toast */
.toast {
  position: fixed;
  bottom: 80px;
  left: 50%;
  transform: translateX(-50%);
  padding: 10px 24px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  color: var(--text-primary);
  font-size: 13px;
  box-shadow: var(--shadow);
  z-index: 2000;
  transition: opacity 0.3s;
}

/* Loading */
.loading {
  text-align: center;
  color: var(--text-muted);
  padding: 40px;
  grid-column: 1 / -1;
}

/* Footer */
footer {
  position: fixed;
  bottom: 0;
  left: 0;
  width: 100%;
  padding: 8px;
  display: flex;
  flex-direction: column;
  align-items: center;
  background: var(--bg-secondary);
  border-top: 1px solid var(--border);
  font-size: 12px;
  line-height: 1.4;
  z-index: 50;
}
footer a { color: var(--text-muted); }
footer a:hover { color: var(--text-secondary); }
.text-big { font-size: 1.3em; }
.github-icon { vertical-align: text-bottom; fill: currentColor; height: 1em; }

/* Responsive */
@media (max-width: 480px) {
  .cards-grid { grid-template-columns: 1fr; }
  .toolbar { flex-direction: column; gap: 12px; align-items: stretch; }
  .toolbar h2 { text-align: center; }
  .form-row { flex-direction: column; gap: 0; }
  .modal { width: 95%; }
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/server/ui/static/styles.css
git commit -m "feat: modern dark-theme CSS for interactive WebUI"
```

---

## Task 6: Frontend JavaScript

**Files:**
- Create: `internal/server/ui/static/app.js`

- [ ] **Step 1: Create app.js with all SPA logic**

Create `internal/server/ui/static/app.js`:

```js
// DDNS Updater Interactive WebUI
(function () {
  'use strict';

  let providers = {};
  let configEntries = [];
  let editIndex = -1; // -1 = new entry
  let deleteIndex = -1;

  // --- Utility ---
  function $(sel) { return document.querySelector(sel); }
  function $$(sel) { return document.querySelectorAll(sel); }

  function showToast(msg) {
    const toast = $('#toast');
    toast.textContent = msg;
    toast.style.display = 'block';
    toast.style.opacity = '1';
    setTimeout(() => {
      toast.style.opacity = '0';
      setTimeout(() => { toast.style.display = 'none'; }, 300);
    }, 2500);
  }

  async function api(method, path, body) {
    const opts = { method, headers: {} };
    if (body) {
      opts.headers['Content-Type'] = 'application/json';
      opts.body = JSON.stringify(body);
    }
    const res = await fetch(path, opts);
    if (res.status === 204) return null;
    const data = await res.json();
    if (!res.ok) throw new Error(data.error || 'Request failed');
    return data;
  }

  // --- Tabs ---
  function initTabs() {
    $$('.tab').forEach(tab => {
      tab.addEventListener('click', () => {
        $$('.tab').forEach(t => t.classList.remove('active'));
        $$('.tab-content').forEach(tc => tc.classList.remove('active'));
        tab.classList.add('active');
        const target = tab.dataset.tab;
        $('#' + target).classList.add('active');
        window.location.hash = target;
        if (target === 'dashboard') loadStatus();
        if (target === 'configuration') loadConfig();
      });
    });
    // Restore tab from hash
    const hash = window.location.hash.slice(1);
    if (hash === 'configuration') {
      $$('.tab').forEach(t => t.classList.remove('active'));
      $$('.tab-content').forEach(tc => tc.classList.remove('active'));
      $('[data-tab="configuration"]').classList.add('active');
      $('#configuration').classList.add('active');
    }
  }

  // --- Dashboard ---
  function statusClass(status) {
    const s = (status || '').toLowerCase();
    if (s === 'success') return 'success';
    if (s === 'fail') return 'fail';
    if (s === 'uptodate') return 'uptodate';
    if (s === 'updating') return 'updating';
    return 'unset';
  }

  function statusLabel(status) {
    const s = (status || '').toLowerCase();
    if (s === 'success') return 'Success';
    if (s === 'fail') return 'Failure';
    if (s === 'uptodate') return 'Up to date';
    if (s === 'updating') return 'Updating';
    return 'Unset';
  }

  function renderRecords(records) {
    const grid = $('#records-grid');
    if (!records || records.length === 0) {
      grid.innerHTML = '<p class="loading">No DNS records configured.</p>';
      return;
    }
    grid.innerHTML = records.map(rec => {
      const prevIPs = (rec.previous_ips || []).slice(0, 3).join(', ') || 'N/A';
      const currentIP = rec.current_ip || 'N/A';
      const ipLink = rec.current_ip
        ? '<a href="https://ipinfo.io/' + rec.current_ip + '" target="_blank">' + rec.current_ip + '</a>'
        : 'N/A';
      const sc = statusClass(rec.status);
      const timeAgo = rec.last_updated ? timeSince(rec.last_updated) : '';
      return '<div class="card">' +
        '<div class="card-header">' +
          '<span class="card-domain">' + escHtml(rec.domain) + '</span>' +
          '<span class="badge badge-provider">' + escHtml(rec.provider) + '</span>' +
        '</div>' +
        '<div class="card-body">' +
          '<div class="card-row"><span class="card-label">Owner</span><span class="card-value">' + escHtml(rec.owner) + '</span></div>' +
          '<div class="card-row"><span class="card-label">IP Version</span><span class="badge">' + escHtml(rec.ip_version) + '</span></div>' +
          '<div class="card-row"><span class="card-label">Current IP</span><span class="card-value">' + ipLink + '</span></div>' +
          '<div class="card-row"><span class="card-label">Previous IPs</span><span class="card-value">' + escHtml(prevIPs) + '</span></div>' +
        '</div>' +
        '<div class="card-footer">' +
          '<span class="status-dot ' + sc + '"></span>' +
          '<span class="status-text">' + statusLabel(rec.status) +
            (rec.message ? ' (' + escHtml(rec.message) + ')' : '') +
            (timeAgo ? ' &middot; ' + timeAgo : '') +
          '</span>' +
        '</div>' +
      '</div>';
    }).join('');
  }

  function timeSince(isoStr) {
    const diff = Date.now() - new Date(isoStr).getTime();
    const secs = Math.floor(diff / 1000);
    if (secs < 60) return secs + 's ago';
    const mins = Math.floor(secs / 60);
    if (mins < 60) return mins + 'm ago';
    const hours = Math.floor(mins / 60);
    if (hours < 24) return hours + 'h ago';
    return Math.floor(hours / 24) + 'd ago';
  }

  function escHtml(str) {
    const d = document.createElement('div');
    d.textContent = str || '';
    return d.innerHTML;
  }

  async function loadStatus() {
    try {
      const data = await api('GET', 'api/status');
      renderRecords(data.records);
    } catch (e) {
      $('#records-grid').innerHTML = '<p class="loading">Failed to load: ' + escHtml(e.message) + '</p>';
    }
  }

  // --- Configuration ---
  function renderConfig(settings) {
    const list = $('#config-list');
    if (!settings || settings.length === 0) {
      list.innerHTML = '<p class="loading">No entries configured. Click "+ Add Entry" to get started.</p>';
      return;
    }
    list.innerHTML = settings.map((entry, i) => {
      return '<div class="card">' +
        '<div class="card-header">' +
          '<span class="card-domain">' + escHtml(entry.domain || '') + '</span>' +
          '<div class="card-actions">' +
            '<button class="btn-icon" onclick="window._editEntry(' + i + ')" title="Edit">&#9998;</button>' +
            '<button class="btn-icon danger" onclick="window._deleteEntry(' + i + ')" title="Delete">&#128465;</button>' +
          '</div>' +
        '</div>' +
        '<div class="card-body">' +
          '<div class="card-row"><span class="card-label">Provider</span><span class="badge badge-provider">' + escHtml(entry.provider || '') + '</span></div>' +
          '<div class="card-row"><span class="card-label">IP Version</span><span class="badge">' + escHtml(entry.ip_version || 'ipv4 or ipv6') + '</span></div>' +
        '</div>' +
      '</div>';
    }).join('');
  }

  async function loadConfig() {
    try {
      const data = await api('GET', 'api/config');
      configEntries = data.settings || [];
      renderConfig(configEntries);
    } catch (e) {
      $('#config-list').innerHTML = '<p class="loading">Failed to load: ' + escHtml(e.message) + '</p>';
    }
  }

  async function loadProviders() {
    try {
      const data = await api('GET', 'api/providers');
      providers = data.providers || {};
    } catch (e) {
      console.error('Failed to load providers', e);
    }
  }

  // --- Modal ---
  function openModal(title, entry, index) {
    editIndex = index;
    $('#modal-title').textContent = title;
    $('#modal-overlay').style.display = 'flex';

    // Populate provider select
    const sel = $('#provider-select');
    sel.innerHTML = '<option value="">Select a provider...</option>';
    Object.keys(providers).sort().forEach(key => {
      const opt = document.createElement('option');
      opt.value = key;
      opt.textContent = providers[key].name || key;
      sel.appendChild(opt);
    });

    // Reset form
    $('#domain-input').value = '';
    $('#ip-version-select').value = 'ipv4 or ipv6';
    $('#ipv6-suffix-input').value = '';
    $('#ipv6-suffix-group').style.display = 'none';
    $('#provider-fields-container').innerHTML = '';
    $('#auth-groups-container').innerHTML = '';

    if (entry) {
      sel.value = entry.provider || '';
      $('#domain-input').value = entry.domain || '';
      $('#ip-version-select').value = entry.ip_version || 'ipv4 or ipv6';
      $('#ipv6-suffix-input').value = entry.ipv6_suffix || '';
      if (entry.provider) renderProviderFields(entry.provider, entry);
    }
    updateIpv6Visibility();
  }

  function closeModal() {
    $('#modal-overlay').style.display = 'none';
    editIndex = -1;
  }

  function updateIpv6Visibility() {
    const v = $('#ip-version-select').value;
    $('#ipv6-suffix-group').style.display = (v === 'ipv6' || v === 'ipv4 or ipv6') ? '' : 'none';
  }

  function renderProviderFields(providerKey, existingEntry) {
    const def = providers[providerKey];
    if (!def) return;

    const fieldsContainer = $('#provider-fields-container');
    const authContainer = $('#auth-groups-container');
    fieldsContainer.innerHTML = '';
    authContainer.innerHTML = '';

    // Auth groups
    if (def.auth_groups && def.auth_groups.length > 0) {
      let selectedGroup = 0;
      // Try to detect which auth group matches existing data
      if (existingEntry) {
        for (let g = 0; g < def.auth_groups.length; g++) {
          const group = def.auth_groups[g];
          const hasField = group.fields.some(f => existingEntry[f.name] && existingEntry[f.name] !== '');
          if (hasField) { selectedGroup = g; break; }
        }
      }

      let html = '<fieldset class="auth-group-selector"><legend>Authentication Method</legend>';
      html += '<div class="auth-radio-group">';
      def.auth_groups.forEach((group, i) => {
        html += '<label><input type="radio" name="auth-group" value="' + i + '"' +
          (i === selectedGroup ? ' checked' : '') + '> ' + escHtml(group.name) + '</label>';
      });
      html += '</div>';
      html += '<div class="auth-fields" id="auth-fields"></div>';
      html += '</fieldset>';
      authContainer.innerHTML = html;

      // Render initial auth fields
      renderAuthFields(def.auth_groups[selectedGroup], existingEntry);

      // Switch auth group on radio change
      authContainer.querySelectorAll('input[name="auth-group"]').forEach(radio => {
        radio.addEventListener('change', () => {
          renderAuthFields(def.auth_groups[parseInt(radio.value)], existingEntry);
        });
      });
    }

    // Regular fields
    fieldsContainer.innerHTML = def.fields.map(f => renderField(f, existingEntry)).join('');
  }

  function renderAuthFields(group, existingEntry) {
    const container = document.getElementById('auth-fields');
    if (!container) return;
    container.innerHTML = group.fields.map(f => renderField(f, existingEntry)).join('');
  }

  function renderField(f, existingEntry) {
    const val = existingEntry ? (existingEntry[f.name] || '') : '';
    const req = f.required ? ' required' : '';

    if (f.type === 'boolean') {
      const checked = val === true || val === 'true' ? ' checked' : '';
      return '<div class="form-group"><label class="checkbox-label">' +
        '<input type="checkbox" data-field="' + f.name + '"' + checked + '> ' + escHtml(f.label) +
        '</label>' +
        (f.help ? '<div class="help-text">' + escHtml(f.help) + '</div>' : '') +
        '</div>';
    }

    if (f.type === 'select' && f.options) {
      let opts = f.options.map(o =>
        '<option value="' + escHtml(o) + '"' + (val === o ? ' selected' : '') + '>' + escHtml(o) + '</option>'
      ).join('');
      return '<div class="form-group"><label>' + escHtml(f.label) + '</label>' +
        '<select data-field="' + f.name + '"' + req + '>' +
        '<option value="">Select...</option>' + opts + '</select>' +
        (f.help ? '<div class="help-text">' + escHtml(f.help) + '</div>' : '') +
        '</div>';
    }

    const inputType = f.type === 'password' ? 'password' : f.type === 'number' ? 'number' : 'text';
    return '<div class="form-group"><label>' + escHtml(f.label) + '</label>' +
      '<input type="' + inputType + '" data-field="' + f.name + '" value="' + escHtml(String(val)) + '"' +
      (f.placeholder ? ' placeholder="' + escHtml(f.placeholder) + '"' : '') +
      req + '>' +
      (f.help ? '<div class="help-text">' + escHtml(f.help) + '</div>' : '') +
      '</div>';
  }

  function collectFormData() {
    const data = {};
    data.provider = $('#provider-select').value;
    data.domain = $('#domain-input').value;
    const ipv = $('#ip-version-select').value;
    if (ipv !== 'ipv4 or ipv6') data.ip_version = ipv;
    const ipv6s = $('#ipv6-suffix-input').value.trim();
    if (ipv6s) data.ipv6_suffix = ipv6s;

    // Collect all data-field inputs
    $$('#entry-form [data-field]').forEach(el => {
      const name = el.dataset.field;
      if (el.type === 'checkbox') {
        if (el.checked) data[name] = true;
      } else if (el.type === 'number' && el.value) {
        data[name] = parseInt(el.value, 10);
      } else if (el.value) {
        data[name] = el.value;
      }
    });
    return data;
  }

  async function saveEntry(e) {
    e.preventDefault();
    const data = collectFormData();
    if (!data.provider || !data.domain) {
      showToast('Provider and domain are required');
      return;
    }
    try {
      if (editIndex >= 0) {
        await api('PUT', 'api/config/' + editIndex, data);
        showToast('Entry updated');
      } else {
        await api('POST', 'api/config', data);
        showToast('Entry added');
      }
      closeModal();
      loadConfig();
      $('#restart-banner').style.display = 'block';
    } catch (e) {
      showToast('Error: ' + e.message);
    }
  }

  // --- Delete ---
  function openDeleteDialog(index) {
    deleteIndex = index;
    const entry = configEntries[index];
    $('#delete-message').textContent = 'Delete entry for ' +
      (entry.domain || 'unknown') + ' (' + (entry.provider || 'unknown') + ')?';
    $('#delete-overlay').style.display = 'flex';
  }

  async function confirmDelete() {
    if (deleteIndex < 0) return;
    try {
      await api('DELETE', 'api/config/' + deleteIndex);
      showToast('Entry deleted');
      $('#delete-overlay').style.display = 'none';
      deleteIndex = -1;
      loadConfig();
      $('#restart-banner').style.display = 'block';
    } catch (e) {
      showToast('Error: ' + e.message);
    }
  }

  // --- Force Update ---
  async function forceUpdate() {
    const btn = $('#force-update-btn');
    btn.disabled = true;
    btn.textContent = 'Updating...';
    try {
      const res = await fetch('update');
      const text = await res.text();
      if (res.ok) {
        showToast(text);
      } else {
        showToast('Update failed');
      }
      loadStatus();
    } catch (e) {
      showToast('Error: ' + e.message);
    } finally {
      btn.disabled = false;
      btn.textContent = 'Force Update All';
    }
  }

  // --- Global handlers (used by onclick in rendered HTML) ---
  window._editEntry = function (i) {
    openModal('Edit Entry', configEntries[i], i);
  };
  window._deleteEntry = function (i) {
    openDeleteDialog(i);
  };

  // --- Init ---
  async function init() {
    initTabs();
    await loadProviders();

    // Event listeners
    $('#force-update-btn').addEventListener('click', forceUpdate);
    $('#add-entry-btn').addEventListener('click', () => openModal('Add Entry', null, -1));
    $('#modal-close').addEventListener('click', closeModal);
    $('#modal-cancel').addEventListener('click', closeModal);
    $('#modal-overlay').addEventListener('click', (e) => {
      if (e.target === $('#modal-overlay')) closeModal();
    });
    $('#entry-form').addEventListener('submit', saveEntry);
    $('#provider-select').addEventListener('change', (e) => {
      renderProviderFields(e.target.value, null);
    });
    $('#ip-version-select').addEventListener('change', updateIpv6Visibility);
    $('#delete-cancel').addEventListener('click', () => {
      $('#delete-overlay').style.display = 'none';
    });
    $('#delete-confirm').addEventListener('click', confirmDelete);

    // Load initial data
    const hash = window.location.hash.slice(1);
    if (hash === 'configuration') {
      loadConfig();
    } else {
      loadStatus();
    }

    // Auto-refresh dashboard every 30s
    setInterval(() => {
      if ($('#dashboard').classList.contains('active')) loadStatus();
    }, 30000);
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();
```

- [ ] **Step 2: Commit**

```bash
git add internal/server/ui/static/app.js
git commit -m "feat: add SPA JavaScript for interactive WebUI"
```

---

## Task 7: Integration Verification

**Files:** None new — this task verifies everything works together.

- [ ] **Step 1: Build the entire project**

Run: `cd /c/Users/repti/OneDrive/Dokumente/Programming/DDNS-updater-v3 && go build ./...`
Expected: No errors

- [ ] **Step 2: Run all tests**

Run: `cd /c/Users/repti/OneDrive/Dokumente/Programming/DDNS-updater-v3 && go test ./internal/server/ -v`
Expected: All PASS

- [ ] **Step 3: Run linter if available**

Run: `cd /c/Users/repti/OneDrive/Dokumente/Programming/DDNS-updater-v3 && go vet ./...`
Expected: No errors

- [ ] **Step 4: Final commit with all files**

Verify no unstaged files remain:
```bash
git status
```

If any files were missed, add and commit them:
```bash
git add -A
git commit -m "feat: complete interactive WebUI with config management"
```
