package settings

import (
	"fmt"
	"os"

	"github.com/qdm12/gosettings"
	"github.com/qdm12/gosettings/validate"
	"github.com/qdm12/gotree"
)

type Server struct {
	Port    *uint16
	RootURL string
}

func (s *Server) setDefaults() {
	const defaultPort = 8000
	s.Port = gosettings.DefaultPointer(s.Port, defaultPort)
	s.RootURL = gosettings.DefaultString(s.RootURL, "/")
}

func (s Server) mergeWith(other Server) (merged Server) {
	merged.Port = gosettings.MergeWithPointer(s.Port, other.Port)
	merged.RootURL = gosettings.MergeWithString(s.RootURL, other.RootURL)
	return merged
}

func (s Server) Validate() (err error) {
	listeningAddress := ":" + fmt.Sprint(*s.Port)
	err = validate.ListeningAddress(listeningAddress, os.Getuid())
	if err != nil {
		return fmt.Errorf("listening address: %w", err)
	}

	// TODO validate RootURL

	return nil
}

func (s Server) String() string {
	return s.toLinesNode().String()
}

func (s Server) toLinesNode() *gotree.Node {
	node := gotree.New("Server")
	node.Appendf("Port: %d", *s.Port)
	node.Appendf("Root URL: %s", s.RootURL)
	return node
}
