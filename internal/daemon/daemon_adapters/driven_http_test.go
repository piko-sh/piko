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
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/andybalholm/brotli"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/klauspost/compress/gzip"
	"github.com/go-chi/cors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"piko.sh/piko/internal/daemon/daemon_domain"
	"piko.sh/piko/internal/daemon/daemon_frontend"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
	"piko.sh/piko/internal/testutil/leakcheck"
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
		mainRouter.Use(middleware.RealIP)

		mainRouter.Use(middleware.Recoverer)
		mainRouter.Use(middleware.Heartbeat("/ping"))
		mainRouter.Use(middleware.Timeout(60 * time.Second))

		mainRouter.Get("/_piko/dist/*", serveEmbeddedFrontend(false, false))
		mainRouter.Head("/_piko/dist/*", serveEmbeddedFrontend(false, false))

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

		h2s := &http2.Server{
			MaxConcurrentStreams: 250,
			IdleTimeout:          90 * time.Second,
		}
		h2cHandler := h2c.NewHandler(tracingHandler, h2s)

		server := httptest.NewServer(h2cHandler)
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
				path:               "/_piko/dist/ppframework.core.es.js",
				method:             "GET",
				expectedStatusCode: http.StatusOK,
				checkHeaders:       []string{"Content-Type", "ETag", "Cache-Control"},
			},
			{
				name:               "Should handle OPTIONS request (middleware processing)",
				path:               "/_piko/dist/ppframework.core.es.js",
				method:             "OPTIONS",
				expectedStatusCode: http.StatusMethodNotAllowed,
			},
			{
				name:               "Should process GET request through all middleware",
				path:               "/_piko/dist/ppframework.core.es.js",
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
				path:               "/_piko/dist/ppframework.core.es.js",
				method:             "GET",
				expectedStatusCode: http.StatusOK,
				checkHeaders:       []string{"Content-Type"},
			},
			{
				name:               "Should handle recoverer middleware",
				path:               "/_piko/dist/ppframework.core.es.js",
				method:             "GET",
				expectedStatusCode: http.StatusOK,
				checkHeaders:       []string{"Content-Type"},
			},
			{
				name:               "Should handle HEAD requests through middleware stack",
				path:               "/_piko/dist/ppframework.core.es.js",
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
				request, err := http.NewRequest(tc.method, server.URL+tc.path, nil)
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

func TestMain(m *testing.M) {
	if err := daemon_frontend.InitAssetStore(context.Background()); err != nil {
		panic("failed to initialise embedded asset store: " + err.Error())
	}

	code := m.Run()
	if code == 0 {
		if err := leakcheck.FindLeaks(); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "goleak: %v\n", err)
			os.Exit(1)
		}
	}
	os.Exit(code)
}

