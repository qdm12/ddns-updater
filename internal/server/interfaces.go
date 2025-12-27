package server

import (
	"context"
	"net/netip"

	"github.com/qdm12/ddns-updater/internal/records"
)

type Database interface {
	SelectAll() (records []records.Record)
}

type UpdateForcer interface {
	ForceUpdate(ctx context.Context) (errors []error)
}

type Logger interface {
	Info(s string)
	Warn(s string)
	Error(s string)
}

type PublicIPFetcher interface {
	IP4(ctx context.Context) (ipv4 netip.Addr, err error)
	IP6(ctx context.Context) (ipv6 netip.Addr, err error)
}
