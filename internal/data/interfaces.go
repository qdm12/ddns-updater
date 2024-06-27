package data

import (
	"net/netip"
	"time"
)

type PersistentDatabase interface {
	Close() error
	StoreNewIP(domain, owner string, ip netip.Addr, t time.Time) (err error)
}