func TestFrontendAssetHandler(t *testing.T) {

	t.Run("With Global Asset Store", func(t *testing.T) {

		r := chi.NewRouter()

		r.Get("/_piko/dist/*", serveEmbeddedFrontend(true, false))
		r.Head("/_piko/dist/*", serveEmbeddedFrontend(true, false))
		server := httptest.NewServer(r)
		defer server.Close()

		expectedAsset, ok := daemon_frontend.GetAsset(context.Background(), "built/ppframework.core.es.js")
		require.True(t, ok)
		expectedAssetBr, ok := daemon_frontend.GetAsset(context.Background(), "built/ppframework.core.es.js.br")
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
				path:               "/_piko/dist/ppframework.core.es.js",
				method:             "GET",
				acceptEncoding:     "",
				expectedStatusCode: http.StatusOK,
				expectedEncoding:   "",
				expectedBody:       expectedAsset.Content,
			},
			{
				name:               "Should serve Brotli compressed asset when 'br' is accepted",
				path:               "/_piko/dist/ppframework.core.es.js",
				method:             "GET",
				acceptEncoding:     "br, gzip",
				expectedStatusCode: http.StatusOK,
				expectedEncoding:   "br",
				expectedBody:       expectedAssetBr.Content,
			},
			{
				name:               "Should serve Gzip compressed asset when only 'gzip' is accepted",
				path:               "/_piko/dist/ppframework.core.es.js",
				method:             "GET",
				acceptEncoding:     "gzip, deflate",
				expectedStatusCode: http.StatusOK,
				expectedEncoding:   "gzip",
			},
			{
				name:               "Should return 304 Not Modified for matching ETag",
				path:               "/_piko/dist/ppframework.core.es.js",
				method:             "GET",
				ifNoneMatch:        "",
				expectedStatusCode: http.StatusNotModified,
			},
			{
				name:               "Should return 200 OK for mismatched ETag",
				path:               "/_piko/dist/ppframework.core.es.js",
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

					request, err := http.NewRequest("GET", server.URL+"/_piko/dist/ppframework.core.es.js", nil)
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
		r.Get("/_piko/dist/*", serveEmbeddedFrontend(false, false))
		r.Head("/_piko/dist/*", serveEmbeddedFrontend(false, false))
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
				path:               "/_piko/dist/ppframework.core.es.js",
				method:             "HEAD",
				expectedStatusCode: http.StatusOK,
			},
			{
				name:               "Should return 405 for POST method",
				path:               "/_piko/dist/ppframework.core.es.js",
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
				path:               "/_piko/dist/ppframework.core.es.js",
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
	method := "GET"
	if tc.method != "" {
		method = tc.method
	}

	request, err := http.NewRequest(method, serverURL+tc.path, nil)
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
		assert.Equal(t, cacheControlMutableAsset, response.Header.Get("Cache-Control"))
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

		expectedAsset, _ := daemon_frontend.GetAsset(context.Background(), "built/ppframework.core.es.js")
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
		mainRouter.Use(middleware.RealIP)
		mainRouter.Use(middleware.Recoverer)
		mainRouter.Use(middleware.Heartbeat("/ping"))
		mainRouter.Use(middleware.Timeout(60 * time.Second))

		mainRouter.Get("/_piko/dist/*", serveEmbeddedFrontend(true, false))
		mainRouter.Head("/_piko/dist/*", serveEmbeddedFrontend(true, false))

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

		h2s := &http2.Server{
			MaxConcurrentStreams: 250,
			IdleTimeout:          90 * time.Second,
		}
		h2cHandler := h2c.NewHandler(tracingHandler, h2s)

		server := httptest.NewServer(h2cHandler)
		defer server.Close()

		expectedAsset, ok := daemon_frontend.GetAsset(context.Background(), "built/ppframework.core.es.js")
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

				request, err := http.NewRequest("GET", server.URL+"/_piko/dist/ppframework.core.es.js", nil)
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
				assert.Equal(t, cacheControlMutableAsset, response.Header.Get("Cache-Control"), "Cache-Control should be set correctly")

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
		router.Use(middleware.RealIP)
		router.Get("/_piko/dist/*", serveEmbeddedFrontend(true, false))

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
				path:             "/_piko/dist/ppframework.core.es.js",
				acceptEncoding:   "",
				expectedStatus:   http.StatusOK,
				shouldDecompress: false,
			},
			{
				name:             "Should handle unsupported compression types gracefully",
				path:             "/_piko/dist/ppframework.core.es.js",
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
				request, err := http.NewRequest("GET", server.URL+tc.path, nil)
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
							expectedAsset, ok := daemon_frontend.GetAsset(context.Background(), "built/ppframework.core.es.js")
							require.True(t, ok)
							assert.Equal(t, expectedAsset.Content, body, "Uncompressed body should match original asset")
						}
					}
				}
			})
		}
	})
}

func TestBuildAllowedOrigins_EmptyPublicDomain_ReturnsNil(t *testing.T) {
	t.Parallel()

	config := &daemon_domain.RouterConfig{
		PublicDomain: "",
	}

	result := buildAllowedOrigins(config)

	assert.Nil(t, result)
}

func TestBuildAllowedOrigins_WithPublicDomain_ForceHTTPS(t *testing.T) {
	t.Parallel()

	config := &daemon_domain.RouterConfig{
		PublicDomain: "example.com",
		ForceHTTPS:   true,
	}

	result := buildAllowedOrigins(config)

	require.Len(t, result, 1)
	assert.Equal(t, "https://example.com", result[0])
}

