package update

import (
	"net/netip"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_mustMaskIPv6(t *testing.T) {
	t.Parallel()

	const maskBits = 24
	ip := netip.AddrFrom4([4]byte{1, 2, 3, 4})
	maskedIP := mustMaskIPv6(ip, maskBits)

	expected := netip.AddrFrom4([4]byte{1, 2, 3, 0})
	assert.Equal(t, expected, maskedIP)
}
