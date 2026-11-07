package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"

	"github.com/qdm12/ddns-updater/internal/configfile"
	"github.com/qdm12/ddns-updater/internal/records"
	"github.com/qdm12/ddns-updater/pkg/publicip"
)

type dbFake struct{}

func (dbFake) SelectAll() []records.Record { return nil }

type runnerFake struct{}

func (runnerFake) ForceUpdate(context.Context) []error { return nil }

type loggerFake struct{ infos []string }

func (l *loggerFake) Info(s string) { l.infos = append(l.infos, s) }
func (l *loggerFake) Warn(string)   {}
func (l *loggerFake) Error(string)  {}

type checkerFake struct{ results []publicip.CheckResult }

func (c checkerFake) Check(context.Context) []publicip.CheckResult { return c.results }

type configStoreFake struct {
	content string
	fromEnv bool
	saved   string
}

func (c *configStoreFake) Path() string          { return "/data/config.json" }
func (c *configStoreFake) FromEnvironment() bool { return c.fromEnv }
func (c *configStoreFake) Read() ([]byte, error) { return []byte(c.content), nil }
func (c *configStoreFake) Validate(content []byte) ([]string, error) {
	if !json.Valid(content) {
		return nil, errors.Join(configfile.ErrInvalid, errors.New("bad json"))
	}
	return []string{"careful"}, nil
}

func (c *configStoreFake) Save(content []byte) ([]string, error) {
	if c.fromEnv {
		return nil, configfile.ErrFromEnvironment
	}
	warnings, err := c.Validate(content)
	if err != nil {
		return nil, err
	}
	c.saved = string(content)
	return warnings, nil
}

func newTestHandler(rootURL string, features Features) (http.Handler, *loggerFake) {
	logger := &loggerFake{}
	handler := newHandler(context.Background(), rootURL, dbFake{}, runnerFake{}, logger, features)
	return handler, logger
}

func do(handler http.Handler, method, target, body string, headers map[string]string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

var jsonHeader = map[string]string{"Content-Type": "application/json"}

func fullFeatures(store *configStoreFake) Features {
	return Features{
		PublicIPChecker: checkerFake{results: []publicip.CheckResult{
			{Method: publicip.MethodDNS,
				IPv4: publicip.FetchResult{IP: mustAddr("203.0.113.7")},
				IPv6: publicip.FetchResult{Err: errors.New("no ipv6 here")}},
			{Method: publicip.MethodHTTP,
				IPv4: publicip.FetchResult{IP: mustAddr("203.0.113.7")},
				IPv6: publicip.FetchResult{IP: mustAddr("2001:db8::1")}},
		}},
		PublicIPSettings: "Public IP fetching <settings>",
		ConfigStore:      store,
	}
}

func Test_index_toolbar(t *testing.T) {
	t.Parallel()

	handler, _ := newTestHandler("/", fullFeatures(&configStoreFake{}))
	body := do(handler, http.MethodGet, "/", "", nil).Body.String()
	for _, want := range []string{`href="publicip"`, `href="config"`} {
		if !strings.Contains(body, want) {
			t.Errorf("index page is missing %s", want)
		}
	}

	handler, _ = newTestHandler("/", Features{})
	body = do(handler, http.MethodGet, "/", "", nil).Body.String()
	for _, unwanted := range []string{`href="publicip"`, `href="config"`, `class="toolbar"`} {
		if strings.Contains(body, unwanted) {
			t.Errorf("index page should not contain %s when the features are off", unwanted)
		}
	}
}

func Test_publicIP(t *testing.T) {
	t.Parallel()

	handler, _ := newTestHandler("/", fullFeatures(&configStoreFake{}))

	page := do(handler, http.MethodGet, "/publicip", "", nil)
	if page.Code != http.StatusOK {
		t.Fatalf("page status %d", page.Code)
	}
	if !strings.Contains(page.Body.String(), "Public IP fetching &lt;settings&gt;") {
		t.Errorf("settings must be shown HTML escaped")
	}

	response := do(handler, http.MethodPost, "/publicip/check", "", nil)
	if response.Code != http.StatusOK {
		t.Fatalf("check status %d: %s", response.Code, response.Body.String())
	}
	var body publicIPResponseJSON
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(body.Results))
	}
	if body.Results[0].Method != "DNS" || body.Results[0].IPv4.IP != "203.0.113.7" ||
		body.Results[0].IPv6.Error != "no ipv6 here" || body.Results[0].IPv6.IP != "" {
		t.Errorf("unexpected first result: %+v", body.Results[0])
	}
	if body.Results[1].IPv6.IP != "2001:db8::1" {
		t.Errorf("unexpected second result: %+v", body.Results[1])
	}

	cross := do(handler, http.MethodPost, "/publicip/check", "",
		map[string]string{"Sec-Fetch-Site": "cross-site"})
	if cross.Code != http.StatusForbidden {
		t.Errorf("cross-site check status %d instead of 403", cross.Code)
	}
}

