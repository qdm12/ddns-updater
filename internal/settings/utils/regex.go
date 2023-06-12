package utils

import (
	"net/netip"
	"regexp"
)

var (
	regexEmail = regexp.MustCompile(`[a-zA-Z0-9-_.+]+@[a-zA-Z0-9-_.]+\.[a-zA-Z]{2,10}`)
	regexIPv4  = regexp.MustCompile(`(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)(\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)){3}`)
	regexIPv6  = regexp.MustCompile(`(([0-9a-fA-F]{1,4}:){7,7}[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,7}:|([0-9a-fA-F]{1,4}:){1,6}:[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,5}(:[0-9a-fA-F]{1,4}){1,2}|([0-9a-fA-F]{1,4}:){1,4}(:[0-9a-fA-F]{1,4}){1,3}|([0-9a-fA-F]{1,4}:){1,3}(:[0-9a-fA-F]{1,4}){1,4}|([0-9a-fA-F]{1,4}:){1,2}(:[0-9a-fA-F]{1,4}){1,5}|[0-9a-fA-F]{1,4}:((:[0-9a-fA-F]{1,4}){1,6})|:((:[0-9a-fA-F]{1,4}){1,7}|:)|fe80:(:[0-9a-fA-F]{0,4}){0,4}%[0-9a-zA-Z]{1,}|::(ffff(:0{1,4}){0,1}:){0,1}((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])|([0-9a-fA-F]{1,4}:){1,4}:((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9]))`) //nolint:lll
)

func MatchEmail(email string) bool {
	return regexEmail.MatchString(email)
}

func FindIPv4Addresses(text string) (addresses []netip.Addr) {
	const n = -1
	ipv4Strings := regexIPv4.FindAllString(text, n)
	return mustParseIPAddresses(ipv4Strings)
}

func FindIPv6Addresses(text string) (addresses []netip.Addr) {
	const n = -1
	ipv6Strings := regexIPv6.FindAllString(text, n)
	return mustParseIPAddresses(ipv6Strings)
}

func mustParseIPAddresses(ipStrings []string) (addresses []netip.Addr) {
	if len(ipStrings) == 0 {
		return nil
	}

	addresses = make([]netip.Addr, len(ipStrings))
	for i, ipString := range ipStrings {
		addresses[i] = netip.MustParseAddr(ipString)
	}

	return addresses
}
