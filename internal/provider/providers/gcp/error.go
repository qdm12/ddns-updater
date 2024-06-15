package gcp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/qdm12/ddns-updater/internal/provider/utils"
)

type gcpErrorData struct {
	gcpError `json:"error"`
}

type gcpError struct {
	// Code is the HTTP response status code and will always be populated.
	Code int `json:"code"`
	// Message is the server response message and is only populated when
	// explicitly referenced by the JSON server response.
	Message string `json:"message"`
	// Details can be JSON encoded for readable details.
	Details []any       `json:"details"`
	Errors  []errorItem `json:"errors"`
}

// errorItem is a detailed error code & message from the Google API frontend.
type errorItem struct {
	// Reason is the typed error code. For example: "some_example".
	Reason string `json:"reason"`
	// Message is the human-readable description of the error.
	Message string `json:"message"`
}

func (e gcpError) String() string {
	elements := make([]string, 0, 3+len(e.Errors)) //nolint:gomnd
	if e.Code != 0 {
		element := fmt.Sprintf("status %d", e.Code)
		elements = append(elements, element)
	}
	if e.Message != "" {
		elements = append(elements, e.Message)
	}

	if len(e.Details) > 0 {
		buffer := bytes.NewBuffer(nil)
		encoder := json.NewEncoder(buffer)
		err := encoder.Encode(e.Details)
		if err == nil {
			elements = append(elements, "details: "+buffer.String())
		}
	}

	for _, errorItem := range e.Errors {
		element := "reason: " + errorItem.Reason
		if errorItem.Message != "" && errorItem.Message != e.Message {
			element += ", message: " + errorItem.Message
		}
		elements = append(elements, element)
	}

	if len(elements) == 0 {
		return "[no error details]"
	}

	return strings.Join(elements, "; ")
}

func decodeError(body io.ReadCloser) (message string) {
	b, err := io.ReadAll(body)
	if err != nil {
		return "reading body: " + err.Error()
	}
	var jsonErrReply gcpErrorData
	err = json.Unmarshal(b, &jsonErrReply)
	_ = body.Close()
	if err != nil {
		return "failed JSON decoding body: " + err.Error() + ": " + utils.ToSingleLine(string(b))
	}
	return jsonErrReply.String()
}