func Test_publicIP_disabled(t *testing.T) {
	t.Parallel()
	handler, _ := newTestHandler("/", Features{})
	if code := do(handler, http.MethodGet, "/publicip", "", nil).Code; code != http.StatusNotFound {
		t.Errorf("status %d instead of 404", code)
	}
}

func Test_config_page_escapesContent(t *testing.T) {
	t.Parallel()
	store := &configStoreFake{content: `{"x":"</textarea><script>alert(1)</script>"}`}
	handler, _ := newTestHandler("/", fullFeatures(store))

	response := do(handler, http.MethodGet, "/config", "", nil)
	if response.Code != http.StatusOK {
		t.Fatalf("status %d", response.Code)
	}
	body := response.Body.String()
	if strings.Contains(body, "<script>alert(1)") {
		t.Errorf("config content was not escaped")
	}
	if !strings.Contains(body, "&lt;/textarea&gt;") {
		t.Errorf("escaped content not found in page")
	}
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Errorf("config page must not be cacheable")
	}
}

func Test_config_save(t *testing.T) {
	t.Parallel()

	t.Run("valid", func(t *testing.T) {
		t.Parallel()
		store := &configStoreFake{}
		handler, logger := newTestHandler("/", fullFeatures(store))
		request := `{"content":"{\"settings\":[]}"}`
		response := do(handler, http.MethodPut, "/config", request, jsonHeader)
		if response.Code != http.StatusOK {
			t.Fatalf("status %d: %s", response.Code, response.Body.String())
		}
		if store.saved != `{"settings":[]}` {
			t.Errorf("unexpected saved content %q", store.saved)
		}
		var body configResponseJSON
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if !body.OK || !body.RestartRequired || len(body.Warnings) != 1 {
			t.Errorf("unexpected response %+v", body)
		}
		if len(logger.infos) != 1 || strings.Contains(logger.infos[0], "settings") {
			t.Errorf("expected one log line without the content, got %v", logger.infos)
		}
	})

	t.Run("invalid", func(t *testing.T) {
		t.Parallel()
		store := &configStoreFake{}
		handler, _ := newTestHandler("/", fullFeatures(store))
		response := do(handler, http.MethodPut, "/config", `{"content":"nope"}`, jsonHeader)
		if response.Code != http.StatusUnprocessableEntity {
			t.Errorf("status %d instead of 422", response.Code)
		}
		if store.saved != "" {
			t.Errorf("invalid content must not be saved")
		}
	})

	t.Run("from environment", func(t *testing.T) {
		t.Parallel()
		store := &configStoreFake{fromEnv: true}
		handler, _ := newTestHandler("/", fullFeatures(store))
		response := do(handler, http.MethodPut, "/config", `{"content":"{}"}`, jsonHeader)
		if response.Code != http.StatusConflict {
			t.Errorf("status %d instead of 409", response.Code)
		}
	})

	t.Run("not json content type", func(t *testing.T) {
		t.Parallel()
		store := &configStoreFake{}
		handler, _ := newTestHandler("/", fullFeatures(store))
		form := map[string]string{"Content-Type": "application/x-www-form-urlencoded"}
		response := do(handler, http.MethodPut, "/config", "content=%7B%7D", form)
		if response.Code != http.StatusUnsupportedMediaType {
			t.Errorf("status %d instead of 415", response.Code)
		}
		if store.saved != "" {
			t.Errorf("nothing must be saved")
		}
	})

	t.Run("cross site", func(t *testing.T) {
		t.Parallel()
		store := &configStoreFake{}
		handler, _ := newTestHandler("/", fullFeatures(store))
		headers := map[string]string{"Content-Type": "application/json", "Sec-Fetch-Site": "cross-site"}
		response := do(handler, http.MethodPut, "/config", `{"content":"{}"}`, headers)
		if response.Code != http.StatusForbidden {
			t.Errorf("status %d instead of 403", response.Code)
		}
		if store.saved != "" {
			t.Errorf("nothing must be saved")
		}
	})

	t.Run("malformed body", func(t *testing.T) {
		t.Parallel()
		handler, _ := newTestHandler("/", fullFeatures(&configStoreFake{}))
		response := do(handler, http.MethodPut, "/config", `not json`, jsonHeader)
		if response.Code != http.StatusBadRequest {
			t.Errorf("status %d instead of 400", response.Code)
		}
	})
}

