package dns

import (
	"errors"
	"net/netip"
	"testing"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ValidateProvider(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		provider Provider
		err      error
	}{
		"valid provider": {
			provider: Cloudflare,
		},
		"invalid provider": {
			provider: Provider("invalid"),
			err:      errors.New("unknown public IP echo DNS provider: invalid"),
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := ValidateProvider(testCase.provider)
			if testCase.err != nil {
				require.Error(t, err)
				assert.Equal(t, testCase.err.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_data(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		provider     Provider
		data         providerData
		panicMessage string
	}{
		"cloudflare": {
			provider: Cloudflare,
			data: providerData{
				Address: "1dot1dot1dot1.cloudflare-dns.com",
				IPv4:    netip.AddrFrom4([4]byte{1, 1, 1, 1}),
				IPv6:    netip.AddrFrom16([16]byte{0x26, 0x6, 0x47, 0x0, 0x47, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x11, 0x11}), //nolint:lll
				TLSName: "cloudflare-dns.com",
				fqdn:    "whoami.cloudflare.",
				class:   dns.ClassCHAOS,
				qType:   dns.Type(dns.TypeTXT),
			},
		},
		"opendns": {
			provider: OpenDNS,
			data: providerData{
				Address: "dns.opendns.com",
				IPv4:    netip.AddrFrom4([4]byte{208, 67, 222, 222}),
				IPv6:    netip.AddrFrom16([16]byte{0x26, 0x20, 0x1, 0x19, 0x0, 0x35, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x35}), //nolint:lll
				TLSName: "dns.opendns.com",
				fqdn:    "myip.opendns.com.",
				class:   dns.ClassINET,
				qType:   dns.Type(dns.TypeANY),
			},
		},
		"invalid provider": {
			provider:     Provider("invalid"),
			panicMessage: `provider unknown: "invalid"`,
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if testCase.panicMessage != "" {
				assert.PanicsWithValue(t, testCase.panicMessage, func() {
					testCase.provider.data()
				})
				return
			}
			data := testCase.provider.data()
			assert.Equal(t, testCase.data, data)
		})
	}
}
