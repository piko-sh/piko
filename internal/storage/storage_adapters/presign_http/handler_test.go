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

package presign_http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/storage/storage_domain"
)

func TestCheckIfModifiedSince_NotModified(t *testing.T) {
	t.Parallel()

	lastModified := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	info := &storage_domain.ObjectInfo{
		LastModified: lastModified,
	}

	request := httptest.NewRequest(http.MethodGet, "/file", nil)
	request.Header.Set(headerIfModifiedSince, lastModified.UTC().Format(http.TimeFormat))
	recorder := httptest.NewRecorder()

	result := checkIfModifiedSince(recorder, request, info)

	assert.True(t, result)
	assert.Equal(t, http.StatusNotModified, recorder.Code)
}

func TestCheckIfModifiedSince_Modified(t *testing.T) {
	t.Parallel()

	lastModified := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	clientTime := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	info := &storage_domain.ObjectInfo{
		LastModified: lastModified,
	}

	request := httptest.NewRequest(http.MethodGet, "/file", nil)
	request.Header.Set(headerIfModifiedSince, clientTime.UTC().Format(http.TimeFormat))
	recorder := httptest.NewRecorder()

	result := checkIfModifiedSince(recorder, request, info)

	assert.False(t, result)
	assert.Equal(t, http.StatusOK, recorder.Code)
}

func TestCheckIfModifiedSince_NoHeader(t *testing.T) {
	t.Parallel()

	info := &storage_domain.ObjectInfo{
		LastModified: time.Now(),
	}

	request := httptest.NewRequest(http.MethodGet, "/file", nil)
	recorder := httptest.NewRecorder()

	result := checkIfModifiedSince(recorder, request, info)

	assert.False(t, result)
}

func TestCheckIfModifiedSince_ZeroLastModified(t *testing.T) {
	t.Parallel()

	info := &storage_domain.ObjectInfo{}

	request := httptest.NewRequest(http.MethodGet, "/file", nil)
	request.Header.Set(headerIfModifiedSince, time.Now().UTC().Format(http.TimeFormat))
	recorder := httptest.NewRecorder()

	result := checkIfModifiedSince(recorder, request, info)

	assert.False(t, result)
}

func TestCheckIfModifiedSince_InvalidDateFormat(t *testing.T) {
	t.Parallel()

	info := &storage_domain.ObjectInfo{
		LastModified: time.Now(),
	}

	request := httptest.NewRequest(http.MethodGet, "/file", nil)
	request.Header.Set(headerIfModifiedSince, "not-a-date")
	recorder := httptest.NewRecorder()

	result := checkIfModifiedSince(recorder, request, info)

	assert.False(t, result)
}

func TestCheckIfModifiedSince_SkippedWhenIfNoneMatchPresent(t *testing.T) {
	t.Parallel()

	info := &storage_domain.ObjectInfo{
		LastModified: time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC),
	}

	request := httptest.NewRequest(http.MethodGet, "/file", nil)
	request.Header.Set(headerIfNoneMatch, `"some-etag"`)
	request.Header.Set(headerIfModifiedSince, info.LastModified.UTC().Format(http.TimeFormat))
	recorder := httptest.NewRecorder()

	result := checkIfModifiedSince(recorder, request, info)

	assert.False(t, result)
}

func TestCheckETagMatch_Match(t *testing.T) {
	t.Parallel()

	info := &storage_domain.ObjectInfo{
		ETag: `"abc123"`,
	}

	request := httptest.NewRequest(http.MethodGet, "/file", nil)
	request.Header.Set(headerIfNoneMatch, `"abc123"`)
	recorder := httptest.NewRecorder()

	result := checkETagMatch(recorder, request, info)

	assert.True(t, result)
	assert.Equal(t, http.StatusNotModified, recorder.Code)
}

func TestCheckETagMatch_NoMatch(t *testing.T) {
	t.Parallel()

	info := &storage_domain.ObjectInfo{
		ETag: `"abc123"`,
	}

	request := httptest.NewRequest(http.MethodGet, "/file", nil)
	request.Header.Set(headerIfNoneMatch, `"different"`)
	recorder := httptest.NewRecorder()

	result := checkETagMatch(recorder, request, info)

	assert.False(t, result)
}

