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

package templater_domain_test

import (
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/daemon/daemon_dto"
	"piko.sh/piko/internal/templater/templater_domain"
	"piko.sh/piko/internal/templater/templater_dto"
)

func TestParseRequestData_DetectLocale_Priority(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		path           string
		setupRequest   func(*http.Request) *http.Request
		expectedLocale string
		description    string
	}{
		{
			name: "route context locale takes highest priority",
			path: "/",
			setupRequest: func(request *http.Request) *http.Request {
				request.AddCookie(&http.Cookie{Name: "piko_locale", Value: "cookie_locale"})
				request.Header.Set("Accept-Language", "header_locale")
				ctx := daemon_dto.WithPikoRequestCtx(request.Context(), &daemon_dto.PikoRequestCtx{
					Locale: "route_locale",
				})
				return request.WithContext(ctx)
			},
			expectedLocale: "route_locale",
			description:    "Route context locale should override all other sources",
		},
		{
			name: "query parameter when no route locale",
			path: "/?locale=query_locale",
			setupRequest: func(request *http.Request) *http.Request {
				request.AddCookie(&http.Cookie{Name: "piko_locale", Value: "cookie_locale"})
				request.Header.Set("Accept-Language", "header_locale")
				return request
			},
			expectedLocale: "query_locale",
			description:    "Query parameter should be second priority",
		},
		{
			name: "cookie when no route or query locale",
			path: "/",
			setupRequest: func(request *http.Request) *http.Request {
				request.AddCookie(&http.Cookie{Name: "piko_locale", Value: "cookie_locale"})
				request.Header.Set("Accept-Language", "header_locale")
				return request
			},
			expectedLocale: "cookie_locale",
			description:    "Cookie should be third priority",
		},
		{
			name: "accept-language header when no other sources",
			path: "/",
			setupRequest: func(request *http.Request) *http.Request {
				request.Header.Set("Accept-Language", "fr-FR,fr;q=0.9,en;q=0.8")
				return request
			},
			expectedLocale: "fr-FR",
			description:    "Should extract first locale from Accept-Language header",
		},
		{
			name: "default locale when no sources provided",
			path: "/",
			setupRequest: func(request *http.Request) *http.Request {
				return request
			},
			expectedLocale: "en_GB",
			description:    "Should fallback to default locale",
		},
		{
			name: "empty route context locale is ignored",
			path: "/?locale=query_locale",
			setupRequest: func(request *http.Request) *http.Request {
				ctx := daemon_dto.WithPikoRequestCtx(request.Context(), &daemon_dto.PikoRequestCtx{
					Locale: "",
				})
				return request.WithContext(ctx)
			},
			expectedLocale: "query_locale",
			description:    "Empty string in route context should be skipped",
		},
		{
			name: "empty query parameter is ignored",
			path: "/?locale=",
			setupRequest: func(request *http.Request) *http.Request {
				request.AddCookie(&http.Cookie{Name: "piko_locale", Value: "cookie_locale"})
				return request
			},
			expectedLocale: "cookie_locale",
			description:    "Empty query param should fall through to cookie",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			router := chi.NewRouter()
			var capturedRD *templater_dto.RequestData

			router.Get("/", func(w http.ResponseWriter, r *http.Request) {
				var err error
				capturedRD, err = templater_domain.ParseRequestData(r, "")
				require.NoError(t, err, tc.description)
			})

			request := httptest.NewRequest("GET", tc.path, nil)
			request = tc.setupRequest(request)

			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, request)

			require.NotNil(t, capturedRD)
			assert.Equal(t, tc.expectedLocale, capturedRD.Locale(), tc.description)
		})
	}
}

