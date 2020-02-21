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

type ParamsReader interface {
	GetSettings(filePath string) (settings []models.Settings, warnings []string, err error)
	GetDataDir(currentDir string) (string, error)
	GetListeningPort() (listeningPort, warning string, err error)
	GetLoggerConfig() (encoding logging.Encoding, level logging.Level, nodeID int, err error)
	GetGotifyURL(setters ...libparams.GetEnvSetter) (URL *url.URL, err error)
	GetGotifyToken(setters ...libparams.GetEnvSetter) (token string, err error)
	GetRootURL(setters ...libparams.GetEnvSetter) (rootURL string, err error)
	GetDuration(key string, setters ...libparams.GetEnvSetter) (duration time.Duration, err error)
	GetExeDir() (dir string, err error)
	GetHTTPTimeout(setters ...libparams.GetEnvSetter) (duration time.Duration, err error)
}

type params struct {
	envParams libparams.EnvParams
	verifier  verification.Verifier
	readFile  func(filename string) ([]byte, error)
}

func NewParamsReader() ParamsReader {
	return &params{
		envParams: libparams.NewEnvParams(),
		verifier:  verification.NewVerifier(),
		readFile:  ioutil.ReadFile,
	}
}

// GetDataDir obtains the data directory from the environment
// variable DATADIR
func (p *params) GetDataDir(currentDir string) (string, error) {
	return p.envParams.GetEnv("DATADIR", libparams.Default(currentDir+"/data"))
}

func (p *params) GetListeningPort() (listeningPort, warning string, err error) {
	return p.envParams.GetListeningPort()
}

func (p *params) GetLoggerConfig() (encoding logging.Encoding, level logging.Level, nodeID int, err error) {
	return p.envParams.GetLoggerConfig()
}

func (p *params) GetGotifyURL(setters ...libparams.GetEnvSetter) (URL *url.URL, err error) {
	return p.envParams.GetGotifyURL()
}

func (p *params) GetGotifyToken(setters ...libparams.GetEnvSetter) (token string, err error) {
	return p.envParams.GetGotifyToken()
}

func (p *params) GetRootURL(setters ...libparams.GetEnvSetter) (rootURL string, err error) {
	return p.envParams.GetRootURL()
}

func (p *params) GetDuration(key string, setters ...libparams.GetEnvSetter) (duration time.Duration, err error) {
	return p.envParams.GetDuration(key, setters...)
}

func (p *params) GetExeDir() (dir string, err error) {
	return p.envParams.GetExeDir()
}

func (p *params) GetHTTPTimeout(setters ...libparams.GetEnvSetter) (duration time.Duration, err error) {
	return p.envParams.GetHTTPTimeout()
}
