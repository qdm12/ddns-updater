package server

import (
	"context"
	"encoding/json"
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

// EntryManager lists, saves and deletes the settings entries of the
// configuration file, and applies them to the running records.
type EntryManager interface {
	// ConfigPath returns the path of the configuration file.
	ConfigPath() string
	// EditingDisabledReason returns a non empty reason when the
	// configuration cannot be edited from the web interface.
	EditingDisabledReason() string
	// Entries returns the settings entries of the configuration file.
	Entries() (entries []reload.Entry, err error)
	// SaveEntry validates and writes a settings entry, then applies it.
	// A nil index appends a new entry, a non nil index replaces an
	// existing entry.
	SaveEntry(ctx context.Context, index *int, entry json.RawMessage) (
		result reload.Result, err error)
	// DeleteEntry removes the settings entry at the given index, then
	// applies the change.
	DeleteEntry(ctx context.Context, index int) (result reload.Result, err error)
}

type Logger interface {
	Info(s string)
	Warn(s string)
	Error(s string)
}
