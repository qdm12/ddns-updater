package backup

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func Test_run(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		// configFilePath is the configuration file path, relative to a
		// temporary directory. No configuration file is given if it is empty.
		configFilePath string
		zippedNames    []string
	}{
		"no_config_file": {
			zippedNames: []string{"updates.json"},
		},
		"config_file_in_data_directory": {
			configFilePath: filepath.Join("data", "config.json"),
			zippedNames:    []string{"config.json", "updates.json"},
		},
		"config_file_outside_data_directory": {
			configFilePath: filepath.Join("config", "custom.json"),
			zippedNames:    []string{"custom.json", "updates.json"},
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// The fake clock of synctest makes the backup period instant.
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

				const backupPeriod = time.Hour
				ready := make(chan struct{})
				runError := make(chan error)
				stopCh := make(chan struct{})
				done := make(chan struct{})
				go run(ready, runError, stopCh, done, outputDir, dataDir, configFilepath, backupPeriod, logger)
				<-ready

				timer := time.NewTimer(backupPeriod)
				<-timer.C
				synctest.Wait()
				close(stopCh)
				<-done

				entries, err := os.ReadDir(outputDir)
				require.NoError(t, err)
				require.Len(t, entries, 1)
				reader, err := zip.OpenReader(filepath.Join(outputDir, entries[0].Name()))
				require.NoError(t, err)
				defer reader.Close()
				zippedNames := make([]string, len(reader.File))
				for i, file := range reader.File {
					zippedNames[i] = file.Name
				}
				assert.Equal(t, testCase.zippedNames, zippedNames)
			})
		})
	}
}
