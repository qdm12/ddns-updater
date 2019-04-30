package database

import (
	"database/sql"
	"strings"
	"sync"
)

// A sqlite database is used to store previous IPs, when re launching the program.

// DB contains the database connection pool pointer.
// It is used so that methods are declared on it, in order
// to mock the database easily, through the help of the Datastore interface
// WARNING: Use in one single go routine, it is not thread safe !
type DB struct {
	sqlite *sql.DB
	sync.Mutex
}

// NewDb opens or creates the database if necessary.
func NewDb(dataDir string) (*DB, error) {
	dataDir = strings.TrimSuffix(dataDir, "/")
	sqlite, err := sql.Open("sqlite3", dataDir+"/updates.db")
	if err != nil {
		return nil, err
	}
	_, err = sqlite.Exec(
		`CREATE TABLE IF NOT EXISTS updates_ips (
		domain TEXT NOT NULL,
		host TEXT NOT NULL,
		ip TEXT NOT NULL,
		t_new DATETIME NOT NULL,
		t_last DATETIME NOT NULL,
		current INTEGER DEFAULT 1 NOT NULL,
		PRIMARY KEY(domain, host, ip, t_new)
		);`)
	return &DB{sqlite: sqlite}, err
}
