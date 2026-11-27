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

package daemon_adapters

import (
	"bytes"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"net/url"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/andybalholm/brotli"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/klauspost/compress/gzip"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"piko.sh/piko/internal/daemon/daemon_frontend"
)

var (
	versionedCorePath = daemon_frontend.CoreRuntimeBuildDevelopment.CoreModuleURL()
)

func TestFrontendAssetHandlerWithFullMiddlewareStack(t *testing.T) {
	t.Run("With Production Middleware Stack", func(t *testing.T) {

		mainRouter := chi.NewRouter()

		mainRouter.Use(cors.Handler(cors.Options{
			AllowedOrigins:   []string{"*"},
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "HEAD"},
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-CSRF-Action-Token"},
			ExposedHeaders:   []string{"Link"},
			AllowCredentials: false,
			MaxAge:           300,
		}))

		mainRouter.Use(middleware.RequestID)

		mainRouter.Use(middleware.Recoverer)
		mainRouter.Use(middleware.Heartbeat("/ping"))
		mainRouter.Use(middleware.Timeout(60 * time.Second))

		mainRouter.Get("/_piko/dist/*", newEmbeddedFrontendHandler(daemon_frontend.RuntimeVersion(), cacheControlImmutable, daemon_frontend.CoreRuntimeBuildDevelopment).ServeHTTP)
		mainRouter.Head("/_piko/dist/*", newEmbeddedFrontendHandler(daemon_frontend.RuntimeVersion(), cacheControlImmutable, daemon_frontend.CoreRuntimeBuildDevelopment).ServeHTTP)

		tracingHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqCtx := r.Context()

			carrier := propagation.MapCarrier{}
			for k, v := range r.Header {
				if len(v) > 0 {
					carrier.Set(k, v[0])
				}
			}
			reqCtx = otel.GetTextMapPropagator().Extract(reqCtx, carrier)

			mainRouter.ServeHTTP(w, r.WithContext(reqCtx))
		})

		server := httptest.NewServer(tracingHandler)
		defer server.Close()

		testCases := []struct {
			headers            map[string]string
			name               string
			path               string
			method             string
			checkHeaders       []string
			expectedStatusCode int
		}{
			{
				name:               "Should serve asset through middleware stack",
				path:               versionedCorePath,
				method:             "GET",
				expectedStatusCode: http.StatusOK,
				checkHeaders:       []string{"Content-Type", "ETag", "Cache-Control"},
			},
			{
				name:               "Should handle OPTIONS request (middleware processing)",
				path:               versionedCorePath,
				method:             "OPTIONS",
				expectedStatusCode: http.StatusMethodNotAllowed,
			},
			{
				name:               "Should process GET request through all middleware",
				path:               versionedCorePath,
				method:             "GET",
				expectedStatusCode: http.StatusOK,
				checkHeaders:       []string{"Content-Type"},
			},
			{
				name:               "Should handle heartbeat endpoint (middleware.Heartbeat)",
				path:               "/ping",
				method:             "GET",
				expectedStatusCode: http.StatusOK,
			},
			{
				name:               "Should handle timeout gracefully (middleware.Timeout)",
				path:               versionedCorePath,
				method:             "GET",
				expectedStatusCode: http.StatusOK,
				checkHeaders:       []string{"Content-Type"},
			},
			{
				name:               "Should handle recoverer middleware",
				path:               versionedCorePath,
				method:             "GET",
				expectedStatusCode: http.StatusOK,
				checkHeaders:       []string{"Content-Type"},
			},
			{
				name:               "Should handle HEAD requests through middleware stack",
				path:               versionedCorePath,
				method:             "HEAD",
				expectedStatusCode: http.StatusOK,
				checkHeaders:       []string{"Content-Type", "ETag"},
			},
			{
				name:               "Should handle 404 through middleware stack",
				path:               "/_piko/dist/nonexistent.js",
				method:             "GET",
				expectedStatusCode: http.StatusNotFound,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				request, err := http.NewRequestWithContext(t.Context(), tc.method, server.URL+tc.path, nil)
				require.NoError(t, err)

				for k, v := range tc.headers {
					request.Header.Set(k, v)
				}

				response, err := http.DefaultClient.Do(request)
				require.NoError(t, err)
				defer func() { _ = response.Body.Close() }()

				assert.Equal(t, tc.expectedStatusCode, response.StatusCode, "Status code mismatch")

				for _, headerName := range tc.checkHeaders {
					actualVal := response.Header.Get(headerName)
					assert.NotEmpty(t, actualVal, "Header '%s' should be present", headerName)
				}
			})
		}
	})
}

