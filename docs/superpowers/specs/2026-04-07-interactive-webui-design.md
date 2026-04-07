# DDNS Updater Interactive WebUI - Design Spec

## Overview

Transform the current static HTML table UI into an interactive single-page application with two tabs: **Dashboard** (modernized status overview) and **Configuration** (CRUD management for DDNS entries). The config.json remains the single source of truth. The existing update logic is untouched.

## Architecture

### Frontend

- **Technology:** Vanilla JS embedded via Go's `embed` package (no build step, no framework)
- **Location:** `internal/server/ui/` — replaces current `index.html` and `styles.css`
- **Structure:**
  - `index.html` — SPA shell with tab navigation
  - `static/styles.css` — modern responsive CSS with dark/light mode
  - `static/app.js` — all client-side logic (tab routing, API calls, dynamic forms, modals)

### Backend

- **New file:** `internal/server/api.go` — REST API handlers for config CRUD and status
- **Modified file:** `internal/server/handler.go` — register new API routes on the existing chi router
- **New file:** `internal/provider/fielddefs.go` — provider field definitions for dynamic form generation

### What is NOT changed

- `internal/provider/providers/*/provider.go` — all 50+ provider implementations
- `internal/records/` — record management and history
- `internal/config/` — environment-variable-based application config
- `cmd/ddns-updater/main.go` — startup orchestration (minimal changes to pass config path to server)
- `internal/params/json.go` — JSON parsing logic (reused by API handlers)
- Health check server, backup logic, notification system

---

## REST API Endpoints

All new endpoints are prefixed with `/api/`.

### `GET /api/status`

Returns current status of all DDNS records for the Dashboard tab.

**Response:**
```json
{
  "records": [
    {
      "domain": "example.com",
      "owner": "@",
      "provider": "cloudflare",
      "ip_version": "ipv4",
      "status": "success",
      "message": "",
      "current_ip": "203.0.113.1",
      "previous_ips": ["203.0.113.2", "203.0.113.3"],
      "last_updated": "2026-04-07T10:30:00Z"
    }
  ]
}
```

### `GET /api/config`

Returns all settings entries from config.json.

**Response:**
```json
{
  "settings": [
    {
      "provider": "cloudflare",
      "domain": "example.com",
      "zone_identifier": "abc123",
      "token": "***",
      "ttl": 1,
      "ip_version": "ipv4"
    }
  ]
}
```

**Security note:** Sensitive fields (password, token, key, secret, etc.) are masked with `"***"` in GET responses. The frontend shows masked values and only sends new values on edit (empty string = keep existing).

### `POST /api/config`

Add a new DDNS entry.

**Request body:** A single settings object (same shape as one entry in config.json).

```json
{
  "provider": "cloudflare",
  "domain": "new.example.com",
  "zone_identifier": "abc123",
  "token": "my-api-token",
  "ttl": 1,
  "ip_version": "ipv4"
}
```

**Response:** `201 Created` with the created entry (masked), or `400 Bad Request` with validation errors.

**Behavior:** Appends to the `settings` array in config.json and writes the file.

### `PUT /api/config/{index}`

Update an existing entry by its 0-based index in the settings array.

**Request body:** Full settings object (fields with `""` keep existing value for sensitive fields).

**Response:** `200 OK` with updated entry (masked), or `400`/`404`.

**Behavior:** Replaces the entry at `index` in config.json and writes the file.

### `DELETE /api/config/{index}`

Delete an entry by its 0-based index.

**Response:** `204 No Content`, or `404`.

**Behavior:** Removes the entry from the settings array, writes config.json.

### `GET /api/providers`

Returns the list of all supported providers with their field definitions for dynamic form rendering.

