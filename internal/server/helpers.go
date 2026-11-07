package server

import (
	"bytes"
	"encoding/json"
	"io"
	"mime"
	"net/http"
)

// templateExecutor is satisfied by both text/template and html/template.
type templateExecutor interface {
	ExecuteTemplate(wr io.Writer, name string, data any) error
}

// renderPage renders a page. It is rendered to a buffer first so that a
// template error does not leave a half written page. The page is never
// cached since it can contain secrets.
func renderPage(w http.ResponseWriter, tmpl templateExecutor, name string, data any) {
	buffer := bytes.NewBuffer(nil)
	err := tmpl.ExecuteTemplate(buffer, name, data)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "failed generating webpage: "+err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(buffer.Bytes())
}

// writeJSON writes v as JSON. The response is never cached since it
// can contain secrets.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	buffer := bytes.NewBuffer(nil)
	err := json.NewEncoder(buffer).Encode(v)
	if err != nil {
		http.Error(w, "encoding response: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(status)
	_, _ = w.Write(buffer.Bytes())
}

// writeJSONError writes {"error": "..."} with the given status.
func writeJSONError(w http.ResponseWriter, status int, message string) {
	if message == "" {
		message = http.StatusText(status)
	}
	writeJSON(w, status, errJSONWrapper{Error: message})
}

// rejectCrossSite refuses requests that the browser flags as coming
// from another site, and returns true if it did.
func rejectCrossSite(w http.ResponseWriter, r *http.Request) (rejected bool) {
	if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
		writeJSONError(w, http.StatusForbidden, "cross-site requests are not allowed")
		return true
	}
	return false
}

// requireJSON protects state changing endpoints against cross-site request
// forgery: browsers cannot send a cross-origin application/json request
// without a CORS preflight, which this server never approves.
// It returns true if the request was refused.
func requireJSON(w http.ResponseWriter, r *http.Request) (rejected bool) {
	if rejectCrossSite(w, r) {
		return true
	}
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		writeJSONError(w, http.StatusUnsupportedMediaType, "content type must be application/json")
		return true
	}
	return false
}
