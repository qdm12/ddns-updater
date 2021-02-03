package server

import (
	"net/http"
	"text/template"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/qdm12/ddns-updater/internal/data"
)

type handlers struct {
	// Objects
	forceUpdate   chan<- struct{}
	db            data.Database
	indexTemplate *template.Template
	// Mockable functions
	timeNow func() time.Time
}

func newHandler(rootURL, uiDir string, db data.Database, forceUpdate chan<- struct{}) http.Handler {
	indexTemplate := template.Must(template.ParseFiles(uiDir + "/index.html"))

	handlers := &handlers{
		db:            db,
		indexTemplate: indexTemplate,
		// TODO build information
		timeNow:     time.Now,
		forceUpdate: forceUpdate,
	}

	router := chi.NewRouter()

	router.Use(middleware.Logger, middleware.CleanPath)

	router.Get(rootURL+"/", handlers.index)

	router.Get(rootURL+"/update", handlers.update)

	// UI file server for other paths
	fileServer(router, rootURL+"/", http.Dir(uiDir))

	return router
}
