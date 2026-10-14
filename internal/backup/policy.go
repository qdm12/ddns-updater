package backup

import (
	"archive/zip"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// pruneBackups deletes the oldest backups in outputDir so that at most
// keep backups remain. Files which are not readable backup archives are
// neither counted nor deleted. A keep value of zero keeps all backups.
func pruneBackups(outputDir string, keep int) error {
	if keep <= 0 {
		return nil
	}

	entries, err := os.ReadDir(outputDir)
	if err != nil {
		return err
	}

	type backup struct {
		path    string
		modTime time.Time
	}
	backups := make([]backup, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasPrefix(name, backupFilePrefix) ||
			!strings.HasSuffix(name, ".zip") {
			continue
		}
		path := filepath.Join(outputDir, name)
		if !isBackupArchive(path) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		backups = append(backups, backup{path: path, modTime: info.ModTime()})
	}

	if len(backups) <= keep {
		return nil
	}

	sort.SliceStable(backups, func(i, j int) bool {
		return backups[i].modTime.Before(backups[j].modTime)
	})
	for _, backup := range backups[:len(backups)-keep] {
		err = os.Remove(backup.path)
		if err != nil {
			return err
		}
	}
	return nil
}

// isBackupArchive reports whether the file at path is a readable
// ZIP archive containing the updates file. Every file in the archive
// is read in full to verify its checksum.
func isBackupArchive(path string) bool {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return false
	}
	defer reader.Close()
	hasUpdates := false
	for _, file := range reader.File {
		if !isReadable(file) {
			return false
		}
		if file.Name == updatesArchiveName {
			hasUpdates = true
		}
	}
	return hasUpdates
}

// maxArchivedFileSize limits how much is decompressed per archived file,
// which is far above the size of the data files.
const maxArchivedFileSize = 64 << 20

func isReadable(file *zip.File) bool {
	reader, err := file.Open()
	if err != nil {
		return false
	}
	defer reader.Close()
	n, err := io.CopyN(io.Discard, reader, maxArchivedFileSize+1)
	return errors.Is(err, io.EOF) && n <= maxArchivedFileSize
}
