package params

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/qdm12/ddns-updater/internal/constants"
	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/regex"
	"github.com/qdm12/ddns-updater/internal/settings"
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
func (r *reader) JSONSettings(filePath string) (allSettings []settings.Settings, warnings []string, err error) {
	allSettings, warnings, err = r.getSettingsFromEnv()
	if allSettings != nil || warnings != nil || err != nil {
		return allSettings, warnings, err
	}
	return r.getSettingsFromFile(filePath)
}

// getSettingsFromFile obtain the update settings from config.json.
func (r *reader) getSettingsFromFile(filePath string) (allSettings []settings.Settings, warnings []string, err error) {
	bytes, err := r.readFile(filePath)
	if err != nil {
		return nil, nil, err
	}
	return extractAllSettings(bytes)
}

// getSettingsFromEnv obtain the update settings from the environment variable CONFIG.
func (r *reader) getSettingsFromEnv() (allSettings []settings.Settings, warnings []string, err error) {
	s, err := r.env.Get("CONFIG")
	if err != nil {
		return nil, nil, err
	} else if len(s) == 0 {
		return nil, nil, nil
	}
	return extractAllSettings([]byte(s))
}

func extractAllSettings(jsonBytes []byte) (allSettings []settings.Settings, warnings []string, err error) {
	config := struct {
		CommonSettings []commonSettings `json:"settings"`
	}{}
	rawConfig := struct {
		Settings []json.RawMessage `json:"settings"`
	}{}
	if err := json.Unmarshal(jsonBytes, &config); err != nil {
		return nil, nil, err
	}
	if err := json.Unmarshal(jsonBytes, &rawConfig); err != nil {
		return nil, nil, err
	}
	matcher, err := regex.NewMatcher()
	if err != nil {
		return nil, nil, err
	}
	for i, common := range config.CommonSettings {
		newSettings, newWarnings, err := makeSettingsFromObject(common, rawConfig.Settings[i], matcher)
		warnings = append(warnings, newWarnings...)
		if err != nil {
			return nil, warnings, err
		}
		allSettings = append(allSettings, newSettings...)
	}
	if len(allSettings) == 0 {
		warnings = append(warnings, "no settings found in JSON data")
	}
	return allSettings, warnings, nil
}

// TODO remove gocyclo.
//nolint:gocyclo
func makeSettingsFromObject(common commonSettings, rawSettings json.RawMessage, matcher regex.Matcher) (
	settingsSlice []settings.Settings, warnings []string, err error) {
	provider := models.Provider(common.Provider)
	if provider == constants.DUCKDNS { // only hosts, no domain
		if len(common.Domain) > 0 { // retro compatibility
			if len(common.Host) == 0 {
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
	ipVersion := models.IPVersion(common.IPVersion)
	if len(ipVersion) == 0 {
		ipVersion = constants.IPv4OrIPv6 // default
	}
	if ipVersion != constants.IPv4OrIPv6 && ipVersion != constants.IPv4 && ipVersion != constants.IPv6 {
		return nil, warnings, fmt.Errorf("ip version %q is not valid", ipVersion)
	}
	var settingsConstructor settings.Constructor
	switch provider {
	case constants.CLOUDFLARE:
		settingsConstructor = settings.NewCloudflare
	case constants.DIGITALOCEAN:
		settingsConstructor = settings.NewDigitalOcean
	case constants.DDNSSDE:
		settingsConstructor = settings.NewDdnss
	case constants.DONDOMINIO:
		settingsConstructor = settings.NewDonDominio
	case constants.DNSOMATIC:
		settingsConstructor = settings.NewDNSOMatic
	case constants.DNSPOD:
		settingsConstructor = settings.NewDNSPod
	case constants.DREAMHOST:
		settingsConstructor = settings.NewDreamhost
	case constants.DUCKDNS:
		settingsConstructor = settings.NewDuckdns
	case constants.FREEDNS:
		settingsConstructor = settings.NewFreedns
	case constants.GANDI:
		settingsConstructor = settings.NewGandi
	case constants.GODADDY:
		settingsConstructor = settings.NewGodaddy
	case constants.GOOGLE:
		settingsConstructor = settings.NewGoogle
	case constants.HE:
		settingsConstructor = settings.NewHe
	case constants.INFOMANIAK:
		settingsConstructor = settings.NewInfomaniak
	case constants.LINODE:
		settingsConstructor = settings.NewLinode
	case constants.LUADNS:
		settingsConstructor = settings.NewLuaDNS
	case constants.NAMECHEAP:
		settingsConstructor = settings.NewNamecheap
	case constants.NOIP:
		settingsConstructor = settings.NewNoip
	case constants.DYN:
		settingsConstructor = settings.NewDyn
	case constants.SELFHOSTDE:
		settingsConstructor = settings.NewSelfhostde
	case constants.STRATO:
		settingsConstructor = settings.NewStrato
	case constants.OVH:
		settingsConstructor = settings.NewOVH
	case constants.DYNV6:
		settingsConstructor = settings.NewDynV6
	case constants.OPENDNS:
		settingsConstructor = settings.NewOpendns
	case constants.VARIOMEDIA:
		settingsConstructor = settings.NewVariomedia
	default:
		return nil, warnings, fmt.Errorf("provider %q is not supported", provider)
	}
	settingsSlice = make([]settings.Settings, len(hosts))
	for i, host := range hosts {
		settingsSlice[i], err = settingsConstructor(rawSettings, common.Domain, host, ipVersion, matcher)
		if err != nil {
			return nil, warnings, err
		}
	}
	return settingsSlice, warnings, nil
}
