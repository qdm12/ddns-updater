package dnsexit

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_validateSettings(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		domain     string
		owner      string
		apiKey     string
		errMessage string
	}{
		"valid": {
			domain: "domain.com",
			owner:  "sub",
			apiKey: "api-key",
		},
		"invalid_domain": {
			domain: "domain..com",
			apiKey: "api-key",
			errMessage: "domain is not valid: domain name has invalid character: " +
				`label starts with '.' for domain "domain..com"`,
		},
		"wildcard_owner": {
			domain:     "domain.com",
			owner:      "*",
			apiKey:     "api-key",
			errMessage: `owner cannot be "*"`,
		},
		"empty_api_key": {
			domain:     "domain.com",
			owner:      "sub",
			errMessage: "API key is not set",
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := validateSettings(testCase.domain, testCase.owner, testCase.apiKey)

			if testCase.errMessage == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, testCase.errMessage)
			}
		})
	}
}
