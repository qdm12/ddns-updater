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

// Test_Provider_Update_wildcardOwner is a regression test for the wildcard
// rapid update loop reported in
// https://github.com/qdm12/ddns-updater/issues/1039
//
// A wildcard owner "*" is handled like any other owner: the "*" appears
// literally in the rrset path, since the Hetzner Cloud API returns 404 for the
// percent encoded "%2A", and the asynchronous Zone action returned by the write
// is polled on the Zone actions endpoint. Getting either of these wrong made
// the provider consider the update failed and retry it on every cycle.
func Test_Provider_Update_wildcardOwner(t *testing.T) {
	t.Parallel()

	const (
		domain   = "example.com"
		actionID = 42
	)
	newIP := netip.AddrFrom4([4]byte{185, 28, 78, 5})

	runningBody := fmt.Sprintf(`{"action":{"id":%d,"status":"running"}}`, actionID)
	successBody := fmt.Sprintf(`{"action":{"id":%d,"status":"success"}}`, actionID)
	createdBody := fmt.Sprintf(`{"action":{"id":%d,"status":"running"},`+
		`"rrset":{"id":"*/A","name":"*","type":"A"}}`, actionID)
	staleRecordBody := `{"rrset":{"records":[{"value":"192.0.2.1"}]}}`
	notFoundBody := `{"error":{"code":"not_found","message":"rrset not found"}}`

	rrsetsPath := "/v1/zones/" + domain + "/rrsets"
	wildcardPath := rrsetsPath + "/*/A"
	setRecordsPath := wildcardPath + "/actions/set_records"
	zoneActionPath := fmt.Sprintf("/v1/zones/actions/%d", actionID)
	serverActionPath := fmt.Sprintf("/v1/servers/actions/%d", actionID)

	testCases := map[string]struct {
		// responses is keyed by "METHOD PATH".
		responses map[string]bodyResponse
		// wantRequests are the "METHOD PATH" keys expected, in order.
		wantRequests []string
	}{
		// The rrset does not exist yet, so getRecord gets a 404 and the
		// provider creates the rrset before polling the resulting Zone action.
		"create": {
			responses: map[string]bodyResponse{
				"GET " + wildcardPath:   {http.StatusNotFound, notFoundBody},
				"POST " + rrsetsPath:    {http.StatusCreated, createdBody},
				"GET " + zoneActionPath: {http.StatusOK, successBody},
			},
			wantRequests: []string{
				"GET " + wildcardPath,
				"POST " + rrsetsPath,
				"GET " + zoneActionPath,
			},
		},
		// The rrset exists with a stale IP address, so the provider replaces it
		// with set_records before polling the resulting Zone action.
		"set_records": {
			responses: map[string]bodyResponse{
				"GET " + wildcardPath:    {http.StatusOK, staleRecordBody},
				"POST " + setRecordsPath: {http.StatusCreated, runningBody},
				"GET " + zoneActionPath:  {http.StatusOK, successBody},
			},
			wantRequests: []string{
				"GET " + wildcardPath,
				"POST " + setRecordsPath,
				"GET " + zoneActionPath,
			},
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var gotRequests, gotRequestURIs []string
			client := newTestAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
				key := r.Method + " " + r.URL.Path
				gotRequests = append(gotRequests, key)
				gotRequestURIs = append(gotRequestURIs, r.RequestURI)
				response, ok := testCase.responses[key]
				if !ok {
					t.Errorf("unexpected request %s", key)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				w.WriteHeader(response.statusCode)
				_, _ = io.WriteString(w, response.body)
			})

			provider := &Provider{
				domain:    domain,
				owner:     "*",
				ipVersion: ipversion.IP4,
				token:     testToken,
				// The poll period is shortened to keep the test fast.
				actionPollPeriod: time.Millisecond,
			}

			gotIP, err := provider.Update(t.Context(), client, newIP)

			require.NoError(t, err)
			assert.Equal(t, newIP, gotIP)
			assert.Equal(t, testCase.wantRequests, gotRequests)

			joinedRequestURIs := strings.Join(gotRequestURIs, "\n")
			// The wildcard must not be percent encoded, since the Hetzner
			// Cloud API returns 404 for "%2A".
			assert.NotContains(t, joinedRequestURIs, "%2A")
			assert.NotContains(t, joinedRequestURIs, "%2a")
			// The Zone action must not be polled on the Server actions
			// endpoint, which is what issue #1136 was about.
			assert.NotContains(t, joinedRequestURIs, serverActionPath)
		})
	}
}
