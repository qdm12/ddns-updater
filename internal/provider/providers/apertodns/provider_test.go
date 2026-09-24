package apertodns

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"

	ddnserrors "github.com/qdm12/ddns-updater/internal/provider/errors"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_responseError(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		code       string
		message    string
		errWrapped error
		errMessage string
	}{
		"invalid_token": {
			code:       "invalid_token",
			message:    "token is invalid",
			errWrapped: ddnserrors.ErrAuth,
			errMessage: "bad authentication: token is invalid",
		},
		"unauthorized": {
			code:       "unauthorized",
			message:    "unauthorized",
			errWrapped: ddnserrors.ErrAuth,
			errMessage: "bad authentication: unauthorized",
		},
		"hostname_not_found": {
			code:       "hostname_not_found",
			message:    "no such hostname",
			errWrapped: ddnserrors.ErrHostnameNotExists,
			errMessage: "hostname does not exist: no such hostname",
		},
		"not_fqdn": {
			code:       "not_fqdn",
			message:    "hostname is not a valid FQDN",
			errWrapped: ddnserrors.ErrBadRequest,
			errMessage: "bad request sent: hostname is not a valid FQDN",
		},
		"invalid_ip": {
			code:       "invalid_ip",
			message:    "ip is invalid",
			errWrapped: ddnserrors.ErrBadRequest,
			errMessage: "bad request sent: ip is invalid",
		},
		"rate_limited": {
			code:       "rate_limited",
			message:    "too many requests",
			errWrapped: ddnserrors.ErrBannedAbuse,
			errMessage: "banned due to abuse: too many requests",
		},
		"server_error": {
			code:       "server_error",
			message:    "internal error",
			errWrapped: ddnserrors.ErrUnknownResponse,
			errMessage: "unknown response received: internal error",
		},
		"unknown_code": {
			code:       "teapot",
			message:    "I'm a teapot",
			errWrapped: ddnserrors.ErrUnsuccessful,
			errMessage: "unsuccessful result: teapot: I'm a teapot",
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := responseError(testCase.code, testCase.message)

			assert.ErrorIs(t, err, testCase.errWrapped)
			assert.EqualError(t, err, testCase.errMessage)
		})
	}
}

func Test_Provider_Update(t *testing.T) {
	t.Parallel()

	const token = "test-token"
	ipv4 := netip.AddrFrom4([4]byte{1, 2, 3, 4})
	ipv6 := netip.MustParseAddr("2001:db8::1")

	testCases := map[string]struct {
		handler    http.HandlerFunc
		ipVersion  ipversion.IPVersion
		ip         netip.Addr
		expectedIP netip.Addr
		errWrapped error
		errMessage string
	}{
		"success_ipv4": {
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "/.well-known/apertodns/v1/update", r.URL.Path)
				assert.Equal(t, "Bearer "+token, r.Header.Get("Authorization"))

				var body map[string]any
				err := json.NewDecoder(r.Body).Decode(&body)
				assert.NoError(t, err)
				assert.Equal(t, "host.example.com", body["hostname"])
				assert.Equal(t, "1.2.3.4", body["ipv4"])
				_, hasIPv6 := body["ipv6"]
				assert.False(t, hasIPv6, "request must not contain the ipv6 key")

				fmt.Fprint(w, `{"success":true,"data":{"hostname":"host.example.com","ipv4":"1.2.3.4"}}`)
			},
			ipVersion:  ipversion.IP4,
			ip:         ipv4,
			expectedIP: ipv4,
		},
		"success_ipv6": {
			handler: func(w http.ResponseWriter, r *http.Request) {
				var body map[string]any
				err := json.NewDecoder(r.Body).Decode(&body)
				assert.NoError(t, err)
				assert.Equal(t, "2001:db8::1", body["ipv6"])
				_, hasIPv4 := body["ipv4"]
				assert.False(t, hasIPv4, "request must not contain the ipv4 key")

				fmt.Fprint(w, `{"success":true,"data":{"hostname":"host.example.com","ipv6":"2001:db8::1"}}`)
			},
			ipVersion:  ipversion.IP6,
			ip:         ipv6,
			expectedIP: ipv6,
		},
		"invalid_token": {
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
				fmt.Fprint(w, `{"success":false,"error":{"code":"invalid_token","message":"Invalid or expired token"}}`)
			},
			ipVersion:  ipversion.IP4,
			ip:         ipv4,
			errWrapped: ddnserrors.ErrAuth,
			errMessage: "bad authentication: Invalid or expired token",
		},
		"hostname_not_found": {
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				fmt.Fprint(w, `{"success":false,"error":{"code":"hostname_not_found","message":"no such hostname"}}`)
			},
			ipVersion:  ipversion.IP4,
			ip:         ipv4,
			errWrapped: ddnserrors.ErrHostnameNotExists,
			errMessage: "hostname does not exist: no such hostname",
		},
		"server_error_html": {
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprint(w, "<html><body>500 Internal Server Error</body></html>")
			},
			ipVersion:  ipversion.IP4,
			ip:         ipv4,
			errWrapped: ddnserrors.ErrHTTPStatusNotValid,
		},
		"ip_mismatch": {
			handler: func(w http.ResponseWriter, _ *http.Request) {
				fmt.Fprint(w, `{"success":true,"data":{"hostname":"host.example.com","ipv4":"5.6.7.8"}}`)
			},
			ipVersion:  ipversion.IP4,
			ip:         ipv4,
			errWrapped: ddnserrors.ErrIPReceivedMismatch,
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(testCase.handler)
			defer server.Close()

			data, err := json.Marshal(map[string]string{
				"token":        token,
				"api_endpoint": server.URL,
			})
			require.NoError(t, err)

			p, err := New(data, "host.example.com", "@", testCase.ipVersion, netip.Prefix{})
			require.NoError(t, err)

			returnedIP, err := p.Update(t.Context(), server.Client(), testCase.ip)

			assert.ErrorIs(t, err, testCase.errWrapped)
			if testCase.errMessage != "" {
				assert.EqualError(t, err, testCase.errMessage)
			}
			assert.Equal(t, testCase.expectedIP, returnedIP)
		})
	}
}
