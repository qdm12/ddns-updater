package dnsomatic

import (
	"testing"

	"github.com/qdm12/ddns-updater/internal/provider/errors"
	"github.com/stretchr/testify/assert"
)

func Test_validateSettings(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		username   string
		password   string
		errWrapped error
		errMessage string
	}{
		"empty_username": {
			password:   "password",
			errWrapped: errors.ErrUsernameNotSet,
			errMessage: `username is not set`,
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := validateSettings(testCase.username, testCase.password)

			assert.ErrorIs(t, err, testCase.errWrapped)
			if testCase.errWrapped != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
		})
	}
}
