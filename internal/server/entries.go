package server

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/qdm12/ddns-updater/internal/provider/schema"
	"github.com/qdm12/ddns-updater/internal/reload"
)

// entriesPageData is the data rendered by the records list page.
type entriesPageData struct {
	RootURL               string
	ConfigPath            string
	EditingDisabledReason string
	Error                 string
	Entries               []reload.Entry
}

// entryFormPageData is the data rendered by the record form page.
type entryFormPageData struct {
	RootURL               string
	ConfigPath            string
	EditingDisabledReason string
	Error                 string
	Title                 string
	Action                string
	LoadAction            string
	IsEdit                bool
	Providers             []schema.Provider
	Selected              schema.Provider
	Domain                string
	IPVersion             string
	IPv6Suffix            string
	FieldValues           map[string]string
}

// entries lists the settings entries of the configuration file.
func (h *handlers) entries(w http.ResponseWriter, _ *http.Request) {
	h.renderEntries(w, http.StatusOK, "")
}

func (h *handlers) renderEntries(w http.ResponseWriter, status int, errorMessage string) {
	entries, err := h.entryManager.Entries()
	if err != nil {
		httpError(w, http.StatusInternalServerError,
			"reading configuration: "+err.Error())
		return
	}

	h.render(w, status, "entries.html", entriesPageData{
		RootURL:               h.rootURL,
		ConfigPath:            h.entryManager.ConfigPath(),
		EditingDisabledReason: h.entryManager.EditingDisabledReason(),
		Error:                 errorMessage,
		Entries:               entries,
	})
}

// entryNew renders an empty record form.
func (h *handlers) entryNew(w http.ResponseWriter, r *http.Request) {
	data := h.formPageData(nil)

	providerName := r.URL.Query().Get(formProviderKey)
	if provider, found := schema.Lookup(providerName); found {
		data.Selected = provider
	}
	h.render(w, http.StatusOK, "entry_form.html", data)
}

// entryEdit renders the record form filled with an existing entry.
func (h *handlers) entryEdit(w http.ResponseWriter, r *http.Request) {
	index, err := indexFromRequest(r)
	if err != nil {
		httpError(w, http.StatusNotFound, err.Error())
		return
	}

	entries, err := h.entryManager.Entries()
	if err != nil {
		httpError(w, http.StatusInternalServerError,
			"reading configuration: "+err.Error())
		return
	}
	if index >= len(entries) {
		httpError(w, http.StatusNotFound,
			fmt.Sprintf("%s: index %d", reload.ErrEntryNotFound, index))
		return
	}

	values, err := parseEntryValues(entries[index].Raw)
	if err != nil {
		httpError(w, http.StatusInternalServerError, err.Error())
		return
	}

	providerName := values.Provider
	// The provider can be changed from the form, in which case the
	// provider specific values of the previous provider no longer apply.
	if override := r.URL.Query().Get(formProviderKey); override != "" &&
		!strings.EqualFold(override, providerName) {
		providerName = override
		values.Fields = map[string]string{}
	}

	data := h.formPageData(&index)
	data.Domain = values.Domain
	data.IPVersion = values.IPVersion
	data.IPv6Suffix = values.IPv6Suffix
	data.FieldValues = values.Fields
	if provider, found := schema.Lookup(providerName); found {
		data.Selected = provider
	}
	h.render(w, http.StatusOK, "entry_form.html", data)
}

// entryCreate validates and appends a new settings entry.
func (h *handlers) entryCreate(w http.ResponseWriter, r *http.Request) {
	h.saveEntry(w, r, nil)
}

// entryUpdate validates and replaces an existing settings entry.
func (h *handlers) entryUpdate(w http.ResponseWriter, r *http.Request) {
	index, err := indexFromRequest(r)
	if err != nil {
		httpError(w, http.StatusNotFound, err.Error())
		return
	}

	entries, err := h.entryManager.Entries()
	if err != nil {
		httpError(w, http.StatusInternalServerError,
			"reading configuration: "+err.Error())
		return
	}
	if index >= len(entries) {
		httpError(w, http.StatusNotFound,
			fmt.Sprintf("%s: index %d", reload.ErrEntryNotFound, index))
		return
	}

	h.saveEntry(w, r, &index)
}