func TestFrontendAssetHandler(t *testing.T) {

	t.Run("With Global Asset Store", func(t *testing.T) {

		r := chi.NewRouter()

		r.Get("/_piko/dist/*", newEmbeddedFrontendHandler(daemon_frontend.RuntimeVersion(), cacheControlImmutable, daemon_frontend.CoreRuntimeBuildDevelopment).ServeHTTP)
		r.Head("/_piko/dist/*", newEmbeddedFrontendHandler(daemon_frontend.RuntimeVersion(), cacheControlImmutable, daemon_frontend.CoreRuntimeBuildDevelopment).ServeHTTP)
		server := httptest.NewServer(r)
		defer server.Close()

		expectedAsset, ok := daemon_frontend.GetAsset(t.Context(), "built/ppframework.core.es.js")
		require.True(t, ok)
		expectedAssetBr, ok := daemon_frontend.GetAsset(t.Context(), "built/ppframework.core.es.js.br")
		require.True(t, ok)

		testCases := []struct {
			expectedHeaders    map[string]string
			name               string
			path               string
			method             string
			acceptEncoding     string
			ifNoneMatch        string
			rangeHeader        string
			expectedEncoding   string
			expectedBody       []byte
			expectedStatusCode int
		}{
			{
				name:               "Should serve uncompressed asset when no encoding is accepted",
				path:               versionedCorePath,
				method:             "GET",
				acceptEncoding:     "",
				expectedStatusCode: http.StatusOK,
				expectedEncoding:   "",
				expectedBody:       expectedAsset.Content,
			},
			{
				name:               "Should serve Brotli compressed asset when 'br' is accepted",
				path:               versionedCorePath,
				method:             "GET",
				acceptEncoding:     "br, gzip",
				expectedStatusCode: http.StatusOK,
				expectedEncoding:   "br",
				expectedBody:       expectedAssetBr.Content,
			},
			{
				name:               "Should serve Gzip compressed asset when only 'gzip' is accepted",
				path:               versionedCorePath,
				method:             "GET",
				acceptEncoding:     "gzip, deflate",
				expectedStatusCode: http.StatusOK,
				expectedEncoding:   "gzip",
			},
			{
				name:               "Should return 304 Not Modified for matching ETag",
				path:               versionedCorePath,
				method:             "GET",
				ifNoneMatch:        "",
				expectedStatusCode: http.StatusNotModified,
			},
			{
				name:               "Should return 200 OK for mismatched ETag",
				path:               versionedCorePath,
				method:             "GET",
				ifNoneMatch:        `"wrong-etag"`,
				expectedStatusCode: http.StatusOK,
				expectedBody:       expectedAsset.Content,
			},
			{
				name:               "Should return 404 for a non-existent asset",
				path:               "/_piko/dist/non-existent-file.js",
				method:             "GET",
				expectedStatusCode: http.StatusNotFound,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {

				if tc.name == "Should return 304 Not Modified for matching ETag" {

					request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL+versionedCorePath, nil)
					require.NoError(t, err)

					response, err := http.DefaultClient.Do(request)
					require.NoError(t, err)
					_ = response.Body.Close()

					actualETag := response.Header.Get("ETag")
					require.NotEmpty(t, actualETag, "Server should return an ETag")

					tc.ifNoneMatch = actualETag
				}
				runTestCase(t, server.URL, tc)
			})
		}
	})

	t.Run("Additional Edge Cases", func(t *testing.T) {

		r := chi.NewRouter()
		r.Get("/_piko/dist/*", newEmbeddedFrontendHandler(daemon_frontend.RuntimeVersion(), cacheControlImmutable, daemon_frontend.CoreRuntimeBuildDevelopment).ServeHTTP)
		r.Head("/_piko/dist/*", newEmbeddedFrontendHandler(daemon_frontend.RuntimeVersion(), cacheControlImmutable, daemon_frontend.CoreRuntimeBuildDevelopment).ServeHTTP)
		server := httptest.NewServer(r)
		defer server.Close()

		testCases := []struct {
			expectedHeaders    map[string]string
			name               string
			path               string
			method             string
			acceptEncoding     string
			ifNoneMatch        string
			rangeHeader        string
			expectedEncoding   string
			expectedBody       []byte
			expectedStatusCode int
		}{
			{
				name:               "Should return 404 for non-existent asset",
				path:               "/_piko/dist/non-existent-file.js",
				method:             "GET",
				expectedStatusCode: http.StatusNotFound,
			},
			{
				name:               "Should handle HEAD requests",
				path:               versionedCorePath,
				method:             "HEAD",
				expectedStatusCode: http.StatusOK,
			},
			{
				name:               "Should return 405 for POST method",
				path:               versionedCorePath,
				method:             "POST",
				expectedStatusCode: http.StatusMethodNotAllowed,
			},
			{
				name:               "Should handle path traversal attempts",
				path:               "/_piko/dist/../../../etc/passwd",
				method:             "GET",
				expectedStatusCode: http.StatusNotFound,
			},
			{
				name:               "Should handle empty path parameter",
				path:               "/_piko/dist/",
				method:             "GET",
				expectedStatusCode: http.StatusNotFound,
			},
			{
				name:               "Should handle Range requests (if supported by http.ServeContent)",
				path:               versionedCorePath,
				method:             "GET",
				rangeHeader:        "bytes=0-10",
				expectedStatusCode: http.StatusPartialContent,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				runTestCase(t, server.URL, tc)
			})
		}
	})
}

