package dns

import (
	"errors"
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
			provider: Google,
		},
		"invalid provider": {
			provider: Provider("invalid"),
			err:      errors.New("unknown provider: invalid"),
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
		"google": {
			provider: Google,
			data: providerData{
				nameserver: "ns1.google.com:53",
				fqdn:       "o-o.myaddr.l.google.com.",
				class:      dns.ClassINET,
				qType:      dns.Type(dns.TypeTXT),
			},
		},
		"cloudflare": {
			provider: Cloudflare,
			data: providerData{
				nameserver: "one.one.one.one:53",
				fqdn:       "whoami.cloudflare.",
				class:      dns.ClassCHAOS,
				qType:      dns.Type(dns.TypeTXT),
			},
		},
		"opendns": {
			provider: OpenDNS,
			data: providerData{
				nameserver: "resolver1.opendns.com:53",
				fqdn:       "myip.opendns.com.",
				class:      dns.ClassINET,
				qType:      dns.Type(dns.TypeANY),
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
