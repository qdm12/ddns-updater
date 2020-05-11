package update

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"

	"github.com/qdm12/ddns-updater/internal/network"
	libnetwork "github.com/qdm12/golibs/network"
)

func updateGoDaddy(client libnetwork.Client, host, domain, key, secret string, ip net.IP) error {
	if ip == nil {
		return fmt.Errorf("IP address was not given to updater")
	}
	type goDaddyPutBody struct {
		Data string `json:"data"` // IP address to update to
	}
	u := url.URL{
		Scheme: "https",
		Host:   "api.godaddy.com",
		Path:   fmt.Sprintf("/v1/domains/%s/records/A/%s", domain, host),
	}
	r, err := network.BuildHTTPPut(u.String(), []goDaddyPutBody{{ip.String()}})
	if err != nil {
		return err
	}
	r.Header.Set("User-Agent", "DDNS-Updater quentin.mcgaw@gmail.com")
	r.Header.Set("Authorization", "sso-key "+key+":"+secret)
	status, content, err := client.DoHTTPRequest(r)
	if err != nil {
		return err
	} else if status != http.StatusOK {
		var parsedJSON struct {
			Message string `json:"message"`
		}
		if err := json.Unmarshal(content, &parsedJSON); err != nil {
			return err
		} else if len(parsedJSON.Message) > 0 {
			return fmt.Errorf("HTTP status %d - %s", status, parsedJSON.Message)
		}
		return fmt.Errorf("HTTP status %d", status)
	}
	return nil
}
