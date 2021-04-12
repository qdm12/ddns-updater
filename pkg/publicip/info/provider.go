package info

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
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
	return fmt.Errorf("%w: %q", ErrUnknownProvider, provider)
}

type provider interface {
	get(ctx context.Context, ip net.IP) (result Result, err error)
}

func newProvider(p Provider, client *http.Client) provider {
	switch p {
	case Ipinfo:
		return newIpinfo(client)
	default:
		return nil
	}
}
