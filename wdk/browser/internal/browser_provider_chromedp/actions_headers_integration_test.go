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

package browser_provider_chromedp

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testHTMLHeaders = `<!DOCTYPE html>
	<html>
	<head><title>Headers Test</title></head>
	<body>
	<div id="content">Headers Test Page</div>
	</body>
	</html>`
)

func TestSetExtraHTTPHeaders(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	server := newTestServer(testHTMLHeaders)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("sets multiple headers", func(t *testing.T) {
			headers := map[string]string{
				"X-Custom-Header": "custom-value",
				"X-Another":       "another-value",
			}
			err := SetExtraHTTPHeaders(ctx, headers)
			if err != nil {
				t.Fatalf("SetExtraHTTPHeaders() error = %v", err)
			}
		})

		t.Run("sets empty headers (clears)", func(t *testing.T) {
			err := SetExtraHTTPHeaders(ctx, map[string]string{})
			if err != nil {
				t.Fatalf("SetExtraHTTPHeaders() error = %v", err)
			}
		})
	})
}

func TestSetAuthorizationHeader(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	server := newTestServer(testHTMLHeaders)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("sets Bearer token", func(t *testing.T) {
			err := SetAuthorizationHeader(ctx, "test-token-12345")
			if err != nil {
				t.Fatalf("SetAuthorizationHeader() error = %v", err)
			}
		})
	})
}

func TestSetBasicAuthHeader(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	server := newTestServer(testHTMLHeaders)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("sets Basic auth credentials", func(t *testing.T) {
			err := SetBasicAuthHeader(ctx, "dXNlcm5hbWU6cGFzc3dvcmQ=")
			if err != nil {
				t.Fatalf("SetBasicAuthHeader() error = %v", err)
			}
		})
	})
}

func TestSetUserAgent(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	server := newTestServer(testHTMLHeaders)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("sets custom user agent", func(t *testing.T) {
			customUA := "Mozilla/5.0 (Custom Test Agent)"
			err := SetUserAgent(ctx, customUA)
			if err != nil {
				t.Fatalf("SetUserAgent() error = %v", err)
			}
		})
	})
}

func TestClearExtraHTTPHeaders(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	server := newTestServer(testHTMLHeaders)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("clears headers after setting", func(t *testing.T) {

			err := SetExtraHTTPHeaders(ctx, map[string]string{
				"X-Test": "value",
			})
			if err != nil {
				t.Fatalf("SetExtraHTTPHeaders() error = %v", err)
			}

			err = ClearExtraHTTPHeaders(ctx)
			if err != nil {
				t.Fatalf("ClearExtraHTTPHeaders() error = %v", err)
			}
		})
	})
}

func TestSetAcceptLanguageHeader(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	server := newTestServer(testHTMLHeaders)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("sets Accept-Language header", func(t *testing.T) {
			err := SetAcceptLanguageHeader(ctx, "fr-FR,fr;q=0.9,en-US;q=0.8")
			if err != nil {
				t.Fatalf("SetAcceptLanguageHeader() error = %v", err)
			}
		})
	})
}

func TestSetCustomHeader(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	server := newTestServer(testHTMLHeaders)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("sets single custom header", func(t *testing.T) {
			err := SetCustomHeader(ctx, "X-Custom", "custom-value")
			if err != nil {
				t.Fatalf("SetCustomHeader() error = %v", err)
			}
		})
	})
}

func newHeaderServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("X-Page", strings.TrimPrefix(r.URL.Path, "/"))
		_, _ = w.Write([]byte(testHTMLHeaders))
	}))
}

func TestGetResponseHeaders(t *testing.T) {
	t.Parallel()
	server := newHeaderServer()
	defer server.Close()

	withTestPage(t, server.URL+"/first", func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)
		ctx.ServerURL = server.URL

		headers, err := GetResponseHeaders(ctx)
		require.NoError(t, err)
		assert.Equal(t, "first", headers["x-page"])
		assert.Equal(t, "text/html; charset=utf-8", headers["content-type"])
		assert.Equal(t, "text/html", headers["contentType"])
		assert.Contains(t, headers, "lastModified")

		require.NoError(t, Navigate(ctx, "/second"))
		headers, err = GetResponseHeaders(ctx)
		require.NoError(t, err)
		assert.Equal(t, "second", headers["x-page"])

		require.NoError(t, GoBack(ctx))
		headers, err = GetResponseHeaders(ctx)
		require.NoError(t, err)
		assert.Equal(t, "first", headers["x-page"])
	})
}

func TestGetResponseHeadersNoHelper(t *testing.T) {
	t.Parallel()
	server := newHeaderServer()
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		headers, err := GetResponseHeaders(&ActionContext{Ctx: page.Ctx()})
		require.NoError(t, err)
		assert.Equal(t, "text/html", headers["contentType"])
		assert.NotContains(t, headers, "x-page")
	})
}
