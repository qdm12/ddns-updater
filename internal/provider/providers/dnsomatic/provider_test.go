package dnsomatic

import (
	"testing"

	"github.com/qdm12/ddns-updater/internal/provider/errors"
	"github.com/stretchr/testify/assert"
)

func Test_Provider_isValid(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		provider   Provider
		errWrapped error
		errMessage string
	}{
		"empty_username": {
			provider: Provider{
				password: "password",
			},
			errWrapped: errors.ErrUsernameNotSet,
			errMessage: `username is not set`,
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := testCase.provider.isValid()

			assert.ErrorIs(t, err, testCase.errWrapped)
			if testCase.errWrapped != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
		})
	}
}
