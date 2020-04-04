package params

import (
	"fmt"
	"net/url"

	"github.com/qdm12/ddns-updater/internal/constants"
	"github.com/qdm12/ddns-updater/internal/models"
)

func settingsGeneralChecks(settings models.Settings, matchDomain func(s string) bool) error {
	switch {
	case !ipMethodIsValid(settings.IPMethod):
		return fmt.Errorf("IP method %q is not recognized", settings.IPMethod)
	case settings.IPVersion != constants.IPv4 && settings.IPVersion != constants.IPv6:
		return fmt.Errorf("IP version %q is not recognized", settings.IPVersion)
	case !matchDomain(settings.Domain):
		return fmt.Errorf("invalid domain name format")
	case len(settings.Host) == 0:
		return fmt.Errorf("host cannot be empty")
	default:
		return nil
	}
}

func settingsIPVersionChecks(ipVersion models.IPVersion, ipMethod models.IPMethod, provider models.Provider) error {
	switch ipVersion {
	case constants.IPv4:
		switch ipMethod {
		case constants.IPIFY6, constants.DDNSS6:
			return fmt.Errorf("IP method %s is only for IPv6 addresses", ipMethod)
		}
	case constants.IPv6:
		switch ipMethod {
		case constants.IPIFY, constants.DDNSS4:
			return fmt.Errorf("IP method %s is only for IPv4 addresses", ipMethod)
		}
		switch provider {
		case constants.GODADDY, constants.CLOUDFLARE, constants.DNSPOD, constants.DREAMHOST, constants.DUCKDNS, constants.NOIP:
			return fmt.Errorf("IPv6 support for %s is not supported yet", provider)
		}
	}
	return nil
}

func settingsIPMethodChecks(ipMethod models.IPMethod, provider models.Provider) error {
	if ipMethod == constants.PROVIDER {
		switch provider {
		case constants.GODADDY, constants.DREAMHOST, constants.CLOUDFLARE, constants.DNSPOD, constants.DDNSSDE:
			return fmt.Errorf("unsupported IP update method %q", ipMethod)
		}
	}
	return nil
}

func settingsNamecheapChecks(password string) error {
	if !constants.MatchNamecheapPassword(password) {
		return fmt.Errorf("invalid password format")
	}
	return nil
}

func settingsGoDaddyChecks(key, secret string) error {
	switch {
	case !constants.MatchGodaddyKey(key):
		return fmt.Errorf("invalid key format")
	case !constants.MatchGodaddySecret(secret):
		return fmt.Errorf("invalid secret format")
	}
	return nil
}

func settingsDuckDNSChecks(token, host string) error {
	switch {
	case !constants.MatchDuckDNSToken(token):
		return fmt.Errorf("invalid token format")
	case host != "@":
		return fmt.Errorf(`host can only be "@"`)
	}
	return nil
}

func settingsDreamhostChecks(key, host string) error {
	switch {
	case !constants.MatchDreamhostKey(key):
		return fmt.Errorf("invalid key format")
	case host != "@":
		return fmt.Errorf(`host can only be "@"`)
	}
	return nil
}

func settingsCloudflareChecks(key, email, userServiceKey, token, zoneIdentifier, identifier string, ttl uint, matchEmail func(s string) bool) error {
	switch {
	case len(key) > 0: // email and key must be provided
		switch {
		case !constants.MatchCloudflareKey(key):
			return fmt.Errorf("invalid key format")
		case !matchEmail(email):
			return fmt.Errorf("invalid email format")
		}
	case len(userServiceKey) > 0: // only user service key
		if !constants.MatchCloudflareKey(key) {
			return fmt.Errorf("invalid user service key format")
		}
	default: // API token only
		if !constants.MatchCloudflareToken(token) {
			return fmt.Errorf("invalid API token key format")
		}
	}
	switch {
	case len(zoneIdentifier) == 0:
		return fmt.Errorf("zone identifier cannot be empty")
	case len(identifier) == 0:
		return fmt.Errorf("identifier cannot be empty")
	case ttl == 0:
		return fmt.Errorf("TTL cannot be left to 0")
	}
	return nil
}

func settingsNoIPChecks(username, password, host string) error {
	switch {
	case len(username) == 0:
		return fmt.Errorf("username cannot be empty")
	case len(username) > 50:
		return fmt.Errorf("username cannot be longer than 50 characters")
	case len(password) == 0:
		return fmt.Errorf("password cannot be empty")
	case host == "*":
		return fmt.Errorf(`host cannot be "*"`)
	}
	return nil
}

func settingsDNSPodChecks(token string) error {
	if len(token) == 0 {
		return fmt.Errorf("token cannot be empty")
	}
	return nil
}

func settingsInfomaniakChecks(username, password, host string) error {
	switch {
	case len(username) == 0:
		return fmt.Errorf("username cannot be empty")
	case len(password) == 0:
		return fmt.Errorf("password cannot be empty")
	case host == "*":
		return fmt.Errorf(`host cannot be "*"`)
	}
	return nil
}

func settingsDdnssdeChecks(username, password, host string) error {
	switch {
	case len(username) == 0:
		return fmt.Errorf("username cannot be empty")
	case len(password) == 0:
		return fmt.Errorf("password cannot be empty")
	case host == "*":
		return fmt.Errorf(`host cannot be "*"`)
	}
	return nil
}

func (r *reader) isConsistent(settings models.Settings) error {
	if err := settingsGeneralChecks(settings, r.verifier.MatchDomain); err != nil {
		return err
	}
	if err := settingsIPVersionChecks(settings.IPVersion, settings.IPMethod, settings.Provider); err != nil {
		return err
	}
	if err := settingsIPMethodChecks(settings.IPMethod, settings.Provider); err != nil {
		return err
	}

	// Checks for each DNS provider
	switch settings.Provider {
	case constants.NAMECHEAP:
		if err := settingsNamecheapChecks(settings.Password); err != nil {
			return err
		}
	case constants.GODADDY:
		if err := settingsGoDaddyChecks(settings.Key, settings.Secret); err != nil {
			return err
		}
	case constants.DUCKDNS:
		if err := settingsDuckDNSChecks(settings.Token, settings.Host); err != nil {
			return err
		}
	case constants.DREAMHOST:
		if err := settingsDreamhostChecks(settings.Key, settings.Host); err != nil {
			return err
		}
	case constants.CLOUDFLARE:
		if err := settingsCloudflareChecks(settings.Key, settings.Email, settings.UserServiceKey, settings.Token, settings.ZoneIdentifier, settings.Identifier, settings.TTL, r.verifier.MatchEmail); err != nil {
			return err
		}
	case constants.NOIP:
		if err := settingsNoIPChecks(settings.Username, settings.Password, settings.Host); err != nil {
			return err
		}
	case constants.DNSPOD:
		if err := settingsDNSPodChecks(settings.Password); err != nil {
			return err
		}
	case constants.INFOMANIAK:
		if err := settingsInfomaniakChecks(settings.Username, settings.Password, settings.Host); err != nil {
			return err
		}
	case constants.DDNSSDE:
		if err := settingsDdnssdeChecks(settings.Username, settings.Password, settings.Host); err != nil {
			return err
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
