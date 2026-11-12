package config

import (
	"io/fs"
	"testing"

	"github.com/qdm12/gosettings/reader"
	"github.com/qdm12/gosettings/reader/sources/env"
	"github.com/stretchr/testify/assert"
)

func Test_Paths_read(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		environ    []string
		paths      Paths
		errMessage string
	}{
		"config_persist_not_set": {
			environ: []string{"DATADIR=/data"},
			paths:   Paths{DataDir: new("/data")},
		},
		"config_persist_disabled": {
			environ: []string{"CONFIG_PERSIST=no"},
			paths:   Paths{ConfigPersist: new(false)},
		},
		"config_persist_not_a_boolean": {
			environ:    []string{"CONFIG_PERSIST=abc"},
			errMessage: "environment variable CONFIG_PERSIST",
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			source := env.New(env.Settings{Environ: testCase.environ})
			settingsReader := reader.New(reader.Settings{Sources: []reader.Source{source}})

			var paths Paths
			err := paths.read(settingsReader)

			if testCase.errMessage != "" {
				assert.ErrorContains(t, err, testCase.errMessage)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, testCase.paths, paths)
		})
	}
}

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