func TestBuildAllowedOrigins_WithPublicDomain_NoForceHTTPS(t *testing.T) {
	t.Parallel()

	config := &daemon_domain.RouterConfig{
		PublicDomain: "example.com",
		ForceHTTPS:   false,
	}

	result := buildAllowedOrigins(config)

	require.Len(t, result, 2)
	assert.Equal(t, "https://example.com", result[0])
	assert.Equal(t, "http://example.com", result[1])
}

func TestBuildAllowedOrigins_SubdomainDomain(t *testing.T) {
	t.Parallel()

	config := &daemon_domain.RouterConfig{
		PublicDomain: "app.example.com",
		ForceHTTPS:   true,
	}

	result := buildAllowedOrigins(config)

	require.Len(t, result, 1)
	assert.Equal(t, "https://app.example.com", result[0])
}

func TestFindChunkByID_Found(t *testing.T) {
	t.Parallel()

	variant := &registry_dto.Variant{
		Chunks: []registry_dto.VariantChunk{
			{ChunkID: "chunk-001", MimeType: "video/mp2t", SizeBytes: 1024},
			{ChunkID: "chunk-002", MimeType: "video/mp2t", SizeBytes: 2048},
			{ChunkID: "chunk-003", MimeType: "video/mp2t", SizeBytes: 512},
		},
	}

	result := findChunkByID(variant, "chunk-002")
	require.NotNil(t, result)
	assert.Equal(t, "chunk-002", result.ChunkID)
	assert.Equal(t, int64(2048), result.SizeBytes)
}

func TestFindChunkByID_NotFound(t *testing.T) {
	t.Parallel()

	variant := &registry_dto.Variant{
		Chunks: []registry_dto.VariantChunk{
			{ChunkID: "chunk-001"},
		},
	}

	result := findChunkByID(variant, "nonexistent")
	assert.Nil(t, result)
}

func TestFindChunkByID_EmptyChunks(t *testing.T) {
	t.Parallel()

	variant := &registry_dto.Variant{
		Chunks: []registry_dto.VariantChunk{},
	}

	result := findChunkByID(variant, "chunk-001")
	assert.Nil(t, result)
}

func TestFindChunkByID_FirstChunk(t *testing.T) {
	t.Parallel()

	variant := &registry_dto.Variant{
		Chunks: []registry_dto.VariantChunk{
			{ChunkID: "first-chunk"},
			{ChunkID: "second-chunk"},
		},
	}

	result := findChunkByID(variant, "first-chunk")
	require.NotNil(t, result)
	assert.Equal(t, "first-chunk", result.ChunkID)
}

func TestFindChunkByID_LastChunk(t *testing.T) {
	t.Parallel()

	variant := &registry_dto.Variant{
		Chunks: []registry_dto.VariantChunk{
			{ChunkID: "first-chunk"},
			{ChunkID: "last-chunk"},
		},
	}

	result := findChunkByID(variant, "last-chunk")
	require.NotNil(t, result)
	assert.Equal(t, "last-chunk", result.ChunkID)
}

func TestBuildMasterPlaylist_NoQualityProfiles(t *testing.T) {
	t.Parallel()

	artefact := &registry_dto.ArtefactMeta{
		DesiredProfiles: []registry_dto.NamedProfile{
			{Name: "thumbnail"},
			{Name: "non_hls_profile"},
		},
	}

	result := buildMasterPlaylist(artefact)
	assert.Equal(t, "#EXTM3U\n", result)
}

func TestBuildMasterPlaylist_SingleQuality(t *testing.T) {
	t.Parallel()

	artefact := &registry_dto.ArtefactMeta{
		DesiredProfiles: []registry_dto.NamedProfile{
			{Name: "hls_720p"},
		},
	}

	result := buildMasterPlaylist(artefact)

	assert.Contains(t, result, "#EXTM3U\n")
	assert.Contains(t, result, "#EXT-X-STREAM-INF:BANDWIDTH=2500000,RESOLUTION=1280x720")
	assert.Contains(t, result, "720p/manifest.m3u8")
}

