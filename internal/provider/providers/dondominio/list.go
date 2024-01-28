package dondominio

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/qdm12/ddns-updater/internal/provider/constants"
)

// See https://dondominio.dev/en/api/docs/api/#dns-zone-list-service-dnslist
func (p *Provider) list(ctx context.Context, client *http.Client) (aIDs, aaaaIDs []string, err error) {
	requestData := struct {
		APIUser     string `json:"apiuser"`
		APIPasswd   string `json:"apipasswd"`
		ServiceName string `json:"serviceName"`
	}{
		APIUser:     p.username,
		APIPasswd:   p.password,
		ServiceName: p.name,
	}

	data, err := apiCall(ctx, client, "/service/dnslist", requestData)
	if err != nil {
		return nil, nil, err
	}

	var responseData struct {
		DNS []struct {
			EntityID string `json:"entityID"`
			Name     string `json:"name"`
			Type     string `json:"type"`
		} `json:"dns"`
	}
	err = json.Unmarshal(data, &responseData)
	if err != nil {
		return nil, nil, fmt.Errorf("json decoding response data: %w", err)
	}

	for _, record := range responseData.DNS {
		if record.Name != p.BuildDomainName() {
			continue
		}
		switch record.Type {
		case constants.A:
			aIDs = append(aIDs, record.EntityID)
		case constants.AAAA:
			aaaaIDs = append(aaaaIDs, record.EntityID)
		}
	}

	return aIDs, aaaaIDs, nil
}
