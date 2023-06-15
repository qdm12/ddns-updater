package namecom

import (
	"net/http"

	"github.com/qdm12/ddns-updater/internal/provider/headers"
)

func setHeaders(request *http.Request) {
	headers.SetContentType(request, "application/json")
	headers.SetAccept(request, "application/json")
	headers.SetUserAgent(request)
}
