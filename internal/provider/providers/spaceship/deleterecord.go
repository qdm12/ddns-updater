package spaceship

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/qdm12/ddns-updater/internal/provider/errors"
	"github.com/qdm12/ddns-updater/internal/provider/utils"
)

func (p *Provider) deleteRecord(ctx context.Context, client *http.Client, record Record) error {
	u := url.URL{
		Scheme: "https",
		Host:   "spaceship.dev",
		Path:   fmt.Sprintf("/api/v1/dns/records/%s", p.domain),
	}

	deleteData := []Record{record}

	var requestBody bytes.Buffer
	if err := json.NewEncoder(&requestBody).Encode(deleteData); err != nil {
		return fmt.Errorf("encoding request body: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodDelete, u.String(), &requestBody)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	p.setHeaders(request)

	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusNoContent {
		return fmt.Errorf("%w: %d: %s",
			errors.ErrHTTPStatusNotValid, response.StatusCode,
			utils.BodyToSingleLine(response.Body))
	}

	return nil
}
