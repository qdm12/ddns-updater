package json

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
	"time"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Database_write(t *testing.T) {
	t.Parallel()

	const existingContent = `{"records":[]}`
	data := dataModel{Records: []record{{Domain: "example.com", Owner: "home"}}}
	const dataContent = `{
  "records": [
    {
      "domain": "example.com",
      "owner": "home",
      "ips": null
    }
  ]
}
`

	testCases := map[string]struct {
		// existingPerm is the permission of the existing database file,
		// which does not exist if it is zero.
		existingPerm fs.FileMode
		// symlink is true if the database file path is a symbolic link to
		// the existing database file, located in another directory.
		symlink bool
		// directory is true if the database file path is a directory.
		directory bool
		data      dataModel
		content   string
		// replaced is true if the existing database file is expected to be
		// replaced by another file, and not written in place.
		replaced   bool
		errMessage string
	}{
		"database_created": {
			data:    data,
			content: dataContent,
		},
		"database_replaced_with_its_permissions": {
			existingPerm: 0o640,
			data:         data,
			content:      dataContent,
			replaced:     true,
		},
		"database_replaced_behind_symbolic_link": {
			existingPerm: 0o600,
			symlink:      true,
			data:         data,
			content:      dataContent,
			replaced:     true,
		},
		"database_kept_on_encoding_error": {
			existingPerm: 0o600,
			data: dataModel{Records: []record{{
				Events: []models.HistoryEvent{{Time: time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)}},
			}}},
			content:    existingContent,
			errMessage: "encoding data",
		},
		"database_path_is_a_directory": {
			directory:  true,
			data:       data,
			errMessage: "replacing database file: renaming temporary file",
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			directory := t.TempDir()
			databaseFilepath := filepath.Join(directory, "updates.json")
			existingFilepath := databaseFilepath
			if testCase.symlink {
				existingFilepath = filepath.Join(t.TempDir(), "target.json")
				err := os.Symlink(existingFilepath, databaseFilepath)
				require.NoError(t, err)
			}
			var existingFileInfo fs.FileInfo
			if testCase.existingPerm != 0 {
				err := os.WriteFile(existingFilepath, []byte(existingContent), testCase.existingPerm)
				require.NoError(t, err)
				// The permission given when creating the file is subject to the umask.
				err = os.Chmod(existingFilepath, testCase.existingPerm)
				require.NoError(t, err)
				existingFileInfo, err = os.Stat(existingFilepath)
				require.NoError(t, err)
			}
			if testCase.directory {
				err := os.Mkdir(databaseFilepath, 0o700)
				require.NoError(t, err)
			}
			db := &Database{
				filepath: databaseFilepath,
				data:     testCase.data,
			}

			err := db.write()

			if testCase.errMessage != "" {
				assert.ErrorContains(t, err, testCase.errMessage)
			} else {
				assert.NoError(t, err)
			}

			// No temporary file is left behind.
			for _, filePath := range [...]string{databaseFilepath, existingFilepath} {
				entries, err := os.ReadDir(filepath.Dir(filePath))
				require.NoError(t, err)
				require.Len(t, entries, 1)
				assert.Equal(t, filepath.Base(filePath), entries[0].Name())
			}

			if testCase.content == "" {
				return
			}
			content, err := os.ReadFile(databaseFilepath)
			require.NoError(t, err)
			assert.Equal(t, testCase.content, string(content))

			if testCase.symlink {
				linkInfo, err := os.Lstat(databaseFilepath)
				require.NoError(t, err)
				assert.Equal(t, fs.ModeSymlink, linkInfo.Mode().Type())
			}

			if existingFileInfo == nil {
				return
			}
			fileInfo, err := os.Stat(existingFilepath)
			require.NoError(t, err)
			assert.Equal(t, testCase.replaced, !os.SameFile(existingFileInfo, fileInfo))
			// Windows does not support POSIX permission bits.
			if runtime.GOOS != "windows" {
				assert.Equal(t, testCase.existingPerm, fileInfo.Mode().Perm())
			}
		})
	}
}

func Test_isRenameUnsupported(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		err         error
		unsupported bool
	}{
		"no_error": {},
		"bind_mounted_file": {
			err:         &os.LinkError{Op: "rename", Err: syscall.EBUSY},
			unsupported: true,
		},
		"read_only_file_system": {
			err:         &fs.PathError{Op: "open", Err: syscall.EROFS},
			unsupported: true,
		},
		"permission_denied": {
			err:         &fs.PathError{Op: "open", Err: syscall.EACCES},
			unsupported: true,
		},
		"cross_device": {
			err:         &os.LinkError{Op: "rename", Err: syscall.EXDEV},
			unsupported: true,
		},
		// Writing the file in place could lose it, for example if
		// no space is left on the device.
		"no_space_left": {
			err: &fs.PathError{Op: "open", Err: syscall.ENOSPC},
		},
		"other_error": {
			err: errors.New("other"),
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			unsupported := isRenameUnsupported(testCase.err)

			assert.Equal(t, testCase.unsupported, unsupported)
		})
	}
}
