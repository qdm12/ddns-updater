package resolver

import (
	"strings"
	"testing"
)

func Test_splitTransport(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		address string
		network string
		rest    string
	}{
		"no scheme":            {address: "1.1.1.1:53", network: "udp", rest: "1.1.1.1:53"},
		"udp scheme":           {address: "udp://1.1.1.1", network: "udp", rest: "1.1.1.1"},
		"tcp scheme":           {address: "tcp://1.1.1.1", network: "tcp", rest: "1.1.1.1"},
		"tls scheme":           {address: "tls://223.5.5.5", network: "tls", rest: "223.5.5.5"},
		"tls scheme with port": {address: "tls://dns.google:853", network: "tls", rest: "dns.google:853"},
	}
	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			network, rest := splitTransport(tc.address)
			if network != tc.network {
				t.Errorf("network: got %q, expected %q", network, tc.network)
			}
			if rest != tc.rest {
				t.Errorf("rest: got %q, expected %q", rest, tc.rest)
			}
		})
	}
}

func Test_applyDefaultPort(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		address  string
		network  string
		expected string
	}{
		"plain gets udp default port":   {"1.1.1.1", "udp", "1.1.1.1:53"},
		"tls default port is 853":       {"223.5.5.5", "tls", "223.5.5.5:853"},
		"explicit port kept for plain":  {"1.1.1.1:5353", "udp", "1.1.1.1:5353"},
		"explicit port kept for tls":    {"dns.google:8853", "tls", "dns.google:8853"},
		"ipv6 host with default port 1": {"2001:db8::1", "udp", "[2001:db8::1]:53"},
	}
	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := applyDefaultPort(tc.address, tc.network)
			if got != tc.expected {
				t.Errorf("got %q, expected %q", got, tc.expected)
			}
		})
	}
}

func Test_validate_scheme(t *testing.T) {
	t.Parallel()

	validAddresses := []string{
		"",
		"1.1.1.1",
		"1.1.1.1:53",
		"udp://1.1.1.1",
		"tcp://1.0.0.1:53",
		"tls://223.5.5.5",
		"tls://dns.google:853",
	}
	for _, address := range validAddresses {
		settings := Settings{Address: &address}
		settings.setDefaults()
		if err := settings.validate(); err != nil {
			t.Errorf("%q should be valid, got error: %s", address, err)
		}
	}

	invalidAddresses := []string{"ftp://1.1.1.1", "https://dns.google/dns-query"}
	for _, address := range invalidAddresses {
		settings := Settings{Address: &address}
		settings.setDefaults()
		err := settings.validate()
		if err == nil {
			t.Errorf("%q should be rejected", address)
			continue
		}
		if !strings.Contains(err.Error(), ErrUnsupportedScheme.Error()) {
			t.Errorf("%q error should mention unsupported scheme, got: %s", address, err)
		}
	}
}

func Test_validate_emptyHost(t *testing.T) {
	t.Parallel()
	for _, address := range []string{":53", "tls://"} {
		settings := Settings{Address: &address}
		settings.setDefaults()
		err := settings.validate()
		if err == nil {
			t.Errorf("%q should be rejected (empty host)", address)
		}
	}
}
