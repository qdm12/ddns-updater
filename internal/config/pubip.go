package config

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/qdm12/ddns-updater/pkg/publicip/dns"
	"github.com/qdm12/ddns-updater/pkg/publicip/http"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
	"github.com/qdm12/gosettings"
	"github.com/qdm12/gosettings/reader"
	"github.com/qdm12/gosettings/validate"
	"github.com/qdm12/gotree"
)

type PubIP struct {
	HTTPEnabled       *bool
	HTTPIPProviders   []string
	HTTPIPv4Providers []string
	HTTPIPv6Providers []string
	DNSEnabled        *bool
	DNSProviders      []string
	DNSTimeout        time.Duration
}

func (p *PubIP) setDefaults() {
	p.HTTPEnabled = gosettings.DefaultPointer(p.HTTPEnabled, true)
	p.HTTPIPProviders = gosettings.DefaultSlice(p.HTTPIPProviders, []string{all})
	p.HTTPIPv4Providers = gosettings.DefaultSlice(p.HTTPIPv4Providers, []string{all})
	p.HTTPIPv6Providers = gosettings.DefaultSlice(p.HTTPIPv6Providers, []string{all})
	p.DNSEnabled = gosettings.DefaultPointer(p.DNSEnabled, true)
	p.DNSProviders = gosettings.DefaultSlice(p.DNSProviders, []string{all})
	const defaultDNSTimeout = 3 * time.Second
	p.DNSTimeout = gosettings.DefaultComparable(p.DNSTimeout, defaultDNSTimeout)
}

func (p PubIP) Validate() (err error) {
	err = p.validateHTTPIPProviders()
	if err != nil {
		return fmt.Errorf("HTTP IP providers: %w", err)
	}

	err = p.validateHTTPIPv4Providers()
	if err != nil {
		return fmt.Errorf("HTTP IPv4 providers: %w", err)
	}

	err = p.validateHTTPIPv6Providers()
	if err != nil {
		return fmt.Errorf("HTTP IPv6 providers: %w", err)
	}

	err = p.validateDNSProviders()
	if err != nil {
		return fmt.Errorf("DNS providers: %w", err)
	}

	return nil
}

func (p *PubIP) String() string {
	return p.toLinesNode().String()
}

func (p *PubIP) toLinesNode() (node *gotree.Node) {
	node = gotree.New("Public IP fetching")

	node.Appendf("HTTP enabled: %s", gosettings.BoolToYesNo(p.HTTPEnabled))
	if *p.HTTPEnabled {
		childNode := node.Appendf("HTTP IP providers")
		for _, provider := range p.HTTPIPProviders {
			childNode.Appendf(provider)
		}

		childNode = node.Appendf("HTTP IPv4 providers")
		for _, provider := range p.HTTPIPv4Providers {
			childNode.Appendf(provider)
		}

		childNode = node.Appendf("HTTP IPv6 providers")
		for _, provider := range p.HTTPIPv6Providers {
			childNode.Appendf(provider)
		}
	}

	node.Appendf("DNS enabled: %s", gosettings.BoolToYesNo(p.DNSEnabled))
	if *p.DNSEnabled {
		node.Appendf("DNS timeout: %s", p.DNSTimeout)
		childNode := node.Appendf("DNS over TLS providers")
		for _, provider := range p.DNSProviders {
			childNode.Appendf(provider)
		}
	}

	return node
}

// ToHTTPOptions assumes the settings have been validated.
func (p *PubIP) ToHTTPOptions() (options []http.Option) {
	httpIPProviders := stringsToHTTPProviders(p.HTTPIPProviders, ipversion.IP4or6)
	httpIPv4Providers := stringsToHTTPProviders(p.HTTPIPv4Providers, ipversion.IP4)
	httpIPv6Providers := stringsToHTTPProviders(p.HTTPIPv6Providers, ipversion.IP6)
	return []http.Option{
		http.SetProvidersIP(httpIPProviders[0], httpIPProviders[1:]...),
		http.SetProvidersIP4(httpIPv4Providers[0], httpIPv4Providers[1:]...),
		http.SetProvidersIP6(httpIPv6Providers[0], httpIPv6Providers[1:]...),
	}
}

