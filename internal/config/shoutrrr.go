package config

import (
	"fmt"
	"net/url"
	"path"

	"github.com/containrrr/shoutrrr"
	"github.com/qdm12/gosettings"
	"github.com/qdm12/gosettings/reader"
	"github.com/qdm12/gotree"
)

type Shoutrrr struct {
	Addresses    []string
	DefaultTitle string
}

func (s *Shoutrrr) setDefaults() {
	s.Addresses = gosettings.DefaultSlice(s.Addresses, []string{})
	s.DefaultTitle = gosettings.DefaultComparable(s.DefaultTitle, "DDNS Updater")
}

func (s Shoutrrr) Validate() (err error) {
	_, err = shoutrrr.CreateSender(s.Addresses...)
	if err != nil {
		return fmt.Errorf("shoutrrr addresses: %w", err)
	}
	return nil
}

func (s Shoutrrr) String() string {
	return s.ToLinesNode().String()
}

func (s Shoutrrr) ToLinesNode() *gotree.Node {
	if len(s.Addresses) == 0 {
		return nil // no address means shoutrrr is disabled
	}

	node := gotree.New("Shoutrrr")
	node.Appendf("Default title: %s", s.DefaultTitle)

	childNode := node.Appendf("Addresses")
	for _, address := range s.Addresses {
		childNode.Appendf(address)
	}

	return node
}

func (s *Shoutrrr) read(r *reader.Reader, warner Warner) (err error) {
	s.Addresses = r.CSV("SHOUTRRR_ADDRESSES", reader.ForceLowercase(false))

	// Retro-compatibility: GOTIFY_URL and GOTIFY_TOKEN
	gotifyURLString := r.Get("GOTIFY_URL", reader.ForceLowercase(false))
	if gotifyURLString != nil {
		handleDeprecated(warner, "GOTIFY_URL", "SHOUTRRR_ADDRESSES")
		gotifyURL, err := url.Parse(*gotifyURLString)
		if err != nil {
			return fmt.Errorf("gotify URL: %w", err)
		}

		gotifyToken := r.String("GOTIFY_TOKEN", reader.ForceLowercase(false))
		handleDeprecated(warner, "GOTIFY_TOKEN", "SHOUTRRR_ADDRESSES")
		gotifyShoutrrrAddress := gotifyURLTokenToShoutrrr(gotifyURL, gotifyToken)
		s.Addresses = append(s.Addresses, gotifyShoutrrrAddress)
	}

	// Retro-compatibility
	shoutrrrParamsCSV := r.Get("SHOUTRRR_PARAMS")
	if shoutrrrParamsCSV != nil {
		warner.Warnf("SHOUTRRR_PARAMS is disabled, you can use SHOUTRRR_DEFAULT_TITLE and SHOUTRRR_ADDRESSES")
	}

	s.DefaultTitle = r.String("SHOUTRRR_DEFAULT_TITLE", reader.ForceLowercase(false))
	return nil
}

func gotifyURLTokenToShoutrrr(url *url.URL, token string) (address string) {
	hostAndPath := path.Join(url.Host, url.Path)
	address = "gotify://" + hostAndPath + "/" + token
	if url.Scheme == "http" {
		address += "?DisableTLS=Yes"
	}
	return address
}
