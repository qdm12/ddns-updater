package privateip

import (
	"context"
	"fmt"
	"net"
	"net/netip"
)

// Settings structure for configuring the Private Fetcher.
type Settings struct {
	Enabled bool
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
func New(settings Settings, retriever InterfaceRetriever) (*Fetcher, error) {
	if !settings.Enabled {
		return nil, fmt.Errorf("private IP fetcher is disabled")
	}

	if retriever == nil {
		return nil, fmt.Errorf("interface retriever cannot be nil")
	}

	return &Fetcher{
		settings:           settings,
		interfaceRetriever: retriever,
	}, nil
}

// IP fetches the private IP address of the machine.
func (f *Fetcher) IP(ctx context.Context) (netip.Addr, error) {
	interfaces, err := f.interfaceRetriever.Interfaces()
	if err != nil {
		return netip.Addr{}, err
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := f.interfaceRetriever.Addrs(iface)
		if err != nil {
			return netip.Addr{}, err
		}

		for _, addr := range addrs {
			switch v := addr.(type) {
			case *net.IPNet:
				if isPrivateIP(v.IP) {
					privateAddr, err := netip.ParseAddr(v.IP.String())
					if err != nil {
						return netip.Addr{}, err
					}
					return privateAddr, nil
				}
			}
		}
	}

	return netip.Addr{}, fmt.Errorf("no private IP address found")
}

// IP4 fetches the IPv4 address of the machine.
func (f *Fetcher) IP4(ctx context.Context) (netip.Addr, error) {
	ip, err := f.IP(ctx)
	if err != nil {
		return netip.Addr{}, err
	}
	if ip.Is4() {
		return ip, nil
	}
	return netip.Addr{}, fmt.Errorf("no IPv4 address found")
}

// IP6 fetches the IPv6 address of the machine.
func (f *Fetcher) IP6(ctx context.Context) (netip.Addr, error) {
	ip, err := f.IP(ctx)
	if err != nil {
		return netip.Addr{}, err
	}
	if ip.Is6() {
		return ip, nil
	}
	return netip.Addr{}, fmt.Errorf("no IPv6 address found")
}

// isPrivateIP checks if an IP is from a private range.
func isPrivateIP(ip net.IP) bool {
	privateBlocks := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
	}

	for _, block := range privateBlocks {
		_, cidr, _ := net.ParseCIDR(block)
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}
