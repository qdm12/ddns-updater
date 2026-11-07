package publicip

import (
	"context"
	"errors"
	"net/netip"
	"testing"

	"github.com/qdm12/ddns-updater/pkg/publicip/privateip"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errFakePrivate = errors.New("fake private failure")

type fakeSubFetcher struct {
	ip  netip.Addr
	err error
}

func (f fakeSubFetcher) IP(context.Context) (netip.Addr, error)  { return f.ip, f.err }
func (f fakeSubFetcher) IP4(context.Context) (netip.Addr, error) { return f.ip, f.err }
func (f fakeSubFetcher) IP6(context.Context) (netip.Addr, error) { return f.ip, f.err }

func Test_NewFetcher_privateIPInterfacePropagation(t *testing.T) {
	t.Parallel()

	t.Run("interface missing", func(t *testing.T) {
		t.Parallel()

		fetcher, err := NewFetcher(DNSSettings{}, HTTPSettings{}, PrivateIPSettings{Enabled: true})

		require.ErrorIs(t, err, privateip.ErrInterfaceNotSet)
		assert.Nil(t, fetcher)
	})

	// Read-only probe: looks up a name in the host's real interface list that cannot exist.
	t.Run("interface reaches the live retriever", func(t *testing.T) {
		t.Parallel()

		const absent = "ddns-test-absent0"
		fetcher, err := NewFetcher(DNSSettings{}, HTTPSettings{},
			PrivateIPSettings{Enabled: true, Interface: absent})
		require.NoError(t, err)

		ip, err := fetcher.IP4(t.Context())

		require.ErrorIs(t, err, privateip.ErrInterfaceNotFound)
		assert.False(t, ip.IsValid())
		assert.Contains(t, err.Error(), absent)
	})
}

// This pins the existing mixed-list behavior (subfetcher.go is unchanged): sub-fetchers
// are cycled on every call, so a failed private discovery is followed by a public address.
func Test_Fetcher_mixedListCycling(t *testing.T) {
	t.Parallel()

	public := netip.MustParseAddr("203.0.113.7")
	fetcher := &Fetcher{
		fetchers: []ipFetcher{fakeSubFetcher{ip: public}, fakeSubFetcher{err: errFakePrivate}},
		counter:  new(uint32),
	}

	for range 2 {
		ip, err := fetcher.IP4(t.Context())
		require.ErrorIs(t, err, errFakePrivate)
		assert.False(t, ip.IsValid())

		ip, err = fetcher.IP4(t.Context())
		require.NoError(t, err)
		assert.Equal(t, public, ip)
	}
}
