package update

import (
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/qdm12/ddns-updater/internal/constants"
	libnetwork "github.com/qdm12/golibs/network"
	"github.com/qdm12/golibs/verification"
)

func updateNoIP(client libnetwork.Client, hostname, username, password string, ip net.IP) (newIP net.IP, err error) {
	url := strings.ToLower(constants.NoIPURL + "?hostname=" + hostname)
	if ip != nil {
		url += "&myip=" + ip.String()
	}
	r, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	r.Header.Set("Authorization", "Basic "+username+":"+password)
	r.Header.Set("User-Agent", "DDNS-Updater quentin.mcgaw@gmail.com")
	status, content, err := client.DoHTTPRequest(r)
	if err != nil {
		return nil, err
	}
	s := string(content)
	switch s {
	case "":
		return nil, fmt.Errorf("HTTP status %d", status)
	case "911":
		return nil, fmt.Errorf("NoIP's internal server error 911")
	case "abuse":
		return nil, fmt.Errorf("username is banned due to abuse")
	case "!donator":
		return nil, fmt.Errorf("user has not this extra feature")
	case "badagent":
		return nil, fmt.Errorf("user agent is banned")
	case "badauth":
		return nil, fmt.Errorf("invalid username password combination")
	case "nohost":
		return nil, fmt.Errorf("hostname does not exist")
	}
	if strings.Contains(s, "nochg") || strings.Contains(s, "good") {
		ips := verification.NewVerifier().SearchIPv4(s)
		if ips == nil {
			return nil, fmt.Errorf("no IP address in response")
		}
		newIP = net.ParseIP(ips[0])
		if newIP == nil {
			return nil, fmt.Errorf("IP address received %q is malformed", ips[0])
		}
		if ip != nil && !ip.Equal(newIP) {
			return nil, fmt.Errorf("new IP address %s is not %s", newIP.String(), ip.String())
		}
		return newIP, nil
	}
	return nil, fmt.Errorf("invalid response %q", s)
}
