package hetznernetworking

import (
	"net/http"

	"github.com/qdm12/ddns-updater/internal/provider/headers"
)

func (p *Provider) setHeaders(request *http.Request) {
	headers.SetUserAgent(request)
	headers.SetContentType(request, "application/json")
	headers.SetAccept(request, "application/json")
	request.Header.Set("Authorization", "Bearer "+p.token)
}
