package dns

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"

	"github.com/miekg/dns"
)

var (
	ErrNetworkNotSupported    = errors.New("network not supported")
	ErrAnswerNotReceived      = errors.New("response answer not received")
	ErrAnswerTypeMismatch     = errors.New("answer type is not expected")
	ErrAnswerTypeNotSupported = errors.New("answer type not supported")
	ErrRecordEmpty            = errors.New("record is empty")
	ErrIPMalformed            = errors.New("IP address malformed")
)

func fetch(ctx context.Context, client Client, network string,
	providerData providerData) (publicIPs []netip.Addr, err error) {
	var serverHost string
	switch network {
	case "tcp":
		serverHost = providerData.Address
	case "tcp4":
		serverHost = providerData.IPv4.String()
	case "tcp6":
		serverHost = providerData.IPv6.String()
	default:
		return nil, fmt.Errorf("%w: %s", ErrNetworkNotSupported, network)
	}
	serverAddress := net.JoinHostPort(serverHost, "853")

	message := &dns.Msg{
		MsgHdr: dns.MsgHdr{
			Opcode: dns.OpcodeQuery,
		},
		Question: []dns.Question{
			{
				Name:   providerData.fqdn,
				Qtype:  uint16(providerData.qType),
				Qclass: uint16(providerData.class),
			},
		},
	}

	r, _, err := client.ExchangeContext(ctx, message, serverAddress)
	if err != nil {
		return nil, err
	}

	if len(r.Answer) == 0 {
		return nil, fmt.Errorf("%w", ErrAnswerNotReceived)
	}

	publicIPs = make([]netip.Addr, 0, len(r.Answer))
	for _, answer := range r.Answer {
		var publicIP netip.Addr
		switch uint16(providerData.qType) {
		case dns.TypeTXT:
			publicIP, err = handleAnswerTXT(answer)
		case dns.TypeANY:
			publicIP, err = handleAnswerANY(answer)
		default:
			return nil, fmt.Errorf("%w: %s",
				ErrAnswerTypeNotSupported, dns.TypeToString[uint16(providerData.qType)])
		}

		if err != nil {
			return nil, fmt.Errorf("handling %s answer: %w",
				providerData.qType.String(), err)
		}

		publicIPs = append(publicIPs, publicIP)
	}

	return publicIPs, nil
}

var (
	ErrTooManyTXTRecords = errors.New("too many TXT records")
)

func handleAnswerTXT(answer dns.RR) (publicIP netip.Addr, err error) {
	answerTXT, ok := answer.(*dns.TXT)
	if !ok {
		return netip.Addr{}, fmt.Errorf("%w: %T instead of *dns.TXT",
			ErrAnswerTypeMismatch, answer)
	}

	switch len(answerTXT.Txt) {
	case 0:
		return netip.Addr{}, fmt.Errorf("%w", ErrRecordEmpty)
	case 1:
	default:
		return netip.Addr{}, fmt.Errorf("%w: %d instead of 1",
			ErrTooManyTXTRecords, len(answerTXT.Txt))
	}

	publicIP, err = netip.ParseAddr(answerTXT.Txt[0])
	if err != nil {
		return netip.Addr{}, fmt.Errorf("%w: %w", ErrIPMalformed, err)
	}

	return publicIP, nil
}

func handleAnswerANY(answer dns.RR) (publicIP netip.Addr, err error) {
	rrType := answer.Header().Rrtype
	switch rrType {
	case dns.TypeA:
		publicIP, err = handleAnswerA(answer)
	case dns.TypeAAAA:
		publicIP, err = handleAnswerAAAA(answer)
	default:
		return netip.Addr{}, fmt.Errorf("%w: %s",
			ErrAnswerTypeNotSupported, dns.TypeToString[rrType])
	}

	if err != nil {
		return netip.Addr{}, fmt.Errorf("handling %s answer: %w",
			dns.TypeToString[rrType], err)
	}

	return publicIP, nil
}

func handleAnswerA(answer dns.RR) (publicIP netip.Addr, err error) {
	answerA, ok := answer.(*dns.A)
	if !ok {
		return netip.Addr{}, fmt.Errorf("%w: %T instead of *dns.A",
			ErrAnswerTypeMismatch, answer)
	}

	if len(answerA.A) == 0 {
		return netip.Addr{}, fmt.Errorf("%w", ErrRecordEmpty)
	}

	return netip.AddrFrom4([4]byte(answerA.A)), nil
}

func handleAnswerAAAA(answer dns.RR) (publicIP netip.Addr, err error) {
	answerAAAA, ok := answer.(*dns.AAAA)
	if !ok {
		return netip.Addr{}, fmt.Errorf("%w: %T instead of *dns.AAAA",
			ErrAnswerTypeMismatch, answer)
	}

	if len(answerAAAA.AAAA) == 0 {
		return netip.Addr{}, fmt.Errorf("%w", ErrRecordEmpty)
	}

	return netip.AddrFrom16([16]byte(answerAAAA.AAAA)), nil
}
