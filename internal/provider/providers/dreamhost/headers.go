package dreamhost

import (
	"net/http"

	"github.com/qdm12/ddns-updater/internal/provider/headers"
)

func setHeaders(request *http.Request) {
	headers.SetUserAgent(request)
	headers.SetAccept(request, "application/json")
}
