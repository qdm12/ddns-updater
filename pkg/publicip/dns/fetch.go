package dns

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/miekg/dns"
)

var (
	ErrNoTXTRecordFound  = errors.New("no TXT record found")
	ErrTooManyAnswers    = errors.New("too many answers")
	ErrInvalidAnswerType = errors.New("invalid answer type")
	ErrTooManyTXTRecords = errors.New("too many TXT records")
	ErrIPMalformed       = errors.New("IP address malformed")
)

func fetch(ctx context.Context, client Client, providerData providerData) (
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
		return nil, fmt.Errorf("%w: %d instead of 1", ErrTooManyAnswers, L)
	}

	answer := r.Answer[0]
	txt, ok := answer.(*dns.TXT)
	if !ok {
		return nil, fmt.Errorf("%w: %T instead of *dns.TXT",
			ErrInvalidAnswerType, answer)
	}

	L = len(txt.Txt)
	if L == 0 {
		return nil, ErrNoTXTRecordFound
	} else if L > 1 {
		return nil, fmt.Errorf("%w: %d instead of 1", ErrTooManyTXTRecords, L)
	}
	ipString := txt.Txt[0]

	publicIP = net.ParseIP(ipString)
	if publicIP == nil {
		return nil, fmt.Errorf("%w: %q", ErrIPMalformed, ipString)
	}

	return publicIP, nil
}
