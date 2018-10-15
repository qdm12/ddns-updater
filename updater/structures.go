package main

import "time"

type updateSettings struct {
	domain   string
	host     string
	provider string
	ipmethod string
	password string
}

func (u *updateSettings) String() (s string) {
	s = u.domain + "|" + u.host + "|" + u.provider + "|" + u.ipmethod + "|"
	for i := range u.password {
		if i < 3 || i > len(u.password)-4 {
			s += string(u.password[i])
			continue
		} else if i < 8 {
			s += "*"
		}
	}
	return s
}

func (u *updateSettings) buildDomainName() string {
	if u.host == "@" {
		return u.domain
	} else if u.host == "*" {
		return u.domain // TODO random subdomain
	} else {
		return u.host + "." + u.domain
	}
}

func (u *updateSettings) htmlDomain() string {
	return "<a href=\"http://" + u.buildDomainName() + "\">" + u.domain + "</a>"
}

func (u *updateSettings) htmlProvider() string {
	switch u.provider {
	case "namecheap":
		return "<a href=\"https://namecheap.com\">Namecheap</a>"
	case "godaddy":
		return "<a href=\"https://godaddy.com\">GoDaddy</a>"
	case "duckdns":
		return "<a href=\"https://duckdns.org\">DuckDNS</a>"
	default:
		return u.provider
	}
}

// TODO map to icons
func (u *updateSettings) htmlIpmethod() string {
	switch u.ipmethod {
	case "provider":
		return u.htmlProvider()
	case "duckduckgo":
		return "<a href=\"https://duckduckgo.com/?q=ip\">DuckDuckGo</a>"
	case "opendns":
		return "<a href=\"https://diagnostic.opendns.com/myip\">OpenDNS</a>"
	default:
		return u.ipmethod
	}
}

type statusCode uint8

func (c *statusCode) String() (s string) {
	switch *c {
	case SUCCESS:
		return "Success"
	case FAIL:
		return "Failure"
	case UPTODATE:
		return "Up to date"
	case UPDATING:
		return "Already updating..."
	default:
		return "Unknown status code!"
	}
}

func (c *statusCode) html() (s string) {
	switch *c {
	case SUCCESS:
		return `<font color="green">Success</font>`
	case FAIL:
		return `<font color="red">Failure</font>`
	case UPTODATE:
		return `<font color="#00CC66">Up to date</font>`
	case UPDATING:
		return `<font color="orange">Already updating...</font>`
	default:
		return `<font color="red">Unknown status code!</font>`
	}
}

const (
	FAIL statusCode = iota
	SUCCESS
	UPTODATE
	UPDATING
)

type updateStatus struct {
	code    statusCode
	message string
	time    time.Time
}

func (u *updateStatus) String() (s string) {
	s += u.code.String()
	if u.message != "" {
		s += " (" + u.message + ")"
	}
	s += " at " + u.time.Format("2006-01-02 15:04:05 MST")
	return s
}

func (u *updateStatus) html() (s string) {
	s += u.code.html()
	if u.message != "" {
		s += " (" + u.message + ")"
	}
	s += ", " + time.Since(u.time).Round(time.Second).String() + " ago"
	return s
}

type updateExtras struct {
	ips      []string // current and previous ips
	tSuccess time.Time
}

func (u *updateExtras) String() (s string) {
	if len(u.ips) > 0 {
		s += "Last success update: " + u.tSuccess.Format("2006-01-02 15:04:05 MST") + "; Current & previous IPs: "
		for i := range u.ips {
			s += u.ips[i]
			if i != len(u.ips)-1 {
				s += ","
			}
		}
	}
	return s
}

type updateType struct { // internal
	settings updateSettings // fixed
	status   updateStatus   // changes for each update
	extras   updateExtras   // past information
}

func (u *updateType) String() string {
	return u.settings.String() + ": " + u.status.String() + "; " + u.extras.String()
}
