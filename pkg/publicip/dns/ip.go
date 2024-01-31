package dns

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/netip"
	"sync/atomic"

	"github.com/miekg/dns"
)

var (
	ErrIPNotFoundForVersion = errors.New("IP addresses found but not for IP version")
)

func (f *Fetcher) IP(ctx context.Context) (publicIP netip.Addr, err error) {
	publicIPs, err := f.ip(ctx, "tcp")
	if err != nil {
		return netip.Addr{}, err
	}
	return publicIPs[0], nil
}

func (f *Fetcher) IP4(ctx context.Context) (publicIP netip.Addr, err error) {
	publicIPs, err := f.ip(ctx, "tcp4")
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
	publicIPs, err := f.ip(ctx, "tcp6")
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

func (f *Fetcher) ip(ctx context.Context, network string) (
	publicIPs []netip.Addr, err error) {
	index := int(atomic.AddUint32(f.ring.counter, 1)) % len(f.ring.providers)
	providerData := f.ring.providers[index].data()

	client := &dns.Client{
		Net:         network + "-tls",
		Timeout:     f.timeout,
		DialTimeout: f.timeout,
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
			ServerName: providerData.TLSName,
		},
	}

	return fetch(ctx, client, network, providerData)
}
