package models

import (
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	"github.com/qdm12/golibs/verification"
)

const (
	regexGoDaddyKey               = `[A-Za-z0-9]{10,14}\_[A-Za-z0-9]{22}`
	regexGodaddySecret            = `[A-Za-z0-9]{22}`
	regexDuckDNSToken             = `[a-f0-9]{8}\-[a-f0-9]{4}\-[a-f0-9]{4}\-[a-f0-9]{4}\-[a-f0-9]{12}`
	regexNamecheapPassword        = `[a-f0-9]{32}`
	regexDreamhostKey             = `[a-zA-Z0-9]{16}`
	regexCloudflareKey            = `[a-zA-Z0-9]+`
	regexCloudflareToken          = `[a-zA-Z0-9_]{40}`
	regexCloudflareUserServiceKey = `v1\.0.+`
)

// Regex MatchString functions
var (
	matchGodaddyKey               = regexp.MustCompile("^" + regexGoDaddyKey + "$").MatchString
	matchGodaddySecret            = regexp.MustCompile("^" + regexGodaddySecret + "$").MatchString
	matchDuckDNSToken             = regexp.MustCompile("^" + regexDuckDNSToken + "$").MatchString
	matchNamecheapPassword        = regexp.MustCompile("^" + regexNamecheapPassword + "$").MatchString
	matchDreamhostKey             = regexp.MustCompile("^" + regexDreamhostKey + "$").MatchString
	matchCloudflareKey            = regexp.MustCompile("^" + regexCloudflareKey + "$").MatchString
	matchCloudflareToken          = regexp.MustCompile("^" + regexCloudflareToken + "$").MatchString
	matchCloudflareUserServiceKey = regexp.MustCompile("^" + regexCloudflareUserServiceKey + "$").MatchString
)

// SettingsType contains the elements to update the DNS record
type SettingsType struct {
	Domain      string
	Host        string
	Provider    ProviderType
	IPmethod    IPMethodType
	Delay       time.Duration
	NoDNSLookup bool
	// Provider dependent fields
	Password       string // Namecheap and NoIP only
	Key            string // GoDaddy, Dreamhost and Cloudflare only
	Secret         string // GoDaddy only
	Token          string // Cloudflare and DuckDNS only
	Email          string // Cloudflare only
	UserServiceKey string // Cloudflare only
	ZoneIdentifier string // Cloudflare only
	Identifier     string // Cloudflare only
	Proxied        bool   // Cloudflare only
	Ttl            uint   // Cloudflare only
	Username       string // NoIP only
}

func (settings *SettingsType) String() string {
	b, _ := json.Marshal(
		struct {
			Domain   string `json:"domain"`
			Host     string `json:"host"`
			Provider string `json:"provider"`
		}{
			settings.Domain,
			settings.Host,
			settings.Provider.String(),
		},
	)
	return string(b)
}

// BuildDomainName builds the domain name from the domain and the host of the settings
func (settings *SettingsType) BuildDomainName() string {
	if settings.Host == "@" {
		return settings.Domain
	} else if settings.Host == "*" {
		return "any." + settings.Domain
	} else {
		return settings.Host + "." + settings.Domain
	}
}

func (settings *SettingsType) getHTMLDomain() string {
	return "<a href=\"http://" + settings.BuildDomainName() + "\">" + settings.Domain + "</a>"
}

func (settings *SettingsType) getHTMLProvider() string {
	switch settings.Provider {
	case PROVIDERNAMECHEAP:
		return "<a href=\"https://namecheap.com\">Namecheap</a>"
	case PROVIDERGODADDY:
		return "<a href=\"https://godaddy.com\">GoDaddy</a>"
	case PROVIDERDUCKDNS:
		return "<a href=\"https://duckdns.org\">DuckDNS</a>"
	case PROVIDERDREAMHOST:
		return "<a href=\"https://https://www.dreamhost.com/\">Dreamhost</a>"
	default:
		return settings.Provider.String()
	}
}

