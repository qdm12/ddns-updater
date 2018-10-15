package main

import (
	"errors"
	"regexp"
)

var regexIP = regexp.MustCompile(`(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)(\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)){3}`).FindString

func getPublicIP(address string) (ip string, err error) {
	r, err := buildHTTPGet(address)
	if err != nil {
		return ip, err
	}
	status, content, err := doHTTPRequest(r, httpGetTimeout)
	if err != nil {
		return ip, err
	}
	if status != "200" {
		return ip, errors.New(address + " responded with a status " + status)
	}
	ip = regexIP(string(content))
	if ip == "" {
		return ip, errors.New("No public IP found at " + address)
	}
	return ip, nil
}
