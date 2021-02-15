package params

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"time"

	"github.com/qdm12/ddns-updater/internal/constants"
	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/settings"
	"github.com/qdm12/golibs/logging"
	"github.com/qdm12/golibs/params"
)

const https = "https"

type Reader interface {
	// JSON
	JSONSettings(filePath string) (allSettings []settings.Settings, warnings []string, err error)

	// Core
	Period() (period time.Duration, warnings []string, err error)
	IPMethod() (method models.IPMethod, err error)
	IPv4Method() (method models.IPMethod, err error)
	IPv6Method() (method models.IPMethod, err error)
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
}

func NewReader(logger logging.Logger) Reader {
	return &reader{
		env:      params.NewEnv(),
		os:       params.NewOS(),
		readFile: ioutil.ReadFile,
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

func (r *reader) IPMethod() (method models.IPMethod, err error) {
	s, err := r.env.Get("IP_METHOD", params.Default("cycle"))
	if err != nil {
		return method, err
	}
	for _, choice := range constants.IPMethods() {
		if choice.Name == s {
			return choice, nil
		}
	}
	url, err := url.Parse(s)
	if err != nil || url == nil || url.Scheme != https {
		return method, fmt.Errorf("ip method %q is not valid", s)
	}
	return models.IPMethod{
		Name: s,
		URL:  s,
		IPv4: true,
		IPv6: true,
	}, nil
}

func (r *reader) IPv4Method() (method models.IPMethod, err error) {
	s, err := r.env.Get("IPV4_METHOD", params.Default("cycle"))
	if err != nil {
		return method, err
	}
	for _, choice := range constants.IPMethods() {
		if choice.Name == s {
			if s != "cycle" && !choice.IPv4 {
				return method, fmt.Errorf("ip method %s does not support IPv4", s)
			}
			return choice, nil
		}
	}
	url, err := url.Parse(s)
	if err != nil || url == nil || url.Scheme != https {
		return method, fmt.Errorf("ipv4 method %q is not valid", s)
	}
	return models.IPMethod{
		Name: s,
		URL:  s,
		IPv4: true,
	}, nil
}

func (r *reader) IPv6Method() (method models.IPMethod, err error) {
	s, err := r.env.Get("IPV6_METHOD", params.Default("cycle"))
	if err != nil {
		return method, err
	}
	for _, choice := range constants.IPMethods() {
		if choice.Name == s {
			if s != "cycle" && !choice.IPv6 {
				return method, fmt.Errorf("ip method %s does not support IPv6", s)
			}
			return choice, nil
		}
	}
	url, err := url.Parse(s)
	if err != nil || url == nil || url.Scheme != https {
		return method, fmt.Errorf("ipv6 method %q is not valid", s)
	}
	return models.IPMethod{
		Name: s,
		URL:  s,
		IPv6: true,
	}, nil
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