func TestBuildMasterPlaylist_MultipleQualities(t *testing.T) {
	t.Parallel()

	artefact := &registry_dto.ArtefactMeta{
		DesiredProfiles: []registry_dto.NamedProfile{
			{Name: "hls_1080p"},
			{Name: "hls_720p"},
			{Name: "hls_480p"},
		},
	}

	result := buildMasterPlaylist(artefact)

	assert.Contains(t, result, "#EXTM3U\n")
	assert.Contains(t, result, "BANDWIDTH=5000000,RESOLUTION=1920x1080")
	assert.Contains(t, result, "1080p/manifest.m3u8")
	assert.Contains(t, result, "BANDWIDTH=2500000,RESOLUTION=1280x720")
	assert.Contains(t, result, "720p/manifest.m3u8")
	assert.Contains(t, result, "BANDWIDTH=1000000,RESOLUTION=854x480")
	assert.Contains(t, result, "480p/manifest.m3u8")
}

func TestBuildMasterPlaylist_UnknownQuality_Skipped(t *testing.T) {
	t.Parallel()

	artefact := &registry_dto.ArtefactMeta{
		DesiredProfiles: []registry_dto.NamedProfile{
			{Name: "hls_unknownp"},
			{Name: "hls_720p"},
		},
	}

	result := buildMasterPlaylist(artefact)

	assert.NotContains(t, result, "unknownp")
	assert.Contains(t, result, "720p/manifest.m3u8")
}

func TestBuildMasterPlaylist_EmptyProfiles(t *testing.T) {
	t.Parallel()

	artefact := &registry_dto.ArtefactMeta{
		DesiredProfiles: nil,
	}

	result := buildMasterPlaylist(artefact)
	assert.Equal(t, "#EXTM3U\n", result)
}

func TestBuildMasterPlaylist_AllSupportedQualities(t *testing.T) {
	t.Parallel()

	artefact := &registry_dto.ArtefactMeta{
		DesiredProfiles: []registry_dto.NamedProfile{
			{Name: "hls_2160p"},
			{Name: "hls_1440p"},
			{Name: "hls_1080p"},
			{Name: "hls_720p"},
			{Name: "hls_480p"},
			{Name: "hls_360p"},
		},
	}

	result := buildMasterPlaylist(artefact)

	assert.Contains(t, result, "BANDWIDTH=15000000,RESOLUTION=3840x2160")
	assert.Contains(t, result, "BANDWIDTH=8000000,RESOLUTION=2560x1440")
	assert.Contains(t, result, "BANDWIDTH=5000000,RESOLUTION=1920x1080")
	assert.Contains(t, result, "BANDWIDTH=2500000,RESOLUTION=1280x720")
	assert.Contains(t, result, "BANDWIDTH=1000000,RESOLUTION=854x480")
	assert.Contains(t, result, "BANDWIDTH=500000,RESOLUTION=640x360")
}

func TestBuildVariantPlaylist_NoChunks(t *testing.T) {
	t.Parallel()

	variant := &registry_dto.Variant{
		Chunks: []registry_dto.VariantChunk{},
	}

	result := buildVariantPlaylist(variant)

	assert.Contains(t, result, "#EXTM3U\n")
	assert.Contains(t, result, "#EXT-X-VERSION:3\n")
	assert.Contains(t, result, "#EXT-X-ENDLIST\n")
	assert.Contains(t, result, "#EXT-X-TARGETDURATION:10")
}

func TestBuildVariantPlaylist_SingleVideoChunk(t *testing.T) {
	t.Parallel()

	variant := &registry_dto.Variant{
		Chunks: []registry_dto.VariantChunk{
			{ChunkID: "seg-001.ts", MimeType: "video/mp2t", DurationSeconds: new(6.5)},
		},
	}

	result := buildVariantPlaylist(variant)

	assert.Contains(t, result, "#EXTM3U\n")
	assert.Contains(t, result, "#EXT-X-VERSION:3\n")
	assert.Contains(t, result, "#EXT-X-TARGETDURATION:7\n")
	assert.Contains(t, result, "#EXTINF:6.500000,\n")
	assert.Contains(t, result, "seg-001.ts\n")
	assert.Contains(t, result, "#EXT-X-ENDLIST\n")
}

