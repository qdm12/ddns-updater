package server

import (
	"context"

	"github.com/qdm12/goservices/httpserver"
)

func New(ctx context.Context, address, rootURL string, db Database,
	logger Logger, runner UpdateForcer, configPath string,
	parseConfig ConfigParser,
) (server *httpserver.Server, err error) {
	return httpserver.New(httpserver.Settings{
		Handler: newHandler(ctx, rootURL, db, runner, configPath, parseConfig),
		Address: &address,
		Logger:  logger,
	})
}
