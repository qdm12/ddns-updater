package feno

import (
	"net/http"
	"net/netip"
	"testing"

	"github.com/qdm12/ddns-updater/internal/provider/errors"
	"github.com/stretchr/testify/assert"
)

func Test_validateSettings(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		domain     string
		owner      string
		apiKey     string
		errWrapped error
		errMessage string
	}{
		"valid": {
			domain: "example.no",
			owner:  "home",
			apiKey: "feno_live_abc",
		},
		"invalid_domain": {
			domain:     "",
			owner:      "@",
			apiKey:     "feno_live_abc",
			errWrapped: errors.ErrDomainNotValid,
			errMessage: `domain is not valid: domain is not set`,
		},
		"wildcard_owner": {
			domain:     "example.no",
			owner:      "*",
			apiKey:     "feno_live_abc",
			errWrapped: errors.ErrOwnerWildcard,
			errMessage: `owner cannot be "*"`,
		},
		"empty_api_key": {
			domain:     "example.no",
			owner:      "@",
			errWrapped: errors.ErrAPIKeyNotSet,
			errMessage: `API key is not set`,
		},
		"api_key_bad_prefix": {
			domain:     "example.no",
			owner:      "@",
			apiKey:     "abc",
			errWrapped: errors.ErrKeyNotValid,
			errMessage: `key is not valid: it should start with feno_live_`,
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := validateSettings(testCase.domain, testCase.owner, testCase.apiKey)

			assert.ErrorIs(t, err, testCase.errWrapped)
			if testCase.errWrapped != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
		})
	}
}

func Test_parseResponse(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		statusCode int
		body       string
		ip         netip.Addr
		newIP      netip.Addr
		errWrapped error
		errMessage string
	}{
		"good_ipv4": {
			statusCode: http.StatusOK,
			body:       "good 203.0.113.7",
			ip:         netip.MustParseAddr("203.0.113.7"),
			newIP:      netip.MustParseAddr("203.0.113.7"),
		},
		"nochg_ipv4": {
			statusCode: http.StatusOK,
			body:       "nochg 203.0.113.7",
			ip:         netip.MustParseAddr("203.0.113.7"),
			newIP:      netip.MustParseAddr("203.0.113.7"),
		},
		"good_ipv6": {
			statusCode: http.StatusOK,
			body:       "good 2001:db8::1",
			ip:         netip.MustParseAddr("2001:db8::1"),
			newIP:      netip.MustParseAddr("2001:db8::1"),
		},
		"good_dual_stack_echo": {
			statusCode: http.StatusOK,
			body:       "good 203.0.113.7, 2001:db8::1",
			ip:         netip.MustParseAddr("2001:db8::1"),
			newIP:      netip.MustParseAddr("2001:db8::1"),
		},
		"ip_mismatch": {
			statusCode: http.StatusOK,
			body:       "good 203.0.113.8",
			ip:         netip.MustParseAddr("203.0.113.7"),
			errWrapped: errors.ErrIPReceivedMismatch,
			errMessage: "mismatching IP address received: sent ip 203.0.113.7 to update but received 203.0.113.8",
		},
		"good_without_ip": {
			statusCode: http.StatusOK,
			body:       "good",
			ip:         netip.MustParseAddr("203.0.113.7"),
			errWrapped: errors.ErrReceivedNoIP,
			errMessage: "received no IP address in response",
		},
		"empty": {
			statusCode: http.StatusOK,
			body:       "",
			ip:         netip.MustParseAddr("203.0.113.7"),
			errWrapped: errors.ErrReceivedNoResult,
			errMessage: "received no result in response",
		},
		"badauth": {
			statusCode: http.StatusUnauthorized,
			body:       "badauth",
			ip:         netip.MustParseAddr("203.0.113.7"),
			errWrapped: errors.ErrAuth,
			errMessage: "bad authentication",
		},
		"notfqdn": {
			statusCode: http.StatusOK,
			body:       "notfqdn",
			ip:         netip.MustParseAddr("203.0.113.7"),
			errWrapped: errors.ErrHostnameNotExists,
			errMessage: "hostname does not exist: hostname is not a fully qualified domain name",
		},
		"nohost": {
			statusCode: http.StatusOK,
			body:       "nohost",
			ip:         netip.MustParseAddr("203.0.113.7"),
			errWrapped: errors.ErrHostnameNotExists,
			errMessage: "hostname does not exist: hostname is not in a zone hosted on FENO nameservers",
		},
		"numhost": {
			statusCode: http.StatusOK,
			body:       "numhost",
			ip:         netip.MustParseAddr("203.0.113.7"),
			errWrapped: errors.ErrBadRequest,
			errMessage: "bad request sent: too many hostnames",
		},
		"badagent": {
			statusCode: http.StatusOK,
			body:       "badagent",
			ip:         netip.MustParseAddr("203.0.113.7"),
			errWrapped: errors.ErrIPSentMalformed,
			errMessage: "malformed IP address sent",
		},
		"abuse": {
			statusCode: http.StatusOK,
			body:       "abuse",
			ip:         netip.MustParseAddr("203.0.113.7"),
			errWrapped: errors.ErrBannedAbuse,
			errMessage: "banned due to abuse",
		},
		"dnserr": {
			statusCode: http.StatusOK,
			body:       "dnserr",
			ip:         netip.MustParseAddr("203.0.113.7"),
			errWrapped: errors.ErrDNSServerSide,
			errMessage: "server side DNS error",
		},
		"911": {
			statusCode: http.StatusOK,
			body:       "911",
			ip:         netip.MustParseAddr("203.0.113.7"),
			errWrapped: errors.ErrDNSServerSide,
			errMessage: "server side DNS error: temporary server side error",
		},
		"http_status_not_ok": {
			statusCode: http.StatusInternalServerError,
			body:       "Internal server error",
			ip:         netip.MustParseAddr("203.0.113.7"),
			errWrapped: errors.ErrHTTPStatusNotValid,
			errMessage: "HTTP status is not valid: 500: Internal server error",
		},
		"unknown_response": {
			statusCode: http.StatusOK,
			body:       "something else",
			ip:         netip.MustParseAddr("203.0.113.7"),
			errWrapped: errors.ErrUnknownResponse,
			errMessage: "unknown response received: something else",
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			newIP, err := parseResponse(testCase.statusCode, testCase.body, testCase.ip)

			assert.Equal(t, testCase.newIP, newIP)
			assert.ErrorIs(t, err, testCase.errWrapped)
			if testCase.errWrapped != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
		})
	}
}
