package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/qdm12/ddns-updater/internal/configfile"
)

type configRequestJSON struct {
	Content string `json:"content"`
}

type configResponseJSON struct {
	OK       bool     `json:"ok"`
	Warnings []string `json:"warnings"`
	// RestartRequired is true if the change only applies after a restart.
	RestartRequired bool `json:"restartRequired"`
}

func (h *handlers) configPage(w http.ResponseWriter, _ *http.Request) {
	data := struct {
		Path      string
		Content   string
		ReadError string
		FromEnv   bool
		MaxSize   int
	}{
		Path:    h.configStore.Path(),
		FromEnv: h.configStore.FromEnvironment(),
		MaxSize: configfile.MaxSize,
	}
	content, err := h.configStore.Read()
	if err != nil {
		data.ReadError = err.Error()
	} else {
		data.Content = string(content)
	}
	renderPage(w, h.configTemplate, "config.html", data)
}

// configValidate checks the configuration sent without saving it.
func (h *handlers) configValidate(w http.ResponseWriter, r *http.Request) {
	content, ok := readConfigRequest(w, r)
	if !ok {
		return
	}
	warnings, err := h.configStore.Validate(content)
	if err != nil {
		writeConfigError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, configResponseJSON{OK: true, Warnings: nonNil(warnings)})
}

// configSave validates and saves the configuration sent.
func (h *handlers) configSave(w http.ResponseWriter, r *http.Request) {
	content, ok := readConfigRequest(w, r)
	if !ok {
		return
	}
	warnings, err := h.configStore.Save(content)
	if err != nil {
		writeConfigError(w, err)
		return
	}
	// The content is deliberately never logged since it holds credentials.
	h.logger.Info("configuration file " + h.configStore.Path() +
		" saved from the web UI by " + r.RemoteAddr)
	writeJSON(w, http.StatusOK, configResponseJSON{
		OK:              true,
		Warnings:        nonNil(warnings),
		RestartRequired: true,
	})
}

func readConfigRequest(w http.ResponseWriter, r *http.Request) (content []byte, ok bool) {
	if requireJSON(w, r) {
		return nil, false
	}

	// JSON escaping can make the body larger than the content itself.
	const bodyLimit = 4 * configfile.MaxSize
	r.Body = http.MaxBytesReader(w, r.Body, bodyLimit)

	var request configRequestJSON
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeJSONError(w, http.StatusRequestEntityTooLarge, "request body is too large")
			return nil, false
		}
		writeJSONError(w, http.StatusBadRequest, "invalid request: "+err.Error())
		return nil, false
	}
	return []byte(request.Content), true
}

func writeConfigError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, configfile.ErrFromEnvironment):
		writeJSONError(w, http.StatusConflict, err.Error())
	case errors.Is(err, configfile.ErrTooLarge):
		writeJSONError(w, http.StatusRequestEntityTooLarge, err.Error())
	case errors.Is(err, configfile.ErrInvalid):
		writeJSONError(w, http.StatusUnprocessableEntity, err.Error())
	default:
		writeJSONError(w, http.StatusInternalServerError, err.Error())
	}
}

// nonNil makes sure the warnings are encoded as [] instead of null.
func nonNil(warnings []string) []string {
	if warnings == nil {
		return []string{}
	}
	return warnings
}
