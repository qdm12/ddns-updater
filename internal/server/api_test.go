package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"
	"time"

	"github.com/qdm12/ddns-updater/internal/constants"
	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/provider"
	providerconstants "github.com/qdm12/ddns-updater/internal/provider/constants"
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

func newTestRecord(t *testing.T, owner string, ipVersion ipversion.IPVersion,
	status models.Status, ips ...string,
) records.Record {
	t.Helper()
	settings := []byte(`{"token":"secret-token","zone_identifier":"secret-zone","proxied":true,"ttl":600}`)
	cloudflareProvider, err := provider.New(providerconstants.Cloudflare, settings, "example.com", owner,
		ipVersion, netip.Prefix{})
	require.NoError(t, err)

	successTime := time.Date(2026, 8, 12, 22, 13, 7, 0, time.UTC)
	events := make([]models.HistoryEvent, len(ips))
	for i, ip := range ips {
		events[i] = models.HistoryEvent{
			IP:   netip.MustParseAddr(ip),
			Time: successTime.Add(time.Duration(i-len(ips)+1) * time.Hour),
		}
	}
	record := records.New(cloudflareProvider, events)
	record.Status = status
	if status != constants.UNSET {
		record.Time = time.Date(2026, 8, 14, 9, 0, 0, 0, time.UTC)
	}
	return record
}

func Test_handlers_apiRecords(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		records    func(t *testing.T) []records.Record
		rootURL    string
		path       string
		statusCode int
		body       string
	}{
		"empty_database": {
			records:    func(*testing.T) []records.Record { return nil },
			path:       "/api/v1/records",
			statusCode: http.StatusOK,
			body:       `{"records":[],"summary":{"total":0,"healthy":true,"by_status":{}}}`,
		},
		"all_records": {
			records: func(t *testing.T) []records.Record {
				t.Helper()
				failed := newTestRecord(t, "www", ipversion.IP6, constants.FAIL)
				failed.Message = "bad authentication"
				return []records.Record{
					newTestRecord(t, "www", ipversion.IP4, constants.UPTODATE,
						"203.0.113.1", "203.0.113.2", "203.0.113.2", "203.0.113.4"),
					failed,
					newTestRecord(t, "api", ipversion.IP4, constants.UNSET),
				}
			},
			path:       "/api/v1/records",
			statusCode: http.StatusOK,
			body: `{"records":[
				{"domain":"example.com","owner":"www","fqdn":"www.example.com","ip_version":"ipv4","proxied":true,
				 "status":"up to date","current_ip":"203.0.113.4","previous_ips":["203.0.113.2","203.0.113.1"],
				 "previous_ips_total":2,"last_attempt":"2026-08-14T09:00:00Z","last_success":"2026-08-12T22:13:07Z",
				 "last_ban":null},
				{"domain":"example.com","owner":"www","fqdn":"www.example.com","ip_version":"ipv6","proxied":true,
				 "status":"failure","message":"bad authentication","previous_ips":[],"previous_ips_total":0,
				 "last_attempt":"2026-08-14T09:00:00Z","last_success":null,"last_ban":null},
				{"domain":"example.com","owner":"api","fqdn":"api.example.com","ip_version":"ipv4","proxied":true,
				 "status":"unset","previous_ips":[],"previous_ips_total":0,
				 "last_attempt":null,"last_success":null,"last_ban":null}],
				"summary":{"total":3,"healthy":false,"by_status":{"up to date":1,"failure":1,"unset":1}}}`,
		},
		"previous_ips_capped": {
			records: func(t *testing.T) []records.Record {
				t.Helper()
				return []records.Record{newTestRecord(t, "www", ipversion.IP4, constants.UPTODATE,
					"10.0.0.1", "10.0.0.2", "10.0.0.3", "10.0.0.4", "10.0.0.5", "10.0.0.6", "10.0.0.7",
					"10.0.0.8", "10.0.0.9", "10.0.0.10", "10.0.0.11", "10.0.0.12", "10.0.0.13")}
			},
			path:       "/api/v1/records",
			statusCode: http.StatusOK,
			body: `{"records":[
				{"domain":"example.com","owner":"www","fqdn":"www.example.com","ip_version":"ipv4","proxied":true,
				 "status":"up to date","current_ip":"10.0.0.13",
				 "previous_ips":["10.0.0.12","10.0.0.11","10.0.0.10","10.0.0.9","10.0.0.8",
				 "10.0.0.7","10.0.0.6","10.0.0.5","10.0.0.4","10.0.0.3"],
				 "previous_ips_total":12,"last_attempt":"2026-08-14T09:00:00Z","last_success":"2026-08-12T22:13:07Z",
				 "last_ban":null}],
				"summary":{"total":1,"healthy":true,"by_status":{"up to date":1}}}`,
		},
		"by_fqdn_case_insensitive": {
			records: func(t *testing.T) []records.Record {
				t.Helper()
				return []records.Record{
					newTestRecord(t, "www", ipversion.IP4, constants.SUCCESS, "203.0.113.4"),
					newTestRecord(t, "api", ipversion.IP4, constants.FAIL),
				}
			},
			path:       "/api/v1/records/WWW.example.com",
			statusCode: http.StatusOK,
			body: `{"records":[
				{"domain":"example.com","owner":"www","fqdn":"www.example.com","ip_version":"ipv4","proxied":true,
				 "status":"success","current_ip":"203.0.113.4","previous_ips":[],"previous_ips_total":0,
				 "last_attempt":"2026-08-14T09:00:00Z","last_success":"2026-08-12T22:13:07Z","last_ban":null}],
				"summary":{"total":1,"healthy":true,"by_status":{"success":1}}}`,
		},
		"by_fqdn_not_found": {
			records: func(t *testing.T) []records.Record {
				t.Helper()
				return []records.Record{newTestRecord(t, "www", ipversion.IP4, constants.SUCCESS)}
			},
			path:       "/api/v1/records/other.example.com",
			statusCode: http.StatusNotFound,
		},
		"root_url": {
			records: func(t *testing.T) []records.Record {
				t.Helper()
				return []records.Record{newTestRecord(t, "www", ipversion.IP4, constants.UNSET)}
			},
			rootURL:    "/ddns/",
			path:       "/ddns/api/v1/records/www.example.com",
			statusCode: http.StatusOK,
			body: `{"records":[
				{"domain":"example.com","owner":"www","fqdn":"www.example.com","ip_version":"ipv4","proxied":true,
				 "status":"unset","previous_ips":[],"previous_ips_total":0,
				 "last_attempt":null,"last_success":null,"last_ban":null}],
				"summary":{"total":1,"healthy":false,"by_status":{"unset":1}}}`,
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			database := fakeDatabase{records: testCase.records(t)}
			handler := newHandler(t.Context(), testCase.rootURL, database, fakeRunner{})
			recorder := httptest.NewRecorder()
			request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, testCase.path, nil)

			handler.ServeHTTP(recorder, request)

			require.Equal(t, testCase.statusCode, recorder.Code)
			body := recorder.Body.String()
			assert.NotContains(t, body, "secret-token")
			assert.NotContains(t, body, "secret-zone")
			if testCase.statusCode != http.StatusOK {
				return
			}
			assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
			assert.JSONEq(t, testCase.body, body)
		})
	}
}
