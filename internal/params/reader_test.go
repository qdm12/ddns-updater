package params

import (
	"io/fs"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func Test_Reader_JSONProviders(t *testing.T) {
	t.Parallel()

	type writtenFile struct {
		path string
		data string
		perm fs.FileMode
	}

	testCases := map[string]struct {
		environmentConfig        string
		persistEnvironmentConfig bool
		// fileContent is the content of the configuration file.
		fileContent string
		logged      []string
		// fileRead is true if the configuration file is expected to be read.
		fileRead     bool
		writtenFiles []writtenFile
	}{
		"config_from_file": {
			persistEnvironmentConfig: true,
			fileContent:              `{"settings":[]}`,
			logged: []string{
				"reading JSON config from file config.json",
				`config read: {"settings":[]}`,
			},
			fileRead: true,
		},
		"config_from_environment_persisted": {
			environmentConfig:        `{"settings":[]}`,
			persistEnvironmentConfig: true,
			logged: []string{
				"reading JSON config from environment variable CONFIG",
				`config read: {"settings":[]}`,
			},
			writtenFiles: []writtenFile{{
				path: "config.json",
				data: "{\n  \"settings\": []\n}",
				perm: 0o666,
			}},
		},
		// The configuration file is neither written nor read, although the
		// configuration given has no settings.
		"config_from_environment_not_persisted": {
			environmentConfig: `{"settings":[]}`,
			fileContent:       `{"settings":[{"provider":"invalid"}]}`,
			logged: []string{
				"reading JSON config from environment variable CONFIG",
				`config read: {"settings":[]}`,
			},
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			logger := NewMockLogger(ctrl)
			logger.EXPECT().Info(testCase.logged[0])
			logger.EXPECT().Debug(testCase.logged[1])

			fileRead := false
			var writtenFiles []writtenFile
			reader := &Reader{
				logger:                   logger,
				environmentConfig:        testCase.environmentConfig,
				persistEnvironmentConfig: testCase.persistEnvironmentConfig,
				readFile: func(filename string) ([]byte, error) {
					assert.Equal(t, "config.json", filename)
					fileRead = true
					return []byte(testCase.fileContent), nil
				},
				writeFile: func(filename string, data []byte, perm fs.FileMode) error {
					writtenFiles = append(writtenFiles, writtenFile{
						path: filename,
						data: string(data),
						perm: perm,
					})
					return nil
				},
			}

			providers, warnings, err := reader.JSONProviders("config.json")

			assert.NoError(t, err)
			assert.Empty(t, providers)
			assert.Empty(t, warnings)
			assert.Equal(t, testCase.fileRead, fileRead)
			assert.Equal(t, testCase.writtenFiles, writtenFiles)
		})
	}
}
