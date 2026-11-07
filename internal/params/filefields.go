package params

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// fileFieldSuffix marks a setting whose value is read from a file:
// "token_file": "/run/secrets/token" sets "token" to the content of
// /run/secrets/token. This keeps secrets such as tokens and passwords
// out of the JSON configuration, for example with Docker secrets.
const fileFieldSuffix = "_file"

var (
	ErrFileFieldNotString = errors.New("file field value is not a string")
	ErrFileFieldEmptyPath = errors.New("file field path is empty")
	ErrFileFieldConflict  = errors.New("field is set both directly and with its file variant")
	ErrFileFieldRead      = errors.New("cannot read file field")
	errDecodeSettings     = errors.New("cannot decode settings object")
)

// resolveFileFields replaces every "<field>_file" key of a single settings
// object by "<field>", whose value is the content of the file at the path
// given. A single trailing newline (\n or \r\n) is removed from the content.
// The settings object is returned unchanged if it has no file field.
func resolveFileFields(rawSettings json.RawMessage,
	readFile func(filename string) ([]byte, error),
) (resolved json.RawMessage, err error) {
	var fields map[string]json.RawMessage
	err = json.Unmarshal(rawSettings, &fields)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errDecodeSettings, err)
	}

	fileKeys := make([]string, 0, len(fields))
	for key := range fields {
		base, isFileKey := strings.CutSuffix(key, fileFieldSuffix)
		if isFileKey && base != "" {
			fileKeys = append(fileKeys, key)
		}
	}
	if len(fileKeys) == 0 {
		return rawSettings, nil
	}
	sort.Strings(fileKeys) // deterministic error reporting

	for _, fileKey := range fileKeys {
		key := strings.TrimSuffix(fileKey, fileFieldSuffix)

		var path string
		err = json.Unmarshal(fields[fileKey], &path)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrFileFieldNotString, fileKey)
		}
		if path == "" {
			return nil, fmt.Errorf("%w: %s", ErrFileFieldEmptyPath, fileKey)
		}

		_, keySet := fields[key]
		if keySet {
			return nil, fmt.Errorf("%w: %s and %s", ErrFileFieldConflict, key, fileKey)
		}

		content, err := readFile(path)
		if err != nil {
			return nil, fmt.Errorf("%w %s: %w", ErrFileFieldRead, fileKey, err)
		}
		value := strings.TrimSuffix(string(content), "\n")
		value = strings.TrimSuffix(value, "\r")

		encoded, err := json.Marshal(value)
		if err != nil {
			return nil, fmt.Errorf("encoding value of %s: %w", fileKey, err)
		}
		fields[key] = encoded
		delete(fields, fileKey)
	}

	return json.Marshal(fields)
}
