// Package settings provides a JSON settings file which can be re-read
// at runtime, contrary to environment variables which are fixed for the
// lifetime of a process.
//
// The file contains a flat JSON object of key to value pairs, where the
// keys are the same as the environment variable names, for example:
//
//	{
//	  "PERIOD": "1m",
//	  "UPDATE_COOLDOWN_PERIOD": "1m"
//	}
package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	// PeriodKey is the key for the period between two public IP address checks.
	PeriodKey = "PERIOD"
	// CooldownKey is the key for the cooldown between two updates of a record.
	CooldownKey = "UPDATE_COOLDOWN_PERIOD"
)

var ErrInvalidValue = errors.New("invalid value")

// File is a JSON settings file with a flat map of keys to values.
type File struct {
	path   string
	values map[string]string
}

// ReadFile reads the settings file at the given path.
// If the file does not exist, an empty settings file is returned
// without an error, since the file is optional.
func ReadFile(path string) (file *File, err error) {
	file = &File{path: path, values: make(map[string]string)}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return file, nil
		}
		return nil, fmt.Errorf("reading settings file: %w", err)
	}

	if len(strings.TrimSpace(string(data))) == 0 {
		return file, nil
	}

	var values map[string]string
	err = json.Unmarshal(data, &values)
	if err != nil {
		return nil, fmt.Errorf("%w: decoding settings file %s: %w", ErrInvalidValue, path, err)
	}

	for key, value := range values {
		file.values[strings.ToUpper(strings.TrimSpace(key))] = value
	}

	return file, nil
}

// Path returns the path of the settings file.
func (f *File) Path() string {
	return f.path
}

// Duration returns the duration set for the given key.
// The returned boolean is false if the key is not set in the file.
func (f *File) Duration(key string) (duration time.Duration, set bool, err error) {
	value, set := f.values[strings.ToUpper(strings.TrimSpace(key))]
	if !set {
		return 0, false, nil
	}

	duration, err = time.ParseDuration(strings.TrimSpace(value))
	if err != nil {
		return 0, true, fmt.Errorf("%w: parsing duration for %s: %w", ErrInvalidValue, key, err)
	}

	return duration, true, nil
}

// SetDuration sets the duration for the given key and writes the
// settings file back to disk, so that the value is kept across restarts.
func (f *File) SetDuration(key string, duration time.Duration) (err error) {
	f.values[strings.ToUpper(strings.TrimSpace(key))] = duration.String()

	data, err := json.MarshalIndent(f.values, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding settings file: %w", err)
	}

	err = os.MkdirAll(filepath.Dir(f.path), 0o755)
	if err != nil {
		return fmt.Errorf("creating settings file directory: %w", err)
	}

	err = os.WriteFile(f.path, data, 0o644)
	if err != nil {
		return fmt.Errorf("writing settings file: %w", err)
	}

	return nil
}

// Values returns a copy of all the key to values pairs in the file.
func (f *File) Values() (values map[string]string) {
	values = make(map[string]string, len(f.values))
	for key, value := range f.values {
		values[key] = value
	}
	return values
}
