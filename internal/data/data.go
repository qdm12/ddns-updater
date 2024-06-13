package data

import (
	"context"
	"sync"

	"github.com/qdm12/ddns-updater/internal/records"
)

type Database struct {
	data []records.Record
	sync.RWMutex
	persistentDB PersistentDatabase
}

// NewDatabase creates a new in memory database.
func NewDatabase(data []records.Record, persistentDB PersistentDatabase) *Database {
	return &Database{
		data:         data,
		persistentDB: persistentDB,
	}
}

func (db *Database) String() string {
	return "database"
}

func (db *Database) Start(_ context.Context) (_ <-chan error, err error) {
	return nil, nil //nolint:nilnil
}

func (db *Database) Stop() (err error) {
	db.Lock() // ensure write operation finishes
	defer db.Unlock()
	return db.persistentDB.Close()
}
