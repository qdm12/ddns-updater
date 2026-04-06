package server

import (
	"context"

	"github.com/qdm12/ddns-updater/internal/records"
)

type Database interface {
	SelectAll() (records []records.Record)
}

type UpdateForcer interface {
	ForceUpdate(ctx context.Context) (errors []error)
}

type Logger interface {
	Info(s string)
	Warn(s string)
	Error(s string)
}

// StatusRecord holds JSON-serializable record status for the API.
type StatusRecord struct {
	Domain      string   `json:"domain"`
	Owner       string   `json:"owner"`
	Provider    string   `json:"provider"`
	IPVersion   string   `json:"ip_version"`
	Status      string   `json:"status"`
	Message     string   `json:"message"`
	CurrentIP   string   `json:"current_ip"`
	PreviousIPs []string `json:"previous_ips"`
	LastUpdated string   `json:"last_updated"`
}
