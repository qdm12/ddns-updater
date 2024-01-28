package dondominio

import (
	"context"
	"net/http"
	"net/netip"

	"github.com/qdm12/ddns-updater/internal/provider/constants"
)

// See https://dondominio.dev/en/api/docs/api/#dns-zone-create-service-dnscreate
func (p *Provider) create(ctx context.Context, client *http.Client, ip netip.Addr) (err error) {
	recordType := constants.A
	if ip.Is6() {
		recordType = constants.AAAA
	}
	requestData := struct {
		APIUser     string `json:"apiuser"`
		APIPasswd   string `json:"apipasswd"`
		ServiceName string `json:"serviceName"`
		Name        string `json:"name"` // Name for the DNS zone
		Type        string `json:"type"`
		Value       string `json:"value"`
	}{
		APIUser:     p.username,
		APIPasswd:   p.password,
		ServiceName: p.name,
		Name:        p.BuildDomainName(),
		Type:        recordType,
		Value:       ip.String(),
	}

	_, err = apiCall(ctx, client, "/service/dnscreate", requestData)
	if err != nil {
		return err
	}

	return nil
}
