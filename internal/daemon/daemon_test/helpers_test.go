// Copyright 2026 PolitePixels Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This project stands against fascism, authoritarianism, and all forms of
// oppression. We built this to empower people, not to enable those who would
// strip others of their rights and dignity.

package daemon_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/andybalholm/brotli"
	"github.com/klauspost/compress/gzip"
	"piko.sh/piko/internal/json"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/bootstrap"
	"piko.sh/piko/internal/daemon/daemon_adapters"
	"piko.sh/piko/internal/daemon/daemon_domain"
	"piko.sh/piko/internal/registry/registry_dto"
	"piko.sh/piko/internal/render/render_domain"
	"piko.sh/piko/internal/security/security_domain"
	"piko.sh/piko/internal/security/security_dto"
	"piko.sh/piko/internal/templater/templater_domain"
)

type noopValidator struct{}

func (*noopValidator) Struct(any) error { return nil }

type TestHarness struct {
	T                *testing.T
	Router           *chi.Mux
	RegistryService  *testRegistryService
	TemplaterService *templater_domain.MockTemplaterService
	CSRFService      *security_domain.MockCSRFTokenService
	ManifestStore    *testManifestStoreView
	VariantGenerator *daemon_domain.MockOnDemandVariantGenerator
	RegistryPort     *render_domain.MockRegistryPort
	RateLimitService *security_domain.MockRateLimitService
	ServerConfig     *bootstrap.ServerConfig
	SiteConfig       *config.WebsiteConfig
	Validator        daemon_domain.StructValidator
	CSPConfig        security_dto.CSPRuntimeConfig
}

func NewTestRouterBuilder(t *testing.T) daemon_domain.RouterBuilder {
	t.Helper()
	builder := daemon_adapters.NewHTTPRouterBuilder(nil)
	t.Cleanup(builder.Close)
	return builder
}

func NewTestHarness(t *testing.T) *TestHarness {
	t.Helper()

	return &TestHarness{
		T:                t,
		Router:           chi.NewRouter(),
		RegistryService:  newTestRegistryService(),
		TemplaterService: newMockTemplaterService(),
		CSRFService:      newMockCSRFService(),
		ManifestStore:    newTestManifestStoreView(),
		VariantGenerator: newMockOnDemandVariantGenerator(),
		RegistryPort:     &render_domain.MockRegistryPort{},
		RateLimitService: newMockRateLimitService(),
		ServerConfig:     defaultServerConfig(),
		SiteConfig:       defaultSiteConfig(),
		CSPConfig:        security_dto.CSPRuntimeConfig{},
		Validator:        &noopValidator{},
	}
}

func (h *TestHarness) RouterConfig() *daemon_domain.RouterConfig {
	return &daemon_domain.RouterConfig{
		Port:                  *h.ServerConfig.Network.Port,
		PublicDomain:          *h.ServerConfig.Network.PublicDomain,
		ForceHTTPS:            *h.ServerConfig.Network.ForceHTTPS,
		RequestTimeoutSeconds: *h.ServerConfig.Network.RequestTimeoutSeconds,
		MaxConcurrentRequests: *h.ServerConfig.Network.MaxConcurrentRequests,
		DistServePath:         *h.ServerConfig.Paths.DistServePath,
		ArtefactServePath:     *h.ServerConfig.Paths.ArtefactServePath,
		SecurityHeaders:       security_dto.SecurityHeadersValues{Enabled: true},
		RateLimit: security_dto.RateLimitValues{
			TrustedProxies:    h.ServerConfig.Security.RateLimit.TrustedProxies,
			CloudflareEnabled: h.ServerConfig.Security.RateLimit.CloudflareEnabled != nil && *h.ServerConfig.Security.RateLimit.CloudflareEnabled,
		},
		Reporting: security_dto.ReportingValues{},
		WatchMode: false,
	}
}

func (h *TestHarness) DoRequest(request *http.Request) *httptest.ResponseRecorder {
	h.T.Helper()
	recorder := httptest.NewRecorder()
	h.Router.ServeHTTP(recorder, request)
	return recorder
}

func (h *TestHarness) DoGet(path string) *httptest.ResponseRecorder {
	h.T.Helper()
	request := httptest.NewRequest(http.MethodGet, path, nil)
	return h.DoRequest(request)
}

func (h *TestHarness) DoPost(path string, body any) *httptest.ResponseRecorder {
	h.T.Helper()
	jsonBody, err := json.Marshal(body)
	require.NoError(h.T, err)
	request := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(jsonBody))
	request.Header.Set("Content-Type", "application/json")
	return h.DoRequest(request)
}

