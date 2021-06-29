package config

import (
	"github.com/qdm12/golibs/params"
)

type Server struct {
	Port    uint16
	RootURL string
}

func (s *Server) get(env params.Env) (warning string, err error) {
	s.RootURL, err = env.RootURL("ROOT_URL")
	if err != nil {
		return "", err
	}
	s.Port, warning, err = env.ListeningPort("LISTENING_PORT", params.Default("8000"))
	return warning, err
}
