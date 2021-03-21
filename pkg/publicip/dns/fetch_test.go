package dns

import (
	"context"
	"errors"
	"net"
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
		nameserver: "nameserver",
		fqdn:       "record",
		class:      dns.ClassNONE,
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
		publicIP    net.IP
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
			publicIP: net.IP{55, 55, 55, 55},
		},
		"exchange error": {
			exchangeErr: errors.New("dummy"),
			err:         errors.New("dummy"),
		},
		"no answer": {
			response: &dns.Msg{},
			err:      ErrNoTXTRecordFound,
		},
		"too many answers": {
			response: &dns.Msg{
				Answer: []dns.RR{&dns.TXT{}, &dns.TXT{}},
			},
			err: errors.New("too many answers: 2 instead of 1"),
		},
		"wrong answer type": {
			response: &dns.Msg{
				Answer: []dns.RR{&dns.A{}},
			},
			err: errors.New("invalid answer type: *dns.A instead of *dns.TXT"),
		},
		"no TXT record": {
			response: &dns.Msg{
				Answer: []dns.RR{&dns.TXT{}},
			},
			err: errors.New("no TXT record found"),
		},
		"too many TXT record": {
			response: &dns.Msg{
				Answer: []dns.RR{&dns.TXT{
					Txt: []string{"a", "b"},
				}},
			},
			err: errors.New("too many TXT records: 2 instead of 1"),
		},
		"invalid IP address": {
			response: &dns.Msg{
				Answer: []dns.RR{&dns.TXT{
					Txt: []string{"invalid"},
				}},
			},
			err: errors.New(`IP address malformed: "invalid"`),
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			ctx := context.Background()

			client := mock_dns.NewMockClient(ctrl)
			client.EXPECT().
				ExchangeContext(ctx, expectedMessage, providerData.nameserver).
				Return(testCase.response, time.Millisecond, testCase.exchangeErr)

			publicIP, err := fetch(ctx, client, providerData)

			if testCase.err != nil {
				require.Error(t, err)
				assert.Equal(t, testCase.err.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}

			if !testCase.publicIP.Equal(publicIP) {
				t.Errorf("IP address mismatch: expected %s and got %s", testCase.publicIP, publicIP)
			}
		})
	}
}