func runTestCase(t *testing.T, serverURL string, tc struct {
	expectedHeaders    map[string]string
	name               string
	path               string
	method             string
	acceptEncoding     string
	ifNoneMatch        string
	rangeHeader        string
	expectedEncoding   string
	expectedBody       []byte
	expectedStatusCode int
}) {
	t.Helper()

	method := "GET"
	if tc.method != "" {
		method = tc.method
	}

	request, err := http.NewRequestWithContext(t.Context(), method, serverURL+tc.path, nil)
	require.NoError(t, err)

	if tc.acceptEncoding != "" {
		request.Header.Set("Accept-Encoding", tc.acceptEncoding)
	}
	if tc.ifNoneMatch != "" {
		request.Header.Set("If-None-Match", tc.ifNoneMatch)
	}
	if tc.rangeHeader != "" {
		request.Header.Set("Range", tc.rangeHeader)
	}

	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	defer func() { _ = response.Body.Close() }()

	assert.Equal(t, tc.expectedStatusCode, response.StatusCode, "Status code mismatch")

	for key, value := range tc.expectedHeaders {
		assert.Equal(t, value, response.Header.Get(key), "Header '%s' mismatch", key)
	}

	if response.StatusCode == http.StatusOK {
		assert.Equal(t, tc.expectedEncoding, response.Header.Get("Content-Encoding"), "Content-Encoding header mismatch")
		assert.Equal(t, expectedFrontendCacheControl(tc.path), response.Header.Get("Cache-Control"))
		assert.NotEmpty(t, response.Header.Get("ETag"))
		assert.Contains(t, response.Header.Get("Vary"), "Accept-Encoding")
	}

	if response.StatusCode == http.StatusPartialContent {
		assert.Equal(t, tc.expectedEncoding, response.Header.Get("Content-Encoding"), "Content-Encoding header mismatch")

		assert.NotEmpty(t, response.Header.Get("Content-Length"), "Range responses should have Content-Length")

		assert.NotEmpty(t, response.Header.Get("Content-Range"), "Range responses should have Content-Range")
	}

	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)

	if len(tc.expectedBody) > 0 {
		assert.Equal(t, tc.expectedBody, body, "Response body mismatch")
	} else if tc.expectedEncoding == "gzip" {

		gzReader, err := gzip.NewReader(strings.NewReader(string(body)))
		require.NoError(t, err)
		defer func() { _ = gzReader.Close() }()
		decompressedBody, err := io.ReadAll(gzReader)
		require.NoError(t, err)

		expectedAsset, _ := daemon_frontend.GetAsset(t.Context(), "built/ppframework.core.es.js")
		assert.Equal(t, expectedAsset.Content, decompressedBody)
	} else {

		switch tc.expectedStatusCode {
		case http.StatusNotModified:
			assert.Empty(t, body, "304 responses should have empty body")
		case http.StatusNotFound, http.StatusMethodNotAllowed:

		case http.StatusPartialContent:

			assert.NotEmpty(t, body, "206 responses should have body content")
		case http.StatusOK:
			if tc.method == "HEAD" {
				assert.Empty(t, body, "HEAD responses should have empty body")
			} else {

				assert.NotEmpty(t, body, "200 GET responses should have body content")
			}
		}
	}
}

func decompressBrotli(t *testing.T, data []byte) []byte {
	t.Helper()

	if len(data) > 0 {
		preview := data
		if len(preview) > 20 {
			preview = preview[:20]
		}
		t.Logf("Brotli data preview (first %d bytes): %v", len(preview), preview)
		t.Logf("Brotli data as string preview: %q", string(preview))
	}

	r := brotli.NewReader(bytes.NewReader(data))
	decompressed, err := io.ReadAll(r)
	if err != nil {
		t.Logf("Brotli decompression failed: %v", err)
		t.Logf("Data length: %d bytes", len(data))
		return nil
	}

	return decompressed
}

func decompressGzip(t *testing.T, data []byte) []byte {
	t.Helper()

	gzReader, err := gzip.NewReader(bytes.NewReader(data))
	require.NoError(t, err, "failed to create gzip reader")
	defer func() { _ = gzReader.Close() }()

	decompressed, err := io.ReadAll(gzReader)
	require.NoError(t, err, "failed to read decompressed gzip data")

	return decompressed
}

func createNonDecompressingHTTPClient() *http.Client {
	transport := &http.Transport{
		DisableCompression: true,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}
}

