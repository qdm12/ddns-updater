package models

// SettingsType contains the elements to update the DNS record
type SettingsType struct {
	Domain   string
	Host     string
	Provider string
	IPmethod string
	Password string
}

func (settings *SettingsType) string() (s string) {
	s = settings.Domain + "|" + settings.Host + "|" + settings.Provider + "|" + settings.IPmethod + "|"
	for i := range settings.Password {
		if i < 3 || i > len(settings.Password)-4 {
			s += string(settings.Password[i])
			continue
		} else if i < 8 {
			s += "*"
		}
	}
	return s
}

// BuildDomainName builds the domain name from the domain and the host of the settings
func (settings *SettingsType) BuildDomainName() string {
	if settings.Host == "@" {
		return settings.Domain
	} else if settings.Host == "*" {
		return settings.Domain // TODO random subdomain
	} else {
		return settings.Host + "." + settings.Domain
	}
}

func (settings *SettingsType) getHTMLDomain() string {
	return "<a href=\"http://" + settings.BuildDomainName() + "\">" + settings.Domain + "</a>"
}

func (settings *SettingsType) getHTMLProvider() string {
	switch settings.Provider {
	case "namecheap":
		return "<a href=\"https://namecheap.com\">Namecheap</a>"
	case "godaddy":
		return "<a href=\"https://godaddy.com\">GoDaddy</a>"
	case "duckdns":
		return "<a href=\"https://duckdns.org\">DuckDNS</a>"
	default:
		return settings.Provider
	}
}

// TODO map to icons
func (settings *SettingsType) getHTMLIPMethod() string {
	switch settings.IPmethod {
	case "provider":
		return settings.getHTMLProvider()
	case "duckduckgo":
		return "<a href=\"https://duckduckgo.com/?q=ip\">DuckDuckGo</a>"
	case "opendns":
		return "<a href=\"https://diagnostic.opendns.com/myip\">OpenDNS</a>"
	default:
		return settings.IPmethod
	}
}
