package server

import (
	"context"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/goservices/httpserver"
)

func New(ctx context.Context, address, rootURL string, db Database,
	logger Logger, runner UpdateForcer, buildInfo models.BuildInformation,
) (server *httpserver.Server, err error) {
	return httpserver.New(httpserver.Settings{
		Handler: newHandler(ctx, rootURL, db, runner, buildInfo),
		Address: &address,
		Logger:  logger,
	})
}
