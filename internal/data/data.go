package data

import (
	"sync"

	"github.com/qdm12/ddns-updater/internal/records"
)

type database struct {
	data []records.Record
	sync.RWMutex
	persistentDB PersistentDatabase
}

// NewDatabase creates a new in memory database.
func NewDatabase(data []records.Record, persistentDB PersistentDatabase) *database {
	return &database{
		data:         data,
		persistentDB: persistentDB,
	}
}
