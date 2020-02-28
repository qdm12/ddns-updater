package params

import (
	"fmt"

	"github.com/qdm12/ddns-updater/internal/constants"
	"github.com/qdm12/ddns-updater/internal/models"
)

func (p *params) isConsistent(settings models.Settings) error {
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
	switch settings.Provider {
	case constants.NAMECHEAP:
		if !constants.MatchNamecheapPassword(settings.Password) {
			return fmt.Errorf("invalid password format")
		} else if settings.IPVersion == constants.IPv6 {
			return fmt.Errorf("IPv6 support for %s is not supported yet", settings.Provider)
		}
	case constants.GODADDY:
		switch {
		case !constants.MatchGodaddyKey(settings.Key):
			return fmt.Errorf("invalid key format")
		case !constants.MatchGodaddySecret(settings.Secret):
			return fmt.Errorf("invalid secret format")
		case settings.IPMethod == constants.PROVIDER:
			return fmt.Errorf("unsupported IP update method %q", settings.IPMethod)
		case settings.IPVersion == constants.IPv6:
			return fmt.Errorf("IPv6 support for %s is not supported yet", settings.Provider)
		}
	case constants.DUCKDNS:
		switch {
		case !constants.MatchDuckDNSToken(settings.Token):
			return fmt.Errorf("invalid token format")
		case settings.Host != "@":
			return fmt.Errorf(`host can only be "@"`)
		case settings.IPVersion == constants.IPv6:
			return fmt.Errorf("IPv6 support for %s is not supported yet", settings.Provider)
		}
	case constants.DREAMHOST:
		switch {
		case !constants.MatchDreamhostKey(settings.Key):
			return fmt.Errorf("invalid key format")
		case settings.Host != "@":
			return fmt.Errorf(`host can only be "@"`)
		case settings.IPMethod == constants.PROVIDER:
			return fmt.Errorf("unsupported IP update method %q", settings.IPMethod)
		case settings.IPVersion == constants.IPv6:
			return fmt.Errorf("IPv6 support for %s is not supported yet", settings.Provider)
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
		case settings.IPMethod == constants.PROVIDER:
			return fmt.Errorf("unsupported IP update method %q", settings.IPMethod)
		case settings.Ttl == 0:
			return fmt.Errorf("TTL cannot be left to 0")
		case settings.IPVersion == constants.IPv6:
			return fmt.Errorf("IPv6 support for %s is not supported yet", settings.Provider)
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
		case settings.IPVersion == constants.IPv6:
			return fmt.Errorf("IPv6 support for %s is not supported yet", settings.Provider)
		}
	case constants.DNSPOD:
		switch {
		case len(settings.Token) == 0:
			return fmt.Errorf("token cannot be empty")
		case settings.IPMethod == constants.PROVIDER:
			return fmt.Errorf("unsupported IP update method")
		case settings.IPVersion == constants.IPv6:
			return fmt.Errorf("IPv6 support for %s is not supported yet", settings.Provider)
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
	return false
}
