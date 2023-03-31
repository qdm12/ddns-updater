package regex

import "regexp"

type Matcher struct {
	goDaddyKey, duckDNSToken, namecheapPassword, dreamhostKey, cloudflareKey,
	cloudflareUserServiceKey, dnsOMaticUsername, dnsOMaticPassword, gandiKey *regexp.Regexp
}

var (
	gandiKey                 = regexp.MustCompile(`^[A-Za-z0-9]{24}$`)
	goDaddyKey               = regexp.MustCompile(`^[A-Za-z0-9]{8,14}\_[A-Za-z0-9]{21,22}$`)
	duckDNSToken             = regexp.MustCompile(`^[a-f0-9]{8}\-[a-f0-9]{4}\-[a-f0-9]{4}\-[a-f0-9]{4}\-[a-f0-9]{12}$`)
	namecheapPassword        = regexp.MustCompile(`^[a-f0-9]{32}$`)
	dreamhostKey             = regexp.MustCompile(`^[a-zA-Z0-9]{16}$`)
	cloudflareKey            = regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	cloudflareUserServiceKey = regexp.MustCompile(`^v1\.0.+$`)
	dnsOMaticUsername        = regexp.MustCompile(`^[a-zA-Z0-9@._-]{3,25}$`)
	dnsOMaticPassword        = regexp.MustCompile(`^[a-zA-Z0-9 !@#$€%&+*._-]{5,19}$`)
)

func NewMatcher() *Matcher {
	return &Matcher{
		gandiKey:                 gandiKey,
		goDaddyKey:               goDaddyKey,
		duckDNSToken:             duckDNSToken,
		namecheapPassword:        namecheapPassword,
		dreamhostKey:             dreamhostKey,
		cloudflareKey:            cloudflareKey,
		cloudflareUserServiceKey: cloudflareUserServiceKey,
		dnsOMaticUsername:        dnsOMaticUsername,
		dnsOMaticPassword:        dnsOMaticPassword,
	}
}

func (m *Matcher) GandiKey(s string) bool          { return m.gandiKey.MatchString(s) }
func (m *Matcher) GodaddyKey(s string) bool        { return m.goDaddyKey.MatchString(s) }
func (m *Matcher) DuckDNSToken(s string) bool      { return m.duckDNSToken.MatchString(s) }
func (m *Matcher) NamecheapPassword(s string) bool { return m.namecheapPassword.MatchString(s) }
func (m *Matcher) DreamhostKey(s string) bool      { return m.dreamhostKey.MatchString(s) }
func (m *Matcher) CloudflareKey(s string) bool     { return m.cloudflareKey.MatchString(s) }
func (m *Matcher) CloudflareUserServiceKey(s string) bool {
	return m.cloudflareUserServiceKey.MatchString(s)
}
func (m *Matcher) DNSOMaticUsername(s string) bool { return m.dnsOMaticUsername.MatchString(s) }
func (m *Matcher) DNSOMaticPassword(s string) bool { return m.dnsOMaticPassword.MatchString(s) }
