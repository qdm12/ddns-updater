package ionos

import (
	"net/http"

	"github.com/qdm12/ddns-updater/internal/provider/headers"
)

type apiZone struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type apiRecord struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	RootName string `json:"rootName"`
	Type     string `json:"type"`
	Content  string `json:"content"`
	TTL      uint32 `json:"ttl"`
	Prio     uint32 `json:"prio"`
	Disabled bool   `json:"disabled"`
}

func (p *Provider) setHeaders(request *http.Request) {
	headers.SetUserAgent(request)
	headers.SetAccept(request, "application/json")
	headers.SetXAPIKey(request, p.apiKey)
	switch request.Method {
	case http.MethodPost, http.MethodPut:
		headers.SetContentType(request, "application/json")
	}
}
