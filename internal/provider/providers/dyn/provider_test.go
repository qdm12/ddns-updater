package dyn

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// rewriteTransport redirects all requests (to members.dyndns.org) to the
// test server, without making the provider code test-aware.
type rewriteTransport struct {
	host string
	base http.RoundTripper
}

func (t rewriteTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	request.URL.Scheme = "http"
	request.URL.Host = t.host
	return t.base.RoundTrip(request)
}

func Test_Update(t *testing.T) {
	t.Parallel()

	ip := netip.MustParseAddr("203.218.80.114")

	testCases := map[string]struct {
		body       string
		newIP      netip.Addr
		errMessage string
	}{
		"good": {
			body:  "good 203.218.80.114",
			newIP: ip,
		},
		"nochg": {
			body:  "nochg 203.218.80.114",
			newIP: ip,
		},
		"notfqdn": {
			body:       "notfqdn",
			errMessage: "hostname does not exist",
		},
		"badrequest": {
			body:       "badrequest",
			errMessage: "bad request sent",
		},
		"unknown": {
			body:       "unexpected response",
			errMessage: "unknown response received",
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = io.WriteString(w, testCase.body)
			}))
			defer server.Close()

			client := &http.Client{Transport: rewriteTransport{
				host: strings.TrimPrefix(server.URL, "http://"),
				base: http.DefaultTransport,
			}}

			provider := &Provider{
				domain:    "domain.com",
				owner:     "",
				username:  "user",
				clientKey: "key",
			}

			newIP, err := provider.Update(context.Background(), client, ip)

			if testCase.errMessage != "" {
				require.ErrorContains(t, err, testCase.errMessage)
			} else {
				require.NoError(t, err)
				assert.Equal(t, testCase.newIP, newIP)
			}
		})
	}
}
