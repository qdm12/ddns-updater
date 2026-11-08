package schema

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/qdm12/ddns-updater/internal/provider/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Providers(t *testing.T) {
	t.Parallel()

	providers := Providers()
	require.NotEmpty(t, providers)

	names := make(map[string]struct{}, len(providers))
	for _, provider := range providers {
		_, duplicate := names[provider.Name]
		assert.False(t, duplicate, "duplicate provider %q", provider.Name)
		names[provider.Name] = struct{}{}

		assert.NotEmpty(t, provider.Label, provider.Name)
		assert.NotEmpty(t, provider.DocsPath, provider.Name)
		assert.NotEmpty(t, provider.Fields, provider.Name)
		assert.NotEmpty(t, provider.DocsURL(), provider.Name)

		for _, field := range provider.Fields {
			assert.NotEmpty(t, field.Key, provider.Name)
			assert.NotEmpty(t, field.Label, provider.Name+"."+field.Key)
			assert.Contains(t,
				[]Kind{KindString, KindBool, KindNumber, KindJSON},
				field.Kind, provider.Name+"."+field.Key)
			assert.False(t, IsCommonKey(field.Key),
				"%s.%s must not repeat a common setting", provider.Name, field.Key)
		}
	}

	for _, choice := range constants.ProviderChoices() {
		_, found := names[string(choice)]
		assert.True(t, found, "provider %q is missing from the schema", choice)
	}
}

func Test_Lookup(t *testing.T) {
	t.Parallel()

	provider, found := Lookup("  CloudFlare ")
	require.True(t, found)
	assert.Equal(t, "cloudflare", provider.Name)
	assert.Equal(t, "Cloudflare", provider.Label)
	assert.Equal(t, "cloudflare.md", provider.DocsPath)
	assert.Equal(t,
		"https://github.com/qdm12/ddns-updater/blob/master/docs/cloudflare.md",
		provider.DocsURL())

	_, found = Lookup("not-a-provider")
	assert.False(t, found)
}

func Test_LookupField(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		provider string
		key      string
		expected Field
		found    bool
	}{
		"cloudflare_token": {
			provider: "cloudflare",
			key:      "token",
			expected: Field{Key: "token", Label: "Token", Kind: KindString, Secret: true},
			found:    true,
		},
		"cloudflare_ttl": {
			provider: "cloudflare",
			key:      "ttl",
			expected: Field{Key: "ttl", Label: "TTL (seconds)", Kind: KindNumber, Help: "Record time to live in seconds, when the provider supports it."},
			found:    true,
		},
		"cloudflare_proxied": {
			provider: "cloudflare",
			key:      "proxied",
			expected: Field{Key: "proxied", Label: "Proxied through Cloudflare", Kind: KindBool, Help: "Route the record through the provider proxy."},
			found:    true,
		},
		"gcp_credentials": {
			provider: "gcp",
			key:      "credentials",
			expected: Field{Key: "credentials", Label: "Credentials JSON", Kind: KindJSON, Secret: true, Help: "Service account credentials as a JSON object."},
			found:    true,
		},
		"unknown_provider": {
			provider: "nope",
			key:      "token",
		},
		"unknown_key": {
			provider: "cloudflare",
			key:      "nope",
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			field, found := LookupField(testCase.provider, testCase.key)
			assert.Equal(t, testCase.found, found)
			assert.Equal(t, testCase.expected, field)
		})
	}
}

func Test_Keys(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []string{
		"email", "key", "proxied", "token", "ttl", "zone_identifier",
	}, Keys("cloudflare"))
	assert.Nil(t, Keys("nope"))
}

func Test_ProviderDocumentationExists(t *testing.T) {
	t.Parallel()

	docsDirectory := filepath.Join("..", "..", "..", "docs")
	for _, provider := range Providers() {
		path := filepath.Join(docsDirectory, provider.DocsPath)
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("provider %q: %s", provider.Name, err)
			continue
		}
		assert.False(t, info.IsDir(), "provider %q: %s is a directory",
			provider.Name, path)
	}
}

func Test_ProvidersReturnsCopy(t *testing.T) {
	t.Parallel()

	first := Providers()
	first[0].Fields[0].Key = "mutated"
	first[0].Name = "mutated"

	second := Providers()
	assert.NotEqual(t, "mutated", second[0].Name)
	assert.NotEqual(t, "mutated", second[0].Fields[0].Key)
}