func TestParseRequestData_ExtractPathParams(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		expectedParams map[string]string
		name           string
		routePattern   string
		requestPath    string
	}{
		{
			name:         "single path parameter",
			routePattern: "/users/{id}",
			requestPath:  "/users/123",
			expectedParams: map[string]string{
				"id": "123",
			},
		},
		{
			name:         "multiple path parameters",
			routePattern: "/users/{userId}/posts/{postId}",
			requestPath:  "/users/42/posts/99",
			expectedParams: map[string]string{
				"userId": "42",
				"postId": "99",
			},
		},
		{
			name:           "no path parameters",
			routePattern:   "/static/page",
			requestPath:    "/static/page",
			expectedParams: map[string]string{},
		},
		{
			name:         "path parameter with special characters",
			routePattern: "/files/{filename}",
			requestPath:  "/files/my-file_v2.txt",
			expectedParams: map[string]string{
				"filename": "my-file_v2.txt",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			router := chi.NewRouter()
			var capturedRD *templater_dto.RequestData
			router.Get(tc.routePattern, func(w http.ResponseWriter, r *http.Request) {
				var err error
				capturedRD, err = templater_domain.ParseRequestData(r, "")
				require.NoError(t, err)
			})

			request := httptest.NewRequest("GET", tc.requestPath, nil)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, request)

			require.NotNil(t, capturedRD)
			assert.Equal(t, tc.expectedParams, capturedRD.PathParams())
		})
	}
}

func TestParseRequestData_ExtractQueryParams(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		queryString   string
		expectedQuery map[string][]string
		expectedPath  map[string]string
		description   string
	}{
		{
			name:        "single query parameter",
			queryString: "?search=test",
			expectedQuery: map[string][]string{
				"search": {"test"},
			},
			expectedPath: map[string]string{},
			description:  "Should parse single query param",
		},
		{
			name:        "multiple query parameters",
			queryString: "?page=1&limit=10&sort=desc",
			expectedQuery: map[string][]string{
				"page":  {"1"},
				"limit": {"10"},
				"sort":  {"desc"},
			},
			expectedPath: map[string]string{},
			description:  "Should parse multiple query params",
		},
		{
			name:        "query parameter with multiple values",
			queryString: "?tag=go&tag=testing&tag=unit",
			expectedQuery: map[string][]string{
				"tag": {"go", "testing", "unit"},
			},
			expectedPath: map[string]string{},
			description:  "Should handle multiple values for same key",
		},
		{
			name:          "special params parameter with bracketed syntax",
			queryString:   "?params=[id=123][name=test]",
			expectedQuery: map[string][]string{},
			expectedPath: map[string]string{
				"id":   "123",
				"name": "test",
			},
			description: "Should parse params into PathParams",
		},
		{
			name:        "params parameter is excluded from query params",
			queryString: "?params=[id=123]&search=test",
			expectedQuery: map[string][]string{
				"search": {"test"},
			},
			expectedPath: map[string]string{
				"id": "123",
			},
			description: "params should be excluded from QueryParams",
		},
		{
			name:          "empty query string",
			queryString:   "",
			expectedQuery: map[string][]string{},
			expectedPath:  map[string]string{},
			description:   "Should handle empty query string",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			router := chi.NewRouter()
			var capturedRD *templater_dto.RequestData

			router.Get("/test", func(w http.ResponseWriter, r *http.Request) {
				var err error
				capturedRD, err = templater_domain.ParseRequestData(r, "")
				require.NoError(t, err, tc.description)
			})

			request := httptest.NewRequest("GET", "/test"+tc.queryString, nil)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, request)

			require.NotNil(t, capturedRD)
			assert.Equal(t, tc.expectedQuery, capturedRD.QueryParams(), tc.description)

			for k, v := range tc.expectedPath {
				assert.Equal(t, v, capturedRD.PathParam(k), "PathParams[%s] should match", k)
			}
		})
	}
}

