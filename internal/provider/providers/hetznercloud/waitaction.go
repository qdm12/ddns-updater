package hetznercloud

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	ddnserrors "github.com/qdm12/ddns-updater/internal/provider/errors"
)

func (p *Provider) waitAction(ctx context.Context, client *http.Client, id uint64) (err error) {
	if id == 0 {
		return fmt.Errorf("%w: action id is zero", ddnserrors.ErrReceivedNoResult)
	}

	const sleepDuration = time.Second
	const tries = 3
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
			err = fmt.Errorf("unsuccessful result: action id %d failed", action.ID)
			if action.Error != nil {
				err = fmt.Errorf("%w: %v", err, action.Error)
			}
			return err
		default:
			return fmt.Errorf("server side DNS error: unknown action status %q for action id %d", action.Status, action.ID)
		}
	}
	return fmt.Errorf("unsuccessful result: action id %d did not complete after %d tries", id, tries)
}
