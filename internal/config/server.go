package config

import (
	"errors"
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
	// ConfigEditor enables the configuration editor page of the web UI.
	ConfigEditor *bool
	// AuthUsername and AuthPassword enable HTTP basic authentication
	// for the whole web UI, if the password is not empty.
	AuthUsername string
	AuthPassword string
}

func (s *Server) setDefaults() {
	s.Enabled = gosettings.DefaultPointer(s.Enabled, true)
	s.ListeningAddress = gosettings.DefaultComparable(s.ListeningAddress, ":8000")
	s.RootURL = gosettings.DefaultComparable(s.RootURL, "/")
	s.ConfigEditor = gosettings.DefaultPointer(s.ConfigEditor, true)
}

var ErrAuthIncomplete = errors.New("username and password must be set together")

func (s Server) Validate() (err error) {
	err = validate.ListeningAddress(s.ListeningAddress, os.Getuid())
	if err != nil {
		return fmt.Errorf("listening address: %w", err)
	}

	// TODO validate RootURL

	if (s.AuthUsername == "") != (s.AuthPassword == "") {
		return fmt.Errorf("%w: SERVER_AUTH_USERNAME and SERVER_AUTH_PASSWORD",
			ErrAuthIncomplete)
	}

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
	node.Appendf("Configuration editor: %s", gosettings.BoolToYesNo(s.ConfigEditor))
	if s.AuthPassword != "" {
		node.Appendf("Basic authentication: enabled (user %s)", s.AuthUsername)
	} else {
		node.Append("Basic authentication: disabled")
	}
	return node
}

func (s *Server) read(reader *reader.Reader, warner Warner) (err error) {
	s.Enabled, err = reader.BoolPtr("SERVER_ENABLED")
	if err != nil {
		return err
	}

	s.RootURL = reader.String("ROOT_URL")

	s.ConfigEditor, err = reader.BoolPtr("SERVER_CONFIG_EDITOR")
	if err != nil {
		return err
	}
	s.AuthUsername = reader.String("SERVER_AUTH_USERNAME")
	s.AuthPassword = reader.String("SERVER_AUTH_PASSWORD")

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