// TODO map to icons
func (settings *SettingsType) getHTMLIPMethod() string {
	switch settings.IPmethod {
	case IPMETHODPROVIDER:
		return settings.getHTMLProvider()
	case IPMETHODGOOGLE:
		return "<a href=\"https://google.com/search?q=ip\">Google</a>"
	case IPMETHODOPENDNS:
		return "<a href=\"https://diagnostic.opendns.com/myip\">OpenDNS</a>"
	default:
		return string(settings.IPmethod)
	}
}

// Verify verifies all the settings provided are valid
func (settings *SettingsType) Verify() error {
	if !verification.MatchDomain(settings.Domain) {
		return fmt.Errorf("invalid domain name format for settings %s", settings)
	} else if len(settings.Host) == 0 {
		return fmt.Errorf("host cannot be empty for settings %s", settings)
	}
	switch settings.Provider {
	case PROVIDERNAMECHEAP:
		if !matchNamecheapPassword(settings.Password) {
			return fmt.Errorf("invalid password format for settings %s", settings)
		}
	case PROVIDERGODADDY:
		if !matchGodaddyKey(settings.Key) {
			return fmt.Errorf("invalid key format for settings %s", settings)
		} else if !matchGodaddySecret(settings.Secret) {
			return fmt.Errorf("invalid secret format for settings %s", settings)
		} else if settings.IPmethod == IPMETHODPROVIDER {
			return fmt.Errorf("unsupported IP update method for settings %s", settings)
		}
	case PROVIDERDUCKDNS:
		if !matchDuckDNSToken(settings.Token) {
			return fmt.Errorf("invalid token format for settings %s", settings)
		} else if settings.Host != "@" {
			return fmt.Errorf("host can only be \"@\" for settings %s", settings)
		}
	case PROVIDERDREAMHOST:
		if !matchDreamhostKey(settings.Key) {
			return fmt.Errorf("invalid key format for settings %s", settings)
		} else if settings.Host != "@" {
			return fmt.Errorf("host can only be \"@\" for settings %s", settings)
		} else if settings.IPmethod == IPMETHODPROVIDER {
			return fmt.Errorf("unsupported IP update method for settings %s", settings)
		}
	case PROVIDERCLOUDFLARE:
		if settings.Key != "" { // email and key must be provided
			if !matchCloudflareKey(settings.Key) {
				return fmt.Errorf("invalid key format for settings %s", settings)
			} else if !verification.MatchEmail(settings.Email) {
				return fmt.Errorf("invalid email format for settings %s", settings)
			}
		} else if settings.UserServiceKey != "" { // only user service key
			if !matchCloudflareUserServiceKey(settings.UserServiceKey) {
				return fmt.Errorf("invalid user service key format for settings %s", settings)
			}
		} else { // otherwise use an API token
			if !matchCloudflareToken(settings.Token) {
				return fmt.Errorf("invalid API token key format for settings %s", settings)
			}
		}
		if len(settings.ZoneIdentifier) == 0 {
			return fmt.Errorf("zone identifier cannot be empty to settings %s", settings)
		} else if len(settings.Identifier) == 0 {
			return fmt.Errorf("identifier cannot be empty to settings %s", settings)
		} else if settings.IPmethod == IPMETHODPROVIDER {
			return fmt.Errorf("unsupported IP update method for settings %s", settings)
		} else if settings.Ttl == 0 {
			return fmt.Errorf("TTL cannot be left to 0 for settings %s", settings)
		}
	case PROVIDERNOIP:
		if len(settings.Username) == 0 {
			return fmt.Errorf("username cannot be empty for settings %s", settings)
		} else if len(settings.Username) > 50 {
			return fmt.Errorf("username cannot be longer than 50 characters for settings %s", settings)
		} else if len(settings.Password) == 0 {
			return fmt.Errorf("password cannot be empty for settings %s", settings)
		} else if settings.Host == "*" {
			return fmt.Errorf("host cannot be * for settings %s", settings)
		}
	case PROVIDERDNSPOD:
		if len(settings.Token) == 0 {
			return fmt.Errorf("token cannot be empty for settings %s", settings)
		} else if settings.IPmethod == IPMETHODPROVIDER {
			return fmt.Errorf("unsupported IP update method for settings %s", settings)
		}
	default:
		return fmt.Errorf("provider \"%s\" is not supported", settings.Provider)
	}
	return nil
}
