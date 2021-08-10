package data

import (
	"sync"

	"github.com/qdm12/ddns-updater/internal/persistence"
	"github.com/qdm12/ddns-updater/internal/records"
)

var _ Database = (*database)(nil)

type Database interface {
	EphemeralDatabase
	PersistentDatabase
}

type database struct {
	data []records.Record
	sync.RWMutex
	persistentDB persistence.Database
}

// NewDatabase creates a new in memory database.
func NewDatabase(data []records.Record, persistentDB persistence.Database) Database {
	return &database{
		data:         data,
		persistentDB: persistentDB,
	}
}
