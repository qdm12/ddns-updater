package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/qdm12/ddns-updater/internal/provider/schema"
)

// errInvalidForm is returned when the submitted form cannot be turned into
// a settings entry.
var errInvalidForm = errors.New("invalid form input")

const (
	formProviderKey   = "provider"
	formDomainKey     = "domain"
	formIPVersionKey  = "ip_version"
	formIPv6SuffixKey = "ipv6_suffix"
	// formFieldPrefix prefixes the input names of the provider specific
	// settings, so that they cannot collide with the common settings.
	formFieldPrefix = "field_"
)

// buildEntry turns the submitted form values into a settings entry, using
// the provider schema to know which settings are accepted and how to
// convert them.
func buildEntry(values url.Values, provider schema.Provider) (
	entry json.RawMessage, err error,
) {
	entryObject := map[string]any{
		formProviderKey: provider.Name,
	}

	domain := strings.TrimSpace(values.Get(formDomainKey))
	if domain == "" {
		return nil, fmt.Errorf("%w: the domain is required", errInvalidForm)
	}
	entryObject[formDomainKey] = domain

	ipVersion := strings.TrimSpace(values.Get(formIPVersionKey))
	if ipVersion != "" {
		entryObject[formIPVersionKey] = ipVersion
	}

	ipv6Suffix := strings.TrimSpace(values.Get(formIPv6SuffixKey))
	if ipv6Suffix != "" {
		entryObject[formIPv6SuffixKey] = ipv6Suffix
	}

	fields := make(map[string]schema.Field, len(provider.Fields))
	for _, field := range provider.Fields {
		fields[field.Key] = field
	}

	for name, rawValues := range values {
		if !strings.HasPrefix(name, formFieldPrefix) {
			continue
		}
		key := strings.TrimPrefix(name, formFieldPrefix)
		field, found := fields[key]
		if !found {
			return nil, fmt.Errorf("%w: %q is not a setting of provider %q",
				errInvalidForm, key, provider.Name)
		}

		raw := strings.TrimSpace(rawValues[len(rawValues)-1])
		value, set, err := convertFieldValue(field, raw)
		if err != nil {
			return nil, err
		}
		if set {
			entryObject[key] = value
		}
	}

	entry, err = json.Marshal(entryObject)
	if err != nil {
		return nil, fmt.Errorf("encoding settings entry: %w", err)
	}
	return entry, nil
}

// convertFieldValue converts a form value to the JSON value of the setting.
// The boolean set is false when the setting should be omitted, which is
// the case for empty optional settings.
func convertFieldValue(field schema.Field, raw string) (
	value any, set bool, err error,
) {
	if field.Kind == schema.KindBool {
		if raw == "" {
			return nil, false, nil
		}
		value, err = strconv.ParseBool(raw)
		if err != nil {
			// An HTML checkbox submits the value "on" when checked.
			if raw != "on" {
				return nil, false, fmt.Errorf("%w: %s: %w",
					errInvalidForm, field.Label, err)
			}
			value = true
		}
		return value, true, nil
	}

	if raw == "" {
		return nil, false, nil
	}

	switch field.Kind {
	case schema.KindNumber:
		_, err = strconv.ParseFloat(raw, 64)
		if err != nil {
			return nil, false, fmt.Errorf("%w: %s must be a number: %w",
				errInvalidForm, field.Label, err)
		}
		return json.Number(raw), true, nil
	case schema.KindJSON:
		if !json.Valid([]byte(raw)) {
			return nil, false, fmt.Errorf("%w: %s must be valid JSON",
				errInvalidForm, field.Label)
		}
		return json.RawMessage(raw), true, nil
	case schema.KindString:
		return raw, true, nil
	default:
		return nil, false, fmt.Errorf("%w: %s has an unsupported type %q",
			errInvalidForm, field.Label, field.Kind)
	}
}

// entryValues are the values of an existing settings entry, ready to be
// displayed in the form.
type entryValues struct {
	Provider   string
	Domain     string
	IPVersion  string
	IPv6Suffix string
	Fields     map[string]string
}

// parseEntryValues extracts the form values of an existing settings entry.
func parseEntryValues(entry json.RawMessage) (values entryValues, err error) {
	var object map[string]json.RawMessage
	err = json.Unmarshal(entry, &object)
	if err != nil {
		return entryValues{}, fmt.Errorf("decoding settings entry: %w", err)
	}

	values.Fields = make(map[string]string, len(object))
	for key, raw := range object {
		switch key {
		case formProviderKey:
			values.Provider = rawToString(raw)
		case formDomainKey:
			values.Domain = rawToString(raw)
		case formIPVersionKey:
			values.IPVersion = rawToString(raw)
		case formIPv6SuffixKey:
			values.IPv6Suffix = rawToString(raw)
		default:
			values.Fields[key] = rawToDisplayString(raw)
		}
	}
	return values, nil
}

// rawToString decodes a JSON string.
func rawToString(raw json.RawMessage) (value string) {
	//nolint:errcheck // a non string value is displayed as its JSON content
	_ = json.Unmarshal(raw, &value)
	return value
}

// rawToDisplayString renders a JSON value so that it can be put in a text
// input or a text area. Objects and arrays are indented.
func rawToDisplayString(raw json.RawMessage) string {
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return text
	}

	var compacted bytes.Buffer
	if err := json.Compact(&compacted, raw); err != nil {
		return string(raw)
	}

	content := compacted.Bytes()
	if len(content) == 0 || (content[0] != '{' && content[0] != '[') {
		return compacted.String()
	}

	var indented bytes.Buffer
	if err := json.Indent(&indented, content, "", "  "); err != nil {
		return compacted.String()
	}
	return indented.String()
}
