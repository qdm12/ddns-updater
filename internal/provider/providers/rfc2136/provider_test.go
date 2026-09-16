package rfc2136

import (
	"encoding/json"
	"net/netip"
	"testing"

	"github.com/miekg/dns"
	"github.com/qdm12/ddns-updater/internal/provider/errors"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	exampleZone       = "example.com."
	testServerAddress = "ns1.example.com:53"
)

func Test_New(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		data             string
		expectedProvider *Provider
		expectedErr      error
	}{
		"defaults": {
			data: `{"server":"` + testServerAddress + `"}`,
			expectedProvider: &Provider{
				domain:        "example.com",
				owner:         "home",
				ipVersion:     ipversion.IP4or6,
				server:        testServerAddress,
				zone:          exampleZone,
				tsigAlgorithm: "hmac-sha256.",
				ttl:           300,
			},
		},
		"tsig and explicit settings": {
			data: `{"server":"[2001:db8::1]:5353","zone":"sub.example.com",` +
				`"tsig_key_name":"ddns-key","tsig_secret":"c2VjcmV0","tsig_algorithm":"hmac-sha512","ttl":60}`,
			expectedProvider: &Provider{ //nolint:gosec
				domain:        "example.com",
				owner:         "home",
				ipVersion:     ipversion.IP4or6,
				server:        "[2001:db8::1]:5353",
				zone:          "sub.example.com.",
				tsigKeyName:   "ddns-key.",
				tsigSecret:    "c2VjcmV0",
				tsigAlgorithm: "hmac-sha512.",
				ttl:           60,
			},
		},
		"server not set": {
			data:        `{}`,
			expectedErr: errors.ErrServerNotSet,
		},
		"server without a port": {
			data:        `{"server":"ns1.example.com"}`,
			expectedErr: errors.ErrServerNotValid,
		},
		"tsig key without secret": {
			data:        `{"server":"` + testServerAddress + `","tsig_key_name":"ddns-key"}`,
			expectedErr: errors.ErrSecretNotSet,
		},
		"tsig secret without key": {
			data:        `{"server":"` + testServerAddress + `","tsig_secret":"c2VjcmV0"}`,
			expectedErr: errors.ErrKeyNotSet,
		},
		"tsig secret not base64": {
			data:        `{"server":"` + testServerAddress + `","tsig_key_name":"k","tsig_secret":"not base64!"}`,
			expectedErr: errors.ErrSecretNotValid,
		},
		"unsupported algorithm": {
			data:        `{"server":"` + testServerAddress + `","tsig_algorithm":"hmac-md5"}`,
			expectedErr: errors.ErrAlgorithmNotValid,
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			provider, err := New(json.RawMessage(testCase.data), "example.com", "home",
				ipversion.IP4or6, netip.Prefix{})

			if testCase.expectedErr != nil {
				require.ErrorIs(t, err, testCase.expectedErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, testCase.expectedProvider, provider)
		})
	}
}

func Test_Provider_newUpdateMessage(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		owner          string
		ip             netip.Addr
		expectedName   string
		expectedType   uint16
		expectedRecord string
	}{
		"ipv4 subdomain": {
			owner:          "home",
			ip:             netip.MustParseAddr("1.2.3.4"),
			expectedName:   "home.example.com.",
			expectedType:   dns.TypeA,
			expectedRecord: "home.example.com.\t300\tIN\tA\t1.2.3.4",
		},
		"ipv6 root domain": {
			owner:          "@",
			ip:             netip.MustParseAddr("2001:db8::1"),
			expectedName:   "example.com.",
			expectedType:   dns.TypeAAAA,
			expectedRecord: "example.com.\t300\tIN\tAAAA\t2001:db8::1",
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			provider := &Provider{
				domain: "example.com",
				owner:  testCase.owner,
				zone:   exampleZone,
				ttl:    300,
			}

			message := provider.newUpdateMessage(testCase.ip)

			// The zone to update is carried in the question section.
			require.Len(t, message.Question, 1)
			assert.Equal(t, exampleZone, message.Question[0].Name)
			assert.Equal(t, dns.TypeSOA, message.Question[0].Qtype)
			assert.True(t, message.Opcode == dns.OpcodeUpdate)

			// The update section deletes the existing record set before
			// adding the new record, so the change is applied atomically.
			require.Len(t, message.Ns, 2)

			deletion := message.Ns[0]
			assert.Equal(t, testCase.expectedName, deletion.Header().Name)
			assert.Equal(t, testCase.expectedType, deletion.Header().Rrtype)
			assert.Equal(t, uint16(dns.ClassANY), deletion.Header().Class)

			addition := message.Ns[1]
			assert.Equal(t, uint16(dns.ClassINET), addition.Header().Class)
			assert.Equal(t, testCase.expectedRecord, addition.String())
		})
	}
}
