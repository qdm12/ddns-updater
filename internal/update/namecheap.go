package update

import (
	"encoding/xml"
	"fmt"
	"net"
	"net/http"
	"net/url"

	"github.com/qdm12/golibs/network"
)

func updateNamecheap(client network.Client, host, domain, password string, ip net.IP) (newIP net.IP, err error) {
	u := url.URL{
		Scheme: "https",
		Host:   "dynamicdns.park-your-domain.com",
		Path:   "/update",
		// User:   url.UserPassword(username, password),
	}
	values := url.Values{}
	values.Set("host", host)
	values.Set("domain", domain)
	values.Set("password", password)
	if ip != nil {
		values.Set("ip", ip.String())
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
	} else if status != http.StatusOK {
		return nil, fmt.Errorf("HTTP status %d", status)
	}
	var parsedXML struct {
		Errors struct {
			Error string `xml:"Err1"`
		} `xml:"errors"`
		IP string `xml:"IP"`
	}
	err = xml.Unmarshal(content, &parsedXML)
	if err != nil {
		return nil, err
	} else if parsedXML.Errors.Error != "" {
		return nil, fmt.Errorf(parsedXML.Errors.Error)
	}
	newIP = net.ParseIP(parsedXML.IP)
	if newIP == nil {
		return nil, fmt.Errorf("IP address received %q is malformed", parsedXML.IP)
	}
	if ip != nil && !ip.Equal(newIP) {
		return nil, fmt.Errorf("new IP address %s is not %s", newIP.String(), ip.String())
	}
	return newIP, nil
}
