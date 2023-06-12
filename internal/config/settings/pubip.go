package settings

import (
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/qdm12/ddns-updater/pkg/publicip/dns"
	"github.com/qdm12/ddns-updater/pkg/publicip/http"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
	"github.com/qdm12/gosettings"
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
	p.DNSTimeout = gosettings.DefaultNumber(p.DNSTimeout, defaultDNSTimeout)
}

func (p PubIP) mergeWith(other PubIP) (merged PubIP) {
	merged.HTTPEnabled = gosettings.MergeWithPointer(p.HTTPEnabled, other.HTTPEnabled)
	merged.HTTPIPProviders = gosettings.MergeWithSlice(p.HTTPIPProviders, other.HTTPIPProviders)
	merged.HTTPIPv4Providers = gosettings.MergeWithSlice(p.HTTPIPv4Providers, other.HTTPIPv4Providers)
	merged.HTTPIPv6Providers = gosettings.MergeWithSlice(p.HTTPIPv6Providers, other.HTTPIPv6Providers)
	merged.DNSEnabled = gosettings.MergeWithPointer(p.DNSEnabled, other.DNSEnabled)
	merged.DNSProviders = gosettings.MergeWithSlice(p.DNSProviders, other.DNSProviders)
	merged.DNSTimeout = gosettings.MergeWithNumber(p.DNSTimeout, other.DNSTimeout)
	return merged
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
		childNode := node.Appendf("DNS providers")
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

	providers := make([]dns.Provider, 0, len(p.HTTPIPProviders))
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

func (p *PubIP) validateDNSProviders() (err error) {
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

func (p *PubIP) validateHTTPIPProviders() (err error) {
	return validateHTTPIPProviders(p.HTTPIPProviders, ipversion.IP4or6)
}

func (p *PubIP) validateHTTPIPv4Providers() (err error) {
	return validateHTTPIPProviders(p.HTTPIPv4Providers, ipversion.IP4)
}

func (p *PubIP) validateHTTPIPv6Providers() (err error) {
	return validateHTTPIPProviders(p.HTTPIPv6Providers, ipversion.IP6)
}

var (
	ErrNoPublicIPHTTPProvider = errors.New("no public IP HTTP provider specified")
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
		// Custom URL check
		url, err := url.Parse(providerString)
		if err == nil && url != nil && url.Scheme == "https" {
			continue
		}

		_, ok := choices[providerString]
		if !ok {
			return fmt.Errorf("%w: %s", validate.ErrValueNotOneOf, providerString)
		}
	}

	return nil
}
