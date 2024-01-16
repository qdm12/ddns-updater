package config

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/qdm12/gosettings"
	"github.com/qdm12/gosettings/reader"
	"github.com/qdm12/gotree"
)

type IPv6 struct {
	// Prefix is the IPv6 mask, for example /128
	Prefix string
}

func (i *IPv6) setDefaults() {
	i.Prefix = gosettings.DefaultComparable(i.Prefix, "/128")
}

func (i IPv6) Validate() (err error) {
	err = validateIPv6Prefix(i.Prefix)
	if err != nil {
		return err
	}

	return nil
}

var (
	ErrIPv6PrefixFormat = errors.New("IPv6 prefix format is incorrect")
)

func validateIPv6Prefix(prefix string) (err error) {
	prefix = strings.TrimPrefix(prefix, "/")

	const base, bits = 10, 8
	_, err = strconv.ParseUint(prefix, base, bits)
	if err != nil {
		return fmt.Errorf("%w: cannot parse %q as uint8", ErrIPv6PrefixFormat, prefix)
	}

	const maxBits = 128
	if bits > maxBits {
		return fmt.Errorf("%w: %d bits cannot be larger than %d",
			ErrIPv6PrefixFormat, bits, maxBits)
	}

	return nil
}

func (i IPv6) String() string {
	return i.toLinesNode().String()
}

func (i IPv6) toLinesNode() *gotree.Node {
	node := gotree.New("IPv6")
	node.Appendf("Prefix: %s", i.Prefix)
	return node
}

func (i *IPv6) read(reader *reader.Reader) {
	i.Prefix = reader.String("IPV6_PREFIX")
}
