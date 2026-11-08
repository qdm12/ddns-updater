package resolver

import (
	"context"
	"fmt"
	"net"
	"net/netip"

	"github.com/qdm12/ddns-updater/internal/doh"
)

// LookupIPer resolves host names to IP addresses.
type LookupIPer interface {
	LookupNetIP(ctx context.Context, network, host string) (ips []netip.Addr, err error)
}

// IPResolver adapts a LookupIPer to resolve host names to net.IP addresses.
type IPResolver struct {
	lookupIPer LookupIPer
}

// NewIPResolver creates an IPResolver from the lookupIPer given.
func NewIPResolver(lookupIPer LookupIPer) IPResolver {
	return IPResolver{lookupIPer: lookupIPer}
}

// LookupIP resolves the host name given and returns its IP addresses.
func (r IPResolver) LookupIP(ctx context.Context, network, host string) (
	ips []net.IP, err error,
) {
	addresses, err := r.lookupIPer.LookupNetIP(ctx, network, host)
	if err != nil {
		return nil, err
	}

	ips = make([]net.IP, 0, len(addresses))
	for _, address := range addresses {
		ips = append(ips, net.IP(address.AsSlice()))
	}

	return ips, nil
}

// New creates a resolver to look up host names.
// If a DNS over HTTPS URL is set, it is used to resolve the host names,
// since it bypasses any local DNS server filtering or blocking.
// Otherwise, the Go default resolver is used, or the resolver at the
// address given if it is set.
func New(settings Settings) (resolver LookupIPer, err error) {
	settings.setDefaults()
	err = settings.validate()
	if err != nil {
		return nil, fmt.Errorf("validating settings: %w", err)
	}

	if *settings.DOHURL != "" {
		dohClient, err := doh.New(*settings.DOHURL, settings.Timeout)
		if err != nil {
			return nil, fmt.Errorf("creating DNS over HTTPS client: %w", err)
		}
		return dohClient, nil
	}

	if *settings.Address == "" {
		return net.DefaultResolver, nil
	}

	dialer := net.Dialer{Timeout: settings.Timeout}
	return &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, _, _ string) (net.Conn, error) {
			const protocol = "udp"
			return dialer.DialContext(ctx, protocol, *settings.Address)
		},
	}, nil
}