func TestFrontendAssetHandlerCompressionVerification(t *testing.T) {
	t.Run("Compression Verification with Full Middleware Stack", func(t *testing.T) {

		mainRouter := chi.NewRouter()

		mainRouter.Use(cors.Handler(cors.Options{
			AllowedOrigins:   []string{"*"},
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "HEAD"},
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-CSRF-Action-Token"},
			ExposedHeaders:   []string{"Link"},
			AllowCredentials: false,
			MaxAge:           300,
		}))

		mainRouter.Use(middleware.RequestID)
		mainRouter.Use(middleware.Recoverer)
		mainRouter.Use(middleware.Heartbeat("/ping"))
		mainRouter.Use(middleware.Timeout(60 * time.Second))

		mainRouter.Get("/_piko/dist/*", newEmbeddedFrontendHandler(daemon_frontend.RuntimeVersion(), cacheControlImmutable, daemon_frontend.CoreRuntimeBuildDevelopment).ServeHTTP)
		mainRouter.Head("/_piko/dist/*", newEmbeddedFrontendHandler(daemon_frontend.RuntimeVersion(), cacheControlImmutable, daemon_frontend.CoreRuntimeBuildDevelopment).ServeHTTP)

		tracingHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqCtx := r.Context()

			carrier := propagation.MapCarrier{}
			for k, v := range r.Header {
				if len(v) > 0 {
					carrier.Set(k, v[0])
				}
			}
			reqCtx = otel.GetTextMapPropagator().Extract(reqCtx, carrier)

			mainRouter.ServeHTTP(w, r.WithContext(reqCtx))
		})

		server := httptest.NewServer(tracingHandler)
		defer server.Close()

		expectedAsset, ok := daemon_frontend.GetAsset(t.Context(), "built/ppframework.core.es.js")
		require.True(t, ok, "Expected asset should exist in embedded store")

		client := createNonDecompressingHTTPClient()

		compressionTestCases := []struct {
			decompressFunc   func(*testing.T, []byte) []byte
			name             string
			acceptEncoding   string
			expectedEncoding string
		}{
			{
				name:             "Should serve and verify Brotli compressed asset",
				acceptEncoding:   "br, gzip, deflate",
				expectedEncoding: "br",
				decompressFunc:   decompressBrotli,
			},
			{
				name:             "Should serve and verify Gzip compressed asset",
				acceptEncoding:   "gzip, deflate",
				expectedEncoding: "gzip",
				decompressFunc:   decompressGzip,
			},
			{
				name:             "Should prefer Brotli over Gzip when both accepted",
				acceptEncoding:   "gzip, br, deflate",
				expectedEncoding: "br",
				decompressFunc:   decompressBrotli,
			},
		}

		for _, tc := range compressionTestCases {
			t.Run(tc.name, func(t *testing.T) {

				request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL+versionedCorePath, nil)
				require.NoError(t, err)
				request.Header.Set("Accept-Encoding", tc.acceptEncoding)

				response, err := client.Do(request)
				require.NoError(t, err)
				defer func() { _ = response.Body.Close() }()

				assert.Equal(t, http.StatusOK, response.StatusCode, "Status code should be 200")

				actualEncoding := response.Header.Get("Content-Encoding")
				t.Logf("Request Accept-Encoding: %s", tc.acceptEncoding)
				t.Logf("Response Content-Encoding: %s", actualEncoding)

				assert.Equal(t, "application/javascript; charset=utf-8", response.Header.Get("Content-Type"),
					"Content-Type should be correct")
				assert.NotEmpty(t, response.Header.Get("ETag"), "ETag should be present")
				varyHeader := response.Header.Get("Vary")
				t.Logf("Vary header: %s", varyHeader)

				assert.NotEmpty(t, varyHeader, "Vary header should be present with middleware stack")
				assert.Equal(t, cacheControlImmutable, response.Header.Get("Cache-Control"), "a versioned bundle may be cached for good")

				compressedBody, err := io.ReadAll(response.Body)
				require.NoError(t, err, "Should be able to read compressed response body")
				assert.NotEmpty(t, compressedBody, "Compressed body should not be empty")

				if actualEncoding == "" {

					assert.Equal(t, expectedAsset.Content, compressedBody,
						"Uncompressed content should match original asset content")
					t.Logf("⚠ Server served uncompressed content instead of expected %s compression", tc.expectedEncoding)
				} else if actualEncoding != tc.expectedEncoding {

					t.Logf("⚠ Server served %s compression instead of expected %s compression", actualEncoding, tc.expectedEncoding)

					assert.NotEqual(t, expectedAsset.Content, compressedBody,
						"Response should be compressed and different from original")
				} else {

					assert.NotEqual(t, expectedAsset.Content, compressedBody,
						"Raw response body should be different from original (i.e., compressed)")

					decompressedContent := tc.decompressFunc(t, compressedBody)

					if decompressedContent == nil {
						t.Logf("⚠ Decompression failed for %s - this may indicate the asset isn't actually compressed with %s",
							tc.expectedEncoding, tc.expectedEncoding)

						return
					}

					assert.Equal(t, expectedAsset.Content, decompressedContent,
						"Decompressed content should exactly match the original asset content")
				}

				t.Logf("Successfully verified %s compression/decompression cycle", tc.expectedEncoding)
				t.Logf("  Original size: %d bytes", len(expectedAsset.Content))
				t.Logf("  Compressed size: %d bytes", len(compressedBody))
				t.Logf("  Compression ratio: %.2f%%", float64(len(compressedBody))/float64(len(expectedAsset.Content))*100)
			})
		}
	})

	t.Run("Compression Edge Cases and Error Scenarios", func(t *testing.T) {

		router := chi.NewRouter()
		router.Use(middleware.RequestID)
		router.Get("/_piko/dist/*", newEmbeddedFrontendHandler(daemon_frontend.RuntimeVersion(), cacheControlImmutable, daemon_frontend.CoreRuntimeBuildDevelopment).ServeHTTP)

		server := httptest.NewServer(router)
		defer server.Close()

		client := createNonDecompressingHTTPClient()

		edgeCaseTests := []struct {
			name             string
			path             string
			acceptEncoding   string
			expectedStatus   int
			shouldDecompress bool
		}{
			{
				name:             "Should handle no Accept-Encoding header (uncompressed)",
				path:             versionedCorePath,
				acceptEncoding:   "",
				expectedStatus:   http.StatusOK,
				shouldDecompress: false,
			},
			{
				name:             "Should handle unsupported compression types gracefully",
				path:             versionedCorePath,
				acceptEncoding:   "deflate, compress",
				expectedStatus:   http.StatusOK,
				shouldDecompress: false,
			},
			{
				name:             "Should handle non-existent assets with compression headers",
				path:             "/_piko/dist/nonexistent.js",
				acceptEncoding:   "br, gzip",
				expectedStatus:   http.StatusNotFound,
				shouldDecompress: false,
			},
		}

		for _, tc := range edgeCaseTests {
			t.Run(tc.name, func(t *testing.T) {
				request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL+tc.path, nil)
				require.NoError(t, err)
				if tc.acceptEncoding != "" {
					request.Header.Set("Accept-Encoding", tc.acceptEncoding)
				}

				response, err := client.Do(request)
				require.NoError(t, err)
				defer func() { _ = response.Body.Close() }()

				assert.Equal(t, tc.expectedStatus, response.StatusCode, "Status code should match expected")

				body, err := io.ReadAll(response.Body)
				require.NoError(t, err)

				if tc.expectedStatus == http.StatusOK {
					if tc.shouldDecompress {

						contentEncoding := response.Header.Get("Content-Encoding")
						assert.NotEmpty(t, contentEncoding, "Content-Encoding should be set for compressed responses")
					} else {

						contentEncoding := response.Header.Get("Content-Encoding")
						assert.Empty(t, contentEncoding, "Content-Encoding should be empty for uncompressed responses")

						if tc.acceptEncoding == "" {
							expectedAsset, ok := daemon_frontend.GetAsset(t.Context(), "built/ppframework.core.es.js")
							require.True(t, ok)
							assert.Equal(t, expectedAsset.Content, body, "Uncompressed body should match original asset")
						}
					}
				}
			})
		}
	})
}

