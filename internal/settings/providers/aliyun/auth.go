package aliyun

//nolint:gosec
import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"net/url"
	"sort"
	"strings"
)

func sign(method string, urlValues url.Values, accessKeySecret string) {
	sortedParams := make(sort.StringSlice, 0, len(urlValues))
	for key, values := range urlValues {
		s := key + "=" + values[0]
		sortedParams = append(sortedParams, s)
	}
	sortedParams.Sort()

	stringToSign := strings.ToUpper(method) + "&%2F&" +
		strings.Join(sortedParams, "&")

	key := []byte(accessKeySecret + "&")
	hmac := hmac.New(sha1.New, key)
	_, _ = hmac.Write([]byte(stringToSign))
	signedBytes := hmac.Sum(nil)
	signature := base64.StdEncoding.EncodeToString(signedBytes)
	urlValues.Set("Signature", signature)
}
