package server

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi"
)

func fileServer(router chi.Router, path string, root http.FileSystem) {
	if path != "/" && path[len(path)-1] != '/' {
		router.Get(path,
			http.RedirectHandler(path+"/", http.StatusMovedPermanently).ServeHTTP)
		path += "/"
	}
	path += "*"

	router.Get(path, func(w http.ResponseWriter, r *http.Request) {
		rctx := chi.RouteContext(r.Context())
		pathPrefix := strings.TrimSuffix(rctx.RoutePattern(), "/*")
		fs := http.StripPrefix(pathPrefix, http.FileServer(root))
		fs.ServeHTTP(w, r)
	})
}
