package hetznercloud

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"
	"net/url"

	"github.com/qdm12/ddns-updater/internal/provider/constants"
	"github.com/qdm12/ddns-updater/internal/provider/errors"
	"github.com/qdm12/ddns-updater/internal/provider/utils"
)

// setRecords ändert die Records eines bereits bestehenden RRsets über die
// set_records-Action. Die Action läuft server-seitig asynchron; wir werten
// nur den unmittelbaren Status aus und pollen bewusst nicht, da ddns-updater
// den RRset im nächsten Zyklus ohnehin erneut liest.
// Siehe https://docs.hetzner.cloud/reference/cloud#tag/zone-actions/POST/zones/{zone_id}/rrsets/{name}/{type}/actions/set_records
func (p *Provider) setRecords(ctx context.Context, client *http.Client, zoneID string, ip netip.Addr) (
	newIP netip.Addr, err error,
) {
	recordType := constants.A
	if ip.Is6() {
		recordType = constants.AAAA
	}

	u := url.URL{
		Scheme: "https",
		Host:   "api.hetzner.cloud",
		Path: "/v1/zones/" + zoneID + "/rrsets/" +
			url.PathEscape(p.owner) + "/" + recordType + "/actions/set_records",
	}

	// set_records setzt ausschließlich die Records; die TTL wird über die
	// change_ttl-Action verwaltet und bleibt hier unverändert.
	requestData := struct {
		Records []struct {
			Value string `json:"value"`
		} `json:"records"`
	}{
		Records: []struct {
			Value string `json:"value"`
		}{{Value: ip.String()}},
	}

	buffer := bytes.NewBuffer(nil)
	if err = json.NewEncoder(buffer).Encode(requestData); err != nil {
		return netip.Addr{}, fmt.Errorf("json encoding request data: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), buffer)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("creating http request: %w", err)
	}
	p.setHeaders(request)

	response, err := client.Do(request)
	if err != nil {
		return netip.Addr{}, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusCreated {
		return netip.Addr{}, fmt.Errorf("%w: %d: %s",
			errors.ErrHTTPStatusNotValid, response.StatusCode, utils.BodyToSingleLine(response.Body))
	}

	var result struct {
		Action struct {
			Status string `json:"status"`
			Error  *struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		} `json:"action"`
	}
	if err = json.NewDecoder(response.Body).Decode(&result); err != nil {
		return netip.Addr{}, fmt.Errorf("json decoding response body: %w", err)
	}

	// Nur ein bereits fehlgeschlagener Action-Status wird als Fehler gewertet;
	// "running" und "success" gelten als angenommen.
	if result.Action.Status == "error" {
		message := "action failed"
		if result.Action.Error != nil {
			message = result.Action.Error.Code + ": " + result.Action.Error.Message
		}
		return netip.Addr{}, fmt.Errorf("%w: %s", errors.ErrHTTPStatusNotValid, message)
	}

	// Die Action-Antwort enthält keine Records; bei erfolgreicher Annahme wird
	// die gesendete IP zurückgegeben.
	return ip, nil
}
