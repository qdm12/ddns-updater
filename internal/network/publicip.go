package network

import (
	"fmt"
	"net"
	"net/http"
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
		return nil, fmt.Errorf("cannot get public %s address from %s: %s", ipVersion, url, err)
	} else if status != http.StatusOK {
		return nil, fmt.Errorf("cannot get public %s address from %s: HTTP status code %d", ipVersion, url, status)
	}
	verifier := verification.NewVerifier()
	regexSearch := verifier.SearchIPv4
	if ipVersion == constants.IPv6 {
		regexSearch = verifier.SearchIPv6
	}
	ips := regexSearch(string(content))
	if ips == nil {
		return nil, fmt.Errorf("no public %s address found at %s", ipVersion, url)
	} else if len(ips) > 1 {
		return nil, fmt.Errorf("multiple public %s addresses found at %s: %s", ipVersion, url, strings.Join(ips, " "))
	}
	ip = net.ParseIP(ips[0])
	if ip == nil { // in case the regex is not restrictive enough
		return nil, fmt.Errorf("Public IP address %q found at %s is not valid", ips[0], url)
	}
	return ip, nil
}
