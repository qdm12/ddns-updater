package database

import (
	"database/sql"
	"sync"
	"time"
)

// A sqlite database is used to store previous IPs, when re launching the program.

// SQL represents the database store actions.
// It is implemented with the database struct and methods.
// WARNING: Use in one single go routine, it is not thread safe !
type SQL interface {
	Lock()
	Unlock()
	UpdateIPTime(domain, host, ip string) (err error)
	StoreNewIP(domain, host, ip string) (err error)
	GetIps(domain, host string) (ips []string, tNew time.Time, err error)
	Close() error
}

type database struct {
	sqlite *sql.DB
	sync.Mutex
}

func (db *database) Close() error {
	return db.sqlite.Close()
}

// NewDB opens or creates the database if necessary.
func NewDB(dataDir string) (SQL, error) {
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
