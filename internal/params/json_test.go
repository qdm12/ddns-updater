package params

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func Test_ValidateEntry(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		entry          string
		expectedOwners []string
		errWrapped     error
		errContains    string
	}{
		"duckdns": {
			entry:          `{"provider":"duckdns","domain":"example.duckdns.org","token":"00000000-0000-0000-0000-000000000000"}`,
			expectedOwners: []string{"@"},
		},
		"multiple_owners": {
			entry:          `{"provider":"namecheap","domain":"a.example.com,b.example.com","password":"e5322165c1d74692bfa6d807100c0310"}`,
			expectedOwners: []string{"a", "b"},
		},
		"wildcard_owner": {
			entry:          `{"provider":"namecheap","domain":"*.example.com","password":"e5322165c1d74692bfa6d807100c0310"}`,
			expectedOwners: []string{"*"},
		},
		"missing_provider": {
			entry:       `{"domain":"example.com"}`,
			errWrapped:  ErrInvalidEntry,
			errContains: "provider is not set",
		},
		"unknown_provider": {
			entry:       `{"provider":"nope","domain":"example.com"}`,
			errWrapped:  ErrInvalidEntry,
			errContains: "unknown provider",
		},
		"invalid_ip_version": {
			entry:       `{"provider":"namecheap","domain":"example.com","password":"x","ip_version":"ipv7"}`,
			errWrapped:  ErrInvalidEntry,
			errContains: "IP version",
		},
		"invalid_domain": {
			entry:       `{"provider":"namecheap","domain":"notadomain","password":"x"}`,
			errWrapped:  ErrInvalidEntry,
			errContains: "domain",
		},
		"missing_credentials": {
			entry:       `{"provider":"namecheap","domain":"example.com"}`,
			errWrapped:  ErrInvalidEntry,
			errContains: "password",
		},
		"not_an_object": {
			entry:       `[1,2]`,
			errWrapped:  ErrInvalidEntry,
			errContains: "decoding entry",
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			providers, _, err := ValidateEntry(json.RawMessage(testCase.entry))
			assert.ErrorIs(t, err, testCase.errWrapped)
			if testCase.errWrapped != nil {
				require.Error(t, err)
				assert.Contains(t, err.Error(), testCase.errContains)
				return
			}

			require.NoError(t, err)
			owners := make([]string, len(providers))
			for i, provider := range providers {
				owners[i] = provider.Owner()
			}
			assert.Equal(t, testCase.expectedOwners, owners)
		})
	}
}

func Test_ParseSettings(t *testing.T) {
	t.Parallel()

	providers, warnings, err := ParseSettings([]byte(`{
		"settings": [
			{"provider": "namecheap", "domain": "example.com", "password": "e5322165c1d74692bfa6d807100c0310", "host": "www"}
		]
	}`))
	require.NoError(t, err)
	require.Len(t, providers, 1)
	assert.Equal(t, "www.example.com", providers[0].BuildDomainName())
	assert.NotEmpty(t, warnings, "the host field is deprecated and should warn")

	_, _, err = ParseSettings([]byte(`{"settings":[{"provider":"nope","domain":"example.com"}]}`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown provider")
}
