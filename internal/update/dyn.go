package update

import (
	"fmt"
	"net"
	"net/http"
	"net/url"

	"github.com/qdm12/golibs/network"
)

func updateDyn(client network.Client, username, password, domain, host string, ipv4, ipv6 net.IP) (err error) {
	u := url.URL{
		Scheme: "https",
		User:   url.UserPassword(username, password),
		Host:   "members.dyndns.org",
		Path:   "/v3/update",
	}
	var values url.Values
	switch host {
	case "@":
		values.Set("hostname", domain)
	default:
		values.Set("hostname", fmt.Sprintf("%s.%s", host, domain))
	}
	if ipv4 != nil {
		values.Set("myip", ipv4.String())
	}
	if ipv6 != nil {
		values.Add("myip", ipv6.String())
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
	if status != http.StatusOK {
		return fmt.Errorf("HTTP status %d", status)
	}
	s := string(content)
	switch s {
	case "notfqdn":
		return fmt.Errorf("fully qualified domain name is not valid")
	case "badrequest":
		return fmt.Errorf("bad request")
	case "success":
		return nil
	default:
		return fmt.Errorf("unknown response: %s", s)
	}
}
