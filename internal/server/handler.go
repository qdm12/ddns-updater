package server

import (
	"context"
	"embed"
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

//go:embed ui/*
var uiFS embed.FS //nolint:gochecknoglobals

func newHandler(ctx context.Context, rootURL string,
	db data.Database, runner update.Runner) http.Handler {
	indexTemplate := template.Must(template.ParseFS(uiFS, "ui/index.html"))

	handlers := &handlers{
		ctx:           ctx,
		db:            db,
		indexTemplate: indexTemplate,
		// TODO build information
		timeNow: time.Now,
		runner:  runner,
	}

	router := chi.NewRouter()

	router.Use(middleware.Logger)

	router.Get(rootURL+"/", handlers.index)

	router.Get(rootURL+"/update", handlers.update)

	return router
}
