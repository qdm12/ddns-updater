package aliyun

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"

	"github.com/qdm12/ddns-updater/internal/settings/constants"
	"github.com/qdm12/ddns-updater/internal/settings/errors"
	"github.com/qdm12/ddns-updater/internal/settings/utils"
)

func (p *Provider) updateRecord(ctx context.Context, client *http.Client,
	recordID string, ip net.IP) (err error) {
	recordType := constants.A
	if ip.To4() == nil {
		recordType = constants.AAAA
	}

	u := &url.URL{
		Scheme: "https",
		Host:   "alidns.aliyuncs.com",
	}
	values := newURLValues(p.accessKeyID)
	values.Set("Action", "UpdateDomainRecord")
	values.Set("RecordId", recordID)
	values.Set("RR", p.host)
	values.Set("Type", recordType)
	values.Set("Value", ip.String())

	sign(http.MethodGet, values, p.accessSecret)

	u.RawQuery = values.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return fmt.Errorf("%w: %w", errors.ErrBadRequest, err)
	}

	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	switch response.StatusCode {
	case http.StatusOK:
	default:
		return fmt.Errorf("%w: %d: %s",
			errors.ErrBadHTTPStatus, response.StatusCode,
			utils.BodyToSingleLine(response.Body))
	}

	return nil
}
