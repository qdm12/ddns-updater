package persistence

import (
	"net"
	"time"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/persistence/json"
)

type Database interface {
	Close() error
	StoreNewIP(domain, host string, ip net.IP, t time.Time) (err error)
	GetEvents(domain, host string) (events []models.HistoryEvent, err error)
	GetAllDomainsHosts() (domainshosts []models.DomainHost, err error)
	Check() error
}

func NewJSON(dataDir string) (Database, error) {
	return json.NewDatabase(dataDir)
}
