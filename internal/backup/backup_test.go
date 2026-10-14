package backup

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestZipFilesUsesArchiveNames(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	configPath := filepath.Join(directory, "custom-name.json")
	updatesPath := filepath.Join(directory, "updates.json")
	require.NoError(t, os.WriteFile(configPath, []byte(`{"settings":[]}`), 0o600))
	require.NoError(t, os.WriteFile(updatesPath, []byte(`{"records":[]}`), 0o600))
	outputPath := filepath.Join(directory, backupFilePrefix+"1.zip")

	err := zipFiles(outputPath,
		archiveFile{path: updatesPath, name: updatesArchiveName},
		archiveFile{path: configPath, name: configArchiveName},
	)
	require.NoError(t, err)

	reader, err := zip.OpenReader(outputPath)
	require.NoError(t, err)
	defer reader.Close()
	names := make([]string, len(reader.File))
	for i, file := range reader.File {
		names[i] = file.Name
	}
	assert.Equal(t, []string{updatesArchiveName, configArchiveName}, names)
}

func TestPruneBackupsKeepsNewestBackups(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	updatesPath := filepath.Join(directory, "updates.json")
	require.NoError(t, os.WriteFile(updatesPath, []byte(`{"records":[]}`), 0o600))

	now := time.Now()
	paths := make([]string, 3)
	for i := range paths {
		paths[i] = writeBackup(t, directory, updatesPath, now.Add(time.Duration(i)*time.Hour))
	}

	err := pruneBackups(directory, 2)
	require.NoError(t, err)

	assert.NoFileExists(t, paths[0])
	assert.FileExists(t, paths[1])
	assert.FileExists(t, paths[2])
}

func TestPruneBackupsIgnoresOtherFiles(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	updatesPath := filepath.Join(directory, "updates.json")
	otherPath := filepath.Join(directory, "other.json")
	require.NoError(t, os.WriteFile(updatesPath, []byte(`{"records":[]}`), 0o600))
	require.NoError(t, os.WriteFile(otherPath, []byte(`{}`), 0o600))

	oldTime := time.Now().Add(-time.Hour)
	brokenPath := filepath.Join(directory, backupFilePrefix+"1.zip")
	require.NoError(t, os.WriteFile(brokenPath, []byte("not a zip file"), 0o600))
	corruptedPath := filepath.Join(directory, backupFilePrefix+"3.zip")
	writeCorruptedBackup(t, corruptedPath)
	withoutUpdatesPath := filepath.Join(directory, backupFilePrefix+"2.zip")
	require.NoError(t, zipFiles(withoutUpdatesPath, archiveFile{path: otherPath, name: "other.json"}))
	for _, path := range []string{brokenPath, corruptedPath, withoutUpdatesPath, otherPath} {
		require.NoError(t, os.Chtimes(path, oldTime, oldTime))
	}
	backupPath := writeBackup(t, directory, updatesPath, time.Now())

	err := pruneBackups(directory, 1)
	require.NoError(t, err)

	assert.FileExists(t, brokenPath)
	assert.FileExists(t, corruptedPath)
	assert.FileExists(t, withoutUpdatesPath)
	assert.FileExists(t, otherPath)
	assert.FileExists(t, backupPath)
}

func TestPruneBackupsKeepZeroKeepsAll(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	updatesPath := filepath.Join(directory, "updates.json")
	require.NoError(t, os.WriteFile(updatesPath, []byte(`{"records":[]}`), 0o600))
	backupPath := writeBackup(t, directory, updatesPath, time.Now())

	err := pruneBackups(directory, 0)
	require.NoError(t, err)

	assert.FileExists(t, backupPath)
}

func writeBackup(t *testing.T, directory, updatesPath string, modTime time.Time) (path string) {
	t.Helper()
	path = filepath.Join(directory, backupFilePrefix+modTime.Format("150405.000000000")+".zip")
	require.NoError(t, zipFiles(path, archiveFile{path: updatesPath, name: updatesArchiveName}))
	require.NoError(t, os.Chtimes(path, modTime, modTime))
	return path
}

// writeCorruptedBackup writes an uncompressed backup archive and then
// changes its content so that its checksum no longer matches.
func writeCorruptedBackup(t *testing.T, path string) {
	t.Helper()
	const content = "updates content"
	buffer := bytes.NewBuffer(nil)
	writer := zip.NewWriter(buffer)
	fileWriter, err := writer.CreateHeader(&zip.FileHeader{
		Name:   updatesArchiveName,
		Method: zip.Store,
	})
	require.NoError(t, err)
	_, err = fileWriter.Write([]byte(content))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	data := buffer.Bytes()
	index := bytes.Index(data, []byte(content))
	require.NotEqual(t, -1, index)
	data[index] ^= 0xff
	require.NoError(t, os.WriteFile(path, data, 0o600))
}
