package data

import (
	"sync"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/persistence"
)

type Database interface {
	Close() error
	Insert(record models.Record) (id int)
	Select(id int) (record models.Record, err error)
	SelectAll() (records []models.Record)
	Update(id int, record models.Record) error
	// From persistence database
	GetEvents(domain, host string) (events []models.HistoryEvent, err error)
}

type database struct {
	data []models.Record
	sync.RWMutex
	persistentDB persistence.Database
}

// NewDatabase creates a new in memory database
func NewDatabase(data []models.Record, persistentDB persistence.Database) Database {
	return &database{
		data:         data,
		persistentDB: persistentDB,
	}
}

func (db *database) Close() error {
	db.Lock() // ensure write operation finishes
	defer db.Unlock()
	return db.persistentDB.Close()
}
