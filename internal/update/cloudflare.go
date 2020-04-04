package update

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"

	"github.com/qdm12/ddns-updater/internal/constants"
	"github.com/qdm12/ddns-updater/internal/network"
	libnetwork "github.com/qdm12/golibs/network"
)

func updateCloudflare(client libnetwork.Client, zoneIdentifier, identifier, host, email, key, userServiceKey, token string, proxied bool, ttl uint, ip net.IP) (err error) {
	if ip == nil {
		return fmt.Errorf("IP address was not given to updater")
	}
	type cloudflarePutBody struct {
		Type    string `json:"type"`    // forced to A
		Name    string `json:"name"`    // DNS record name i.e. example.com
		Content string `json:"content"` // ip address
		Proxied bool   `json:"proxied"` // whether the record is receiving the performance and security benefits of Cloudflare
		TTL     uint   `json:"ttl"`
	}
	URL := constants.CloudflareURL + "/zones/" + zoneIdentifier + "/dns_records/" + identifier
	r, err := network.BuildHTTPPut(
		URL,
		cloudflarePutBody{
			Type:    "A",
			Name:    host,
			Content: ip.String(),
			Proxied: proxied,
			TTL:     ttl,
		},
	)
	if err != nil {
		return err
	}
	switch {
	case len(token) > 0:
		r.Header.Set("Authorization", "Bearer "+token)
	case len(userServiceKey) > 0:
		r.Header.Set("X-Auth-User-Service-Key", userServiceKey)
	case len(email) > 0 && len(key) > 0:
		r.Header.Set("X-Auth-Email", email)
		r.Header.Set("X-Auth-Key", key)
	default:
		return fmt.Errorf("email and key are both unset and user service key is not set and no token was provided")
	}
	status, content, err := client.DoHTTPRequest(r)
	if err != nil {
		return err
	} else if status > http.StatusUnsupportedMediaType {
		return fmt.Errorf("HTTP status %d", status)
	}
	var parsedJSON struct {
		Success bool `json:"success"`
		Errors  []struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"errors"`
		Result struct {
			Content string `json:"content"`
		} `json:"result"`
	}
	if err := json.Unmarshal(content, &parsedJSON); err != nil {
		return err
	} else if !parsedJSON.Success {
		var errStr string
		for _, e := range parsedJSON.Errors {
			errStr += fmt.Sprintf("error %d: %s; ", e.Code, e.Message)
		}
		return fmt.Errorf(errStr)
	}
	newIP := net.ParseIP(parsedJSON.Result.Content)
	if newIP == nil {
		return fmt.Errorf("new IP %q is malformed", parsedJSON.Result.Content)
	} else if !newIP.Equal(ip) {
		return fmt.Errorf("new IP address %s is not %s", newIP.String(), ip.String())
	}
	return nil
}
