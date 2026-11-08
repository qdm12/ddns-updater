package server

import (
	"context"
	"net/http"
	"net/netip"
	"testing"
	"time"

	"github.com/qdm12/ddns-updater/internal/constants"
	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/records"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubProvider only implements what the index page needs: its HTML row.
type stubProvider struct{}

func (stubProvider) String() string                 { return "duckdns" }
func (stubProvider) Domain() string                 { return "example.duckdns.org" }
func (stubProvider) Owner() string                  { return "@" }
func (stubProvider) BuildDomainName() string        { return "example.duckdns.org" }
func (stubProvider) Proxied() bool                  { return false }
func (stubProvider) IPVersion() ipversion.IPVersion { return ipversion.IP4 }
func (stubProvider) IPv6Suffix() netip.Prefix       { return netip.Prefix{} }

func (stubProvider) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    `<a href="http://example.duckdns.org">example.duckdns.org</a>`,
		Owner:     "@",
		Provider:  `<a href="https://www.duckdns.org/">DuckDNS</a>`,
		IPVersion: "ipv4",
	}
}

func (stubProvider) Update(context.Context, *http.Client, netip.Addr) (
	netip.Addr, error,
) {
	return netip.Addr{}, nil
}

// Test_indexRendersRecordMarkup checks the index table renders the HTML built
// by the provider and record code instead of escaping it. This regressed when
// the templates were switched from text/template to html/template.
func Test_indexRendersRecordMarkup(t *testing.T) {
	t.Parallel()

	database := stubDatabase{records: []records.Record{
		{
			Provider: stubProvider{},
			Status:   constants.UPTODATE,
			Time:     time.Now(),
		},
	}}
	handler := newHandler(context.Background(), "", database,
		stubUpdateForcer{}, &stubReloader{}, newStubEntryManager())

	recorder := get(t, handler, "/")

	require.Equal(t, http.StatusOK, recorder.Code)
	body := recorder.Body.String()
	assert.Contains(t, body,
		`<a href="http://example.duckdns.org">example.duckdns.org</a>`)
	assert.Contains(t, body, `<a href="https://www.duckdns.org/">DuckDNS</a>`)
	assert.Contains(t, body, `<span class="uptodate">Up to date</span>`)
	assert.NotContains(t, body, "&lt;a href")
	assert.NotContains(t, body, "&lt;span")
}
