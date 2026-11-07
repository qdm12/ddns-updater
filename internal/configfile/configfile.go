// Package configfile reads and safely rewrites the JSON configuration file
// from the web UI.
package configfile

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
)

// MaxSize is the maximum size in bytes accepted for the configuration file.
const MaxSize = 1 << 20 // 1 MiB

var (
	// ErrTooLarge is returned if the content exceeds MaxSize.
	ErrTooLarge = errors.New("configuration is too large")
	// ErrInvalid wraps validation errors of the content.
	ErrInvalid = errors.New("configuration is invalid")
	// ErrFromEnvironment is returned when saving while the CONFIG environment
	// variable is set, since the file is overwritten from it at every start.
	ErrFromEnvironment = errors.New("the CONFIG environment variable is set, " +
		"so the file is overwritten from it at every start")
)

// Validator validates the configuration content and returns warnings.
type Validator func(content []byte) (warnings []string, err error)

// Store gives access to the configuration file.
type Store struct {
	path     string
	validate Validator
	// fromEnvironment tells if the configuration comes from the CONFIG
	// environment variable. It is a field so it can be mocked in tests.
	fromEnvironment func() bool
	mutex           sync.Mutex
}

// New creates a Store for the file at path, using validate to check
// the content before it is saved.
func New(path string, validate Validator) *Store {
	return &Store{
		path:     path,
		validate: validate,
		fromEnvironment: func() bool {
			return os.Getenv("CONFIG") != ""
		},
	}
}

// Path returns the path of the configuration file.
func (s *Store) Path() string { return s.path }

// FromEnvironment returns true if the configuration is provided by the CONFIG
// environment variable, in which case the file is overwritten at each start.
func (s *Store) FromEnvironment() bool { return s.fromEnvironment() }

// Read returns the current content of the configuration file.
func (s *Store) Read() (content []byte, err error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	file, err := os.Open(s.path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	content, err = io.ReadAll(io.LimitReader(file, MaxSize+1))
	if err != nil {
		return nil, err
	}
	return content, nil
}

// Validate checks the content without saving it.
func (s *Store) Validate(content []byte) (warnings []string, err error) {
	if len(content) > MaxSize {
		return nil, fmt.Errorf("%w: more than %d bytes", ErrTooLarge, MaxSize)
	}
	warnings, err = s.validate(content)
	if err != nil {
		return warnings, fmt.Errorf("%w: %w", ErrInvalid, err)
	}
	return warnings, nil
}

// Save validates the content and writes it to the configuration file.
// The previous content is kept in a ".bak" file next to it. Changes only
// apply once the program is restarted.
func (s *Store) Save(content []byte) (warnings []string, err error) {
	if s.fromEnvironment() {
		return nil, ErrFromEnvironment
	}

	warnings, err = s.Validate(content)
	if err != nil {
		return warnings, err
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	mode := fs.FileMode(0o600)
	previous, err := os.ReadFile(s.path)
	switch {
	case err == nil:
		info, statErr := os.Stat(s.path)
		if statErr == nil {
			mode = info.Mode().Perm()
		}
		// Backups hold the same secrets as the file, keep them private.
		err = os.WriteFile(s.path+".bak", previous, 0o600)
		if err != nil {
			return warnings, fmt.Errorf("writing backup: %w", err)
		}
	case errors.Is(err, fs.ErrNotExist):
	default:
		return warnings, fmt.Errorf("reading current file: %w", err)
	}

	err = writeFile(s.path, content, mode)
	if err != nil {
		return warnings, fmt.Errorf("writing file: %w", err)
	}
	return warnings, nil
}

// writeFile replaces the file atomically if possible, so a crash cannot leave
// a half written configuration. If that is not possible (for example the
// file is a Docker bind mount or its directory is read only), it falls back
// to rewriting the file in place.
func writeFile(path string, content []byte, mode fs.FileMode) (err error) {
	err = writeFileAtomic(path, content, mode)
	if err == nil {
		return nil
	}
	return os.WriteFile(path, content, mode)
}

func writeFileAtomic(path string, content []byte, mode fs.FileMode) (err error) {
	temp, err := os.CreateTemp(filepath.Dir(path), ".config-*.tmp")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer func() {
		if err != nil {
			_ = temp.Close()
			_ = os.Remove(tempPath)
		}
	}()

	_, err = temp.Write(content)
	if err != nil {
		return err
	}
	err = temp.Chmod(mode)
	if err != nil {
		return err
	}
	err = temp.Sync()
	if err != nil {
		return err
	}
	err = temp.Close()
	if err != nil {
		return err
	}
	return os.Rename(tempPath, path)
}
