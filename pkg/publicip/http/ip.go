package http

import (
	"context"
	"net"
)

func (f *fetcher) IP(ctx context.Context) (publicIP net.IP, err error) {
	return f.ip(ctx, &f.ip4or6)
}

func (f *fetcher) IP4(ctx context.Context) (publicIP net.IP, err error) {
	return f.ip(ctx, &f.ip4)
}

func (f *fetcher) IP6(ctx context.Context) (publicIP net.IP, err error) {
	return f.ip(ctx, &f.ip6)
}

func (f *fetcher) ip(ctx context.Context, ring *urlsRing) (
	publicIP net.IP, err error) {
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

	return fetch(ctx, f.client, url)
}
