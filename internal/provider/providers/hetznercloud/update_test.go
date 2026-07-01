package hetznercloud

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"

	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// rewriteTransport leitet alle Anfragen (an api.hetzner.cloud) auf den
// Test-Server um, ohne den Provider-Code für Tests anpassen zu müssen.
type rewriteTransport struct {
	host string
	base http.RoundTripper
}

func (t rewriteTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	request.URL.Scheme = "http"
	request.URL.Host = t.host
	return t.base.RoundTrip(request)
}

func Test_Update_routing(t *testing.T) {
	t.Parallel()

	const zoneID = "zone123"
	ip := netip.MustParseAddr("2.2.2.2")

	testCases := map[string]struct {
		rrsetsResponse string // Antwort auf GET .../rrsets
		wantMethod     string
		wantPath       string // erwarteter Schreib-Endpoint
		writeResponse  string
	}{
		"existing_record_uses_set_records_action": {
			rrsetsResponse: `{"rrsets":[{"name":"git","type":"A","records":[{"value":"1.1.1.1"}]}]}`,
			wantMethod:     http.MethodPost,
			wantPath:       "/v1/zones/zone123/rrsets/git/A/actions/set_records",
			writeResponse:  `{"action":{"status":"success"}}`,
		},
		"missing_record_creates_rrset": {
			rrsetsResponse: `{"rrsets":[]}`,
			wantMethod:     http.MethodPost,
			wantPath:       "/v1/zones/zone123/rrsets",
			writeResponse:  `{"rrset":{"records":[{"value":"2.2.2.2"}]}}`,
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var writePath, writeMethod string
			handler := func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodGet && r.URL.Path == "/v1/zones/"+zoneID+"/rrsets" {
					_, _ = io.WriteString(w, testCase.rrsetsResponse)
					return
				}
				// jeder andere Aufruf ist der Schreib-Endpoint
				writeMethod = r.Method
				writePath = r.URL.Path
				w.WriteHeader(http.StatusOK)
				_, _ = io.WriteString(w, testCase.writeResponse)
			}
			server := httptest.NewServer(http.HandlerFunc(handler))
			defer server.Close()

			client := &http.Client{Transport: rewriteTransport{
				host: strings.TrimPrefix(server.URL, "http://"),
				base: http.DefaultTransport,
			}}

			provider := &Provider{
				domain:         "example.com",
				owner:          "git",
				ipVersion:      ipversion.IP4,
				token:          "token",
				zoneIdentifier: zoneID,
				ttl:            3600,
			}

			newIP, err := provider.Update(context.Background(), client, ip)
			require.NoError(t, err)
			assert.Equal(t, ip, newIP)
			assert.Equal(t, testCase.wantMethod, writeMethod)
			assert.Equal(t, testCase.wantPath, writePath)
		})
	}
}
