package json

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Database struct {
	data     dataModel
	filepath string
	sync.RWMutex
}

func (db *Database) Close() error {
	db.Lock() // ensure a write operation finishes
	defer db.Unlock()
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
		const perm fs.FileMode = 0700
		err = os.MkdirAll(filepath.Dir(filePath), perm)
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

	err = checkData(data)
	if err != nil {
		return nil, fmt.Errorf("%s validation error: %w", filePath, err)
	}

	return &Database{
		data:     data,
		filepath: filePath,
	}, nil
}

var (
	ErrDomainEmpty         = errors.New("domain is empty")
	ErrHostIsEmpty         = errors.New("host is empty")
	ErrIPRecordsMisordered = errors.New("IP records are not ordered correctly by time")
	ErrIPEmpty             = errors.New("IP is empty")
	ErrIPTimeEmpty         = errors.New("time of IP is empty")
)

func checkData(data dataModel) error {
	for _, record := range data.Records {
		switch {
		case record.Domain == "":
			return fmt.Errorf("%w: for record %s", ErrDomainEmpty, record)
		case record.Host == "":
			return fmt.Errorf("%w: for record %s", ErrHostIsEmpty, record)
		}
		var t time.Time
		for i, event := range record.Events {
			if event.Time.Before(t) {
				return fmt.Errorf("%w", ErrIPRecordsMisordered)
			}
			t = event.Time
			switch {
			case !event.IP.IsValid():
				return fmt.Errorf("%w: IP %d of %d for record %s",
					ErrIPEmpty, i+1, len(record.Events), record)
			case event.Time.IsZero():
				return fmt.Errorf("%w: IP %d of %d for record %s",
					ErrIPTimeEmpty, i+1, len(record.Events), record)
			}
		}
	}
	return nil
}

func (db *Database) write() error {
	const createPerms fs.FileMode = 0600
	file, err := os.OpenFile(db.filepath, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, createPerms)
	if err != nil {
		return fmt.Errorf("opening file: %w", err)
	}

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	err = encoder.Encode(db.data)
	if err != nil {
		_ = file.Close()
		return fmt.Errorf("encoding data to file: %w", err)
	}

	err = file.Close()
	if err != nil {
		return fmt.Errorf("closing database file: %w", err)
	}
	return nil
}
