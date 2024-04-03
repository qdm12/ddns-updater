package server

import (
	"context"
	"embed"
	"net/http"
	"strings"
	"text/template"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type handlers struct {
	ctx context.Context //nolint:containedctx
	// Objects
	db            Database
	runner        UpdateForcer
	indexTemplate *template.Template
	// Mockable functions
	timeNow func() time.Time
}

//go:embed ui/*
var uiFS embed.FS

func newHandler(ctx context.Context, rootURL string,
	db Database, runner UpdateForcer) http.Handler {
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
	rootURL = strings.TrimSuffix(rootURL, "/")

	if rootURL != "" {
		router.Handle(rootURL, http.RedirectHandler(rootURL+"/", http.StatusPermanentRedirect))
	}
	router.Get(rootURL+"/", handlers.index)

	router.Get(rootURL+"/update", handlers.update)

	router.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("ui/static"))))

	return router
}
