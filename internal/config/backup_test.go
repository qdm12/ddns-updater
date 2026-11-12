package config

import (
	"testing"
	"time"

	"github.com/qdm12/gosettings/reader"
	"github.com/qdm12/gosettings/reader/sources/env"
	"github.com/stretchr/testify/assert"
)

func Test_Backup_read(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		environ    []string
		backup     Backup
		errMessage string
	}{
		"keep_not_set": {
			environ: []string{"BACKUP_DIRECTORY=/backups"},
			backup:  Backup{Directory: new("/backups")},
		},
		"keep_set": {
			environ: []string{"BACKUP_KEEP=3"},
			backup:  Backup{Keep: new(uint(3))},
		},
		"keep_negative": {
			environ:    []string{"BACKUP_KEEP=-1"},
			errMessage: "environment variable BACKUP_KEEP",
		},
		"keep_not_a_number": {
			environ:    []string{"BACKUP_KEEP=abc"},
			errMessage: "environment variable BACKUP_KEEP",
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			source := env.New(env.Settings{Environ: testCase.environ})
			settingsReader := reader.New(reader.Settings{Sources: []reader.Source{source}})

			var backup Backup
			err := backup.read(settingsReader)

			if testCase.errMessage != "" {
				assert.ErrorContains(t, err, testCase.errMessage)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, testCase.backup, backup)
		})
	}
}

func Test_Backup_String(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		backup  Backup
		summary string
	}{
		"disabled": {
			backup:  Backup{},
			summary: "Backup: disabled",
		},
		"keep_all_backups": {
			backup: Backup{Period: new(time.Hour)},
			summary: `Backup
├── Period: 1h0m0s
├── Directory: ./data
└── Keep: all backups`,
		},
		"keep_some_backups": {
			backup: Backup{Period: new(time.Hour), Keep: new(uint(3))},
			summary: `Backup
├── Period: 1h0m0s
├── Directory: ./data
└── Keep: 3 backups`,
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			testCase.backup.setDefaults()

			summary := testCase.backup.String()

			assert.Equal(t, testCase.summary, summary)
		})
	}
}
