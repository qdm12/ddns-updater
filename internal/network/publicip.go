package network

import (
	"fmt"

	"github.com/qdm12/golibs/network"
	"github.com/qdm12/golibs/verification"
)

// GetPublicIP downloads a webpage and extracts the IP address from it
func GetPublicIP(client network.Client, URL string) (ip string, err error) {
	content, status, err := client.GetContent(URL)
	if err != nil {
		return ip, fmt.Errorf("cannot get public IP address from %s: %s", URL, err)
	} else if status != 200 {
		return ip, fmt.Errorf("cannot get public IP address from %s: HTTP status code %d", URL, status)
	}
	ips := verification.SearchIPv4(string(content))
	if ips == nil {
		return ip, fmt.Errorf("no public IP found at %s: %s", URL, err)
	}
	ip = ips[0]
	return ip, nil
}
