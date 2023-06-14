package ovh

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/qdm12/ddns-updater/internal/provider/errors"
)

func (p *Provider) getAdjustedUnixTimestamp(ctx context.Context, client *http.Client) (
	unix int64, err error) {
	delta, err := p.getTimeDelta(ctx, client)
	if err != nil {
		return 0, err
	}

	now := p.timeNow()
	adjustedTime := now.Add(-delta)
	return adjustedTime.Unix(), nil
}

// getTimeDelta obtains the delta between the OVH server time and this machine time.
// If it is the first time executing, it fetches the time from OVH servers to calculate the
// delta. Otherwise, it uses the delta calculated previously.
func (p *Provider) getTimeDelta(ctx context.Context, client *http.Client) (delta time.Duration, err error) {
	if p.serverDelta > 0 {
		return p.serverDelta, nil
	}

	ovhTime, err := p.getOVHTime(ctx, client)
	if err != nil {
		return 0, err
	}

	now := p.timeNow()
	p.serverDelta = now.Sub(ovhTime) // server delta should not change
	return p.serverDelta, nil
}

func (p *Provider) getOVHTime(ctx context.Context, client *http.Client) (ovhTime time.Time, err error) {
	u := url.URL{
		Scheme: p.apiURL.Scheme,
		Host:   p.apiURL.Host,
		Path:   p.apiURL.Path + "/auth/time",
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return ovhTime, fmt.Errorf("%w: %w", errors.ErrBadRequest, err)
	}
	p.setHeaderCommon(request.Header)

	response, err := client.Do(request)
	if err != nil {
		return ovhTime, err
	}

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return ovhTime, extractAPIError(response)
	}

	decoder := json.NewDecoder(response.Body)
	decoder.UseNumber() // from OVH's Go library, not sure why though

	var unixTimestamp int64
	err = decoder.Decode(&unixTimestamp)
	if err != nil {
		return ovhTime, fmt.Errorf("%w: %w", errors.ErrUnmarshalResponse, err)
	}

	err = response.Body.Close()
	if err != nil {
		return ovhTime, err
	}

	return time.Unix(unixTimestamp, 0), nil
}
