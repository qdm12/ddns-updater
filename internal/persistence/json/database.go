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

	"github.com/qdm12/ddns-updater/internal/models"
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

var (
	ErrDomainEmpty         = errors.New("domain is empty")
	ErrOwnerNotSet         = errors.New("owner is not set")
	ErrIPRecordsMisordered = errors.New("IP records are not ordered correctly by time")
	ErrIPEmpty             = errors.New("IP is empty")
	ErrIPTimeEmpty         = errors.New("time of IP is empty")
)

func checkData(data dataModel) (err error) {
	for i, record := range data.Records {
		switch {
		case record.Domain == "":
			return fmt.Errorf("%w: for record %d of %d", ErrDomainEmpty,
				i+1, len(data.Records))
		case record.Owner == "":
			return fmt.Errorf("%w: for record %d of %d with domain %s",
				ErrOwnerNotSet, i+1, len(data.Records), record.Domain)
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
			return fmt.Errorf("%w: for event %d of %d (IP %s)",
				ErrIPTimeEmpty, i+1, len(events), event.IP)
		case event.Time.Before(previousEventTime):
			return fmt.Errorf("%w: event %d of %d (IP %s and time %s) "+
				" is before event %d of %d (IP %s and time %s)",
				ErrIPRecordsMisordered, i+1, len(events), event.IP, event.Time,
				i, len(events), events[i-1].IP, events[i-1].Time)
		case !event.IP.IsValid():
			return fmt.Errorf("%w: for event %d of %d",
				ErrIPEmpty, i+1, len(events))
		}
		previousEventTime = event.Time
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
