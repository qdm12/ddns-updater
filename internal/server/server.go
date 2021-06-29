package server

import (
	"context"
	"net/http"
	"time"

	"github.com/qdm12/ddns-updater/internal/data"
	"github.com/qdm12/ddns-updater/internal/update"
	"github.com/qdm12/golibs/logging"
)

type Server interface {
	Run(ctx context.Context, done chan<- struct{})
}

type server struct {
	address string
	logger  logging.Logger
	handler http.Handler
}

func New(ctx context.Context, address, rootURL string, db data.Database, logger logging.Logger,
	runner update.Runner) Server {
	handler := newHandler(ctx, rootURL, db, runner)
	return &server{
		address: address,
		logger:  logger,
		handler: handler,
	}
}

func (s *server) Run(ctx context.Context, done chan<- struct{}) {
	defer close(done)
	server := http.Server{Addr: s.address, Handler: s.handler}
	go func() {
		<-ctx.Done()
		s.logger.Warn("shutting down (context canceled)")
		defer s.logger.Warn("shut down")
		const shutdownGraceDuration = 2 * time.Second
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownGraceDuration)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			s.logger.Error("failed shutting down: %s", err)
		}
	}()
	for ctx.Err() == nil {
		s.logger.Info("listening on %s", s.address)
		err := server.ListenAndServe()
		if err != nil && ctx.Err() == nil { // server crashed
			s.logger.Error(err)
			s.logger.Info("restarting")
		}
	}
}
