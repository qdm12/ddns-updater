package namecom

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"strings"

	ddnserrors "github.com/qdm12/ddns-updater/internal/provider/errors"
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
		return wrapErrorAndDetails(ddnserrors.ErrRecordNotFound, errorResponse.Details)
	case "permission denied", "unauthenticated":
		return wrapErrorAndDetails(errors.New("bad authentication"), errorResponse.Details)
	case "invalid argument":
		return wrapErrorAndDetails(errors.New("bad request sent"), errorResponse.Details)
	}

	return fmt.Errorf("unknown response received: %s: %s (status code %d)",
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
		return fmt.Errorf("malformed IP address received: %s", responseData.Answer)
	} else if sentIP.Compare(receivedIP) != 0 {
		return fmt.Errorf("mismatching IP address received: sent ip %s to update but received %s", sentIP, receivedIP)
	}
	return nil
}

func wrapErrorAndDetails(sentinelErr error, details string) (wrappedErr error) {
	if details == "" {
		return fmt.Errorf("%w", sentinelErr)
	}
	return fmt.Errorf("%w: %s", sentinelErr, details)
}
