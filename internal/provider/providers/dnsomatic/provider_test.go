package dnsomatic

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_validateSettings(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		domain     string
		username   string
		password   string
		errMessage string
	}{
		"empty_username": {
			domain:     "domain.com",
			password:   "password",
			errMessage: `username is not set`,
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := validateSettings(testCase.domain, testCase.username, testCase.password)

			if testCase.errMessage == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, testCase.errMessage)
			}
		})
	}
}
