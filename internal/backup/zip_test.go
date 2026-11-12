package backup

import (
	"archive/zip"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_zipFiles(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		// files maps the name of each file to create to its content.
		files map[string]string
		// inputFileNames are the names of the files to zip.
		inputFileNames []string
		// zipped maps the name of each zipped file to its content.
		zipped map[string]string
		errIs  error
	}{
		"files_zipped": {
			files: map[string]string{
				"config.json":  `{"settings":[]}`,
				"updates.json": `{"records":[]}`,
			},
			inputFileNames: []string{"config.json", "updates.json"},
			zipped: map[string]string{
				"config.json":  `{"settings":[]}`,
				"updates.json": `{"records":[]}`,
			},
		},
		"zip_file_removed_on_missing_file": {
			files:          map[string]string{"config.json": `{"settings":[]}`},
			inputFileNames: []string{"config.json", "updates.json"},
			errIs:          fs.ErrNotExist,
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			directory := t.TempDir()
			for fileName, content := range testCase.files {
				err := os.WriteFile(filepath.Join(directory, fileName), []byte(content), 0o600)
				require.NoError(t, err)
			}
			inputFilepaths := make([]string, len(testCase.inputFileNames))
			for i, fileName := range testCase.inputFileNames {
				inputFilepaths[i] = filepath.Join(directory, fileName)
			}
			outputFilepath := filepath.Join(directory, "backup.zip")

			err := zipFiles(outputFilepath, inputFilepaths...)

			if testCase.errIs != nil {
				require.ErrorIs(t, err, testCase.errIs)
				assert.NoFileExists(t, outputFilepath)
				return
			}
			require.NoError(t, err)
			reader, err := zip.OpenReader(outputFilepath)
			require.NoError(t, err)
			defer reader.Close()
			zipped := make(map[string]string, len(reader.File))
			for _, file := range reader.File {
				fileReader, err := file.Open()
				require.NoError(t, err)
				content, err := io.ReadAll(fileReader)
				require.NoError(t, err)
				err = fileReader.Close()
				require.NoError(t, err)
				zipped[file.Name] = string(content)
			}
			assert.Equal(t, testCase.zipped, zipped)
		})
	}
}
