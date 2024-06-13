package health

import (
	"context"

	"github.com/qdm12/goservices/httpserver"
)

func NewServer(address string, logger Logger, healthcheck func(context.Context) error) (
	server *httpserver.Server, err error) {
	name := "health"
	return httpserver.New(httpserver.Settings{
		Handler: newHandler(healthcheck),
		Name:    &name,
		Address: &address,
		Logger:  logger,
	})
}