func TestCheckETagMatch_EmptyClientETag(t *testing.T) {
	t.Parallel()

	info := &storage_domain.ObjectInfo{
		ETag: `"abc123"`,
	}

	request := httptest.NewRequest(http.MethodGet, "/file", nil)
	recorder := httptest.NewRecorder()

	result := checkETagMatch(recorder, request, info)

	assert.False(t, result)
}

func TestCheckETagMatch_EmptyServerETag(t *testing.T) {
	t.Parallel()

	info := &storage_domain.ObjectInfo{}

	request := httptest.NewRequest(http.MethodGet, "/file", nil)
	request.Header.Set(headerIfNoneMatch, `"abc123"`)
	recorder := httptest.NewRecorder()

	result := checkETagMatch(recorder, request, info)

	assert.False(t, result)
}

func TestDetermineDisposition(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		contentType string
		metadata    map[string]string
		expected    string
	}{
		{
			name:        "metadata inline preference",
			contentType: "application/octet-stream",
			metadata:    map[string]string{storage_domain.MetadataKeyContentDisposition: "inline"},
			expected:    "inline",
		},
		{
			name:        "metadata attachment preference",
			contentType: "image/png",
			metadata:    map[string]string{storage_domain.MetadataKeyContentDisposition: "attachment"},
			expected:    "attachment",
		},
		{
			name:        "metadata invalid preference falls through to attachment",
			contentType: "application/octet-stream",
			metadata:    map[string]string{storage_domain.MetadataKeyContentDisposition: "invalid-value"},
			expected:    "attachment",
		},
		{
			name:        "nil metadata with inline content type",
			contentType: "image/png",
			metadata:    nil,
			expected:    "inline",
		},
		{
			name:        "nil metadata with non-inline content type",
			contentType: "application/octet-stream",
			metadata:    nil,
			expected:    "attachment",
		},
		{
			name:        "empty metadata without disposition key",
			contentType: "image/png",
			metadata:    map[string]string{},
			expected:    "attachment",
		},
		{
			name:        "nil metadata with empty content type",
			contentType: "",
			metadata:    nil,
			expected:    "attachment",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			result := determineDisposition(testCase.contentType, testCase.metadata)
			assert.Equal(t, testCase.expected, result)
		})
	}
}

func TestResolveContentType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		tokenCT  string
		infoCT   string
		expected string
	}{
		{name: "token takes precedence", tokenCT: "image/png", infoCT: "image/jpeg", expected: "image/png"},
		{name: "falls back to info", tokenCT: "", infoCT: "image/jpeg", expected: "image/jpeg"},
		{name: "both empty", tokenCT: "", infoCT: "", expected: ""},
		{name: "only token set", tokenCT: "text/html", infoCT: "", expected: "text/html"},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			result := resolveContentType(testCase.tokenCT, testCase.infoCT)
			assert.Equal(t, testCase.expected, result)
		})
	}
}

func TestSetBasicHeaders(t *testing.T) {
	t.Parallel()

	t.Run("all fields populated", func(t *testing.T) {
		t.Parallel()

		recorder := httptest.NewRecorder()
		lastModified := time.Date(2026, 3, 15, 10, 30, 0, 0, time.UTC)
		info := &storage_domain.ObjectInfo{
			Size:         1024,
			ETag:         `"etag-value"`,
			LastModified: lastModified,
		}

		setBasicHeaders(recorder, info)

		assert.Equal(t, "1024", recorder.Header().Get("Content-Length"))
		assert.Equal(t, `"etag-value"`, recorder.Header().Get(headerETag))
		assert.Equal(t, lastModified.UTC().Format(http.TimeFormat), recorder.Header().Get(headerLastModified))
	})

	t.Run("zero size omitted", func(t *testing.T) {
		t.Parallel()

		recorder := httptest.NewRecorder()
		info := &storage_domain.ObjectInfo{
			ETag: `"etag-value"`,
		}

		setBasicHeaders(recorder, info)

		assert.Empty(t, recorder.Header().Get("Content-Length"))
		assert.Equal(t, `"etag-value"`, recorder.Header().Get(headerETag))
	})

	t.Run("empty etag omitted", func(t *testing.T) {
		t.Parallel()

		recorder := httptest.NewRecorder()
		info := &storage_domain.ObjectInfo{
			Size: 512,
		}

		setBasicHeaders(recorder, info)

		assert.Equal(t, "512", recorder.Header().Get("Content-Length"))
		assert.Empty(t, recorder.Header().Get(headerETag))
	})

	t.Run("zero last modified omitted", func(t *testing.T) {
		t.Parallel()

		recorder := httptest.NewRecorder()
		info := &storage_domain.ObjectInfo{}

		setBasicHeaders(recorder, info)

		assert.Empty(t, recorder.Header().Get(headerLastModified))
	})
}

