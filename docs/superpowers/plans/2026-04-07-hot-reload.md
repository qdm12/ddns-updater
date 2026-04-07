# Hot-Reload Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Config changes via WebUI take effect immediately without app restart.

**Architecture:** After each API write operation (POST/PUT/DELETE), re-parse the entire config.json, create new Provider objects, match existing History by domain+owner+ipversion key, and atomically swap the Database record slice. The update loop picks up changes automatically via `db.SelectAll()`.

**Tech Stack:** Go, existing `internal/params` parser, `internal/data` Database, `internal/server` API handlers

**Spec:** `docs/superpowers/specs/2026-04-07-hot-reload-design.md`

---

### Task 1: Add `ReplaceAll` method to Database

**Files:**
- Modify: `internal/data/memory.go`
- Create: `internal/data/memory_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/data/memory_test.go`:

```go
package data

import (
	"testing"

	"github.com/qdm12/ddns-updater/internal/constants"
	"github.com/qdm12/ddns-updater/internal/records"
)

type mockProvider struct {
	domain    string
	owner     string
	ipVersion string
}

func (m mockProvider) String() string                              { return m.domain }
func (m mockProvider) Domain() string                              { return m.domain }
func (m mockProvider) Owner() string                               { return m.owner }
func (m mockProvider) BuildDomainName() string                     { return m.owner + "." + m.domain }
func (m mockProvider) HTML() (h struct{ Provider string })         { return }
func (m mockProvider) Proxied() bool                               { return false }
func (m mockProvider) IPVersion() (v struct{ String func() string }) { return }
func (m mockProvider) IPv6Suffix() (p struct{ IsValid func() bool }) { return }

func TestReplaceAll(t *testing.T) {
	t.Parallel()
	rec1 := records.Record{Status: constants.UNSET}
	db := NewDatabase([]records.Record{rec1}, nil)

	if len(db.SelectAll()) != 1 {
		t.Fatal("expected 1 record")
	}

	rec2 := records.Record{Status: constants.UPTODATE}
	rec3 := records.Record{Status: constants.UNSET}
	db.ReplaceAll([]records.Record{rec2, rec3})

	all := db.SelectAll()
	if len(all) != 2 {
		t.Fatalf("expected 2 records after ReplaceAll, got %d", len(all))
	}
	if all[0].Status != constants.UPTODATE {
		t.Fatalf("expected first record UPTODATE, got %s", all[0].Status)
	}
}

func TestReplaceAllEmpty(t *testing.T) {
	t.Parallel()
	rec1 := records.Record{Status: constants.UNSET}
	db := NewDatabase([]records.Record{rec1}, nil)

	db.ReplaceAll([]records.Record{})

	if len(db.SelectAll()) != 0 {
		t.Fatal("expected 0 records after ReplaceAll with empty slice")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/data/ -run TestReplaceAll -v`
Expected: FAIL — `db.ReplaceAll` undefined

- [ ] **Step 3: Implement `ReplaceAll`**

Add to `internal/data/memory.go` after the `SelectAll` function:

```go
func (db *Database) ReplaceAll(newData []records.Record) {
	db.Lock()
	defer db.Unlock()
	db.data = newData
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/data/ -run TestReplaceAll -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/data/memory.go internal/data/memory_test.go
git commit -m "feat: add ReplaceAll method to Database for hot-reload"
```

---

### Task 2: Export config parser function

**Files:**
- Modify: `internal/params/json.go`

- [ ] **Step 1: Add exported `ParseProviders` function**

Add to the end of `internal/params/json.go`:

```go
// ParseProviders parses config JSON bytes into provider objects.
// Exported for use by the API hot-reload mechanism.
func ParseProviders(configBytes []byte) ([]provider.Provider, []string, error) {
	return extractAllSettings(configBytes)
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./internal/params/`
Expected: success

- [ ] **Step 3: Commit**

```bash
git add internal/params/json.go
git commit -m "feat: export ParseProviders for API hot-reload"
```

---

### Task 3: Extend Database interface and add reload logic to API handlers

**Files:**
- Modify: `internal/server/interfaces.go`
- Modify: `internal/server/api.go`
- Modify: `internal/server/handler.go`
- Modify: `internal/server/server.go`

- [ ] **Step 1: Extend Database interface**

In `internal/server/interfaces.go`, add `ReplaceAll` to the `Database` interface:

```go
type Database interface {
	SelectAll() (records []records.Record)
	ReplaceAll(records []records.Record)
}
```

- [ ] **Step 2: Add reload fields and function to API handlers**

In `internal/server/api.go`, update the `apiHandlers` struct and constructor.

Replace the struct and constructor:

```go
// ConfigParser parses config JSON bytes into providers.
type ConfigParser func(configBytes []byte) ([]provider.Provider, []string, error)

type apiHandlers struct {
	configPath  string
	configMu    sync.Mutex
	db          Database
	parseConfig ConfigParser
}

func newAPIHandlers(configPath string, db Database, parseConfig ConfigParser) *apiHandlers {
	return &apiHandlers{
		configPath:  configPath,
		db:          db,
		parseConfig: parseConfig,
	}
}
```

