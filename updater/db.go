package main

import (
	"database/sql"
	"strings"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// A sqlite database is used to store previous IPs, when re launching the program.

// DB contains the database connection pool pointer.
// It is used so that methods are declared on it, in order
// to mock the database easily, through the help of the Datastore interface
type DB struct {
	db *sql.DB
	m  sync.Mutex
}

// initializes the database schema if it does not exist yet.
func initializeDatabase(dataDir string) (*DB, error) {
	dataDir = strings.TrimSuffix(dataDir, "/")
	db, err := sql.Open("sqlite3", dataDir+"/updates.db")
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(
		`CREATE TABLE IF NOT EXISTS updates_ips (
		domain TEXT NOT NULL,
		host TEXT NOT NULL,
		ip TEXT NOT NULL,
		t_new DATETIME NOT NULL,
		t_last DATETIME NOT NULL,
		current INTEGER DEFAULT 1 NOT NULL,
		PRIMARY KEY(domain, host, ip, t_new)
		);`)
	return &DB{db: db}, err
}

func (dbContainer *DB) updateIPTime(domain, host, ip string) (err error) {
	dbContainer.m.Lock()
	_, err = dbContainer.db.Exec(
		`UPDATE updates_ips
		SET t_last = ?
		WHERE domain = ? AND host = ? AND ip = ? AND current = 1`,
		time.Now(),
		domain,
		host,
		ip,
	)
	dbContainer.m.Unlock()
	return err
}

func (dbContainer *DB) storeNewIP(domain, host, ip string) (err error) {
	// Disable the current IP
	dbContainer.m.Lock()
	_, err = dbContainer.db.Exec(
		`UPDATE updates_ips
		SET current = 0
		WHERE domain = ? AND host = ? AND current = 1`,
		domain,
		host,
	)
	dbContainer.m.Unlock()
	if err != nil {
		return err
	}
	// Inserts new IP
	dbContainer.m.Lock()
	_, err = dbContainer.db.Exec(
		`INSERT INTO updates_ips(domain,host,ip,t_new,t_last,current)
		VALUES(?, ?, ?, ?, ?, ?);`,
		domain,
		host,
		ip,
		time.Now(),
		time.Now(),
		1,
	)
	dbContainer.m.Unlock()
	return err
}

func (dbContainer *DB) getIps(domain, host string) (ips []string, tNew time.Time, err error) {
	dbContainer.m.Lock()
	rows, err := dbContainer.db.Query(
		`SELECT ip, t_new
		FROM updates_ips
		WHERE domain = ? AND host = ?
		ORDER BY t_new DESC`,
		domain,
		host,
	)
	dbContainer.m.Unlock()
	if err != nil {
		return nil, tNew, err
	}
	defer rows.Close()
	var ip string
	var t time.Time
	var tNewSet bool
	for rows.Next() {
		err = rows.Scan(&ip, &t)
		if err != nil {
			return ips, tNew, err
		}
		if !tNewSet {
			tNew = t
			tNewSet = true
		}
		ips = append(ips, ip)
	}
	if !tNewSet {
		tNew = time.Now()
	}
	return ips, tNew, rows.Err()
}
