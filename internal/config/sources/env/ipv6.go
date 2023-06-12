package env

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/qdm12/ddns-updater/internal/config/settings"
)

func (s *Source) readIPv6() (settings settings.IPv6, err error) {
	maskStr := s.env.String("IPV6_PREFIX")
	if maskStr == "" {
		return settings, nil
	}

	settings.MaskBits, err = ipv6DecimalPrefixToBits(maskStr)
	if err != nil {
		return settings, fmt.Errorf("%w: for environment variable IPV6_PREFIX", err)
	}

	return settings, nil
}

var ErrParsePrefix = errors.New("cannot parse IP prefix")

func ipv6DecimalPrefixToBits(prefixDecimal string) (maskBits uint8, err error) {
	prefixDecimal = strings.TrimPrefix(prefixDecimal, "/")

	const base, bits = 10, 8
	maskBitsUint64, err := strconv.ParseUint(prefixDecimal, base, bits)
	if err != nil {
		return 0, fmt.Errorf("parsing prefix decimal as uint8: %w", err)
	}

	const maxBits = 128
	if bits > maxBits {
		return 0, fmt.Errorf("%w: %d bits cannot be larger than %d",
			ErrParsePrefix, bits, maxBits)
	}

	return uint8(maskBitsUint64), nil
}
