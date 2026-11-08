// Package configfile reads and writes the JSON records configuration file,
// preserving the parts of the file it does not manage, such as unknown
// top-level keys.
package configfile

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

var (
	// ErrIndexOutOfRange is returned when an entry index is out of range.
	ErrIndexOutOfRange = errors.New("entry index out of range")
	// ErrInvalidConfig is returned when the configuration file is not a JSON
	// object with a "settings" array of objects.
	ErrInvalidConfig = errors.New("invalid configuration file")
)

// Document is the content of a configuration file: a JSON object with a
// "settings" array of entry objects. Other top-level keys are preserved as
// they are.
type Document struct {
	root    map[string]json.RawMessage
	entries []json.RawMessage
}

// NewDocument returns an empty configuration document.
func NewDocument() Document {
	return Document{
		root:    make(map[string]json.RawMessage),
		entries: []json.RawMessage{},
	}
}

// Parse parses the JSON content of a configuration file.
func Parse(data []byte) (document Document, err error) {
	document = NewDocument()
	if len(data) == 0 {
		return document, nil
	}

	root := make(map[string]json.RawMessage)
	err = json.Unmarshal(data, &root)
	if err != nil {
		return Document{}, fmt.Errorf("%w: %w", ErrInvalidConfig, err)
	}

	settings, ok := root["settings"]
	if ok {
		entries := []json.RawMessage{}
		err = json.Unmarshal(settings, &entries)
		if err != nil {
			return Document{}, fmt.Errorf("%w: settings: %w", ErrInvalidConfig, err)
		}
		for i, entry := range entries {
			var object map[string]json.RawMessage
			err = json.Unmarshal(entry, &object)
			if err != nil {
				return Document{}, fmt.Errorf("%w: settings entry %d is not a JSON object: %w",
					ErrInvalidConfig, i, err)
			}
		}
		document.entries = entries
	}

	document.root = root
	return document, nil
}

// Entries returns the settings entries of the document.
func (d Document) Entries() (entries []json.RawMessage) {
	entries = make([]json.RawMessage, len(d.entries))
	copy(entries, d.entries)
	return entries
}

// Len returns the number of settings entries.
func (d Document) Len() int {
	return len(d.entries)
}

// Add appends a settings entry.
func (d *Document) Add(entry json.RawMessage) {
	d.entries = append(d.entries, entry)
}

// Set replaces the settings entry at the given index.
func (d *Document) Set(index int, entry json.RawMessage) error {
	if index < 0 || index >= len(d.entries) {
		return fmt.Errorf("%w: %d", ErrIndexOutOfRange, index)
	}
	d.entries[index] = entry
	return nil
}

// Delete removes the settings entry at the given index.
func (d *Document) Delete(index int) error {
	if index < 0 || index >= len(d.entries) {
		return fmt.Errorf("%w: %d", ErrIndexOutOfRange, index)
	}
	d.entries = append(d.entries[:index], d.entries[index+1:]...)
	return nil
}

// Marshal returns the JSON content of the document, indented with four
// spaces.
func (d Document) Marshal() (data []byte, err error) {
	settings, err := json.Marshal(d.entries)
	if err != nil {
		return nil, fmt.Errorf("encoding settings: %w", err)
	}

	root := make(map[string]json.RawMessage, len(d.root)+1)
	for key, value := range d.root {
		root[key] = value
	}
	root["settings"] = settings

	data, err = json.MarshalIndent(root, "", "    ")
	if err != nil {
		return nil, fmt.Errorf("encoding configuration: %w", err)
	}
	return append(data, '\n'), nil
}

// File is a configuration file on disk.
type File struct {
	path string
}

// NewFile returns a configuration file for the given path.
func NewFile(path string) *File {
	return &File{path: path}
}

// Path returns the path of the configuration file.
func (f *File) Path() string {
	return f.path
}

// Read reads the configuration file. A missing file is not an error and
// returns an empty document, since a configuration file is created on first
// write.
func (f *File) Read() (document Document, err error) {
	data, err := os.ReadFile(f.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return NewDocument(), nil
		}
		return Document{}, fmt.Errorf("reading configuration file: %w", err)
	}

	document, err = Parse(data)
	if err != nil {
		return Document{}, fmt.Errorf("parsing configuration file %s: %w", f.path, err)
	}
	return document, nil
}

// Write writes the document to the configuration file atomically, so that a
// crash never leaves a partially written file. The file permissions are
// kept when the file already exists.
func (f *File) Write(document Document) (err error) {
	data, err := document.Marshal()
	if err != nil {
		return err
	}

	directory := filepath.Dir(f.path)
	err = os.MkdirAll(directory, 0o755)
	if err != nil {
		return fmt.Errorf("creating configuration directory: %w", err)
	}

	mode := fs.FileMode(0o666)
	info, err := os.Stat(f.path)
	switch {
	case err == nil:
		mode = info.Mode().Perm()
	case !errors.Is(err, os.ErrNotExist):
		return fmt.Errorf("stating configuration file: %w", err)
	}

	temporaryFile, err := os.CreateTemp(directory, ".config-*.json.tmp")
	if err != nil {
		return fmt.Errorf("creating temporary configuration file: %w", err)
	}
	temporaryPath := temporaryFile.Name()
	defer func() {
		_ = os.Remove(temporaryPath) // no-op once renamed
	}()

	_, err = temporaryFile.Write(data)
	if err != nil {
		_ = temporaryFile.Close()
		return fmt.Errorf("writing temporary configuration file: %w", err)
	}

	err = temporaryFile.Chmod(mode)
	if err != nil {
		_ = temporaryFile.Close()
		return fmt.Errorf("setting temporary configuration file permissions: %w", err)
	}

	err = temporaryFile.Sync()
	if err != nil {
		_ = temporaryFile.Close()
		return fmt.Errorf("syncing temporary configuration file: %w", err)
	}

	err = temporaryFile.Close()
	if err != nil {
		return fmt.Errorf("closing temporary configuration file: %w", err)
	}

	err = os.Rename(temporaryPath, f.path)
	if err != nil {
		return fmt.Errorf("replacing configuration file: %w", err)
	}
	return nil
}
