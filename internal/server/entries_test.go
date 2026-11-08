package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/qdm12/ddns-updater/internal/provider/schema"
	"github.com/qdm12/ddns-updater/internal/records"
	"github.com/qdm12/ddns-updater/internal/reload"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testDuckDNSEntry = `{"domain":"example.duckdns.org","provider":"duckdns",` +
		`"token":"00000000-0000-0000-0000-000000000000"}`
	testCloudflareEntry = `{"domain":"example.com","provider":"cloudflare",` +
		`"ip_version":"ipv4","token":"token-value","proxied":true,"ttl":300}`
)

type stubDatabase struct {
	records []records.Record
}

func (s stubDatabase) SelectAll() []records.Record { return s.records }

type stubUpdateForcer struct{}

func (stubUpdateForcer) ForceUpdate(context.Context) []error { return nil }

type stubEntryManager struct {
	entries        []reload.Entry
	configPath     string
	disabledReason string
	entriesErr     error
	saveErr        error
	deleteErr      error
	savedIndex     *int
	savedEntry     json.RawMessage
	deletedIndex   int
	saveResult     reload.Result
	deleteResult   reload.Result
}

func (s *stubEntryManager) ConfigPath() string { return s.configPath }

func (s *stubEntryManager) EditingDisabledReason() string { return s.disabledReason }

func (s *stubEntryManager) Entries() ([]reload.Entry, error) {
	return s.entries, s.entriesErr
}

func (s *stubEntryManager) SaveEntry(_ context.Context, index *int,
	entry json.RawMessage,
) (reload.Result, error) {
	s.savedIndex = index
	s.savedEntry = entry
	if s.saveErr != nil {
		return reload.Result{}, s.saveErr
	}
	if index == nil {
		entryIndex := len(s.entries)
		s.entries = append(s.entries, reload.Entry{Index: entryIndex, Raw: entry})
	}
	return s.saveResult, nil
}

func (s *stubEntryManager) DeleteEntry(_ context.Context, index int) (
	reload.Result, error,
) {
	s.deletedIndex = index
	if s.deleteErr != nil {
		return reload.Result{}, s.deleteErr
	}
	s.entries = append(s.entries[:index], s.entries[index+1:]...)
	return s.deleteResult, nil
}

func newTestHandler(t *testing.T, manager EntryManager) http.Handler {
	t.Helper()
	return newHandler(context.Background(), "", stubDatabase{},
		stubUpdateForcer{}, &stubReloader{}, manager)
}

func newStubEntryManager() *stubEntryManager {
	return &stubEntryManager{
		configPath: "/config/config.json",
		entries: []reload.Entry{
			{
				Index:    0,
				Provider: "duckdns",
				Domain:   "example.duckdns.org",
				Raw:      json.RawMessage(testDuckDNSEntry),
			},
			{
				Index:     1,
				Provider:  "cloudflare",
				Domain:    "example.com",
				IPVersion: "ipv4",
				Raw:       json.RawMessage(testCloudflareEntry),
			},
		},
	}
}

func get(t *testing.T, handler http.Handler, target string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, target, nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func postForm(t *testing.T, handler http.Handler, target string,
	form url.Values,
) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, target,
		strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func Test_entries_listPage(t *testing.T) {
	t.Parallel()

	handler := newTestHandler(t, newStubEntryManager())

	recorder := get(t, handler, "/entries")
	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "text/html; charset=utf-8", recorder.Header().Get("Content-Type"))

	body := recorder.Body.String()
	assert.Contains(t, body, "example.duckdns.org")
	assert.Contains(t, body, "example.com")
	assert.Contains(t, body, "/entries/0/edit")
	assert.Contains(t, body, "/entries/1/edit")
	assert.Contains(t, body, "/entries/0/delete")
	assert.Contains(t, body, "/entries/new")
	assert.Contains(t, body, "/config/config.json")
}

func Test_entries_listPage_editingDisabled(t *testing.T) {
	t.Parallel()

	manager := newStubEntryManager()
	manager.disabledReason = "editing is disabled for a good reason"
	handler := newTestHandler(t, manager)

	recorder := get(t, handler, "/entries")
	require.Equal(t, http.StatusOK, recorder.Code)

	body := recorder.Body.String()
	assert.Contains(t, body, "editing is disabled for a good reason")
	assert.NotContains(t, body, "/entries/0/edit")
	assert.NotContains(t, body, "/entries/0/delete")
}