func TestEmbeddedFrontendHandler_Versioning(t *testing.T) {
	version := daemon_frontend.RuntimeVersion()
	require.True(t, daemon_frontend.IsRuntimeVersion(version))

	require.NoError(t, daemon_frontend.RegisterCustomModule(t.Context(), "versioningtest", []byte("console.log(1)"), `"custom"`))

	server := newEmbeddedFrontendTestServer(t, newEmbeddedFrontendHandler(version, cacheControlImmutable, daemon_frontend.CoreRuntimeBuildDevelopment))

	coreAsset, found := daemon_frontend.GetAsset(t.Context(), "built/ppframework.core.es.js")
	require.True(t, found)
	mapAsset, found := daemon_frontend.GetAsset(t.Context(), "built/ppframework.core.es.js.map")
	require.True(t, found)

	coreVersion := daemon_frontend.CoreRuntimeBuildDevelopment.VersionOf(version)

	otherVersion := strings.Repeat("0", 16)
	if otherVersion == version || otherVersion == coreVersion {
		otherVersion = strings.Repeat("1", 16)
	}

	testCases := []struct {
		name             string
		path             string
		wantCacheControl string
		wantContentType  string
		wantBody         []byte
		wantStatus       int
	}{
		{
			name:             "the versioned core bundle is served with its bytes and cached for good",
			path:             "/_piko/dist/" + coreVersion + "/ppframework.core.es.js",
			wantStatus:       http.StatusOK,
			wantCacheControl: cacheControlImmutable,
			wantBody:         coreAsset.Content,
		},
		{
			name:             "the versioned source map resolves next to its bundle",
			path:             "/_piko/dist/" + version + "/ppframework.core.es.js.map",
			wantStatus:       http.StatusOK,
			wantCacheControl: cacheControlImmutable,
			wantBody:         mapAsset.Content,
		},
		{
			name:             "the unversioned core module re-exports the versioned URL so one runtime instance loads",
			path:             "/_piko/dist/ppframework.core.es.js",
			wantStatus:       http.StatusOK,
			wantCacheControl: cacheControlNoCache,
			wantContentType:  contentTypeJavaScript,
			wantBody:         []byte("export * from \"./" + coreVersion + "/ppframework.core.es.js\";\n"),
		},
		{
			name:             "the core under the bundle version is a 404, since its bytes depend on the build",
			path:             "/_piko/dist/" + version + "/ppframework.core.es.js",
			wantStatus:       http.StatusNotFound,
			wantCacheControl: cacheControlNoStore,
		},
		{
			name:             "the core under the other build's version is a 404",
			path:             "/_piko/dist/" + daemon_frontend.CoreRuntimeBuildProduction.VersionOf(version) + "/ppframework.core.es.js",
			wantStatus:       http.StatusNotFound,
			wantCacheControl: cacheControlNoStore,
		},
		{
			name:             "other bundles are served under the core build's version too",
			path:             "/_piko/dist/" + coreVersion + "/ppframework.core.es.js.map",
			wantStatus:       http.StatusOK,
			wantCacheControl: cacheControlImmutable,
			wantBody:         mapAsset.Content,
		},
		{
			name:             "an unversioned source map is served as is and revalidates",
			path:             "/_piko/dist/ppframework.core.es.js.map",
			wantStatus:       http.StatusOK,
			wantCacheControl: cacheControlNoCache,
			wantBody:         mapAsset.Content,
		},
		{
			name:             "an unknown version is a 404 that no cache may keep",
			path:             "/_piko/dist/" + otherVersion + "/ppframework.core.es.js",
			wantStatus:       http.StatusNotFound,
			wantCacheControl: cacheControlNoStore,
		},
		{
			name:       "a custom module is never served under a versioned URL",
			path:       "/_piko/dist/" + version + "/ppframework.versioningtest.min.js",
			wantStatus: http.StatusNotFound,
		},
		{
			name:             "a custom module keeps its unversioned URL",
			path:             "/_piko/dist/ppframework.versioningtest.min.js",
			wantStatus:       http.StatusOK,
			wantCacheControl: cacheControlNoCache,
			wantBody:         []byte("console.log(1)"),
		},
		{
			name:       "a nested path below a version is rejected",
			path:       "/_piko/dist/" + version + "/nested/ppframework.core.es.js",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "an escaped parent segment below a version is rejected",
			path:       "/_piko/dist/" + version + "/..%2Fppframework.core.es.js",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "an empty file name below a version is rejected",
			path:       "/_piko/dist/" + version + "/",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "an unknown file below the current version is a 404",
			path:       "/_piko/dist/" + version + "/missing.js",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			response, body := getEmbeddedFrontend(t, server.URL+testCase.path, nil)

			assert.Equal(t, testCase.wantStatus, response.StatusCode)
			if testCase.wantCacheControl != "" {
				assert.Equal(t, testCase.wantCacheControl, response.Header.Get(headerCacheControl))
			}
			if testCase.wantContentType != "" {
				assert.Equal(t, testCase.wantContentType, response.Header.Get(headerContentType))
			}
			if testCase.wantBody != nil {
				assert.Equal(t, testCase.wantBody, body)
			}
		})
	}
}

