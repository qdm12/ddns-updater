package params

import (
	"errors"
	"fmt"
	"net/netip"
	"os"
	"strconv"
	"strings"
)

func getRetroIPv6Suffix() (suffix netip.Prefix, err error) {
	prefixBitsString := os.Getenv("IPV6_PREFIX")
	if prefixBitsString == "" {
		return netip.Prefix{}, nil
	}

	return makeIPv6Suffix(prefixBitsString)
}

var (
	ErrIPv6PrefixFormat = errors.New("IPv6 prefix format is incorrect")
)

func makeIPv6Suffix(prefixBitsString string) (suffix netip.Prefix, err error) {
	prefixBitsString = strings.TrimPrefix(prefixBitsString, "/")

	const base, bitSize = 10, 8
	prefixBits, err := strconv.ParseUint(prefixBitsString, base, bitSize)
	if err != nil {
		return netip.Prefix{}, fmt.Errorf("%w: cannot parse %q as uint8",
			ErrIPv6PrefixFormat, prefixBitsString)
	}

	const ipv6Bits = 128
	if prefixBits > ipv6Bits {
		return netip.Prefix{}, fmt.Errorf("%w: %d bits cannot be greater than %d",
			ErrIPv6PrefixFormat, prefixBits, ipv6Bits)
	}

	suffixBits := ipv6Bits - int(prefixBits)
	suffix = netip.PrefixFrom(netip.AddrFrom16([16]byte{}), suffixBits)

	return suffix, nil
}
