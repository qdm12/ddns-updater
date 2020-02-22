package constants

import "regexp"

const (
	RegexGoDaddyKey               string = `[A-Za-z0-9]{10,14}\_[A-Za-z0-9]{22}`
	RegexGodaddySecret            string = `[A-Za-z0-9]{22}`
	RegexDuckDNSToken             string = `[a-f0-9]{8}\-[a-f0-9]{4}\-[a-f0-9]{4}\-[a-f0-9]{4}\-[a-f0-9]{12}`
	RegexNamecheapPassword        string = `[a-f0-9]{32}`
	RegexDreamhostKey             string = `[a-zA-Z0-9]{16}`
	RegexCloudflareKey            string = `[a-zA-Z0-9]+`
	RegexCloudflareUserServiceKey string = `v1\.0.+`
	RegexCloudflareToken          string = `[a-zA-Z0-9_]{40}`
)

func MatchGodaddyKey(s string) bool {
	return regexp.MustCompile("^" + RegexGoDaddyKey + "$").MatchString(s)
}

func MatchGodaddySecret(s string) bool {
	return regexp.MustCompile("^" + RegexGodaddySecret + "$").MatchString(s)
}

func MatchDuckDNSToken(s string) bool {
	return regexp.MustCompile("^" + RegexDuckDNSToken + "$").MatchString(s)
}

func MatchNamecheapPassword(s string) bool {
	return regexp.MustCompile("^" + RegexNamecheapPassword + "$").MatchString(s)
}

func MatchDreamhostKey(s string) bool {
	return regexp.MustCompile("^" + RegexDreamhostKey + "$").MatchString(s)
}

func MatchCloudflareKey(s string) bool {
	return regexp.MustCompile("^" + RegexCloudflareKey + "$").MatchString(s)
}

func MatchCloudflareUserServiceKey(s string) bool {
	return regexp.MustCompile("^" + RegexCloudflareUserServiceKey + "$").MatchString(s)
}

func MatchCloudflareToken(s string) bool {
	return regexp.MustCompile("^" + RegexCloudflareToken + "$").MatchString(s)
}