func TestParseRequestData_ParseFormData(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		setupRequest func() *http.Request
		expectedForm map[string][]string
		name         string
		expectError  bool
	}{
		{
			name: "url-encoded form data",
			setupRequest: func() *http.Request {
				body := strings.NewReader("username=testuser&email=test@example.com")
				request := httptest.NewRequest("POST", "/", body)
				request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				return request
			},
			expectedForm: map[string][]string{
				"username": {"testuser"},
				"email":    {"test@example.com"},
			},
			expectError: false,
		},
		{
			name: "multipart form data",
			setupRequest: func() *http.Request {
				body := strings.NewReader("--boundary\r\n" +
					"Content-Disposition: form-data; name=\"field1\"\r\n\r\n" +
					"value1\r\n" +
					"--boundary\r\n" +
					"Content-Disposition: form-data; name=\"field2\"\r\n\r\n" +
					"value2\r\n" +
					"--boundary--\r\n")
				request := httptest.NewRequest("POST", "/", body)
				request.Header.Set("Content-Type", "multipart/form-data; boundary=boundary")
				return request
			},
			expectedForm: map[string][]string{
				"field1": {"value1"},
				"field2": {"value2"},
			},
			expectError: false,
		},
		{
			name: "no form data for GET request",
			setupRequest: func() *http.Request {
				return httptest.NewRequest("GET", "/", nil)
			},
			expectedForm: map[string][]string{},
			expectError:  false,
		},
		{
			name: "no form data for JSON content type",
			setupRequest: func() *http.Request {
				body := strings.NewReader(`{"key":"value"}`)
				request := httptest.NewRequest("POST", "/", body)
				request.Header.Set("Content-Type", "application/json")
				return request
			},
			expectedForm: map[string][]string{},
			expectError:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			router := chi.NewRouter()
			var capturedRD *templater_dto.RequestData
			var capturedErr error

			handler := func(w http.ResponseWriter, r *http.Request) {
				capturedRD, capturedErr = templater_domain.ParseRequestData(r, "")
			}

			router.Post("/", handler)
			router.Get("/", handler)

			request := tc.setupRequest()
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, request)

			if tc.expectError {
				assert.Error(t, capturedErr)
			} else {
				require.NoError(t, capturedErr)
				require.NotNil(t, capturedRD)
				assert.Equal(t, tc.expectedForm, capturedRD.FormData())
			}
		})
	}
}

func TestParseRequestData_FullRequest(t *testing.T) {
	t.Parallel()

	router := chi.NewRouter()
	var capturedRD *templater_dto.RequestData

	router.Post("/users/{userId}/posts", func(w http.ResponseWriter, r *http.Request) {
		var err error
		capturedRD, err = templater_domain.ParseRequestData(r, "")
		require.NoError(t, err)
	})

	body := strings.NewReader("title=My+Post&content=Post+content")
	request := httptest.NewRequest("POST", "/users/42/posts?page=1&limit=10", body)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Accept-Language", "fr-FR")
	request.AddCookie(&http.Cookie{Name: "piko_locale", Value: "en_US"})

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, request)

	require.NotNil(t, capturedRD)

	assert.Equal(t, "POST", capturedRD.Method())
	assert.Equal(t, "42", capturedRD.PathParam("userId"))
	assert.Equal(t, []string{"1"}, capturedRD.QueryParamValues("page"))
	assert.Equal(t, []string{"10"}, capturedRD.QueryParamValues("limit"))
	assert.Equal(t, []string{"My Post"}, capturedRD.FormValues("title"))
	assert.Equal(t, []string{"Post content"}, capturedRD.FormValues("content"))
	assert.Equal(t, "en_US", capturedRD.Locale())
	assert.Equal(t, "en_GB", capturedRD.DefaultLocale())
}

func TestParseRequestData_NilRequest(t *testing.T) {
	t.Parallel()

	rd, err := templater_domain.ParseRequestData(nil, "")

	require.NoError(t, err)
	require.NotNil(t, rd)

	assert.Equal(t, "GET", rd.Method())
	assert.Equal(t, "", rd.Host())
	assert.Equal(t, "en_GB", rd.Locale())
	assert.Equal(t, "en_GB", rd.DefaultLocale())
	assert.NotNil(t, rd.URL())
	assert.NotNil(t, rd.PathParams())
	assert.NotNil(t, rd.QueryParams())
	assert.NotNil(t, rd.FormData())
	assert.NotNil(t, rd.Context())
}

func TestNewRequestDataFromAction(t *testing.T) {
	t.Parallel()

	router := chi.NewRouter()
	var capturedRD *templater_dto.RequestData

	router.Post("/actions/{actionId}", func(w http.ResponseWriter, r *http.Request) {

		_ = r.ParseForm()
		capturedRD = templater_domain.NewRequestDataFromAction(r)
	})

	body := strings.NewReader("field1=value1&field2=value2")
	request := httptest.NewRequest("POST", "/actions/submit?ref=homepage", body)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Accept-Language", "de-DE")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, request)

	require.NotNil(t, capturedRD)
	assert.Equal(t, "POST", capturedRD.Method())
	assert.Equal(t, "submit", capturedRD.PathParam("actionId"))
	assert.Equal(t, []string{"homepage"}, capturedRD.QueryParamValues("ref"))
	assert.Equal(t, []string{"value1"}, capturedRD.FormValues("field1"))
	assert.Equal(t, "de-DE", capturedRD.Locale())
}

