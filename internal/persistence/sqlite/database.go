package sqlite

import (
	"database/sql"
	"sync"
)

type database struct {
	sqlite *sql.DB
	sync.Mutex
}

func (db *database) Close() error {
	return db.sqlite.Close()
}

// NewDatabase opens or creates the database if necessary.
func NewDatabase(dataDir string) (*database, error) {
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
	return &database{sqlite: sqlite}, err
}
