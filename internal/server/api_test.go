package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"
	"time"

	"github.com/qdm12/ddns-updater/internal/constants"
	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/provider"
	providerconstants "github.com/qdm12/ddns-updater/internal/provider/constants"
	"github.com/qdm12/ddns-updater/internal/provider/providers/cloudflare"
	"github.com/qdm12/ddns-updater/internal/records"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeDatabase struct {
	records []records.Record
}

func (f fakeDatabase) SelectAll() []records.Record { return f.records }

type fakeRunner struct{}

func (fakeRunner) ForceUpdate(_ context.Context) []error { return nil }

//nolint:ireturn
func newTestProvider(t *testing.T) provider.Provider {
	t.Helper()
	settings := []byte(`{"token":"secret-token","zone_identifier":"secret-zone","proxied":true,"ttl":600}`)
	p, err := provider.New(providerconstants.Cloudflare, settings, "example.com", "www",
		ipversion.IP4, netip.Prefix{})
	require.NoError(t, err)
	return p
}

func testHandler(t *testing.T, recordsList []records.Record) http.Handler {
	t.Helper()
	return newHandler(context.Background(), "", fakeDatabase{records: recordsList},
		fakeRunner{}, models.BuildInformation{Version: "v1.2.3", Commit: "abcdef1", Created: "2026-01-01"})
}

func Test_apiRecords(t *testing.T) {
	t.Parallel()

	lastAttempt := time.Date(2026, 8, 14, 9, 0, 0, 0, time.UTC)
	lastSuccess := time.Date(2026, 8, 12, 22, 13, 7, 0, time.UTC)

	record := records.New(newTestProvider(t), []models.HistoryEvent{
		{IP: netip.MustParseAddr("203.0.113.1"), Time: lastSuccess.Add(-time.Hour)},
		{IP: netip.MustParseAddr("203.0.113.2"), Time: lastSuccess.Add(-time.Minute)},
		{IP: netip.MustParseAddr("203.0.113.4"), Time: lastSuccess},
	})
	record.Status = constants.UPTODATE
	record.Time = lastAttempt

	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/records", nil)
	testHandler(t, []records.Record{record}).ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))

	var response apiRecordsResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))

	require.Len(t, response.Records, 1)
	got := response.Records[0]

	assert.Equal(t, "example.com", got.Domain)
	assert.Equal(t, "www", got.Owner)
	assert.Equal(t, "www.example.com", got.FQDN)
	assert.Equal(t, "cloudflare", got.Provider)
	assert.Equal(t, "ipv4", got.IPVersion)
	assert.True(t, got.Proxied)
	assert.Equal(t, "up to date", got.Status)
	assert.Equal(t, "203.0.113.4", got.CurrentIP)

	// Full history, unlike the HTML view which truncates to two.
	assert.Equal(t, []string{"203.0.113.2", "203.0.113.1"}, got.PreviousIPs)

	require.NotNil(t, got.LastAttempt)
	assert.Equal(t, lastAttempt, got.LastAttempt.UTC())
	require.NotNil(t, got.LastSuccess)
	assert.Equal(t, lastSuccess, got.LastSuccess.UTC())
	assert.Nil(t, got.LastBan)

	assert.Equal(t, 1, response.Summary.Total)
	assert.Equal(t, map[string]int{"up to date": 1}, response.Summary.ByStatus)
}

// Test_apiRecords_noCredentials is the important one: provider settings carry
// secrets, so the response must never contain them.
func Test_apiRecords_noCredentials(t *testing.T) {
	t.Parallel()

	record := records.New(newTestProvider(t), nil)
	record.Status = constants.SUCCESS

	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/records", nil)
	testHandler(t, []records.Record{record}).ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	body := recorder.Body.String()
	assert.NotContains(t, body, "secret-token")
	assert.NotContains(t, body, "secret-zone")
}

func Test_apiRecords_emptyDatabase(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/records", nil)
	testHandler(t, nil).ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.JSONEq(t, `{"records":[],"summary":{"total":0,"by_status":{}}}`, recorder.Body.String())
}

// Test_apiRecords_unsetRecord covers a record which has never updated, where
// the IP and both timestamps are zero values.
func Test_apiRecords_unsetRecord(t *testing.T) {
	t.Parallel()

	record := records.New(newTestProvider(t), nil)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/records", nil)
	testHandler(t, []records.Record{record}).ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)

	var response apiRecordsResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Len(t, response.Records, 1)
	got := response.Records[0]

	assert.Equal(t, "unset", got.Status)
	assert.Empty(t, got.CurrentIP)
	assert.Empty(t, got.PreviousIPs)
	assert.Nil(t, got.LastAttempt)
	assert.Nil(t, got.LastSuccess)
}

func Test_apiVersion(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/version", nil)
	testHandler(t, nil).ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.JSONEq(t, `{"version":"v1.2.3","commit":"abcdef1","buildDate":"2026-01-01"}`,
		recorder.Body.String())
}

// Test_apiRecords_rootURL checks the endpoints honor the configured root URL,
// the same way the index and update routes do.
func Test_apiRecords_rootURL(t *testing.T) {
	t.Parallel()

	handler := newHandler(context.Background(), "/ddns", fakeDatabase{}, fakeRunner{},
		models.BuildInformation{})

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/ddns/api/v1/records", nil))
	assert.Equal(t, http.StatusOK, recorder.Code)

	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/records", nil))
	assert.Equal(t, http.StatusNotFound, recorder.Code)
}

// Test_providerName covers the decorator added in provider.New, since the
// concrete providers only expose their name pre-formatted for HTML.
func Test_providerName(t *testing.T) {
	t.Parallel()

	p := newTestProvider(t)
	namer, ok := p.(provider.Namer)
	require.True(t, ok, "provider built through provider.New must implement Namer")
	assert.Equal(t, providerconstants.Cloudflare, namer.Name())

	// The concrete type on its own does not, which is why the decorator exists.
	var concrete any = &cloudflare.Provider{}
	_, ok = concrete.(provider.Namer)
	assert.False(t, ok)
}
