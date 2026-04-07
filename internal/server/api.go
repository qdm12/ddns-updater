package server

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/qdm12/ddns-updater/internal/provider"
	"github.com/qdm12/ddns-updater/internal/records"
)

const (
	maskedValue    = "***"
	configFilePerm = fs.FileMode(0o600)
	historyKeySep  = "|"
)

// sensitiveFieldNames lists the JSON field names whose values are masked in
// API responses to avoid leaking secrets to the WebUI.
//
//nolint:gochecknoglobals // intentional package-level lookup table
var sensitiveFieldNames = map[string]bool{
	"password":              true,
	"token":                 true,
	"key":                   true,
	"secret":                true,
	"api_key":               true,
	"secret_api_key":        true,
	"access_key":            true,
	"secret_key":            true,
	"access_key_id":         true,
	"access_secret":         true,
	"consumer_key":          true,
	"app_key":               true,
	"app_secret":            true,
	"client_key":            true,
	"user_service_key":      true,
	"credentials":           true,
	"personal_access_token": true,
	"apikey":                true,
	"customer_number":       true,
}

// stripHTMLTags removes HTML tags from a string. It is used to clean
// HTML-formatted provider names returned by Provider.HTML().
var stripHTMLTags = regexp.MustCompile(`<[^>]*>`)

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

// statusResponse is the JSON wrapper for GET /api/status.
type statusResponse struct {
	Records []StatusRecord `json:"records"`
}

// configResponse is the JSON wrapper for GET /api/config.
type configResponse struct {
	Settings []json.RawMessage `json:"settings"`
}

// providersResponse is the JSON wrapper for GET /api/providers.
type providersResponse struct {
	Providers map[string]provider.Definition `json:"providers"`
}

// writeJSON encodes a value as JSON and writes it to the response writer.
// It panics on encoding errors, matching the convention in error.go.
func writeJSON(w http.ResponseWriter, v any) {
	if err := json.NewEncoder(w).Encode(v); err != nil {
		panic(err)
	}
}

// getStatus handles GET /api/status.
func (a *apiHandlers) getStatus(w http.ResponseWriter, _ *http.Request) {
	allRecords := a.db.SelectAll()
	statusRecords := make([]StatusRecord, len(allRecords))
	for i, rec := range allRecords {
		htmlRow := rec.Provider.HTML()
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
		statusRecords[i] = StatusRecord{
			Domain:      rec.Provider.BuildDomainName(),
			Owner:       rec.Provider.Owner(),
			Provider:    stripHTMLTags.ReplaceAllString(htmlRow.Provider, ""),
			IPVersion:   rec.Provider.IPVersion().String(),
			Status:      string(rec.Status),
			Message:     rec.Message,
			CurrentIP:   currentIPStr,
			PreviousIPs: prevIPStrs,
			LastUpdated: lastUpdated,
		}
	}
	w.Header().Set("Content-Type", "application/json")
	writeJSON(w, statusResponse{Records: statusRecords})
}

// configFile is the in-memory representation of config.json.
type configFile struct {
	Settings []json.RawMessage `json:"settings"`
}

func (a *apiHandlers) readConfig() (*configFile, map[string]any, error) {
	data, err := os.ReadFile(a.configPath)
	if err != nil {
		return nil, nil, err
	}
	var cf configFile
	if err := json.Unmarshal(data, &cf); err != nil {
		return nil, nil, err
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, nil, err
	}
	return &cf, raw, nil
}

func (a *apiHandlers) writeConfig(raw map[string]any) error {
	data, err := json.MarshalIndent(raw, "", "  ")
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
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, a.configPath); err != nil {
		_ = os.Remove(tmpPath)
		return os.WriteFile(a.configPath, data, configFilePerm)
	}
	return nil
}

// validateConfig parses the config to check it produces valid providers.
// Returns the parsed providers for use by applyProviders.
func (a *apiHandlers) validateConfig(raw map[string]any) ([]provider.Provider, error) {
	if a.parseConfig == nil {
		return nil, nil
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("marshaling config: %w", err)
	}
	providers, _, err := a.parseConfig(data)
	if err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	return providers, nil
}

