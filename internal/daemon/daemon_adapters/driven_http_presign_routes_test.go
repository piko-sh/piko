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
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/daemon/daemon_domain"
)

type recordingHandler struct {
	method string
	called bool
}

func (h *recordingHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	h.called = true
	h.method = request.Method

	writer.WriteHeader(http.StatusOK)
}

func newPresignRouter(t *testing.T, upload, download http.Handler) chi.Router {
	t.Helper()

	router := chi.NewRouter()
	builder := &HTTPRouterBuilder{}

	builder.setupPresignRoutes(router, daemon_domain.RouterDependencies{
		PresignUploadHandler:   upload,
		PresignDownloadHandler: download,
	})

	return router
}

func TestPresignPreflightIsAnsweredByCORS(t *testing.T) {
	t.Parallel()

	builder := &HTTPRouterBuilder{}
	upload := &recordingHandler{}
	router := chi.NewRouter()

	router.Group(func(group chi.Router) {
		group.Use(cors.Handler(cors.Options{
			AllowedOrigins: []string{"*"},
			AllowedMethods: []string{http.MethodPut, http.MethodPost, http.MethodOptions},
		}))

		builder.setupPresignRoutes(group, daemon_domain.RouterDependencies{PresignUploadHandler: upload})

		group.Handle("/*", http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			writer.WriteHeader(http.StatusOK)
		}))
	})

	request := httptest.NewRequest(http.MethodOptions, presignUploadPath, nil)
	request.Header.Set("Origin", "https://example.test")
	request.Header.Set("Access-Control-Request-Method", http.MethodPut)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	assert.False(t, upload.called, "a preflight must not reach the upload handler")
	assert.NotEmpty(t, recorder.Header().Get("Access-Control-Allow-Origin"),
		"the preflight must be answered with CORS headers, or the browser blocks every upload")
	assert.Less(t, recorder.Code, http.StatusBadRequest)
}

func TestPresignRoutesRegisterTheirMethods(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		method   string
		path     string
		reaches  bool
		expected int
	}{
		{name: "UploadPut", method: http.MethodPut, path: presignUploadPath, reaches: true, expected: http.StatusOK},
		{name: "UploadPost", method: http.MethodPost, path: presignUploadPath, reaches: true, expected: http.StatusOK},
		{name: "UploadGet", method: http.MethodGet, path: presignUploadPath, reaches: false, expected: http.StatusMethodNotAllowed},
		{name: "DownloadGet", method: http.MethodGet, path: presignDownloadPath, reaches: true, expected: http.StatusOK},
		{name: "DownloadHead", method: http.MethodHead, path: presignDownloadPath, reaches: true, expected: http.StatusOK},
		{name: "DownloadPut", method: http.MethodPut, path: presignDownloadPath, reaches: false, expected: http.StatusMethodNotAllowed},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			upload := &recordingHandler{}
			download := &recordingHandler{}
			router := newPresignRouter(t, upload, download)

			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(testCase.method, testCase.path, nil))

			require.Equal(t, testCase.expected, recorder.Code)

			reached := upload.called || download.called
			assert.Equal(t, testCase.reaches, reached)
		})
	}
}

func TestPresignRoutesSkippedWhenHandlersAbsent(t *testing.T) {
	t.Parallel()

	router := newPresignRouter(t, nil, nil)

	for _, path := range []string{presignUploadPath, presignDownloadPath} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPut, path, nil))

		assert.Equal(t, http.StatusNotFound, recorder.Code, "path %s", path)
	}
}

func TestDynamicRoutesSurviveBothPresignAndAuthGuard(t *testing.T) {
	t.Parallel()

	builder := &HTTPRouterBuilder{}
	router := chi.NewRouter()

	deps := daemon_domain.RouterDependencies{
		PresignUploadHandler:   &recordingHandler{},
		PresignDownloadHandler: &recordingHandler{},
	}

	require.NotPanics(t, func() {
		router.Group(func(group chi.Router) {
			builder.setupPresignRoutes(group, deps)

			group.Use(func(next http.Handler) http.Handler { return next })
			group.Handle("/*", http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.WriteHeader(http.StatusOK)
			}))
		})
	}, "registering presign routes must not stop later middleware from being installed")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPut, presignUploadPath, nil))
	assert.Equal(t, http.StatusOK, recorder.Code)
}

func TestPresignRoutesBypassLaterMiddleware(t *testing.T) {
	t.Parallel()

	builder := &HTTPRouterBuilder{}
	router := chi.NewRouter()
	upload := &recordingHandler{}

	guardCalls := 0

	router.Group(func(group chi.Router) {
		builder.setupPresignRoutes(group, daemon_domain.RouterDependencies{PresignUploadHandler: upload})

		group.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				guardCalls++

				next.ServeHTTP(writer, request)
			})
		})
		group.Handle("/*", http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			writer.WriteHeader(http.StatusOK)
		}))
	})

	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPut, presignUploadPath, nil))
	assert.True(t, upload.called)
	assert.Zero(t, guardCalls, "the presigned token is the credential; the guard must not run")

	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/page", nil))
	assert.Equal(t, 1, guardCalls, "ordinary routes must still run the guard")
}
