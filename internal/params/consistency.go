package params

import (
	"fmt"
	"net/url"

	"github.com/qdm12/ddns-updater/internal/constants"
	"github.com/qdm12/ddns-updater/internal/models"
)

func (p *params) isConsistent(settings models.Settings) error {
	// General validity checks
	switch {
	case !ipMethodIsValid(settings.IPMethod):
		return fmt.Errorf("IP method %q is not recognized", settings.IPMethod)
	case settings.IPVersion != constants.IPv4 && settings.IPVersion != constants.IPv6:
		return fmt.Errorf("IP version %q is not recognized", settings.IPVersion)
	case !p.verifier.MatchDomain(settings.Domain):
		return fmt.Errorf("invalid domain name format")
	case len(settings.Host) == 0:
		return fmt.Errorf("host cannot be empty")
	}

	// Checks for each IP versions
	switch settings.IPVersion {
	case constants.IPv4:
		switch settings.IPMethod {
		case constants.IPIFY6, constants.DDNSS6:
			return fmt.Errorf("IP method %s is only for IPv6 addresses", settings.IPMethod)
		}
	case constants.IPv6:
		switch settings.IPMethod {
		case constants.IPIFY, constants.DDNSS4:
			return fmt.Errorf("IP method %s is only for IPv4 addresses", settings.IPMethod)
		}
		switch settings.Provider {
		case constants.GODADDY, constants.CLOUDFLARE, constants.DNSPOD, constants.DREAMHOST, constants.DUCKDNS, constants.NOIP:
			return fmt.Errorf("IPv6 support for %s is not supported yet", settings.Provider)
		}
	}

	// Check provider ipmethod is available
	if settings.IPMethod == constants.PROVIDER {
		switch settings.Provider {
		case constants.GODADDY, constants.DREAMHOST, constants.CLOUDFLARE, constants.DNSPOD, constants.DDNSSDE:
			return fmt.Errorf("unsupported IP update method %q", settings.IPMethod)
		}
	}

	// Checks for each DNS provider
	switch settings.Provider {
	case constants.NAMECHEAP:
		if !constants.MatchNamecheapPassword(settings.Password) {
			return fmt.Errorf("invalid password format")
		}
	case constants.GODADDY:
		switch {
		case !constants.MatchGodaddyKey(settings.Key):
			return fmt.Errorf("invalid key format")
		case !constants.MatchGodaddySecret(settings.Secret):
			return fmt.Errorf("invalid secret format")
		}
	case constants.DUCKDNS:
		switch {
		case !constants.MatchDuckDNSToken(settings.Token):
			return fmt.Errorf("invalid token format")
		case settings.Host != "@":
			return fmt.Errorf(`host can only be "@"`)
		}
	case constants.DREAMHOST:
		switch {
		case !constants.MatchDreamhostKey(settings.Key):
			return fmt.Errorf("invalid key format")
		case settings.Host != "@":
			return fmt.Errorf(`host can only be "@"`)
		}
	case constants.CLOUDFLARE:
		switch {
		case len(settings.Key) > 0: // email and key must be provided
			switch {
			case !constants.MatchCloudflareKey(settings.Key):
				return fmt.Errorf("invalid key format")
			case !p.verifier.MatchEmail(settings.Email):
				return fmt.Errorf("invalid email format")
			}
		case len(settings.UserServiceKey) > 0: // only user service key
			if !constants.MatchCloudflareKey(settings.Key) {
				return fmt.Errorf("invalid user service key format")
			}
		default: // API token only
			if !constants.MatchCloudflareToken(settings.Token) {
				return fmt.Errorf("invalid API token key format")
			}
		}
		switch {
		case len(settings.ZoneIdentifier) == 0:
			return fmt.Errorf("zone identifier cannot be empty")
		case len(settings.Identifier) == 0:
			return fmt.Errorf("identifier cannot be empty")
		case settings.Ttl == 0:
			return fmt.Errorf("TTL cannot be left to 0")
		}
	case constants.NOIP:
		switch {
		case len(settings.Username) == 0:
			return fmt.Errorf("username cannot be empty")
		case len(settings.Username) > 50:
			return fmt.Errorf("username cannot be longer than 50 characters")
		case len(settings.Password) == 0:
			return fmt.Errorf("password cannot be empty")
		case settings.Host == "*":
			return fmt.Errorf(`host cannot be "*"`)
		}
	case constants.DNSPOD:
		switch {
		case len(settings.Token) == 0:
			return fmt.Errorf("token cannot be empty")
		}
	case constants.INFOMANIAK:
		switch {
		case len(settings.Username) == 0:
			return fmt.Errorf("username cannot be empty")
		case len(settings.Password) == 0:
			return fmt.Errorf("password cannot be empty")
		case settings.Host == "*":
			return fmt.Errorf(`host cannot be "*"`)
		}
	case constants.DDNSSDE:
		switch {
		case len(settings.Username) == 0:
			return fmt.Errorf("username cannot be empty")
		case len(settings.Password) == 0:
			return fmt.Errorf("password cannot be empty")
		case settings.Host == "*":
			return fmt.Errorf(`host cannot be "*"`)
		}
	default:
		return fmt.Errorf("provider %q is not supported", settings.Provider)
	}
	return nil
}

func ipMethodIsValid(ipMethod models.IPMethod) bool {
	for _, possibility := range constants.IPMethodChoices() {
		if ipMethod == possibility {
			return true
		}
	}
	url, err := url.Parse(string(ipMethod))
	if err != nil || url == nil || url.Scheme != "https" {
		return false
	}
	return true
}
