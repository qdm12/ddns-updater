package settings

import "net/http"

func setUserAgent(request *http.Request) {
	request.Header.Set("User-Agent", "DDNS-Updater quentin.mcgaw@gmail.com")
}

func setContentType(request *http.Request, contentType string) {
	request.Header.Set("Content-Type", contentType)
}

func setAccept(request *http.Request, acceptContent string) {
	request.Header.Set("Accept", acceptContent)
}

func setAuthBearer(request *http.Request, token string) {
	request.Header.Set("Authorization", "Bearer "+token)
}

func setAuthSSOKey(request *http.Request, key, secret string) {
	request.Header.Set("Authorization", "sso-key "+key+":"+secret)
}
