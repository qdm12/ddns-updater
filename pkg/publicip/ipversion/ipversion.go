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
	IP4and6
)

func (v IPVersion) String() string {
	switch v {
	case IP4or6:
		return "ipv4 or ipv6"
	case IP4:
		return "ipv4"
	case IP6:
		return "ipv6"
	case IP4and6:
		return "ipv4 and ipv6"
	default:
		panic(fmt.Sprintf("ip version %d not programmed", v))
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
	case "ipv4 and ipv6":
		return IP4and6, nil
	default:
		return IP4or6, fmt.Errorf("%w: %q", ErrInvalidIPVersion, s)
	}
}
