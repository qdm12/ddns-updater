package http

import (
	"context"
	"net"

	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

func (f *fetcher) IP(ctx context.Context) (publicIP net.IP, err error) {
	return f.ip(ctx, ipversion.IP4or6)
}

func (f *fetcher) IP4(ctx context.Context) (publicIP net.IP, err error) {
	return f.ip(ctx, ipversion.IP4)
}

func (f *fetcher) IP6(ctx context.Context) (publicIP net.IP, err error) {
	return f.ip(ctx, ipversion.IP6)
}

func (f *fetcher) ip(ctx context.Context, version ipversion.IPVersion) (
	publicIP net.IP, err error) {
	var ring *urlsRing
	switch version {
	case ipversion.IP4:
		ring = &f.ip4
	case ipversion.IP6:
		ring = &f.ip6
	default:
		ring = &f.ip4or6
	}

	var url string
	ring.mutex.Lock()
	url = ring.urls[ring.index]
	ring.index++
	if ring.index == len(ring.urls) {
		ring.index = 0
	}
	ring.mutex.Unlock()

	ctx, cancel := context.WithTimeout(ctx, f.timeout)
	defer cancel()

	return fetch(ctx, f.client, url, version)
}
