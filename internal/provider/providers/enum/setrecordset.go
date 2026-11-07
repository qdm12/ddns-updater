package enum

import (
	"context"
	"net/http"
	"net/netip"
)

// See https://docs.enum.co/api/operations/dnsservice_createrecordset/
// See https://docs.enum.co/api/operations/dnsservice_updaterecordset/
func (p *Provider) setRecordSet(ctx context.Context, client *http.Client, method, zoneID, recordType string,
	ttl uint32, ip netip.Addr,
) (err error) {
	type record struct {
		Content string `json:"content"`
	}
	requestData := struct {
		ProjectID string   `json:"projectId"`
		ZoneID    string   `json:"zoneId"`
		Name      string   `json:"name"`
		Type      string   `json:"type"`
		TTL       uint32   `json:"ttl,omitempty"`
		Records   []record `json:"records"`
	}{
		ProjectID: p.projectID,
		ZoneID:    zoneID,
		Name:      p.recordName(),
		Type:      recordType,
		TTL:       ttl,
		Records:   []record{{Content: ip.String()}},
	}

	response, err := p.doRequest(ctx, client, method, requestData)
	if err != nil {
		return err
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return handleErrorResponse(response)
	}

	return nil
}
