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
	exampleZone = "example.com."
	ipv6Server  = "[2001:db8::1]:5353"
)

func Test_New(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		data          string
		expectedErr   error
		expectedZone  string
		expectedTTL   uint32
		expectedSrv   string
		expectedAlgo  string
		expectedKeyNm string
	}{
		"defaults": {
			data: `{"server":"ns1.example.com"}`,
			// zone defaults to the domain and the server to port 53.
			expectedZone: exampleZone,
			expectedTTL:  300,
			expectedSrv:  "ns1.example.com:53",
			expectedAlgo: "hmac-sha256.",
		},
		"tsig and explicit settings": {
			data: `{"server":"` + ipv6Server + `","zone":"sub.example.com",` +
				`"tsig_key_name":"ddns-key","tsig_secret":"c2VjcmV0","tsig_algorithm":"hmac-sha512","ttl":60}`,
			expectedZone:  "sub.example.com.",
			expectedTTL:   60,
			expectedSrv:   ipv6Server,
			expectedAlgo:  "hmac-sha512.",
			expectedKeyNm: "ddns-key.",
		},
		"server not set": {
			data:        `{}`,
			expectedErr: errors.ErrServerNotSet,
		},
		"tsig key without secret": {
			data:        `{"server":"ns1.example.com","tsig_key_name":"ddns-key"}`,
			expectedErr: errors.ErrSecretNotSet,
		},
		"tsig secret without key": {
			data:        `{"server":"ns1.example.com","tsig_secret":"c2VjcmV0"}`,
			expectedErr: errors.ErrKeyNotSet,
		},
		"tsig secret not base64": {
			data:        `{"server":"ns1.example.com","tsig_key_name":"k","tsig_secret":"not base64!"}`,
			expectedErr: errors.ErrSecretNotValid,
		},
		"unsupported algorithm": {
			data:        `{"server":"ns1.example.com","tsig_algorithm":"hmac-md5"}`,
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
			assert.Equal(t, testCase.expectedZone, provider.zone)
			assert.Equal(t, testCase.expectedTTL, provider.ttl)
			assert.Equal(t, testCase.expectedSrv, provider.server)
			assert.Equal(t, testCase.expectedAlgo, provider.tsigAlgorithm)
			assert.Equal(t, testCase.expectedKeyNm, provider.tsigKeyName)
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

func Test_addressWithDefaultPort(t *testing.T) {
	t.Parallel()

	testCases := map[string]string{
		"ns1.example.com":      "ns1.example.com:53",
		"ns1.example.com:5353": "ns1.example.com:5353",
		"192.168.1.1":          "192.168.1.1:53",
		"2001:db8::1":          "[2001:db8::1]:53",
		ipv6Server:             ipv6Server,
	}

	for server, expected := range testCases {
		t.Run(server, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, expected, addressWithDefaultPort(server))
		})
	}
}
