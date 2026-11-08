// Package schema describes the provider specific JSON settings accepted by
// every provider, so that a form can be rendered for any of them, for
// example from the web UI.
//
// The field list is generated from the provider source code, which is the
// single source of truth for the accepted JSON keys. Regenerate it with:
//
//	go generate ./internal/provider/schema
package schema

//go:generate go run ./gen/main.go

import (
	"fmt"
	"sort"
	"strings"
)

// Kind is the JSON value kind of a provider specific setting.
type Kind string

const (
	KindString Kind = "string"
	KindBool   Kind = "bool"
	KindNumber Kind = "number"
	// KindJSON is a setting holding a JSON value, such as an object, an
	// array or a free-form JSON document.
	KindJSON Kind = "json"
)

// Field is one provider specific setting.
type Field struct {
	// Key is the JSON key of the setting, for example "api_key".
	Key string `json:"key"`
	// Label is a human readable label for the setting.
	Label string `json:"label"`
	// Kind is the JSON value kind of the setting.
	Kind Kind `json:"kind"`
	// Secret is true if the value should be hidden by default, such as a
	// password, a token or an API key.
	Secret bool `json:"secret,omitempty"`
	// Help is a short description of the setting, when known.
	Help string `json:"help,omitempty"`
}

// Provider describes the settings of one provider.
type Provider struct {
	// Name is the provider name to use in the "provider" JSON field.
	Name string `json:"name"`
	// Label is a human readable provider name.
	Label string `json:"label"`
	// DocsPath is the file name of the provider documentation in the docs
	// directory, for example "cloudflare.md".
	DocsPath string `json:"docs_path"`
	// Fields are the provider specific settings, excluding the common
	// settings such as domain, ip_version and ipv6_suffix.
	Fields []Field `json:"fields"`
}

// DocsBaseURL is the base URL of the provider documentation on GitHub.
const DocsBaseURL = "https://github.com/qdm12/ddns-updater/blob/master/docs/"

// DocsURL returns the documentation URL of the provider.
func (p Provider) DocsURL() string {
	return DocsBaseURL + p.DocsPath
}

// Providers returns all the provider schemas, sorted by provider name.
func Providers() []Provider {
	all := make([]Provider, len(providers))
	copy(all, providers)
	for i := range all {
		fields := make([]Field, len(providers[i].Fields))
		copy(fields, providers[i].Fields)
		all[i].Fields = fields
	}
	return all
}

// Lookup returns the schema of the provider with the given name.
func Lookup(name string) (provider Provider, found bool) {
	name = strings.TrimSpace(strings.ToLower(name))
	for _, provider := range providers {
		if provider.Name == name {
			return provider, true
		}
	}
	return Provider{}, false
}

// Field returns the schema of the setting with the given key for the
// provider with the given name.
func LookupField(providerName, key string) (field Field, found bool) {
	provider, found := Lookup(providerName)
	if !found {
		return Field{}, false
	}
	for _, field := range provider.Fields {
		if field.Key == key {
			return field, true
		}
	}
	return Field{}, false
}

// Keys returns the sorted setting keys of the provider with the given name.
func Keys(providerName string) []string {
	provider, found := Lookup(providerName)
	if !found {
		return nil
	}
	keys := make([]string, len(provider.Fields))
	for i, field := range provider.Fields {
		keys[i] = field.Key
	}
	sort.Strings(keys)
	return keys
}

// CommonKeys are the JSON keys accepted by all providers, in addition to
// the provider specific fields.
var CommonKeys = []string{"provider", "domain", "ip_version", "ipv6_suffix"}

// IsCommonKey returns true if the JSON key is a common setting accepted by
// all providers.
func IsCommonKey(key string) bool {
	for _, commonKey := range CommonKeys {
		if key == commonKey {
			return true
		}
	}
	return false
}

