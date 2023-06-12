package utils

import (
	"net/netip"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_FindIPv4Addresses(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		text      string
		addresses []netip.Addr
	}{
		"empty_string": {},
		"no_address": {
			text: "dsadsa 232.323 s",
		},
		"single_address_exact": {
			text:      "192.168.1.5",
			addresses: []netip.Addr{netip.AddrFrom4([4]byte{192, 168, 1, 5})},
		},
		"multiple_in_text": {
			text:      "sd 192.168.1.5 1.5 1.3.5.4",
			addresses: []netip.Addr{netip.AddrFrom4([4]byte{192, 168, 1, 5}), netip.AddrFrom4([4]byte{1, 3, 5, 4})},
		},
		"longer_than_normal_ip": {
			text:      "0.0.0.0.0",
			addresses: []netip.Addr{netip.AddrFrom4([4]byte{0, 0, 0, 0})},
		},
	}
	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			addresses := FindIPv4Addresses(testCase.text)
			assert.Equal(t, testCase.addresses, addresses)
		})
	}
}

func Test_FindIPv6Addresses(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		text      string
		addresses []netip.Addr
	}{
		"empty_string": {},
		"no_address": {
			text: "dsadsa 232.323 s",
		},
		"ignore_ipv4_address": {
			text: "192.168.1.5",
		},
		"single_address_exact": {
			text:      "::1",
			addresses: []netip.Addr{netip.MustParseAddr("::1")},
		},
		"multiple_in_text": {
			text: "2001:0db8:85a3:0000:0000:8a2e:0370:7334 sdas ::1",
			addresses: []netip.Addr{
				netip.MustParseAddr("2001:0db8:85a3:0000:0000:8a2e:0370:7334"),
				netip.MustParseAddr("::1"),
			},
		},
	}
	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			addresses := FindIPv6Addresses(testCase.text)
			assert.Equal(t, testCase.addresses, addresses)
		})
	}
}
