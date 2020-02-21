package network

import (
	"fmt"
	"net"

	"github.com/qdm12/golibs/network"
	"github.com/qdm12/golibs/verification"
)

// GetPublicIP downloads a webpage and extracts the IP address from it
func GetPublicIP(client network.Client, URL string) (ip net.IP, err error) {
	content, status, err := client.GetContent(URL)
	if err != nil {
		return nil, fmt.Errorf("cannot get public IP address from %s: %s", URL, err)
	} else if status != 200 {
		return nil, fmt.Errorf("cannot get public IP address from %s: HTTP status code %d", URL, status)
	}
	ips := verification.NewVerifier().SearchIPv4(string(content))
	if ips == nil {
		return nil, fmt.Errorf("no public IP found at %s: %s", URL, err)
	}
	ip = net.ParseIP(ips[0])
	if ip == nil {
		return nil, fmt.Errorf("Public IP address %q found at %s is not valid", ips[0], URL)
	}
	return ip, nil
}
