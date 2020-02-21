package params

import (
	"fmt"

	"github.com/qdm12/ddns-updater/internal/constants"
	"github.com/qdm12/ddns-updater/internal/models"
)

func (p *params) isConsistent(settings models.Settings) error {
	switch {
	case !ipMethodIsValid(settings.IPMethod, constants.IPMethodChoices()):
		return fmt.Errorf("IP method %q is not recognized", settings.IPMethod)
	case !p.verifier.MatchDomain(settings.Domain):
		return fmt.Errorf("invalid domain name format")
	case len(settings.Host) == 0:
		return fmt.Errorf("host cannot be empty")
	}
	switch settings.Provider {
	case constants.PROVIDERNAMECHEAP:
		if !constants.MatchNamecheapPassword(settings.Password) {
			return fmt.Errorf("invalid password format")
		}
	case constants.PROVIDERGODADDY:
		switch {
		case !constants.MatchGodaddyKey(settings.Key):
			return fmt.Errorf("invalid key format")
		case !constants.MatchGodaddySecret(settings.Secret):
			return fmt.Errorf("invalid secret format")
		case settings.IPMethod == constants.IPMETHODPROVIDER:
			return fmt.Errorf("unsupported IP update method %q", settings.IPMethod)
		}
	case constants.PROVIDERDUCKDNS:
		switch {
		case !constants.MatchDuckDNSToken(settings.Token):
			return fmt.Errorf("invalid token format")
		case settings.Host != "@":
			return fmt.Errorf(`host can only be "@"`)
		}
	case constants.PROVIDERDREAMHOST:
		switch {
		case !constants.MatchDreamhostKey(settings.Key):
			return fmt.Errorf("invalid key format")
		case settings.Host != "@":
			return fmt.Errorf(`host can only be "@"`)
		case settings.IPMethod == constants.IPMETHODPROVIDER:
			return fmt.Errorf("unsupported IP update method %q", settings.IPMethod)
		}
	case constants.PROVIDERCLOUDFLARE:
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
		case settings.IPMethod == constants.IPMETHODPROVIDER:
			return fmt.Errorf("unsupported IP update method %q", settings.IPMethod)
		case settings.Ttl == 0:
			return fmt.Errorf("TTL cannot be left to 0")
		}
	case constants.PROVIDERNOIP:
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
	case constants.PROVIDERDNSPOD:
		switch {
		case len(settings.Token) == 0:
			return fmt.Errorf("token cannot be empty")
		case settings.IPMethod == constants.IPMETHODPROVIDER:
			return fmt.Errorf("unsupported IP update method")
		}
	default:
		return fmt.Errorf("provider %q is not supported", settings.Provider)
	}
	return nil
}

func ipMethodIsValid(ipMethod models.IPMethod, possibilities []models.IPMethod) bool {
	for i := range possibilities {
		if ipMethod == possibilities[i] {
			return true
		}
	}
	return false
}