**Response:**
```json
{
  "providers": {
    "cloudflare": {
      "name": "Cloudflare",
      "url": "https://www.cloudflare.com",
      "fields": [
        {
          "name": "zone_identifier",
          "label": "Zone Identifier",
          "type": "text",
          "required": true,
          "placeholder": "e.g. abc123def456",
          "help": "Found in your Cloudflare dashboard under Overview"
        },
        {
          "name": "ttl",
          "label": "TTL",
          "type": "number",
          "required": true,
          "placeholder": "1",
          "help": "Set to 1 for automatic"
        },
        {
          "name": "proxied",
          "label": "Proxied",
          "type": "boolean",
          "required": false,
          "help": "Enable Cloudflare proxy"
        }
      ],
      "auth_groups": [
        {
          "name": "API Token (recommended)",
          "fields": [
            {
              "name": "token",
              "label": "API Token",
              "type": "password",
              "required": true,
              "placeholder": "Your Cloudflare API token"
            }
          ]
        },
        {
          "name": "Global API Key",
          "fields": [
            {
              "name": "email",
              "label": "Email",
              "type": "text",
              "required": true
            },
            {
              "name": "key",
              "label": "Global API Key",
              "type": "password",
              "required": true
            }
          ]
        },
        {
          "name": "User Service Key",
          "fields": [
            {
              "name": "user_service_key",
              "label": "User Service Key",
              "type": "password",
              "required": true
            }
          ]
        }
      ]
    }
  }
}
```

Providers with a single auth method have no `auth_groups` — their fields are all in `fields`. Providers with multiple auth options (Cloudflare, OVH, Spdyn) use `auth_groups` with radio-button selection in the UI.

---

## Frontend Design

### Tab Navigation

A horizontal tab bar at the top of the page:
- **Dashboard** — default active tab
- **Configuration** — CRUD management

Tab switching is client-side (no page reload). URL hash tracks active tab (`#dashboard`, `#configuration`).

### Dashboard Tab

**Layout:** Responsive card grid (CSS Grid, 1 column mobile, 2 columns tablet, 3 columns desktop).

**Each card contains:**
- **Header row:** Domain name (bold, linked to `http://domain`) + Provider badge (small colored pill)
- **Body:**
  - IP Version tag (small badge: "IPv4", "IPv6", "IPv4/v6")
  - Current IP (monospace, linked to ipinfo.io) or "N/A"
  - Previous IPs (smaller, muted text)
- **Footer:** Status indicator (colored dot + label + time since last update)
  - Green dot = Success / Up to date
  - Red dot = Failure
  - Orange dot = Unset
  - Purple dot = Updating

**Top bar:** "Force Update All" button. On click: calls `GET /update`, shows spinner, then refreshes status via `GET /api/status`.

**Auto-refresh:** Dashboard polls `GET /api/status` every 30 seconds to keep status current.

### Configuration Tab

**Layout:** Vertical list of entry cards + "Add Entry" button at top.

**Each entry card shows:**
- Provider icon/badge + domain name + owner
- IP version tag
- Edit button (pencil icon) + Delete button (trash icon)

**Add/Edit Modal:**

A centered modal overlay with the form:

1. **Provider select** — dropdown with all 50+ providers alphabetically. On change: fetches field definitions and re-renders form fields dynamically.

2. **Common fields** (always visible after provider selection):
   - `domain` — text input, required
   - `ip_version` — select dropdown: "IPv4", "IPv6", "IPv4 or IPv6" (default)
   - `ipv6_suffix` — text input, optional, only shown when ip_version includes IPv6

3. **Provider-specific fields** — rendered dynamically based on `/api/providers` response:
   - `text` → `<input type="text">`
   - `password` → `<input type="password">` with show/hide toggle
   - `number` → `<input type="number">`
   - `boolean` → `<input type="checkbox">`
   - `select` → `<select>` with predefined options

4. **Auth group selector** — for providers with multiple auth methods (Cloudflare, OVH, Spdyn): radio buttons at the top of the auth section. Selecting one shows only that group's fields.

5. **Action buttons:** "Save" (primary) + "Cancel" (secondary)

**Validation:**
- Client-side: required fields must be non-empty before submit
- Server-side: full validation via existing provider parsing logic, errors returned as JSON and displayed inline

**Delete confirmation:** Simple modal: "Delete entry for {domain} ({provider})?" with Confirm/Cancel buttons.

**After save/delete:** A toast notification appears briefly ("Entry saved", "Entry deleted"), the entry list refreshes.

---

## Provider Field Definitions

A new Go data structure in `internal/provider/fielddefs.go` that maps each provider to its field requirements. This is a static definition derived from the existing provider docs.

