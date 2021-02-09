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
)

type handlers struct {
	ctx context.Context
	// Objects
	db            data.Database
	runner        update.Runner
	indexTemplate *template.Template
	// Mockable functions
	timeNow func() time.Time
}

func newHandler(ctx context.Context, rootURL, uiDir string,
	db data.Database, runner update.Runner) http.Handler {
	indexTemplate := template.Must(template.ParseFiles(uiDir + "/index.html"))

	handlers := &handlers{
		ctx:           ctx,
		db:            db,
		indexTemplate: indexTemplate,
		// TODO build information
		timeNow: time.Now,
		runner:  runner,
	}

	router := chi.NewRouter()

	router.Use(middleware.Logger, middleware.CleanPath)

	router.Get(rootURL+"/", handlers.index)

	router.Get(rootURL+"/update", handlers.update)

	// UI file server for other paths
	fileServer(router, rootURL+"/", http.Dir(uiDir))

	return router
}
