package regex

import "regexp"

type Matcher interface {
	GandiKey(s string) bool
	GodaddyKey(s string) bool
	DuckDNSToken(s string) bool
	NamecheapPassword(s string) bool
	DreamhostKey(s string) bool
	CloudflareKey(s string) bool
	CloudflareUserServiceKey(s string) bool
	DNSOMaticUsername(s string) bool
	DNSOMaticPassword(s string) bool
}

type matcher struct {
	goDaddyKey, duckDNSToken, namecheapPassword, dreamhostKey, cloudflareKey,
	cloudflareUserServiceKey, dnsOMaticUsername, dnsOMaticPassword, gandiKey *regexp.Regexp
}

func NewMatcher() (m Matcher, err error) {
	matcher := &matcher{}
	matcher.gandiKey, err = regexp.Compile(`^[A-Za-z0-9]{24}$`)
	if err != nil {
		return nil, err
	}
	matcher.goDaddyKey, err = regexp.Compile(`^[A-Za-z0-9]{8,14}\_[A-Za-z0-9]{21,22}$`)
	if err != nil {
		return nil, err
	}
	matcher.duckDNSToken, err = regexp.Compile(`^[a-f0-9]{8}\-[a-f0-9]{4}\-[a-f0-9]{4}\-[a-f0-9]{4}\-[a-f0-9]{12}$`)
	if err != nil {
		return nil, err
	}
	matcher.namecheapPassword, err = regexp.Compile(`^[a-f0-9]{32}$`)
	if err != nil {
		return nil, err
	}
	matcher.dreamhostKey, err = regexp.Compile(`^[a-zA-Z0-9]{16}$`)
	if err != nil {
		return nil, err
	}
	matcher.cloudflareKey, err = regexp.Compile(`^[a-zA-Z0-9]+$`)
	if err != nil {
		return nil, err
	}
	matcher.cloudflareUserServiceKey, err = regexp.Compile(`^v1\.0.+$`)
	if err != nil {
		return nil, err
	}
	matcher.dnsOMaticUsername, err = regexp.Compile(`^[a-zA-Z0-9._-]{3,25}$`)
	if err != nil {
		return nil, err
	}
	matcher.dnsOMaticPassword, err = regexp.Compile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{5,19}$`)
	if err != nil {
		return nil, err
	}
	return matcher, nil
}

func (m *matcher) GandiKey(s string) bool          { return m.gandiKey.MatchString(s) }
func (m *matcher) GodaddyKey(s string) bool        { return m.goDaddyKey.MatchString(s) }
func (m *matcher) DuckDNSToken(s string) bool      { return m.duckDNSToken.MatchString(s) }
func (m *matcher) NamecheapPassword(s string) bool { return m.namecheapPassword.MatchString(s) }
func (m *matcher) DreamhostKey(s string) bool      { return m.dreamhostKey.MatchString(s) }
func (m *matcher) CloudflareKey(s string) bool     { return m.cloudflareKey.MatchString(s) }
func (m *matcher) CloudflareUserServiceKey(s string) bool {
	return m.cloudflareUserServiceKey.MatchString(s)
}
func (m *matcher) DNSOMaticUsername(s string) bool { return m.dnsOMaticUsername.MatchString(s) }
func (m *matcher) DNSOMaticPassword(s string) bool { return m.dnsOMaticPassword.MatchString(s) }
