package ipextract

import (
	"net/netip"
	"strings"
)

// IPv4 extracts all valid IPv4 addresses from a given
// text string. Each IPv4 address must be separated by a character
// not part of the IPv4 alphabet (0123456789.).
// Performance-wise, this extraction is at least x3 times faster
// than using a regular expression.
func IPv4(text string) (addresses []netip.Addr) {
	const ipv4Alphabet = "0123456789."
	return extract(text, ipv4Alphabet)
}

// IPv6 extracts all valid IPv6 addresses from a given
// text string. Each IPv6 address must be separated by a character
// not part of the IPv6 alphabet (0123456789abcdefABCDEF:).
// Performance-wise, this extraction is at least x3 times faster
// than using a regular expression.
func IPv6(text string) (addresses []netip.Addr) {
	const ipv6Alphabet = "0123456789abcdefABCDEF:"
	return extract(text, ipv6Alphabet)
}

func extract(text string, alphabet string) (addresses []netip.Addr) {
	addressesSeen := make(map[netip.Addr]struct{})
	var start, end int
	for {
		for i := start; i < len(text); i++ {
			r := rune(text[i])
			if !strings.ContainsRune(alphabet, r) {
				break
			}
			end++
		}

		possibleIPString := text[start:end]
		ipAddress, err := netip.ParseAddr(possibleIPString)
		if err == nil { // Valid IP address found
			_, seen := addressesSeen[ipAddress]
			if !seen {
				addressesSeen[ipAddress] = struct{}{}
				addresses = append(addresses, ipAddress)
			}
		}

		if end == len(text) {
			return addresses
		}

		start = end + 1 // + 1 to skip non alphabet match character
		end = start
	}
}
