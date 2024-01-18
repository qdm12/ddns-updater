package porkbun

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/qdm12/ddns-updater/internal/provider/utils"
)

func makeErrorMessage(body io.Reader) (message string) {
	bytes, err := io.ReadAll(body)
	if err != nil {
		return "failed to read response body: " + err.Error()
	}

	var errorResponse struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	}
	err = json.Unmarshal(bytes, &errorResponse)
	if err != nil { // the encoding may change in the future
		return utils.ToSingleLine(string(bytes))
	}

	if errorResponse.Status != "ERROR" {
		return fmt.Sprintf("status %q is not expected ERROR: message is: %s",
			errorResponse.Status, errorResponse.Message)
	}

	return errorResponse.Message
}
