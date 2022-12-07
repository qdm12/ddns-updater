package update

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/qdm12/ddns-updater/internal/settings/utils"
)

//go:generate mockgen -destination=mock_$GOPACKAGE/$GOFILE . DebugLogger

type DebugLogger interface {
	Debug(s string)
}

func makeLogClient(client *http.Client, logger DebugLogger) (newClient *http.Client) {
	newClient = &http.Client{
		Timeout: client.Timeout,
	}

	originalTransport := client.Transport
	if originalTransport == nil {
		originalTransport = http.DefaultTransport
	}

	transport, ok := originalTransport.(*http.Transport)
	if !ok {
		panic(fmt.Sprintf("transport %T is not *http.Transport", originalTransport))
	}

	clonedTransport := transport.Clone()

	newClient.Transport = &loggingRoundTripper{
		proxied: clonedTransport,
		logger:  logger,
	}

	return newClient
}

type loggingRoundTripper struct {
	proxied http.RoundTripper
	logger  DebugLogger
}

func (lrt *loggingRoundTripper) RoundTrip(request *http.Request) (
	response *http.Response, err error) {
	lrt.logger.Debug(requestToString(request))

	response, err = lrt.proxied.RoundTrip(request)
	if err != nil {
		return response, err
	}

	lrt.logger.Debug(responseToString(response))

	return response, nil
}

func requestToString(request *http.Request) (s string) {
	s = request.Method + " " + request.URL.String()

	if request.Header != nil {
		s += " | headers: " + headerToString(request.Header)
	}

	if request.Body != nil {
		newBody, bodyString := readAndResetBody(request.Body)
		request.Body = newBody
		s += " | body: " + bodyString
	}

	return s
}

func responseToString(response *http.Response) (s string) {
	s = response.Status

	if response.Header != nil {
		s += " | headers: " + headerToString(response.Header)
	}

	if response.Body != nil {
		newBody, bodyString := readAndResetBody(response.Body)
		response.Body = newBody
		s += " | body: " + bodyString
	}

	return s
}

func headerToString(header http.Header) (s string) {
	headers := make([]string, 0, len(header))
	for key, values := range header {
		headerString := key + ": " + strings.Join(values, ",")
		headers = append(headers, headerString)
	}
	return strings.Join(headers, "; ")
}

func readAndResetBody(body io.ReadCloser) (
	newBody io.ReadCloser, bodyString string) {
	b, err := io.ReadAll(body)
	if err != nil {
		bodyString = "error reading body: " + err.Error()
	} else {
		bodyString = utils.ToSingleLine(string(b))
		_ = body.Close()
		newBody = io.NopCloser(bytes.NewBuffer(b))
	}
	return newBody, bodyString
}
