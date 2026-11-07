package hetznercloud

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/qdm12/ddns-updater/internal/provider/headers"
	"github.com/qdm12/ddns-updater/internal/provider/utils"
)

func (p *Provider) setHeaders(request *http.Request) {
	headers.SetUserAgent(request)
	headers.SetContentType(request, "application/json")
	headers.SetAccept(request, "application/json")
	headers.SetAuthBearer(request, p.token)
}

func (p *Provider) handleActionResponse(ctx context.Context, client *http.Client, parsed actionResponse) (err error) {
	switch parsed.Action.Status {
	case "success":
		return nil
	case "running":
		err = p.waitAction(ctx, client, parsed.Action.ID)
		if err != nil {
			return fmt.Errorf("waiting for action to complete: %w", err)
		}
		return nil
	case "error":
		err = fmt.Errorf("unsuccessful result: action id %d failed", parsed.Action.ID)
		if parsed.Action.Error != nil {
			err = fmt.Errorf("%w: %v", err, parsed.Action.Error)
		}
		return err
	default:
		return fmt.Errorf("unknown response received: unexpected action status %q for action id %d",
			parsed.Action.Status, parsed.Action.ID)
	}
}

func handleErrorResponse(response *http.Response) (err error) {
	if response.StatusCode < http.StatusBadRequest {
		panic("handleErrorResponse called with non error HTTP status code")
	}
	data, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("HTTP status is not valid: %d: unable to read response body: %w", response.StatusCode, err)
	}
	var parsed errorResponse
	err = json.Unmarshal(data, &parsed)
	if err != nil {
		return fmt.Errorf("HTTP status is not valid: %d: %s", response.StatusCode, utils.ToSingleLine(string(data)))
	}
	return fmt.Errorf("HTTP status is not valid: %d: %s", response.StatusCode, parsed)
}
