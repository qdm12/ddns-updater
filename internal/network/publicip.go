package network

import (
	"bytes"
	"fmt"
	"net"
	"net/http"
	"sort"
	"strings"

	"github.com/qdm12/ddns-updater/internal/constants"
	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/golibs/network"
	"github.com/qdm12/golibs/verification"
)

// GetPublicIP downloads a webpage and extracts the IP address from it
func GetPublicIP(client network.Client, url string, ipVersion models.IPVersion) (ip net.IP, err error) {
	content, status, err := client.GetContent(url)
	if err != nil {
		return nil, fmt.Errorf("cannot get public %s address: %w", ipVersion, err)
	} else if status != http.StatusOK {
		return nil, fmt.Errorf("cannot get public %s address from %s: HTTP status code %d", ipVersion, url, status)
	}
	s := string(content)
	switch ipVersion {
	case constants.IPv4:
		return searchIP(constants.IPv4, s)
	case constants.IPv6:
		return searchIP(constants.IPv6, s)
	case constants.IPv4OrIPv6:
		var ipv4Err, ipv6Err error
		ip, ipv4Err = searchIP(constants.IPv4, s)
		if ipv4Err != nil {
			ip, ipv6Err = searchIP(constants.IPv6, s)
		}
		if ipv6Err != nil {
			return nil, fmt.Errorf("%s, %s", ipv4Err, ipv6Err)
		}
		return ip, nil
	default:
		return nil, fmt.Errorf("ip version %q not supported", ipVersion)
	}
}

func searchIP(version models.IPVersion, s string) (ip net.IP, err error) {
	verifier := verification.NewVerifier()
	var regexSearch func(s string) []string
	switch version {
	case constants.IPv4:
		regexSearch = verifier.SearchIPv4
	case constants.IPv6:
		regexSearch = verifier.SearchIPv6
	default:
		return nil, fmt.Errorf("ip version %q is not supported for regex search", version)
	}
	ips := regexSearch(s)
	if ips == nil {
		return nil, fmt.Errorf("no public %s address found", version)
	}
	uniqueIPs := make(map[string]struct{})
	for _, ipString := range ips {
		uniqueIPs[ipString] = struct{}{}
	}
	netIPs := []net.IP{}
	for ipString := range uniqueIPs {
		netIP := net.ParseIP(ipString)
		if netIP == nil || netIPIsPrivate(netIP) {
			// in case the regex is not restrictive enough
			// or the IP address is private
			continue
		}
		netIPs = append(netIPs, netIP)
	}
	switch len(netIPs) {
	case 0:
		return nil, fmt.Errorf("no public %s address found", version)
	case 1:
		return netIPs[0], nil
	default:
		sort.Slice(netIPs, func(i, j int) bool {
			return bytes.Compare(netIPs[i], netIPs[j]) < 0
		})
		ips = make([]string, len(netIPs))
		for i := range netIPs {
			ips[i] = netIPs[i].String()
		}
		return nil, fmt.Errorf("multiple public %s addresses found: %s", version, strings.Join(ips, " "))
	}
}

func netIPIsPrivate(netIP net.IP) bool {
	for _, privateCIDRBlock := range [8]string{
		"127.0.0.1/8",    // localhost
		"10.0.0.0/8",     // 24-bit block
		"172.16.0.0/12",  // 20-bit block
		"192.168.0.0/16", // 16-bit block
		"169.254.0.0/16", // link local address
		"::1/128",        // localhost IPv6
		"fc00::/7",       // unique local address IPv6
		"fe80::/10",      // link local address IPv6
	} {
		_, CIDR, _ := net.ParseCIDR(privateCIDRBlock)
		if CIDR.Contains(netIP) {
			return true
		}
	}
	return false
}