func (h *handlers) saveEntry(w http.ResponseWriter, r *http.Request, index *int) {
	if !checkBrowserRequest(r) {
		httpError(w, http.StatusForbidden, "cross origin request rejected")
		return
	}
	if !checkFormContentType(r) {
		httpError(w, http.StatusUnsupportedMediaType,
			"expected an application/x-www-form-urlencoded body")
		return
	}
	if err := r.ParseForm(); err != nil {
		h.renderForm(w, index, http.StatusBadRequest,
			"Reading the form: "+err.Error(), r.PostForm, schema.Provider{})
		return
	}

	provider, found := schema.Lookup(r.PostForm.Get(formProviderKey))
	if !found {
		h.renderForm(w, index, http.StatusBadRequest,
			"Choose one of the supported providers.", r.PostForm, schema.Provider{})
		return
	}

	entry, err := buildEntry(r.PostForm, provider)
	if err != nil {
		h.renderForm(w, index, http.StatusBadRequest, err.Error(), r.PostForm, provider)
		return
	}

	_, err = h.entryManager.SaveEntry(h.ctx, index, entry)
	if err != nil {
		h.renderForm(w, index, entryErrorStatus(err), err.Error(), r.PostForm, provider)
		return
	}

	http.Redirect(w, r, h.rootURL+"/entries", http.StatusSeeOther)
}

// entryDelete removes an existing settings entry.
func (h *handlers) entryDelete(w http.ResponseWriter, r *http.Request) {
	if !checkBrowserRequest(r) {
		httpError(w, http.StatusForbidden, "cross origin request rejected")
		return
	}

	index, err := indexFromRequest(r)
	if err != nil {
		httpError(w, http.StatusNotFound, err.Error())
		return
	}

	_, err = h.entryManager.DeleteEntry(h.ctx, index)
	if err != nil {
		h.renderEntries(w, entryErrorStatus(err), err.Error())
		return
	}

	http.Redirect(w, r, h.rootURL+"/entries", http.StatusSeeOther)
}

// renderForm renders the record form again, keeping the values the user
// submitted so that they can be fixed.
func (h *handlers) renderForm(w http.ResponseWriter, index *int,
	status int, errorMessage string, values url.Values, provider schema.Provider,
) {
	data := h.formPageData(index)
	data.Error = errorMessage
	data.Selected = provider
	data.Domain = values.Get(formDomainKey)
	data.IPVersion = values.Get(formIPVersionKey)
	data.IPv6Suffix = values.Get(formIPv6SuffixKey)
	data.FieldValues = make(map[string]string, len(provider.Fields))
	for _, field := range provider.Fields {
		name := formFieldPrefix + field.Key
		if !values.Has(name) {
			continue
		}
		data.FieldValues[field.Key] = values.Get(name)
	}
	h.render(w, status, "entry_form.html", data)
}

func (h *handlers) formPageData(index *int) entryFormPageData {
	data := entryFormPageData{
		RootURL:               h.rootURL,
		ConfigPath:            h.entryManager.ConfigPath(),
		EditingDisabledReason: h.entryManager.EditingDisabledReason(),
		Providers:             schema.Providers(),
		FieldValues:           map[string]string{},
	}
	if index != nil {
		data.IsEdit = true
		data.Action = fmt.Sprintf("%s/entries/%d", h.rootURL, *index)
		data.LoadAction = fmt.Sprintf("%s/entries/%d/edit", h.rootURL, *index)
		data.Title = fmt.Sprintf("Edit record %d", *index)
	} else {
		data.Action = h.rootURL + "/entries"
		data.LoadAction = h.rootURL + "/entries/new"
		data.Title = "Add a record"
	}
	return data
}

// render buffers the template output so that a template error can still be
// reported as a proper HTTP error.
func (h *handlers) render(w http.ResponseWriter, status int, name string, data any) {
	var buffer bytes.Buffer
	err := h.templates.ExecuteTemplate(&buffer, name, data)
	if err != nil {
		httpError(w, http.StatusInternalServerError,
			"failed generating webpage: "+err.Error())
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(buffer.Bytes())
}

func indexFromRequest(r *http.Request) (index int, err error) {
	rawIndex := chi.URLParam(r, "index")
	index, err = strconv.Atoi(rawIndex)
	if err != nil || index < 0 {
		return 0, fmt.Errorf("%w: %q", reload.ErrEntryNotFound, rawIndex)
	}
	return index, nil
}

func entryErrorStatus(err error) int {
	switch {
	case errors.Is(err, reload.ErrEditingDisabled):
		return http.StatusConflict
	case errors.Is(err, reload.ErrEntryNotFound):
		return http.StatusNotFound
	case errors.Is(err, reload.ErrInvalidEntry),
		errors.Is(err, errInvalidForm):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

// checkBrowserRequest rejects mutations coming from another origin, which
// protects the configuration from being changed by a malicious web page
// open in the same browser. The server has no authentication, so this is
// the only protection available.
func checkBrowserRequest(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		origin = r.Header.Get("Referer")
	}
	if origin == "" {
		// Not a browser request, or a browser which does not send the
		// header. Accept it, since there is no authentication anyway.
		return true
	}

	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}
	return strings.EqualFold(parsed.Host, r.Host)
}

func checkFormContentType(r *http.Request) bool {
	const formContentType = "application/x-www-form-urlencoded"
	contentType := r.Header.Get("Content-Type")
	return strings.HasPrefix(contentType, formContentType)
}
