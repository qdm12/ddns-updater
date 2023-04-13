package http

import (
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

type Fetcher struct {
	client  *http.Client
	timeout time.Duration
	ip4or6  *urlsRing // URLs to get ipv4 or ipv6
	ip4     *urlsRing // URLs to get ipv4 only
	ip6     *urlsRing // URLs to get ipv6 only
}

type urlsRing struct {
	index  int
	urls   []string
	banned map[int]string // urls indices <-> ban error string
	mutex  sync.Mutex
}

func New(client *http.Client, options ...Option) (f *Fetcher, err error) {
	settings := newDefaultSettings()
	for _, option := range options {
		err = option(&settings)
		if err != nil {
			return nil, err
		}
	}

	return &Fetcher{
		client:  client,
		timeout: settings.timeout,
		ip4or6:  newRing(settings.providersIP, ipversion.IP4or6),
		ip4:     newRing(settings.providersIP4, ipversion.IP4),
		ip6:     newRing(settings.providersIP6, ipversion.IP6),
	}, nil
}

func newRing(providers []Provider, ipVersion ipversion.IPVersion) (ring *urlsRing) {
	ring = new(urlsRing)
	ring.banned = make(map[int]string)
	ring.urls = make([]string, len(providers))
	for i, provider := range providers {
		ring.urls[i], _ = provider.url(ipVersion)
	}
	return ring
}

func (u *urlsRing) banString() string {
	parts := make([]string, 0, len(u.banned))
	for i, errString := range u.banned {
		part := errString + " (" + u.urls[i] + ")"
		parts = append(parts, part)
	}
	sort.Strings(parts) // for predicability
	return strings.Join(parts, ", ")
}
