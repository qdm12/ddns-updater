package dns

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/miekg/dns"
	"github.com/qdm12/ddns-updater/pkg/publicip/dns/mock_dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_fetch(t *testing.T) {
	t.Parallel()

	providerData := providerData{
		TLSName: "nameserver",
		fqdn:    "record",
		class:   dns.ClassNONE,
		qType:   dns.Type(dns.TypeTXT),
	}

	expectedMessage := &dns.Msg{
		MsgHdr: dns.MsgHdr{
			Opcode: dns.OpcodeQuery,
		},
		Question: []dns.Question{
			{
				Name:   providerData.fqdn,
				Qtype:  dns.TypeTXT,
				Qclass: uint16(providerData.class),
			},
		},
	}

	testCases := map[string]struct {
		response    *dns.Msg
		exchangeErr error
		publicIPs   []netip.Addr
		err         error
	}{
		"success": {
			response: &dns.Msg{
				Answer: []dns.RR{
					&dns.TXT{
						Txt: []string{"55.55.55.55"},
					},
				},
			},
			publicIPs: []netip.Addr{netip.AddrFrom4([4]byte{55, 55, 55, 55})},
		},
		"exchange error": {
			exchangeErr: errors.New("dummy"),
			err:         errors.New("dummy"),
		},
		"no answer": {
			response: &dns.Msg{},
			err:      ErrAnswerNotReceived,
		},
		"wrong answer type": {
			response: &dns.Msg{
				Answer: []dns.RR{&dns.A{}},
			},
			err: errors.New("handling TXT answer: answer type is not expected: " +
				"*dns.A instead of *dns.TXT"),
		},
		"no TXT record": {
			response: &dns.Msg{
				Answer: []dns.RR{&dns.TXT{}},
			},
			err: errors.New("handling TXT answer: record is empty"),
		},
		"too many TXT record": {
			response: &dns.Msg{
				Answer: []dns.RR{&dns.TXT{
					Txt: []string{"a", "b"},
				}},
			},
			err: errors.New("handling TXT answer: too many TXT records: 2 instead of 1"),
		},
		"invalid IP address": {
			response: &dns.Msg{
				Answer: []dns.RR{&dns.TXT{
					Txt: []string{"invalid"},
				}},
			},
			err: errors.New(`handling TXT answer: IP address malformed: ParseAddr("invalid"): unable to parse IP`),
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			ctx := context.Background()

			client := mock_dns.NewMockClient(ctrl)
			expectedAddress := net.JoinHostPort(providerData.Address, "853")
			client.EXPECT().
				ExchangeContext(ctx, expectedMessage, expectedAddress).
				Return(testCase.response, time.Millisecond, testCase.exchangeErr)

			const network = "tcp" // so it picks the Address field as the address
			publicIPs, err := fetch(ctx, client, network, providerData)

			if testCase.err != nil {
				require.Error(t, err)
				assert.Equal(t, testCase.err.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}

			for i := range testCase.publicIPs {
				expectedPublicIP := testCase.publicIPs[i]
				actualPublicIP := publicIPs[i]
				if expectedPublicIP.Compare(actualPublicIP) != 0 {
					t.Errorf("IP address mismatch: expected %s and got %s",
						expectedPublicIP, actualPublicIP)
				}
			}
		})
	}
}