func TestParseCustomParams(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		input          string
		expectedParams map[string]string
		description    string
	}{
		{
			name:  "single bracketed parameter",
			input: "[id=123]",
			expectedParams: map[string]string{
				"id": "123",
			},
			description: "Should parse single parameter",
		},
		{
			name:  "multiple bracketed parameters",
			input: "[id=123][name=test][status=active]",
			expectedParams: map[string]string{
				"id":     "123",
				"name":   "test",
				"status": "active",
			},
			description: "Should parse multiple parameters",
		},
		{
			name:           "empty string",
			input:          "",
			expectedParams: map[string]string{},
			description:    "Should return empty map for empty input",
		},
		{
			name:           "whitespace only",
			input:          "   ",
			expectedParams: map[string]string{},
			description:    "Should return empty map for whitespace",
		},
		{
			name:           "malformed brackets",
			input:          "[id=123",
			expectedParams: map[string]string{},
			description:    "Should ignore malformed brackets",
		},
		{
			name:           "missing value",
			input:          "[id=]",
			expectedParams: map[string]string{},
			description:    "Should ignore params with missing value",
		},
		{
			name:  "special characters in value",
			input: "[url=https://example.com][path=/foo/bar]",
			expectedParams: map[string]string{
				"url":  "https://example.com",
				"path": "/foo/bar",
			},
			description: "Should handle special characters in values",
		},
		{
			name:  "mixed valid and invalid params",
			input: "[id=123][invalid][name=test]",
			expectedParams: map[string]string{
				"id":   "123",
				"name": "test",
			},
			description: "Should parse valid params and skip invalid ones",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			router := chi.NewRouter()
			var capturedRD *templater_dto.RequestData

			router.Get("/test", func(w http.ResponseWriter, r *http.Request) {
				var err error
				capturedRD, err = templater_domain.ParseRequestData(r, "")
				require.NoError(t, err, tc.description)
			})

			request := httptest.NewRequest("GET", "/test?params="+url.QueryEscape(tc.input), nil)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, request)

			require.NotNil(t, capturedRD)
			for k, v := range tc.expectedParams {
				assert.Equal(t, v, capturedRD.PathParam(k), "PathParams[%s] should match for: %s", k, tc.description)
			}
		})
	}
}

func TestParseRequestData_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("request with empty host", func(t *testing.T) {
		t.Parallel()

		router := chi.NewRouter()
		var capturedRD *templater_dto.RequestData

		router.Get("/test", func(w http.ResponseWriter, r *http.Request) {
			var err error
			capturedRD, err = templater_domain.ParseRequestData(r, "")
			require.NoError(t, err)
		})

		request := httptest.NewRequest("GET", "/test", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, request)

		require.NotNil(t, capturedRD)
		assert.Contains(t, []string{"", "example.com"}, capturedRD.Host())
	})

	t.Run("preserves request context", func(t *testing.T) {
		t.Parallel()

		type contextKey string
		customKey := contextKey("custom")
		customValue := "test-value"

		router := chi.NewRouter()
		var capturedRD *templater_dto.RequestData

		router.Get("/test", func(w http.ResponseWriter, r *http.Request) {
			var err error
			capturedRD, err = templater_domain.ParseRequestData(r, "")
			require.NoError(t, err)
		})

		request := httptest.NewRequest("GET", "/test", nil)
		ctx := context.WithValue(request.Context(), customKey, customValue)
		request = request.WithContext(ctx)

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, request)

		require.NotNil(t, capturedRD)
		assert.Equal(t, customValue, capturedRD.Context().Value(customKey))
	})

	t.Run("query parameter with URL encoded values", func(t *testing.T) {
		t.Parallel()

		router := chi.NewRouter()
		var capturedRD *templater_dto.RequestData

		router.Get("/search", func(w http.ResponseWriter, r *http.Request) {
			var err error
			capturedRD, err = templater_domain.ParseRequestData(r, "")
			require.NoError(t, err)
		})

		request := httptest.NewRequest("GET", "/search?q=hello+world&tag=go%2Ftesting", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, request)

		require.NotNil(t, capturedRD)
		assert.Equal(t, []string{"hello world"}, capturedRD.QueryParamValues("q"))
		assert.Equal(t, []string{"go/testing"}, capturedRD.QueryParamValues("tag"))
	})
}