func stringsToHTTPProviders(providers []string, ipVersion ipversion.IPVersion) (
	updatedProviders []http.Provider) {
	updatedProvidersSet := make(map[string]struct{}, len(providers))
	for _, provider := range providers {
		if provider != all {
			updatedProvidersSet[provider] = struct{}{}
			continue
		}

		allProviders := http.ListProvidersForVersion(ipVersion)
		for _, provider := range allProviders {
			updatedProvidersSet[string(provider)] = struct{}{}
		}
	}

	updatedProviders = make([]http.Provider, 0, len(updatedProvidersSet))
	for provider := range updatedProvidersSet {
		updatedProviders = append(updatedProviders, http.Provider(provider))
	}

	return updatedProviders
}

// ToDNSPOptions assumes the settings have been validated.
func (p *PubIP) ToDNSPOptions() (options []dns.Option) {
	uniqueProviders := make(map[string]struct{}, len(p.DNSProviders))
	for _, provider := range p.DNSProviders {
		if provider != all {
			uniqueProviders[provider] = struct{}{}
		}

		allProviders := dns.ListProviders()
		for _, provider := range allProviders {
			uniqueProviders[string(provider)] = struct{}{}
		}
	}

	providers := make([]dns.Provider, 0, len(uniqueProviders))
	for providerString := range uniqueProviders {
		providers = append(providers, dns.Provider(providerString))
	}

	return []dns.Option{
		dns.SetTimeout(p.DNSTimeout),
		dns.SetProviders(providers[0], providers[1:]...),
	}
}

var (
	ErrNoPublicIPDNSProvider = errors.New("no public IP DNS provider specified")
)

func (p PubIP) validateDNSProviders() (err error) {
	if len(p.DNSProviders) == 0 {
		return fmt.Errorf("%w", ErrNoPublicIPDNSProvider)
	}

	availableProviders := dns.ListProviders()
	validChoices := make([]string, len(availableProviders)+1)
	for i, provider := range availableProviders {
		validChoices[i] = string(provider)
	}
	validChoices[len(validChoices)-1] = all
	return validate.AreAllOneOf(p.DNSProviders, validChoices)
}

func (p PubIP) validateHTTPIPProviders() (err error) {
	return validateHTTPIPProviders(p.HTTPIPProviders, ipversion.IP4or6)
}

func (p PubIP) validateHTTPIPv4Providers() (err error) {
	return validateHTTPIPProviders(p.HTTPIPv4Providers, ipversion.IP4)
}

func (p PubIP) validateHTTPIPv6Providers() (err error) {
	return validateHTTPIPProviders(p.HTTPIPv6Providers, ipversion.IP6)
}

var (
	ErrNoPublicIPHTTPProvider = errors.New("no public IP HTTP provider specified")
	ErrURLIsNotValidHTTPS     = errors.New("URL is not valid or not HTTPS")
)

func validateHTTPIPProviders(providerStrings []string,
	version ipversion.IPVersion) (err error) {
	if len(providerStrings) == 0 {
		return fmt.Errorf("%w", ErrNoPublicIPHTTPProvider)
	}

	availableProviders := http.ListProvidersForVersion(version)
	choices := make(map[string]struct{}, len(availableProviders)+1)
	choices[all] = struct{}{}
	for i := range availableProviders {
		choices[string(availableProviders[i])] = struct{}{}
	}

	for _, providerString := range providerStrings {
		if providerString == "noip" {
			// NoIP is no longer supported because the echo service
			// only works over plaintext HTTP and could be tempered with.
			// Silently discard it and it will default to another HTTP IP
			// echo service.
			continue
		}

		// Custom URL check
		if strings.HasPrefix(providerString, "url:") {
			url, err := url.Parse(providerString[4:])
			if err != nil || url.Scheme != "https" {
				return fmt.Errorf("%w: %s", ErrURLIsNotValidHTTPS, providerString)
			}
			continue
		}

		_, ok := choices[providerString]
		if !ok {
			return fmt.Errorf("%w: %s", validate.ErrValueNotOneOf, providerString)
		}
	}

	return nil
}

