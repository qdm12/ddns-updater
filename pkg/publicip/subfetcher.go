package publicip

import (
	"errors"
	"fmt"
)

var ErrFetcherUndefined = errors.New("fetcher type undefined")

func (f *fetcher) getSubFetcher() (subFetcher Fetcher, err error) {
	fetcherType := f.fetchTypes[0]
	if len(f.fetchTypes) > 1 { // cycling effect
		randInt := int(f.randSource.Int63())
		index := randInt % len(f.fetchTypes)
		fetcherType = f.fetchTypes[index]
	}

	switch fetcherType {
	case dnsFetch:
		return f.dns, nil
	case httpFetch:
		return f.http, nil
	default:
		return nil, fmt.Errorf("%w: %d", ErrFetcherUndefined, fetcherType)
	}
}
