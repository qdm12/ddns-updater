package network

import (
	"ddns-updater/pkg/regex"
	"fmt"
	"net/http"
)

// GetPublicIP downloads a webpage and extracts the IP address from it
func GetPublicIP(client *http.Client, URL string) (ip string, err error) {
	content, err := GetContent(client, URL)
	if err != nil {
		return ip, fmt.Errorf("cannot get public IP address from %s: %s", URL, err)
	}
	ips := regex.SearchIP(string(content))
	if ips == nil {
		return ip, fmt.Errorf("no public IP found at %s: %s", URL, err)
	}
	ip = ips[0]
	return ip, nil
}
