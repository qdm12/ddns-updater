package dyn

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"

	"github.com/qdm12/ddns-updater/internal/provider/errors"
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
		errWrapped error
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
			errWrapped: errors.ErrHostnameNotExists,
		},
		"badrequest": {
			body:       "badrequest",
			errWrapped: errors.ErrBadRequest,
		},
		"unknown": {
			body:       "unexpected response",
			errWrapped: errors.ErrUnknownResponse,
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

			if testCase.errWrapped != nil {
				require.ErrorIs(t, err, testCase.errWrapped)
			} else {
				require.NoError(t, err)
				assert.Equal(t, testCase.newIP, newIP)
			}
		})
	}
}
