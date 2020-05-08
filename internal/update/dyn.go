package update

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"

	"github.com/qdm12/ddns-updater/internal/constants"
	"github.com/qdm12/golibs/network"
)

// See https://help.dyn.com/update-a-record-api/
// Token obtained using https://help.dyn.com/session-log-in/
func updateDyn(client network.Client, zone, fqdn, recordID, token string, ip net.IP) (err error) {
	if ip == nil {
		return fmt.Errorf("IP address was not given to updater")
	}
	url := fmt.Sprintf("%s/REST/ARecord/%s/%s/%s/", constants.DynURL, zone, fqdn, recordID)
	var body struct {
		Rdata struct {
			Address string `json:"address"`
		} `json:"rdata"`
		TTL string `json:"ttl"`
	}
	body.TTL = "0"
	body.Rdata.Address = ip.String()
	r, err := http.NewRequest(http.MethodPut, url, nil)
	if err != nil {
		return err
	}
	r.Header.Set("User-Agent", "DDNS-Updater quentin.mcgaw@gmail.com")
	status, content, err := client.DoHTTPRequest(r)
	if err != nil {
		return err
	}
	var response struct {
		Rdata struct {
			Address string `json:"address"`
		} `json:"rdata"`
	}
	if status != http.StatusOK {
		return fmt.Errorf("HTTP status %d", status)
	} else if err := json.Unmarshal(content, &response); err != nil {
		return err
	}
	newIP := net.ParseIP(response.Rdata.Address)
	if newIP == nil {
		return fmt.Errorf("IP address received %q is malformed", response.Rdata.Address)
	} else if ip != nil && !ip.Equal(newIP) {
		return fmt.Errorf("new IP address %s is not %s", newIP.String(), ip.String())
	}
	return nil
}
