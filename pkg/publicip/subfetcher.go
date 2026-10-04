package publicip

import (
	"sync/atomic"
)

func (f *Fetcher) getSubFetcher() ipFetcher { //nolint:ireturn
	fetcher := f.fetchers[0]
	if len(f.fetchers) > 1 { // cycling effect
		index := int(atomic.AddUint32(f.counter, 1)) % len(f.fetchers)
		fetcher = f.fetchers[index]
	}
	return fetcher
}