// fieldLabels overrides the label inferred from the JSON key.
var fieldLabels = map[string]string{ //nolint:gochecknoglobals
	"api_key":          "API key",
	"apikey":           "API key",
	"api_secret":       "API secret",
	"apisecret":        "API secret",
	"api_endpoint":     "API endpoint",
	"access_key":       "Access key",
	"access_key_id":    "Access key ID",
	"access_secret":    "Access secret",
	"client_key":       "Client key",
	"consumer_key":     "Consumer key",
	"app_key":          "Application key",
	"app_secret":       "Application secret",
	"secret_key":       "Secret key",
	"secretapikey":     "Secret API key",
	"secret_api_key":   "Secret API key",
	"user_service_key": "User service key",
	"ipv4key":          "IPv4 query key",
	"ipv6key":          "IPv6 query key",
	"success_regex":    "Success regex",
	"ttl":              "TTL (seconds)",
	"url":              "URL",
	"zone_id":          "Zone ID",
	"zone_identifier":  "Zone identifier",
	"customer_number":  "Customer number",
	"customernumber":   "Customer number",
	"email":            "Email",
	"username":         "Username",
	"password":         "Password",
	"token":            "Token",
	"key":              "Key",
	"region":           "Region",
	"project":          "Project",
	"zone":             "Zone",
	"credentials":      "Credentials JSON",
	"proxied":          "Proxied through Cloudflare",
	"dual_stack":       "Dual stack",
	"mode":             "Mode",
	"line":             "Line",
}

// fieldHelp documents the settings whose meaning is not obvious from the
// key alone. It is applied to every provider.
var fieldHelp = map[string]string{ //nolint:gochecknoglobals
	"api_key":          "API key created in the provider account.",
	"apikey":           "API key created in the provider account.",
	"api_secret":       "API secret created in the provider account.",
	"api_endpoint":     "API endpoint to use, usually a short name such as ovh-eu.",
	"credentials":      "Service account credentials as a JSON object.",
	"dual_stack":       "Update both the IPv4 and the IPv6 records.",
	"ipv4key":          "URL query parameter name receiving the IPv4 address.",
	"ipv6key":          "URL query parameter name receiving the IPv6 address.",
	"key":              "API key or account key, depending on the provider.",
	"mode":             "Update mode required by the provider API.",
	"proxied":          "Route the record through the provider proxy.",
	"success_regex":    "Regular expression matching a successful API response.",
	"ttl":              "Record time to live in seconds, when the provider supports it.",
	"url":              "Update URL, without the IP address query parameter.",
	"user_service_key": "Cloudflare user service key, if not using an API token.",
	"zone_identifier":  "Zone identifier of the domain in the provider account.",
	"zone_id":          "Zone identifier of the domain in the provider account.",
	"zone":             "Zone name of the domain in the provider account.",
	"project":          "Google Cloud project identifier.",
	"line":             "DNS line to update, for example \"default\".",
}

// secretKeys are the JSON keys whose value must be hidden by default.
var secretKeys = map[string]struct{}{ //nolint:gochecknoglobals
	"access_key":       {},
	"access_key_id":    {},
	"access_secret":    {},
	"api_key":          {},
	"apikey":           {},
	"api_secret":       {},
	"apisecret":        {},
	"app_key":          {},
	"app_secret":       {},
	"client_key":       {},
	"consumer_key":     {},
	"credentials":      {},
	"key":              {},
	"password":         {},
	"secret":           {},
	"secret_api_key":   {},
	"secret_key":       {},
	"secretapikey":     {},
	"token":            {},
	"user_service_key": {},
}

func init() { //nolint:gochecknoinits
	for i := range providers {
		for j := range providers[i].Fields {
			field := &providers[i].Fields[j]
			if label, ok := fieldLabels[field.Key]; ok {
				field.Label = label
			} else {
				field.Label = humanizeKey(field.Key)
			}
			field.Secret = isSecretKey(field.Key)
			field.Help = fieldHelp[field.Key]
		}
	}
}

func isSecretKey(key string) bool {
	_, secret := secretKeys[key]
	return secret
}

// humanizeKey turns a JSON key such as "customer_number" into
// "Customer number".
func humanizeKey(key string) string {
	words := strings.FieldsFunc(key, func(r rune) bool {
		return r == '_' || r == '-' || r == '.'
	})
	for i, word := range words {
		if word == "" {
			continue
		}
		words[i] = strings.ToUpper(word[:1]) + word[1:]
	}
	label := strings.Join(words, " ")
	if label == "" {
		return fmt.Sprintf("%q", key)
	}
	return label
}
