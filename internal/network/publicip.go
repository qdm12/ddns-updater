package network

import (
	"fmt"
	"net/http"

	libnetwork "github.com/qdm12/golibs/network"
	"github.com/qdm12/golibs/verification"
)

// GetPublicIP downloads a webpage and extracts the IP address from it
func GetPublicIP(client *http.Client, URL string) (ip string, err error) {
	content, err := libnetwork.GetContent(client, URL, libnetwork.GetContentParamsType{})
	if err != nil {
		return ip, fmt.Errorf("cannot get public IP address from %s: %s", URL, err)
	}
	ips := verification.SearchIPv4(string(content))
	if ips == nil {
		return ip, fmt.Errorf("no public IP found at %s: %s", URL, err)
	}
	ip = ips[0]
	return ip, nil
}