func TestSetCacheControl(t *testing.T) {
	t.Parallel()

	t.Run("default when nil metadata", func(t *testing.T) {
		t.Parallel()

		recorder := httptest.NewRecorder()
		info := &storage_domain.ObjectInfo{}

		setCacheControl(recorder, info)

		assert.Equal(t, defaultCacheControl, recorder.Header().Get(headerCacheControl))
	})

	t.Run("default when metadata lacks cache control", func(t *testing.T) {
		t.Parallel()

		recorder := httptest.NewRecorder()
		info := &storage_domain.ObjectInfo{
			Metadata: map[string]string{"other-key": "other-value"},
		}

		setCacheControl(recorder, info)

		assert.Equal(t, defaultCacheControl, recorder.Header().Get(headerCacheControl))
	})

	t.Run("custom cache control from metadata", func(t *testing.T) {
		t.Parallel()

		recorder := httptest.NewRecorder()
		info := &storage_domain.ObjectInfo{
			Metadata: map[string]string{storage_domain.MetadataKeyCacheControl: "public, max-age=86400"},
		}

		setCacheControl(recorder, info)

		assert.Equal(t, "public, max-age=86400", recorder.Header().Get(headerCacheControl))
	})

	t.Run("default when cache control value is empty", func(t *testing.T) {
		t.Parallel()

		recorder := httptest.NewRecorder()
		info := &storage_domain.ObjectInfo{
			Metadata: map[string]string{storage_domain.MetadataKeyCacheControl: ""},
		}

		setCacheControl(recorder, info)

		assert.Equal(t, defaultCacheControl, recorder.Header().Get(headerCacheControl))
	})
}

func TestSetContentDisposition(t *testing.T) {
	t.Parallel()

	t.Run("inline with filename", func(t *testing.T) {
		t.Parallel()

		recorder := httptest.NewRecorder()

		setContentDisposition(recorder, "photo.jpg", "image/jpeg", nil)

		assert.Equal(t, `inline; filename="photo.jpg"`, recorder.Header().Get(headerContentDisposition))
	})

	t.Run("attachment with filename", func(t *testing.T) {
		t.Parallel()

		recorder := httptest.NewRecorder()

		setContentDisposition(recorder, "archive.zip", "application/zip", nil)

		assert.Equal(t, `attachment; filename="archive.zip"`, recorder.Header().Get(headerContentDisposition))
	})

	t.Run("no filename inline type", func(t *testing.T) {
		t.Parallel()

		recorder := httptest.NewRecorder()

		setContentDisposition(recorder, "", "image/png", nil)

		assert.Equal(t, "inline", recorder.Header().Get(headerContentDisposition))
	})

	t.Run("no filename attachment type", func(t *testing.T) {
		t.Parallel()

		recorder := httptest.NewRecorder()

		setContentDisposition(recorder, "", "application/octet-stream", nil)

		assert.Equal(t, "attachment", recorder.Header().Get(headerContentDisposition))
	})

	t.Run("metadata overrides content type", func(t *testing.T) {
		t.Parallel()

		recorder := httptest.NewRecorder()
		metadata := map[string]string{storage_domain.MetadataKeyContentDisposition: "attachment"}

		setContentDisposition(recorder, "", "image/png", metadata)

		assert.Equal(t, "attachment", recorder.Header().Get(headerContentDisposition))
	})
}

