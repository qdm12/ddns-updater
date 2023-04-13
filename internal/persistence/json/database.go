package json

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/qdm12/golibs/files"
)

type Database struct {
	data        dataModel
	filepath    string
	fileManager files.FileManager
	sync.RWMutex
}

func (db *Database) Close() error {
	db.Lock() // ensure a write operation finishes
	defer db.Unlock()
	return nil
}

// NewDatabase opens or creates the JSON file database.
func NewDatabase(dataDir string) (*Database, error) {
	db := Database{
		filepath:    dataDir + "/updates.json",
		fileManager: files.NewFileManager(),
	}
	exists, err := db.fileManager.FileExists(db.filepath)
	if err != nil {
		return nil, err
	}

	if !exists {
		data, err := json.Marshal(db.data)
		if err != nil {
			return nil, err
		}
		err = db.fileManager.WriteToFile(db.filepath, data)
		if err != nil {
			return nil, err
		}
		return &db, nil
	}
	data, err := db.fileManager.ReadFile(db.filepath)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(data, &db.data)
	if err != nil {
		return nil, err
	}
	err = db.Check()
	if err != nil {
		return nil, fmt.Errorf("%s validation error: %w", db.filepath, err)
	}
	return &db, nil
}

var (
	ErrDomainEmpty         = errors.New("domain is empty")
	ErrHostIsEmpty         = errors.New("host is empty")
	ErrIPRecordsMisordered = errors.New("IP records are not ordered correctly by time")
	ErrIPEmpty             = errors.New("IP is empty")
	ErrIPTimeEmpty         = errors.New("time of IP is empty")
)

func (db *Database) Check() error {
	for _, record := range db.data.Records {
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
			case event.IP == nil:
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
	data, err := json.MarshalIndent(db.data, "", "  ")
	if err != nil {
		return err
	}
	return db.fileManager.WriteToFile(db.filepath, data)
}
