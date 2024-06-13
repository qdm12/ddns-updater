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
	Ipinfo      Provider = "ipinfo"
	IP2Location Provider = "ip2location"
)

func ListProviders() []Provider {
	return []Provider{
		Ipinfo,
		IP2Location,
	}
}

var ErrUnknownProvider = errors.New("unknown public IP information provider")

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

//nolint:ireturn
func newProvider(providerName Provider, client *http.Client) provider {
	switch providerName {
	case Ipinfo:
		return newIpinfo(client)
	case IP2Location:
		return newIP2Location(client)
	default:
		panic(fmt.Sprintf("provider %s not implemented", providerName))
	}
}
