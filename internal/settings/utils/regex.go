package utils

import (
	"regexp"
)

var (
	regexEmail = regexp.MustCompile(`[a-zA-Z0-9-_.+]+@[a-zA-Z0-9-_.]+\.[a-zA-Z]{2,10}`)
)

func MatchEmail(email string) bool {
	return regexEmail.MatchString(email)
}