func TestParseRequestData_NilRequest_WithDefaultLocale(t *testing.T) {
	t.Parallel()

	rd, err := templater_domain.ParseRequestData(nil, "fr_FR")

	require.NoError(t, err)
	require.NotNil(t, rd)
	assert.Equal(t, "fr_FR", rd.Locale())
	assert.Equal(t, "fr_FR", rd.DefaultLocale())
	assert.Equal(t, "GET", rd.Method())
}

func TestParseRequestData_NilRequest_WithEmptyLocale(t *testing.T) {
	t.Parallel()

	rd, err := templater_domain.ParseRequestData(nil, "")

	require.NoError(t, err)
	require.NotNil(t, rd)
	assert.Equal(t, "en_GB", rd.Locale(), "should use fallback locale when empty")
	assert.Equal(t, "en_GB", rd.DefaultLocale())
}

func TestParseRequestData_DefaultLocale_UsedInBuilder(t *testing.T) {
	t.Parallel()

	router := chi.NewRouter()
	var capturedRD *templater_dto.RequestData

	router.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		var err error
		capturedRD, err = templater_domain.ParseRequestData(r, "ja_JP")
		require.NoError(t, err)
	})

	request := httptest.NewRequest("GET", "/test", nil)
	request.Header.Set("Accept-Language", "de-DE")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, request)

	require.NotNil(t, capturedRD)
	assert.Equal(t, "ja_JP", capturedRD.DefaultLocale())
	assert.Equal(t, "de-DE", capturedRD.Locale())
}

func TestParseRequestData_DefaultLocale_Empty_FallsBack(t *testing.T) {
	t.Parallel()

	router := chi.NewRouter()
	var capturedRD *templater_dto.RequestData

	router.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		var err error
		capturedRD, err = templater_domain.ParseRequestData(r, "")
		require.NoError(t, err)
	})

	request := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, request)

	require.NotNil(t, capturedRD)
	assert.Equal(t, "en_GB", capturedRD.DefaultLocale(), "empty defaultLocale should fall back to en_GB")
}

func TestParseRequestData_AcceptLanguage_SingleLocale(t *testing.T) {
	t.Parallel()

	router := chi.NewRouter()
	var capturedRD *templater_dto.RequestData

	router.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		var err error
		capturedRD, err = templater_domain.ParseRequestData(r, "")
		require.NoError(t, err)
	})

	request := httptest.NewRequest("GET", "/test", nil)
	request.Header.Set("Accept-Language", "es-ES")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, request)

	require.NotNil(t, capturedRD)
	assert.Equal(t, "es-ES", capturedRD.Locale(), "single Accept-Language should be used directly")
}

func TestNewRequestDataFromAction_NoQueryString(t *testing.T) {
	t.Parallel()

	router := chi.NewRouter()
	var capturedRD *templater_dto.RequestData

	router.Post("/actions/{id}", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		capturedRD = templater_domain.NewRequestDataFromAction(r)
	})

	body := strings.NewReader("field=value")
	request := httptest.NewRequest("POST", "/actions/42", body)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, request)

	require.NotNil(t, capturedRD)
	assert.Equal(t, "POST", capturedRD.Method())
	assert.Equal(t, "42", capturedRD.PathParam("id"))
	assert.Equal(t, []string{"value"}, capturedRD.FormValues("field"))
	assert.Empty(t, capturedRD.QueryParams())
}

func TestNewRequestDataFromAction_WithMultipartForm(t *testing.T) {
	t.Parallel()

	router := chi.NewRouter()
	var capturedRD *templater_dto.RequestData

	router.Post("/upload", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseMultipartForm(32 << 20)
		capturedRD = templater_domain.NewRequestDataFromAction(r)
	})

	body := &strings.Builder{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("name", "test-file")
	_ = writer.WriteField("description", "a test upload")
	_ = writer.Close()

	request := httptest.NewRequest("POST", "/upload?ref=main", strings.NewReader(body.String()))
	request.Header.Set("Content-Type", writer.FormDataContentType())
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, request)

	require.NotNil(t, capturedRD)
	assert.Equal(t, "POST", capturedRD.Method())
	assert.Equal(t, []string{"main"}, capturedRD.QueryParamValues("ref"))
	assert.Equal(t, []string{"test-file"}, capturedRD.FormValues("name"))
	assert.Equal(t, []string{"a test upload"}, capturedRD.FormValues("description"))
}