func (p *PubIP) read(r *reader.Reader, warner Warner) (err error) {
	p.HTTPEnabled, p.DNSEnabled, err = getFetchers(r)
	if err != nil {
		return err
	}

	p.HTTPIPProviders = r.CSV("PUBLICIP_HTTP_PROVIDERS",
		reader.RetroKeys("IP_METHOD"))
	p.HTTPIPv4Providers = r.CSV("PUBLICIPV4_HTTP_PROVIDERS",
		reader.RetroKeys("IPV4_METHOD"))
	p.HTTPIPv6Providers = r.CSV("PUBLICIPV6_HTTP_PROVIDERS",
		reader.RetroKeys("IPV6_METHOD"))

	// Retro-compatibility
	for i := range p.HTTPIPProviders {
		p.HTTPIPProviders[i] = handleRetroProvider(p.HTTPIPProviders[i])
	}
	for i := range p.HTTPIPv4Providers {
		p.HTTPIPv4Providers[i] = handleRetroProvider(p.HTTPIPv4Providers[i])
	}
	for i := range p.HTTPIPv6Providers {
		p.HTTPIPv6Providers[i] = handleRetroProvider(p.HTTPIPv6Providers[i])
	}

	// Retro-compatibility for now defunct opendns http provider for ipv4 or ipv6
	if len(p.HTTPIPProviders) > 0 { // check to avoid transforming `nil` to `[]`
		httpIPProvidersTemp := make([]string, len(p.HTTPIPProviders))
		copy(httpIPProvidersTemp, p.HTTPIPProviders)
		p.HTTPIPProviders = make([]string, 0, len(p.HTTPIPProviders))
		for _, provider := range httpIPProvidersTemp {
			if provider != "opendns" {
				p.HTTPIPProviders = append(p.HTTPIPProviders, provider)
			}
		}
	}

	p.DNSProviders = r.CSV("PUBLICIP_DNS_PROVIDERS")

	// Retro-compatibility
	for i, provider := range p.DNSProviders {
		if provider == "google" {
			warner.Warnf("dns provider google will be ignored " +
				"since it is no longer supported, " +
				"see https://github.com/qdm12/ddns-updater/issues/492")
			p.DNSProviders[i] = p.DNSProviders[len(p.DNSProviders)-1]
			p.DNSProviders = p.DNSProviders[:len(p.DNSProviders)-1]
		}
	}

	p.DNSTimeout, err = r.Duration("PUBLICIP_DNS_TIMEOUT")
	if err != nil {
		return err
	}

	return nil
}

var ErrFetcherNotValid = errors.New("fetcher is not valid")

func getFetchers(reader *reader.Reader) (http, dns *bool, err error) {
	// TODO change to use reader.BoolPtr with retro-compatibility
	s := reader.String("PUBLICIP_FETCHERS")
	if s == "" {
		return nil, nil, nil
	}

	http, dns = new(bool), new(bool)
	fields := strings.Split(s, ",")
	for i, field := range fields {
		switch strings.ToLower(field) {
		case "all":
			*http = true
			*dns = true
		case "http":
			*http = true
		case "dns":
			*dns = true
		default:
			return nil, nil, fmt.Errorf(
				"%w: %q at position %d of %d",
				ErrFetcherNotValid, field, i+1, len(fields))
		}
	}

	return http, dns, nil
}

func handleRetroProvider(provider string) (updatedProvider string) {
	switch provider {
	case "ipify6":
		return "ipify"
	case "noip4", "noip6", "noip8245_4", "noip8245_6":
		return "noip"
	case "cycle":
		return "all"
	default:
		return provider
	}
}
