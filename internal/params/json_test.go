package params

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_extractAllSettings(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		settings   string
		length     int
		errMessage string
	}{
		"opendns": {
			settings: `{"provider": "opendns", "domain": "sub.example.com", ` +
				`"username": "username", "password": "password", "hostname": "Home"}`,
			length: 1,
		},
		// The name of the network declared in the account is not a domain name,
		// and is set in the hostname setting, not in the domain one.
		"opendns_network_name_as_domain": {
			settings: `{"provider": "opendns", "domain": "Home", ` +
				`"username": "username", "password": "password", "hostname": "Home"}`,
			errMessage: "the OpenDNS network name goes in the hostname setting",
		},
		// The hostname setting is optional, to keep existing configurations working.
		"opendns_hostname_unset": {
			settings: `{"provider": "opendns", "domain": "sub.example.com", ` +
				`"username": "username", "password": "password"}`,
			length: 1,
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			providers, _, err := extractAllSettings([]byte(`{"settings": [` + testCase.settings + `]}`))

			switch {
			case testCase.errMessage != "":
				require.ErrorContains(t, err, testCase.errMessage)
			default:
				require.NoError(t, err)
				assert.Equal(t, testCase.length, len(providers))
			}
		})
	}
}

func Test_extractFromDomainField(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		domainField      string
		domainRegistered string
		owners           []string
		errMessage       string
	}{
		"root_domain": {
			domainField:      "example.com",
			domainRegistered: "example.com",
			owners:           []string{"@"},
		},
		"subdomain": {
			domainField:      "abc.example.com",
			domainRegistered: "example.com",
			owners:           []string{"abc"},
		},
		"two_dots_tld": {
			domainField:      "abc.example.co.uk",
			domainRegistered: "example.co.uk",
			owners:           []string{"abc"},
		},
		"wildcard": {
			domainField:      "*.example.com",
			domainRegistered: "example.com",
			owners:           []string{"*"},
		},
		"multiple": {
			domainField:      "*.example.com,example.com",
			domainRegistered: "example.com",
			owners:           []string{"*", "@"},
		},
		"different_domains": {
			domainField: "*.example.com,abc.something.com",
			errMessage:  "multiple domains specified: \"example.com\" and \"something.com\"",
		},
		"goip.de": {
			domainField:      "my.domain.goip.de",
			domainRegistered: "domain.goip.de",
			owners:           []string{"my"},
		},
		"duckdns.org": {
			domainField:      "my.domain.duckdns.org",
			domainRegistered: "domain.duckdns.org",
			owners:           []string{"my"},
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			domainRegistered, owners, err := extractFromDomainField(testCase.domainField)

			if testCase.errMessage == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, testCase.errMessage)
			}
			assert.Equal(t, testCase.domainRegistered, domainRegistered)
			assert.Equal(t, testCase.owners, owners)
		})
	}
}
