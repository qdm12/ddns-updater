package utils

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	ddnserrors "github.com/qdm12/ddns-updater/internal/provider/errors"
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

var (
	ErrDomainTooLong          = errors.New("domain name is too long")
	ErrDomainLabelTooLong     = errors.New("domain label is too long")
	ErrDomainInvalidCharacter = errors.New("domain name has invalid character")
	ErrDomainTLDMissing       = errors.New("domain has missing top level domain")
)

// CheckDomain returns an non-nil error if the domain name is not valid.
// https://tools.ietf.org/html/rfc1034#section-3.5
// https://tools.ietf.org/html/rfc1123#section-2.
func CheckDomain(domainString string) (err error) {
	const maxDomainLength = 255
	switch {
	case len(domainString) == 0:
		return fmt.Errorf("%w", ddnserrors.ErrDomainNotSet)
	case len(domainString) > maxDomainLength:
		return fmt.Errorf("%w: %q has a length of %d characters exceeding the maximum of %d",
			ErrDomainTooLong, domainString, len(domainString), maxDomainLength)
	}

	labelStartIndex := 0
	for i, character := range domainString {
		if character == '.' {
			const maxLabelLength = 63
			switch {
			case i == labelStartIndex:
				return fmt.Errorf("%w: label starts with '.' for domain %q", ErrDomainInvalidCharacter, domainString)
			case i-labelStartIndex > maxLabelLength:
				return fmt.Errorf("%w: for domain %q", ErrDomainLabelTooLong, domainString)
			case domainString[labelStartIndex] == '-':
				return fmt.Errorf("%w: label starts with '-' for domain %q", ErrDomainInvalidCharacter, domainString)
			case domainString[i-1] == '-':
				return fmt.Errorf("%w: label ends with '-' for domain %q", ErrDomainInvalidCharacter, domainString)
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
				return fmt.Errorf("%w: invalid rune at offset %d for domain %q",
					ErrDomainInvalidCharacter, i, domainString)
			}
			return fmt.Errorf("%w: '%c' for domain %q",
				ErrDomainInvalidCharacter, r, domainString)
		}
	}

	// check top level domain validity
	const maxLabelLength = 63
	switch {
	case labelStartIndex == len(domainString):
		return fmt.Errorf("%w: %q", ErrDomainTLDMissing, domainString)
	case len(domainString)-labelStartIndex > maxLabelLength:
		return fmt.Errorf("%w: TLD label in domain %q",
			ErrDomainLabelTooLong, domainString)
	case domainString[labelStartIndex] == '-':
		return fmt.Errorf("%w: TLD label starts with '-' in domain %q",
			ErrDomainInvalidCharacter, domainString)
	case domainString[len(domainString)-1] == '-':
		return fmt.Errorf("%w: TLD label ends with '-' in domain %q",
			ErrDomainInvalidCharacter, domainString)
	case domainString[labelStartIndex] >= '0' && domainString[labelStartIndex] <= '9':
		return fmt.Errorf("%w: TLD label begins with a digit in domain %q",
			ErrDomainInvalidCharacter, domainString)
	}
	return nil
}
