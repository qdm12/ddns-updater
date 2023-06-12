package settings

import (
	"errors"
	"fmt"
	"net/netip"

	"github.com/qdm12/gosettings"
)

type IPv6 struct {
	// MaskBits is the IPv6 mask in bits, for example 128 for /128
	MaskBits uint8
}

func (i *IPv6) setDefaults() {
	i.MaskBits = gosettings.DefaultNumber(i.MaskBits,
		uint8(netip.IPv6Unspecified().BitLen()))
}

func (i IPv6) mergeWith(other IPv6) (merged IPv6) {
	merged.MaskBits = gosettings.MergeWithNumber(i.MaskBits, other.MaskBits)
	return merged
}

var (
	ErrMaskBitsTooHigh = errors.New("mask bits is too high")
)

func (i IPv6) Validate() (err error) {
	const maxMaskBits = 128
	if i.MaskBits > maxMaskBits {
		return fmt.Errorf("%w: %d must be equal or below to %d",
			ErrMaskBitsTooHigh, i.MaskBits, maxMaskBits)
	}

	return nil
}