func Test_entries_indexPageLinksRecords(t *testing.T) {
	t.Parallel()

	handler := newTestHandler(t, newStubEntryManager())

	recorder := get(t, handler, "/")
	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), `href="entries"`)
}

func Test_entries_newForm(t *testing.T) {
	t.Parallel()

	handler := newTestHandler(t, newStubEntryManager())

	recorder := get(t, handler, "/entries/new")
	require.Equal(t, http.StatusOK, recorder.Code)

	body := recorder.Body.String()
	assert.Contains(t, body, "Choose a provider...")
	assert.Contains(t, body, `value="duckdns"`)
	assert.Contains(t, body, "Choose a provider to see its settings.")
	assert.NotContains(t, body, `name="field_token"`)
}

func Test_entries_newFormWithProvider(t *testing.T) {
	t.Parallel()

	handler := newTestHandler(t, newStubEntryManager())

	recorder := get(t, handler, "/entries/new?provider=cloudflare")
	require.Equal(t, http.StatusOK, recorder.Code)

	body := recorder.Body.String()
	assert.Contains(t, body, `name="field_token"`)
	assert.Contains(t, body, "Zone identifier")
	assert.Contains(t, body, "Proxied through Cloudflare")
	assert.Contains(t, body, `type="checkbox"`)
	assert.Contains(t, body, `name="domain"`)
	assert.Contains(t, body, "https://github.com/qdm12/ddns-updater/blob/master/docs/cloudflare.md")
}

func Test_entries_editForm(t *testing.T) {
	t.Parallel()

	handler := newTestHandler(t, newStubEntryManager())

	recorder := get(t, handler, "/entries/1/edit")
	require.Equal(t, http.StatusOK, recorder.Code)

	body := recorder.Body.String()
	assert.Contains(t, body, `action="/entries/1"`)
	assert.Contains(t, body, `value="example.com"`)
	assert.Contains(t, body, `value="token-value"`)
	assert.Contains(t, body, `value="300"`)
	assert.Contains(t, body, "checked>")
	assert.Contains(t, body, `value="ipv4" selected`)
}

func Test_entries_editFormWithProviderOverride(t *testing.T) {
	t.Parallel()

	handler := newTestHandler(t, newStubEntryManager())

	recorder := get(t, handler, "/entries/1/edit?provider=duckdns")
	require.Equal(t, http.StatusOK, recorder.Code)

	body := recorder.Body.String()
	assert.Contains(t, body, `action="/entries/1"`)
	assert.Contains(t, body, `value="example.com"`, "the domain is kept")
	assert.Contains(t, body, `name="field_token"`)
	assert.NotContains(t, body, `value="token-value"`,
		"the settings of the previous provider are dropped")
	assert.NotContains(t, body, "Proxied through Cloudflare")
}

func Test_entries_editFormUnknownIndex(t *testing.T) {
	t.Parallel()

	handler := newTestHandler(t, newStubEntryManager())

	for _, target := range []string{"/entries/2/edit", "/entries/-1/edit", "/entries/x/edit"} {
		recorder := get(t, handler, target)
		assert.Equal(t, http.StatusNotFound, recorder.Code, target)
	}
}

func Test_entries_create(t *testing.T) {
	t.Parallel()

	manager := newStubEntryManager()
	handler := newTestHandler(t, manager)

	form := url.Values{
		"provider":      {"cloudflare"},
		"domain":        {"sub.example.com"},
		"ip_version":    {"ipv6"},
		"ipv6_suffix":   {"::1234"},
		"field_token":   {"secret-token"},
		"field_proxied": {"on"},
		"field_ttl":     {"120"},
		"field_email":   {""},
	}
	recorder := postForm(t, handler, "/entries", form)
	require.Equal(t, http.StatusSeeOther, recorder.Code)
	assert.Equal(t, "/entries", recorder.Header().Get("Location"))

	assert.Nil(t, manager.savedIndex)

	entry := decodeEntry(t, manager.savedEntry)
	assert.Equal(t, map[string]any{
		"provider":    "cloudflare",
		"domain":      "sub.example.com",
		"ip_version":  "ipv6",
		"ipv6_suffix": "::1234",
		"token":       "secret-token",
		"proxied":     true,
		"ttl":         float64(120),
	}, entry)
}

