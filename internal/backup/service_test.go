package backup

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func Test_run(t *testing.T) {
	t.Parallel()

	const backupPeriod = time.Hour

	testCases := map[string]struct {
		// configFilePath is the configuration file path, relative to a
		// temporary directory. No configuration file is given if it is empty.
		configFilePath string
		keep           uint
		// duration is how long the backups run for.
		duration time.Duration
		// remaining lists the backups expected to remain, each as the
		// duration after which it was written.
		remaining []time.Duration
		// zippedNames are the names of the files in each remaining backup.
		zippedNames []string
	}{
		"no_config_file": {
			duration:    backupPeriod,
			remaining:   []time.Duration{backupPeriod},
			zippedNames: []string{"updates.json"},
		},
		"config_file_in_data_directory": {
			configFilePath: filepath.Join("data", "config.json"),
			duration:       backupPeriod,
			remaining:      []time.Duration{backupPeriod},
			zippedNames:    []string{"config.json", "updates.json"},
		},
		"config_file_outside_data_directory": {
			configFilePath: filepath.Join("config", "custom.json"),
			duration:       backupPeriod,
			remaining:      []time.Duration{backupPeriod},
			zippedNames:    []string{"custom.json", "updates.json"},
		},
		"all_backups_kept": {
			configFilePath: filepath.Join("data", "config.json"),
			keep:           0,
			duration:       3 * backupPeriod,
			remaining:      []time.Duration{backupPeriod, 2 * backupPeriod, 3 * backupPeriod},
			zippedNames:    []string{"config.json", "updates.json"},
		},
		"oldest_backups_removed": {
			configFilePath: filepath.Join("data", "config.json"),
			keep:           2,
			duration:       3 * backupPeriod,
			remaining:      []time.Duration{2 * backupPeriod, 3 * backupPeriod},
			zippedNames:    []string{"config.json", "updates.json"},
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// The fake clock of synctest makes the backup periods instant
			// and the timestamps in the backup file names predictable.
			synctest.Test(t, func(t *testing.T) {
				ctrl := gomock.NewController(t)

				directory := t.TempDir()
				dataDir := filepath.Join(directory, "data")
				outputDir := filepath.Join(directory, "backups")
				filePaths := []string{filepath.Join(dataDir, "updates.json")}
				configFilepath := ""
				if testCase.configFilePath != "" {
					configFilepath = filepath.Join(directory, testCase.configFilePath)
					filePaths = append(filePaths, configFilepath)
				}
				err := os.Mkdir(outputDir, 0o700)
				require.NoError(t, err)
				for _, filePath := range filePaths {
					err = os.MkdirAll(filepath.Dir(filePath), 0o700)
					require.NoError(t, err)
					err = os.WriteFile(filePath, []byte("{}"), 0o600)
					require.NoError(t, err)
				}

				logger := NewMockLogger(ctrl)
				logger.EXPECT().Info("each 1h0m0s; writing zip files to directory " + outputDir)
				if configFilepath == "" {
					logger.EXPECT().Info("the configuration file is not backed up")
				}

				start := time.Now()
				ready := make(chan struct{})
				runError := make(chan error)
				stopCh := make(chan struct{})
				done := make(chan struct{})
				go run(ready, runError, stopCh, done, outputDir, dataDir, configFilepath,
					backupPeriod, logger, testCase.keep)
				<-ready

				timer := time.NewTimer(testCase.duration)
				<-timer.C
				synctest.Wait()
				close(stopCh)
				<-done

				remaining := make([]string, len(testCase.remaining))
				for i, duration := range testCase.remaining {
					timestamp := start.Add(duration).UnixNano()
					remaining[i] = "ddns-updater-backup-" + strconv.Itoa(int(timestamp)) + ".zip"
				}
				entries, err := os.ReadDir(outputDir)
				require.NoError(t, err)
				names := make([]string, len(entries))
				for i, entry := range entries {
					names[i] = entry.Name()
				}
				require.ElementsMatch(t, remaining, names)

				for _, name := range names {
					reader, err := zip.OpenReader(filepath.Join(outputDir, name))
					require.NoError(t, err)
					zippedNames := make([]string, len(reader.File))
					for i, file := range reader.File {
						zippedNames[i] = file.Name
					}
					err = reader.Close()
					require.NoError(t, err)
					assert.Equal(t, testCase.zippedNames, zippedNames)
				}
			})
		})
	}
}
