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
	ring    ring
	client  Client
	client4 Client
	client6 Client
}

type ring struct {
	mutex     sync.RWMutex
	index     int // index in the providers slice
	providers []Provider
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

	return &fetcher{
		ring: ring{
			providers: settings.providers,
		},
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
	}, nil
}
