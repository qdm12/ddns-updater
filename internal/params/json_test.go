package params

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_extractFromDomainField(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		domainField      string
		domainRegistered string
		owners           []string
		errWrapped       error
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
			errWrapped:  ErrMultipleDomainsSpecified,
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
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			domainRegistered, owners, err := extractFromDomainField(testCase.domainField)

			assert.ErrorIs(t, err, testCase.errWrapped)
			if testCase.errWrapped != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
			assert.Equal(t, testCase.domainRegistered, domainRegistered)
			assert.Equal(t, testCase.owners, owners)
		})
	}
}
