package dns

import (
	"context"
	"net"
)

func (f *fetcher) IP(ctx context.Context) (publicIP net.IP, err error) {
	return f.ip(ctx, f.client)
}

func (f *fetcher) IP4(ctx context.Context) (publicIP net.IP, err error) {
	return f.ip(ctx, f.client4)
}

func (f *fetcher) IP6(ctx context.Context) (publicIP net.IP, err error) {
	return f.ip(ctx, f.client6)
}

func (f *fetcher) ip(ctx context.Context, client Client) (
	publicIP net.IP, err error) {
	f.ring.mutex.Lock()
	provider := f.ring.providers[f.ring.index]
	f.ring.index++
	if f.ring.index == len(f.ring.providers) {
		f.ring.index = 0
	}
	f.ring.mutex.Unlock()

	return fetch(ctx, client, provider.data())
}