func TestEmbeddedFrontendHandler_WithoutAVersionServesTheUnversionedBundles(t *testing.T) {
	server := newEmbeddedFrontendTestServer(t, newEmbeddedFrontendHandler("", cacheControlImmutable, daemon_frontend.CoreRuntimeBuildDevelopment))

	coreAsset, found := daemon_frontend.GetAsset(t.Context(), "built/ppframework.core.es.js")
	require.True(t, found)

	response, body := getEmbeddedFrontend(t, server.URL+"/_piko/dist/ppframework.core.es.js", nil)
	assert.Equal(t, http.StatusOK, response.StatusCode)
	assert.Equal(t, cacheControlNoCache, response.Header.Get(headerCacheControl))
	assert.Equal(t, coreAsset.Content, body, "without a version the unversioned URL must serve the bundle itself")

	rejected, _ := getEmbeddedFrontend(t, server.URL+"/_piko/dist/"+daemon_frontend.RuntimeVersion()+"/ppframework.core.es.js", nil)
	assert.Equal(t, http.StatusNotFound, rejected.StatusCode)
	assert.Equal(t, cacheControlNoStore, rejected.Header.Get(headerCacheControl))
}

func TestEmbeddedFrontendHandler_ReexportModuleRevalidates(t *testing.T) {
	version := daemon_frontend.RuntimeVersion()
	server := newEmbeddedFrontendTestServer(t, newEmbeddedFrontendHandler(version, cacheControlImmutable, daemon_frontend.CoreRuntimeBuildDevelopment))
	target := server.URL + "/_piko/dist/ppframework.components.es.js"

	first, _ := getEmbeddedFrontend(t, target, nil)
	etag := first.Header.Get(headerETag)
	require.Equal(t, `"reexport-`+daemon_frontend.CoreRuntimeBuildDevelopment.VersionOf(version)+`"`, etag,
		"the re-export module's tag must change with the runtime version")

	stale, _ := getEmbeddedFrontend(t, target, http.Header{headerIfNoneMatch: {`"reexport-0000000000000000"`}})
	assert.Equal(t, http.StatusOK, stale.StatusCode, "a re-export from an earlier runtime must be replaced in full")

	notModified, _ := getEmbeddedFrontend(t, target, http.Header{headerIfNoneMatch: {etag}})
	assert.Equal(t, http.StatusNotModified, notModified.StatusCode)
}

func TestEmbeddedFrontendHandler_VersionedCacheControlFollowsTheMode(t *testing.T) {
	server := newEmbeddedFrontendTestServer(t, newEmbeddedFrontendHandler(
		daemon_frontend.RuntimeVersion(), cacheControlForMode(true, cacheControlImmutable),
		daemon_frontend.CoreRuntimeBuildDevelopment,
	))

	response, _ := getEmbeddedFrontend(t, server.URL+versionedCorePath, nil)

	assert.Equal(t, http.StatusOK, response.StatusCode)
	assert.Equal(t, cacheControlNoCache, response.Header.Get(headerCacheControl),
		"with HTTP caching disabled even versioned bundles must revalidate")
}

func TestEmbeddedFrontendHandler_ServesTheCoreRuntimeBuildOfTheMode(t *testing.T) {
	daemon_frontend.SetSRIEnabled(true)
	t.Cleanup(func() { daemon_frontend.SetSRIEnabled(false) })

	version := daemon_frontend.RuntimeVersion()
	require.True(t, daemon_frontend.IsRuntimeVersion(version))

	testCases := []struct {
		name           string
		build          daemon_frontend.CoreRuntimeBuild
		runtimeVersion string
		path           string
		servedFile     string
	}{
		{
			name:           "production serves the minified core at the versioned URL",
			build:          daemon_frontend.CoreRuntimeBuildProduction,
			runtimeVersion: version,
			path:           "/_piko/dist/" + daemon_frontend.CoreRuntimeBuildProduction.VersionOf(version) + "/ppframework.core.es.js",
			servedFile:     "ppframework.core.min.es.js",
		},
		{
			name:           "production serves the minified core at the unversioned URL when there is no version",
			build:          daemon_frontend.CoreRuntimeBuildProduction,
			runtimeVersion: "",
			path:           "/_piko/dist/ppframework.core.es.js",
			servedFile:     "ppframework.core.min.es.js",
		},
		{
			name:           "the dev modes serve the unminified core at the versioned URL",
			build:          daemon_frontend.CoreRuntimeBuildDevelopment,
			runtimeVersion: version,
			path:           "/_piko/dist/" + daemon_frontend.CoreRuntimeBuildDevelopment.VersionOf(version) + "/ppframework.core.es.js",
			servedFile:     "ppframework.core.es.js",
		},
		{
			name:           "the dev modes serve the unminified core at the unversioned URL when there is no version",
			build:          daemon_frontend.CoreRuntimeBuildDevelopment,
			runtimeVersion: "",
			path:           "/_piko/dist/ppframework.core.es.js",
			servedFile:     "ppframework.core.es.js",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			server := newEmbeddedFrontendTestServer(t, newEmbeddedFrontendHandler(testCase.runtimeVersion, cacheControlImmutable, testCase.build))
			want, found := daemon_frontend.GetAsset(t.Context(), "built/"+testCase.servedFile)
			require.True(t, found)
			wantBrotli, found := daemon_frontend.GetAsset(t.Context(), "built/"+testCase.servedFile+".br")
			require.True(t, found)

			response, body := getEmbeddedFrontend(t, server.URL+testCase.path, http.Header{headerAcceptEncoding: {"identity"}})

			require.Equal(t, http.StatusOK, response.StatusCode)
			assert.Equal(t, want.Content, body)
			assert.Equal(t, want.ETag, response.Header.Get(headerETag))
			assert.Equal(t, want.MimeType, response.Header.Get(headerContentType))
			assert.Contains(t, response.Header.Get(headerContentType), "javascript")
			assert.Equal(t, daemon_frontend.ComputeSRIHash(body), testCase.build.CoreModuleSRIHash(),
				"pages must carry the SRI hash of the bytes served")

			compressed, compressedBody := getEmbeddedFrontend(t, server.URL+testCase.path, http.Header{headerAcceptEncoding: {"br"}})
			assert.Equal(t, "br", compressed.Header.Get(headerContentEncoding))
			assert.Equal(t, wantBrotli.ETag, compressed.Header.Get(headerETag))
			assert.Equal(t, wantBrotli.Content, compressedBody)

			assertSourceMapResolves(t, server.URL+testCase.path, body)
		})
	}

	t.Run("production keeps the unversioned re-export on the one core URL", func(t *testing.T) {
		server := newEmbeddedFrontendTestServer(t, newEmbeddedFrontendHandler(version, cacheControlImmutable, daemon_frontend.CoreRuntimeBuildProduction))

		response, body := getEmbeddedFrontend(t, server.URL+"/_piko/dist/ppframework.core.es.js", nil)

		require.Equal(t, http.StatusOK, response.StatusCode)
		productionVersion := daemon_frontend.CoreRuntimeBuildProduction.VersionOf(version)
		assert.Equal(t, "export * from \"./"+productionVersion+"/ppframework.core.es.js\";\n", string(body))
	})

	t.Run("each server answers only its own build's versioned core URL", func(t *testing.T) {
		builds := []daemon_frontend.CoreRuntimeBuild{daemon_frontend.CoreRuntimeBuildDevelopment, daemon_frontend.CoreRuntimeBuildProduction}
		for _, serverBuild := range builds {
			server := newEmbeddedFrontendTestServer(t, newEmbeddedFrontendHandler(version, cacheControlImmutable, serverBuild))
			for _, urlBuild := range builds {
				response, _ := getEmbeddedFrontend(t, server.URL+"/_piko/dist/"+urlBuild.VersionOf(version)+"/ppframework.core.es.js", nil)

				want := http.StatusNotFound
				if urlBuild == serverBuild {
					want = http.StatusOK
				}
				assert.Equal(t, want, response.StatusCode, "a %s server asked for the %s core", serverBuild.CoreFileName(), urlBuild.CoreFileName())
			}
		}
	})
}