func (h *TestHarness) GetHTTPHandlerDependencies() *daemon_domain.HTTPHandlerDependencies {
	return &daemon_domain.HTTPHandlerDependencies{
		Templater: h.TemplaterService,
		Validator: h.Validator,
	}
}

func defaultServerConfig() *bootstrap.ServerConfig {
	return &bootstrap.ServerConfig{
		Paths: config.PathsConfig{
			BaseDir:           new("/test"),
			DistServePath:     new("/_piko/dist"),
			ArtefactServePath: new("/_piko/assets"),
		},
		Network: config.NetworkConfig{
			Port:                  new("8080"),
			PublicDomain:          new("localhost:8080"),
			ForceHTTPS:            new(false),
			RequestTimeoutSeconds: new(60),
			MaxConcurrentRequests: new(0),
		},
	}
}

func defaultSiteConfig() *config.WebsiteConfig {
	return &config.WebsiteConfig{
		I18n: config.I18nConfig{
			DefaultLocale: "en",
			Locales:       []string{"en"},
		},
	}
}

type RequestBuilder struct {
	method      string
	path        string
	headers     map[string]string
	queryParams map[string]string
	body        io.Reader
	contentType string
}

func NewRequest(method, path string) *RequestBuilder {
	return &RequestBuilder{
		method:      method,
		path:        path,
		headers:     make(map[string]string),
		queryParams: make(map[string]string),
	}
}

func Get(path string) *RequestBuilder {
	return NewRequest(http.MethodGet, path)
}

func Post(path string) *RequestBuilder {
	return NewRequest(http.MethodPost, path)
}

func (rb *RequestBuilder) WithHeader(key, value string) *RequestBuilder {
	rb.headers[key] = value
	return rb
}

func (rb *RequestBuilder) WithAcceptEncoding(encoding string) *RequestBuilder {
	return rb.WithHeader("Accept-Encoding", encoding)
}

func (rb *RequestBuilder) WithIfNoneMatch(etag string) *RequestBuilder {
	return rb.WithHeader("If-None-Match", etag)
}

func (rb *RequestBuilder) WithQueryParam(key, value string) *RequestBuilder {
	rb.queryParams[key] = value
	return rb
}

func (rb *RequestBuilder) WithJSONBody(body any) *RequestBuilder {
	jsonBytes, _ := json.Marshal(body)
	rb.body = bytes.NewReader(jsonBytes)
	rb.contentType = "application/json"
	return rb
}

func (rb *RequestBuilder) WithFormBody(values url.Values) *RequestBuilder {
	rb.body = strings.NewReader(values.Encode())
	rb.contentType = "application/x-www-form-urlencoded"
	return rb
}

func (rb *RequestBuilder) WithRawBody(body []byte, contentType string) *RequestBuilder {
	rb.body = bytes.NewReader(body)
	rb.contentType = contentType
	return rb
}

func (rb *RequestBuilder) Build() *http.Request {

	targetURL := rb.path
	if len(rb.queryParams) > 0 {
		params := url.Values{}
		for k, v := range rb.queryParams {
			params.Set(k, v)
		}
		targetURL = rb.path + "?" + params.Encode()
	}

	request := httptest.NewRequest(rb.method, targetURL, rb.body)

	for k, v := range rb.headers {
		request.Header.Set(k, v)
	}

	if rb.contentType != "" {
		request.Header.Set("Content-Type", rb.contentType)
	}

	return request
}

func AssertStatus(t *testing.T, recorder *httptest.ResponseRecorder, expected int) {
	t.Helper()
	assert.Equal(t, expected, recorder.Code, "unexpected status code")
}

func AssertHeader(t *testing.T, recorder *httptest.ResponseRecorder, key, expected string) {
	t.Helper()
	assert.Equal(t, expected, recorder.Header().Get(key), "unexpected header value for %s", key)
}

func AssertHeaderExists(t *testing.T, recorder *httptest.ResponseRecorder, key string) {
	t.Helper()
	assert.NotEmpty(t, recorder.Header().Get(key), "expected header %s to be present", key)
}

func AssertHeaderNotExists(t *testing.T, recorder *httptest.ResponseRecorder, key string) {
	t.Helper()
	assert.Empty(t, recorder.Header().Get(key), "expected header %s to not be present", key)
}

func AssertBodyContains(t *testing.T, recorder *httptest.ResponseRecorder, expected string) {
	t.Helper()
	body := recorder.Body.String()
	assert.Contains(t, body, expected, "body should contain %q", expected)
}

func AssertBodyNotContains(t *testing.T, recorder *httptest.ResponseRecorder, notExpected string) {
	t.Helper()
	body := recorder.Body.String()
	assert.NotContains(t, body, notExpected, "body should not contain %q", notExpected)
}

