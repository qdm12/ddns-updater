package info

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/netip"
)

type Provider string

const (
	Ipinfo Provider = "ipinfo"
)

func ListProviders() []Provider {
	return []Provider{
		Ipinfo,
	}
}

var ErrUnknownProvider = errors.New("unknown provider")

func ValidateProvider(provider Provider) error {
	for _, possible := range ListProviders() {
		if provider == possible {
			return nil
		}
	}
	return fmt.Errorf("%w: %s", ErrUnknownProvider, provider)
}

type provider interface {
	get(ctx context.Context, ip netip.Addr) (result Result, err error)
}

func newProvider(providerName Provider, client *http.Client) provider { //nolint:ireturn
	switch providerName {
	case Ipinfo:
		return newIpinfo(client)
	default:
		panic(fmt.Sprintf("provider %s not implemented", providerName))
	}
}
