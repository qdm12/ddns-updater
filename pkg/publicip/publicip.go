package publicip

import (
	"context"
	"errors"
	"math/rand"
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
	// Cycling effect if both are enabled, without a mutex
	randSource rand.Source
	fetchTypes []fetchType
}

var ErrNoFetchTypeSpecified = errors.New("at least one fetcher type must be specified")

func NewFetcher(options ...Option) (f Fetcher, err error) {
	settings := defaultSettings()
	for _, option := range options {
		if err := option(&settings); err != nil {
			return nil, err
		}
	}

	fetcher := &fetcher{
		settings:   settings,
		randSource: rand.NewSource(0),
	}

	if settings.dns.enabled {
		fetcher.dns, err = dns.New(settings.dns.options...)
		if err != nil {
			return nil, err
		}
		fetcher.fetchTypes = append(fetcher.fetchTypes, dnsFetch)
	}

	if settings.http.enabled {
		fetcher.http, err = http.New(settings.http.client, settings.http.options...)
		if err != nil {
			return nil, err
		}
		fetcher.fetchTypes = append(fetcher.fetchTypes, httpFetch)
	}

	if len(fetcher.fetchTypes) == 0 {
		return nil, ErrNoFetchTypeSpecified
	}

	return fetcher, nil
}

func (f *fetcher) IP(ctx context.Context) (ip net.IP, err error) {
	subFetcher, err := f.getSubFetcher()
	if err != nil {
		return nil, err
	}
	return subFetcher.IP(ctx)
}

func (f *fetcher) IP4(ctx context.Context) (ipv4 net.IP, err error) {
	subFetcher, err := f.getSubFetcher()
	if err != nil {
		return nil, err
	}
	return subFetcher.IP4(ctx)
}

func (f *fetcher) IP6(ctx context.Context) (ipv6 net.IP, err error) {
	subFetcher, err := f.getSubFetcher()
	if err != nil {
		return nil, err
	}
	return subFetcher.IP6(ctx)
}
