package privateip

import (
	"context"
	"net/netip"
)

// Provider retrieves the private IP address
type Provider struct {
	options *Options
}

// NewProvider creates a new private IP provider with the given options
func NewProvider(options *Options) *Provider {
	return &Provider{
		options: options,
	}
}

// Fetch fetches the private IP address of the machine
func (p *Provider) Fetch(ctx context.Context) (netip.Addr, error) {
	return fetch()
}
