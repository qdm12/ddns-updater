package allinkl

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"

	ddnserrors "github.com/qdm12/ddns-updater/internal/provider/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// rewriteTransport redirects all requests (to dyndns.kasserver.com) to the test
// server, without making the provider code test-aware.
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

	testCases := map[string]struct {
		ipv6       bool
		statusCode int
		body       string
		errMessage string
		errIs      error
	}{
		// All-Inkl responds with "good" followed by a single space and never
		// writes the IP address back, see
		// https://github.com/qdm12/ddns-updater/issues/1229.
		"good_ipv4": {
			body: "good ",
		},
		"nochg_ipv4": {
			body: "nochg ",
		},
		"good_ipv6": {
			ipv6: true,
			body: "good ",
		},
		// An unexpected IP address is ignored, only the status matters.
		"good_unexpected_ip_written_back": {
			body: "good 203.0.113.46",
		},
		"error_911": {
			body:       "911",
			errMessage: "server side DNS error",
		},
		"error_abuse": {
			body:  "abuse",
			errIs: ddnserrors.ErrBannedAbuse,
		},
		"error_donator": {
			body:       "!donator",
			errMessage: "feature is not available to the user",
		},
		"error_badagent": {
			body:       "badagent",
			errMessage: "user agent is banned",
		},
		"error_badauth": {
			body:       "badauth",
			errMessage: "bad authentication",
		},
		"error_nohost": {
			body:       "nohost",
			errMessage: "hostname does not exist",
		},
		"unknown": {
			body:       "this is not a status",
			errMessage: "unknown response received",
		},
		"empty_body": {
			errIs: ddnserrors.ErrReceivedNoResult,
		},
		"internal_server_error": {
			statusCode: http.StatusInternalServerError,
			body:       "good 203.0.113.45",
			errMessage: "HTTP status is not valid: 500",
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ip := netip.MustParseAddr("203.0.113.45")
			if testCase.ipv6 {
				ip = netip.MustParseAddr("2001:db8::45")
			}

			statusCode := testCase.statusCode
			if statusCode == 0 {
				statusCode = http.StatusOK
			}

			var receivedRequest *http.Request
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedRequest = r
				w.WriteHeader(statusCode)
				_, _ = io.WriteString(w, testCase.body)
			}))
			defer server.Close()

			client := &http.Client{Transport: rewriteTransport{
				host: strings.TrimPrefix(server.URL, "http://"),
				base: http.DefaultTransport,
			}}

			provider := &Provider{
				domain:   "example.com",
				owner:    "sub",
				username: "dyn123456",
				password: "password",
			}

			newIP, err := provider.Update(t.Context(), client, ip)

			switch {
			case testCase.errIs != nil:
				require.ErrorIs(t, err, testCase.errIs)
			case testCase.errMessage != "":
				require.ErrorContains(t, err, testCase.errMessage)
			default:
				require.NoError(t, err)
				// The IP address is never written back, so the updated IP address
				// is the one sent.
				assert.Equal(t, ip, newIP)
			}

			require.NotNil(t, receivedRequest)
			assert.Equal(t, "sub.example.com", receivedRequest.URL.Query().Get("host"))
			switch {
			case ip.Is4():
				assert.Equal(t, ip.String(), receivedRequest.URL.Query().Get("myip"))
				assert.Empty(t, receivedRequest.URL.Query().Get("myip6"))
			default:
				assert.Equal(t, ip.String(), receivedRequest.URL.Query().Get("myip6"))
				assert.Empty(t, receivedRequest.URL.Query().Get("myip"))
			}
			receivedUsername, receivedPassword, ok := receivedRequest.BasicAuth()
			require.True(t, ok)
			assert.Equal(t, "dyn123456", receivedUsername)
			assert.Equal(t, "password", receivedPassword)
		})
	}
}
