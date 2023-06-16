package shoutrrr

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_addDefaultTitle(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		address        string
		defaultTitle   string
		updatedAddress string
	}{
		"generic_with_empty_title": {
			address:        "generic://example.com?title=",
			defaultTitle:   "DDNS Updater",
			updatedAddress: "generic://example.com?title=",
		},
		"generic_with_title": {
			address:        "generic://example.com?title=MyTitle",
			defaultTitle:   "DDNS Updater",
			updatedAddress: "generic://example.com?title=MyTitle",
		},
		"generic_without_title": {
			address:        "generic://example.com",
			defaultTitle:   "DDNS Updater",
			updatedAddress: "generic://example.com?title=DDNS+Updater",
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			updatedAddress := addDefaultTitle(testCase.address, testCase.defaultTitle)

			assert.Equal(t, testCase.updatedAddress, updatedAddress)
		})
	}
}
