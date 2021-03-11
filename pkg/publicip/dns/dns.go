package dns

import (
	"context"
	"net"
	"sync"

	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

type Fetcher interface {
	IP(ctx context.Context) (publicIP net.IP, err error)
	IP4(ctx context.Context) (publicIP net.IP, err error)
	IP6(ctx context.Context) (publicIP net.IP, err error)
}

type fetcher struct {
	mutex     sync.RWMutex
	index     int // index in providers slice if cycle is true
	providers []Provider
	ip4or6    map[Provider]providerObj
	ip4       map[Provider]providerObj
	ip6       map[Provider]providerObj
}

type providerObj struct {
	resolver  *net.Resolver
	txtRecord string
}

func New(options ...Option) (f Fetcher, err error) {
	settings := newDefaultSettings()
	for _, option := range options {
		if err := option(&settings); err != nil {
			return nil, err
		}
	}

	fetcher := &fetcher{
		providers: settings.providers,
		ip4or6:    make(map[Provider]providerObj, len(settings.providers)),
		ip4:       make(map[Provider]providerObj, len(settings.providers)),
		ip6:       make(map[Provider]providerObj, len(settings.providers)),
	}

	dialer := &net.Dialer{
		Timeout: settings.timeout,
	}

	for _, provider := range settings.providers {
		nameserver, txtRecord4or6 := getProviderData(provider, ipversion.IP4or6)
		nameserver4, txtRecord4 := getProviderData(provider, ipversion.IP4)
		nameserver6, txtRecord6 := getProviderData(provider, ipversion.IP6)

		fetcher.ip4or6[provider] = providerObj{
			resolver:  newResolver(dialer, "udp", nameserver),
			txtRecord: txtRecord4or6,
		}

		fetcher.ip4[provider] = providerObj{
			resolver:  newResolver(dialer, "udp4", nameserver4),
			txtRecord: txtRecord4,
		}

		fetcher.ip6[provider] = providerObj{
			resolver:  newResolver(dialer, "udp6", nameserver6),
			txtRecord: txtRecord6,
		}
	}

	return fetcher, nil
}
