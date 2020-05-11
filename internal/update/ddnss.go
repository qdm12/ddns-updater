package update

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/qdm12/golibs/network"
)

func updateDDNSS(client network.Client, domain, host, username, password string, ip net.IP) error {
	u := url.URL{
		Scheme: "https",
		Host:   "www.ddnss.de",
		Path:   "/upd.php",
	}
	var values url.Values
	values.Set("user", username)
	values.Set("pwd", password)
	fqdn := domain
	if host != "@" {
		fqdn = host + "." + domain
	}
	values.Set("host", fqdn)
	if ip != nil {
		if ip.To4() == nil { // ipv6
			values.Set("ip6", ip.String())
		} else {
			values.Set("ip", ip.String())
		}
	}
	u.RawQuery = values.Encode()
	r, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}
	r.Header.Set("User-Agent", "DDNS-Updater quentin.mcgaw@gmail.com")
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
		return fmt.Errorf("ddnss.de: hostname %q does not exist", fqdn)
	case strings.Contains(s, "Updated 1 hostname"):
		return nil
	default:
		return fmt.Errorf("unknown response received from ddnss.de: %s", s)
	}
}
