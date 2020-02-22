package network

import (
	"bytes"
	"encoding/json"
	"net/http"
)

// BuildHTTPPut is used for GoDaddy and Cloudflare only
func BuildHTTPPut(URL string, body interface{}) (request *http.Request, err error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	request, err = http.NewRequest(http.MethodPut, URL, bytes.NewBuffer(b))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	return request, nil
}
