package server

import (
	"context"
	"net/http"
	"text/template"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/qdm12/ddns-updater/internal/data"
	"github.com/qdm12/ddns-updater/internal/update"
	"github.com/qdm12/golibs/logging"
)

type handlers struct {
	ctx context.Context
	// Objects
	db            data.Database
	runner        update.Runner
	indexTemplate *template.Template
	logger        logging.Logger
	// Mockable functions
	timeNow func() time.Time
}

func newHandler(ctx context.Context, rootURL, uiDir string, logger logging.Logger,
	db data.Database, runner update.Runner) http.Handler {
	indexTemplate := template.Must(template.ParseFiles(uiDir + "/index.html"))

	handlers := &handlers{
		ctx:           ctx,
		db:            db,
		indexTemplate: indexTemplate,
		logger:        logger,
		// TODO build information
		timeNow: time.Now,
		runner:  runner,
	}

	router := chi.NewRouter()

	router.Use(middleware.Logger, middleware.CleanPath) // TODO use custom logging middleware

	router.Get(rootURL+"/", handlers.index)
	router.Get(rootURL+"/api/v1/records", handlers.getRecords)

	router.Get(rootURL+"/update", handlers.update)

	// UI file server for other paths
	fileServer(router, rootURL+"/", http.Dir(uiDir))

	return router
}
