package enum

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/qdm12/ddns-updater/internal/provider/errors"
)

// See https://docs.enum.co/api/operations/dnsservice_getzonebyname/
func (p *Provider) getZoneID(ctx context.Context, client *http.Client) (zoneID string, err error) {
	requestData := struct {
		ProjectID string `json:"projectId"`
		Name      string `json:"name"`
	}{
		ProjectID: p.projectID,
		Name:      strings.ToLower(p.domain),
	}

	response, err := p.doRequest(ctx, client, "GetZoneByName", requestData)
	if err != nil {
		return "", err
	}

	defer response.Body.Close()

	switch response.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound:
		return "", fmt.Errorf("%w: %s", errors.ErrZoneNotFound, p.domain)
	default:
		return "", handleErrorResponse(response)
	}

	decoder := json.NewDecoder(response.Body)
	var responseData struct {
		Zone struct {
			ID string `json:"id"`
		} `json:"zone"`
	}

	err = decoder.Decode(&responseData)
	if err != nil {
		return "", fmt.Errorf("json decoding response body: %w", err)
	}

	if responseData.Zone.ID == "" {
		return "", fmt.Errorf("%w", errors.ErrReceivedNoResult)
	}

	return responseData.Zone.ID, nil
}
