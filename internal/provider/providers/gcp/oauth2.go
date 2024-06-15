package gcp

import (
	"context"
	"fmt"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

func createOauth2Client(ctx context.Context, client *http.Client, credentialsJSON []byte) (
	oauth2Client *http.Client, err error) {
	scopes := []string{
		"https://www.googleapis.com/auth/cloud-platform",
		"https://www.googleapis.com/auth/cloud-platform.read-only",
		"https://www.googleapis.com/auth/ndev.clouddns.readonly",
		"https://www.googleapis.com/auth/ndev.clouddns.readwrite",
	}
	credentials, err := google.CredentialsFromJSON(ctx, credentialsJSON, scopes...)
	if err != nil {
		return nil, fmt.Errorf("creating Google credentials: %w", err)
	}
	oauth2Client = &http.Client{
		Timeout: client.Timeout,
		Transport: &oauth2.Transport{
			Base:   client.Transport,
			Source: oauth2.ReuseTokenSource(nil, credentials.TokenSource),
		},
	}

	return oauth2Client, nil
}
