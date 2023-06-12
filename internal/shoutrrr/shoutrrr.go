package shoutrrr

import (
	"fmt"
	"strings"

	"github.com/containrrr/shoutrrr"
	"github.com/containrrr/shoutrrr/pkg/router"
	"github.com/containrrr/shoutrrr/pkg/types"
)

type Client struct {
	serviceRouter *router.ServiceRouter
	serviceNames  []string
	params        types.Params
	logger        Erroer
}

func New(settings Settings) (client *Client, err error) {
	settings.SetDefaults()
	err = settings.Validate()
	if err != nil {
		return nil, fmt.Errorf("validating settings: %w", err)
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
		params:        settings.Params,
		logger:        settings.Logger,
	}, nil
}

func (c *Client) Notify(message string) {
	errs := c.serviceRouter.Send(message, &c.params)
	for i, err := range errs {
		if err != nil {
			c.logger.Error(c.serviceNames[i] + ": " + err.Error())
		}
	}
}
