package http

import (
	"context"
	"net"
	"sync/atomic"

	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

func (f *fetcher) IP(ctx context.Context) (publicIP net.IP, err error) {
	return f.ip(ctx, f.ip4or6, ipversion.IP4or6)
}

func (f *fetcher) IP4(ctx context.Context) (publicIP net.IP, err error) {
	return f.ip(ctx, f.ip4, ipversion.IP4)
}

func (f *fetcher) IP6(ctx context.Context) (publicIP net.IP, err error) {
	return f.ip(ctx, f.ip6, ipversion.IP6)
}

func (f *fetcher) ip(ctx context.Context, ring urlsRing, version ipversion.IPVersion) (
	publicIP net.IP, err error) {
	index := int(atomic.AddUint32(ring.counter, 1)) % len(ring.urls)
	url := ring.urls[index]

	ctx, cancel := context.WithTimeout(ctx, f.timeout)
	defer cancel()

	return fetch(ctx, f.client, url, version)
}
