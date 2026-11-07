package server

import (
	"context"

	"github.com/qdm12/goservices/httpserver"
)

// Features are the optional features of the web UI.
type Features struct {
	// PublicIPChecker is used by the public IP check page.
	// The page is not served if it is nil.
	PublicIPChecker PublicIPChecker
	// PublicIPSettings is a description of the public IP fetching settings
	// shown on the public IP check page.
	PublicIPSettings string
	// ConfigStore is used by the configuration editor page.
	// The page is not served if it is nil.
	ConfigStore ConfigStore
	// BasicAuthUsername and BasicAuthPassword protect the whole web UI with
	// HTTP basic authentication. It is disabled if the password is empty.
	BasicAuthUsername string
	BasicAuthPassword string
}

func New(ctx context.Context, address, rootURL string, db Database,
	logger Logger, runner UpdateForcer, features Features,
) (server *httpserver.Server, err error) {
	return httpserver.New(httpserver.Settings{
		Handler: newHandler(ctx, rootURL, db, runner, logger, features),
		Address: &address,
		Logger:  logger,
	})
}