// applyProviders swaps the in-memory database records with new providers,
// preserving history for matching entries (by domain+owner+ipversion key).
func (a *apiHandlers) applyProviders(providers []provider.Provider) {
	if providers == nil {
		return
	}
	makeKey := func(domain, owner, ipVersion string) string {
		return domain + historyKeySep + owner + historyKeySep + ipVersion
	}

	existing := a.db.SelectAll()
	historyMap := make(map[string]records.Record, len(existing))
	for _, rec := range existing {
		p := rec.Provider
		historyMap[makeKey(p.Domain(), p.Owner(), p.IPVersion().String())] = rec
	}

	newRecords := make([]records.Record, len(providers))
	for i, p := range providers {
		key := makeKey(p.Domain(), p.Owner(), p.IPVersion().String())
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
}

func maskSensitive(entry map[string]any) map[string]any {
	masked := make(map[string]any, len(entry))
	for k, v := range entry {
		if sensitiveFieldNames[k] {
			if str, ok := v.(string); ok && str != "" {
				masked[k] = maskedValue
			} else {
				masked[k] = v
			}
		} else {
			masked[k] = v
		}
	}
	return masked
}

// maskRawSettings unmarshals each settings entry, masks its sensitive fields,
// and returns the masked entries as raw JSON messages.
func maskRawSettings(rawSettings []json.RawMessage) ([]json.RawMessage, error) {
	out := make([]json.RawMessage, len(rawSettings))
	for i, raw := range rawSettings {
		var entry map[string]any
		if err := json.Unmarshal(raw, &entry); err != nil {
			return nil, err
		}
		masked := maskSensitive(entry)
		b, err := json.Marshal(masked)
		if err != nil {
			return nil, err
		}
		out[i] = b
	}
	return out, nil
}

// getConfig handles GET /api/config.
func (a *apiHandlers) getConfig(w http.ResponseWriter, _ *http.Request) {
	a.configMu.Lock()
	defer a.configMu.Unlock()

	cf, _, err := a.readConfig()
	if err != nil {
		httpError(w, http.StatusInternalServerError, "failed to read config: "+err.Error())
		return
	}
	maskedSettings, err := maskRawSettings(cf.Settings)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "failed to mask config: "+err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	writeJSON(w, configResponse{Settings: maskedSettings})
}

// applyAndPersist validates the new config, writes it to disk, and reloads
// providers in memory. It returns an HTTP error code and message if anything
// fails.
func (a *apiHandlers) applyAndPersist(raw map[string]any) (int, string) {
	providers, err := a.validateConfig(raw)
	if err != nil {
		return http.StatusBadRequest, err.Error()
	}
	if err := a.writeConfig(raw); err != nil {
		return http.StatusInternalServerError, "failed to write config: " + err.Error()
	}
	a.applyProviders(providers)
	return 0, ""
}

// postConfig handles POST /api/config.
func (a *apiHandlers) postConfig(w http.ResponseWriter, r *http.Request) {
	a.configMu.Lock()
	defer a.configMu.Unlock()

	var newEntry map[string]any
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

	_, raw, err := a.readConfig()
	if err != nil {
		httpError(w, http.StatusInternalServerError, "failed to read config: "+err.Error())
		return
	}
	settings, _ := raw["settings"].([]any)
	settings = append(settings, newEntry)
	raw["settings"] = settings

	if status, msg := a.applyAndPersist(raw); status != 0 {
		httpError(w, status, msg)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	maskedJSON, err := json.Marshal(maskSensitive(newEntry))
	if err != nil {
		panic(err)
	}
	if _, err := w.Write(maskedJSON); err != nil {
		panic(err)
	}
}

// putConfig handles PUT /api/config/{index}.
func (a *apiHandlers) putConfig(w http.ResponseWriter, r *http.Request) {
	a.configMu.Lock()
	defer a.configMu.Unlock()

	indexStr := chi.URLParam(r, "index")
	index, err := strconv.Atoi(indexStr)
	if err != nil {
		httpError(w, http.StatusBadRequest, "invalid index")
		return
	}

	var updatedEntry map[string]any
	if err := json.NewDecoder(r.Body).Decode(&updatedEntry); err != nil {
		httpError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	_, raw, err := a.readConfig()
	if err != nil {
		httpError(w, http.StatusInternalServerError, "failed to read config: "+err.Error())
		return
	}
	settings, _ := raw["settings"].([]any)
	if index < 0 || index >= len(settings) {
		httpError(w, http.StatusNotFound, "index out of range")
		return
	}

	existing, ok := settings[index].(map[string]any)
	if !ok {
		existing = map[string]any{}
	}

	preserveSensitiveFields(updatedEntry, existing)

	settings[index] = updatedEntry
	raw["settings"] = settings

	if status, msg := a.applyAndPersist(raw); status != 0 {
		httpError(w, status, msg)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	maskedJSON, err := json.Marshal(maskSensitive(updatedEntry))
	if err != nil {
		panic(err)
	}
	if _, err := w.Write(maskedJSON); err != nil {
		panic(err)
	}
}

// preserveSensitiveFields keeps the existing sensitive values when the
// updated entry has them empty or set to the masked placeholder.
func preserveSensitiveFields(updated, existing map[string]any) {
	for k, v := range updated {
		if !sensitiveFieldNames[k] {
			continue
		}
		str, isStr := v.(string)
		if isStr && (str == "" || str == maskedValue) {
			if oldVal, exists := existing[k]; exists {
				updated[k] = oldVal
			}
		}
	}
	for k, v := range existing {
		if !sensitiveFieldNames[k] {
			continue
		}
		if _, exists := updated[k]; !exists {
			updated[k] = v
		}
	}
}

// deleteConfig handles DELETE /api/config/{index}.
func (a *apiHandlers) deleteConfig(w http.ResponseWriter, r *http.Request) {
	a.configMu.Lock()
	defer a.configMu.Unlock()

	indexStr := chi.URLParam(r, "index")
	index, err := strconv.Atoi(indexStr)
	if err != nil {
		httpError(w, http.StatusBadRequest, "invalid index")
		return
	}

	_, raw, err := a.readConfig()
	if err != nil {
		httpError(w, http.StatusInternalServerError, "failed to read config: "+err.Error())
		return
	}
	settings, _ := raw["settings"].([]any)
	if index < 0 || index >= len(settings) {
		httpError(w, http.StatusNotFound, "index out of range")
		return
	}

	settings = append(settings[:index], settings[index+1:]...)
	raw["settings"] = settings

	if status, msg := a.applyAndPersist(raw); status != 0 {
		httpError(w, status, msg)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// getProviders handles GET /api/providers.
func (a *apiHandlers) getProviders(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	writeJSON(w, providersResponse{Providers: provider.Definitions})
}
