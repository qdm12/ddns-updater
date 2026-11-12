package config

import (
	"testing"

	"github.com/qdm12/ddns-updater/pkg/publicip/dns"
	"github.com/stretchr/testify/assert"
)

func Test_stringsToDNSProviders(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		providers        []string
		updatedProviders []dns.Provider
	}{
		"single_provider": {
			providers:        []string{"cloudflare"},
			updatedProviders: []dns.Provider{dns.Cloudflare},
		},
		"duplicate_providers": {
			providers:        []string{"cloudflare", "cloudflare"},
			updatedProviders: []dns.Provider{dns.Cloudflare},
		},
		"all_providers": {
			providers:        []string{"all"},
			updatedProviders: dns.ListProviders(),
		},
		"all_and_single_provider": {
			providers:        []string{"cloudflare", "all"},
			updatedProviders: dns.ListProviders(),
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			updatedProviders := stringsToDNSProviders(testCase.providers)

			assert.ElementsMatch(t, testCase.updatedProviders, updatedProviders)
		})
	}
}
