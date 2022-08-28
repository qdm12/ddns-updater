package http

import (
	"net/http"
	"time"

	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

type Fetcher struct {
	client  *http.Client
	timeout time.Duration
	ip4or6  urlsRing // URLs to get ipv4 or ipv6
	ip4     urlsRing // URLs to get ipv4 only
	ip6     urlsRing // URLs to get ipv6 only
}

type urlsRing struct {
	counter *uint32
	urls    []string
}

func New(client *http.Client, options ...Option) (f *Fetcher, err error) {
	settings := newDefaultSettings()
	for _, option := range options {
		if err := option(&settings); err != nil {
			return nil, err
		}
	}

	fetcher := &Fetcher{
		client:  client,
		timeout: settings.timeout,
	}

	fetcher.ip4or6.counter = new(uint32)
	for _, provider := range settings.providersIP {
		url, _ := provider.url(ipversion.IP4or6)
		fetcher.ip4or6.urls = append(fetcher.ip4or6.urls, url)
	}

	fetcher.ip4.counter = new(uint32)
	for _, provider := range settings.providersIP4 {
		url, _ := provider.url(ipversion.IP4)
		fetcher.ip4.urls = append(fetcher.ip4.urls, url)
	}

	fetcher.ip6.counter = new(uint32)
	for _, provider := range settings.providersIP6 {
		url, _ := provider.url(ipversion.IP6)
		fetcher.ip6.urls = append(fetcher.ip6.urls, url)
	}

	return fetcher, nil
}
