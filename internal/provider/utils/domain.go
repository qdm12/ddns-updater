package utils

import "strings"

func BuildDomainName(owner, domain string) string {
	if owner == "@" {
		return domain
	}
	owner = strings.ReplaceAll(owner, "*", "any")
	return owner + "." + domain
}

func BuildURLQueryHostname(owner, domain string) string {
	if owner == "@" {
		return domain
	}
	return owner + "." + domain
}
