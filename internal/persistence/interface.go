package persistence

import (
	"net"
	"time"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/persistence/json"
	"github.com/qdm12/ddns-updater/internal/persistence/sqlite"
)

type Database interface {
	Close() error
	StoreNewIP(domain, host string, ip net.IP) (err error)
	GetIPs(domain, host string) (ips []net.IP, successTime time.Time, err error)
	GetAllDomainsHosts() (domainshosts []models.DomainHost, err error)
	SetSuccessTime(domain, host string, successTime time.Time) error
}

func NewSQLite(dataDir string) (Database, error) {
	return sqlite.NewDatabase(dataDir)
}

func NewJSON(dataDir string) (Database, error) {
	return json.NewDatabase(dataDir)
}
