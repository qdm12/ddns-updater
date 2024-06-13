package aliyun

import (
	"crypto/rand"
	"encoding/binary"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/qdm12/ddns-updater/internal/provider/headers"
)

func newURLValues(accessKeyID string) (values url.Values) {
	randBytes := make([]byte, 8) //nolint:gomnd
	_, _ = rand.Read(randBytes)
	randInt64 := int64(binary.BigEndian.Uint64(randBytes))

	values = make(url.Values)
	values.Set("AccessKeyId", accessKeyID)
	values.Set("Format", "JSON")
	values.Set("Version", "2015-01-09")
	values.Set("SignatureMethod", "HMAC-SHA1")
	values.Set("Timestamp", time.Now().UTC().Format("2006-01-02T15:04:05Z"))
	values.Set("SignatureVersion", "1.0")
	values.Set("SignatureNonce", strconv.FormatInt(randInt64, 10))
	return values
}

func setHeaders(request *http.Request) {
	headers.SetUserAgent(request)
	headers.SetAccept(request, "application/json")
}
