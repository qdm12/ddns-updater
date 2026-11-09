package opendns

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"

	ddnserrors "github.com/qdm12/ddns-updater/internal/provider/errors"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// rewriteTransport redirects all requests (to updates.opendns.com) to the test
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

func Test_validateSettings(t *testing.T) {
	t.Parallel()

	const (
		domain   = "example.com"
		owner    = "sub"
		username = "username"
		password = "password"
	)

	testCases := map[string]struct {
		domain     string
		owner      string
		username   string
		password   string
		errMessage string
	}{
		"valid": {
			domain:   domain,
			owner:    owner,
			username: username,
			password: password,
		},
		"invalid_domain": {
			domain:     "exa mple.com",
			owner:      owner,
			username:   username,
			password:   password,
			errMessage: "domain is not valid",
		},
		"wildcard_owner": {
			domain:     domain,
			owner:      "*",
			username:   username,
			password:   password,
			errMessage: "owner cannot be \"*\"",
		},
		"username_missing": {
			domain:     domain,
			owner:      owner,
			password:   password,
			errMessage: "username is not set",
		},
		"password_missing": {
			domain:     domain,
			owner:      owner,
			username:   username,
			errMessage: "password is not set",
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := validateSettings(testCase.domain, testCase.owner,
				testCase.username, testCase.password)

			if testCase.errMessage == "" {
				assert.NoError(t, err)
				return
			}
			assert.ErrorContains(t, err, testCase.errMessage)
		})
	}
}

// Test_New covers the optional hostname setting, naming the network declared in
// the OpenDNS account, and defaulting to the domain name of the record.
func Test_New(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		data             string
		expectedHostname string
		errMessage       string
	}{
		// The network name defaults to the domain name of the record, which is
		// valid when the network was named after a domain name.
		"hostname_unset": {
			data:             `{"username": "username", "password": "password"}`,
			expectedHostname: "sub.example.com",
		},
		"hostname_set": {
			data:             `{"username": "username", "password": "password", "hostname": "Office"}`,
			expectedHostname: "Office",
		},
		"username_unset": {
			data:       `{"password": "password"}`,
			errMessage: "username is not set",
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			provider, err := New(json.RawMessage(testCase.data), "example.com", "sub",
				ipversion.IP4, netip.Prefix{})

			if testCase.errMessage != "" {
				require.ErrorContains(t, err, testCase.errMessage)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, testCase.expectedHostname, provider.hostname)
			assert.Equal(t, "example.com", provider.Domain())
			assert.Equal(t, "sub", provider.Owner())
			assert.True(t, provider.Proxied())
		})
	}
}

func Test_Update(t *testing.T) {
	t.Parallel()

	const (
		domain      = "example.org"
		owner       = "sub"
		username    = "username"
		password    = "password"
		networkName = "Home"
	)

	ip := netip.MustParseAddr("203.0.113.45")

	testCases := map[string]struct {
		statusCode int
		body       string
		newIP      netip.Addr
		errMessage string
		errIs      error
	}{
		"good": {
			body:  "good 203.0.113.45",
			newIP: ip,
		},
		// A nochg response means the record is already up to date, which is a
		// success and not an error.
		"nochg": {
			body:  "nochg 203.0.113.45",
			newIP: ip,
		},
		"good_mismatching_ip": {
			body:       "good 203.0.113.46",
			errMessage: "mismatching IP address received",
		},
		"nochg_malformed_ip": {
			body:       "nochg not-an-ip",
			errMessage: "malformed IP address received",
		},
		"noauth": {
			body:       "noauth",
			errMessage: "update is not authorized",
		},
		"nohost": {
			body:       "nohost",
			errMessage: "hostname does not exist: check the network name",
		},
		"911": {
			body:       "911",
			errMessage: "server side DNS error",
		},
		"badauth": {
			body:       "badauth",
			errMessage: "bad authentication",
		},
		"badagent": {
			body:       "badagent",
			errMessage: "user agent is banned",
		},
		"abuse": {
			body:  "abuse",
			errIs: ddnserrors.ErrBannedAbuse,
		},
		"dnserr": {
			body:       "dnserr",
			errMessage: "server side DNS error",
		},
		"unknown": {
			body:       "this is not a status",
			errMessage: "unknown response received",
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
				domain:   domain,
				owner:    owner,
				username: username,
				password: password,
				hostname: networkName,
			}

			newIP, err := provider.Update(t.Context(), client, ip)

			switch {
			case testCase.errIs != nil:
				require.ErrorIs(t, err, testCase.errIs)
			case testCase.errMessage != "":
				require.ErrorContains(t, err, testCase.errMessage)
			default:
				require.NoError(t, err)
				assert.Equal(t, testCase.newIP, newIP)
			}

			// The network name declared in the account is sent as it is, with
			// no owner prefixed to it.
			require.NotNil(t, receivedRequest)
			assert.Equal(t, networkName, receivedRequest.URL.Query().Get("hostname"))
			assert.Equal(t, ip.String(), receivedRequest.URL.Query().Get("myip"))
			receivedUsername, receivedPassword, ok := receivedRequest.BasicAuth()
			require.True(t, ok)
			assert.Equal(t, username, receivedUsername)
			assert.Equal(t, password, receivedPassword)
		})
	}
}
