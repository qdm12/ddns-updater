package params

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/regex"
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
	s, err := r.env.Get("CONFIG", params.CaseSensitiveValue())
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
	matcher := regex.NewMatcher()

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
	if provider == constants.DuckDNS { // only hosts, no domain
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

	if len(common.IPVersion) == 0 {
		common.IPVersion = ipversion.IP4or6.String()
	}
	ipVersion, err := ipversion.Parse(common.IPVersion)
	if err != nil {
		return nil, nil, err
	}

	var settingsConstructor settings.Constructor
	switch provider {
	case constants.Cloudflare:
		settingsConstructor = settings.NewCloudflare
	case constants.DdnssDe:
		settingsConstructor = settings.NewDdnss
	case constants.DigitalOcean:
		settingsConstructor = settings.NewDigitalOcean
	case constants.DnsOMatic:
		settingsConstructor = settings.NewDNSOMatic
	case constants.DNSPod:
		settingsConstructor = settings.NewDNSPod
	case constants.DonDominio:
		settingsConstructor = settings.NewDonDominio
	case constants.Dreamhost:
		settingsConstructor = settings.NewDreamhost
	case constants.DuckDNS:
		settingsConstructor = settings.NewDuckdns
	case constants.Dyn:
		settingsConstructor = settings.NewDyn
	case constants.DynV6:
		settingsConstructor = settings.NewDynV6
	case constants.FreeDNS:
		settingsConstructor = settings.NewFreedns
	case constants.Gandi:
		settingsConstructor = settings.NewGandi
	case constants.GoDaddy:
		settingsConstructor = settings.NewGodaddy
	case constants.Google:
		settingsConstructor = settings.NewGoogle
	case constants.HE:
		settingsConstructor = settings.NewHe
	case constants.Infomaniak:
		settingsConstructor = settings.NewInfomaniak
	case constants.Linode:
		settingsConstructor = settings.NewLinode
	case constants.LuaDNS:
		settingsConstructor = settings.NewLuaDNS
	case constants.Namecheap:
		settingsConstructor = settings.NewNamecheap
	case constants.Njalla:
		settingsConstructor = settings.NewNjalla
	case constants.NoIP:
		settingsConstructor = settings.NewNoip
	case constants.OpenDNS:
		settingsConstructor = settings.NewOpendns
	case constants.OVH:
		settingsConstructor = settings.NewOVH
	case constants.SelfhostDe:
		settingsConstructor = settings.NewSelfhostde
	case constants.Spdyn:
		settingsConstructor = settings.NewSpdyn
	case constants.Strato:
		settingsConstructor = settings.NewStrato
	case constants.Variomedia:
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
