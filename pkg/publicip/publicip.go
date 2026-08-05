package publicip

import (
	"context"
	"errors"
	"net/netip"

	"github.com/qdm12/ddns-updater/pkg/publicip/dns"
	"github.com/qdm12/ddns-updater/pkg/publicip/http"
	"github.com/qdm12/ddns-updater/pkg/publicip/privateip"
)

type ipFetcher interface {
	IP(ctx context.Context) (ip netip.Addr, err error)
	IP4(ctx context.Context) (ipv4 netip.Addr, err error)
	IP6(ctx context.Context) (ipv6 netip.Addr, err error)
}

type Fetcher struct {
	settings settings
	fetchers []ipFetcher
	// Cycling effect if both are enabled
	counter *uint32 // 32 bit for 32 bit systems
}

var ErrNoFetchTypeSpecified = errors.New("at least one fetcher type must be specified")

func NewFetcher(
	dnsSettings DNSSettings,
	httpSettings HTTPSettings,
	privateIPSettings PrivateIPSettings) (f *Fetcher, err error) {
	settings := settings{
		dns:       dnsSettings,
		http:      httpSettings,
		privateIP: privateIPSettings,
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
	}

	if settings.http.Enabled {
		subFetcher, err := http.New(settings.http.Client, settings.http.Options...)
		if err != nil {
			return nil, err
		}
		fetcher.fetchers = append(fetcher.fetchers, subFetcher)
	}

	if settings.privateIP.Enabled {
		// Instantiate the InterfaceRetriever
		retriever := privateip.RealInterfaceRetriever{}

		// Pass both Settings and the retriever to privateip.New
		subFetcher, err := privateip.New(privateip.Settings{Enabled: true}, retriever)
		if err != nil {
			return nil, err
		}
		fetcher.fetchers = append(fetcher.fetchers, subFetcher)
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
