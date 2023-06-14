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
	"github.com/qdm12/ddns-updater/internal/provider"
	"github.com/qdm12/ddns-updater/internal/provider/constants"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
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

// JSONProviders obtain the update settings from the JSON content,
// first trying from the environment variable CONFIG and then from
// the file config.json.
func (r *Reader) JSONProviders(filePath string) (
	providers []provider.Provider, warnings []string, err error) {
	providers, warnings, err = r.getProvidersFromEnv(filePath)
	if providers != nil || warnings != nil || err != nil {
		return providers, warnings, err
	}
	return r.getProvidersFromFile(filePath)
}

var errWriteConfigToFile = errors.New("cannot write configuration to file")

// getProvidersFromFile obtain the update settings from config.json.
func (r *Reader) getProvidersFromFile(filePath string) (
	providers []provider.Provider, warnings []string, err error) {
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

// getProvidersFromEnv obtain the update settings from the environment variable CONFIG.
// If the settings are valid, they are written to the filePath.
func (r *Reader) getProvidersFromEnv(filePath string) (
	providers []provider.Provider, warnings []string, err error) {
	s := os.Getenv("CONFIG")
	if s == "" {
		return nil, nil, nil
	}
	r.logger.Info("reading JSON config from environment variable CONFIG")
	r.logger.Debug("config read: " + s)

	b := []byte(s)

	providers, warnings, err = extractAllSettings(b)
	if err != nil {
		return providers, warnings, fmt.Errorf("configuration given: %w", err)
	}

	buffer := bytes.NewBuffer(nil)
	err = json.Indent(buffer, b, "", "  ")
	if err != nil {
		return providers, warnings, fmt.Errorf("%w: %w", errWriteConfigToFile, err)
	}
	const mode = fs.FileMode(0600)
	err = r.writeFile(filePath, buffer.Bytes(), mode)
	if err != nil {
		return providers, warnings, fmt.Errorf("%w: %w", errWriteConfigToFile, err)
	}

	return providers, warnings, nil
}

var (
	errUnmarshalCommon = errors.New("cannot unmarshal common settings")
	errUnmarshalRaw    = errors.New("cannot unmarshal raw configuration")
)

func extractAllSettings(jsonBytes []byte) (
	allProviders []provider.Provider, warnings []string, err error) {
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
		newProvider, newWarnings, err := makeSettingsFromObject(common, rawConfig.Settings[i])
		warnings = append(warnings, newWarnings...)
		if err != nil {
			return nil, warnings, err
		}
		allProviders = append(allProviders, newProvider...)
	}

	return allProviders, warnings, nil
}

func makeSettingsFromObject(common commonSettings, rawSettings json.RawMessage) (
	providers []provider.Provider, warnings []string, err error) {
	providerName := models.Provider(common.Provider)
	if providerName == constants.DuckDNS { // only hosts, no domain
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

	providers = make([]provider.Provider, len(hosts))
	for i, host := range hosts {
		providers[i], err = provider.New(providerName, rawSettings, common.Domain,
			host, ipVersion)
		if err != nil {
			return nil, warnings, err
		}
	}
	return providers, warnings, nil
}
