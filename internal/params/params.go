package params

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"strings"
	"time"

	"github.com/qdm12/ddns-updater/internal/settings"
	"github.com/qdm12/ddns-updater/pkg/publicip/dns"
	"github.com/qdm12/ddns-updater/pkg/publicip/http"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
	"github.com/qdm12/golibs/logging"
	"github.com/qdm12/golibs/params"
)

const (
	all = "all"
)

type Reader interface {
	// JSON
	JSONSettings(filePath string) (allSettings []settings.Settings, warnings []string, err error)

	// Core
	Period() (period time.Duration, warnings []string, err error)
	PublicIPHTTPProviders() (providers []http.Provider, err error)
	PublicIPv4HTTPProviders() (providers []http.Provider, err error)
	PublicIPv6HTTPProviders() (providers []http.Provider, err error)
	PublicIPDNSProviders() (providers []dns.Provider, err error)
	HTTPTimeout() (duration time.Duration, err error)
	CooldownPeriod() (duration time.Duration, err error)

	// File paths
	ExeDir() (dir string, err error)
	DataDir(currentDir string) (string, error)

	// Web UI
	ListeningPort() (listeningPort uint16, warning string, err error)
	RootURL() (rootURL string, err error)

	// Backup
	BackupPeriod() (duration time.Duration, err error)
	BackupDirectory() (directory string, err error)

	// Other
	LoggerConfig() (level logging.Level, caller logging.Caller, err error)
	GotifyURL() (URL *url.URL, err error)
	GotifyToken() (token string, err error)
}

type reader struct {
	env      params.Env
	os       params.OS
	readFile func(filename string) ([]byte, error)
	retroFn  func(oldKey, newKey string)
}

func NewReader(logger logging.Logger) Reader {
	return &reader{
		env:      params.NewEnv(),
		os:       params.NewOS(),
		readFile: ioutil.ReadFile,
		retroFn: func(oldKey, newKey string) {
			logger.Warn("You are using the old environment variable %s, please consider switching to %s instead", oldKey, newKey)
		},
	}
}

// GetDataDir obtains the data directory from the environment
// variable DATADIR.
func (r *reader) DataDir(currentDir string) (string, error) {
	return r.env.Get("DATADIR", params.Default(currentDir+"/data"))
}

func (r *reader) ListeningPort() (listeningPort uint16, warning string, err error) {
	return r.env.ListeningPort("LISTENING_PORT", params.Default("8000"))
}

func (r *reader) LoggerConfig() (level logging.Level, caller logging.Caller, err error) {
	caller, err = r.env.LogCaller("LOG_CALLER", params.Default("hidden"))
	if err != nil {
		return level, caller, err
	}

	level, err = r.env.LogLevel("LOG_LEVEL", params.Default("info"))
	if err != nil {
		return level, caller, err
	}

	return level, caller, nil
}

func (r *reader) GotifyURL() (url *url.URL, err error) {
	return r.env.URL("GOTIFY_URL")
}

func (r *reader) GotifyToken() (token string, err error) {
	return r.env.Get("GOTIFY_TOKEN",
		params.CaseSensitiveValue(),
		params.Compulsory(),
		params.Unset())
}

func (r *reader) RootURL() (rootURL string, err error) {
	return r.env.RootURL("ROOT_URL")
}

func (r *reader) Period() (period time.Duration, warnings []string, err error) {
	// Backward compatibility
	n, err := r.env.Int("DELAY", params.Compulsory())
	if err == nil { // integer only, treated as seconds
		return time.Duration(n) * time.Second,
			[]string{
				"the environment variable DELAY should be changed to PERIOD",
				fmt.Sprintf(`the value for the duration period of the updater does not have a time unit, you might want to set it to "%ds" instead of "%d"`, n, n), //nolint:lll
			}, nil
	}
	period, err = r.env.Duration("DELAY", params.Compulsory())
	if err == nil {
		return period,
			[]string{
				"the environment variable DELAY should be changed to PERIOD",
			}, nil
	}
	period, err = r.env.Duration("PERIOD", params.Default("10m"))
	return period, nil, err
}

