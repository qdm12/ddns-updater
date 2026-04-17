package hetznercloud

import (
	"fmt"
	"strings"
)

// recordValue represents a single DNS record value.
type recordValue struct {
	Value string `json:"value"`
}

// recordsRequest represents the request body for creating/updating DNS records.
type recordsRequest struct {
	TTL     uint32        `json:"ttl,omitempty"`
	Records []recordValue `json:"records"`
}

// actionResponse represents the response from Hetzner Cloud API actions.
type actionResponse struct {
	Action struct {
		ID     uint64 `json:"id"`
		Status string `json:"status"`
	} `json:"action"`
}

type errorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Details any    `json:"details,omitempty"`
	} `json:"error"`
}

func (e *errorResponse) String() string {
	const maxParts = 3
	parts := make([]string, 0, maxParts)
	parts = append(parts, "code: "+e.Error.Code)
	parts = append(parts, "message: "+e.Error.Message)
	if e.Error.Details != nil {
		parts = append(parts, fmt.Sprintf("details: %v", e.Error.Details))
	}
	return strings.Join(parts, ": ")
}
