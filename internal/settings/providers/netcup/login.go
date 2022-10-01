package netcup

import (
	"fmt"
	"net/http"

	"github.com/qdm12/ddns-updater/internal/settings/errors"
	"golang.org/x/net/context"
)

func (p *Provider) login(ctx context.Context, client *http.Client) (
	session string, err error) {
	type jsonParams struct {
		APIKey         string `json:"apikey"`
		APIPassword    string `json:"apipassword"`
		CustomerNumber string `json:"customernumber"`
	}

	type jsonRequest struct {
		Action string     `json:"action"`
		Param  jsonParams `json:"param"`
	}

	requestBody := jsonRequest{
		Action: "login",
		Param: jsonParams{
			APIKey:         p.apiKey,
			APIPassword:    p.password,
			CustomerNumber: p.customerNumber,
		},
	}

	var responseBody struct {
		Session string `json:"apisessionid"`
	}
	err = doJSONHTTP(ctx, client, requestBody, &responseBody)
	if err != nil {
		return "", fmt.Errorf("doing JSON HTTP exchange: %w", err)
	}

	if responseBody.Session == "" {
		return "", fmt.Errorf("%w", errors.ErrSessionIsEmpty)
	}

	return responseBody.Session, nil
}
