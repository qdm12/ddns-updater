package data

import (
	"net"
	"time"

	"github.com/qdm12/ddns-updater/internal/models"
)

type PersistentDatabase interface {
	Close() error
	StoreNewIP(domain, host string, ip net.IP, t time.Time) (err error)
	GetEvents(domain, host string) (events []models.HistoryEvent, err error)
	Check() error
}
