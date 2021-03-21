package publicip

import (
	"errors"
	"fmt"
)

var ErrFetcherUndefined = errors.New("fetcher type undefined")

func (f *fetcher) getSubFetcher() Fetcher {
	fetcherType := f.fetchTypes[0]
	if len(f.fetchTypes) > 1 { // cycling effect
		randInt := int(f.randSource.Int63())
		index := randInt % len(f.fetchTypes)
		fetcherType = f.fetchTypes[index]
	}

	switch fetcherType {
	case DNS:
		return f.dns
	case HTTP:
		return f.http
	default:
		panic(fmt.Sprintf("fetcher type undefined: %d", fetcherType))
	}
}
