//nolint:gosec
package ovh

import (
	"crypto/sha1"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

func (p *Provider) setHeaderCommon(header http.Header) {
	header.Add("Accept", "application/json;charset=utf-8")
	header.Add("X-Ovh-Application", p.appKey)
}

func (p *Provider) setHeaderAuth(header http.Header, timestamp int64,
	httpMethod string, url *url.URL, body []byte) {
	header.Add("X-Ovh-Timestamp", strconv.Itoa(int(timestamp)))
	header.Add("X-Ovh-Consumer", p.consumerKey)

	sha1Sum := sha1.Sum([]byte(
		p.appSecret + "+" + p.consumerKey + "+" + httpMethod + "+" +
			url.String() + "+" + string(body) + "+" + strconv.Itoa(int(timestamp)),
	))

	header.Add("X-Ovh-Signature", fmt.Sprintf("$1$%x", sha1Sum))
}
