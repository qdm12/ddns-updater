package dns

import (
	"context"
	"errors"
	"fmt"
	"net"
)

var (
	ErrNoTXTRecordFound  = errors.New("no TXT record found")
	ErrTooManyTXTRecords = errors.New("too many TXT records")
	ErrIPMalformed       = errors.New("IP address malformed")
)

func fetch(ctx context.Context, resolver *net.Resolver, txtRecord string) (
	publicIP net.IP, err error) {
	records, err := resolver.LookupTXT(ctx, txtRecord)
	if err != nil {
		return nil, err
	}

	L := len(records)
	if L == 0 {
		return nil, ErrNoTXTRecordFound
	} else if L > 1 {
		return nil, fmt.Errorf("%w: %d instead of 1", ErrTooManyTXTRecords, L)
	}

	publicIP = net.ParseIP(records[0])
	if publicIP == nil {
		return nil, fmt.Errorf("%w: %s", ErrIPMalformed, records[0])
	}

	return publicIP, nil
}