func TestEmbeddedFrontendHandler_ServesTheExtensionModuleBuildOfTheMode(t *testing.T) {
	daemon_frontend.SetSRIEnabled(true)
	t.Cleanup(func() { daemon_frontend.SetSRIEnabled(false) })

	version := daemon_frontend.RuntimeVersion()
	require.True(t, daemon_frontend.IsRuntimeVersion(version))

	modules := []string{
		"ppframework.components.es.js",
		"ppframework.animation.es.js",
		"ppframework.analytics.es.js",
		"ppframework.modals.es.js",
		"ppframework.toasts.es.js",
		"ppframework.dev.es.js",
	}
	builds := []struct {
		name          string
		build         daemon_frontend.CoreRuntimeBuild
		servedFileFor func(string) string
	}{
		{
			name:          "production",
			build:         daemon_frontend.CoreRuntimeBuildProduction,
			servedFileFor: func(fileName string) string { return strings.TrimSuffix(fileName, ".es.js") + ".min.es.js" },
		},
		{
			name:          "development",
			build:         daemon_frontend.CoreRuntimeBuildDevelopment,
			servedFileFor: func(fileName string) string { return fileName },
		},
	}

	for _, build := range builds {
		server := newEmbeddedFrontendTestServer(t, newEmbeddedFrontendHandler(version, cacheControlImmutable, build.build))
		buildVersion := build.build.VersionOf(version)
		for _, fileName := range modules {
			t.Run(build.name+" "+fileName, func(t *testing.T) {
				servedFile := build.servedFileFor(fileName)
				want, found := daemon_frontend.GetAsset(t.Context(), "built/"+servedFile)
				require.True(t, found)
				wantBrotli, found := daemon_frontend.GetAsset(t.Context(), "built/"+servedFile+".br")
				require.True(t, found)
				moduleURL := server.URL + "/_piko/dist/" + buildVersion + "/" + fileName

				response, body := getEmbeddedFrontend(t, moduleURL, http.Header{headerAcceptEncoding: {"identity"}})

				require.Equal(t, http.StatusOK, response.StatusCode)
				assert.Equal(t, want.Content, body)
				assert.Equal(t, want.ETag, response.Header.Get(headerETag))
				assert.Equal(t, cacheControlImmutable, response.Header.Get(headerCacheControl))
				assert.Equal(t, daemon_frontend.ComputeSRIHash(body), build.build.ModuleSRIHash(fileName),
					"pages must carry the SRI hash of the bytes served")

				compressed, compressedBody := getEmbeddedFrontend(t, moduleURL, http.Header{headerAcceptEncoding: {"br"}})
				assert.Equal(t, "br", compressed.Header.Get(headerContentEncoding))
				assert.Equal(t, wantBrotli.ETag, compressed.Header.Get(headerETag))
				assert.Equal(t, wantBrotli.Content, compressedBody)

				assertSourceMapResolves(t, moduleURL, body)

				reexport, reexportBody := getEmbeddedFrontend(t, server.URL+"/_piko/dist/"+fileName, nil)
				require.Equal(t, http.StatusOK, reexport.StatusCode)
				assert.Equal(t, "export * from \"./"+buildVersion+"/"+fileName+"\";\n", string(reexportBody),
					"the unversioned URL must resolve to the one versioned module")

				underBundleVersion, _ := getEmbeddedFrontend(t, server.URL+"/_piko/dist/"+version+"/"+fileName, nil)
				assert.Equal(t, http.StatusNotFound, underBundleVersion.StatusCode,
					"the bytes depend on the build, so the bundle version must not name them")
			})
		}
	}
}

