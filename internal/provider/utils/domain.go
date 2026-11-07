package utils

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
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

// CheckDomain returns an non-nil error if the domain name is not valid.
// https://tools.ietf.org/html/rfc1034#section-3.5
// https://tools.ietf.org/html/rfc1123#section-2.
func CheckDomain(domainString string) (err error) {
	const maxDomainLength = 255
	switch {
	case len(domainString) == 0:
		return errors.New("domain is not set")
	case len(domainString) > maxDomainLength:
		return fmt.Errorf("domain name is too long: %q has a length of %d characters exceeding the maximum of %d",
			domainString, len(domainString), maxDomainLength)
	}

	labelStartIndex := 0
	for i, character := range domainString {
		if character == '.' {
			const maxLabelLength = 63
			switch {
			case i == labelStartIndex:
				return fmt.Errorf("domain name has invalid character: label starts with '.' for domain %q", domainString)
			case i-labelStartIndex > maxLabelLength:
				return fmt.Errorf("domain label is too long: for domain %q", domainString)
			case domainString[labelStartIndex] == '-':
				return fmt.Errorf("domain name has invalid character: label starts with '-' for domain %q", domainString)
			case domainString[i-1] == '-':
				return fmt.Errorf("domain name has invalid character: label ends with '-' for domain %q", domainString)
			}
			labelStartIndex = i + 1
			continue
		}

		if (character < 'a' || character > 'z') &&
			(character < '0' || character > '9') &&
			character != '-' &&
			(character < 'A' || character > 'Z') {
			r, _ := utf8.DecodeRuneInString(domainString[i:])
			if r == utf8.RuneError {
				return fmt.Errorf("domain name has invalid character: invalid rune at offset %d for domain %q", i, domainString)
			}
			return fmt.Errorf("domain name has invalid character: '%c' for domain %q", r, domainString)
		}
	}

	// check top level domain validity
	const maxLabelLength = 63
	switch {
	case labelStartIndex == len(domainString):
		return fmt.Errorf("domain has missing top level domain: %q", domainString)
	case len(domainString)-labelStartIndex > maxLabelLength:
		return fmt.Errorf("domain label is too long: TLD label in domain %q", domainString)
	case domainString[labelStartIndex] == '-':
		return fmt.Errorf("domain name has invalid character: TLD label starts with '-' in domain %q", domainString)
	case domainString[len(domainString)-1] == '-':
		return fmt.Errorf("domain name has invalid character: TLD label ends with '-' in domain %q", domainString)
	case domainString[labelStartIndex] >= '0' && domainString[labelStartIndex] <= '9':
		return fmt.Errorf("domain name has invalid character: TLD label begins with a digit in domain %q", domainString)
	}
	return nil
}
