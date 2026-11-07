package server

import (
	"context"
	"embed"
	htmltemplate "html/template"
	"io/fs"
	"net/http"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type handlers struct {
	ctx context.Context //nolint:containedctx
	// Objects
	db               Database
	runner           UpdateForcer
	logger           Logger
	indexTemplate    *template.Template
	publicIPTemplate *htmltemplate.Template
	configTemplate   *htmltemplate.Template
	// Optional features
	publicIPChecker  PublicIPChecker
	publicIPSettings string
	configStore      ConfigStore
	// publicIPMutex prevents several public IP checks from running at once
	// so the IP echo services are not hammered.
	publicIPMutex sync.Mutex
	// Mockable functions
	timeNow func() time.Time
}

//go:embed ui/*
var uiFS embed.FS

func newHandler(ctx context.Context, rootURL string,
	db Database, runner UpdateForcer, logger Logger, features Features,
) http.Handler {
	indexTemplate := template.Must(template.ParseFS(uiFS, "ui/index.html"))
	publicIPTemplate := htmltemplate.Must(htmltemplate.ParseFS(uiFS, "ui/publicip.html"))
	configTemplate := htmltemplate.Must(htmltemplate.ParseFS(uiFS, "ui/config.html"))

	staticFolder, err := fs.Sub(uiFS, "ui/static")
	if err != nil {
		panic(err)
	}

	handlers := &handlers{
		ctx:              ctx,
		db:               db,
		indexTemplate:    indexTemplate,
		publicIPTemplate: publicIPTemplate,
		configTemplate:   configTemplate,
		publicIPChecker:  features.PublicIPChecker,
		publicIPSettings: features.PublicIPSettings,
		configStore:      features.ConfigStore,
		logger:           logger,
		// TODO build information
		timeNow: time.Now,
		runner:  runner,
	}

	router := chi.NewRouter()

	router.Use(middleware.ClientIPFromRemoteAddr)
	router.Use(middleware.Logger)
	if features.BasicAuthPassword != "" {
		router.Use(basicAuth(features.BasicAuthUsername, features.BasicAuthPassword))
	}
	rootURL = strings.TrimSuffix(rootURL, "/")

	if rootURL != "" {
		router.Handle(rootURL, http.RedirectHandler(rootURL+"/", http.StatusPermanentRedirect))
	}
	router.Get(rootURL+"/", handlers.index)

	router.Get(rootURL+"/update", handlers.update)

	if handlers.publicIPChecker != nil {
		router.Get(rootURL+"/publicip", handlers.publicIPPage)
		router.Post(rootURL+"/publicip/check", handlers.publicIPCheck)
	}

	if handlers.configStore != nil {
		router.Get(rootURL+"/config", handlers.configPage)
		router.Post(rootURL+"/config/validate", handlers.configValidate)
		router.Put(rootURL+"/config", handlers.configSave)
	}

	router.Handle(rootURL+"/static/*", http.StripPrefix(rootURL+"/static/", http.FileServerFS(staticFolder)))

	return router
}