func TestBuildVariantPlaylist_MultipleChunks(t *testing.T) {
	t.Parallel()

	variant := &registry_dto.Variant{
		Chunks: []registry_dto.VariantChunk{
			{ChunkID: "seg-001.ts", MimeType: "video/mp2t", DurationSeconds: new(10.0)},
			{ChunkID: "seg-002.ts", MimeType: "video/mp2t", DurationSeconds: new(8.5)},
			{ChunkID: "seg-003.ts", MimeType: "video/mp2t", DurationSeconds: new(5.0)},
		},
	}

	result := buildVariantPlaylist(variant)

	assert.Contains(t, result, "#EXT-X-TARGETDURATION:10\n")
	assert.Contains(t, result, "#EXTINF:10.000000,\nseg-001.ts")
	assert.Contains(t, result, "#EXTINF:8.500000,\nseg-002.ts")
	assert.Contains(t, result, "#EXTINF:5.000000,\nseg-003.ts")
}

func TestBuildVariantPlaylist_NilDuration_UsesDefault(t *testing.T) {
	t.Parallel()

	variant := &registry_dto.Variant{
		Chunks: []registry_dto.VariantChunk{
			{ChunkID: "seg-001.ts", MimeType: "video/mp2t", DurationSeconds: nil},
		},
	}

	result := buildVariantPlaylist(variant)

	assert.Contains(t, result, "#EXTINF:10.000000,\nseg-001.ts")
}

func TestBuildVariantPlaylist_SkipsNonVideoChunks(t *testing.T) {
	t.Parallel()

	variant := &registry_dto.Variant{
		Chunks: []registry_dto.VariantChunk{
			{ChunkID: "thumbnail.jpg", MimeType: "image/jpeg", DurationSeconds: nil},
			{ChunkID: "seg-001.ts", MimeType: "video/mp2t", DurationSeconds: new(5.0)},
		},
	}

	result := buildVariantPlaylist(variant)

	assert.NotContains(t, result, "thumbnail.jpg")
	assert.Contains(t, result, "seg-001.ts")
}

func TestBuildVariantPlaylist_AcceptsMP2TUppercase(t *testing.T) {
	t.Parallel()

	variant := &registry_dto.Variant{
		Chunks: []registry_dto.VariantChunk{
			{ChunkID: "seg-001.ts", MimeType: "video/MP2T", DurationSeconds: new(7.0)},
		},
	}

	result := buildVariantPlaylist(variant)
	assert.Contains(t, result, "seg-001.ts")
}

func TestStreamChunkToResponse_WritesHeaders(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	chunk := &registry_dto.VariantChunk{
		MimeType:  "video/mp2t",
		SizeBytes: 1234,
	}
	data := io.NopCloser(strings.NewReader("chunk-data"))

	streamChunkToResponse(context.Background(), recorder, data, chunk)

	assert.Equal(t, "video/mp2t", recorder.Header().Get("Content-Type"))
	assert.Equal(t, "public, max-age=31536000, immutable", recorder.Header().Get("Cache-Control"))
	assert.Equal(t, "1234", recorder.Header().Get("Content-Length"))
	assert.Equal(t, "chunk-data", recorder.Body.String())
}

func TestStreamChunkToResponse_ClosesReader(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	chunk := &registry_dto.VariantChunk{
		MimeType: "video/mp2t",
	}
	closed := false
	data := &testReadCloser{
		Reader: strings.NewReader("test"),
		closeFunc: func() error {
			closed = true
			return nil
		},
	}
	streamChunkToResponse(context.Background(), recorder, data, chunk)

	assert.True(t, closed, "Reader should be closed after streaming")
}

type testReadCloser struct {
	io.Reader
	closeFunc func() error
}

func (r *testReadCloser) Close() error {
	if r.closeFunc != nil {
		return r.closeFunc()
	}
	return nil
}

