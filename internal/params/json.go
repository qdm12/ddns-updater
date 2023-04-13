package params

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/settings"
	"github.com/qdm12/ddns-updater/internal/settings/constants"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
	"github.com/qdm12/golibs/params"
)

type commonSettings struct {
	Provider  string `json:"provider"`
	Domain    string `json:"domain"`
	Host      string `json:"host"`
	IPVersion string `json:"ip_version"`
	// Retro values for warnings
	IPMethod *string `json:"ip_method,omitempty"`
	Delay    *uint64 `json:"delay,omitempty"`
}

// JSONSettings obtain the update settings from the JSON content, first trying from the environment variable CONFIG
// and then from the file config.json.
func (r *Reader) JSONSettings(filePath string) (
	allSettings []settings.Settings, warnings []string, err error) {
	allSettings, warnings, err = r.getSettingsFromEnv(filePath)
	if allSettings != nil || warnings != nil || err != nil {
		return allSettings, warnings, err
	}
	return r.getSettingsFromFile(filePath)
}

var errWriteConfigToFile = errors.New("cannot write configuration to file")

// getSettingsFromFile obtain the update settings from config.json.
func (r *Reader) getSettingsFromFile(filePath string) (
	allSettings []settings.Settings, warnings []string, err error) {
	r.logger.Info("reading JSON config from file " + filePath)
	bytes, err := r.readFile(filePath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, nil, err
		}

		r.logger.Info("file not found, creating an empty settings file")

		const mode = fs.FileMode(0600)

		err = r.writeFile(filePath, []byte(`{}`), mode)
		if err != nil {
			err = fmt.Errorf("%w: %w", errWriteConfigToFile, err)
		}
		return nil, nil, err
	}
	r.logger.Debug("config read: " + string(bytes))

	return extractAllSettings(bytes)
}

// getSettingsFromEnv obtain the update settings from the environment variable CONFIG.
// If the settings are valid, they are written to the filePath.
func (r *Reader) getSettingsFromEnv(filePath string) (
	allSettings []settings.Settings, warnings []string, err error) {
	s, err := r.env.Get("CONFIG", params.CaseSensitiveValue())
	if err != nil {
		return nil, nil, fmt.Errorf("%w: for environment variable CONFIG", err)
	} else if s == "" {
		return nil, nil, nil
	}
	r.logger.Info("reading JSON config from environment variable CONFIG")
	r.logger.Debug("config read: " + s)

	b := []byte(s)

	allSettings, warnings, err = extractAllSettings(b)
	if err != nil {
		return allSettings, warnings, fmt.Errorf("configuration given: %w", err)
	}

	buffer := bytes.NewBuffer(nil)
	err = json.Indent(buffer, b, "", "  ")
	if err != nil {
		return allSettings, warnings, fmt.Errorf("%w: %w", errWriteConfigToFile, err)
	}
	const mode = fs.FileMode(0600)
	err = r.writeFile(filePath, buffer.Bytes(), mode)
	if err != nil {
		return allSettings, warnings, fmt.Errorf("%w: %w", errWriteConfigToFile, err)
	}

	return allSettings, warnings, nil
}

var (
	errUnmarshalCommon = errors.New("cannot unmarshal common settings")
	errUnmarshalRaw    = errors.New("cannot unmarshal raw configuration")
)

func extractAllSettings(jsonBytes []byte) (
	allSettings []settings.Settings, warnings []string, err error) {
	config := struct {
		CommonSettings []commonSettings `json:"settings"`
	}{}
	rawConfig := struct {
		Settings []json.RawMessage `json:"settings"`
	}{}
	err = json.Unmarshal(jsonBytes, &config)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %w", errUnmarshalCommon, err)
	}
	err = json.Unmarshal(jsonBytes, &rawConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %w", errUnmarshalRaw, err)
	}

	for i, common := range config.CommonSettings {
		newSettings, newWarnings, err := makeSettingsFromObject(common, rawConfig.Settings[i])
		warnings = append(warnings, newWarnings...)
		if err != nil {
			return nil, warnings, err
		}
		allSettings = append(allSettings, newSettings...)
	}

	return allSettings, warnings, nil
}

func makeSettingsFromObject(common commonSettings, rawSettings json.RawMessage) (
	settingsSlice []settings.Settings, warnings []string, err error) {
	provider := models.Provider(common.Provider)
	if provider == constants.DuckDNS { // only hosts, no domain
		if common.Domain != "" { // retro compatibility
			if common.Host == "" {
				common.Host = strings.TrimSuffix(common.Domain, ".duckdns.org")
				warnings = append(warnings,
					fmt.Sprintf("DuckDNS record should have %q specified as host instead of %q as domain",
						common.Host, common.Domain))
			} else {
				warnings = append(warnings,
					fmt.Sprintf("ignoring domain %q because host %q is specified for DuckDNS record",
						common.Domain, common.Host))
			}
		}
	}
	hosts := strings.Split(common.Host, ",")

	if common.IPVersion == "" {
		common.IPVersion = ipversion.IP4or6.String()
	}
	ipVersion, err := ipversion.Parse(common.IPVersion)
	if err != nil {
		return nil, nil, err
	}

	settingsSlice = make([]settings.Settings, len(hosts))
	for i, host := range hosts {
		settingsSlice[i], err = settings.New(provider, rawSettings, common.Domain,
			host, ipVersion)
		if err != nil {
			return nil, warnings, err
		}
	}
	return settingsSlice, warnings, nil
}
