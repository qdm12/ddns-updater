package update

import (
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/qdm12/golibs/network"
)

func updateDDNSS(client network.Client, domain, host, username, password string, ip net.IP) error {
	var hostname string
	if host == "@" {
		hostname = strings.ToLower(domain)
	} else {
		hostname = strings.ToLower(host + "." + domain)
	}
	url := fmt.Sprintf("http://www.ddnss.de/upd.php?user=%s&pwd=%s&host=%s", username, password, hostname)
	if ip != nil {
		if ip.To4() == nil { // ipv6
			url += fmt.Sprintf("&ip6=%s", ip)
		} else {
			url += fmt.Sprintf("&ip=%s", ip)
		}
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
	if status != http.StatusOK {
		return fmt.Errorf("received status %d with message: %s", status, s)
	}
	switch {
	case strings.Contains(s, "badysys"):
		return fmt.Errorf("ddnss.de: invalid system parameter")
	case strings.Contains(s, "badauth"):
		return fmt.Errorf("ddnss.de: bad authentication")
	case strings.Contains(s, "notfqdn"):
		return fmt.Errorf("ddnss.de: hostname %q does not exist", hostname)
	case strings.Contains(s, "Updated 1 hostname"):
		return nil
	default:
		return fmt.Errorf("unknown response received from ddnss.de: %s", s)
	}
}
