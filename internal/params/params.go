package params

//nolint:gci
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
	libparams "github.com/qdm12/golibs/params"
	"github.com/qdm12/golibs/verification"
)

const https = "https"

type Reader interface {
	// JSON
	GetSettings(filePath string) (allSettings []settings.Settings, warnings []string, err error)

	// Core
	GetPeriod() (period time.Duration, warnings []string, err error)
	GetIPMethod() (method models.IPMethod, err error)
	GetIPv4Method() (method models.IPMethod, err error)
	GetIPv6Method() (method models.IPMethod, err error)
	GetHTTPTimeout() (duration time.Duration, err error)

	// File paths
	GetExeDir() (dir string, err error)
	GetDataDir(currentDir string) (string, error)

	// Web UI
	GetListeningPort() (listeningPort uint16, warning string, err error)
	GetRootURL() (rootURL string, err error)

	// Backup
	GetBackupPeriod() (duration time.Duration, err error)
	GetBackupDirectory() (directory string, err error)

	// Other
	GetLoggerConfig() (encoding logging.Encoding, level logging.Level, err error)
	GetGotifyURL() (URL *url.URL, err error)
	GetGotifyToken() (token string, err error)

	// Version getters
	GetVersion() string
	GetBuildDate() string
	GetVcsRef() string
}

type reader struct {
	envParams libparams.EnvParams
	verifier  verification.Verifier
	readFile  func(filename string) ([]byte, error)
}

func NewReader(logger logging.Logger) Reader {
	return &reader{
		envParams: libparams.NewEnvParams(),
		verifier:  verification.NewVerifier(),
		readFile:  ioutil.ReadFile,
	}
}

// GetDataDir obtains the data directory from the environment
// variable DATADIR.
func (r *reader) GetDataDir(currentDir string) (string, error) {
	return r.envParams.GetEnv("DATADIR", libparams.Default(currentDir+"/data"))
}

func (r *reader) GetListeningPort() (listeningPort uint16, warning string, err error) {
	return r.envParams.GetListeningPort("LISTENING_PORT")
}

func (r *reader) GetLoggerConfig() (encoding logging.Encoding, level logging.Level, err error) {
	return r.envParams.GetLoggerConfig()
}

func (r *reader) GetGotifyURL() (url *url.URL, err error) {
	return r.envParams.GetGotifyURL()
}

func (r *reader) GetGotifyToken() (token string, err error) {
	return r.envParams.GetGotifyToken()
}

func (r *reader) GetRootURL() (rootURL string, err error) {
	return r.envParams.GetRootURL()
}

func (r *reader) GetPeriod() (period time.Duration, warnings []string, err error) {
	// Backward compatibility
	n, err := r.envParams.GetEnvInt("DELAY", libparams.Compulsory())
	if err == nil { // integer only, treated as seconds
		return time.Duration(n) * time.Second,
			[]string{
				"the environment variable DELAY should be changed to PERIOD",
				fmt.Sprintf("the value for the duration period of the updater does not have a time unit, you might want to set it to \"%ds\" instead of \"%d\"", n, n), //nolint:lll
			}, nil
	}
	period, err = r.envParams.GetDuration("DELAY", libparams.Compulsory())
	if err == nil {
		return period,
			[]string{
				"the environment variable DELAY should be changed to PERIOD",
			}, nil
	}
	period, err = r.envParams.GetDuration("PERIOD", libparams.Default("10m"))
	return period, nil, err
}

func (r *reader) GetIPMethod() (method models.IPMethod, err error) {
	s, err := r.envParams.GetEnv("IP_METHOD", params.Default("cycle"))
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

func (r *reader) GetIPv4Method() (method models.IPMethod, err error) {
	s, err := r.envParams.GetEnv("IPV4_METHOD", params.Default("cycle"))
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

func (r *reader) GetIPv6Method() (method models.IPMethod, err error) {
	s, err := r.envParams.GetEnv("IPV6_METHOD", params.Default("cycle"))
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

func (r *reader) GetExeDir() (dir string, err error) {
	return r.envParams.GetExeDir()
}

func (r *reader) GetHTTPTimeout() (duration time.Duration, err error) {
	return r.envParams.GetHTTPTimeout(libparams.Default("10s"))
}

func (r *reader) GetBackupPeriod() (duration time.Duration, err error) {
	s, err := r.envParams.GetEnv("BACKUP_PERIOD", libparams.Default("0"))
	if err != nil {
		return 0, err
	}
	return time.ParseDuration(s)
}

func (r *reader) GetBackupDirectory() (directory string, err error) {
	return r.envParams.GetEnv("BACKUP_DIRECTORY", libparams.Default("./data"))
}
