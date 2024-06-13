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
	Enabled          *bool
	ListeningAddress string
	RootURL          string
}

func (s *Server) setDefaults() {
	s.Enabled = gosettings.DefaultPointer(s.Enabled, true)
	s.ListeningAddress = gosettings.DefaultComparable(s.ListeningAddress, ":8000")
	s.RootURL = gosettings.DefaultComparable(s.RootURL, "/")
}

func (s Server) Validate() (err error) {
	err = validate.ListeningAddress(s.ListeningAddress, os.Getuid())
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
	if !*s.Enabled {
		return gotree.New("Server: disabled")
	}
	node := gotree.New("Server")
	node.Appendf("Listening address: %s", s.ListeningAddress)
	node.Appendf("Root URL: %s", s.RootURL)
	return node
}

func (s *Server) read(reader *reader.Reader, warner Warner) (err error) {
	s.Enabled, err = reader.BoolPtr("SERVER_ENABLED")
	if err != nil {
		return err
	}

	s.RootURL = reader.String("ROOT_URL")

	// Retro-compatibility
	port, err := reader.Uint16Ptr("LISTENING_PORT") // TODO change to address
	if err != nil {
		handleDeprecated(warner, "LISTENING_PORT", "LISTENING_ADDRESS")
		return err
	} else if port != nil {
		s.ListeningAddress = fmt.Sprintf(":%d", *port)
	}

	s.ListeningAddress = reader.String("LISTENING_ADDRESS")

	return err
}
