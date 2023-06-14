package ovh

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/qdm12/ddns-updater/internal/provider/errors"
)

func (p *Provider) updateRecord(ctx context.Context, client *http.Client,
	recordID uint64, ipStr string, timestamp int64) (err error) {
	u := url.URL{
		Scheme: p.apiURL.Scheme,
		Host:   p.apiURL.Host,
		Path:   p.apiURL.Path + "/domain/zone/" + p.domain + "/record/" + strconv.Itoa(int(recordID)),
	}
	putRecordsParams := struct {
		Target string `json:"target"`
	}{
		Target: ipStr,
	}
	bodyBytes, err := json.Marshal(putRecordsParams)
	if err != nil {
		return fmt.Errorf("%w: %w", errors.ErrRequestMarshal, err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPut, u.String(), bytes.NewBuffer(bodyBytes))
	if err != nil {
		return fmt.Errorf("%w: %w", errors.ErrBadRequest, err)
	}
	request.Header.Add("Content-Type", "application/json;charset=utf-8")
	p.setHeaderCommon(request.Header)
	p.setHeaderAuth(request.Header, timestamp, request.Method, request.URL, bodyBytes)

	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("%w: %w", errors.ErrUnsuccessfulResponse, err)
	}

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return extractAPIError(response)
	}

	_ = response.Body.Close()

	return nil
}
