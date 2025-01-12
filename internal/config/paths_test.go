package config

import (
	"io/fs"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_parseUmask(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		s          string
		umask      fs.FileMode
		errMessage string
	}{
		"invalid": {
			s:          "a",
			errMessage: `strconv.ParseUint: parsing "a": invalid syntax`,
		},
		"704": {
			s:     "704",
			umask: 0o704,
		},
		"0704": {
			s:     "0704",
			umask: 0o0704,
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			umask, err := parseUmask(testCase.s)

			if testCase.errMessage != "" {
				assert.EqualError(t, err, testCase.errMessage)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, testCase.umask, umask)
		})
	}
}
