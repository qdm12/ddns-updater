package server

import (
	"context"

	"github.com/qdm12/ddns-updater/internal/records"
	"github.com/qdm12/ddns-updater/pkg/publicip"
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

// PublicIPChecker queries the public IP addresses with each enabled method.
type PublicIPChecker interface {
	Check(ctx context.Context) (results []publicip.CheckResult)
}

// ConfigStore gives access to the JSON configuration file.
type ConfigStore interface {
	Path() string
	FromEnvironment() bool
	Read() (content []byte, err error)
	Validate(content []byte) (warnings []string, err error)
	Save(content []byte) (warnings []string, err error)
}
