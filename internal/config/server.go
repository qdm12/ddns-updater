package config

import (
	"fmt"
	"os"

	"github.com/qdm12/gosettings"
	"github.com/qdm12/gosettings/reader"
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
	s.RootURL = gosettings.DefaultComparable(s.RootURL, "/")
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

func (s *Server) read(reader *reader.Reader) (err error) {
	s.RootURL = reader.String("ROOT_URL")
	s.Port, err = reader.Uint16Ptr("LISTENING_PORT") // TODO change to address
	return err
}
