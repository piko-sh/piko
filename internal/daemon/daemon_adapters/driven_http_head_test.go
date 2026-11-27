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

package daemon_adapters

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newHeadTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	router := chi.NewRouter()
	router.Use(headAsGetMiddleware)
	router.Get("/page", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.WriteString(w, "<html>hello</html>")
	})
	router.Post("/action", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	return server
}

func TestHeadAsGetMiddleware_ServesGetRouteWithoutBody(t *testing.T) {
	t.Parallel()

	server := newHeadTestServer(t)

	request, err := http.NewRequestWithContext(t.Context(), http.MethodHead, server.URL+"/page", nil)
	require.NoError(t, err)
	response, err := server.Client().Do(request)
	require.NoError(t, err)
	defer func() { _ = response.Body.Close() }()

	assert.Equal(t, http.StatusOK, response.StatusCode)
	assert.Equal(t, "text/html; charset=utf-8", response.Header.Get("Content-Type"))
	assert.Equal(t, int64(len("<html>hello</html>")), response.ContentLength)
	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	assert.Empty(t, body)
}

func TestHeadAsGetMiddleware_LeavesOtherMethodsAlone(t *testing.T) {
	t.Parallel()

	server := newHeadTestServer(t)

	headAction, err := http.NewRequestWithContext(t.Context(), http.MethodHead, server.URL+"/action", nil)
	require.NoError(t, err)
	response, err := server.Client().Do(headAction)
	require.NoError(t, err)
	_ = response.Body.Close()
	assert.Equal(t, http.StatusMethodNotAllowed, response.StatusCode, "HEAD must not reach a POST-only route")

	getPage, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL+"/page", nil)
	require.NoError(t, err)
	response, err = server.Client().Do(getPage)
	require.NoError(t, err)
	defer func() { _ = response.Body.Close() }()
	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	assert.Equal(t, "<html>hello</html>", string(body))
}
