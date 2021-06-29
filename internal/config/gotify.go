package config

import (
	"net/url"

	"github.com/qdm12/golibs/params"
)

type Gotify struct {
	URL   *url.URL
	Token string
}

func (g *Gotify) get(env params.Env) (err error) {
	g.URL, err = env.URL("GOTIFY_URL")
	if err != nil {
		return err
	} else if g.URL == nil {
		return nil
	}

	g.Token, err = env.Get("GOTIFY_TOKEN", params.CaseSensitiveValue(),
		params.Compulsory(), params.Unset())
	return err
}
