package server

import (
	"context"
	"embed"
	"html/template"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type handlers struct {
	ctx context.Context //nolint:containedctx
	// Objects
	db           Database
	runner       UpdateForcer
	reloader     ConfigReloader
	entryManager EntryManager
	templates    *template.Template
	rootURL      string
	// Mockable functions
	timeNow func() time.Time
}

//go:embed ui/*
var uiFS embed.FS

func newHandler(ctx context.Context, rootURL string,
	db Database, runner UpdateForcer, reloader ConfigReloader,
	entryManager EntryManager,
) http.Handler {
	templates := template.Must(template.ParseFS(uiFS, "ui/*.html"))

	staticFolder, err := fs.Sub(uiFS, "ui/static")
	if err != nil {
		panic(err)
	}

	rootURL = strings.TrimSuffix(rootURL, "/")

	handlers := &handlers{
		ctx:       ctx,
		db:        db,
		templates: templates,
		rootURL:   rootURL,
		// TODO build information
		timeNow:      time.Now,
		runner:       runner,
		reloader:     reloader,
		entryManager: entryManager,
	}

	router := chi.NewRouter()

	router.Use(middleware.ClientIPFromRemoteAddr)
	router.Use(middleware.Logger)

	if rootURL != "" {
		router.Handle(rootURL, http.RedirectHandler(rootURL+"/", http.StatusPermanentRedirect))
	}
	router.Get(rootURL+"/", handlers.index)

	router.Get(rootURL+"/update", handlers.update)

	router.Get(rootURL+"/reload", handlers.reload)

	router.Get(rootURL+"/entries", handlers.entries)
	router.Get(rootURL+"/entries/new", handlers.entryNew)
	router.Get(rootURL+"/entries/{index}/edit", handlers.entryEdit)
	router.Post(rootURL+"/entries", handlers.entryCreate)
	router.Post(rootURL+"/entries/{index}", handlers.entryUpdate)
	router.Post(rootURL+"/entries/{index}/delete", handlers.entryDelete)

	router.Handle(rootURL+"/static/*", http.StripPrefix(rootURL+"/static/", http.FileServerFS(staticFolder)))

	return router
}