Add the following imports to the import block (keep existing ones):

```go
"github.com/qdm12/ddns-updater/internal/records"
"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
```

Add the `reload` method after the `writeConfig` method:

```go
func (a *apiHandlers) reload() error {
	if a.parseConfig == nil {
		return nil
	}
	data, err := os.ReadFile(a.configPath)
	if err != nil {
		return fmt.Errorf("reading config for reload: %w", err)
	}
	providers, _, err := a.parseConfig(data)
	if err != nil {
		return fmt.Errorf("parsing config for reload: %w", err)
	}

	existing := a.db.SelectAll()
	historyMap := make(map[string]records.Record, len(existing))
	for _, rec := range existing {
		key := rec.Provider.Domain() + "|" + rec.Provider.Owner() + "|" + rec.Provider.IPVersion().String()
		historyMap[key] = rec
	}

	newRecords := make([]records.Record, len(providers))
	for i, p := range providers {
		key := p.Domain() + "|" + p.Owner() + "|" + p.IPVersion().String()
		if old, ok := historyMap[key]; ok {
			newRecords[i] = records.Record{
				Provider: p,
				History:  old.History,
				Status:   old.Status,
				Message:  old.Message,
				Time:     old.Time,
				LastBan:  old.LastBan,
			}
		} else {
			newRecords[i] = records.New(p, nil)
		}
	}

	a.db.ReplaceAll(newRecords)
	return nil
}
```

- [ ] **Step 3: Call reload after each write operation**

In `postConfig`, after the successful `writeConfig` call (after line `a.writeConfig(config)`), add reload. Replace the success response block:

Find this block in `postConfig`:
```go
	if err := a.writeConfig(config); err != nil {
		httpError(w, http.StatusInternalServerError, "failed to write config: "+err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(maskSensitive(newEntry))
```

Replace with:
```go
	if err := a.writeConfig(config); err != nil {
		httpError(w, http.StatusInternalServerError, "failed to write config: "+err.Error())
		return
	}

	if err := a.reload(); err != nil {
		httpError(w, http.StatusInternalServerError, "config saved but reload failed: "+err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(maskSensitive(newEntry))
```

In `putConfig`, after the successful `writeConfig` call, add the same reload pattern. Find:
```go
	if err := a.writeConfig(config); err != nil {
		httpError(w, http.StatusInternalServerError, "failed to write config: "+err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(maskSensitive(updatedEntry))
```

Replace with:
```go
	if err := a.writeConfig(config); err != nil {
		httpError(w, http.StatusInternalServerError, "failed to write config: "+err.Error())
		return
	}

	if err := a.reload(); err != nil {
		httpError(w, http.StatusInternalServerError, "config saved but reload failed: "+err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(maskSensitive(updatedEntry))
```

In `deleteConfig`, after the successful `writeConfig` call, add reload. Find:
```go
	if err := a.writeConfig(config); err != nil {
		httpError(w, http.StatusInternalServerError, "failed to write config: "+err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
```

Replace with:
```go
	if err := a.writeConfig(config); err != nil {
		httpError(w, http.StatusInternalServerError, "failed to write config: "+err.Error())
		return
	}

	if err := a.reload(); err != nil {
		httpError(w, http.StatusInternalServerError, "config saved but reload failed: "+err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
```

- [ ] **Step 4: Update handler.go to pass parser**

In `internal/server/handler.go`, update the `newHandler` signature and the `newAPIHandlers` call.

Change the function signature:
```go
func newHandler(ctx context.Context, rootURL string,
	db Database, runner UpdateForcer, configPath string,
	parseConfig ConfigParser,
) http.Handler {
```

Update the `newAPIHandlers` call inside:
```go
	api := newAPIHandlers(configPath, db, parseConfig)
```

- [ ] **Step 5: Update server.go to pass parser**

In `internal/server/server.go`, update the `New` function signature and call.

```go
func New(ctx context.Context, address, rootURL string, db Database,
	logger Logger, runner UpdateForcer, configPath string,
	parseConfig ConfigParser,
) (server *httpserver.Server, err error) {
	return httpserver.New(httpserver.Settings{
		Handler: newHandler(ctx, rootURL, db, runner, configPath, parseConfig),
		Address: &address,
		Logger:  logger,
	})
}
```

- [ ] **Step 6: Verify it compiles (ignoring main.go for now)**

Run: `go build ./internal/server/`
Expected: success (or only errors from main.go callers, not from server package itself)

- [ ] **Step 7: Commit**

```bash
git add internal/server/interfaces.go internal/server/api.go internal/server/handler.go internal/server/server.go
git commit -m "feat: add hot-reload to API handlers after config changes"
```

---

