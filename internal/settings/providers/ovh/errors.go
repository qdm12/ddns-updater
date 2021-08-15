package ovh

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/qdm12/ddns-updater/internal/settings/errors"
)

func extractAPIError(response *http.Response) (err error) {
	decoder := json.NewDecoder(response.Body)
	var apiError struct {
		Message string
	}
	if err := decoder.Decode(&apiError); err != nil {
		b, err := io.ReadAll(response.Body)
		if err != nil {
			_ = response.Body.Close()
			return fmt.Errorf("%w: %s", errors.ErrUnmarshalResponse, err)
		}
		apiError.Message = string(b)
	}
	queryID := response.Header.Get("X-Ovh-QueryID")

	_ = response.Body.Close()

	return fmt.Errorf("%w: %s: %s: for query ID: %s",
		errors.ErrBadHTTPStatus, response.Status, apiError.Message, queryID)
}
