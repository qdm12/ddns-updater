package privateip

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"slices"
	"strings"
)

var (
	ErrDisabled             = errors.New("private IP fetcher is disabled")
	ErrInterfaceNotSet      = errors.New("private IP interface is not set")
	ErrRetrieverNil         = errors.New("interface retriever cannot be nil")
	ErrInterfaceNotFound    = errors.New("interface not found")
	ErrInterfaceDown        = errors.New("interface is down")
	ErrInterfaceLoopback    = errors.New("interface is a loopback interface")
	ErrNoPrivateIPv4        = errors.New("no private IPv4 address found on interface")
	ErrAmbiguousPrivateIPv4 = errors.New("multiple private IPv4 addresses found on interface")
	ErrIPv6NotSupported     = errors.New("IPv6 is not supported by the private IP fetcher")
)

// Settings structure for configuring the Private Fetcher.
type Settings struct {
	Enabled bool
	// Interface is the exact (case sensitive) name of the only
	// network interface the address is read from.
	Interface string
}

// InterfaceRetriever defines methods to retrieve network interfaces and their addresses.
type InterfaceRetriever interface {
	Interfaces() ([]net.Interface, error)
	Addrs(iface net.Interface) ([]net.Addr, error)
}

// RealInterfaceRetriever is the production implementation of InterfaceRetriever.
type RealInterfaceRetriever struct{}

// Interfaces retrieves the list of network interfaces.
func (rir RealInterfaceRetriever) Interfaces() ([]net.Interface, error) {
	return net.Interfaces()
}

// Addrs retrieves the addresses for a given network interface.
func (rir RealInterfaceRetriever) Addrs(iface net.Interface) ([]net.Addr, error) {
	return iface.Addrs()
}

// Fetcher struct to represent the private IP fetcher.
type Fetcher struct {
	settings           Settings
	interfaceRetriever InterfaceRetriever
}

// New creates a new instance of the Private Fetcher with dependency injection.
// The interface does not have to exist yet: it is looked up on every fetch.
func New(settings Settings, retriever InterfaceRetriever) (*Fetcher, error) {
	switch {
	case !settings.Enabled:
		return nil, ErrDisabled
	case strings.TrimSpace(settings.Interface) == "":
		return nil, ErrInterfaceNotSet
	case retriever == nil:
		return nil, ErrRetrieverNil
	}

	return &Fetcher{
		settings:           settings,
		interfaceRetriever: retriever,
	}, nil
}

// IP fetches the single private IPv4 address currently set on the configured
// interface. Nothing is cached and no other interface is ever considered.
func (f *Fetcher) IP(_ context.Context) (netip.Addr, error) {
	name := f.settings.Interface

	interfaces, err := f.interfaceRetriever.Interfaces()
	if err != nil {
		return netip.Addr{}, fmt.Errorf("listing network interfaces for %q: %w", name, err)
	}

	index := slices.IndexFunc(interfaces, func(iface net.Interface) bool { return iface.Name == name })
	if index < 0 {
		return netip.Addr{}, fmt.Errorf("%w: %q", ErrInterfaceNotFound, name)
	}
	iface := interfaces[index]
	if iface.Flags&net.FlagUp == 0 {
		return netip.Addr{}, fmt.Errorf("%w: %q", ErrInterfaceDown, name)
	}
	if iface.Flags&net.FlagLoopback != 0 {
		return netip.Addr{}, fmt.Errorf("%w: %q", ErrInterfaceLoopback, name)
	}

	addrs, err := f.interfaceRetriever.Addrs(iface)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("getting addresses of interface %q: %w", name, err)
	}

	var found netip.Addr
	for _, addr := range addrs {
		candidate, ok := privateIPv4(addr)
		switch {
		case !ok || candidate == found:
		case found.IsValid():
			return netip.Addr{}, fmt.Errorf("%w: %q has %s and %s", ErrAmbiguousPrivateIPv4, name, found, candidate)
		default:
			found = candidate
		}
	}
	if !found.IsValid() {
		return netip.Addr{}, fmt.Errorf("%w: %q", ErrNoPrivateIPv4, name)
	}
	return found, nil
}

// IP4 fetches the private IPv4 address of the configured interface.
func (f *Fetcher) IP4(ctx context.Context) (netip.Addr, error) {
	return f.IP(ctx)
}

// IP6 is not supported: only private IPv4 addresses are fetched.
func (f *Fetcher) IP6(_ context.Context) (netip.Addr, error) {
	return netip.Addr{}, fmt.Errorf("%w: %q", ErrIPv6NotSupported, f.settings.Interface)
}

// privateIPv4 returns the RFC1918 IPv4 address of addr, skipping
// nil, unsupported and malformed entries.
func privateIPv4(addr net.Addr) (netip.Addr, bool) {
	var ip net.IP
	switch v := addr.(type) {
	case *net.IPNet:
		if v != nil {
			ip = v.IP
		}
	case *net.IPAddr:
		if v != nil {
			ip = v.IP
		}
	}
	parsed, ok := netip.AddrFromSlice(ip)
	parsed = parsed.Unmap()
	return parsed, ok && parsed.Is4() && parsed.IsPrivate()
}
