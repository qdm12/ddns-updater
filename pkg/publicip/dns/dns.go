package dns

import (
	"context"
	"net"

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
	// counter is used to get an index in the providers slice
	counter   *uint32 // uint32 for 32 bit systems atomic operations
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
			counter:   new(uint32),
			providers: settings.providers,
		},
		client: &dns.Client{
			Net:     "udp",
			Dialer:  dialer,
			Timeout: settings.timeout,
		},
		client4: &dns.Client{
			Net:     "udp4",
			Dialer:  dialer,
			Timeout: settings.timeout,
		},
		client6: &dns.Client{
			Net:     "udp6",
			Dialer:  dialer,
			Timeout: settings.timeout,
		},
	}, nil
}