func Test_config_validate_doesNotSave(t *testing.T) {
	t.Parallel()
	store := &configStoreFake{}
	handler, _ := newTestHandler("/", fullFeatures(store))
	response := do(handler, http.MethodPost, "/config/validate", `{"content":"{}"}`, jsonHeader)
	if response.Code != http.StatusOK {
		t.Fatalf("status %d: %s", response.Code, response.Body.String())
	}
	if store.saved != "" {
		t.Errorf("validation must not save")
	}
}

func Test_config_disabled(t *testing.T) {
	t.Parallel()
	features := fullFeatures(nil)
	features.ConfigStore = nil
	handler, _ := newTestHandler("/", features)
	for _, target := range []string{"/config"} {
		if code := do(handler, http.MethodGet, target, "", nil).Code; code != http.StatusNotFound {
			t.Errorf("%s: status %d instead of 404", target, code)
		}
	}
}

func Test_basicAuth(t *testing.T) {
	t.Parallel()
	features := fullFeatures(&configStoreFake{})
	features.BasicAuthUsername = "thomas"
	features.BasicAuthPassword = "s3cret"
	handler, _ := newTestHandler("/", features)

	targets := []struct{ method, target string }{
		{http.MethodGet, "/"},
		{http.MethodGet, "/config"},
		{http.MethodGet, "/publicip"},
		{http.MethodPost, "/publicip/check"},
		{http.MethodGet, "/update"},
	}
	for _, tc := range targets {
		response := do(handler, tc.method, tc.target, "", nil)
		if response.Code != http.StatusUnauthorized {
			t.Errorf("%s %s without credentials: status %d instead of 401", tc.method, tc.target, response.Code)
		}
		if response.Header().Get("WWW-Authenticate") == "" {
			t.Errorf("%s %s: missing WWW-Authenticate header", tc.method, tc.target)
		}
	}

	request := httptest.NewRequest(http.MethodGet, "/config", nil)
	request.SetBasicAuth("thomas", "wrong")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Errorf("wrong password: status %d instead of 401", recorder.Code)
	}

	request = httptest.NewRequest(http.MethodGet, "/config", nil)
	request.SetBasicAuth("thomas", "s3cret")
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Errorf("good credentials: status %d instead of 200", recorder.Code)
	}
}

func Test_rootURL(t *testing.T) {
	t.Parallel()
	handler, _ := newTestHandler("/ddns/", fullFeatures(&configStoreFake{}))
	for _, target := range []string{"/ddns/", "/ddns/publicip", "/ddns/config"} {
		if code := do(handler, http.MethodGet, target, "", nil).Code; code != http.StatusOK {
			t.Errorf("%s: status %d instead of 200", target, code)
		}
	}
	if code := do(handler, http.MethodPost, "/ddns/publicip/check", "", nil).Code; code != http.StatusOK {
		t.Errorf("check under root URL: status %d instead of 200", code)
	}
}

func mustAddr(s string) netip.Addr {
	addr, err := netip.ParseAddr(s)
	if err != nil {
		panic(err)
	}
	return addr
}
