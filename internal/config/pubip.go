package config

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/qdm12/ddns-updater/pkg/publicip"
	"github.com/qdm12/ddns-updater/pkg/publicip/dns"
	"github.com/qdm12/ddns-updater/pkg/publicip/http"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
	"github.com/qdm12/golibs/params"
)

const all = "all"

type PubIP struct {
	HTTPSettings publicip.HTTPSettings
	DNSSettings  publicip.DNSSettings
}

func (p *PubIP) get(env params.Interface) (warnings []string, err error) {
	err = p.getFetchers(env)
	if err != nil {
		return nil, err
	}

	httpIPProviders, warning, err := p.getIPHTTPProviders(env)
	warnings = appendIfNotEmpty(warnings, warning)
	if err != nil {
		return warnings, err
	}
	httpIP4Providers, warning, err := p.getIPv4HTTPProviders(env)
	warnings = appendIfNotEmpty(warnings, warning)
	if err != nil {
		return warnings, err
	}
	httpIP6Providers, warning, err := p.getIPv6HTTPProviders(env)
	warnings = appendIfNotEmpty(warnings, warning)
	if err != nil {
		return warnings, err
	}
	p.HTTPSettings.Options = []http.Option{
		http.SetProvidersIP(httpIPProviders[0], httpIPProviders[1:]...),
		http.SetProvidersIP4(httpIP4Providers[0], httpIP4Providers[1:]...),
		http.SetProvidersIP6(httpIP6Providers[0], httpIP6Providers[1:]...),
	}

	dnsIPProviders, err := p.getDNSProviders(env)
	if err != nil {
		return warnings, err
	}

	dnsTimeout, err := env.Duration("PUBLICIP_DNS_TIMEOUT", params.Default("3s"))
	if err != nil {
		return warnings, err
	}

	p.DNSSettings.Options = []dns.Option{
		dns.SetTimeout(dnsTimeout),
		dns.SetProviders(dnsIPProviders[0], dnsIPProviders[1:]...),
	}

	return warnings, nil
}

var ErrInvalidFetcher = errors.New("invalid fetcher specified")

func (p *PubIP) getFetchers(env params.Interface) (err error) {
	s, err := env.Get("PUBLICIP_FETCHERS", params.Default(all))
	if err != nil {
		return fmt.Errorf("%w: for environment variable PUBLICIP_FETCHERS", err)
	}

	fields := strings.Split(s, ",")
	for i, field := range fields {
		switch strings.ToLower(field) {
		case all:
			p.HTTPSettings.Enabled = true
			p.DNSSettings.Enabled = true
		case "http":
			p.HTTPSettings.Enabled = true
		case "dns":
			p.DNSSettings.Enabled = true
		default:
			err = fmt.Errorf(
				"%w: %q at position %d of %d",
				ErrInvalidFetcher, field, i+1, len(fields))
		}
	}

	return err
}

// getDNSProviders obtains the DNS providers to obtain your public IPv4 and/or IPv6 address.
func (p *PubIP) getDNSProviders(env params.Interface) (providers []dns.Provider, err error) {
	s, err := env.Get("PUBLICIP_DNS_PROVIDERS", params.Default(all))
	if err != nil {
		return nil, fmt.Errorf("%w: for environment variable PUBLICIP_DNS_PROVIDERS", err)
	}

	availableProviders := dns.ListProviders()

	fields := strings.Split(s, ",")
	providers = make([]dns.Provider, len(fields))
	for i, field := range fields {
		if field == all {
			return availableProviders, nil
		}

		providers[i] = dns.Provider(field)
		err = dns.ValidateProvider(providers[i])
		if err != nil {
			return nil, err
		}
	}

	return providers, nil
}

// getHTTPProviders obtains the HTTP providers to obtain your public IPv4 or IPv6 address.
func (p *PubIP) getIPHTTPProviders(env params.Interface) (
	providers []http.Provider, warning string, err error) {
	return httpIPMethod(env, "PUBLICIP_HTTP_PROVIDERS", "IP_METHOD", ipversion.IP4or6)
}

// getIPv4HTTPProviders obtains the HTTP providers to obtain your public IPv4 address.
func (p *PubIP) getIPv4HTTPProviders(env params.Interface) (
	providers []http.Provider, warning string, err error) {
	return httpIPMethod(env, "PUBLICIPV4_HTTP_PROVIDERS", "IPV4_METHOD", ipversion.IP4)
}

// getIPv6HTTPProviders obtains the HTTP providers to obtain your public IPv6 address.
func (p *PubIP) getIPv6HTTPProviders(env params.Interface) (
	providers []http.Provider, warning string, err error) {
	return httpIPMethod(env, "PUBLICIPV6_HTTP_PROVIDERS", "IPV6_METHOD", ipversion.IP6)
}

var (
	ErrInvalidPublicIPHTTPProvider = errors.New("invalid public IP HTTP provider")
)

func httpIPMethod(env params.Interface, envKey, retroKey string, version ipversion.IPVersion) (
	providers []http.Provider, warning string, err error) {
	retroKeyOption := params.RetroKeys([]string{retroKey}, func(oldKey, newKey string) {
		warning = "You are using an old environment variable " + oldKey +
			" please change it to " + newKey
	})
	s, err := env.Get(envKey, params.Default("cycle"), retroKeyOption)
	if err != nil {
		return nil, warning, fmt.Errorf("%w: for environment variable %s", err, envKey)
	}

	availableProviders := http.ListProvidersForVersion(version)
	choices := make(map[http.Provider]struct{}, len(availableProviders))
	for _, provider := range availableProviders {
		choices[provider] = struct{}{}
	}

	fields := strings.Split(s, ",")

	for _, field := range fields {
		// Retro-compatibility.
		switch field {
		case "ipify6":
			field = "ipify"
		case "noip4", "noip6", "noip8245_4", "noip8245_6":
			field = "noip"
		case "cycle":
			field = all
		}

		if field == all {
			return availableProviders, warning, nil
		}

		// Custom URL check
		url, err := url.Parse(field)
		if err == nil && url != nil && url.Scheme == "https" {
			providers = append(providers, http.CustomProvider(url))
			continue
		}

		provider := http.Provider(field)
		if _, ok := choices[provider]; !ok {
			return nil, warning, fmt.Errorf("%w: %s", ErrInvalidPublicIPHTTPProvider, provider)
		}
		providers = append(providers, provider)
	}

	if len(providers) == 0 {
		return nil, warning, fmt.Errorf("%w: for IP version %s", ErrInvalidPublicIPHTTPProvider, version)
	}

	return providers, warning, nil
}
