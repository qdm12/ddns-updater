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

func updateDuckDNS(client libnetwork.Client, domain, token string, ip net.IP) (newIP net.IP, err error) {
	url := strings.ToLower(constants.DuckdnsURL + "?domains=" + domain + "&token=" + token + "&verbose=true")
	if ip != nil {
		url += "&ip=" + ip.String()
	}
	r, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	status, content, err := client.DoHTTPRequest(r)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("HTTP status %d", status)
	}
	s := string(content)
	switch {
	case len(s) < 2:
		return nil, fmt.Errorf("response %q is too short", s)
	case s[0:2] == "KO":
		return nil, fmt.Errorf("invalid domain token combination")
	case s[0:2] == "OK":
		ips := verification.NewVerifier().SearchIPv4(s)
		if ips == nil {
			return nil, fmt.Errorf("no IP address in response")
		}
		newIP = net.ParseIP(ips[0])
		if newIP == nil {
			return nil, fmt.Errorf("IP address received %q is malformed", ips[0])
		}
		if ip != nil && !newIP.Equal(ip) {
			return nil, fmt.Errorf("new IP address %s is not %s", newIP.String(), ip.String())
		}
		return newIP, nil
	default:
		return nil, fmt.Errorf("invalid response %q", s)
	}
}
