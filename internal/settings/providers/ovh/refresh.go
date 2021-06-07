package ovh

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/qdm12/ddns-updater/internal/settings/errors"
)

func (p *provider) refresh(ctx context.Context, client *http.Client, timestamp int64) (err error) {
	u := url.URL{
		Scheme: p.apiURL.Scheme,
		Host:   p.apiURL.Host,
		Path:   p.apiURL.Path + "/domain/zone/" + p.domain + "/refresh",
	}

	p.logger.Debug("HTTP POST: " + u.String())

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), nil)
	if err != nil {
		return fmt.Errorf("%w: %s", errors.ErrBadRequest, err)
	}
	p.setHeaderCommon(request.Header)
	p.setHeaderAuth(request.Header, timestamp, request.Method, request.URL, nil)

	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("%w: %s", errors.ErrUnsuccessfulResponse, err)
	}

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return extractAPIError(response)
	}

	_ = response.Body.Close()

	return nil
}