func AssertBodyEquals(t *testing.T, recorder *httptest.ResponseRecorder, expected string) {
	t.Helper()
	assert.Equal(t, expected, recorder.Body.String(), "body mismatch")
}

func AssertJSONEquals(t *testing.T, recorder *httptest.ResponseRecorder, expected any) {
	t.Helper()
	expectedJSON, err := json.Marshal(expected)
	require.NoError(t, err)

	var actualObj, expectedObj any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &actualObj))
	require.NoError(t, json.Unmarshal(expectedJSON, &expectedObj))

	assert.Equal(t, expectedObj, actualObj, "JSON body mismatch")
}

func AssertJSONField(t *testing.T, recorder *httptest.ResponseRecorder, fieldPath string, expected any) {
	t.Helper()
	var body map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))

	actual, ok := body[fieldPath]
	require.True(t, ok, "field %q not found in response", fieldPath)
	assert.Equal(t, expected, actual, "field %q mismatch", fieldPath)
}

func AssertCacheHit(t *testing.T, recorder *httptest.ResponseRecorder) {
	t.Helper()
	cacheHeader := recorder.Header().Get("X-Cache")
	assert.Equal(t, "HIT", cacheHeader, "expected cache hit")
}

func AssertCacheMiss(t *testing.T, recorder *httptest.ResponseRecorder) {
	t.Helper()
	cacheHeader := recorder.Header().Get("X-Cache")
	assert.Equal(t, "MISS", cacheHeader, "expected cache miss")
}

func AssertNotModified(t *testing.T, recorder *httptest.ResponseRecorder) {
	t.Helper()
	AssertStatus(t, recorder, http.StatusNotModified)
	assert.Empty(t, recorder.Body.Bytes(), "304 response should have empty body")
}

func CreateTestArtefact(id string, variants ...registry_dto.Variant) *registry_dto.ArtefactMeta {
	return &registry_dto.ArtefactMeta{
		ID:             id,
		ActualVariants: variants,
		Status:         registry_dto.VariantStatusReady,
	}
}

func CreateTestVariant(variantID, storageKey string, opts ...VariantOption) registry_dto.Variant {
	v := registry_dto.Variant{
		VariantID:  variantID,
		StorageKey: storageKey,
		MimeType:   "application/octet-stream",
		Status:     registry_dto.VariantStatusReady,
	}
	for _, opt := range opts {
		opt(&v)
	}
	return v
}

type VariantOption func(*registry_dto.Variant)

func WithVariantContentHash(hash string) VariantOption {
	return func(v *registry_dto.Variant) {
		v.ContentHash = hash
	}
}

func WithVariantMimeType(mimeType string) VariantOption {
	return func(v *registry_dto.Variant) {
		v.MimeType = mimeType
	}
}

func WithVariantTag(key, value string) VariantOption {
	return func(v *registry_dto.Variant) {
		v.MetadataTags.SetByName(key, value)
	}
}

func WithVariantStatus(status registry_dto.VariantStatus) VariantOption {
	return func(v *registry_dto.Variant) {
		v.Status = status
	}
}

func CompressBrotli(data []byte) []byte {
	var buffer bytes.Buffer
	w := brotli.NewWriter(&buffer)
	_, _ = w.Write(data)
	_ = w.Close()
	return buffer.Bytes()
}

func CompressGzip(data []byte) []byte {
	var buffer bytes.Buffer
	w := gzip.NewWriter(&buffer)
	_, _ = w.Write(data)
	_ = w.Close()
	return buffer.Bytes()
}

func DecompressBrotli(data []byte) ([]byte, error) {
	r := brotli.NewReader(bytes.NewReader(data))
	return io.ReadAll(r)
}

func DecompressGzip(data []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer func() { _ = r.Close() }()
	return io.ReadAll(r)
}

func DecompressResponse(t *testing.T, recorder *httptest.ResponseRecorder) []byte {
	t.Helper()
	encoding := recorder.Header().Get("Content-Encoding")
	body := recorder.Body.Bytes()

	switch encoding {
	case "br":
		result, err := DecompressBrotli(body)
		require.NoError(t, err, "failed to decompress brotli")
		return result
	case "gzip":
		result, err := DecompressGzip(body)
		require.NoError(t, err, "failed to decompress gzip")
		return result
	default:
		return body
	}
}

func (h *TestHarness) CreateTestServer() *httptest.Server {
	return httptest.NewServer(h.Router)
}

func CreateNonDecompressingClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DisableCompression: true,
		},
	}
}
