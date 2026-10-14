package config

import (
	"testing"

	settingsreader "github.com/qdm12/gosettings/reader"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBackupReadRejectsInvalidKeep(t *testing.T) {
	t.Setenv("BACKUP_PERIOD", "")
	t.Setenv("BACKUP_DIRECTORY", "")
	t.Setenv("BACKUP_INCLUDE_CONFIG", "")
	t.Setenv("BACKUP_KEEP", "invalid")

	var backup Backup
	err := backup.read(settingsreader.New(settingsreader.Settings{}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "BACKUP_KEEP")
}

func TestBackupReadRejectsInvalidIncludeConfig(t *testing.T) {
	t.Setenv("BACKUP_PERIOD", "")
	t.Setenv("BACKUP_DIRECTORY", "")
	t.Setenv("BACKUP_INCLUDE_CONFIG", "invalid")
	t.Setenv("BACKUP_KEEP", "")

	var backup Backup
	err := backup.read(settingsreader.New(settingsreader.Settings{}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "BACKUP_INCLUDE_CONFIG")
}

func TestConfigPolicyDefaults(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		environmentConfig bool
		persist           bool
		include           *bool
		wantInclude       bool
		wantError         string
	}{
		"file config preserves historical include default": {
			persist:     false,
			wantInclude: true,
		},
		"persisted environment config is included": {
			environmentConfig: true,
			persist:           true,
			wantInclude:       true,
		},
		"non persisted environment config is excluded": {
			environmentConfig: true,
			persist:           false,
			wantInclude:       false,
		},
		"explicit incompatible include is rejected": {
			environmentConfig: true,
			persist:           false,
			include:           new(true),
			wantInclude:       true,
			wantError:         "BACKUP_INCLUDE_CONFIG cannot be enabled",
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			configuration := Config{}
			configuration.Paths.environmentConfig = testCase.environmentConfig
			configuration.Paths.ConfigPersist = new(testCase.persist)
			configuration.Backup.IncludeConfig = testCase.include
			configuration.SetDefaults()
			assert.Equal(t, testCase.wantInclude, *configuration.Backup.IncludeConfig)
			err := configuration.Validate()
			if testCase.wantError == "" {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), testCase.wantError)
			}
		})
	}
}

func TestBackupValidateRejectsNegativeKeep(t *testing.T) {
	t.Parallel()

	backup := Backup{Keep: new(-1)}
	backup.setDefaults(true)
	err := backup.Validate()
	assert.ErrorIs(t, err, ErrBackupKeepNegative)
}
