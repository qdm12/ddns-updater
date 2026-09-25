package resolver

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"strings"
	"time"
)

const (
	tlsDefaultPort   = "853"
	plainDefaultPort = "53"
)

func New(settings Settings) (resolver *net.Resolver, err error) {
	settings.setDefaults()
	err = settings.validate()
	if err != nil {
		return nil, fmt.Errorf("validating settings: %w", err)
	}

	if *settings.Address == "" {
		return net.DefaultResolver, nil
	}

	network, address := splitTransport(*settings.Address)
	address = applyDefaultPort(address, network)
	if network == "tls" {
		return newDoTResolver(address, settings.Timeout)
	}
	return newConnResolver(network, address, settings.Timeout), nil
}

// splitTransport extracts an optional transport scheme prefix such as
// udp://, tcp:// or tls:// from the address. Addresses without a
// recognized scheme are considered plaintext UDP.
func splitTransport(address string) (network string, rest string) {
	for _, scheme := range [...]string{"udp", "tcp", "tls"} {
		prefix := scheme + "://"
		if strings.HasPrefix(address, prefix) {
			return scheme, strings.TrimPrefix(address, prefix)
		}
	}
	return "udp", address
}

func applyDefaultPort(address string, network string) string {
	if _, _, err := net.SplitHostPort(address); err == nil {
		return address // explicit port is already set
	}
	return net.JoinHostPort(address, defaultPortFor(network))
}

func defaultPortFor(network string) string {
	if network == "tls" {
		return tlsDefaultPort
	}
	return plainDefaultPort
}

func newConnResolver(network, address string, timeout time.Duration) *net.Resolver {
	dialer := net.Dialer{Timeout: timeout}
	return &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return dialer.DialContext(ctx, network, address)
		},
	}
}

// newDoTResolver returns a resolver speaking DNS-over-TLS (RFC 7858).
// A tls.Conn satisfies net.Conn, so the net.Resolver automatically
// switches to its TCP-based DNS exchange mode.
func newDoTResolver(address string, timeout time.Duration) (*net.Resolver, error) {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return nil, fmt.Errorf("splitting host and port from DNS-over-TLS address %q: %w", address, err)
	}
	dialer := net.Dialer{Timeout: timeout}
	tlsConfig := &tls.Config{ServerName: host} //nolint:gosec
	return &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, _, _ string) (net.Conn, error) {
			conn, dialErr := dialer.DialContext(ctx, "tcp", address)
			if dialErr != nil {
				return nil, dialErr
			}
			tlsConn := tls.Client(conn, tlsConfig)
			if handshakeErr := tlsConn.HandshakeContext(ctx); handshakeErr != nil {
				_ = conn.Close()
				return nil, fmt.Errorf("TLS handshake with %q: %w", address, handshakeErr)
			}
			return tlsConn, nil
		},
	}, nil
}
