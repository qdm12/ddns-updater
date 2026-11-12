package backup

import (
	"cmp"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// pruneBackups removes the oldest backup zip files from the directory given,
// so that at most keep of them remain. It does nothing if keep is zero.
func pruneBackups(directory string, keep uint) error {
	if keep == 0 {
		return nil
	}

	entries, err := os.ReadDir(directory)
	if err != nil {
		return fmt.Errorf("reading directory: %w", err)
	}

	type backupFile struct {
		name    string
		modTime time.Time
	}
	backupFiles := make([]backupFile, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		isBackup := entry.Type().IsRegular() &&
			strings.HasPrefix(name, zipFileNamePrefix) && strings.HasSuffix(name, zipFileNameSuffix)
		if !isBackup {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return fmt.Errorf("getting file information: %w", err)
		}
		backupFiles = append(backupFiles, backupFile{name: name, modTime: info.ModTime()})
	}

	if uint(len(backupFiles)) <= keep {
		return nil
	}

	// The modification time is used and not the timestamp in the file name,
	// since that timestamp overflows on 32 bit platforms.
	slices.SortFunc(backupFiles, func(left, right backupFile) int {
		return cmp.Or(left.modTime.Compare(right.modTime), strings.Compare(left.name, right.name))
	})

	var errs []error
	for _, oldest := range backupFiles[:uint(len(backupFiles))-keep] {
		err = os.Remove(filepath.Join(directory, oldest.name))
		if err != nil {
			errs = append(errs, fmt.Errorf("removing file: %w", err))
		}
	}
	return errors.Join(errs...)
}