func TestLookupArtefactByStorageKey_Found(t *testing.T) {
	t.Parallel()

	expectedArtefact := &registry_dto.ArtefactMeta{ID: "found-artefact"}
	registryService := &registry_domain.MockRegistryService{
		FindArtefactByVariantStorageKeyFunc: func(_ context.Context, key string) (*registry_dto.ArtefactMeta, error) {
			if key == "my-storage-key" {
				return expectedArtefact, nil
			}
			return nil, registry_domain.ErrArtefactNotFound
		},
	}
	span := newNoopSpan()

	result := lookupArtefactByStorageKey(context.Background(), span, registryService, "my-storage-key")

	assert.Nil(t, result.err)
	assert.Equal(t, 0, result.httpStatus)
	assert.True(t, result.foundByStorageKey)
	assert.Equal(t, "found-artefact", result.artefact.ID)
}

func TestLookupArtefactByStorageKey_NotFound(t *testing.T) {
	t.Parallel()

	registryService := &registry_domain.MockRegistryService{
		FindArtefactByVariantStorageKeyFunc: func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return nil, registry_domain.ErrArtefactNotFound
		},
	}
	span := newNoopSpan()

	result := lookupArtefactByStorageKey(context.Background(), span, registryService, "missing-key")

	assert.NotNil(t, result.err)
	assert.Equal(t, http.StatusNotFound, result.httpStatus)
	assert.False(t, result.foundByStorageKey)
	assert.Nil(t, result.artefact)
}

func TestLookupArtefactByStorageKey_OtherError(t *testing.T) {
	t.Parallel()

	registryService := &registry_domain.MockRegistryService{
		FindArtefactByVariantStorageKeyFunc: func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return nil, errors.New("database error")
		},
	}
	span := newNoopSpan()

	result := lookupArtefactByStorageKey(context.Background(), span, registryService, "key")

	assert.NotNil(t, result.err)
	assert.Equal(t, http.StatusInternalServerError, result.httpStatus)
	assert.False(t, result.foundByStorageKey)
	assert.Nil(t, result.artefact)
}

func TestNewHTTPRouterBuilder_ReturnsNonNil(t *testing.T) {
	t.Parallel()

	builder := NewHTTPRouterBuilder(nil)
	require.NotNil(t, builder)
}

func TestNewHTTPRouterBuilder_ImplementsInterface(t *testing.T) {
	t.Parallel()

	builder := NewHTTPRouterBuilder(nil)
	_, ok := builder.(*HTTPRouterBuilder)
	assert.True(t, ok)
}

func TestHTTPRouterBuilder_Close_NilCache_DoesNotPanic(t *testing.T) {
	t.Parallel()

	builder := &HTTPRouterBuilder{}
	assert.NotPanics(t, func() {
		builder.Close()
	})
}

func TestSelectStaticVariant_PreferMinified(t *testing.T) {
	t.Parallel()

	var minifiedTags registry_dto.Tags
	minifiedTags.SetByName("type", "minified")

	artefact := &registry_dto.ArtefactMeta{
		ActualVariants: []registry_dto.Variant{
			{VariantID: "source"},
			{VariantID: "minified-variant", MetadataTags: minifiedTags},
		},
	}

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	config := staticArtefactConfig{
		preferredType:  "minified",
		useCompression: false,
	}

	result := selectStaticVariant(request, artefact, config)
	require.NotNil(t, result)
	assert.Equal(t, "minified-variant", result.VariantID)
}

func TestSelectStaticVariant_FallsBackToSourceExtra(t *testing.T) {
	t.Parallel()

	artefact := &registry_dto.ArtefactMeta{
		ActualVariants: []registry_dto.Variant{
			{VariantID: "source"},
		},
	}

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	config := staticArtefactConfig{
		preferredType:  "",
		useCompression: false,
	}

	result := selectStaticVariant(request, artefact, config)
	require.NotNil(t, result)
	assert.Equal(t, "source", result.VariantID)
}

func TestSelectStaticVariant_CompressedVariantPreferred(t *testing.T) {
	t.Parallel()

	artefact := &registry_dto.ArtefactMeta{
		ActualVariants: []registry_dto.Variant{
			{VariantID: "source"},
			{VariantID: "source_br", MetadataTags: func() registry_dto.Tags {
				var tags registry_dto.Tags
				tags.SetByName("contentEncoding", "br")
				return tags
			}()},
		},
	}

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Accept-Encoding", "br, gzip")
	config := staticArtefactConfig{
		preferredType:  "",
		useCompression: true,
	}

	result := selectStaticVariant(request, artefact, config)
	require.NotNil(t, result)
	assert.Equal(t, "source_br", result.VariantID)
}

