package utils

import (
	"fmt"
	"strings"
	"testing"

	"github.com/qdm12/ddns-updater/internal/provider/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_CheckDomain(t *testing.T) {
	t.Parallel()
	testCases := map[string]struct {
		domain     string
		errWrapped error
		errMessage string
	}{
		"empty_domain": {
			domain:     "",
			errWrapped: errors.ErrDomainNotSet,
			errMessage: "domain is not set",
		},
		"lowercase_valid": {
			domain: "example.com",
		},
		"uppercase_valid": {
			domain: "EXAMPLE.com",
		},
		"hyphen_valid": {
			domain: "foo-bar.com",
		},
		"subdomain_valid": {
			domain: "www1.foo-bar.com",
		},
		"digits_valid": {
			domain: "192.168.1.1.example.com",
		},
		"domain_too_long": {
			domain:     strings.Repeat("a", 300),
			errWrapped: ErrDomainTooLong,
			errMessage: fmt.Sprintf(`domain name is too long: "%s" has a length of 300 characters `+
				`exceeding the maximum of 255`, strings.Repeat("a", 300)),
		},
		"label_too_long": {
			domain:     strings.Repeat("a", 70) + ".com",
			errWrapped: ErrDomainLabelTooLong,
			errMessage: "domain label is too long: for domain " +
				"\"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.com\"",
		},
		"tld_too_long": {
			domain:     "example." + strings.Repeat("b", 70),
			errWrapped: ErrDomainLabelTooLong,
			errMessage: "domain label is too long: TLD label in domain " +
				"\"example.bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb\"",
		},
		"invalid_character_?": {
			domain:     "?",
			errWrapped: ErrDomainInvalidCharacter,
			errMessage: "domain name has invalid character: '?' for domain \"?\"",
		},
		"invalid_character_tab": {
			domain:     "\t",
			errWrapped: ErrDomainInvalidCharacter,
			errMessage: "domain name has invalid character: '\t' for domain \"\\t\"",
		},
		"invalid_character_à": {
			domain:     "exàmple.com",
			errWrapped: ErrDomainInvalidCharacter,
			errMessage: "domain name has invalid character: 'à' for domain \"exàmple.com\"",
		},
		"invalid_rune": {
			domain:     "www.\xbd\xb2.com",
			errWrapped: ErrDomainInvalidCharacter,
			errMessage: "domain name has invalid character: invalid rune at offset 4 for domain \"www.\\xbd\\xb2.com\"",
		},
		"invalid_starts_hyphen": {
			domain:     "-example.com",
			errWrapped: ErrDomainInvalidCharacter,
			errMessage: "domain name has invalid character: label starts with '-' for domain \"-example.com\"",
		},
		"invalid_ends_hyphen": {
			domain:     "example-.com",
			errWrapped: ErrDomainInvalidCharacter,
			errMessage: "domain name has invalid character: label ends with '-' for domain \"example-.com\"",
		},
		"invalid_tld_starts_hyphen": {
			domain:     "example.-com",
			errWrapped: ErrDomainInvalidCharacter,
			errMessage: "domain name has invalid character: TLD label starts with '-' in domain \"example.-com\"",
		},
		"invalid_tld_ends_hyphen": {
			domain:     "example.com-",
			errWrapped: ErrDomainInvalidCharacter,
			errMessage: "domain name has invalid character: TLD label ends with '-' in domain \"example.com-\"",
		},
		"invalid_tld_starts_digit": {
			domain:     "example.1com",
			errWrapped: ErrDomainInvalidCharacter,
			errMessage: "domain name has invalid character: TLD label begins with a digit in domain \"example.1com\"",
		},
		"invalid_empty_first_label": {
			domain:     ".example.com",
			errWrapped: ErrDomainInvalidCharacter,
			errMessage: "domain name has invalid character: label starts with '.' for domain \".example.com\"",
		},
		"invalid_empty_middle_label": {
			domain:     "example..com",
			errWrapped: ErrDomainInvalidCharacter,
			errMessage: "domain name has invalid character: label starts with '.' for domain \"example..com\"",
		},
		"invalid_trailing_dot": {
			domain:     "example.com.",
			errWrapped: ErrDomainTLDMissing,
			errMessage: "domain has missing top level domain: \"example.com.\"",
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := CheckDomain(testCase.domain)

			require.ErrorIs(t, err, testCase.errWrapped)
			if testCase.errWrapped != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
		})
	}
}