// PublicIPHTTPProviders obtains the HTTP providers to obtain your public IPv4 and/or IPv6 address.
func (r *reader) PublicIPDNSProviders() (providers []dns.Provider, err error) {
	s, err := r.env.Get("PUBLICIP_DNS_PROVIDERS", params.Default(all))
	if err != nil {
		return nil, err
	}

	availableProviders := dns.ListProviders()

	fields := strings.Split(s, ",")
	providers = make([]dns.Provider, len(fields))
	for i, field := range fields {
		if field == all {
			return availableProviders, nil
		}

		providers[i] = dns.Provider(field)
		if err := dns.ValidateProvider(providers[i]); err != nil {
			return nil, err
		}
	}

	return providers, nil
}

// PublicIPHTTPProviders obtains the HTTP providers to obtain your public IPv4 or IPv6 address.
func (r *reader) PublicIPHTTPProviders() (providers []http.Provider, err error) {
	return r.httpIPMethod("PUBLICIP_HTTP_PROVIDERS", "IP_METHOD", ipversion.IP4or6)
}

// PublicIPv4HTTPProviders obtains the HTTP providers to obtain your public IPv4 address.
func (r *reader) PublicIPv4HTTPProviders() (providers []http.Provider, err error) {
	return r.httpIPMethod("PUBLICIPV4_HTTP_PROVIDERS", "IPV4_METHOD", ipversion.IP4)
}

// PublicIPv6HTTPProviders obtains the HTTP providers to obtain your public IPv6 address.
func (r *reader) PublicIPv6HTTPProviders() (providers []http.Provider, err error) {
	return r.httpIPMethod("PUBLICIPV6_HTTP_PROVIDERS", "IPV6_METHOD", ipversion.IP6)
}

var (
	ErrInvalidPublicIPHTTPProvider = errors.New("invalid public IP HTTP provider")
)

func (r *reader) httpIPMethod(envKey, retroKey string, version ipversion.IPVersion) (
	providers []http.Provider, err error) {
	retroKeyOption := params.RetroKeys([]string{retroKey}, r.retroFn)
	s, err := r.env.Get(envKey, params.Default("cycle"), retroKeyOption)
	if err != nil {
		return nil, err
	}

	availableProviders := http.ListProvidersForVersion(version)
	choices := make(map[http.Provider]struct{}, len(availableProviders))
	for _, provider := range availableProviders {
		choices[provider] = struct{}{}
	}

	fields := strings.Split(s, ",")

	for _, field := range fields {
		// Retro-compatibility.
		switch field {
		case "ipify6":
			field = "ipify"
		case "noip4", "noip6", "noip8245_4", "noip8245_6":
			field = "noip"
		case "cycle":
			field = all
		}

		if field == all {
			return availableProviders, nil
		}

		// Custom URL check
		url, err := url.Parse(field)
		if err == nil && url != nil && url.Scheme == "https" {
			providers = append(providers, http.CustomProvider(url))
			continue
		}

		provider := http.Provider(field)
		if _, ok := choices[provider]; !ok {
			return nil, fmt.Errorf("%w: %s", ErrInvalidPublicIPHTTPProvider, provider)
		}
		providers = append(providers, provider)
	}

	if len(providers) == 0 {
		return nil, fmt.Errorf("%w: for IP version %s", ErrInvalidPublicIPHTTPProvider, version)
	}

	return providers, nil
}

func (r *reader) ExeDir() (dir string, err error) {
	return r.os.ExeDir()
}

func (r *reader) HTTPTimeout() (duration time.Duration, err error) {
	return r.env.Duration("HTTP_TIMEOUT", params.Default("10s"))
}

func (r *reader) BackupPeriod() (duration time.Duration, err error) {
	s, err := r.env.Get("BACKUP_PERIOD", params.Default("0"))
	if err != nil {
		return 0, err
	}
	return time.ParseDuration(s)
}

func (r *reader) BackupDirectory() (directory string, err error) {
	return r.env.Path("BACKUP_DIRECTORY", params.Default("./data"))
}

func (r *reader) CooldownPeriod() (duration time.Duration, err error) {
	return r.env.Duration("UPDATE_COOLDOWN_PERIOD", params.Default("5m"))
}