func TestNewRequestDataFromAction_NoForm(t *testing.T) {
	t.Parallel()

	router := chi.NewRouter()
	var capturedRD *templater_dto.RequestData

	router.Get("/action", func(w http.ResponseWriter, r *http.Request) {
		capturedRD = templater_domain.NewRequestDataFromAction(r)
	})

	request := httptest.NewRequest("GET", "/action", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, request)

	require.NotNil(t, capturedRD)
	assert.Equal(t, "GET", capturedRD.Method())
	assert.Empty(t, capturedRD.FormData())
}

func TestNewRequestDataFromAction_LocaleFromCookie(t *testing.T) {
	t.Parallel()

	router := chi.NewRouter()
	var capturedRD *templater_dto.RequestData

	router.Post("/action", func(w http.ResponseWriter, r *http.Request) {
		capturedRD = templater_domain.NewRequestDataFromAction(r)
	})

	request := httptest.NewRequest("POST", "/action", nil)
	request.AddCookie(&http.Cookie{Name: "piko_locale", Value: "pt_BR"})
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, request)

	require.NotNil(t, capturedRD)
	assert.Equal(t, "pt_BR", capturedRD.Locale())
}

func TestNewRequestDataFromAction_LocaleFromRouteContext(t *testing.T) {
	t.Parallel()

	router := chi.NewRouter()
	var capturedRD *templater_dto.RequestData

	router.Post("/action", func(w http.ResponseWriter, r *http.Request) {
		ctx := daemon_dto.WithPikoRequestCtx(r.Context(), &daemon_dto.PikoRequestCtx{
			Locale: "it_IT",
		})
		r = r.WithContext(ctx)
		capturedRD = templater_domain.NewRequestDataFromAction(r)
	})

	request := httptest.NewRequest("POST", "/action?locale=query_locale", nil)
	request.AddCookie(&http.Cookie{Name: "piko_locale", Value: "cookie_locale"})
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, request)

	require.NotNil(t, capturedRD)
	assert.Equal(t, "it_IT", capturedRD.Locale(), "route context should take priority")
}

func TestParseRequestData_MultipartFormError(t *testing.T) {
	t.Parallel()

	router := chi.NewRouter()
	var capturedErr error

	router.Post("/test", func(w http.ResponseWriter, r *http.Request) {
		_, capturedErr = templater_domain.ParseRequestData(r, "")
	})

	body := strings.NewReader("this is not valid multipart data")
	request := httptest.NewRequest("POST", "/test", body)
	request.Header.Set("Content-Type", "multipart/form-data; boundary=boundary")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, request)

	require.Error(t, capturedErr)
	assert.Contains(t, capturedErr.Error(), "parsing form data")
}

func TestParseRequestData_URLEncodedFormError(t *testing.T) {
	t.Parallel()

	router := chi.NewRouter()
	var capturedErr error

	router.Post("/test", func(w http.ResponseWriter, r *http.Request) {

		r.Body = http.NoBody
		r.ContentLength = 100

		r.PostForm = nil
		r.Form = nil

		r.URL = &url.URL{RawQuery: "%invalid"}
		_, capturedErr = templater_domain.ParseRequestData(r, "")
	})

	request := httptest.NewRequest("POST", "/test", nil)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, request)

	require.Error(t, capturedErr)
}

func TestParseRequestData_HostPreserved(t *testing.T) {
	t.Parallel()

	router := chi.NewRouter()
	var capturedRD *templater_dto.RequestData

	router.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		r.Host = "mysite.com:8080"
		var err error
		capturedRD, err = templater_domain.ParseRequestData(r, "")
		require.NoError(t, err)
	})

	request := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, request)

	require.NotNil(t, capturedRD)
	assert.Equal(t, "mysite.com:8080", capturedRD.Host())
}

func TestParseRequestData_MethodPreserved(t *testing.T) {
	t.Parallel()

	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			t.Parallel()

			router := chi.NewRouter()
			var capturedRD *templater_dto.RequestData

			handler := func(w http.ResponseWriter, r *http.Request) {
				var err error
				capturedRD, err = templater_domain.ParseRequestData(r, "")
				require.NoError(t, err)
			}

			switch method {
			case "GET":
				router.Get("/test", handler)
			case "POST":
				router.Post("/test", handler)
			case "PUT":
				router.Put("/test", handler)
			case "DELETE":
				router.Delete("/test", handler)
			case "PATCH":
				router.Patch("/test", handler)
			}

			request := httptest.NewRequest(method, "/test", nil)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, request)

			require.NotNil(t, capturedRD)
			assert.Equal(t, method, capturedRD.Method())
		})
	}
}

