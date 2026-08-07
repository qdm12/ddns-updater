package hetznercloud

import (
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testToken is a dummy API token used across the hetznercloud provider tests.
const testToken = "token"

// recordingTransport records every request URL it sees and returns canned
// responses keyed by "METHOD PATH". This lets us assert how the wildcard
// owner "*" is turned into request URLs.
type recordingTransport struct {
	t         *testing.T
	responses map[string]bodyResponse
	gotURLs   []string
}

func (rt *recordingTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	rt.gotURLs = append(rt.gotURLs, request.URL.String())
	key := request.Method + " " + request.URL.Path
	response, ok := rt.responses[key]
	if !ok {
		rt.t.Errorf("unexpected request %s", key)
		return &http.Response{
			StatusCode: http.StatusNotImplemented,
			Body:       io.NopCloser(strings.NewReader("{}")),
		}, nil
	}
	return &http.Response{
		StatusCode: int(response.statusCode),
		Body:       io.NopCloser(strings.NewReader(response.body)),
	}, nil
}

// Test_Provider_Update_wildcard is a regression test for the wildcard
// rapid-update loop reported in
// https://github.com/qdm12/ddns-updater/issues/1039 (comment by Daxolion).
//
// A wildcard owner "*" must be treated like any other owner: the "*" has to
// appear literally in the rrset path (the Hetzner Cloud API returns 404 for
// the percent-encoded "%2A"), and the asynchronous zone action returned by the
// write must be polled on the zone actions endpoint. Getting either wrong made
// the provider believe the update failed and retry it on every cycle.
func Test_Provider_Update_wildcard(t *testing.T) {
	t.Parallel()

	const (
		domain   = "example.com"
		actionID = 12345
	)
	newIP := netip.AddrFrom4([4]byte{185, 28, 78, 5})

	runningBody := fmt.Sprintf(`{"action":{"id":%d,"status":"running"}}`, actionID)
	successBody := fmt.Sprintf(`{"action":{"id":%d,"status":"success"}}`, actionID)
	createdActionBody := fmt.Sprintf(`{"action":{"id":%d,"status":"running"},`+
		`"rrset":{"id":"*/A","name":"*","type":"A"}}`, actionID)

	rrsetPath := "/v1/zones/" + domain + "/rrsets"
	wildcardRRSetPath := rrsetPath + "/*/A"
	zoneActionPath := fmt.Sprintf("/v1/zones/actions/%d", actionID)
	serverActionPath := fmt.Sprintf("/v1/servers/actions/%d", actionID)

	testCases := map[string]struct {
		responses map[string]bodyResponse
		// wantURLs are URLs that must have been requested.
		wantURLs []string
	}{
		// The record does not exist yet: getRecord returns 404, so the provider
		// creates the rrset and then polls the resulting zone action.
		"create": {
			responses: map[string]bodyResponse{
				"GET " + wildcardRRSetPath: {http.StatusNotFound, `{"error":{"code":"not_found"}}`},
				"POST " + rrsetPath:        {http.StatusCreated, createdActionBody},
				"GET " + zoneActionPath:    {http.StatusOK, successBody},
			},
			wantURLs: []string{
				"https://api.hetzner.cloud" + wildcardRRSetPath,
				"https://api.hetzner.cloud" + rrsetPath,
				"https://api.hetzner.cloud" + zoneActionPath,
			},
		},
		// The record exists with a different IP: getRecord returns 200 and a
		// stale value, so the provider replaces it via set_records and then
		// polls the resulting zone action.
		"set_records": {
			responses: map[string]bodyResponse{
				"GET " + wildcardRRSetPath: {
					http.StatusOK,
					`{"rrset":{"records":[{"value":"192.0.2.1"}]}}`,
				},
				"POST " + wildcardRRSetPath + "/actions/set_records": {http.StatusCreated, runningBody},
				"GET " + zoneActionPath:                              {http.StatusOK, successBody},
			},
			wantURLs: []string{
				"https://api.hetzner.cloud" + wildcardRRSetPath,
				"https://api.hetzner.cloud" + wildcardRRSetPath + "/actions/set_records",
				"https://api.hetzner.cloud" + zoneActionPath,
			},
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			transport := &recordingTransport{t: t, responses: testCase.responses}
			client := &http.Client{Transport: transport}

			provider := &Provider{
				domain:           domain,
				owner:            "*",
				ipVersion:        ipversion.IP4,
				token:            testToken,
				actionPollPeriod: time.Millisecond,
			}

			gotIP, err := provider.Update(t.Context(), client, newIP)
			require.NoError(t, err)
			assert.Equal(t, newIP, gotIP)

			for _, wantURL := range testCase.wantURLs {
				assert.Contains(t, transport.gotURLs, wantURL)
			}

			joinedURLs := strings.Join(transport.gotURLs, "\n")
			// The wildcard must never be percent-encoded: the Hetzner Cloud API
			// returns 404 for "%2A".
			assert.NotContains(t, joinedURLs, "%2A")
			assert.NotContains(t, joinedURLs, "%2a")
			// The zone action must never be polled on the server actions
			// endpoint (regression from issue #1136, which manifested as the
			// wildcard rapid-update loop in issue #1039).
			assert.NotContains(t, joinedURLs, serverActionPath)
		})
	}
}