```go
type FieldDefinition struct {
    Name        string   `json:"name"`
    Label       string   `json:"label"`
    Type        string   `json:"type"`        // "text", "password", "number", "boolean", "select"
    Required    bool     `json:"required"`
    Placeholder string   `json:"placeholder,omitempty"`
    Help        string   `json:"help,omitempty"`
    Options     []string `json:"options,omitempty"` // for "select" type
}

type AuthGroup struct {
    Name   string            `json:"name"`
    Fields []FieldDefinition `json:"fields"`
}

type ProviderDefinition struct {
    Name       string            `json:"name"`
    URL        string            `json:"url"`
    Fields     []FieldDefinition `json:"fields"`
    AuthGroups []AuthGroup       `json:"auth_groups,omitempty"`
}

var ProviderDefinitions = map[string]ProviderDefinition{
    "cloudflare": { ... },
    "namecheap": { ... },
    // ... all 50+ providers
}
```

This map is the single source for which fields appear in the form for each provider.

---

## Config File I/O

### Reading

The API handlers read config.json using Go's `os.ReadFile` + `json.Unmarshal` into `[]map[string]interface{}` to preserve all fields including unknown ones.

### Writing

After modification, the settings array is marshalled back with `json.MarshalIndent` (2-space indent) and written atomically (write to temp file, then rename).

### Sensitive Field Masking

A hardcoded list of sensitive field names: `password`, `token`, `key`, `secret`, `api_key`, `secret_api_key`, `access_key`, `secret_key`, `access_secret`, `consumer_key`, `app_key`, `app_secret`, `client_key`, `user_service_key`, `credentials`.

On `GET /api/config`: these fields are replaced with `"***"`.
On `PUT /api/config/{index}`: if a sensitive field is `""` or `"***"`, the existing value is preserved.

---

## CSS Design Tokens

```css
:root {
    /* Base */
    --bg-primary: #0f0f0f;
    --bg-secondary: #1a1a2e;
    --bg-card: #16213e;
    --bg-modal: #1a1a2e;
    --text-primary: #e0e0e0;
    --text-secondary: #a0a0a0;
    --text-muted: #666;
    --border: #2a2a4a;
    --border-hover: #3a3a5a;

    /* Accent */
    --accent: #4fc3f7;
    --accent-hover: #81d4fa;

    /* Status */
    --status-success: #4caf50;
    --status-error: #f44336;
    --status-warning: #ff9800;
    --status-updating: #9c27b0;
    --status-unset: #ff9800;

    /* Sizing */
    --radius: 12px;
    --radius-sm: 8px;
    --shadow: 0 4px 6px rgba(0, 0, 0, 0.3);
    --transition: 0.2s ease;
}

/* Light mode override */
@media (prefers-color-scheme: light) {
    :root {
        --bg-primary: #f5f5f5;
        --bg-secondary: #ffffff;
        --bg-card: #ffffff;
        --bg-modal: #ffffff;
        --text-primary: #1a1a1a;
        --text-secondary: #555;
        --text-muted: #999;
        --border: #e0e0e0;
        --border-hover: #ccc;
        --shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
    }
}
```

---

## File Changes Summary

### New files
| File | Purpose |
|------|---------|
| `internal/server/api.go` | REST API handlers (config CRUD, status, providers) |
| `internal/provider/fielddefs.go` | Provider field definitions for form generation |
| `internal/server/ui/static/app.js` | Client-side SPA logic |

### Modified files
| File | Change |
|------|--------|
| `internal/server/handler.go` | Add API routes to chi router |
| `internal/server/ui/index.html` | Replace with SPA shell (tabs, card containers, modal template) |
| `internal/server/ui/static/styles.css` | Complete rewrite with modern card-based design |

### Unchanged files
All files in `internal/provider/providers/`, `internal/records/`, `internal/config/`, `internal/params/`, `cmd/ddns-updater/main.go` (possibly minor change to pass config path).

---

## Edge Cases & Considerations

1. **Concurrent writes:** The API uses a mutex to serialize config.json read/write operations.
2. **Restart requirement:** After config changes, the running updater still uses the old providers in memory. A banner in the UI will note: "Configuration changed. Restart the application to apply changes." Future improvement could add hot-reload.
3. **Index stability:** DELETE shifts indices. The frontend always re-fetches the full list after mutations.
4. **Large configs:** With 50+ entries the card list might get long. No pagination needed — DDNS configs rarely exceed ~20 entries.
5. **Validation errors:** Returned as structured JSON `{"errors": {"field_name": "error message"}}` and displayed inline next to the relevant form field.
6. **File permissions:** Config.json is written respecting the existing umask setting.
