package settings

import (
	"net"

	"github.com/qdm12/gosettings"
)

type IPv6 struct {
	Mask net.IPMask
}

func (i *IPv6) setDefaults() {
	const ipv6Bits = 8 * net.IPv6len
	if i.Mask == nil {
		i.Mask = net.CIDRMask(ipv6Bits, ipv6Bits)
	}
}

func (i IPv6) mergeWith(other IPv6) (merged IPv6) {
	merged.Mask = gosettings.MergeWithSlice(i.Mask, other.Mask)
	return merged
}

func (i IPv6) Validate() (err error) {
	return nil
}
