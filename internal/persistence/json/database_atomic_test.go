package json

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteAtomicallyReplacesDatabase(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	databasePath := filepath.Join(directory, "updates.json")
	require.NoError(t, os.WriteFile(databasePath, []byte(`{"records":[]}`), 0o600))

	db := &Database{
		filepath: databasePath,
		data: dataModel{
			Records: []record{
				{
					Domain: "example.com",
					Owner:  "home",
				},
			},
		},
	}

	require.NoError(t, db.write())

	content, err := os.ReadFile(databasePath)
	require.NoError(t, err)

	var persisted dataModel
	require.NoError(t, json.Unmarshal(content, &persisted))
	require.Len(t, persisted.Records, 1)
	assert.Equal(t, "example.com", persisted.Records[0].Domain)
	assert.Equal(t, "home", persisted.Records[0].Owner)

	tempFiles, err := filepath.Glob(filepath.Join(directory, ".updates.json.tmp-*"))
	require.NoError(t, err)
	assert.Empty(t, tempFiles)
}

func TestWritePreservesExistingPermissions(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits are not supported on Windows")
	}

	path := filepath.Join(t.TempDir(), "updates.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"records":[]}`), 0o600))
	require.NoError(t, os.Chmod(path, 0o640))
	db := &Database{filepath: path}
	require.NoError(t, db.write())

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o640), info.Mode().Perm())
}

func TestWriteEncodingFailurePreservesPreviousDatabase(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	path := filepath.Join(directory, "updates.json")
	previous := []byte(`{"records":[]}`)
	require.NoError(t, os.WriteFile(path, previous, 0o600))
	db := &Database{
		filepath: path,
		data: dataModel{Records: []record{{
			Events: []models.HistoryEvent{{Time: time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)}},
		}}},
	}
	require.ErrorContains(t, db.write(), "encoding data to file")

	content, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, previous, content)
	temporaryFiles, err := filepath.Glob(filepath.Join(directory, ".updates.json.tmp-*"))
	require.NoError(t, err)
	assert.Empty(t, temporaryFiles)
}

func TestWriteRenameFailureRemovesTemporaryFile(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	path := filepath.Join(directory, "updates.json")
	require.NoError(t, os.Mkdir(path, 0o700))
	db := &Database{filepath: path}
	require.ErrorContains(t, db.write(), "replacing database file")

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
	temporaryFiles, err := filepath.Glob(filepath.Join(directory, ".updates.json.tmp-*"))
	require.NoError(t, err)
	assert.Empty(t, temporaryFiles)
}
