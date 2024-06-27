package aliyun

import (
	"context"
	"fmt"
	"net/http"
	"net/netip"
	"net/url"

	"github.com/qdm12/ddns-updater/internal/provider/constants"
	"github.com/qdm12/ddns-updater/internal/provider/errors"
	"github.com/qdm12/ddns-updater/internal/provider/utils"
)

func (p *Provider) updateRecord(ctx context.Context, client *http.Client,
	recordID string, ip netip.Addr) (err error) {
	recordType := constants.A
	if ip.Is6() {
		recordType = constants.AAAA
	}

	u := &url.URL{
		Scheme: "https",
		Host:   "alidns.aliyuncs.com",
	}
	values := newURLValues(p.accessKeyID)
	values.Set("Action", "UpdateDomainRecord")
	values.Set("RecordId", recordID)
	values.Set("RR", p.owner)
	values.Set("Type", recordType)
	values.Set("Value", ip.String())

	sign(http.MethodGet, values, p.accessSecret)

	u.RawQuery = values.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return fmt.Errorf("creating http request: %w", err)
	}
	setHeaders(request)

	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	switch response.StatusCode {
	case http.StatusOK:
	default:
		return fmt.Errorf("%w: %d: %s",
			errors.ErrHTTPStatusNotValid, response.StatusCode,
			utils.BodyToSingleLine(response.Body))
	}

	return nil
}
