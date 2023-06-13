package dns

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"sync/atomic"
)

var (
	ErrIPNotFoundForVersion = errors.New("IP addresses found but not for IP version")
)

func (f *Fetcher) IP(ctx context.Context) (publicIP netip.Addr, err error) {
	publicIPs, err := f.ip(ctx, f.client)
	if err != nil {
		return netip.Addr{}, err
	}
	return publicIPs[0], nil
}

func (f *Fetcher) IP4(ctx context.Context) (publicIP netip.Addr, err error) {
	publicIPs, err := f.ip(ctx, f.client4)
	if err != nil {
		return netip.Addr{}, err
	}

	for _, ip := range publicIPs {
		if ip.Is4() {
			return ip, nil
		}
	}
	return netip.Addr{}, fmt.Errorf("%w: ipv4", ErrIPNotFoundForVersion)
}

func (f *Fetcher) IP6(ctx context.Context) (publicIP netip.Addr, err error) {
	publicIPs, err := f.ip(ctx, f.client6)
	if err != nil {
		return netip.Addr{}, err
	}

	for _, ip := range publicIPs {
		if ip.Is6() {
			return ip, nil
		}
	}
	return netip.Addr{}, fmt.Errorf("%w: ipv6", ErrIPNotFoundForVersion)
}

func (f *Fetcher) ip(ctx context.Context, client Client) (
	publicIPs []netip.Addr, err error) {
	index := int(atomic.AddUint32(f.ring.counter, 1)) % len(f.ring.providers)
	provider := f.ring.providers[index]
	return fetch(ctx, client, provider.data())
}
