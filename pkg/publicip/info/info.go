package info

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"sync"

	"github.com/qdm12/golibs/crypto/random/sources/maphash"
)

type Info struct {
	client    *http.Client
	rand      *rand.Rand
	providers []provider
	banMutex  sync.RWMutex
	banned    map[provider]struct{}
}

func New(client *http.Client, options ...Option) (info *Info, err error) {
	var settings settings
	for _, option := range options {
		err = option(&settings)
		if err != nil {
			return nil, fmt.Errorf("applying option: %w", err)
		}
	}
	settings.setDefaults()

	providers := make([]provider, len(settings.providers))
	for i := range settings.providers {
		providers[i] = newProvider(settings.providers[i], client)
	}

	// fast & thread safe random generator
	generator := rand.New(maphash.New()) //nolint:gosec

	return &Info{
		client:    client,
		rand:      generator,
		providers: providers,
		banned:    make(map[provider]struct{}),
	}, nil
}

// Get finds IP information for the given IP address using one of
// the ip data provider picked at random. A `nil` IP address can be
// given to signal to fetch information on the current public IP address.
func (i *Info) Get(ctx context.Context, ip net.IP) (result Result, err error) {
	if len(i.providers) == 1 {
		return i.providers[0].get(ctx, ip)
	}

	index := i.rand.Intn(len(i.providers))
	failed := 0
	for failed < len(i.providers) {
		provider := i.providers[index]
		if i.isBanned(provider) { // try next provider
			index++
			failed++
			continue
		}

		result, err = provider.get(ctx, ip)
		if err != nil {
			// try next provider
			index++
			failed++
			if errors.Is(err, ErrTooManyRequests) {
				i.ban(provider)
			}
			continue
		}
	}

	return result, err
}

// GetMultiple finds IP information for the given IP addresses, each using
// one of the ip data provider picked at random. It returns a slice of results
// matching the order of the IP addresses given as argument.
func (i *Info) GetMultiple(ctx context.Context, ips []net.IP) (results []Result, err error) {
	type resultWithError struct {
		index  int
		result Result
		err    error
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	channel := make(chan resultWithError)

	for index, ip := range ips {
		go func(ctx context.Context, index int, ip net.IP) {
			result := resultWithError{
				index: index,
			}
			result.result, result.err = i.Get(ctx, ip)
			channel <- result
		}(ctx, index, ip)
	}

	results = make([]Result, len(ips))
	for i := range results {
		result := <-channel
		switch {
		// only collect the first error
		case err != nil:
		case result.err != nil:
			err = result.err
			cancel() // stop other operations
		default:
			results[i] = result.result
		}
	}

	if err != nil {
		return nil, err
	}

	return results, nil
}

func (i *Info) isBanned(p provider) (banned bool) {
	i.banMutex.RLock()
	_, banned = i.banned[p]
	i.banMutex.RUnlock()
	return banned
}

func (i *Info) ban(p provider) {
	i.banMutex.Lock()
	i.banned[p] = struct{}{}
	i.banMutex.Unlock()
}
