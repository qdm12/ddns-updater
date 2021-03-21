package ipversion

import (
	"errors"
	"fmt"
	"strings"
)

type IPVersion uint8

const (
	IP4or6 IPVersion = iota
	IP4
	IP6
)

func (v IPVersion) String() string {
	switch v {
	case IP4or6:
		return "ipv4 or ipv6"
	case IP4:
		return "ipv4"
	case IP6:
		return "ipv6"
	default:
		return "ip?"
	}
}

var ErrInvalidIPVersion = errors.New("invalid IP version")

func Parse(s string) (version IPVersion, err error) {
	switch strings.ToLower(s) {
	case "ipv4 or ipv6":
		return IP4or6, nil
	case "ipv4":
		return IP4, nil
	case "ipv6":
		return IP6, nil
	default:
		return IP4or6, fmt.Errorf("%w: %q", ErrInvalidIPVersion, s)
	}
}
