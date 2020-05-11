package update

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/qdm12/golibs/network"
)

func updateInfomaniak(client network.Client, domain, host, username, password string, ip net.IP) (newIP net.IP, err error) {
	u := url.URL{
		Scheme: "https",
		Host:   "infomaniak.com",
		Path:   "/nic/update",
		User:   url.UserPassword(username, password),
	}
	values := url.Values{}
	values.Set("hostname", domain)
	if host != "@" {
		values.Set("hostname", host+"."+domain)
	}
	if ip != nil {
		values.Set("myip", ip.String())
	}
	u.RawQuery = values.Encode()
	r, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	r.Header.Set("User-Agent", "DDNS-Updater quentin.mcgaw@gmail.com")
	status, content, err := client.DoHTTPRequest(r)
	if err != nil {
		return nil, err
	}
	s := string(content)
	switch status {
	case http.StatusOK:
		switch {
		case strings.HasPrefix(s, "good "):
			newIP = net.ParseIP(s[5:])
			if newIP == nil {
				return nil, fmt.Errorf("no received IP in response %q", s)
			} else if ip != nil && !ip.Equal(newIP) {
				return nil, fmt.Errorf("received IP %s is not equal to expected IP %s", newIP, ip)
			}
			return newIP, nil
		case strings.HasPrefix(s, "nochg "):
			newIP = net.ParseIP(s[6:])
			if newIP == nil {
				return nil, fmt.Errorf("no received IP in response %q", s)
			} else if ip != nil && !ip.Equal(newIP) {
				return nil, fmt.Errorf("received IP %s is not equal to expected IP %s", newIP, ip)
			}
			return newIP, nil
		default:
			return nil, fmt.Errorf("ok status but unknown response %q", s)
		}
	case http.StatusBadRequest:
		switch s {
		case "nohost":
			return nil, fmt.Errorf("infomaniak.com: host %q does not exist for domain %q", host, domain)
		case "badauth":
			return nil, fmt.Errorf("infomaniak.com: bad authentication")
		default:
			return nil, fmt.Errorf("infomaniak.com: bad request: %s", s)
		}
	default:
		return nil, fmt.Errorf("received status %d with message: %s", status, s)
	}
}
