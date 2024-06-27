package utils

import (
	"strings"

	"github.com/chmike/domain"
)

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

func CheckDomain(domainString string) (err error) {
	return domain.Check(domainString)
}