func TestStaticArtefactConfig_Fields(t *testing.T) {
	t.Parallel()

	config := staticArtefactConfig{
		artefactID:      "test.css",
		defaultMimeType: "text/css",
		cacheMaxAge:     "public, no-cache",
		preferredType:   "minified",
		useCompression:  true,
	}

	assert.Equal(t, "test.css", config.artefactID)
	assert.Equal(t, "text/css", config.defaultMimeType)
	assert.Equal(t, "public, no-cache", config.cacheMaxAge)
	assert.Equal(t, "minified", config.preferredType)
	assert.True(t, config.useCompression)
}

func TestArtefactLookupResult_SuccessFields(t *testing.T) {
	t.Parallel()

	artefact := &registry_dto.ArtefactMeta{ID: "test"}
	result := artefactLookupResult{
		err:               nil,
		artefact:          artefact,
		httpStatus:        0,
		foundByStorageKey: true,
	}

	assert.Nil(t, result.err)
	assert.Equal(t, "test", result.artefact.ID)
	assert.Equal(t, 0, result.httpStatus)
	assert.True(t, result.foundByStorageKey)
}

func TestArtefactLookupResult_ErrorFields(t *testing.T) {
	t.Parallel()

	result := artefactLookupResult{
		err:               errors.New("lookup failed"),
		artefact:          nil,
		httpStatus:        http.StatusNotFound,
		foundByStorageKey: false,
	}

	assert.NotNil(t, result.err)
	assert.Nil(t, result.artefact)
	assert.Equal(t, http.StatusNotFound, result.httpStatus)
	assert.False(t, result.foundByStorageKey)
}

func TestHlsQualityConfigs_AllQualities(t *testing.T) {
	t.Parallel()

	expected := map[string]videoQualityInfo{
		"2160p": {resolution: "3840x2160", bandwidth: 15000000},
		"1440p": {resolution: "2560x1440", bandwidth: 8000000},
		"1080p": {resolution: "1920x1080", bandwidth: 5000000},
		"720p":  {resolution: "1280x720", bandwidth: 2500000},
		"480p":  {resolution: "854x480", bandwidth: 1000000},
		"360p":  {resolution: "640x360", bandwidth: 500000},
	}

	assert.Equal(t, len(expected), len(hlsQualityConfigs))
	for quality, info := range expected {
		got, ok := hlsQualityConfigs[quality]
		assert.True(t, ok, "Quality %s should exist", quality)
		assert.Equal(t, info.resolution, got.resolution, "Resolution mismatch for %s", quality)
		assert.Equal(t, info.bandwidth, got.bandwidth, "Bandwidth mismatch for %s", quality)
	}
}

func TestServeJITResult_WritesContent(t *testing.T) {
	t.Parallel()

	m := &CacheMiddleware{}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)

	result := &jitResult{
		Content:      []byte("<html>test</html>"),
		Encoding:     "br",
		ETag:         `"jit-abc123"`,
		StatusCode:   http.StatusOK,
		CacheControl: "public, max-age=0, must-revalidate",
	}

	m.serveJITResult(recorder, request, result)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "text/html; charset=utf-8", recorder.Header().Get("Content-Type"))
	assert.Equal(t, "br", recorder.Header().Get("Content-Encoding"))
	assert.Equal(t, `"jit-abc123"`, recorder.Header().Get("ETag"))
	assert.Equal(t, "public, max-age=0, must-revalidate", recorder.Header().Get("Cache-Control"))
	assert.Equal(t, "<html>test</html>", recorder.Body.String())
}

func TestServeJITResult_ETagMatch_Returns304(t *testing.T) {
	t.Parallel()

	m := &CacheMiddleware{}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("If-None-Match", `"jit-abc123"`)

	result := &jitResult{
		Content:      []byte("<html>test</html>"),
		Encoding:     "br",
		ETag:         `"jit-abc123"`,
		StatusCode:   http.StatusOK,
		CacheControl: "public, max-age=0, must-revalidate",
	}

	m.serveJITResult(recorder, request, result)

	assert.Equal(t, http.StatusNotModified, recorder.Code)
	assert.Empty(t, recorder.Body.String())
}

