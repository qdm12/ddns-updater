package namecom

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"strings"

	"github.com/qdm12/ddns-updater/internal/provider/errors"
)

func parseErrorResponse(response *http.Response) (err error) {
	var errorResponse struct {
		Message string `json:"message"`
		Details string `json:"details"`
	}
	decoder := json.NewDecoder(response.Body)
	err = decoder.Decode(&errorResponse)
	if err != nil {
		return fmt.Errorf("json decoding error message from response body: %w", err)
	}

	switch strings.ToLower(errorResponse.Message) {
	case "not found":
		return wrapErrorAndDetails(errors.ErrRecordNotFound, errorResponse.Details)
	case "permission denied", "unauthenticated":
		return wrapErrorAndDetails(errors.ErrAuth, errorResponse.Details)
	case "invalid argument":
		return wrapErrorAndDetails(errors.ErrBadRequest, errorResponse.Details)
	}

	return fmt.Errorf("%w: %s: %s (status code %d)", errors.ErrUnknownResponse,
		errorResponse.Message, errorResponse.Details, response.StatusCode)
}

func verifySuccessResponseBody(responseBody io.ReadCloser, sentIP netip.Addr) (err error) {
	decoder := json.NewDecoder(responseBody)
	var responseData struct {
		Answer string `json:"answer"`
	}
	err = decoder.Decode(&responseData)
	if err != nil {
		return fmt.Errorf("json decoding response body: %w", err)
	}

	receivedIP, err := netip.ParseAddr(responseData.Answer)
	if err != nil {
		return fmt.Errorf("%w: %s", errors.ErrIPReceivedMalformed, responseData.Answer)
	} else if sentIP.Compare(receivedIP) != 0 {
		return fmt.Errorf("%w: sent ip %s to update but received %s",
			errors.ErrIPReceivedMismatch, sentIP, receivedIP)
	}
	return nil
}

func wrapErrorAndDetails(sentinelErr error, details string) (wrappedErr error) {
	if details == "" {
		return fmt.Errorf("%w", sentinelErr)
	}
	return fmt.Errorf("%w: %s", sentinelErr, details)
}
