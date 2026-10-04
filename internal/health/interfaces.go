package health

import (
	"context"
	"net"

	"github.com/qdm12/ddns-updater/internal/records"
)

type AllSelecter interface {
	SelectAll() (records []records.Record)
}

type LookupIPer interface {
	LookupIP(ctx context.Context, network, host string) (ips []net.IP, err error)
}

// ttlProvider is implemented by providers that know the TTL of the records they manage.
type ttlProvider interface {
	TTL() (ttl *uint32)
}

type Logger interface {
	Info(s string)
	Warn(s string)
	Error(s string)
}