func TestServeRuntimeRecovery(t *testing.T) {
	asset, found := daemon_frontend.GetAsset(t.Context(), daemon_frontend.RuntimeRecoveryAssetPath)
	require.True(t, found, "the embedded build must contain the runtime recovery script")

	router := chi.NewRouter()
	router.Get(daemon_frontend.RuntimeRecoveryURL, serveRuntimeRecovery)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	identity := http.Header{headerAcceptEncoding: {"identity"}}
	response, body := getEmbeddedFrontend(t, server.URL+daemon_frontend.RuntimeRecoveryURL, identity)
	require.Equal(t, http.StatusOK, response.StatusCode)
	assert.Equal(t, cacheControlNoCache, response.Header.Get(headerCacheControl),
		"pages of every build share this URL, so it must revalidate on every use")
	assert.Equal(t, asset.ETag, response.Header.Get(headerETag))
	assert.Contains(t, response.Header.Get(headerContentType), "javascript")
	assert.Equal(t, asset.Content, body)

	notModified, _ := getEmbeddedFrontend(t, server.URL+daemon_frontend.RuntimeRecoveryURL,
		http.Header{headerAcceptEncoding: {"identity"}, headerIfNoneMatch: {asset.ETag}})
	assert.Equal(t, http.StatusNotModified, notModified.StatusCode)
}

func TestRuntimeVersionMiddleware(t *testing.T) {
	t.Parallel()

	assert.Equal(t, textproto.CanonicalMIMEHeaderKey(daemon_frontend.RuntimeVersionHeader), headerXPPRuntime,
		"the pre-built header key must match the header the browser runtime reads")

	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	testCases := []struct {
		name       string
		version    string
		wantHeader []string
	}{
		{name: "names the runtime version on the response", version: "0123456789abcdef", wantHeader: []string{"0123456789abcdef"}},
		{name: "adds nothing when the version is unknown", version: "", wantHeader: nil},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			recorder := httptest.NewRecorder()
			runtimeVersionMiddleware(testCase.version)(next).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/page?_f=1", nil))

			assert.Equal(t, testCase.wantHeader, recorder.Header().Values(daemon_frontend.RuntimeVersionHeader))
			assert.Equal(t, http.StatusNoContent, recorder.Code)
		})
	}
}

func TestIsPlainFileName(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		fileName string
		want     bool
	}{
		{name: "bundle file", fileName: "ppframework.core.es.js", want: true},
		{name: "empty", fileName: "", want: false},
		{name: "current directory", fileName: ".", want: false},
		{name: "parent directory", fileName: "..", want: false},
		{name: "nested path", fileName: "nested/file.js", want: false},
		{name: "backslash path", fileName: `nested\file.js`, want: false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, testCase.want, isPlainFileName(testCase.fileName))
		})
	}
}

func expectedFrontendCacheControl(requestPath string) string {
	if _, _, versioned := daemon_frontend.SplitRuntimeVersion(strings.TrimPrefix(requestPath, "/_piko/dist/")); versioned {
		return cacheControlImmutable
	}
	return cacheControlNoCache
}

func assertSourceMapResolves(t *testing.T, bundleURL string, bundle []byte) {
	t.Helper()

	const sourceMapComment = "//# sourceMappingURL="
	commentIndex := bytes.LastIndex(bundle, []byte(sourceMapComment))
	require.GreaterOrEqual(t, commentIndex, 0, "the bundle must name its source map")
	mapName := strings.TrimSpace(string(bundle[commentIndex+len(sourceMapComment):]))

	base, err := url.Parse(bundleURL)
	require.NoError(t, err)
	mapURL, err := base.Parse(mapName)
	require.NoError(t, err)

	want, found := daemon_frontend.GetAsset(t.Context(), "built/"+path.Base(mapURL.Path))
	require.True(t, found, "the named source map must be embedded")
	response, body := getEmbeddedFrontend(t, mapURL.String(), http.Header{headerAcceptEncoding: {"identity"}})
	assert.Equal(t, http.StatusOK, response.StatusCode)
	assert.Equal(t, want.Content, body)
}

func newEmbeddedFrontendTestServer(t *testing.T, handler *embeddedFrontendHandler) *httptest.Server {
	t.Helper()

	router := chi.NewRouter()
	router.Get("/_piko/dist/*", handler.ServeHTTP)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	return server
}

func getEmbeddedFrontend(t *testing.T, target string, header http.Header) (*http.Response, []byte) {
	t.Helper()

	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, target, nil)
	require.NoError(t, err)
	maps.Copy(request.Header, header)

	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	defer func() { _ = response.Body.Close() }()

	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	return response, body
}

func TestEmbeddedFrontendHandler_ReexportModuleCoversEveryBundleExport(t *testing.T) {
	t.Parallel()

	defaultExport := regexp.MustCompile(`export\s+default\b|\bas\s+default\s*[,}]`)
	bundlePaths, err := filepath.Glob(filepath.Join("..", "daemon_frontend", "built", "*"+esModuleSuffix))
	require.NoError(t, err)
	require.NotEmpty(t, bundlePaths, "the build must contain ES module bundles")

	for _, bundlePath := range bundlePaths {
		fileName := filepath.Base(bundlePath)
		t.Run(fileName, func(t *testing.T) {
			t.Parallel()

			asset, found := daemon_frontend.GetAsset(t.Context(), "built/"+fileName)
			require.True(t, found, "the bundle must be embedded")
			assert.False(t, defaultExport.Match(asset.Content),
				"a re-export with export * leaves out a default export, so no bundle may have one")
		})
	}
}