func Test_entries_create_invalidForms(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		form           url.Values
		expectedStatus int
		expectedBody   string
	}{
		"missing_provider": {
			form:           url.Values{"domain": {"example.com"}},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Choose one of the supported providers",
		},
		"unknown_provider": {
			form: url.Values{
				"provider": {"nope"},
				"domain":   {"example.com"},
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Choose one of the supported providers",
		},
		"missing_domain": {
			form: url.Values{
				"provider": {"cloudflare"},
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "the domain is required",
		},
		"unknown_field": {
			form: url.Values{
				"provider":      {"cloudflare"},
				"domain":        {"example.com"},
				"field_unknown": {"x"},
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "is not a setting of provider",
		},
		"invalid_number": {
			form: url.Values{
				"provider":  {"cloudflare"},
				"domain":    {"example.com"},
				"field_ttl": {"not a number"},
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "must be a number",
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			manager := newStubEntryManager()
			handler := newTestHandler(t, manager)

			recorder := postForm(t, handler, "/entries", testCase.form)
			assert.Equal(t, testCase.expectedStatus, recorder.Code)
			assert.Contains(t, recorder.Body.String(), testCase.expectedBody)
			assert.Empty(t, manager.savedEntry,
				"an invalid form must not be submitted")
		})
	}
}

func Test_entries_create_validationErrorKeepsForm(t *testing.T) {
	t.Parallel()

	manager := newStubEntryManager()
	manager.saveErr = reload.ErrInvalidEntry
	handler := newTestHandler(t, manager)

	form := url.Values{
		"provider":    {"cloudflare"},
		"domain":      {"example.com"},
		"field_email": {"me@example.com"},
	}
	recorder := postForm(t, handler, "/entries", form)
	require.Equal(t, http.StatusBadRequest, recorder.Code)

	body := recorder.Body.String()
	assert.Contains(t, body, "invalid settings entry")
	assert.Contains(t, body, `value="example.com"`)
	assert.Contains(t, body, `value="me@example.com"`)
}

func Test_entries_update(t *testing.T) {
	t.Parallel()

	manager := newStubEntryManager()
	handler := newTestHandler(t, manager)

	form := url.Values{
		"provider": {"duckdns"},
		"domain":   {"other.duckdns.org"},
		"field_token": {
			"11111111-1111-1111-1111-111111111111",
		},
	}
	recorder := postForm(t, handler, "/entries/1", form)
	require.Equal(t, http.StatusSeeOther, recorder.Code)

	require.NotNil(t, manager.savedIndex)
	assert.Equal(t, 1, *manager.savedIndex)
	entry := decodeEntry(t, manager.savedEntry)
	assert.Equal(t, "other.duckdns.org", entry["domain"])
}

func Test_entries_updateUnknownIndex(t *testing.T) {
	t.Parallel()

	handler := newTestHandler(t, newStubEntryManager())

	form := url.Values{
		"provider": {"duckdns"},
		"domain":   {"other.duckdns.org"},
	}
	recorder := postForm(t, handler, "/entries/7", form)
	assert.Equal(t, http.StatusNotFound, recorder.Code)
}

func Test_entries_update_editingDisabled(t *testing.T) {
	t.Parallel()

	manager := newStubEntryManager()
	manager.saveErr = reload.ErrEditingDisabled
	handler := newTestHandler(t, manager)

	form := url.Values{
		"provider":    {"duckdns"},
		"domain":      {"other.duckdns.org"},
		"field_token": {"11111111-1111-1111-1111-111111111111"},
	}
	recorder := postForm(t, handler, "/entries/0", form)
	assert.Equal(t, http.StatusConflict, recorder.Code)
}

func Test_entries_delete(t *testing.T) {
	t.Parallel()

	manager := newStubEntryManager()
	handler := newTestHandler(t, manager)

	recorder := postForm(t, handler, "/entries/0/delete", url.Values{})
	require.Equal(t, http.StatusSeeOther, recorder.Code)
	assert.Equal(t, "/entries", recorder.Header().Get("Location"))
	assert.Equal(t, 0, manager.deletedIndex)
	assert.Len(t, manager.entries, 1)
}

func Test_entries_delete_editingDisabled(t *testing.T) {
	t.Parallel()

	manager := newStubEntryManager()
	manager.deleteErr = reload.ErrEditingDisabled
	handler := newTestHandler(t, manager)

	recorder := postForm(t, handler, "/entries/0/delete", url.Values{})
	assert.Equal(t, http.StatusConflict, recorder.Code)
	assert.Contains(t, recorder.Body.String(), reload.ErrEditingDisabled.Error())
}

func Test_entries_rejectsCrossOriginAndNonFormRequests(t *testing.T) {
	t.Parallel()

	handler := newTestHandler(t, newStubEntryManager())

	t.Run("cross_origin", func(t *testing.T) {
		t.Parallel()

		request := httptest.NewRequest(http.MethodPost, "/entries",
			strings.NewReader("provider=duckdns&domain=example.duckdns.org"))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		request.Header.Set("Origin", "https://evil.example.com")
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		assert.Equal(t, http.StatusForbidden, recorder.Code)
	})

	t.Run("same_origin", func(t *testing.T) {
		t.Parallel()

		request := httptest.NewRequest(http.MethodPost, "/entries",
			strings.NewReader("provider=duckdns&domain=example.duckdns.org"))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		request.Header.Set("Origin", "http://example.com")
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		assert.Equal(t, http.StatusSeeOther, recorder.Code)
	})

	t.Run("json_body", func(t *testing.T) {
		t.Parallel()

		request := httptest.NewRequest(http.MethodPost, "/entries",
			strings.NewReader(`{"provider":"duckdns"}`))
		request.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		assert.Equal(t, http.StatusUnsupportedMediaType, recorder.Code)
	})
}

func Test_entries_rootURL(t *testing.T) {
	t.Parallel()

	manager := newStubEntryManager()
	handler := newHandler(context.Background(), "/ddns", stubDatabase{},
		stubUpdateForcer{}, &stubReloader{}, manager)

	recorder := get(t, handler, "/ddns/entries")
	require.Equal(t, http.StatusOK, recorder.Code)

	body := recorder.Body.String()
	assert.Contains(t, body, `href="/ddns/static/styles.css"`)
	assert.Contains(t, body, `href="/ddns/entries/0/edit"`)

	form := url.Values{
		"provider": {"duckdns"},
		"domain":   {"example.duckdns.org"},
	}
	postRecorder := postForm(t, handler, "/ddns/entries", form)
	require.Equal(t, http.StatusSeeOther, postRecorder.Code)
	assert.Equal(t, "/ddns/entries", postRecorder.Header().Get("Location"))
}

func Test_buildEntry(t *testing.T) {
	t.Parallel()

	provider := mustProvider(t, "cloudflare")

	testCases := map[string]struct {
		values   url.Values
		expected map[string]any
		errText  string
	}{
		"minimal": {
			values: url.Values{
				"domain":      {"example.com"},
				"field_token": {"t"},
			},
			expected: map[string]any{
				"provider": "cloudflare",
				"domain":   "example.com",
				"token":    "t",
			},
		},
		"boolean_true": {
			values: url.Values{
				"domain":        {"example.com"},
				"field_proxied": {"on"},
			},
			expected: map[string]any{
				"provider": "cloudflare",
				"domain":   "example.com",
				"proxied":  true,
			},
		},
		"boolean_false": {
			values: url.Values{
				"domain":        {"example.com"},
				"field_proxied": {"false"},
			},
			expected: map[string]any{
				"provider": "cloudflare",
				"domain":   "example.com",
				"proxied":  false,
			},
		},
		"number": {
			values: url.Values{
				"domain":    {"example.com"},
				"field_ttl": {"60"},
			},
			expected: map[string]any{
				"provider": "cloudflare",
				"domain":   "example.com",
				"ttl":      float64(60),
			},
		},
		"empty_optional_omitted": {
			values: url.Values{
				"domain":      {"example.com"},
				"field_ttl":   {""},
				"field_email": {""},
			},
			expected: map[string]any{
				"provider": "cloudflare",
				"domain":   "example.com",
			},
		},
		"missing_domain": {
			values:  url.Values{"field_token": {"t"}},
			errText: "the domain is required",
		},
		"unknown_field": {
			values: url.Values{
				"domain":      {"example.com"},
				"field_other": {"x"},
			},
			errText: `"other" is not a setting of provider "cloudflare"`,
		},
		"invalid_number": {
			values: url.Values{
				"domain":    {"example.com"},
				"field_ttl": {"abc"},
			},
			errText: "must be a number",
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			entry, err := buildEntry(testCase.values, provider)
			if testCase.errText != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), testCase.errText)
				assert.ErrorIs(t, err, errInvalidForm)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, testCase.expected, decodeEntry(t, entry))
		})
	}
}

