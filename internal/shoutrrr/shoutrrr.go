package shoutrrr

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/containrrr/shoutrrr"
	"github.com/containrrr/shoutrrr/pkg/router"
)

type Client struct {
	serviceRouter *router.ServiceRouter
	serviceNames  []string
	defaultTitle  string
	logger        Erroer
}

func New(settings Settings) (client *Client, err error) {
	settings.setDefaults()
	err = settings.validate()
	if err != nil {
		return nil, fmt.Errorf("validating settings: %w", err)
	}

	for i, address := range settings.Addresses {
		settings.Addresses[i] = addDefaultTitle(address, settings.DefaultTitle)
	}

	serviceRouter, err := shoutrrr.CreateSender(settings.Addresses...)
	if err != nil {
		return nil, fmt.Errorf("creating service router: %w", err)
	}

	serviceNames := make([]string, len(settings.Addresses))
	for i, address := range settings.Addresses {
		serviceNames[i] = strings.Split(address, ":")[0]
	}

	return &Client{
		serviceRouter: serviceRouter,
		serviceNames:  serviceNames,
		defaultTitle:  settings.DefaultTitle,
		logger:        settings.Logger,
	}, nil
}

func (c *Client) Notify(message string) {
	errs := c.serviceRouter.Send(message, nil)
	for i, err := range errs {
		if err != nil {
			c.logger.Error(c.serviceNames[i] + ": " + err.Error())
		}
	}
}

func addDefaultTitle(address, defaultTitle string) (updatedAddress string) {
	u, err := url.Parse(address)
	if err != nil {
		// address should already be validated
		panic(fmt.Sprintf("parsing address as url: %s", err))
	}

	urlValues := u.Query()
	if urlValues.Has("title") {
		return address
	}

	urlValues.Set("title", defaultTitle)
	u.RawQuery = urlValues.Encode()
	return u.String()
}
