package hetznercloud

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/qdm12/ddns-updater/internal/provider/errors"
)

// waitAction polls a Zone action until it completes.
// The action is returned by Zone RRSet operations such as set_records, so it
// must be queried through the Zone actions endpoint and NOT the Server actions
// endpoint, otherwise the lookup returns 404 (action not found).
// See https://docs.hetzner.cloud/reference/cloud#zone-actions-get-an-action
func (p *Provider) waitAction(ctx context.Context, client *http.Client, id uint64) (err error) {
	if id == 0 {
		return fmt.Errorf("%w: action id is zero", errors.ErrReceivedNoResult)
	}

	// Zone actions (e.g. set_rrset_records) are processed asynchronously by
	// Hetzner and observed to take around 20 seconds to reach the success
	// status, so poll up to 12 times waiting p.actionPollPeriod (5 seconds by
	// default) between attempts, for up to 60 seconds in total.
	const tries = 12
	sleepDuration := p.actionPollPeriod
	for range tries {
		url := fmt.Sprintf("https://api.hetzner.cloud/v1/zones/actions/%d", id)
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return fmt.Errorf("creating http request: %w", err)
		}
		p.setHeaders(request)

		response, err := client.Do(request)
		if err != nil {
			return err
		}
		defer response.Body.Close()

		if response.StatusCode != http.StatusOK {
			return handleErrorResponse(response)
		}

		var parsed actionResponse
		decoder := json.NewDecoder(response.Body)
		err = decoder.Decode(&parsed)
		if err != nil {
			return fmt.Errorf("json decoding response body: %w", err)
		}

		action := parsed.Action
		switch action.Status {
		case "success":
			return nil
		case "running":
			timer := time.NewTimer(sleepDuration)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
		case "error":
			err = fmt.Errorf("%w: action id %d failed",
				errors.ErrUnsuccessful, action.ID)
			if action.Error != nil {
				err = fmt.Errorf("%w: %v", err, action.Error)
			}
			return err
		default:
			return fmt.Errorf("%w: unknown action status %q for action id %d",
				errors.ErrDNSServerSide, action.Status, action.ID)
		}
	}
	return fmt.Errorf("%w: action id %d did not complete after %d tries",
		errors.ErrUnsuccessful, id, tries)
}
