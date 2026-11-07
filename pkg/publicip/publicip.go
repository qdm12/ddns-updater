package publicip

import (
	"context"
	"errors"
	"net/netip"

	"github.com/qdm12/ddns-updater/pkg/publicip/dns"
	"github.com/qdm12/ddns-updater/pkg/publicip/http"
)

type ipFetcher interface {
	IP(ctx context.Context) (ip netip.Addr, err error)
	IP4(ctx context.Context) (ipv4 netip.Addr, err error)
	IP6(ctx context.Context) (ipv6 netip.Addr, err error)
}

type Fetcher struct {
	settings settings
	fetchers []ipFetcher
	// names has the same length and order as fetchers and holds
	// a human readable name for each fetcher ("DNS" or "HTTP").
	names []string
	// Cycling effect if both are enabled
	counter *uint32 // 32 bit for 32 bit systems
}

var ErrNoFetchTypeSpecified = errors.New("at least one fetcher type must be specified")

func NewFetcher(dnsSettings DNSSettings, httpSettings HTTPSettings) (f *Fetcher, err error) {
	settings := settings{
		dns:  dnsSettings,
		http: httpSettings,
	}

	fetcher := &Fetcher{
		settings: settings,
		counter:  new(uint32),
	}

	if settings.dns.Enabled {
		subFetcher, err := dns.New(settings.dns.Options...)
		if err != nil {
			return nil, err
		}
		fetcher.fetchers = append(fetcher.fetchers, subFetcher)
		fetcher.names = append(fetcher.names, MethodDNS)
	}

	if settings.http.Enabled {
		subFetcher, err := http.New(settings.http.Client, settings.http.Options...)
		if err != nil {
			return nil, err
		}
		fetcher.fetchers = append(fetcher.fetchers, subFetcher)
		fetcher.names = append(fetcher.names, MethodHTTP)
	}

	if len(fetcher.fetchers) == 0 {
		return nil, ErrNoFetchTypeSpecified
	}

	return fetcher, nil
}

func (f *Fetcher) IP(ctx context.Context) (ip netip.Addr, err error) {
	return f.getSubFetcher().IP(ctx)
}

func (f *Fetcher) IP4(ctx context.Context) (ipv4 netip.Addr, err error) {
	return f.getSubFetcher().IP4(ctx)
}

func (f *Fetcher) IP6(ctx context.Context) (ipv6 netip.Addr, err error) {
	return f.getSubFetcher().IP6(ctx)
}