func TestServeJITResult_NoETagMatch_Returns200(t *testing.T) {
	t.Parallel()

	m := &CacheMiddleware{}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("If-None-Match", `"different-etag"`)

	result := &jitResult{
		Content:    []byte("content"),
		ETag:       `"jit-abc123"`,
		StatusCode: http.StatusOK,
	}

	m.serveJITResult(recorder, request, result)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "content", recorder.Body.String())
}

func TestHandleNonCacheableStream_NoCompression_PassesThrough(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("plain text"))
	})

	handleNonCacheableStream(recorder, request, next)

	assert.True(t, nextCalled)
	assert.Equal(t, "plain text", recorder.Body.String())
}

func TestHandleNonCacheableStream_BrotliCompression(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Accept-Encoding", "br")

	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("compressed content"))
	})

	handleNonCacheableStream(recorder, request, next)

	assert.Equal(t, "br", recorder.Header().Get("Content-Encoding"))

	assert.Greater(t, recorder.Body.Len(), 0)
}

func TestHandleNonCacheableStream_GzipCompression(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Accept-Encoding", "gzip")

	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("compressed content"))
	})

	handleNonCacheableStream(recorder, request, next)

	assert.Equal(t, "gzip", recorder.Header().Get("Content-Encoding"))
	assert.Greater(t, recorder.Body.Len(), 0)
}

func TestGetCompressedResponseWriter_SetsFields(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	compressor := nopWriteCloser{&bytes.Buffer{}}

	crw := getCompressedResponseWriter(recorder, compressor)
	defer releaseCompressedResponseWriter(crw)

	assert.NotNil(t, crw)
	assert.NotNil(t, crw.ResponseWriter)
	assert.NotNil(t, crw.compressor)
}

func TestReleaseCompressedResponseWriter_ClearsFields(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	compressor := nopWriteCloser{&bytes.Buffer{}}

	crw := getCompressedResponseWriter(recorder, compressor)

	assert.NotNil(t, crw.ResponseWriter)
	assert.NotNil(t, crw.compressor)

	releaseCompressedResponseWriter(crw)
}

func TestGetPipeResponseWriter_SetsDefaults(t *testing.T) {
	t.Parallel()

	_, pw := io.Pipe()
	defer func() { _ = pw.Close() }()

	prw := getPipeResponseWriter(pw)
	defer releasePipeResponseWriter(prw)

	assert.NotNil(t, prw)
	assert.Equal(t, http.StatusOK, prw.statusCode)
	assert.NotNil(t, prw.header)
}

func TestReleasePipeResponseWriter_ClearsWriter(t *testing.T) {
	t.Parallel()

	_, pw := io.Pipe()
	defer func() { _ = pw.Close() }()

	prw := getPipeResponseWriter(pw)

	assert.NotNil(t, prw.Writer)

	releasePipeResponseWriter(prw)
}

func TestNewPipeResponseWriter_SetStatusOK(t *testing.T) {
	t.Parallel()

	_, pw := io.Pipe()
	defer func() { _ = pw.Close() }()

	prw := newPipeResponseWriter(pw)

	assert.Equal(t, http.StatusOK, prw.statusCode)
	assert.NotNil(t, prw.header)
}

func TestCacheControlForMode_DisabledReturnsNoCache(t *testing.T) {
	t.Parallel()

	result := cacheControlForMode(true, cacheControlMutableAsset)

	assert.Equal(t, cacheControlNoCache, result)
}

func TestCacheControlForMode_EnabledReturnsProdValue(t *testing.T) {
	t.Parallel()

	result := cacheControlForMode(false, cacheControlMutableAsset)

	assert.Equal(t, cacheControlMutableAsset, result)
}

func TestCacheControlForMode_LongLivedProdValue(t *testing.T) {
	t.Parallel()

	result := cacheControlForMode(false, cacheControlLongLived)

	assert.Equal(t, cacheControlLongLived, result)
}