### Task 4: Wire up parser in main.go

**Files:**
- Modify: `cmd/ddns-updater/main.go`

- [ ] **Step 1: Find the `server.New` call and add the parser argument**

In `cmd/ddns-updater/main.go`, find the call to `server.New(...)` and add `params.ParseProviders` as the last argument.

Also add `"github.com/qdm12/ddns-updater/internal/params"` to the imports if not already present.

The `server.New` call should become:
```go
server.New(ctx, address, rootURL, db, logger, runner, configPath, params.ParseProviders)
```

Note: The exact variable names may differ — match whatever is currently passed to `server.New`.

- [ ] **Step 2: Verify the full project compiles**

Run: `go build ./cmd/ddns-updater/`
Expected: success

- [ ] **Step 3: Commit**

```bash
git add cmd/ddns-updater/main.go
git commit -m "feat: wire up config parser for hot-reload in main"
```

---

### Task 5: Update tests

**Files:**
- Modify: `internal/server/api_test.go`

- [ ] **Step 1: Update `setupTestConfig` to pass nil parser**

The `newAPIHandlers` call in tests needs a third argument. Update `setupTestConfig`:

```go
func setupTestConfig(t *testing.T, content string) (string, *apiHandlers) {
	t.Helper()
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	err := os.WriteFile(configPath, []byte(content), 0o666)
	if err != nil {
		t.Fatal(err)
	}
	api := newAPIHandlers(configPath, nil, nil)
	return configPath, api
}
```

Also update the `TestGetProviders` test which calls `newAPIHandlers("", nil)`:

```go
api := newAPIHandlers("", nil, nil)
```

- [ ] **Step 2: Run existing tests**

Run: `go test ./internal/server/ -v`
Expected: all existing tests PASS

- [ ] **Step 3: Add test for reload behavior**

Add to `internal/server/api_test.go`:

```go
type mockDB struct {
	records []records.Record
}

func (m *mockDB) SelectAll() []records.Record {
	return m.records
}

func (m *mockDB) ReplaceAll(newRecords []records.Record) {
	m.records = newRecords
}

func TestPostConfigTriggersReload(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	err := os.WriteFile(configPath, []byte(`{"settings":[]}`), 0o666)
	if err != nil {
		t.Fatal(err)
	}

	db := &mockDB{}
	parseCalled := false
	parser := func(data []byte) ([]provider.Provider, []string, error) {
		parseCalled = true
		return nil, nil, nil
	}
	api := newAPIHandlers(configPath, db, parser)

	body := `{"provider":"duckdns","domain":"test.duckdns.org","token":"abc"}`
	req := httptest.NewRequest(http.MethodPost, "/api/config", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	api.postConfig(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if !parseCalled {
		t.Fatal("expected parser to be called for reload")
	}
}
```

Add import for `"github.com/qdm12/ddns-updater/internal/provider"` and `"github.com/qdm12/ddns-updater/internal/records"` at the top.

- [ ] **Step 4: Run all tests**

Run: `go test ./internal/server/ -v`
Expected: all tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/server/api_test.go
git commit -m "test: update API tests for hot-reload parser parameter"
```

---

### Task 6: Remove restart banner from frontend

**Files:**
- Modify: `internal/server/ui/static/app.js`
- Modify: `internal/server/ui/index.html`

- [ ] **Step 1: Remove restart-banner display calls from app.js**

In `internal/server/ui/static/app.js`, remove the two lines that show the restart banner:

Line 342: `$('#restart-banner').style.display = 'block';` — remove this line.
Line 364: `$('#restart-banner').style.display = 'block';` — remove this line.

- [ ] **Step 2: Remove restart-banner HTML element from index.html**

In `internal/server/ui/index.html`, remove the restart banner paragraph:

```html
      <p class="config-note" id="restart-banner" style="display:none;">
        Configuration changed. Restart the application to apply changes.
      </p>
```

- [ ] **Step 3: Optionally add status refresh after config changes**

In `app.js`, after the `showToast('Entry updated')` and `showToast('Entry added')` calls inside the config form submit handler, add a call to refresh the dashboard status:

After `loadConfig();` on line 341, add: `loadStatus();`

Similarly after `loadConfig();` on line 363 (in `confirmDelete`), add: `loadStatus();`

This ensures the dashboard tab immediately shows new/removed entries.

- [ ] **Step 4: Commit**

```bash
git add internal/server/ui/static/app.js internal/server/ui/index.html
git commit -m "feat: remove restart banner, auto-refresh status after config changes"
```

---

### Task 7: Full integration verification

- [ ] **Step 1: Run all tests**

Run: `go test ./... 2>&1 | tail -20`
Expected: all tests PASS

- [ ] **Step 2: Build the binary**

Run: `go build -o ddnsapp.exe ./cmd/ddns-updater/`
Expected: compiles successfully

- [ ] **Step 3: Commit any remaining changes**

If any fixes were needed, commit them.
