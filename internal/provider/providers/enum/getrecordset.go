package enum

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"
)

// See https://docs.enum.co/api/operations/dnsservice_getrecordset/
func (p *Provider) getRecordSet(ctx context.Context, client *http.Client, zoneID, recordType string,
	ip netip.Addr,
) (exists, upToDate bool, ttl uint32, err error) {
	requestData := struct {
		ProjectID string `json:"projectId"`
		ZoneID    string `json:"zoneId"`
		Name      string `json:"name"`
		Type      string `json:"type"`
	}{
		ProjectID: p.projectID,
		ZoneID:    zoneID,
		Name:      p.recordName(),
		Type:      recordType,
	}

	response, err := p.doRequest(ctx, client, "GetRecordSet", requestData)
	if err != nil {
		return false, false, 0, err
	}

	defer response.Body.Close()

	switch response.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound:
		return false, false, 0, nil
	default:
		return false, false, 0, handleErrorResponse(response)
	}

	decoder := json.NewDecoder(response.Body)

	var responseData struct {
		RecordSet struct {
			TTL     uint32 `json:"ttl"`
			Records []struct {
				Content  string `json:"content"`
				Disabled bool   `json:"disabled"`
			} `json:"records"`
		} `json:"recordSet"`
	}

	err = decoder.Decode(&responseData)
	if err != nil {
		return true, false, 0, fmt.Errorf("json decoding response body: %w", err)
	}

	ttl = responseData.RecordSet.TTL
	if ttl != 0 {
		p.knownTTL.Store(ttl)
	}

	records := responseData.RecordSet.Records
	if len(records) != 1 || records[0].Disabled {
		return true, false, ttl, nil
	}

	recordIP, err := netip.ParseAddr(records[0].Content)
	upToDate = err == nil && recordIP == ip

	return true, upToDate, ttl, nil
}
