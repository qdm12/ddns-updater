package constants

import "regexp"

const (
	goDaddyKey               string = `[A-Za-z0-9]{10,14}\_[A-Za-z0-9]{22}`
	godaddySecret            string = `[A-Za-z0-9]{22}`
	RegexDuckDNSToken        string = `[a-f0-9]{8}\-[a-f0-9]{4}\-[a-f0-9]{4}\-[a-f0-9]{4}\-[a-f0-9]{12}`
	namecheapPassword        string = `[a-f0-9]{32}`
	dreamhostKey             string = `[a-zA-Z0-9]{16}`
	cloudflareKey            string = `[a-zA-Z0-9]+`
	cloudflareUserServiceKey string = `v1\.0.+`
	cloudflareToken          string = `[a-zA-Z0-9_]{40}`
)

func MatchGodaddyKey(s string) bool {
	return regexp.MustCompile("^" + goDaddyKey + "$").MatchString(s)
}

func MatchGodaddySecret(s string) bool {
	return regexp.MustCompile("^" + godaddySecret + "$").MatchString(s)
}

func MatchDuckDNSToken(s string) bool {
	return regexp.MustCompile("^" + RegexDuckDNSToken + "$").MatchString(s)
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
