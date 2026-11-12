package json

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/qdm12/ddns-updater/internal/models"
)

type Database struct {
	data     dataModel
	filepath string
	mutex    sync.RWMutex
}

func (db *Database) Close() error {
	db.mutex.Lock() // ensure a write operation finishes
	defer db.mutex.Unlock()
	return nil
}

// NewDatabase opens or creates the JSON file database.
func NewDatabase(dataDir string) (*Database, error) {
	filePath := filepath.Join(dataDir, "updates.json")

	file, err := os.Open(filePath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("reading file: %w", err)
		}
		const dirPerm = os.FileMode(0o777)
		err = os.MkdirAll(filepath.Dir(filePath), dirPerm)
		if err != nil {
			return nil, fmt.Errorf("creating data directory: %w", err)
		}
		return &Database{filepath: filePath}, nil
	}

	stat, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("stating file: %w", err)
	} else if stat.Size() == 0 { // empty file
		_ = file.Close()
		return &Database{filepath: filePath}, nil
	}

	decoder := json.NewDecoder(file)
	var data dataModel
	err = decoder.Decode(&data)
	if err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("decoding data from file: %w", err)
	}

	err = file.Close()
	if err != nil {
		return nil, fmt.Errorf("closing database file: %w", err)
	}

	// Migration from older database using "host" instead of "owner".
	for i := range data.Records {
		if data.Records[i].Owner == "" {
			data.Records[i].Owner = data.Records[i].Host
			data.Records[i].Host = ""
		}
	}

	err = checkData(data)
	if err != nil {
		return nil, fmt.Errorf("%s validation error: %w", filePath, err)
	}

	return &Database{
		data:     data,
		filepath: filePath,
	}, nil
}

func checkData(data dataModel) (err error) {
	for i, record := range data.Records {
		switch {
		case record.Domain == "":
			return fmt.Errorf("domain is empty: for record %d of %d",
				i+1, len(data.Records))
		case record.Owner == "":
			return fmt.Errorf("owner is not set: for record %d of %d with domain %s", i+1, len(data.Records), record.Domain)
		}

		err = checkHistoryEvents(record.Events)
		if err != nil {
			return fmt.Errorf("for record %d of %d with domain %s and owner %s: "+
				"history events: %w", i+1, len(data.Records),
				record.Domain, record.Owner, err)
		}
	}
	return nil
}

func checkHistoryEvents(events []models.HistoryEvent) (err error) {
	var previousEventTime time.Time
	for i, event := range events {
		switch {
		case event.Time.IsZero():
			return fmt.Errorf("time of IP is empty: for event %d of %d (IP %s)", i+1, len(events), event.IP)
		case event.Time.Before(previousEventTime):
			return fmt.Errorf("IP records are not ordered correctly by time: event %d of %d (IP %s and time %s) "+
				" is before event %d of %d (IP %s and time %s)", i+1, len(events), event.IP, event.Time,
				i, len(events), events[i-1].IP, events[i-1].Time)
		case !event.IP.IsValid():
			return fmt.Errorf("IP is empty: for event %d of %d", i+1, len(events))
		}
		previousEventTime = event.Time
	}
	return nil
}

func (db *Database) write() error {
	data, err := json.MarshalIndent(db.data, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding data: %w", err)
	}
	data = append(data, '\n')

	replaced, err := replaceFile(db.filepath, data)
	switch {
	case err != nil:
		return fmt.Errorf("replacing database file: %w", err)
	case replaced:
		return nil
	}

	const filePerm = os.FileMode(0o666)
	err = os.WriteFile(db.filepath, data, filePerm)
	if err != nil {
		return fmt.Errorf("writing database file: %w", err)
	}
	return nil
}

// replaceFile replaces the file at the path given with a file containing the
// data given, so that the file is never partially written. The data is
// written to a temporary file in the same directory, which is then renamed.
// It returns false without touching the file if the file does not exist or
// cannot be replaced by renaming a file, for the caller to write it in place.
func replaceFile(path string, data []byte) (replaced bool, err error) {
	// The file a symbolic link points to is replaced, not the symbolic link.
	path, err = filepath.EvalSymlinks(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return false, nil
	case err != nil:
		return false, fmt.Errorf("resolving path: %w", err)
	}

	fileInfo, err := os.Stat(path)
	if err != nil {
		return false, fmt.Errorf("stating file: %w", err)
	}

	temporaryFile, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*.tmp")
	switch {
	case isRenameUnsupported(err):
		return false, nil
	case err != nil:
		return false, fmt.Errorf("creating temporary file: %w", err)
	}
	temporaryPath := temporaryFile.Name()

	err = writeTemporaryFile(temporaryFile, data, fileInfo.Mode().Perm())
	if err != nil {
		_ = os.Remove(temporaryPath)
		return false, fmt.Errorf("writing temporary file: %w", err)
	}

	err = os.Rename(temporaryPath, path)
	if err != nil {
		_ = os.Remove(temporaryPath)
		if isRenameUnsupported(err) {
			return false, nil
		}
		return false, fmt.Errorf("renaming temporary file: %w", err)
	}
	return true, nil
}

// isRenameUnsupported returns true if the error means a file cannot be
// replaced by renaming another file, although it may be writable in place.
// This is the case of a bind mounted file, or of a file in a read only or
// non writable directory.
func isRenameUnsupported(err error) bool {
	return errors.Is(err, fs.ErrPermission) || errors.Is(err, syscall.EROFS) ||
		errors.Is(err, syscall.EBUSY) || errors.Is(err, syscall.EXDEV)
}

func writeTemporaryFile(temporaryFile *os.File, data []byte, perm fs.FileMode) (err error) {
	defer func() {
		closeErr := temporaryFile.Close()
		if err == nil && closeErr != nil {
			err = fmt.Errorf("closing file: %w", closeErr)
		}
	}()

	err = temporaryFile.Chmod(perm)
	if err != nil {
		return fmt.Errorf("setting file permissions: %w", err)
	}

	_, err = temporaryFile.Write(data)
	if err != nil {
		return fmt.Errorf("writing data: %w", err)
	}

	err = temporaryFile.Sync()
	if err != nil {
		return fmt.Errorf("syncing file: %w", err)
	}
	return nil
}
