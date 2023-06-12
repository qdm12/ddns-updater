package update

import (
	"fmt"
	"net/netip"
)

func mustMaskIPv6(ipv6 netip.Addr, ipv6MaskBits uint8) (maskedIPv6 netip.Addr) {
	prefix, err := ipv6.Prefix(int(ipv6MaskBits))
	if err != nil {
		panic(fmt.Sprintf("getting masked IPv6 prefix: %s", err))
	}
	maskedIPv6 = prefix.Addr()
	return maskedIPv6
}
