package params

import (
	"io/ioutil"
	"net/url"
	"time"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/golibs/logging"
	libparams "github.com/qdm12/golibs/params"
	"github.com/qdm12/golibs/verification"
)

type Reader interface {
	GetSettings(filePath string) (settings []models.Settings, warnings []string, err error)
	GetDataDir(currentDir string) (string, error)
	GetListeningPort() (listeningPort, warning string, err error)
	GetLoggerConfig() (encoding logging.Encoding, level logging.Level, nodeID int, err error)
	GetGotifyURL(setters ...libparams.GetEnvSetter) (URL *url.URL, err error)
	GetGotifyToken(setters ...libparams.GetEnvSetter) (token string, err error)
	GetRootURL(setters ...libparams.GetEnvSetter) (rootURL string, err error)
	GetDelay(setters ...libparams.GetEnvSetter) (duration time.Duration, err error)
	GetExeDir() (dir string, err error)
	GetHTTPTimeout() (duration time.Duration, err error)
	GetBackupPeriod() (duration time.Duration, err error)
	GetBackupDirectory() (directory string, err error)

	// Version getters
	GetVersion() string
	GetBuildDate() string
	GetVcsRef() string
}

type reader struct {
	envParams libparams.EnvParams
	verifier  verification.Verifier
	logger    logging.Logger
	readFile  func(filename string) ([]byte, error)
}

func NewReader(logger logging.Logger) Reader {
	return &reader{
		envParams: libparams.NewEnvParams(),
		verifier:  verification.NewVerifier(),
		logger:    logger,
		readFile:  ioutil.ReadFile,
	}
}

// GetDataDir obtains the data directory from the environment
// variable DATADIR
func (r *reader) GetDataDir(currentDir string) (string, error) {
	return r.envParams.GetEnv("DATADIR", libparams.Default(currentDir+"/data"))
}

func (r *reader) GetListeningPort() (listeningPort, warning string, err error) {
	return r.envParams.GetListeningPort()
}

func (r *reader) GetLoggerConfig() (encoding logging.Encoding, level logging.Level, nodeID int, err error) {
	return r.envParams.GetLoggerConfig()
}

func (r *reader) GetGotifyURL(setters ...libparams.GetEnvSetter) (url *url.URL, err error) {
	return r.envParams.GetGotifyURL()
}

func (r *reader) GetGotifyToken(setters ...libparams.GetEnvSetter) (token string, err error) {
	return r.envParams.GetGotifyToken()
}

func (r *reader) GetRootURL(setters ...libparams.GetEnvSetter) (rootURL string, err error) {
	return r.envParams.GetRootURL()
}

func (r *reader) GetDelay(setters ...libparams.GetEnvSetter) (period time.Duration, err error) {
	// Backward compatibility
	n, err := r.envParams.GetEnvInt("DELAY", libparams.Compulsory()) // TODO change to PERIOD
	if err == nil {                                                  // integer only, treated as seconds
		r.logger.Warn("The value for the duration period of the updater does not have a time unit, you might want to set it to \"%ds\" instead of \"%d\"", n, n)
		return time.Duration(n) * time.Second, nil
	}
	return r.envParams.GetDuration("DELAY", setters...)
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
