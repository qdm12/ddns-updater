package utils

import (
	"fmt"
	"io"
	"strings"
)

// ReadAndCleanBody reads the body, closes it, trims spaces from the body data
// and converts it to lowercase.
func ReadAndCleanBody(body io.ReadCloser) (cleanedBody string, err error) {
	b, err := io.ReadAll(body)
	if err != nil {
		return "", fmt.Errorf("reading body: %w", err)
	}
	err = body.Close()
	if err != nil {
		return "", fmt.Errorf("closing body: %w", err)
	}

	cleanedBody = string(b)
	cleanedBody = strings.TrimSpace(cleanedBody)
	cleanedBody = strings.ToLower(cleanedBody)

	return cleanedBody, nil
}
