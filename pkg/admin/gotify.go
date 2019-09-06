package admin

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/gotify/go-api-client/v2/auth"
	"github.com/gotify/go-api-client/v2/client"
	"github.com/gotify/go-api-client/v2/client/message"
	"github.com/gotify/go-api-client/v2/gotify"
	"github.com/gotify/go-api-client/v2/models"
	"go.uber.org/zap"
)

// Gotify contains the Gotify API client and the token for the application
type Gotify struct {
	client *client.GotifyREST
	token  string
}

// NewGotify creates an API client with the token for the Gotify server
func NewGotify(URL *url.URL, token string, httpClient *http.Client) (g *Gotify, err error) {
	if URL == nil {
		return &Gotify{}, fmt.Errorf("Gotify URL not provided")
	} else if token == "" {
		return &Gotify{}, fmt.Errorf("Gotify token not provided")
	}
	client := gotify.NewClient(URL, httpClient)
	_, err = client.Version.GetVersion(nil)
	if err != nil {
		return &Gotify{}, fmt.Errorf("cannot communicate with Gotify server: %w", err)
	}
	return &Gotify{client: client, token: token}, nil
}

// Notify sends a notification to the Gotify server
func (g *Gotify) Notify(title string, priority int, content string, args ...interface{}) {
	if g == nil || g.client == nil {
		return
	}
	content = fmt.Sprintf(content, args...)
	params := message.NewCreateMessageParams()
	params.Body = &models.MessageExternal{
		Title:    title,
		Message:  content,
		Priority: priority,
	}
	_, err := g.client.Message.CreateMessage(params, auth.TokenAuth(g.token))
	if err != nil {
		zap.S().Warn("cannot send message to Gotify: %s", err)
	}
}
