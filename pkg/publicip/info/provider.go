package info

import (
	"context"
	"fmt"
	"net/http"
	"net/netip"
	"slices"
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

func ValidateProvider(provider Provider) error {
	if slices.Contains(ListProviders(), provider) {
		return nil
	}
	return fmt.Errorf("unknown public IP information provider: %s", provider)
}

type provider interface {
	get(ctx context.Context, ip netip.Addr) (result Result, err error)
}

func newProvider(providerName Provider, client *http.Client) provider { //nolint:ireturn
	switch providerName {
	case Ipinfo:
		return newIpinfo(client)
	case IP2Location:
		return newIP2Location(client)
	default:
		panic(fmt.Sprintf("provider %s not implemented", providerName))
	}
}
