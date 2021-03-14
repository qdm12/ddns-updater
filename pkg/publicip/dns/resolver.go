package dns

import (
	"context"
	"net"
)

type dialFunc func(ctx context.Context, network string, address string) (net.Conn, error)

func newResolver(dialer *net.Dialer, udpNetwork, nameserver string) *net.Resolver {
	return &net.Resolver{
		PreferGo: true,
		Dial:     newDial(dialer, udpNetwork, nameserver),
	}
}

func newDial(dialer *net.Dialer, udpNetwork, nameserver string) dialFunc {
	return func(ctx context.Context, _, _ string) (net.Conn, error) {
		return dialer.DialContext(ctx, udpNetwork, nameserver)
	}
}
