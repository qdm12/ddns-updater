package hetznercloud

import (
	"fmt"
	"strings"
)

type record struct {
	Value string `json:"value"`
}

type actionResponse struct {
	Action struct {
		ID     uint64 `json:"id"`
		Status string `json:"status"`
		Error  any    `json:"error,omitempty"`
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