func Test_buildEntry_jsonField(t *testing.T) {
	t.Parallel()

	provider := mustProvider(t, "gcp")

	entry, err := buildEntry(url.Values{
		"domain":            {"example.com"},
		"provider":          {"ignored"},
		"field_credentials": {`{"type":"service_account"}`},
	}, provider)
	require.NoError(t, err)

	assert.JSONEq(t,
		`{"provider":"gcp","domain":"example.com","credentials":{"type":"service_account"}}`,
		string(entry))

	_, err = buildEntry(url.Values{
		"domain":            {"example.com"},
		"field_credentials": {`not json`},
	}, provider)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be valid JSON")
}

func Test_parseEntryValues(t *testing.T) {
	t.Parallel()

	values, err := parseEntryValues(json.RawMessage(`{
		"provider": "gcp",
		"domain": "example.com",
		"ip_version": "ipv4",
		"ipv6_suffix": "::1",
		"ttl": 300,
		"proxied": true,
		"credentials": {"type": "service_account", "project": "p"}
	}`))
	require.NoError(t, err)

	assert.Equal(t, "gcp", values.Provider)
	assert.Equal(t, "example.com", values.Domain)
	assert.Equal(t, "ipv4", values.IPVersion)
	assert.Equal(t, "::1", values.IPv6Suffix)
	assert.Equal(t, "300", values.Fields["ttl"])
	assert.Equal(t, "true", values.Fields["proxied"])
	assert.Equal(t, "{\n  \"type\": \"service_account\",\n  \"project\": \"p\"\n}",
		values.Fields["credentials"])

	_, err = parseEntryValues(json.RawMessage(`[1,2]`))
	require.Error(t, err)
}

