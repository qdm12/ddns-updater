package dns

import (
	"context"
	"net"

	"github.com/miekg/dns"
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

func (f *fetcher) ip(ctx context.Context, client *dns.Client) (
	publicIP net.IP, err error) {
	f.mutex.Lock()
	provider := f.providers[f.index]
	f.index++
	if f.index == len(f.providers) {
		f.index = 0
	}
	f.mutex.Unlock()

	providerData := f.data[provider]

	return fetch(ctx, client, providerData)
}
