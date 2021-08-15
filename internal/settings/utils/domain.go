package utils

import "strings"

func BuildDomainName(host, domain string) string {
	if host == "@" {
		return domain
	}
	host = strings.ReplaceAll(host, "*", "any")
	return host + "." + domain
}

func BuildURLQueryHostname(host, domain string) string {
	if host == "@" {
		return domain
	}
	return host + "." + domain
}