func TestParseCustomParams_EdgeCases(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		expectedParams map[string]string
		name           string
		input          string
	}{
		{
			name:           "missing key",
			input:          "[=value]",
			expectedParams: map[string]string{},
		},
		{
			name:           "only opening bracket",
			input:          "[key=value",
			expectedParams: map[string]string{},
		},
		{
			name:           "nested brackets",
			input:          "[[key=value]]",
			expectedParams: map[string]string{},
		},
		{
			name:  "value with equals sign",
			input: "[key=val=ue]",
			expectedParams: map[string]string{
				"key": "val=ue",
			},
		},
		{
			name:           "garbage before brackets",
			input:          "garbage[key=value]",
			expectedParams: map[string]string{"key": "value"},
		},
		{
			name:           "no closing bracket",
			input:          "abc[key=value",
			expectedParams: map[string]string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			router := chi.NewRouter()
			var capturedRD *templater_dto.RequestData

			router.Get("/test", func(w http.ResponseWriter, r *http.Request) {
				var err error
				capturedRD, err = templater_domain.ParseRequestData(r, "")
				require.NoError(t, err)
			})

			request := httptest.NewRequest("GET", "/test?params="+url.QueryEscape(tc.input), nil)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, request)

			require.NotNil(t, capturedRD)
			for k, v := range tc.expectedParams {
				assert.Equal(t, v, capturedRD.PathParam(k), "PathParams[%s]", k)
			}
		})
	}
}

func TestNewRequestDataFromAction_QueryParamsIncludesParams(t *testing.T) {
	t.Parallel()

	router := chi.NewRouter()
	var capturedRD *templater_dto.RequestData

	router.Get("/action", func(w http.ResponseWriter, r *http.Request) {
		capturedRD = templater_domain.NewRequestDataFromAction(r)
	})

	request := httptest.NewRequest("GET", "/action?params=test&other=val", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, request)

	require.NotNil(t, capturedRD)
	assert.Equal(t, []string{"test"}, capturedRD.QueryParamValues("params"))
	assert.Equal(t, []string{"val"}, capturedRD.QueryParamValues("other"))
}

func TestParseRequestData_EmptyCookieIgnored(t *testing.T) {
	t.Parallel()

	router := chi.NewRouter()
	var capturedRD *templater_dto.RequestData

	router.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		var err error
		capturedRD, err = templater_domain.ParseRequestData(r, "")
		require.NoError(t, err)
	})

	request := httptest.NewRequest("GET", "/test", nil)
	request.AddCookie(&http.Cookie{Name: "piko_locale", Value: ""})
	request.Header.Set("Accept-Language", "ja-JP")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, request)

	require.NotNil(t, capturedRD)
	assert.Equal(t, "ja-JP", capturedRD.Locale(), "empty cookie value should be ignored")
}

func TestParseRequestData_ForwardsCookies(t *testing.T) {
	t.Parallel()

	router := chi.NewRouter()
	var capturedRD *templater_dto.RequestData

	router.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		var err error
		capturedRD, err = templater_domain.ParseRequestData(r, "en_GB")
		require.NoError(t, err)
	})

	request := httptest.NewRequest("GET", "/test", nil)
	request.AddCookie(&http.Cookie{Name: "session_id", Value: "abc123"})
	request.AddCookie(&http.Cookie{Name: "theme", Value: "dark"})

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, request)

	require.NotNil(t, capturedRD)

	sessionCookie, err := capturedRD.Cookie("session_id")
	require.NoError(t, err)
	assert.Equal(t, "abc123", sessionCookie.Value)

	themeCookie, err := capturedRD.Cookie("theme")
	require.NoError(t, err)
	assert.Equal(t, "dark", themeCookie.Value)

	_, err = capturedRD.Cookie("nonexistent")
	assert.ErrorIs(t, err, http.ErrNoCookie)

	allCookies := capturedRD.Cookies()
	assert.Len(t, allCookies, 2)
}
