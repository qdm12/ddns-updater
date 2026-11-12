package backup

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_pruneBackups(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		keep uint
		// fileAges maps the name of each file to create to the age of its
		// last modification.
		fileAges         map[string]time.Duration
		directoryName    string
		missingDirectory bool
		remaining        []string
		errMessage       string
	}{
		"keep_zero_keeps_all": {
			keep: 0,
			fileAges: map[string]time.Duration{
				"ddns-updater-backup-1.zip": 2 * time.Hour,
				"ddns-updater-backup-2.zip": time.Hour,
			},
			remaining: []string{"ddns-updater-backup-1.zip", "ddns-updater-backup-2.zip"},
		},
		"fewer_backups_than_keep": {
			keep: 2,
			fileAges: map[string]time.Duration{
				"ddns-updater-backup-1.zip": time.Hour,
			},
			remaining: []string{"ddns-updater-backup-1.zip"},
		},
		"oldest_modified_backups_removed": {
			keep: 2,
			fileAges: map[string]time.Duration{
				"ddns-updater-backup-1.zip": time.Hour,
				"ddns-updater-backup-2.zip": 4 * time.Hour,
				"ddns-updater-backup-3.zip": 2 * time.Hour,
				"ddns-updater-backup-4.zip": 3 * time.Hour,
			},
			remaining: []string{"ddns-updater-backup-1.zip", "ddns-updater-backup-3.zip"},
		},
		"same_modification_time_ordered_by_name": {
			keep: 1,
			fileAges: map[string]time.Duration{
				"ddns-updater-backup-1.zip": time.Hour,
				"ddns-updater-backup-2.zip": time.Hour,
			},
			remaining: []string{"ddns-updater-backup-2.zip"},
		},
		"other_files_not_counted_nor_removed": {
			keep: 1,
			fileAges: map[string]time.Duration{
				"updates.json":              5 * time.Hour,
				"other.zip":                 5 * time.Hour,
				"ddns-updater-backup-1.txt": 5 * time.Hour,
				"ddns-updater-backup-2.zip": 2 * time.Hour,
				"ddns-updater-backup-3.zip": time.Hour,
			},
			directoryName: "ddns-updater-backup-0.zip",
			remaining: []string{
				"updates.json", "other.zip", "ddns-updater-backup-1.txt",
				"ddns-updater-backup-3.zip", "ddns-updater-backup-0.zip",
			},
		},
		"missing_directory": {
			keep:             1,
			missingDirectory: true,
			errMessage:       "reading directory",
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			directory := t.TempDir()
			if testCase.missingDirectory {
				directory = filepath.Join(directory, "missing")
			}
			now := time.Now()
			for fileName, age := range testCase.fileAges {
				path := filepath.Join(directory, fileName)
				err := os.WriteFile(path, nil, 0o600)
				require.NoError(t, err)
				modTime := now.Add(-age)
				err = os.Chtimes(path, modTime, modTime)
				require.NoError(t, err)
			}
			if testCase.directoryName != "" {
				err := os.Mkdir(filepath.Join(directory, testCase.directoryName), 0o700)
				require.NoError(t, err)
			}

			err := pruneBackups(directory, testCase.keep)

			if testCase.errMessage != "" {
				assert.ErrorContains(t, err, testCase.errMessage)
				return
			}
			require.NoError(t, err)
			entries, err := os.ReadDir(directory)
			require.NoError(t, err)
			remaining := make([]string, len(entries))
			for i, entry := range entries {
				remaining[i] = entry.Name()
			}
			assert.ElementsMatch(t, testCase.remaining, remaining)
		})
	}
}
