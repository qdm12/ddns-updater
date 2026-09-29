package enum

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"sync"
	"testing"

	"github.com/qdm12/ddns-updater/internal/provider/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_validateSettings(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		domain     string
		apiKey     string
		projectID  string
		ttl        uint32
		errWrapped error
		errMessage string
	}{
		"valid": {
			domain:    "domain.com",
			apiKey:    "key",
			projectID: "proj-id",
		},
		"empty_api_key": {
			domain:     "domain.com",
			projectID:  "proj-id",
			errWrapped: errors.ErrAPIKeyNotSet,
			errMessage: "API key is not set",
		},
		"empty_project_id": {
			domain:     "domain.com",
			apiKey:     "key",
			errWrapped: errors.ErrProjectIDNotSet,
			errMessage: "project id is not set",
		},
		"ttl_too_low": {
			domain:     "domain.com",
			apiKey:     "key",
			projectID:  "proj-id",
			ttl:        30,
			errWrapped: errors.ErrTTLTooLow,
			errMessage: "TTL is too low: 30 must be at least 60 seconds",
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := validateSettings(testCase.domain, testCase.apiKey, testCase.projectID, testCase.ttl)

			assert.ErrorIs(t, err, testCase.errWrapped)
			if testCase.errWrapped != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
		})
	}
}

type rewriteTransport struct {
	host string
	base http.RoundTripper
}

func (t rewriteTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	request.URL.Scheme = "http"
	request.URL.Host = t.host
	return t.base.RoundTrip(request)
}

type apiResponse struct {
	statusCode int
	body       string
}

func Test_Update(t *testing.T) {
	t.Parallel()

	zoneFound := apiResponse{http.StatusOK, `{"zone":{"id":"dnszone-1","name":"domain.com."}}`}
	const getZone = `{"projectId":"proj-id","name":"domain.com"}`
	const recordSet = `"projectId":"proj-id","zoneId":"dnszone-1","name":"home.domain.com","type":"A"`

	testCases := map[string]struct {
		responses map[string]apiResponse
		wantCalls map[string]string
		wantErr   error
	}{
		"create_missing_record_set": {
			responses: map[string]apiResponse{
				"GetZoneByName":   zoneFound,
				"GetRecordSet":    {http.StatusNotFound, `{"code":"not_found","message":"record set not found"}`},
				"CreateRecordSet": {http.StatusOK, `{}`},
			},
			wantCalls: map[string]string{
				"GetZoneByName":   getZone,
				"GetRecordSet":    "{" + recordSet + "}",
				"CreateRecordSet": "{" + recordSet + `,"records":[{"content":"1.2.3.4"}]}`,
			},
		},
		"update_keeps_existing_ttl": {
			responses: map[string]apiResponse{
				"GetZoneByName":   zoneFound,
				"GetRecordSet":    {http.StatusOK, `{"recordSet":{"ttl":600,"records":[{"content":"5.6.7.8"}]}}`},
				"UpdateRecordSet": {http.StatusOK, `{}`},
			},
			wantCalls: map[string]string{
				"GetZoneByName":   getZone,
				"GetRecordSet":    "{" + recordSet + "}",
				"UpdateRecordSet": "{" + recordSet + `,"ttl":600,"records":[{"content":"1.2.3.4"}]}`,
			},
		},
		"up_to_date": {
			responses: map[string]apiResponse{
				"GetZoneByName": zoneFound,
				"GetRecordSet":  {http.StatusOK, `{"recordSet":{"ttl":300,"records":[{"content":"1.2.3.4"}]}}`},
			},
			wantCalls: map[string]string{
				"GetZoneByName": getZone,
				"GetRecordSet":  "{" + recordSet + "}",
			},
		},
		"zone_not_found": {
			responses: map[string]apiResponse{
				"GetZoneByName": {http.StatusNotFound, `{"code":"not_found","message":"zone not found"}`},
			},
			wantCalls: map[string]string{
				"GetZoneByName": getZone,
			},
			wantErr: errors.ErrZoneNotFound,
		},
		"unauthenticated": {
			responses: map[string]apiResponse{
				"GetZoneByName": {http.StatusUnauthorized, `{"code":"unauthenticated","message":"invalid token"}`},
			},
			wantCalls: map[string]string{
				"GetZoneByName": getZone,
			},
			wantErr: errors.ErrAuth,
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var mutex sync.Mutex
			calls := map[string]string{}
			handler := func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "Bearer key", r.Header.Get("Authorization"))
				method := strings.TrimPrefix(r.URL.Path, "/enum.api.v1.DnsService/")
				body, _ := io.ReadAll(r.Body)

				mutex.Lock()
				calls[method] = strings.TrimSpace(string(body))
				mutex.Unlock()

				response := testCase.responses[method]
				w.WriteHeader(response.statusCode)
				_, _ = io.WriteString(w, response.body)
			}
			server := httptest.NewServer(http.HandlerFunc(handler))
			defer server.Close()

			client := &http.Client{Transport: rewriteTransport{
				host: strings.TrimPrefix(server.URL, "http://"),
				base: http.DefaultTransport,
			}}

			provider := &Provider{
				domain:    "domain.com",
				owner:     "home",
				apiKey:    "key",
				projectID: "proj-id",
			}
			ip := netip.MustParseAddr("1.2.3.4")

			newIP, err := provider.Update(context.Background(), client, ip)

			assert.Equal(t, testCase.wantCalls, calls)
			if testCase.wantErr != nil {
				require.ErrorIs(t, err, testCase.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, ip, newIP)
		})
	}
}
