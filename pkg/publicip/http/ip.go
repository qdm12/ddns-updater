package http

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

func (f *Fetcher) IP(ctx context.Context) (publicIP net.IP, err error) {
	return f.ip(ctx, f.ip4or6, ipversion.IP4or6)
}

func (f *Fetcher) IP4(ctx context.Context) (publicIP net.IP, err error) {
	return f.ip(ctx, f.ip4, ipversion.IP4)
}

func (f *Fetcher) IP6(ctx context.Context) (publicIP net.IP, err error) {
	return f.ip(ctx, f.ip6, ipversion.IP6)
}

func (f *Fetcher) ip(ctx context.Context, ring *urlsRing, version ipversion.IPVersion) (
	publicIP net.IP, err error) {
	ring.mutex.Lock()

	var index int
	banned := 0
	for {
		ring.index = (ring.index + 1) % len(ring.urls)
		index = ring.index
		_, indexIsBanned := ring.banned[index]
		if !indexIsBanned {
			break
		}
		banned++
		if banned == len(ring.urls) {
			banString := ring.banString()
			ring.mutex.Unlock()
			return nil, fmt.Errorf("%w: %s", ErrBanned, banString)
		}
	}

	ring.mutex.Unlock()

	url := ring.urls[index]

	ctx, cancel := context.WithTimeout(ctx, f.timeout)
	defer cancel()

	publicIP, err = fetch(ctx, f.client, url, version)
	if err != nil {
		if errors.Is(err, ErrBanned) {
			ring.mutex.Lock()
			ring.banned[index] = strings.ReplaceAll(err.Error(), ErrBanned.Error()+": ", "")
			ring.mutex.Unlock()
		}
		return nil, err
	}
	return publicIP, nil
}
