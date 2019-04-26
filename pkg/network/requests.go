package network

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"fmt"
)

// DoHTTPRequest performs an HTTP request and returns the status, content and eventual error
func DoHTTPRequest(client *http.Client, request *http.Request) (status int, content []byte, err error) {
	response, err := client.Do(request)
	if err != nil {
		return status, nil, err
	}
	content, err = ioutil.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		return status, nil, err
	}
	return response.StatusCode, content, nil
}

// GetContent returns the content and eventual error from an HTTP GET to a given URL
func GetContent(httpClient *http.Client, URL string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, URL, nil)
	if err != nil {
		return nil, fmt.Errorf("cannot GET content of URL %s: %s", URL, err)
	}
	status, content, err := DoHTTPRequest(httpClient, req)
	if err != nil {
		return nil, fmt.Errorf("cannot GET content of URL %s: %s", URL, err)
	}
	if status != 200 {
		return nil, fmt.Errorf("cannot GET content of URL %s (status %d)", URL, status)
	}
	return content, nil
}

// Used for GoDaddy only
func BuildHTTPPutJSONAuth(url, authorizationHeader string, body interface{}) (request *http.Request, err error) {
	jsonData, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	request, err = http.NewRequest(http.MethodPut, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Authorization", authorizationHeader)
	request.Header.Set("Content-Type", "application/json; charset=utf-8")
	return request, nil
}
