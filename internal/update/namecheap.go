package update

import (
	"encoding/xml"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/qdm12/ddns-updater/internal/constants"
	libnetwork "github.com/qdm12/golibs/network"
)

func updateNamecheap(client libnetwork.Client, host, domain, password string, ip net.IP) (newIP net.IP, err error) {
	url := strings.ToLower(constants.NamecheapURL + "?host=" + host + "&domain=" + domain + "&password=" + password)
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
