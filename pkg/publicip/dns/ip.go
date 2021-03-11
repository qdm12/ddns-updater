package dns

import (
	"context"
	"net"
)

func (f *fetcher) IP(ctx context.Context) (publicIP net.IP, err error) {
	return f.ip(ctx, f.ip4or6)
}

func (f *fetcher) IP4(ctx context.Context) (publicIP net.IP, err error) {
	return f.ip(ctx, f.ip4)
}

func (f *fetcher) IP6(ctx context.Context) (publicIP net.IP, err error) {
	return f.ip(ctx, f.ip6)
}

func (f *fetcher) ip(ctx context.Context, providerMap map[Provider]providerObj) (
	publicIP net.IP, err error) {
	var provider Provider
	if len(providerMap) > 0 {
		f.mutex.Lock()
		provider = f.providers[f.index]
		f.index++
		if f.index == len(f.providers) {
			f.index = 0
		}
		f.mutex.Unlock()
	} else {
		provider = f.providers[f.index] // f.index is never changed
	}

	obj := providerMap[provider]
	return fetch(ctx, obj.resolver, obj.txtRecord)
}
