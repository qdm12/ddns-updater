package reload

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/qdm12/ddns-updater/internal/configfile"
	"github.com/qdm12/ddns-updater/internal/params"
)

var (
	// ErrEditingDisabled is returned when the configuration cannot be edited,
	// for example because it is set from an environment variable.
	ErrEditingDisabled = errors.New("editing the configuration is disabled")
	// ErrEntryNotFound is returned when a settings entry index does not exist.
	ErrEntryNotFound = errors.New("settings entry not found")
	// ErrInvalidEntry is returned when a settings entry is not valid.
	ErrInvalidEntry = params.ErrInvalidEntry
)

// Entry is one settings entry of the configuration file.
type Entry struct {
	// Index is the position of the entry in the settings array.
	Index int
	// Provider is the provider name, for example "cloudflare".
	Provider string
	// Domain is the domain field of the entry.
	Domain string
	// IPVersion is the ip_version field of the entry, if set.
	IPVersion string
	// IPv6Suffix is the ipv6_suffix field of the entry, if set.
	IPv6Suffix string
	// Raw is the JSON content of the entry.
	Raw json.RawMessage
}

// ConfigPath returns the path of the configuration file.
func (s *Service) ConfigPath() string {
	return s.configFilePath
}

// EditingDisabledReason returns a non empty reason when the configuration
// cannot be edited from the web UI.
func (s *Service) EditingDisabledReason() string {
	if os.Getenv("CONFIG") != "" {
		return "The CONFIG environment variable is set, so changes written to " +
			"the configuration file would be ignored on the next reload. " +
			"Unset CONFIG and restart the program to edit the configuration."
	}
	return ""
}

// Entries returns the settings entries of the configuration file.
func (s *Service) Entries() (entries []Entry, err error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.readEntries()
}

func (s *Service) readEntries() (entries []Entry, err error) {
	file := configfile.NewFile(s.configFilePath)
	document, err := file.Read()
	if err != nil {
		return nil, err
	}

	rawEntries := document.Entries()
	entries = make([]Entry, len(rawEntries))
	for i, raw := range rawEntries {
		var settings struct {
			Provider   string `json:"provider"`
			Domain     string `json:"domain"`
			IPVersion  string `json:"ip_version"`
			IPv6Suffix string `json:"ipv6_suffix"`
		}
		err = json.Unmarshal(raw, &settings)
		if err != nil {
			return nil, fmt.Errorf("decoding settings entry %d: %w", i, err)
		}
		entries[i] = Entry{
			Index:      i,
			Provider:   settings.Provider,
			Domain:     settings.Domain,
			IPVersion:  settings.IPVersion,
			IPv6Suffix: settings.IPv6Suffix,
			Raw:        raw,
		}
	}
	return entries, nil
}

// SaveEntry validates and writes a settings entry, then reloads the
// records so that the change applies without restarting the program.
// A nil index appends a new entry, a non nil index replaces an existing one.
func (s *Service) SaveEntry(_ context.Context, index *int,
	entry json.RawMessage,
) (result Result, err error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	err = s.checkEditable()
	if err != nil {
		return result, err
	}

	entry, err = normalizeEntry(entry)
	if err != nil {
		return result, err
	}

	_, warnings, err := params.ValidateEntry(entry)
	for _, warning := range warnings {
		s.logger.Warn(warning)
	}
	if err != nil {
		return result, err
	}

	file := configfile.NewFile(s.configFilePath)
	document, err := file.Read()
	if err != nil {
		return result, err
	}

	previousData, previousReadErr := os.ReadFile(s.configFilePath)

	switch {
	case index == nil:
		document.Add(entry)
	case *index < 0 || *index >= document.Len():
		return result, fmt.Errorf("%w: index %d", ErrEntryNotFound, *index)
	default:
		err = document.Set(*index, entry)
		if err != nil {
			return result, err
		}
	}

	err = file.Write(document)
	if err != nil {
		return result, err
	}

	result, err = s.reloadLocked()
	if err != nil {
		s.restoreConfig(previousData, previousReadErr)
		return Result{}, fmt.Errorf("applying configuration: %w", err)
	}
	return result, nil
}

// DeleteEntry removes a settings entry, then reloads the records so that
// the change applies without restarting the program.
func (s *Service) DeleteEntry(_ context.Context, index int) (result Result, err error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	err = s.checkEditable()
	if err != nil {
		return result, err
	}

	file := configfile.NewFile(s.configFilePath)
	document, err := file.Read()
	if err != nil {
		return result, err
	}

	previousData, previousReadErr := os.ReadFile(s.configFilePath)

	err = document.Delete(index)
	if err != nil {
		return result, fmt.Errorf("%w: index %d", ErrEntryNotFound, index)
	}

	err = file.Write(document)
	if err != nil {
		return result, err
	}

	result, err = s.reloadLocked()
	if err != nil {
		s.restoreConfig(previousData, previousReadErr)
		return Result{}, fmt.Errorf("applying configuration: %w", err)
	}
	return result, nil
}

func (s *Service) checkEditable() error {
	reason := s.EditingDisabledReason()
	if reason != "" {
		return fmt.Errorf("%w: %s", ErrEditingDisabled, reason)
	}
	return nil
}

// normalizeEntry checks the entry is a JSON object and compacts it.
func normalizeEntry(entry json.RawMessage) (json.RawMessage, error) {
	var object map[string]json.RawMessage
	err := json.Unmarshal(entry, &object)
	if err != nil {
		return nil, fmt.Errorf("%w: entry is not a JSON object: %w", ErrInvalidEntry, err)
	}

	compacted, err := json.Marshal(object)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidEntry, err)
	}
	return compacted, nil
}

// restoreConfig restores the configuration file bytes saved before a write,
// when applying the written configuration failed.
func (s *Service) restoreConfig(data []byte, readErr error) {
	if readErr != nil {
		s.logger.Error("cannot restore the previous configuration file: " + readErr.Error())
		return
	}

	document, err := configfile.Parse(data)
	if err != nil {
		s.logger.Error("cannot restore the previous configuration file: " + err.Error())
		return
	}

	err = configfile.NewFile(s.configFilePath).Write(document)
	if err != nil {
		s.logger.Error("cannot restore the previous configuration file: " + err.Error())
	}
}
