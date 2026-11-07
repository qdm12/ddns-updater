package cloudflare

import (
	"net/http"
	"net/netip"
	"testing"

	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_New(t *testing.T) {
	t.Parallel()

	const (
		domain         = "example.com"
		owner          = "sub"
		zoneIdentifier = "0123456789abcdef0123456789abcdef"
		token          = "token-value"
		email          = "me@example.com"
		key            = "globalapikey123"
	)

	testCases := map[string]struct {
		config string
		// expectedProvider is nil when the configuration must be rejected,
		// and expectedErrText is checked in that case.
		expectedProvider *Provider
		expectedErrText  string
	}{
		"token_only": {
			config: `{"zone_identifier": "` + zoneIdentifier + `",
				"ttl": 600, "token": "token-value"}`,
			expectedProvider: &Provider{
				domain: domain, owner: owner, ipVersion: ipversion.IP4,
				token: token, zoneIdentifier: zoneIdentifier, ttl: 600,
			},
		},
		"email_and_global_api_key": {
			config: `{"zone_identifier": "` + zoneIdentifier + `",
				"ttl": 300, "email": "me@example.com", "key": "globalapikey123"}`,
			expectedProvider: &Provider{
				domain: domain, owner: owner, ipVersion: ipversion.IP4,
				email: email, key: key, zoneIdentifier: zoneIdentifier, ttl: 300,
			},
		},
		"proxied": {
			config: `{"zone_identifier": "` + zoneIdentifier + `",
				"ttl": 600, "token": "token-value", "proxied": true}`,
			expectedProvider: &Provider{
				domain: domain, owner: owner, ipVersion: ipversion.IP4,
				token: token, zoneIdentifier: zoneIdentifier, proxied: true, ttl: 600,
			},
		},
		// Cloudflare removed service key authentication on 2026-09-30, so a user
		// service key is no longer a credential and leaves the token unset.
		"user_service_key_only": {
			config: `{"zone_identifier": "` + zoneIdentifier + `",
				"ttl": 600, "user_service_key": "v1.0.abcdefghij"}`,
			expectedErrText: "token is not set",
		},
		"no_credential": {
			config:          `{"zone_identifier": "` + zoneIdentifier + `", "ttl": 600}`,
			expectedErrText: "token is not set",
		},
		"user_service_key_ignored_with_token": {
			config: `{"zone_identifier": "` + zoneIdentifier + `",
				"ttl": 600, "token": "token-value", "user_service_key": "v1.0.abcdefghij"}`,
			expectedProvider: &Provider{
				domain: domain, owner: owner, ipVersion: ipversion.IP4,
				token: token, zoneIdentifier: zoneIdentifier, ttl: 600,
			},
		},
		"user_service_key_ignored_with_email_and_key": {
			config: `{"zone_identifier": "` + zoneIdentifier + `",
				"ttl": 600, "email": "me@example.com", "key": "globalapikey123",
				"user_service_key": "v1.0.abcdefghij"}`,
			expectedProvider: &Provider{
				domain: domain, owner: owner, ipVersion: ipversion.IP4,
				email: email, key: key, zoneIdentifier: zoneIdentifier, ttl: 600,
			},
		},
		"email_with_invalid_key": {
			config: `{"zone_identifier": "` + zoneIdentifier + `",
				"ttl": 600, "email": "me@example.com", "key": "not a valid key"}`,
			expectedErrText: "key is not valid",
		},
		"invalid_email": {
			config: `{"zone_identifier": "` + zoneIdentifier + `",
				"ttl": 600, "email": "not-an-email", "key": "globalapikey123"}`,
			expectedErrText: "email address is not valid",
		},
		"zone_identifier_missing": {
			config:          `{"ttl": 600, "token": "token-value"}`,
			expectedErrText: "zone identifier is not set",
		},
		"ttl_not_set": {
			config:          `{"zone_identifier": "` + zoneIdentifier + `", "token": "token-value"}`,
			expectedErrText: "TTL is not set",
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			provider, err := New([]byte(testCase.config), domain, owner, ipversion.IP4, netip.Prefix{})

			if testCase.expectedErrText != "" {
				assert.ErrorContains(t, err, testCase.expectedErrText)
				assert.Nil(t, provider)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, testCase.expectedProvider, provider)
		})
	}
}

func Test_Provider_setHeaders(t *testing.T) {
	t.Parallel()

	const (
		token = "token-value"
		email = "me@example.com"
		key   = "globalapikey123"
	)

	testCases := map[string]struct {
		provider      Provider
		expectedToken string
		expectedEmail string
		expectedKey   string
	}{
		"token": {
			provider:      Provider{token: token},
			expectedToken: token,
		},
		"email_and_key": {
			provider:      Provider{email: email, key: key},
			expectedEmail: email,
			expectedKey:   key,
		},
		"token_takes_precedence_over_email_and_key": {
			provider:      Provider{token: token, email: email, key: key},
			expectedToken: token,
		},
		"no_credential": {
			provider: Provider{},
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			request, err := http.NewRequestWithContext(t.Context(), http.MethodGet,
				"https://api.cloudflare.com/client/v4/zones", nil)
			require.NoError(t, err)

			testCase.provider.setHeaders(request)

			assert.Equal(t, "DDNS-Updater quentin.mcgaw@gmail.com", request.Header.Get("User-Agent"))
			assert.Equal(t, "application/json", request.Header.Get("Content-Type"))
			assert.Equal(t, "application/json", request.Header.Get("Accept"))
			// Cloudflare removed the X-Auth-User-Service-Key authentication method
			// on 2026-09-30, so the header must never be sent.
			assert.Empty(t, request.Header.Get("X-Auth-User-Service-Key"))

			if testCase.expectedEmail == "" {
				assert.Empty(t, request.Header.Get("X-Auth-Email"))
				assert.Empty(t, request.Header.Get("X-Auth-Key"))
			} else {
				assert.Equal(t, testCase.expectedEmail, request.Header.Get("X-Auth-Email"))
				assert.Equal(t, testCase.expectedKey, request.Header.Get("X-Auth-Key"))
			}

			if testCase.expectedToken == "" {
				assert.Empty(t, request.Header.Get("Authorization"))
			} else {
				assert.Equal(t, "Bearer "+testCase.expectedToken, request.Header.Get("Authorization"))
			}
		})
	}
}
