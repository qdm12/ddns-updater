package settings

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ReadFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	testCases := map[string]struct {
		content       string
		periodSet     bool
		period        time.Duration
		cooldownSet   bool
		cooldown      time.Duration
		errWrapped    error
		errMessage    string
		writeFileName string
	}{
		"no_file": {
			errWrapped: nil,
		},
		"empty_file": {
			content:       "",
			writeFileName: "empty.json",
		},
		"empty_json_object": {
			content:       "{}",
			writeFileName: "empty_object.json",
		},
		"period_and_cooldown": {
			content:       `{"PERIOD": "30s", "UPDATE_COOLDOWN_PERIOD": "1m"}`,
			periodSet:     true,
			period:        30 * time.Second,
			cooldownSet:   true,
			cooldown:      time.Minute,
			writeFileName: "both.json",
		},
		"lowercase_keys": {
			content:       `{"period": "2m"}`,
			periodSet:     true,
			period:        2 * time.Minute,
			writeFileName: "lowercase.json",
		},
		"invalid_json": {
			content:       `{invalid`,
			errWrapped:    ErrInvalidValue,
			errMessage:    "invalid value",
			writeFileName: "invalid.json",
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			filePath := filepath.Join(dir, "does_not_exist.json")
			if testCase.writeFileName != "" {
				filePath = filepath.Join(dir, testCase.writeFileName)
				err := os.WriteFile(filePath, []byte(testCase.content), 0o600)
				require.NoError(t, err)
			}

			file, err := ReadFile(filePath)
			if testCase.errWrapped != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, testCase.errWrapped)
				assert.Contains(t, err.Error(), testCase.errMessage)
				return
			}
			require.NoError(t, err)

			period, periodSet, err := file.Duration(PeriodKey)
			require.NoError(t, err)
			assert.Equal(t, testCase.periodSet, periodSet)
			if testCase.periodSet {
				assert.Equal(t, testCase.period, period)
			}

			cooldown, cooldownSet, err := file.Duration(CooldownKey)
			require.NoError(t, err)
			assert.Equal(t, testCase.cooldownSet, cooldownSet)
			if testCase.cooldownSet {
				assert.Equal(t, testCase.cooldown, cooldown)
			}
		})
	}
}

func Test_File_Duration_invalid(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "settings.json")

	err := os.WriteFile(filePath, []byte(`{"PERIOD": "abc"}`), 0o600)
	require.NoError(t, err)

	file, err := ReadFile(filePath)
	require.NoError(t, err)

	_, set, err := file.Duration(PeriodKey)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidValue)
	assert.True(t, set)
}

func Test_File_SetDuration(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "sub", "settings.json")

	file, err := ReadFile(filePath)
	require.NoError(t, err)

	err = file.SetDuration(PeriodKey, 90*time.Second)
	require.NoError(t, err)

	// The value must be readable back from the file.
	rereadFile, err := ReadFile(filePath)
	require.NoError(t, err)

	period, set, err := rereadFile.Duration(PeriodKey)
	require.NoError(t, err)
	assert.True(t, set)
	assert.Equal(t, 90*time.Second, period)

	// Setting another key must not drop the previous one.
	err = rereadFile.SetDuration(CooldownKey, 2*time.Minute)
	require.NoError(t, err)

	rereadFile, err = ReadFile(filePath)
	require.NoError(t, err)

	period, set, err = rereadFile.Duration(PeriodKey)
	require.NoError(t, err)
	assert.True(t, set)
	assert.Equal(t, 90*time.Second, period)

	cooldown, set, err := rereadFile.Duration(CooldownKey)
	require.NoError(t, err)
	assert.True(t, set)
	assert.Equal(t, 2*time.Minute, cooldown)
}
