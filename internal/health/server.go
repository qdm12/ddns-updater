package health

import (
	"github.com/qdm12/goservices/httpserver"
)

func NewServer(address string, logger Logger, healthcheck func() error) (
	server *httpserver.Server, err error) {
	name := "health"
	return httpserver.New(httpserver.Settings{
		Handler: newHandler(healthcheck),
		Name:    &name,
		Address: &address,
		Logger:  logger,
	})
}
