package update

import (
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/qdm12/golibs/network"
)

func updateInfomaniak(client network.Client, domain, host, username, password string, ip net.IP) (err error) {
	var hostname string
	if host == "@" {
		hostname = strings.ToLower(domain)
	} else {
		hostname = strings.ToLower(host + "." + domain)
	}
	url := fmt.Sprintf("https://%s:%s@infomaniak.com/nic/update?hostname=%s", username, password, hostname)
	if ip != nil {
		url += fmt.Sprintf("&myip=%s", ip)
	}
	r, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	status, content, err := client.DoHTTPRequest(r)
	if err != nil {
		return err
	}
	s := string(content)
	switch status {
	case http.StatusOK:
		switch {
		case strings.HasPrefix(s, "good "):
			receivedIP := net.ParseIP(s[5:])
			if receivedIP == nil {
				return fmt.Errorf("no received IP in response %q", s)
			} else if ip != nil && !ip.Equal(receivedIP) {
				return fmt.Errorf("received IP %s is not equal to expected IP %s", receivedIP, ip)
			}
			return nil
		case strings.HasPrefix(s, "nochg "):
			receivedIP := net.ParseIP(s[6:])
			if receivedIP == nil {
				return fmt.Errorf("no received IP in response %q", s)
			} else if ip != nil && !ip.Equal(receivedIP) {
				return fmt.Errorf("received IP %s is not equal to expected IP %s", receivedIP, ip)
			}
			return nil
		default:
			return nil
		}
	case http.StatusBadRequest:
		switch s {
		case "nohost":
			return fmt.Errorf("infomaniak.com: host %q does not exist for domain %q", host, domain)
		case "badauth":
			return fmt.Errorf("infomaniak.com: bad authentication")
		default:
			return fmt.Errorf("infomaniak.com: bad request: %s", s)
		}
	default:
		return fmt.Errorf("Received status %d with message: %s", status, s)
	}
}