func Test_entryErrorStatus(t *testing.T) {
	t.Parallel()

	assert.Equal(t, http.StatusConflict, entryErrorStatus(reload.ErrEditingDisabled))
	assert.Equal(t, http.StatusNotFound, entryErrorStatus(reload.ErrEntryNotFound))
	assert.Equal(t, http.StatusBadRequest, entryErrorStatus(reload.ErrInvalidEntry))
	assert.Equal(t, http.StatusBadRequest, entryErrorStatus(errInvalidForm))
	assert.Equal(t, http.StatusInternalServerError, entryErrorStatus(
		errors.New("other")))
}

func Test_checkBrowserRequest(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		origin   string
		referer  string
		expected bool
	}{
		"no_headers":  {expected: true},
		"same_origin": {origin: "http://example.com", expected: true},
		"cross_origin": {
			origin: "http://evil.example.com",
		},
		"cross_port": {
			origin: "http://example.com:9999",
		},
		"invalid_origin": {
			origin: "://bad",
		},
		"referer_fallback": {
			referer:  "http://example.com/entries",
			expected: true,
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			request := httptest.NewRequest(http.MethodPost, "http://example.com/entries", nil)
			if testCase.origin != "" {
				request.Header.Set("Origin", testCase.origin)
			}
			if testCase.referer != "" {
				request.Header.Set("Referer", testCase.referer)
			}
			assert.Equal(t, testCase.expected, checkBrowserRequest(request))
		})
	}
}

func decodeEntry(t *testing.T, entry json.RawMessage) map[string]any {
	t.Helper()
	var object map[string]any
	require.NoError(t, json.Unmarshal(entry, &object))
	return object
}

func mustProvider(t *testing.T, name string) schema.Provider {
	t.Helper()
	provider, found := schema.Lookup(name)
	require.True(t, found, name)
	return provider
}
