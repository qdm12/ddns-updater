// Package reload provides the service reloading the DNS records
// configuration and the runtime settings, without needing to restart
// the application.
package reload

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/qdm12/ddns-updater/internal/data"
	"github.com/qdm12/ddns-updater/internal/params"
	"github.com/qdm12/ddns-updater/internal/persistence/json"
	"github.com/qdm12/ddns-updater/internal/provider"
	librecords "github.com/qdm12/ddns-updater/internal/records"
	"github.com/qdm12/ddns-updater/internal/settings"
	"github.com/qdm12/ddns-updater/internal/shoutrrr"
)

// SettingsUpdater can have its period and cooldown modified at runtime.
type SettingsUpdater interface {
	SetPeriod(period time.Duration)
	SetCooldown(cooldown time.Duration)
}

// Logger is the logger interface used by the service.
type Logger interface {
	Info(s string)
	Debug(s string)
	Warn(s string)
	Error(s string)
}

// Service reloads the records configuration file and the settings file.
type Service struct {
	configFilePath   string
	settingsFilePath string
	jsonReader       *params.Reader
	persistentDB     *json.Database
	db               *data.Database
	updater          SettingsUpdater
	logger           Logger
	shoutrrrClient   *shoutrrr.Client
	mutex            sync.Mutex
}

// Settings are the settings to create a new Service.
type Settings struct {
	ConfigFilePath   string
	SettingsFilePath string
	JSONReader       *params.Reader
	PersistentDB     *json.Database
	Database         *data.Database
	Updater          SettingsUpdater
	Logger           Logger
	ShoutrrrClient   *shoutrrr.Client
}

// New creates a new reload Service.
func New(settings Settings) *Service {
	return &Service{
		configFilePath:   settings.ConfigFilePath,
		settingsFilePath: settings.SettingsFilePath,
		jsonReader:       settings.JSONReader,
		persistentDB:     settings.PersistentDB,
		db:               settings.Database,
		updater:          settings.Updater,
		logger:           settings.Logger,
		shoutrrrClient:   settings.ShoutrrrClient,
	}
}

// Result is the result of a reload.
type Result struct {
	// RecordCount is the number of records loaded from the configuration.
	RecordCount int
	// Period is the period between two public IP address checks.
	Period time.Duration
	// Cooldown is the cooldown between two updates of a record.
	Cooldown time.Duration
	// PeriodSet is true if the period was found in the settings file.
	PeriodSet bool
	// CooldownSet is true if the cooldown was found in the settings file.
	CooldownSet bool
}

// Reload re-reads the records configuration file and the settings file.
// If the configuration file cannot be read or parsed, the records are
// left untouched and an error is returned.
func (s *Service) Reload(_ context.Context) (result Result, err error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.reloadLocked()
}

// reloadLocked re-reads the records configuration file and the settings
// file. The caller must hold the service mutex.
func (s *Service) reloadLocked() (result Result, err error) {
	providers, warnings, err := s.jsonReader.JSONProviders(s.configFilePath)
	for _, warning := range warnings {
		s.logger.Warn(warning)
	}
	if err != nil {
		return result, fmt.Errorf("reading records configuration: %w", err)
	}

	newRecords, err := ReadRecords(providers, s.persistentDB, s.logger, s.shoutrrrClient)
	if err != nil {
		return result, fmt.Errorf("reading records: %w", err)
	}

	s.db.SetAll(newRecords)
	result.RecordCount = len(newRecords)
	s.logger.Info(fmt.Sprintf("Reloaded %d records from %s",
		result.RecordCount, s.configFilePath))

	settingsFile, err := settings.ReadFile(s.settingsFilePath)
	if err != nil {
		return result, fmt.Errorf("reading settings file: %w", err)
	}

	period, periodSet, err := settingsFile.Duration(settings.PeriodKey)
	if err != nil {
		return result, fmt.Errorf("reading period setting: %w", err)
	}
	cooldown, cooldownSet, err := settingsFile.Duration(settings.CooldownKey)
	if err != nil {
		return result, fmt.Errorf("reading cooldown setting: %w", err)
	}

	result.Period = period
	result.PeriodSet = periodSet
	result.Cooldown = cooldown
	result.CooldownSet = cooldownSet

	if periodSet {
		s.updater.SetPeriod(period)
		s.logger.Info("Period set to " + period.String())
	}
	if cooldownSet {
		s.updater.SetCooldown(cooldown)
		s.logger.Info("Cooldown set to " + cooldown.String())
	}

	return result, nil
}

// SetPeriod sets the period between two public IP address checks in the
// settings file and applies it to the updater.
// The period is persisted, so that it is kept across restarts.
func (s *Service) SetPeriod(period time.Duration) (err error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	settingsFile, err := settings.ReadFile(s.settingsFilePath)
	if err != nil {
		return fmt.Errorf("reading settings file: %w", err)
	}

	err = settingsFile.SetDuration(settings.PeriodKey, period)
	if err != nil {
		return fmt.Errorf("writing settings file: %w", err)
	}

	s.updater.SetPeriod(period)
	s.logger.Info("Period set to " + period.String())
	return nil
}

// ReadRecords builds the records for the providers given, restoring
// their history from the persistent database.
func ReadRecords(providers []provider.Provider, persistentDB *json.Database,
	logger Logger, shoutrrrClient *shoutrrr.Client,
) (records []librecords.Record, err error) {
	records = make([]librecords.Record, len(providers))
	for i, provider := range providers {
		logger.Info("Reading history from database: domain " +
			provider.Domain() + " owner " + provider.Owner() +
			" " + provider.IPVersion().String())
		events, err := persistentDB.GetEvents(provider.Domain(),
			provider.Owner(), provider.IPVersion())
		if err != nil {
			if shoutrrrClient != nil {
				shoutrrrClient.Notify(err.Error())
			}
			return nil, err
		}
		records[i] = librecords.New(provider, events)
	}
	return records, nil
}
