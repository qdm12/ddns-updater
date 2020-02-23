package json

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/qdm12/golibs/files"
)

type database struct {
	data        dataModel
	filepath    string
	fileManager files.FileManager
	sync.RWMutex
}

func (db *database) Close() error {
	db.Lock() // ensure a write operation finishes
	defer db.Unlock()
	return nil
}

// NewDatabase opens or creates the JSON file database.
func NewDatabase(dataDir string) (*database, error) {
	db := database{
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
		if err := db.fileManager.WriteToFile(db.filepath, data); err != nil {
			return nil, err
		}
		return &db, nil
	}
	data, err := db.fileManager.ReadFile(db.filepath)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &db.data); err != nil {
		return nil, err
	}
	if err := db.Check(); err != nil {
		return nil, err
	}
	return &db, nil
}

func (db *database) Check() error {
	for _, record := range db.data.Records {
		switch {
		case len(record.Domain) == 0:
			return fmt.Errorf("domain is empty for record %s", record)
		case len(record.Host) == 0:
			return fmt.Errorf("host is empty for record %s", record)
		}
		var t time.Time
		for i, event := range record.Events {
			if event.Time.Before(t) {
				return fmt.Errorf("IP records are not ordered correctly by time")
			}
			t = event.Time
			switch {
			case event.IP == nil:
				return fmt.Errorf("IP %d of %d is empty for record %s", i+1, len(record.Events), record)
			case event.Time.IsZero():
				return fmt.Errorf("Time of IP %d of %d is empty for record %s", i+1, len(record.Events), record)
			}
		}
	}
	return nil
}

func (db *database) write() error {
	data, err := json.MarshalIndent(db.data, "", "  ")
	if err != nil {
		return err
	}
	return db.fileManager.WriteToFile(db.filepath, data)
}
