package constants

import "regexp"

const (
	goDaddyKey               = `[A-Za-z0-9]{10,14}\_[A-Za-z0-9]{22}`
	godaddySecret            = `[A-Za-z0-9]{22}`                                                  // #nosec
	duckDNSToken             = `[a-f0-9]{8}\-[a-f0-9]{4}\-[a-f0-9]{4}\-[a-f0-9]{4}\-[a-f0-9]{12}` // #nosec
	namecheapPassword        = `[a-f0-9]{32}`                                                     // #nosec
	dreamhostKey             = `[a-zA-Z0-9]{16}`
	cloudflareKey            = `[a-zA-Z0-9]+`
	cloudflareUserServiceKey = `v1\.0.+`
	cloudflareToken          = `[a-zA-Z0-9_-]{40}` // #nosec
)

func MatchGodaddyKey(s string) bool {
	return regexp.MustCompile("^" + goDaddyKey + "$").MatchString(s)
}

func MatchGodaddySecret(s string) bool {
	return regexp.MustCompile("^" + godaddySecret + "$").MatchString(s)
}

func MatchDuckDNSToken(s string) bool {
	return regexp.MustCompile("^" + duckDNSToken + "$").MatchString(s)
}

func MatchNamecheapPassword(s string) bool {
	return regexp.MustCompile("^" + namecheapPassword + "$").MatchString(s)
}

func MatchDreamhostKey(s string) bool {
	return regexp.MustCompile("^" + dreamhostKey + "$").MatchString(s)
}

func MatchCloudflareKey(s string) bool {
	return regexp.MustCompile("^" + cloudflareKey + "$").MatchString(s)
}

func MatchCloudflareUserServiceKey(s string) bool {
	return regexp.MustCompile("^" + cloudflareUserServiceKey + "$").MatchString(s)
}

func MatchCloudflareToken(s string) bool {
	return regexp.MustCompile("^" + cloudflareToken + "$").MatchString(s)
}
