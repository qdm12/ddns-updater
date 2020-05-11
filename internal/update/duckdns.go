package update

import (
	"fmt"
	"net"
	"net/http"
	"net/url"

	"github.com/qdm12/golibs/network"
	"github.com/qdm12/golibs/verification"
)

func updateDuckDNS(client network.Client, domain, token string, ip net.IP) (newIP net.IP, err error) {
	u := url.URL{
		Scheme: "https",
		Host:   "www.duckdns.org",
		Path:   "/update",
	}
	var values url.Values
	values.Set("verbose", "true")
	values.Set("domains", domain)
	values.Set("token", token)
	u.RawQuery = values.Encode()
	if ip != nil {
		values.Set("ip", ip.String())
	}
	r, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	r.Header.Set("User-Agent", "DDNS-Updater quentin.mcgaw@gmail.com")
	status, content, err := client.DoHTTPRequest(r)
	if err != nil {
		return nil, err
	} else if status != http.StatusOK {
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
