package dns

import (
	"context"
	"net"
	"sync/atomic"
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
	index := int(atomic.AddUint32(f.ring.counter, 1)) % len(f.ring.providers)
	provider := f.ring.providers[index]
	return fetch(ctx, client, provider.data())
}
