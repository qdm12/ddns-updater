package dns

import (
	"context"
	"net"
	"sync"

	"github.com/miekg/dns"
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
	client    Client
	client4   Client
	client6   Client
	data      map[Provider]providerData
}

type providerData struct {
	nameserver string
	fqdn       string
	class      dns.Class
}

func New(options ...Option) (f Fetcher, err error) {
	settings := newDefaultSettings()
	for _, option := range options {
		if err := option(&settings); err != nil {
			return nil, err
		}
	}

	dialer := &net.Dialer{
		Timeout: settings.timeout,
	}

	fetcher := &fetcher{
		providers: settings.providers,
		client: &dns.Client{
			Net:    "udp",
			Dialer: dialer,
		},
		client4: &dns.Client{
			Net:    "udp4",
			Dialer: dialer,
		},
		client6: &dns.Client{
			Net:    "udp6",
			Dialer: dialer,
		},
		data: make(map[Provider]providerData, len(settings.providers)),
	}

	for _, provider := range settings.providers {
		fetcher.data[provider] = provider.data()
	}

	return fetcher, nil
}
