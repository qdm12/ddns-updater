package dondominio

import (
	"context"
	"fmt"
	"net/http"
	"net/netip"
)

// See https://dondominio.dev/en/api/docs/api/#dns-zone-update-service-dnsupdate
func (p *Provider) update(ctx context.Context, client *http.Client, entityID string,
	ip netip.Addr) (err error) {
	requestData := struct {
		APIUser     string `json:"apiuser"`
		APIPasswd   string `json:"apipasswd"`
		ServiceName string `json:"serviceName"`
		EntityID    string `json:"entityID"`
		Value       string `json:"value"`
	}{
		APIUser:     p.username,
		APIPasswd:   p.password,
		ServiceName: p.name,
		EntityID:    entityID,
		Value:       ip.String(),
	}

	_, err = apiCall(ctx, client, "/service/dnsupdate", requestData)
	if err != nil {
		return fmt.Errorf("for entity id %s: %w", entityID, err)
	}

	return nil
}
