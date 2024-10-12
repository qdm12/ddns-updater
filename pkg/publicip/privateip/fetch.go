package privateip

import (
	"net"
	"net/netip"
)

// InterfaceAddrs is a variable that can be overridden in tests.
var InterfaceAddrs = net.InterfaceAddrs

// fetch retrieves the private IP address of the machine.
func fetch() (netip.Addr, error) {
	addrs, err := InterfaceAddrs()
	if err != nil {
		return netip.Addr{}, err
	}

	for _, addr := range addrs {
		// Check if the address is an IP address and is private
		if ipNet, ok := addr.(*net.IPNet); ok && ipNet.IP.IsPrivate() {
			ip, valid := netip.AddrFromSlice(ipNet.IP)
			if valid {
				return ip, nil
			}
		}
	}
	return netip.Addr{}, nil
}
