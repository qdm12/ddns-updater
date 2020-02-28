package network

import (
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/qdm12/golibs/network"
	"github.com/qdm12/golibs/verification"
)

// GetPublicIP downloads a webpage and extracts the IP address from it
func GetPublicIP(client network.Client, URL string) (ip net.IP, err error) {
	content, status, err := client.GetContent(URL)
	if err != nil {
		return nil, fmt.Errorf("cannot get public IP address from %s: %s", URL, err)
	} else if status != http.StatusOK {
		return nil, fmt.Errorf("cannot get public IP address from %s: HTTP status code %d", URL, status)
	}
	ips := verification.NewVerifier().SearchIPv4(string(content))
	if ips == nil {
		return nil, fmt.Errorf("no public IPv4 address found at %s", URL)
	} else if len(ips) > 1 {
		return nil, fmt.Errorf("multiple public IPv4 addresses found at %s: %s", URL, strings.Join(ips, " "))
	}
	ip = net.ParseIP(ips[0])
	if ip == nil {
		return nil, fmt.Errorf("Public IP address %q found at %s is not valid", ips[0], URL)
	}
	return ip, nil
}
