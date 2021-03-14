package dns

import (
	"errors"
	"fmt"
)

type Provider string

const (
	Cloudflare Provider = "cloudflare"
	Google     Provider = "google"
)

func ListProviders() []Provider {
	return []Provider{
		Cloudflare,
		Google,
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

func (provider Provider) data() (nameserver, txtRecord string) {
	switch provider {
	case Google:
		return "ns1.google.com:53", "o-o.myaddr.l.google.com"
	case Cloudflare:
		return "one.one.one.one:53", "whoami.cloudflare"
	}
	panic(`provider unknown: "` + string(provider) + `"`)
}
