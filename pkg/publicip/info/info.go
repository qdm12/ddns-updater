package info

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"net/http"

	"github.com/qdm12/golibs/crypto/random/sources/maphash"
)

type Info struct {
	client    *http.Client
	rand      *rand.Rand
	providers []provider
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
	}, nil
}

func (i *Info) pickProvider() provider { //nolint:ireturn
	index := 0
	if L := len(i.providers); L > 1 {
		index = i.rand.Intn(L)
	}
	return i.providers[index]
}

// Get finds IP information for the given IP address using one of
// the ip data provider picked at random.
func (i *Info) Get(ctx context.Context, ip net.IP) (result Result, err error) {
	provider := i.pickProvider()
	return provider.get(ctx, ip)
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