func TestHandleTokenValidationError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		inputError      error
		expectedStatus  int
		expectedCode    string
		expectedMessage string
	}{
		{
			name:            "expired token",
			inputError:      storage_domain.ErrPresignTokenExpired,
			expectedStatus:  http.StatusForbidden,
			expectedCode:    "token_expired",
			expectedMessage: "Token has expired",
		},
		{
			name:            "invalid signature",
			inputError:      storage_domain.ErrPresignTokenSignature,
			expectedStatus:  http.StatusUnauthorized,
			expectedCode:    "invalid_signature",
			expectedMessage: "Token signature is invalid",
		},
		{
			name:            "invalid token format",
			inputError:      storage_domain.ErrPresignTokenInvalid,
			expectedStatus:  http.StatusBadRequest,
			expectedCode:    "invalid_token",
			expectedMessage: "Token format is invalid",
		},
		{
			name:            "unknown error",
			inputError:      assert.AnError,
			expectedStatus:  http.StatusBadRequest,
			expectedCode:    "token_error",
			expectedMessage: "Token validation failed",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var capturedStatus int
			var capturedCode string
			var capturedMessage string
			writeFunction := func(_ context.Context, _ http.ResponseWriter, status int, code string, message string) {
				capturedStatus = status
				capturedCode = code
				capturedMessage = message
			}

			recorder := httptest.NewRecorder()
			handleTokenValidationError(context.Background(), recorder, testCase.inputError, writeFunction)

			assert.Equal(t, testCase.expectedStatus, capturedStatus)
			assert.Equal(t, testCase.expectedCode, capturedCode)
			assert.Equal(t, testCase.expectedMessage, capturedMessage)
		})
	}
}

func TestUploadHandler_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	handler := NewHandler(nil, storage_domain.PresignConfig{})

	request := httptest.NewRequest(http.MethodGet, "/upload?token=abc", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "method_not_allowed")
}

func TestUploadHandler_MissingToken(t *testing.T) {
	t.Parallel()

	handler := NewHandler(nil, storage_domain.PresignConfig{})

	request := httptest.NewRequest(http.MethodPut, "/upload", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "missing_token")
}

func TestUploadHandler_InvalidToken(t *testing.T) {
	t.Parallel()

	secret := make([]byte, 32)
	for i := range secret {
		secret[i] = byte(i)
	}
	config := storage_domain.PresignConfig{
		Secret: secret,
	}
	handler := NewHandler(nil, config)

	request := httptest.NewRequest(http.MethodPut, "/upload?token=invalid-token-value", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	require.True(t, recorder.Code == http.StatusBadRequest || recorder.Code == http.StatusUnauthorized || recorder.Code == http.StatusForbidden,
		"expected 400, 401, or 403 but got %d", recorder.Code)
}

func TestDownloadHandler_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	handler := NewDownloadHandler(nil, storage_domain.PresignConfig{})

	request := httptest.NewRequest(http.MethodPost, "/download?token=abc", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "method_not_allowed")
}

func TestDownloadHandler_MissingToken(t *testing.T) {
	t.Parallel()

	handler := NewDownloadHandler(nil, storage_domain.PresignConfig{})

	request := httptest.NewRequest(http.MethodGet, "/download", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "missing_token")
}

func TestDownloadHandler_InvalidToken(t *testing.T) {
	t.Parallel()

	secret := make([]byte, 32)
	for i := range secret {
		secret[i] = byte(i)
	}
	config := storage_domain.PresignConfig{
		Secret: secret,
	}
	handler := NewDownloadHandler(nil, config)

	request := httptest.NewRequest(http.MethodGet, "/download?token=bad.token", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	require.True(t, recorder.Code == http.StatusBadRequest || recorder.Code == http.StatusUnauthorized || recorder.Code == http.StatusForbidden,
		"expected 400, 401, or 403 but got %d", recorder.Code)
}

func TestPublicDownloadHandler_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	handler := NewPublicDownloadHandler(nil)

	request := httptest.NewRequest(http.MethodPost, "/_piko/storage/public/s3/repo/key.txt", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusMethodNotAllowed, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "method_not_allowed")
}

func TestPublicDownloadHandler_InvalidPath(t *testing.T) {
	t.Parallel()

	handler := NewPublicDownloadHandler(nil)

	request := httptest.NewRequest(http.MethodGet, "/too/short", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "invalid_path")
}

func TestNewHandler(t *testing.T) {
	t.Parallel()

	handler := NewHandler(nil, storage_domain.PresignConfig{})
	require.NotNil(t, handler)
}

func TestNewDownloadHandler(t *testing.T) {
	t.Parallel()

	handler := NewDownloadHandler(nil, storage_domain.PresignConfig{})
	require.NotNil(t, handler)
}

func TestNewPublicDownloadHandler(t *testing.T) {
	t.Parallel()

	handler := NewPublicDownloadHandler(nil)
	require.NotNil(t, handler)
}
