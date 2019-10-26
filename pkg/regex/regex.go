package regex

import (
	"regexp"
)

const (
	regexIP                       = `(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)(\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)){3}`
	regexEmail                    = `[a-zA-Z0-9-_.]+@[a-zA-Z0-9-_.]+\.[a-zA-Z][a-zA-Z]+`
	regexDomain                   = `(([a-zA-Z]{1})|([a-zA-Z]{1}[a-zA-Z]{1})|([a-zA-Z]{1}[0-9]{1})|([0-9]{1}[a-zA-Z]{1})|([a-zA-Z0-9][a-zA-Z0-9-_]{1,61}[a-zA-Z0-9]))\.([a-zA-Z]{2,6}|[a-zA-Z0-9-]{2,30}\.[a-zA-Z]{2,3})`
	regexGoDaddyKey               = `[A-Za-z0-9]{10,14}\_[A-Za-z0-9]{22}`
	regexGodaddySecret            = `[A-Za-z0-9]{22}`
	regexDuckDNSToken             = `[a-f0-9]{8}\-[a-f0-9]{4}\-[a-f0-9]{4}\-[a-f0-9]{4}\-[a-f0-9]{12}`
	regexNamecheapPassword        = `[a-f0-9]{32}`
	regexDreamhostKey             = `[a-zA-Z0-9]{16}`
	regexCloudflareKey            = `[a-zA-Z0-9]+`
	regexCloudflareUserServiceKey = `v1\.0.+`
)

// Search functions
var (
	SearchIP = buildSearchFn(regexIP)
)

func buildSearchFn(regex string) func(s string) []string {
	return func(s string) []string {
		return regexp.MustCompile(regex).FindAllString(s, -1)
	}
}

// Regex MatchString functions
var (
	MatchEmail                    = regexp.MustCompile("^" + regexEmail + "$").MatchString
	MatchDomain                   = regexp.MustCompile("^" + regexDomain + "$").MatchString
	MatchGodaddyKey               = regexp.MustCompile("^" + regexGoDaddyKey + "$").MatchString
	MatchGodaddySecret            = regexp.MustCompile("^" + regexGodaddySecret + "$").MatchString
	MatchDuckDNSToken             = regexp.MustCompile("^" + regexDuckDNSToken + "$").MatchString
	MatchNamecheapPassword        = regexp.MustCompile("^" + regexNamecheapPassword + "$").MatchString
	MatchDreamhostKey             = regexp.MustCompile("^" + regexDreamhostKey + "$").MatchString
	MatchCloudflareKey            = regexp.MustCompile("^" + regexCloudflareKey + "$").MatchString
	MatchCloudflareUserServiceKey = regexp.MustCompile("^" + regexCloudflareUserServiceKey + "$").MatchString
)
