package dns

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/miekg/dns"
)

var (
	ErrNoTXTRecordFound  = errors.New("no TXT record found")
	ErrTooManyTXTRecords = errors.New("too many TXT records")
	ErrIPMalformed       = errors.New("IP address malformed")
)

func fetch(ctx context.Context, client *dns.Client, providerData providerData) (
	publicIP net.IP, err error) {
	message := &dns.Msg{
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

	r, _, err := client.ExchangeContext(ctx, message, providerData.nameserver)
	if err != nil {
		return nil, err
	}

	L := len(r.Answer)
	if L == 0 {
		return nil, ErrNoTXTRecordFound
	} else if L > 1 {
		return nil, fmt.Errorf("%w: %d instead of 1", ErrTooManyTXTRecords, L)
	}

	answer := r.Answer[0]
	fields := strings.Fields(answer.String())
	ipString := fields[len(fields)-1]
	ipString = strings.TrimPrefix(ipString, `"`)
	ipString = strings.TrimSuffix(ipString, `"`)

	publicIP = net.ParseIP(ipString)
	if publicIP == nil {
		return nil, fmt.Errorf("%w: %s", ErrIPMalformed, ipString)
	}

	return publicIP, nil
}
