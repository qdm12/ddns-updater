package network

import (
	"net/http"
	"ddns-updater/pkg/regex"
	"fmt"
)

// GetPublicIP downloads a webpage and extracts the IP address from it
func GetPublicIP(client *http.Client, URL string) (ip string, err error) {
	content, err := GetContent(client, URL)
	if err != nil {
		return ip, fmt.Errorf("cannot get public IP address from %s: %s", URL, err)
	}
	ip = regex.FindIP(string(content))
	if ip == "" {
		return ip, fmt.Errorf("no public IP found at %s: %s", URL, err)
	}
	return ip, nil
}
