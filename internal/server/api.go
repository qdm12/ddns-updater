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

// sensitiveFields are masked in GET responses.
var sensitiveFields = map[string]bool{
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

// GET /api/status
func (a *apiHandlers) getStatus(w http.ResponseWriter, _ *http.Request) {
	stripHTML := regexp.MustCompile(`<[^>]*>`)
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
			Provider:    stripHTML.ReplaceAllString(htmlRow.Provider, ""),
			IPVersion:   rec.Provider.IPVersion().String(),
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
		return os.WriteFile(a.configPath, data, fs.FileMode(0o666))
	}
	return nil
}

// validateConfig parses the config to check it produces valid providers.
// Returns the parsed providers for use by applyProviders.
func (a *apiHandlers) validateConfig(config map[string]interface{}) ([]provider.Provider, error) {
	if a.parseConfig == nil {
		return nil, nil
	}
	data, err := json.Marshal(config)
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
}

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

	providers, err := a.validateConfig(config)
	if err != nil {
		httpError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := a.writeConfig(config); err != nil {
		httpError(w, http.StatusInternalServerError, "failed to write config: "+err.Error())
		return
	}
	a.applyProviders(providers)

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
	for k, v := range existing {
		if sensitiveFields[k] {
			if _, exists := updatedEntry[k]; !exists {
				updatedEntry[k] = v
			}
		}
	}

	settings[index] = updatedEntry
	config["settings"] = settings

	providers, err := a.validateConfig(config)
	if err != nil {
		httpError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := a.writeConfig(config); err != nil {
		httpError(w, http.StatusInternalServerError, "failed to write config: "+err.Error())
		return
	}
	a.applyProviders(providers)

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

	providers, err := a.validateConfig(config)
	if err != nil {
		httpError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := a.writeConfig(config); err != nil {
		httpError(w, http.StatusInternalServerError, "failed to write config: "+err.Error())
		return
	}
	a.applyProviders(providers)

	w.WriteHeader(http.StatusNoContent)
}

// GET /api/providers
func (a *apiHandlers) getProviders(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"providers": provider.ProviderDefinitions,
	})
}
