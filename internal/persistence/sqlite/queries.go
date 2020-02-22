package sqlite

import (
	"net"
	"time"
)

/* Access to SQLite is NOT thread safe so we use a mutex */

// StoreNewIP stores a new IP address for a certain
// domain and host.
func (db *database) StoreNewIP(domain, host string, ip net.IP) (err error) {
	// Disable the current IP
	db.Lock()
	defer db.Unlock()
	_, err = db.sqlite.Exec(
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
	_, err = db.sqlite.Exec(
		`INSERT INTO updates_ips(domain,host,ip,t_new,t_last,current)
		VALUES(?, ?, ?, ?, ?, ?);`,
		domain,
		host,
		ip.String(),
		time.Now(),
		time.Now(), // unneeded but it's hard to modify tables in sqlite
		1,
	)
	return err
}

// GetIPs gets all the IP addresses history for a certain domain and host, in the order
// from oldest to newest
func (db *database) GetIPs(domain, host string) (ips []net.IP, tNew time.Time, err error) {
	db.Lock()
	defer db.Unlock()
	rows, err := db.sqlite.Query(
		`SELECT ip, t_new
		FROM updates_ips
		WHERE domain = ? AND host = ?
		ORDER BY t_new ASC`,
		domain,
		host,
	)
	if err != nil {
		return nil, tNew, err
	}
	defer func() {
		err = rows.Close()
	}()
	for rows.Next() {
		var ip string
		var t time.Time
		if err := rows.Scan(&ip, &t); err != nil {
			return nil, tNew, err
		}
		if tNew.IsZero() {
			tNew = t
		}
		ips = append(ips, net.ParseIP(ip))
	}
	if tNew.IsZero() {
		tNew = time.Now()
	}
	return ips, tNew, rows.Err()
}
