package info

import (
	"context"
	"net"
	"net/http"

	"github.com/qdm12/golibs/crypto/random/hashmap"
)

type Info interface {
	Get(ctx context.Context, ip net.IP) (result Result, err error)
}

type info struct {
	client    *http.Client
	rand      hashmap.Rand
	providers []provider
}

func New(client *http.Client, options ...Option) (Info, error) {
	settings := newDefaultSettings()
	for _, option := range options {
		if err := option(&settings); err != nil {
			return nil, err
		}
	}

	providers := make([]provider, len(settings.providers))
	for i := range settings.providers {
		providers[i] = newProvider(settings.providers[i], client)
	}

	return &info{
		client:    client,
		rand:      hashmap.New(), // fast & thread safe
		providers: providers,
	}, nil
}

func (i *info) pickProvider() provider {
	index := 0
	if L := len(i.providers); L > 1 {
		index = i.rand.Intn(len(i.providers))
	}
	return i.providers[index]
}

func (i *info) Get(ctx context.Context, ip net.IP) (result Result, err error) {
	provider := i.pickProvider()
	return provider.get(ctx, ip)
}
