package aliyun

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"net/url"
	"time"
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
	values.Set("SignatureNonce", fmt.Sprint(randInt64))
	return values
}
