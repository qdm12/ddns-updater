package persistence

import (
	"net"
	"time"

	"github.com/qdm12/ddns-updater/internal/persistence/sqlite"
)

type Database interface {
	Close() error
	StoreNewIP(domain, host string, ip net.IP) (err error)
	GetIPs(domain, host string) (ips []net.IP, tNew time.Time, err error)
}

func NewSQLite(dataDir string) (Database, error) {
	return sqlite.NewDatabase(dataDir)
}
