package update

import (
	"net/netip"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_ipv6WithSuffix(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		publicIP   netip.Addr
		ipv6Suffix netip.Prefix
		updateIP   netip.Addr
	}{
		"blank_inputs": {},
		"ipv4_publicip": {
			publicIP: netip.MustParseAddr("1.2.3.4"),
			updateIP: netip.MustParseAddr("1.2.3.4"),
		},
		"invalid_suffix": {
			publicIP: netip.MustParseAddr("2001:db8::1"),
			updateIP: netip.MustParseAddr("2001:db8::1"),
		},
		"zero_suffix": {
			publicIP:   netip.MustParseAddr("e4db:af36:82e:1221:1b7f:2f54:6e9e:5e5f"),
			ipv6Suffix: netip.MustParsePrefix("0:0:0:0:0:0:0:0/0"),
			updateIP:   netip.MustParseAddr("e4db:af36:82e:1221:1b7f:2f54:6e9e:5e5f"),
		},
		"suffix_64": {
			publicIP:   netip.MustParseAddr("e4db:af36:82e:1221:1b7f:2f54:6e9e:5e5f"),
			ipv6Suffix: netip.MustParsePrefix("0:0:0:0:72ad:8fbb:a54e:bedd/64"),
			updateIP:   netip.MustParseAddr("e4db:af36:82e:1221:" + "72ad:8fbb:a54e:bedd"),
		},
		"suffix_56": {
			publicIP:   netip.MustParseAddr("e4db:af36:82e:1221:1b7f:2f54:6e9e:5e5f"),
			ipv6Suffix: netip.MustParsePrefix("bbff:8199:4e2f:b4ba:72ad:8fbb:a54e:bedd/56"),
			updateIP:   netip.MustParseAddr("e4db:af36:82e:1221:1b" + "ad:8fbb:a54e:bedd"),
		},
		"suffix_48": {
			publicIP:   netip.MustParseAddr("e4db:af36:82e:1221:1b7f:2f54:6e9e:5e5f"),
			ipv6Suffix: netip.MustParsePrefix("bbff:8199:4e2f:b4ba:72ad:8fbb:a54e:bedd/48"),
			updateIP:   netip.MustParseAddr("e4db:af36:82e:1221:1b7f:" + "8fbb:a54e:bedd"),
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			updateIP := ipv6WithSuffix(testCase.publicIP, testCase.ipv6Suffix)
			assert.Equal(t, testCase.updateIP.String(), updateIP.String())
		})
	}
}
