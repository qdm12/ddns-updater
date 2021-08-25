package config

import (
	"fmt"

	"github.com/qdm12/golibs/params"
)

type Server struct {
	Port    uint16
	RootURL string
}

func (s *Server) get(env params.Interface) (warning string, err error) {
	s.RootURL, err = env.RootURL("ROOT_URL")
	if err != nil {
		return "", fmt.Errorf("%w: for environment variable ROOT_URL", err)
	}

	s.Port, warning, err = env.ListeningPort("LISTENING_PORT", params.Default("8000"))
	if err != nil {
		return "", fmt.Errorf("%w: for environment variable LISTENING_PORT", err)
	}

	return warning, err
}
