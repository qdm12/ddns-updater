package resolver

import (
	"context"
	"fmt"
	"net"
)

func New(settings Settings) (resolver *net.Resolver, err error) {
	settings.SetDefaults()
	err = settings.Validate()
	if err != nil {
		return nil, fmt.Errorf("validating settings: %w", err)
	}

	if *settings.Address == "" {
		return net.DefaultResolver, nil
	}

	dialer := net.Dialer{Timeout: settings.Timeout}
	return &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			const protocol = "udp"
			return dialer.DialContext(ctx, protocol, *settings.Address)
		},
	}, nil
}
