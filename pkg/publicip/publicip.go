package publicip

import (
	"context"
	"errors"
	"net"

	"github.com/qdm12/ddns-updater/pkg/publicip/dns"
	"github.com/qdm12/ddns-updater/pkg/publicip/http"
)

type Fetcher interface {
	IP(ctx context.Context) (ip net.IP, err error)
	IP4(ctx context.Context) (ipv4 net.IP, err error)
	IP6(ctx context.Context) (ipv6 net.IP, err error)
}

type fetcher struct {
	settings settings
	dns      Fetcher
	http     Fetcher
	// Cycling effect if both are enabled
	counter    *uint32 // 32 bit for 32 bit systems
	fetchTypes []FetchType
}

var ErrNoFetchTypeSpecified = errors.New("at least one fetcher type must be specified")

func NewFetcher(dnsSettings DNSSettings, httpSettings HTTPSettings) (f Fetcher, err error) {
	settings := settings{
		dns:  dnsSettings,
		http: httpSettings,
	}

	fetcher := &fetcher{
		settings: settings,
		counter:  new(uint32),
	}

	if settings.dns.Enabled {
		fetcher.dns, err = dns.New(settings.dns.Options...)
		if err != nil {
			return nil, err
		}
		fetcher.fetchTypes = append(fetcher.fetchTypes, DNS)
	}

	if settings.http.Enabled {
		fetcher.http, err = http.New(settings.http.Client, settings.http.Options...)
		if err != nil {
			return nil, err
		}
		fetcher.fetchTypes = append(fetcher.fetchTypes, HTTP)
	}

	if len(fetcher.fetchTypes) == 0 {
		return nil, ErrNoFetchTypeSpecified
	}

	return fetcher, nil
}

func (f *fetcher) IP(ctx context.Context) (ip net.IP, err error) {
	return f.getSubFetcher().IP(ctx)
}

func (f *fetcher) IP4(ctx context.Context) (ipv4 net.IP, err error) {
	return f.getSubFetcher().IP4(ctx)
}

func (f *fetcher) IP6(ctx context.Context) (ipv6 net.IP, err error) {
	return f.getSubFetcher().IP6(ctx)
}
