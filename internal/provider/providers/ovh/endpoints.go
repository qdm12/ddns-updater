package ovh

import (
	"errors"
	"fmt"
	"net/url"
)

var ErrEndpointUnknown = errors.New("short endpoint name unknown")

func convertShortEndpoint(shortEndpoint string) (url *url.URL, err error) {
	switch shortEndpoint {
	case "", "ovh-eu": // default
		return url.Parse("https://eu.api.ovh.com/1.0")
	case "ovh-ca":
		return url.Parse("https://ca.api.ovh.com/1.0")
	case "ovh-us":
		return url.Parse("https://api.us.ovhcloud.com/1.0")
	case "kimsufi-eu":
		return url.Parse("https://eu.api.kimsufi.com/1.0")
	case "kimsufi-ca":
		return url.Parse("https://ca.api.kimsufi.com/1.0")
	case "soyoustart-eu":
		return url.Parse("https://eu.api.soyoustart.com/1.0")
	case "soyoustart-ca":
		return url.Parse("https://ca.api.soyoustart.com/1.0")
	}
	return nil, fmt.Errorf("%w: %s", ErrEndpointUnknown, shortEndpoint)
}
