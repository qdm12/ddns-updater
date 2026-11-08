package server

import (
	"context"
	"time"

	"github.com/qdm12/ddns-updater/internal/records"
	"github.com/qdm12/ddns-updater/internal/reload"
)

type Database interface {
	SelectAll() (records []records.Record)
}

type UpdateForcer interface {
	ForceUpdate(ctx context.Context) (errors []error)
}

// ConfigReloader reloads the records configuration and the settings.
type ConfigReloader interface {
	// Reload re-reads the configuration and settings files.
	Reload(ctx context.Context) (result reload.Result, err error)
	// SetPeriod sets the period between two public IP address checks
	// and persists it in the settings file.
	SetPeriod(period time.Duration) (err error)
}

type Logger interface {
	Info(s string)
	Warn(s string)
	Error(s string)
}
