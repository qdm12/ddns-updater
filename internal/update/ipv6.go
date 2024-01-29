package update

import (
	"fmt"
	"net/netip"
)

func ipv6WithSuffix(publicIP netip.Addr, ipv6Suffix netip.Prefix) (
	updateIP netip.Addr) {
	if !publicIP.IsValid() || !publicIP.Is6() || !ipv6Suffix.IsValid() {
		return publicIP
	}

	const ipv6Bits = 128
	const bitsInByte = 8
	prefixLength := (ipv6Bits - ipv6Suffix.Bits()) / bitsInByte
	ispPrefix := publicIP.AsSlice()[:prefixLength]
	localSuffix := ipv6Suffix.Addr().AsSlice()[prefixLength:]
	ipv6Bytes := ispPrefix // ispPrefix has already 16 bytes of capacity
	ipv6Bytes = append(ipv6Bytes, localSuffix...)
	updateIP, ok := netip.AddrFromSlice(ipv6Bytes)
	if !ok {
		panic(fmt.Sprintf("failed to create IPv6 address from merged bytes %v", ipv6Bytes))
	}

	return updateIP
}
