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
			errWrapped: errors.ErrUsernameNotValid,
			errMessage: `username is not valid: username "" does not match regex "^[a-zA-Z0-9+@._-]{3,25}$"`,
		},
		"email_username": {
			provider: Provider{
				username: "a@a.ca",
				password: "password",
			},
		},
		"email_alias_username": {
			provider: Provider{
				username: "a+b@a.ca",
				password: "password",
			},
		},
		"dashes_username": {
			provider: Provider{
				username: "a-b-c",
				password: "password",
			},
		},
		"oversized_username": {
			provider: Provider{
				username: "aaaaaaaaaaaaaaaaaaaaaaaaaa",
				password: "password",
			},
			errWrapped: errors.ErrUsernameNotValid,
			errMessage: `username is not valid: username ` +
				`"aaaaaaaaaaaaaaaaaaaaaaaaaa" does not match regex "^[a-zA-Z0-9+@._-]{3,25}$"`,
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
