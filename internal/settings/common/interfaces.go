package common

type Matcher interface {
	GandiKey(s string) bool
	GodaddyKey(s string) bool
	DuckDNSToken(s string) bool
	NamecheapPassword(s string) bool
	DreamhostKey(s string) bool
	CloudflareKey(s string) bool
	CloudflareUserServiceKey(s string) bool
	DNSOMaticUsername(s string) bool
}
