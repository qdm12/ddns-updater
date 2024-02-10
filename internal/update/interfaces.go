package update

import (
	"context"
	"net"
	"net/netip"

	"github.com/qdm12/ddns-updater/internal/healthchecksio"
	"github.com/qdm12/ddns-updater/internal/records"
)

type PublicIPFetcher interface {
	IP(ctx context.Context) (netip.Addr, error)
	IP4(ctx context.Context) (netip.Addr, error)
	IP6(ctx context.Context) (netip.Addr, error)
}

type UpdaterInterface interface {
	Update(ctx context.Context, recordID uint, ip netip.Addr) (err error)
}

type Database interface {
	Select(recordID uint) (record records.Record, err error)
	SelectAll() (records []records.Record)
	Update(recordID uint, record records.Record) (err error)
}

type LookupIPer interface {
	LookupIP(ctx context.Context, network, host string) (ips []net.IP, err error)
}

type ShoutrrrClient interface {
	Notify(message string)
}

type Logger interface {
	DebugLogger
	Info(s string)
	Warn(s string)
	Error(s string)
}

type HealthchecksIOClient interface {
	Ping(ctx context.Context, state healthchecksio.State) (err error)
}
