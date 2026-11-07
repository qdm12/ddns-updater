package ovh

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func extractAPIError(response *http.Response) (err error) {
	b, err := io.ReadAll(response.Body)
	if err != nil {
		_ = response.Body.Close()
		return fmt.Errorf("reading response body: %w", err)
	}

	var apiError struct {
		Message string `json:"Message"`
	}
	err = json.Unmarshal(b, &apiError)
	if err != nil {
		apiError.Message = string(b)
	}
	queryID := response.Header.Get("X-Ovh-Queryid")

	_ = response.Body.Close()

	return fmt.Errorf("HTTP status is not valid: %s: %s: for query ID: %s", response.Status, apiError.Message, queryID)
}
