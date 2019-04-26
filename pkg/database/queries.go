package database

import (
	"time"
)

/* All these methods must be called by a single go routine as they are not
thread safe because of SQLite */

func (db *DB) UpdateIPTime(domain, host, ip string) (err error) {
	_, err = db.Exec(
		`UPDATE updates_ips
		SET t_last = ?
		WHERE domain = ? AND host = ? AND ip = ? AND current = 1`,
		time.Now(),
		domain,
		host,
		ip,
	)
	return err
}

func (db *DB) StoreNewIP(domain, host, ip string) (err error) {
	// Disable the current IP
	_, err = db.Exec(
		`UPDATE updates_ips
		SET current = 0
		WHERE domain = ? AND host = ? AND current = 1`,
		domain,
		host,
	)
	if err != nil {
		return err
	}
	// Inserts new IP
	_, err = db.Exec(
		`INSERT INTO updates_ips(domain,host,ip,t_new,t_last,current)
		VALUES(?, ?, ?, ?, ?, ?);`,
		domain,
		host,
		ip,
		time.Now(),
		time.Now(),
		1,
	)
	return err
}

func (db *DB) GetIps(domain, host string) (ips []string, tNew time.Time, err error) {
	rows, err := db.Query(
		`SELECT ip, t_new
		FROM updates_ips
		WHERE domain = ? AND host = ?
		ORDER BY t_new DESC`,
		domain,
		host,
	)
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
