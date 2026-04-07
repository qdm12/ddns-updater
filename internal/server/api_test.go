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
	"github.com/qdm12/ddns-updater/internal/provider"
	"github.com/qdm12/ddns-updater/internal/records"
)

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
	api := newAPIHandlers("", nil, nil)

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
