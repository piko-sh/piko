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
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/cache/cache_adapters/provider_otter"
	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/captcha/captcha_domain"
	"piko.sh/piko/internal/captcha/captcha_dto"
	"piko.sh/piko/internal/daemon/daemon_domain"
	"piko.sh/piko/internal/daemon/daemon_dto"
	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
	"piko.sh/piko/internal/provider/provider_domain"
	"piko.sh/piko/internal/ratelimiter/ratelimiter_dto"
	"piko.sh/piko/internal/security/security_domain"
	"piko.sh/piko/internal/security/security_dto"
)

type configurableCSRFService struct {
	validateErr    error
	validateResult bool
}

func (m *configurableCSRFService) Name() string { return "ConfigurableCSRF" }

func (m *configurableCSRFService) Check(_ context.Context, _ healthprobe_dto.CheckType) healthprobe_dto.Status {
	return healthprobe_dto.Status{Name: "ConfigurableCSRF"}
}

func (m *configurableCSRFService) GenerateCSRFPair(
	_ http.ResponseWriter,
	_ *http.Request,
	_ *bytes.Buffer,
) (security_dto.CSRFPair, error) {
	return security_dto.CSRFPair{}, nil
}

func (m *configurableCSRFService) ValidateCSRFPair(
	_ *http.Request,
	_ string,
	_ []byte,
) (bool, error) {
	return m.validateResult, m.validateErr
}

var _ security_domain.CSRFTokenService = (*configurableCSRFService)(nil)

func TestNewActionHandler(t *testing.T) {
	t.Run("creates handler with empty registry", func(t *testing.T) {
		handler := NewActionHandler(nil, 10*1024*1024, nil, security_dto.RateLimitValues{}, false, nil, nil)

		require.NotNil(t, handler)
		assert.Empty(t, handler.registry)
		assert.Equal(t, int64(10*1024*1024), handler.maxBodyBytes)
	})

	t.Run("creates handler with custom max body bytes", func(t *testing.T) {
		handler := NewActionHandler(nil, 5*1024, nil, security_dto.RateLimitValues{}, false, nil, nil)

		assert.Equal(t, int64(5*1024), handler.maxBodyBytes)
	})
}

func TestActionHandler_Register(t *testing.T) {
	t.Run("adds action to registry", func(t *testing.T) {
		handler := NewActionHandler(nil, 1024, nil, security_dto.RateLimitValues{}, false, nil, nil)

		entry := ActionHandlerEntry{
			Name:   "user.create",
			Method: http.MethodPost,
			Create: func() any { return nil },
			Invoke: func(_ context.Context, action any, arguments map[string]any) (any, error) { return nil, nil },
		}

		handler.Register(entry)

		assert.Contains(t, handler.registry, "user.create")
		assert.Equal(t, "user.create", handler.registry["user.create"].Name)
	})

	t.Run("overwrites existing entry with same name", func(t *testing.T) {
		handler := NewActionHandler(nil, 1024, nil, security_dto.RateLimitValues{}, false, nil, nil)

		entry1 := ActionHandlerEntry{
			Name:   "test.action",
			Method: http.MethodGet,
		}
		entry2 := ActionHandlerEntry{
			Name:   "test.action",
			Method: http.MethodPost,
		}

		handler.Register(entry1)
		handler.Register(entry2)

		assert.Equal(t, http.MethodPost, handler.registry["test.action"].Method)
	})
}

func TestActionHandler_RegisterAll(t *testing.T) {
	t.Run("registers multiple actions", func(t *testing.T) {
		handler := NewActionHandler(nil, 1024, nil, security_dto.RateLimitValues{}, false, nil, nil)

		entries := map[string]ActionHandlerEntry{
			"action.one": {Method: http.MethodGet},
			"action.two": {Method: http.MethodPost},
		}

		handler.RegisterAll(entries)

		assert.Len(t, handler.registry, 2)
		assert.Contains(t, handler.registry, "action.one")
		assert.Contains(t, handler.registry, "action.two")
	})

	t.Run("sets name from map key", func(t *testing.T) {
		handler := NewActionHandler(nil, 1024, nil, security_dto.RateLimitValues{}, false, nil, nil)

		entries := map[string]ActionHandlerEntry{
			"correct.name": {Name: "wrong.name", Method: http.MethodGet},
		}

		handler.RegisterAll(entries)

		assert.Equal(t, "correct.name", handler.registry["correct.name"].Name)
	})

	t.Run("handles empty map", func(t *testing.T) {
		handler := NewActionHandler(nil, 1024, nil, security_dto.RateLimitValues{}, false, nil, nil)

		handler.RegisterAll(map[string]ActionHandlerEntry{})

		assert.Empty(t, handler.registry)
	})
}

func TestActionHandler_Mount(t *testing.T) {
	t.Run("registers routes for all actions", func(t *testing.T) {
		handler := NewActionHandler(nil, 1024, nil, security_dto.RateLimitValues{}, false, nil, nil)
		handler.Register(ActionHandlerEntry{
			Name:   "test.action",
			Method: http.MethodPost,
			Create: func() any { return &struct{}{} },
			Invoke: func(_ context.Context, action any, arguments map[string]any) (any, error) {
				return map[string]string{"status": "ok"}, nil
			},
		})

		r := chi.NewRouter()
		handler.Mount(r, "/_piko/actions")

		request := httptest.NewRequest(http.MethodPost, "/_piko/actions/test.action", strings.NewReader("{}"))
		request.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()

		r.ServeHTTP(recorder, request)

		assert.NotEqual(t, http.StatusNotFound, recorder.Code)
	})

	t.Run("registers batch endpoint", func(t *testing.T) {
		handler := NewActionHandler(nil, 1024, nil, security_dto.RateLimitValues{}, false, nil, nil)

		r := chi.NewRouter()
		handler.Mount(r, "/_piko/actions")

		request := httptest.NewRequest(http.MethodPost, "/_piko/actions/_batch", strings.NewReader(`{"actions":[]}`))
		request.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()

		r.ServeHTTP(recorder, request)

		assert.Equal(t, http.StatusOK, recorder.Code)
	})
}

func TestActionHandler_ShouldUseSSE(t *testing.T) {
	handler := NewActionHandler(nil, 1024, nil, security_dto.RateLimitValues{}, false, nil, nil)

	testCases := []struct {
		name     string
		accept   string
		hasSSE   bool
		expected bool
	}{
		{
			name:     "returns false when action does not support SSE",
			hasSSE:   false,
			accept:   "text/event-stream",
			expected: false,
		},
		{
			name:     "returns false when accept header is not SSE",
			hasSSE:   true,
			accept:   "application/json",
			expected: false,
		},
		{
			name:     "returns true when action supports SSE and accept is event-stream",
			hasSSE:   true,
			accept:   "text/event-stream",
			expected: true,
		},
		{
			name:     "returns false when accept header is empty",
			hasSSE:   true,
			accept:   "",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			entry := ActionHandlerEntry{HasSSE: tc.hasSSE}
			request := httptest.NewRequest(http.MethodGet, "/", nil)
			request.Header.Set("Accept", tc.accept)

			result := handler.shouldUseSSE(request, entry)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestActionHandler_ParseRequestBody(t *testing.T) {
	handler := NewActionHandler(nil, 1024*1024, nil, security_dto.RateLimitValues{}, false, nil, nil)

	t.Run("returns empty map for GET request", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/", nil)

		arguments, err := handler.parseRequestBody(request)

		require.NoError(t, err)
		assert.Empty(t, arguments)
	})

	t.Run("returns empty map for zero content length", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodPost, "/", nil)
		request.ContentLength = 0

		arguments, err := handler.parseRequestBody(request)

		require.NoError(t, err)
		assert.Empty(t, arguments)
	})

	t.Run("parses JSON body", func(t *testing.T) {
		body := `{"name": "test", "count": 42}`
		request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")

		arguments, err := handler.parseRequestBody(request)

		require.NoError(t, err)
		assert.Equal(t, "test", arguments["name"])
		assert.Equal(t, float64(42), arguments["count"])
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		body := `{invalid json}`
		request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")

		_, err := handler.parseRequestBody(request)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "decoding request body")
	})
}

func TestActionHandler_ParseMultipartBody(t *testing.T) {
	handler := NewActionHandler(nil, 1024*1024, nil, security_dto.RateLimitValues{}, false, nil, nil)

	t.Run("parses form values", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		_ = writer.WriteField("name", "test-value")
		_ = writer.WriteField("email", "test@example.com")
		_ = writer.Close()

		request := httptest.NewRequest(http.MethodPost, "/", body)
		request.Header.Set("Content-Type", writer.FormDataContentType())

		arguments, err := handler.parseMultipartBody(request)

		require.NoError(t, err)
		assert.Equal(t, "test-value", arguments["name"])
		assert.Equal(t, "test@example.com", arguments["email"])
	})

	t.Run("handles multiple values for same field", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		_ = writer.WriteField("tags", "tag1")
		_ = writer.WriteField("tags", "tag2")
		_ = writer.WriteField("tags", "tag3")
		_ = writer.Close()

		request := httptest.NewRequest(http.MethodPost, "/", body)
		request.Header.Set("Content-Type", writer.FormDataContentType())

		arguments, err := handler.parseMultipartBody(request)

		require.NoError(t, err)
		tags, ok := arguments["tags"].([]string)
		require.True(t, ok)
		assert.Equal(t, []string{"tag1", "tag2", "tag3"}, tags)
	})

	t.Run("parses file uploads", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		fileWriter, _ := writer.CreateFormFile("document", "test.txt")
		_, _ = fileWriter.Write([]byte("file content"))
		_ = writer.Close()

		request := httptest.NewRequest(http.MethodPost, "/", body)
		request.Header.Set("Content-Type", writer.FormDataContentType())

		arguments, err := handler.parseMultipartBody(request)

		require.NoError(t, err)
		assert.Contains(t, arguments, "document")
	})
}

func TestActionHandler_ParseRawBody(t *testing.T) {
	handler := NewActionHandler(nil, 1024*1024, nil, security_dto.RateLimitValues{}, false, nil, nil)

	t.Run("reads entire body as raw bytes", func(t *testing.T) {
		content := "raw binary content here"
		request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(content))
		request.Header.Set("Content-Type", "application/octet-stream")

		arguments := make(map[string]any)
		err := handler.parseRawBody(request, arguments)

		require.NoError(t, err)
		assert.Contains(t, arguments, "_rawBody")

		rawBody, ok := arguments["_rawBody"].(daemon_dto.RawBody)
		require.True(t, ok)
		assert.Equal(t, "application/octet-stream", rawBody.ContentType)
		assert.Equal(t, []byte(content), rawBody.Bytes())
	})
}

func TestActionHandler_BuildFullResponse(t *testing.T) {
	handler := NewActionHandler(nil, 1024, nil, security_dto.RateLimitValues{}, false, nil, nil)

	t.Run("returns raw result when no helpers", func(t *testing.T) {
		result := map[string]string{"key": "value"}

		response := handler.buildFullResponse(&struct{}{}, result)

		assert.Equal(t, result, response)
	})

	t.Run("wraps result with helpers when present", func(t *testing.T) {
		result := map[string]string{"key": "value"}
		action := &mockActionWithResponse{
			response: daemon_dto.NewResponseWriter(),
		}
		action.response.AddHelper("updateCounter", map[string]any{"value": 1})

		response := handler.buildFullResponse(action, result)

		fullResp, ok := response.(daemon_dto.ActionFullResponse)
		require.True(t, ok)
		assert.Equal(t, result, fullResp.Data)
		assert.Len(t, fullResp.Helpers, 1)
	})
}

func TestActionHandler_BuildBatchErrorResult(t *testing.T) {
	handler := NewActionHandler(nil, 1024, nil, security_dto.RateLimitValues{}, false, nil, nil)

	t.Run("builds result from ActionError", func(t *testing.T) {
		err := &daemon_dto.ValidationError{
			Fields: map[string]string{"name": "required"},
		}

		result := handler.buildBatchErrorResult("test.action", err, false)

		assert.Equal(t, "test.action", result.Name)
		assert.Equal(t, http.StatusUnprocessableEntity, result.Status)
		assert.Equal(t, "VALIDATION_FAILED", result.Code)
		assert.Contains(t, result.Error, "validation failed")
	})

	t.Run("builds generic error for non-ActionError", func(t *testing.T) {
		err := errors.New("something went wrong")

		result := handler.buildBatchErrorResult("test.action", err, false)

		assert.Equal(t, "test.action", result.Name)
		assert.Equal(t, http.StatusInternalServerError, result.Status)
		assert.Equal(t, "INTERNAL_ERROR", result.Code)
		assert.Equal(t, "An internal error occurred", result.Error)
	})
}

func TestActionHandler_WriteJSON(t *testing.T) {
	handler := NewActionHandler(nil, 1024, nil, security_dto.RateLimitValues{}, false, nil, nil)

	t.Run("writes JSON with correct content type", func(t *testing.T) {
		recorder := httptest.NewRecorder()

		handler.writeJSON(recorder, http.StatusOK, map[string]string{"status": "ok"})

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
		assert.Contains(t, recorder.Body.String(), `"status":"ok"`)
	})

	t.Run("handles nil data", func(t *testing.T) {
		recorder := httptest.NewRecorder()

		handler.writeJSON(recorder, http.StatusNoContent, nil)

		assert.Equal(t, http.StatusNoContent, recorder.Code)
		assert.Empty(t, recorder.Body.String())
	})

	t.Run("writes custom status code", func(t *testing.T) {
		recorder := httptest.NewRecorder()

		handler.writeJSON(recorder, http.StatusCreated, map[string]int{"id": 123})

		assert.Equal(t, http.StatusCreated, recorder.Code)
	})
}

func TestActionHandler_WriteError(t *testing.T) {
	handler := NewActionHandler(nil, 1024, nil, security_dto.RateLimitValues{}, false, nil, nil)

	t.Run("writes error response", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		err := errors.New("database connection failed")

		handler.writeError(recorder, http.StatusInternalServerError, "Internal Server Error", err)

		assert.Equal(t, http.StatusInternalServerError, recorder.Code)
		assert.Contains(t, recorder.Body.String(), "Internal Server Error")
		assert.Contains(t, recorder.Body.String(), "database connection failed")
	})
}

func TestActionHandler_MountRegistersGETForSSE(t *testing.T) {
	handler := NewActionHandler(nil, 1024*1024, nil, security_dto.RateLimitValues{}, false, nil, nil)
	handler.Register(ActionHandlerEntry{
		Name:   "stream.Events",
		Method: "POST",
		HasSSE: true,
		Create: func() any { return &struct{}{} },
		Invoke: func(_ context.Context, action any, arguments map[string]any) (any, error) { return nil, nil },
	})

	r := chi.NewRouter()
	handler.Mount(r, "/_piko/actions")

	request := httptest.NewRequest(http.MethodGet, "/_piko/actions/stream.Events", nil)
	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, request)
	assert.NotEqual(t, http.StatusMethodNotAllowed, recorder.Code)
}

func TestActionHandler_HandleBatch(t *testing.T) {
	t.Run("returns empty results for empty batch", func(t *testing.T) {
		handler := NewActionHandler(nil, 1024*1024, nil, security_dto.RateLimitValues{}, false, nil, nil)
		r := chi.NewRouter()
		handler.Mount(r, "/_piko/actions")

		body := `{"actions":[]}`
		request := httptest.NewRequest(http.MethodPost, "/_piko/actions/_batch", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()

		r.ServeHTTP(recorder, request)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Contains(t, recorder.Body.String(), `"success":true`)
		assert.Contains(t, recorder.Body.String(), `"results":[]`)
	})

	t.Run("executes multiple actions", func(t *testing.T) {
		handler := NewActionHandler(nil, 1024*1024, nil, security_dto.RateLimitValues{}, false, nil, nil)
		handler.Register(ActionHandlerEntry{
			Name:   "test.add",
			Method: http.MethodPost,
			Create: func() any { return &struct{}{} },
			Invoke: func(_ context.Context, action any, arguments map[string]any) (any, error) {
				return map[string]int{"sum": 42}, nil
			},
		})

		r := chi.NewRouter()
		handler.Mount(r, "/_piko/actions")

		body := `{"actions":[{"name":"test.add","args":{"a":1,"b":2}}]}`
		request := httptest.NewRequest(http.MethodPost, "/_piko/actions/_batch", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()

		r.ServeHTTP(recorder, request)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Contains(t, recorder.Body.String(), `"success":true`)
	})

	t.Run("reports not found for unknown action", func(t *testing.T) {
		handler := NewActionHandler(nil, 1024*1024, nil, security_dto.RateLimitValues{}, false, nil, nil)
		r := chi.NewRouter()
		handler.Mount(r, "/_piko/actions")

		body := `{"actions":[{"name":"unknown.action"}]}`
		request := httptest.NewRequest(http.MethodPost, "/_piko/actions/_batch", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()

		r.ServeHTTP(recorder, request)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Contains(t, recorder.Body.String(), `"success":false`)
		assert.Contains(t, recorder.Body.String(), "NOT_FOUND")
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		handler := NewActionHandler(nil, 1024*1024, nil, security_dto.RateLimitValues{}, false, nil, nil)
		r := chi.NewRouter()
		handler.Mount(r, "/_piko/actions")

		body := `{invalid}`
		request := httptest.NewRequest(http.MethodPost, "/_piko/actions/_batch", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()

		r.ServeHTTP(recorder, request)

		assert.Equal(t, http.StatusBadRequest, recorder.Code)
	})
}

func TestActionHandler_HandleActionError(t *testing.T) {
	handler := NewActionHandler(nil, 1024, nil, security_dto.RateLimitValues{}, false, nil, nil)

	t.Run("writes ValidationError with field errors", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/action", nil)
		err := &daemon_dto.ValidationError{
			Fields: map[string]string{
				"email": "must be a valid email",
				"name":  "is required",
			},
		}

		handler.handleActionError(recorder, request, &struct{}{}, err)

		assert.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
		body := recorder.Body.String()
		assert.Contains(t, body, "VALIDATION_FAILED")
		assert.Contains(t, body, "must be a valid email")
	})

	t.Run("writes generic error for non-ActionError", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/action", nil)
		err := errors.New("unexpected error")

		handler.handleActionError(recorder, request, &struct{}{}, err)

		assert.Equal(t, http.StatusInternalServerError, recorder.Code)
		body := recorder.Body.String()
		assert.Contains(t, body, "INTERNAL_ERROR")
		assert.Contains(t, body, "An internal error occurred")
	})

	t.Run("includes helpers in error response when present", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/action", nil)
		action := &mockActionWithResponse{
			response: daemon_dto.NewResponseWriter(),
		}
		action.response.AddHelper("showError", map[string]string{"message": "error"})

		err := errors.New("some error")
		handler.handleActionError(recorder, request, action, err)

		body := recorder.Body.String()
		assert.Contains(t, body, "_helpers")
		assert.Contains(t, body, "showError")
	})
}

func TestActionHandler_CreateHandler(t *testing.T) {
	t.Run("returns handler that processes requests", func(t *testing.T) {
		handler := NewActionHandler(nil, 1024*1024, nil, security_dto.RateLimitValues{}, false, nil, nil)
		entry := ActionHandlerEntry{
			Name:   "test.action",
			Method: http.MethodPost,
			Create: func() any { return &struct{}{} },
			Invoke: func(_ context.Context, action any, arguments map[string]any) (any, error) {
				return map[string]string{"result": "success"}, nil
			},
		}

		httpHandler := handler.createHandler(entry)

		request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{}"))
		request.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()

		httpHandler.ServeHTTP(recorder, request)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Contains(t, recorder.Body.String(), "success")
	})
}

type mockActionWithResponse struct {
	response *daemon_dto.ResponseWriter
	ctx      context.Context
	request  *daemon_dto.RequestMetadata
}

func (m *mockActionWithResponse) Response() *daemon_dto.ResponseWriter {
	return m.response
}

func (m *mockActionWithResponse) SetContext(ctx context.Context) {
	m.ctx = ctx
}

func (m *mockActionWithResponse) SetRequest(request *daemon_dto.RequestMetadata) {
	m.request = request
}

func (m *mockActionWithResponse) SetResponse(response *daemon_dto.ResponseWriter) {
	m.response = response
}

func TestActionHandler_ValidateCSRF(t *testing.T) {
	t.Run("skips validation when csrfService is nil", func(t *testing.T) {
		handler := NewActionHandler(nil, 1024, nil, security_dto.RateLimitValues{}, false, nil, nil)
		arguments := map[string]any{
			"_csrf_ephemeral_token": "some-token",
			"name":                  "test",
		}
		request := httptest.NewRequest(http.MethodPost, "/", nil)
		request.Header.Set("X-CSRF-Action-Token", "action-token")

		err := handler.validateCSRF(request, arguments)

		assert.NoError(t, err)

		assert.NotContains(t, arguments, "_csrf_ephemeral_token")
		assert.Contains(t, arguments, "name")
	})

	t.Run("skips validation when both tokens are empty", func(t *testing.T) {
		csrf := &configurableCSRFService{validateResult: true}
		handler := NewActionHandler(csrf, 1024, nil, security_dto.RateLimitValues{}, false, nil, nil)
		arguments := map[string]any{"name": "test"}
		request := httptest.NewRequest(http.MethodPost, "/", nil)

		err := handler.validateCSRF(request, arguments)

		assert.NoError(t, err)
	})

	t.Run("validates when tokens are present and valid", func(t *testing.T) {
		csrf := &configurableCSRFService{validateResult: true}
		handler := NewActionHandler(csrf, 1024, nil, security_dto.RateLimitValues{}, false, nil, nil)
		arguments := map[string]any{
			"_csrf_ephemeral_token": "ephemeral-123",
			"data":                  "value",
		}
		request := httptest.NewRequest(http.MethodPost, "/", nil)
		request.Header.Set("X-CSRF-Action-Token", "action-token-456")

		err := handler.validateCSRF(request, arguments)

		assert.NoError(t, err)
		assert.NotContains(t, arguments, "_csrf_ephemeral_token")
	})

	t.Run("returns error when validation fails with csrf_expired", func(t *testing.T) {
		csrf := &configurableCSRFService{
			validateResult: false,
			validateErr:    &security_domain.CSRFValidationError{Code: "csrf_expired", Message: "token expired"},
		}
		handler := NewActionHandler(csrf, 1024, nil, security_dto.RateLimitValues{}, false, nil, nil)
		arguments := map[string]any{"_csrf_ephemeral_token": "ephemeral"}
		request := httptest.NewRequest(http.MethodPost, "/", nil)
		request.Header.Set("X-CSRF-Action-Token", "action-token")

		err := handler.validateCSRF(request, arguments)

		require.Error(t, err)
		csrfErr, ok := errors.AsType[*security_domain.CSRFValidationError](err)
		require.True(t, ok)
		assert.Equal(t, "csrf_expired", csrfErr.Code)
	})

	t.Run("returns error when validation fails with csrf_invalid", func(t *testing.T) {
		csrf := &configurableCSRFService{
			validateResult: false,
			validateErr:    &security_domain.CSRFValidationError{Code: "csrf_invalid", Message: "bad token"},
		}
		handler := NewActionHandler(csrf, 1024, nil, security_dto.RateLimitValues{}, false, nil, nil)
		arguments := map[string]any{"_csrf_ephemeral_token": "ephemeral"}
		request := httptest.NewRequest(http.MethodPost, "/", nil)
		request.Header.Set("X-CSRF-Action-Token", "action-token")

		err := handler.validateCSRF(request, arguments)

		require.Error(t, err)
		csrfErr, ok := errors.AsType[*security_domain.CSRFValidationError](err)
		require.True(t, ok)
		assert.Equal(t, "csrf_invalid", csrfErr.Code)
	})

	t.Run("reads ephemeral token from query param for GET requests", func(t *testing.T) {
		csrf := &configurableCSRFService{validateResult: true}
		handler := NewActionHandler(csrf, 1024, nil, security_dto.RateLimitValues{}, false, nil, nil)
		arguments := map[string]any{}
		request := httptest.NewRequest(http.MethodGet, "/?_csrf_ephemeral_token=from-query", nil)
		request.Header.Set("X-CSRF-Action-Token", "action-token")

		err := handler.validateCSRF(request, arguments)

		assert.NoError(t, err)
	})

	t.Run("strips ephemeral token from arguments even when validation fails", func(t *testing.T) {
		csrf := &configurableCSRFService{
			validateResult: false,
			validateErr:    &security_domain.CSRFValidationError{Code: "csrf_invalid", Message: "bad"},
		}
		handler := NewActionHandler(csrf, 1024, nil, security_dto.RateLimitValues{}, false, nil, nil)
		arguments := map[string]any{
			"_csrf_ephemeral_token": "ephemeral",
			"name":                  "test",
		}
		request := httptest.NewRequest(http.MethodPost, "/", nil)
		request.Header.Set("X-CSRF-Action-Token", "action-token")

		_ = handler.validateCSRF(request, arguments)

		assert.NotContains(t, arguments, "_csrf_ephemeral_token")
		assert.Contains(t, arguments, "name")
	})

	t.Run("returns error when only action token present but no ephemeral", func(t *testing.T) {
		csrf := &configurableCSRFService{
			validateResult: false,
			validateErr:    &security_domain.CSRFValidationError{Code: "csrf_invalid", Message: "empty"},
		}
		handler := NewActionHandler(csrf, 1024, nil, security_dto.RateLimitValues{}, false, nil, nil)
		arguments := map[string]any{}
		request := httptest.NewRequest(http.MethodPost, "/", nil)
		request.Header.Set("X-CSRF-Action-Token", "action-token")

		err := handler.validateCSRF(request, arguments)

		require.Error(t, err)
	})
}

func TestActionHandler_SecFetchSiteEnforcement(t *testing.T) {
	t.Run("rejects browser request when enforcement enabled and tokens empty", func(t *testing.T) {
		csrf := &configurableCSRFService{validateResult: true}
		handler := NewActionHandler(csrf, 1024, nil, security_dto.RateLimitValues{}, true, nil, nil)
		arguments := map[string]any{}
		request := httptest.NewRequest(http.MethodPost, "/", nil)
		request.Header.Set("Sec-Fetch-Site", "same-origin")

		err := handler.validateCSRF(request, arguments)

		require.Error(t, err)
		csrfErr, ok := errors.AsType[*security_domain.CSRFValidationError](err)
		require.True(t, ok)
		assert.Equal(t, security_domain.CSRFErrorCodeMissing, csrfErr.Code)
	})

	t.Run("allows server-to-server request when enforcement enabled and tokens empty", func(t *testing.T) {
		csrf := &configurableCSRFService{validateResult: true}
		handler := NewActionHandler(csrf, 1024, nil, security_dto.RateLimitValues{}, true, nil, nil)
		arguments := map[string]any{}
		request := httptest.NewRequest(http.MethodPost, "/", nil)

		err := handler.validateCSRF(request, arguments)

		assert.NoError(t, err)
	})

	t.Run("allows browser request when enforcement enabled and tokens provided", func(t *testing.T) {
		csrf := &configurableCSRFService{validateResult: true}
		handler := NewActionHandler(csrf, 1024, nil, security_dto.RateLimitValues{}, true, nil, nil)
		arguments := map[string]any{"_csrf_ephemeral_token": "ephemeral-123"}
		request := httptest.NewRequest(http.MethodPost, "/", nil)
		request.Header.Set("Sec-Fetch-Site", "same-origin")
		request.Header.Set("X-CSRF-Action-Token", "action-token-456")

		err := handler.validateCSRF(request, arguments)

		assert.NoError(t, err)
	})

	t.Run("allows browser request without tokens when enforcement disabled", func(t *testing.T) {
		csrf := &configurableCSRFService{validateResult: true}
		handler := NewActionHandler(csrf, 1024, nil, security_dto.RateLimitValues{}, false, nil, nil)
		arguments := map[string]any{}
		request := httptest.NewRequest(http.MethodPost, "/", nil)
		request.Header.Set("Sec-Fetch-Site", "cross-site")

		err := handler.validateCSRF(request, arguments)

		assert.NoError(t, err)
	})
}

func TestActionHandler_WriteCSRFError(t *testing.T) {
	handler := NewActionHandler(nil, 1024, nil, security_dto.RateLimitValues{}, false, nil, nil)

	t.Run("writes csrf_expired error response", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		csrfErr := &security_domain.CSRFValidationError{Code: "csrf_expired", Message: "CSRF token has expired"}

		handler.writeCSRFError(recorder, csrfErr)

		assert.Equal(t, http.StatusForbidden, recorder.Code)
		body := recorder.Body.String()
		assert.Contains(t, body, "csrf_expired")
		assert.Contains(t, body, "CSRF token has expired")
	})

	t.Run("writes csrf_invalid error response", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		csrfErr := &security_domain.CSRFValidationError{Code: "csrf_invalid", Message: "CSRF token is invalid"}

		handler.writeCSRFError(recorder, csrfErr)

		assert.Equal(t, http.StatusForbidden, recorder.Code)
		body := recorder.Body.String()
		assert.Contains(t, body, "csrf_invalid")
	})

	t.Run("writes generic error for non-CSRFValidationError", func(t *testing.T) {
		recorder := httptest.NewRecorder()

		handler.writeCSRFError(recorder, errors.New("something else"))

		assert.Equal(t, http.StatusForbidden, recorder.Code)
		body := recorder.Body.String()
		assert.Contains(t, body, "csrf_invalid")
		assert.Contains(t, body, "CSRF validation failed")
	})
}

func TestActionHandler_HandleHTTP_CSRF(t *testing.T) {
	t.Run("action proceeds when CSRF validation passes", func(t *testing.T) {
		csrf := &configurableCSRFService{validateResult: true}
		handler := NewActionHandler(csrf, 1024*1024, nil, security_dto.RateLimitValues{}, false, nil, nil)
		handler.Register(ActionHandlerEntry{
			Name:   "test.action",
			Method: http.MethodPost,
			Create: func() any { return &struct{}{} },
			Invoke: func(_ context.Context, action any, arguments map[string]any) (any, error) {
				return map[string]string{"ok": "true"}, nil
			},
		})

		r := chi.NewRouter()
		handler.Mount(r, "/_piko/actions")

		body := `{"name":"test","_csrf_ephemeral_token":"eph-123"}`
		request := httptest.NewRequest(http.MethodPost, "/_piko/actions/test.action", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("X-CSRF-Action-Token", "act-456")
		recorder := httptest.NewRecorder()

		r.ServeHTTP(recorder, request)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Contains(t, recorder.Body.String(), "true")
	})

	t.Run("returns 403 when CSRF validation fails", func(t *testing.T) {
		csrf := &configurableCSRFService{
			validateResult: false,
			validateErr:    &security_domain.CSRFValidationError{Code: "csrf_expired", Message: "expired"},
		}
		handler := NewActionHandler(csrf, 1024*1024, nil, security_dto.RateLimitValues{}, false, nil, nil)
		handler.Register(ActionHandlerEntry{
			Name:   "test.action",
			Method: http.MethodPost,
			Create: func() any { return &struct{}{} },
			Invoke: func(_ context.Context, action any, arguments map[string]any) (any, error) {
				t.Fatal("Invoke should not be called when CSRF fails")
				return nil, nil
			},
		})

		r := chi.NewRouter()
		handler.Mount(r, "/_piko/actions")

		body := `{"_csrf_ephemeral_token":"bad-token"}`
		request := httptest.NewRequest(http.MethodPost, "/_piko/actions/test.action", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("X-CSRF-Action-Token", "bad-action")
		recorder := httptest.NewRecorder()

		r.ServeHTTP(recorder, request)

		assert.Equal(t, http.StatusForbidden, recorder.Code)
		assert.Contains(t, recorder.Body.String(), "csrf_expired")
	})

	t.Run("strips ephemeral token from arguments before invoking action", func(t *testing.T) {
		csrf := &configurableCSRFService{validateResult: true}
		handler := NewActionHandler(csrf, 1024*1024, nil, security_dto.RateLimitValues{}, false, nil, nil)

		var capturedArgs map[string]any
		handler.Register(ActionHandlerEntry{
			Name:   "test.action",
			Method: http.MethodPost,
			Create: func() any { return &struct{}{} },
			Invoke: func(_ context.Context, action any, arguments map[string]any) (any, error) {
				capturedArgs = arguments
				return map[string]string{"ok": "true"}, nil
			},
		})

		r := chi.NewRouter()
		handler.Mount(r, "/_piko/actions")

		body := `{"name":"test","_csrf_ephemeral_token":"eph-token"}`
		request := httptest.NewRequest(http.MethodPost, "/_piko/actions/test.action", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("X-CSRF-Action-Token", "act-token")
		recorder := httptest.NewRecorder()

		r.ServeHTTP(recorder, request)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.NotContains(t, capturedArgs, "_csrf_ephemeral_token")
		assert.Equal(t, "test", capturedArgs["name"])
	})

	t.Run("existing tests still pass with nil csrfService", func(t *testing.T) {
		handler := NewActionHandler(nil, 1024*1024, nil, security_dto.RateLimitValues{}, false, nil, nil)
		handler.Register(ActionHandlerEntry{
			Name:   "test.action",
			Method: http.MethodPost,
			Create: func() any { return &struct{}{} },
			Invoke: func(_ context.Context, action any, arguments map[string]any) (any, error) {
				return map[string]string{"result": "ok"}, nil
			},
		})

		r := chi.NewRouter()
		handler.Mount(r, "/_piko/actions")

		body := `{"data":"value"}`
		request := httptest.NewRequest(http.MethodPost, "/_piko/actions/test.action", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()

		r.ServeHTTP(recorder, request)

		assert.Equal(t, http.StatusOK, recorder.Code)
	})
}

func TestActionHandler_HandleBatch_CSRF(t *testing.T) {
	t.Run("batch proceeds when CSRF validation passes", func(t *testing.T) {
		csrf := &configurableCSRFService{validateResult: true}
		handler := NewActionHandler(csrf, 1024*1024, nil, security_dto.RateLimitValues{}, false, nil, nil)
		handler.Register(ActionHandlerEntry{
			Name:   "test.action",
			Method: http.MethodPost,
			Create: func() any { return &struct{}{} },
			Invoke: func(_ context.Context, action any, arguments map[string]any) (any, error) {
				return map[string]string{"ok": "true"}, nil
			},
		})

		r := chi.NewRouter()
		handler.Mount(r, "/_piko/actions")

		body := `{"actions":[{"name":"test.action","args":{"a":1}}],"_csrf_ephemeral_token":"eph-123"}`
		request := httptest.NewRequest(http.MethodPost, "/_piko/actions/_batch", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("X-CSRF-Action-Token", "act-456")
		recorder := httptest.NewRecorder()

		r.ServeHTTP(recorder, request)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Contains(t, recorder.Body.String(), `"success":true`)
	})

	t.Run("batch returns 403 when CSRF validation fails", func(t *testing.T) {
		csrf := &configurableCSRFService{
			validateResult: false,
			validateErr:    &security_domain.CSRFValidationError{Code: "csrf_invalid", Message: "invalid"},
		}
		handler := NewActionHandler(csrf, 1024*1024, nil, security_dto.RateLimitValues{}, false, nil, nil)

		r := chi.NewRouter()
		handler.Mount(r, "/_piko/actions")

		body := `{"actions":[{"name":"test.action"}],"_csrf_ephemeral_token":"bad"}`
		request := httptest.NewRequest(http.MethodPost, "/_piko/actions/_batch", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("X-CSRF-Action-Token", "bad-action")
		recorder := httptest.NewRecorder()

		r.ServeHTTP(recorder, request)

		assert.Equal(t, http.StatusForbidden, recorder.Code)
		assert.Contains(t, recorder.Body.String(), "csrf_invalid")
	})
}

func TestBuildCacheKey_BasicKey(t *testing.T) {
	t.Parallel()

	handler := NewActionHandler(nil, 1024, nil, security_dto.RateLimitValues{}, false, nil, nil)
	request := httptest.NewRequest(http.MethodPost, "/action", nil)
	arguments := map[string]any{"name": "test"}
	cc := &daemon_domain.CacheConfig{TTL: time.Minute}

	key := handler.buildCacheKey(request, arguments, "user.get", cc)

	assert.Contains(t, key, "user.get:")
	assert.Contains(t, key, `"name":"test"`)
}

func TestBuildCacheKey_EmptyArgs(t *testing.T) {
	t.Parallel()

	handler := NewActionHandler(nil, 1024, nil, security_dto.RateLimitValues{}, false, nil, nil)
	request := httptest.NewRequest(http.MethodPost, "/action", nil)
	arguments := map[string]any{}
	cc := &daemon_domain.CacheConfig{TTL: time.Minute}

	key := handler.buildCacheKey(request, arguments, "user.get", cc)

	assert.Contains(t, key, "user.get:{}")
}

func TestBuildCacheKey_WithVaryHeaders(t *testing.T) {
	t.Parallel()

	handler := NewActionHandler(nil, 1024, nil, security_dto.RateLimitValues{}, false, nil, nil)
	request := httptest.NewRequest(http.MethodPost, "/action", nil)
	request.Header.Set("Accept-Language", "en-US")
	request.Header.Set("Accept", "application/json")

	arguments := map[string]any{}
	cc := &daemon_domain.CacheConfig{
		TTL:         time.Minute,
		VaryHeaders: []string{"Accept-Language", "Accept"},
	}

	key := handler.buildCacheKey(request, arguments, "user.get", cc)

	assert.Contains(t, key, "Accept-Language=en-US")
	assert.Contains(t, key, "Accept=application/json")
}

func TestBuildCacheKey_VaryHeader_EmptyValue(t *testing.T) {
	t.Parallel()

	handler := NewActionHandler(nil, 1024, nil, security_dto.RateLimitValues{}, false, nil, nil)
	request := httptest.NewRequest(http.MethodPost, "/action", nil)

	arguments := map[string]any{}
	cc := &daemon_domain.CacheConfig{
		TTL:         time.Minute,
		VaryHeaders: []string{"X-Custom-Header"},
	}

	key := handler.buildCacheKey(request, arguments, "user.get", cc)

	assert.Contains(t, key, "X-Custom-Header=")
}

func TestBuildCacheKey_DifferentActions_DifferentKeys(t *testing.T) {
	t.Parallel()

	handler := NewActionHandler(nil, 1024, nil, security_dto.RateLimitValues{}, false, nil, nil)
	request := httptest.NewRequest(http.MethodPost, "/action", nil)
	arguments := map[string]any{"id": 1}
	cc := &daemon_domain.CacheConfig{TTL: time.Minute}

	key1 := handler.buildCacheKey(request, arguments, "user.get", cc)
	key2 := handler.buildCacheKey(request, arguments, "post.get", cc)

	assert.NotEqual(t, key1, key2)
}

func TestBuildCacheKey_DifferentArgs_DifferentKeys(t *testing.T) {
	t.Parallel()

	handler := NewActionHandler(nil, 1024, nil, security_dto.RateLimitValues{}, false, nil, nil)
	request := httptest.NewRequest(http.MethodPost, "/action", nil)
	cc := &daemon_domain.CacheConfig{TTL: time.Minute}

	key1 := handler.buildCacheKey(request, map[string]any{"id": 1}, "user.get", cc)
	key2 := handler.buildCacheKey(request, map[string]any{"id": 2}, "user.get", cc)

	assert.NotEqual(t, key1, key2)
}

func TestRecordSlowAction_ZeroThreshold_NoWarning(t *testing.T) {
	t.Parallel()

	handler := NewActionHandler(nil, 1024, nil, security_dto.RateLimitValues{}, false, nil, nil)

	handler.recordSlowAction(context.Background(), "test.action", time.Now().Add(-time.Hour), 0)
}

func TestRecordSlowAction_NegativeThreshold_NoWarning(t *testing.T) {
	t.Parallel()

	handler := NewActionHandler(nil, 1024, nil, security_dto.RateLimitValues{}, false, nil, nil)

	handler.recordSlowAction(context.Background(), "test.action", time.Now().Add(-time.Hour), -time.Second)
}

func TestRecordSlowAction_UnderThreshold_NoWarning(t *testing.T) {
	t.Parallel()

	handler := NewActionHandler(nil, 1024, nil, security_dto.RateLimitValues{}, false, nil, nil)

	handler.recordSlowAction(context.Background(), "test.action", time.Now(), 5*time.Second)
}

func TestRecordSlowAction_OverThreshold_Warning(t *testing.T) {
	t.Parallel()

	handler := NewActionHandler(nil, 1024, nil, security_dto.RateLimitValues{}, false, nil, nil)

	handler.recordSlowAction(context.Background(), "test.action", time.Now().Add(-10*time.Second), time.Millisecond)
}

func TestCheckRateLimit_NilMiddleware_ReturnsTrue(t *testing.T) {
	t.Parallel()

	handler := NewActionHandler(nil, 1024, nil, security_dto.RateLimitValues{}, false, nil, nil)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/", nil)

	result := handler.checkRateLimit(context.Background(), recorder, request, struct{}{}, ActionHandlerEntry{Name: "test"})

	assert.True(t, result)
}

func TestCheckRateLimit_ActionNotRateLimitable_ReturnsTrue(t *testing.T) {
	t.Parallel()

	handler := NewActionHandler(nil, 1024, &security_domain.MockRateLimitService{
		CheckLimitFunc: func(_ string, _ int, _ time.Duration) (ratelimiter_dto.Result, error) {
			return ratelimiter_dto.Result{Allowed: true}, nil
		},
	}, security_dto.RateLimitValues{Enabled: true}, false, nil, nil)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/", nil)

	result := handler.checkRateLimit(context.Background(), recorder, request, struct{}{}, ActionHandlerEntry{Name: "test"})

	assert.True(t, result)
}

func TestCheckRateLimit_NilRateLimit_ReturnsTrue(t *testing.T) {
	t.Parallel()

	handler := NewActionHandler(nil, 1024, &security_domain.MockRateLimitService{
		CheckLimitFunc: func(_ string, _ int, _ time.Duration) (ratelimiter_dto.Result, error) {
			return ratelimiter_dto.Result{Allowed: true}, nil
		},
	}, security_dto.RateLimitValues{Enabled: true}, false, nil, nil)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/", nil)

	action := &mockRateLimitableAction{rateLimit: nil}

	result := handler.checkRateLimit(context.Background(), recorder, request, action, ActionHandlerEntry{Name: "test"})

	assert.True(t, result)
}

type mockRateLimitableAction struct {
	rateLimit *daemon_domain.RateLimit
}

func (m *mockRateLimitableAction) RateLimit() *daemon_domain.RateLimit {
	return m.rateLimit
}

type mockMetadataInjector struct {
	reqMeta  *daemon_dto.RequestMetadata
	response *daemon_dto.ResponseWriter
}

func (m *mockMetadataInjector) SetRequest(request *daemon_dto.RequestMetadata) {
	m.reqMeta = request
}

func (m *mockMetadataInjector) SetResponse(response *daemon_dto.ResponseWriter) {
	m.response = response
}

func TestInjectMetadata_SetsRequestMetadata(t *testing.T) {
	t.Parallel()

	handler := NewActionHandler(nil, 1024, nil, security_dto.RateLimitValues{}, false, nil, nil)
	action := &mockMetadataInjector{}
	request := httptest.NewRequest(http.MethodPost, "/test/path?q=1", nil)
	request.Header.Set("X-Custom", "value")

	handler.injectMetadata(request, action)

	require.NotNil(t, action.reqMeta)
	assert.Equal(t, http.MethodPost, action.reqMeta.Method)
	assert.Equal(t, "/test/path", action.reqMeta.Path)
	assert.Equal(t, "value", action.reqMeta.Headers.Get("X-Custom"))
	assert.Equal(t, "1", action.reqMeta.QueryParams.Get("q"))
}

func TestInjectMetadata_SetsResponseWriter(t *testing.T) {
	t.Parallel()

	handler := NewActionHandler(nil, 1024, nil, security_dto.RateLimitValues{}, false, nil, nil)
	action := &mockMetadataInjector{}
	request := httptest.NewRequest(http.MethodPost, "/", nil)

	handler.injectMetadata(request, action)

	assert.NotNil(t, action.response)
}

func TestInjectMetadata_NonInjectable_NoOp(t *testing.T) {
	t.Parallel()

	handler := NewActionHandler(nil, 1024, nil, security_dto.RateLimitValues{}, false, nil, nil)
	action := struct{}{}
	request := httptest.NewRequest(http.MethodPost, "/", nil)

	assert.NotPanics(t, func() {
		handler.injectMetadata(request, action)
	})
}

type mockResponseGetterAction struct {
	response *daemon_dto.ResponseWriter
}

func (m *mockResponseGetterAction) Response() *daemon_dto.ResponseWriter {
	return m.response
}

func TestApplyResponseMetadata_SetsCookies(t *testing.T) {
	t.Parallel()

	handler := NewActionHandler(nil, 1024, nil, security_dto.RateLimitValues{}, false, nil, nil)
	recorder := httptest.NewRecorder()

	response := daemon_dto.NewResponseWriter()
	response.SetCookie(&http.Cookie{Name: "session", Value: "abc123"})

	action := &mockResponseGetterAction{response: response}
	handler.applyResponseMetadata(recorder, action)

	cookies := recorder.Result().Cookies()
	require.Len(t, cookies, 1)
	assert.Equal(t, "session", cookies[0].Name)
	assert.Equal(t, "abc123", cookies[0].Value)
}

func TestApplyResponseMetadata_SetsHeaders(t *testing.T) {
	t.Parallel()

	handler := NewActionHandler(nil, 1024, nil, security_dto.RateLimitValues{}, false, nil, nil)
	recorder := httptest.NewRecorder()

	response := daemon_dto.NewResponseWriter()
	response.AddHeader("X-Custom", "value1")

	action := &mockResponseGetterAction{response: response}
	handler.applyResponseMetadata(recorder, action)

	assert.Equal(t, "value1", recorder.Header().Get("X-Custom"))
}

func TestApplyResponseMetadata_NilResponse_NoOp(t *testing.T) {
	t.Parallel()

	handler := NewActionHandler(nil, 1024, nil, security_dto.RateLimitValues{}, false, nil, nil)
	recorder := httptest.NewRecorder()

	action := &mockResponseGetterAction{response: nil}
	handler.applyResponseMetadata(recorder, action)

	assert.Empty(t, recorder.Result().Cookies())
}

func TestApplyResponseMetadata_NonResponseGetter_NoOp(t *testing.T) {
	t.Parallel()

	handler := NewActionHandler(nil, 1024, nil, security_dto.RateLimitValues{}, false, nil, nil)
	recorder := httptest.NewRecorder()

	action := struct{}{}
	assert.NotPanics(t, func() {
		handler.applyResponseMetadata(recorder, action)
	})
}

func TestExecuteBatchActions_AllSuccess(t *testing.T) {
	t.Parallel()

	handler := NewActionHandler(nil, 1024, nil, security_dto.RateLimitValues{}, false, nil, nil)
	handler.Register(ActionHandlerEntry{
		Name:   "test.action",
		Method: http.MethodPost,
		Create: func() any { return struct{}{} },
		Invoke: func(_ context.Context, _ any, _ map[string]any) (any, error) {
			return map[string]string{"ok": "true"}, nil
		},
	})

	request := httptest.NewRequest(http.MethodPost, "/", nil)
	actions := []daemon_dto.BatchActionItem{
		{Name: "test.action", Args: map[string]any{}},
	}

	results, allSuccess := handler.executeBatchActions(context.Background(), request, actions)

	assert.True(t, allSuccess)
	require.Len(t, results, 1)
	assert.Equal(t, http.StatusOK, results[0].Status)
	assert.Equal(t, "test.action", results[0].Name)
}

func TestExecuteBatchActions_UnknownAction_NotFound(t *testing.T) {
	t.Parallel()

	handler := NewActionHandler(nil, 1024, nil, security_dto.RateLimitValues{}, false, nil, nil)
	request := httptest.NewRequest(http.MethodPost, "/", nil)
	actions := []daemon_dto.BatchActionItem{
		{Name: "nonexistent.action"},
	}

	results, allSuccess := handler.executeBatchActions(context.Background(), request, actions)

	assert.False(t, allSuccess)
	require.Len(t, results, 1)
	assert.Equal(t, http.StatusNotFound, results[0].Status)
	assert.Equal(t, "NOT_FOUND", results[0].Code)
}

func TestExecuteBatchActions_MixedResults(t *testing.T) {
	t.Parallel()

	handler := NewActionHandler(nil, 1024, nil, security_dto.RateLimitValues{}, false, nil, nil)
	handler.Register(ActionHandlerEntry{
		Name:   "success.action",
		Method: http.MethodPost,
		Create: func() any { return struct{}{} },
		Invoke: func(_ context.Context, _ any, _ map[string]any) (any, error) { return "ok", nil },
	})
	handler.Register(ActionHandlerEntry{
		Name:   "fail.action",
		Method: http.MethodPost,
		Create: func() any { return struct{}{} },
		Invoke: func(_ context.Context, _ any, _ map[string]any) (any, error) {
			return nil, errors.New("something failed")
		},
	})

	request := httptest.NewRequest(http.MethodPost, "/", nil)
	actions := []daemon_dto.BatchActionItem{
		{Name: "success.action"},
		{Name: "fail.action"},
	}

	results, allSuccess := handler.executeBatchActions(context.Background(), request, actions)

	assert.False(t, allSuccess)
	require.Len(t, results, 2)
	assert.Equal(t, http.StatusOK, results[0].Status)
	assert.Equal(t, http.StatusInternalServerError, results[1].Status)
}

func TestExecuteSingleAction_NotFound(t *testing.T) {
	t.Parallel()

	handler := NewActionHandler(nil, 1024, nil, security_dto.RateLimitValues{}, false, nil, nil)
	request := httptest.NewRequest(http.MethodPost, "/", nil)

	result := handler.executeSingleAction(context.Background(), request, daemon_dto.BatchActionItem{Name: "missing"})

	assert.Equal(t, http.StatusNotFound, result.Status)
	assert.Contains(t, result.Error, "not found")
	assert.Equal(t, "NOT_FOUND", result.Code)
}

func TestExecuteSingleAction_Success(t *testing.T) {
	t.Parallel()

	handler := NewActionHandler(nil, 1024, nil, security_dto.RateLimitValues{}, false, nil, nil)
	handler.Register(ActionHandlerEntry{
		Name:   "test.action",
		Method: http.MethodPost,
		Create: func() any { return struct{}{} },
		Invoke: func(_ context.Context, _ any, _ map[string]any) (any, error) { return "result", nil },
	})

	request := httptest.NewRequest(http.MethodPost, "/", nil)
	result := handler.executeSingleAction(context.Background(), request, daemon_dto.BatchActionItem{
		Name: "test.action",
		Args: map[string]any{"key": "value"},
	})

	assert.Equal(t, http.StatusOK, result.Status)
	assert.Equal(t, "test.action", result.Name)
	assert.Equal(t, "result", result.Data)
}

func TestExecuteSingleAction_NilArgs_DefaultsToEmptyMap(t *testing.T) {
	t.Parallel()

	var receivedArgs map[string]any
	handler := NewActionHandler(nil, 1024, nil, security_dto.RateLimitValues{}, false, nil, nil)
	handler.Register(ActionHandlerEntry{
		Name:   "test.action",
		Method: http.MethodPost,
		Create: func() any { return struct{}{} },
		Invoke: func(_ context.Context, _ any, arguments map[string]any) (any, error) {
			receivedArgs = arguments
			return nil, nil
		},
	})

	request := httptest.NewRequest(http.MethodPost, "/", nil)
	handler.executeSingleAction(context.Background(), request, daemon_dto.BatchActionItem{
		Name: "test.action",
		Args: nil,
	})

	assert.NotNil(t, receivedArgs)
	assert.Empty(t, receivedArgs)
}

func TestExecuteSingleAction_InvokeError_ReturnsErrorResult(t *testing.T) {
	t.Parallel()

	handler := NewActionHandler(nil, 1024, nil, security_dto.RateLimitValues{}, false, nil, nil)
	handler.Register(ActionHandlerEntry{
		Name:   "test.action",
		Method: http.MethodPost,
		Create: func() any { return struct{}{} },
		Invoke: func(_ context.Context, _ any, _ map[string]any) (any, error) {
			return nil, errors.New("invoke failed")
		},
	})

	request := httptest.NewRequest(http.MethodPost, "/", nil)
	result := handler.executeSingleAction(context.Background(), request, daemon_dto.BatchActionItem{
		Name: "test.action",
	})

	assert.Equal(t, http.StatusInternalServerError, result.Status)
	assert.Equal(t, "INTERNAL_ERROR", result.Code)
}

func TestTrackBatchMetrics_DoesNotPanic(t *testing.T) {
	t.Parallel()

	handler := NewActionHandler(nil, 1024, nil, security_dto.RateLimitValues{}, false, nil, nil)
	request := httptest.NewRequest(http.MethodPost, "/_piko/actions/_batch", nil)

	assert.NotPanics(t, func() {
		handler.trackBatchMetrics(context.Background(), request)
	})
}

func TestHandleBatch_EmptyActions_ReturnsSuccess(t *testing.T) {
	t.Parallel()

	handler := NewActionHandler(nil, 1024*1024, nil, security_dto.RateLimitValues{}, false, nil, nil)
	r := chi.NewRouter()
	handler.Mount(r, "/_piko/actions")

	body := `{"actions":[]}`
	request := httptest.NewRequest(http.MethodPost, "/_piko/actions/_batch", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	r.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), `"success":true`)
}

func TestHandleBatch_InvalidJSON_ReturnsBadRequest(t *testing.T) {
	t.Parallel()

	handler := NewActionHandler(nil, 1024*1024, nil, security_dto.RateLimitValues{}, false, nil, nil)
	r := chi.NewRouter()
	handler.Mount(r, "/_piko/actions")

	body := `{invalid json`
	request := httptest.NewRequest(http.MethodPost, "/_piko/actions/_batch", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	r.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestHandleBatch_WithActions_ReturnsResults(t *testing.T) {
	t.Parallel()

	handler := NewActionHandler(nil, 1024*1024, nil, security_dto.RateLimitValues{}, false, nil, nil)
	handler.Register(ActionHandlerEntry{
		Name:   "test.action",
		Method: http.MethodPost,
		Create: func() any { return struct{}{} },
		Invoke: func(_ context.Context, _ any, _ map[string]any) (any, error) {
			return map[string]string{"message": "hello"}, nil
		},
	})

	r := chi.NewRouter()
	handler.Mount(r, "/_piko/actions")

	body := `{"actions":[{"name":"test.action","args":{}}]}`
	request := httptest.NewRequest(http.MethodPost, "/_piko/actions/_batch", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	r.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), `"success":true`)
	assert.Contains(t, recorder.Body.String(), "hello")
}

func TestHandleHTTP_SimpleAction_ReturnsJSON(t *testing.T) {
	t.Parallel()

	handler := NewActionHandler(nil, 1024*1024, nil, security_dto.RateLimitValues{}, false, nil, nil)
	handler.Register(ActionHandlerEntry{
		Name:   "test.action",
		Method: http.MethodPost,
		Create: func() any { return struct{}{} },
		Invoke: func(_ context.Context, _ any, _ map[string]any) (any, error) {
			return map[string]string{"result": "ok"}, nil
		},
	})

	r := chi.NewRouter()
	handler.Mount(r, "/_piko/actions")

	body := `{"key":"value"}`
	request := httptest.NewRequest(http.MethodPost, "/_piko/actions/test.action", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	r.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "ok")
}

func TestHandleHTTP_InvokeError_ReturnsError(t *testing.T) {
	t.Parallel()

	handler := NewActionHandler(nil, 1024*1024, nil, security_dto.RateLimitValues{}, false, nil, nil)
	handler.Register(ActionHandlerEntry{
		Name:   "failing.action",
		Method: http.MethodPost,
		Create: func() any { return struct{}{} },
		Invoke: func(_ context.Context, _ any, _ map[string]any) (any, error) {
			return nil, errors.New("something went wrong")
		},
	})

	r := chi.NewRouter()
	handler.Mount(r, "/_piko/actions")

	request := httptest.NewRequest(http.MethodPost, "/_piko/actions/failing.action", strings.NewReader("{}"))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	r.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "INTERNAL_ERROR")
}

func TestBuildFullResponse_NoHelpers_ReturnsRawResult(t *testing.T) {
	t.Parallel()

	handler := NewActionHandler(nil, 1024, nil, security_dto.RateLimitValues{}, false, nil, nil)
	action := &mockResponseGetterAction{response: daemon_dto.NewResponseWriter()}

	result := handler.buildFullResponse(action, "raw-data")

	assert.Equal(t, "raw-data", result)
}

func TestBuildFullResponse_WithHelpers_WrapsResult(t *testing.T) {
	t.Parallel()

	handler := NewActionHandler(nil, 1024, nil, security_dto.RateLimitValues{}, false, nil, nil)
	response := daemon_dto.NewResponseWriter()
	response.AddHelper("toast", map[string]any{"message": "saved"})
	action := &mockResponseGetterAction{response: response}

	result := handler.buildFullResponse(action, "raw-data")

	fullResp, ok := result.(daemon_dto.ActionFullResponse)
	require.True(t, ok)
	assert.Equal(t, "raw-data", fullResp.Data)
	assert.Len(t, fullResp.Helpers, 1)
}

func TestBuildFullResponse_NonResponseGetter_ReturnsRaw(t *testing.T) {
	t.Parallel()

	handler := NewActionHandler(nil, 1024, nil, security_dto.RateLimitValues{}, false, nil, nil)
	action := struct{}{}

	result := handler.buildFullResponse(action, "data")

	assert.Equal(t, "data", result)
}

func TestCreateHandler_ReturnsHandler(t *testing.T) {
	t.Parallel()

	handler := NewActionHandler(nil, 1024, nil, security_dto.RateLimitValues{}, false, nil, nil)
	entry := ActionHandlerEntry{
		Name:   "test.action",
		Method: http.MethodPost,
		Create: func() any { return struct{}{} },
		Invoke: func(_ context.Context, _ any, _ map[string]any) (any, error) { return nil, nil },
	}

	h := handler.createHandler(entry)
	assert.NotNil(t, h)
}

func TestNewActionHandler_WithRateLimit_CreatesMiddleware(t *testing.T) {
	t.Parallel()

	handler := NewActionHandler(nil, 1024, &security_domain.MockRateLimitService{
		CheckLimitFunc: func(_ string, _ int, _ time.Duration) (ratelimiter_dto.Result, error) {
			return ratelimiter_dto.Result{Allowed: true}, nil
		},
	}, security_dto.RateLimitValues{Enabled: true}, false, nil, nil)

	assert.NotNil(t, handler)
}

func TestNewActionHandler_WithoutRateLimit_NilMiddleware(t *testing.T) {
	t.Parallel()

	handler := NewActionHandler(nil, 1024, nil, security_dto.RateLimitValues{Enabled: false}, false, nil, nil)
	assert.Nil(t, handler.rateLimitMw)
}

func TestNewActionHandler_ResponseCacheNilWhenNotProvided(t *testing.T) {
	t.Parallel()

	handler := NewActionHandler(nil, 1024, nil, security_dto.RateLimitValues{}, false, nil, nil)
	assert.Nil(t, handler.responseCache)
}

func TestHandleCachedAction_StoresAndRetrievesFromCache(t *testing.T) {
	t.Parallel()

	responseCache, err := provider_otter.OtterProviderFactory(cache_dto.Options[string, []byte]{
		Namespace:   "test-action-responses",
		MaximumSize: 100,
	})
	require.NoError(t, err)
	defer func() { _ = responseCache.Close(context.Background()) }()

	invocations := 0
	handler := NewActionHandler(nil, 1024, nil, security_dto.RateLimitValues{}, false, responseCache, nil)
	handler.Register(ActionHandlerEntry{
		Name:   "cached.action",
		Method: http.MethodPost,
		Create: func() any { return &cacheableAction{} },
		Invoke: func(_ context.Context, _ any, _ map[string]any) (any, error) {
			invocations++
			return map[string]any{"result": "ok"}, nil
		},
	})

	r := chi.NewRouter()
	handler.Mount(r, "/_piko/actions")

	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, httptest.NewRequest(http.MethodPost, "/_piko/actions/cached.action", nil))
	assert.Equal(t, http.StatusOK, w1.Code)
	assert.Equal(t, "MISS", w1.Header().Get("X-Action-Cache"))
	assert.Equal(t, 1, invocations)

	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest(http.MethodPost, "/_piko/actions/cached.action", nil))
	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Equal(t, "HIT", w2.Header().Get("X-Action-Cache"))
	assert.Equal(t, 1, invocations, "action should not be invoked on cache hit")

	assert.JSONEq(t, w1.Body.String(), w2.Body.String())
}

func TestHandleCachedAction_NilCache_SkipsCaching(t *testing.T) {
	t.Parallel()

	invocations := 0
	handler := NewActionHandler(nil, 1024, nil, security_dto.RateLimitValues{}, false, nil, nil)
	handler.Register(ActionHandlerEntry{
		Name:   "cached.action",
		Method: http.MethodPost,
		Create: func() any { return &cacheableAction{} },
		Invoke: func(_ context.Context, _ any, _ map[string]any) (any, error) {
			invocations++
			return map[string]any{"result": "ok"}, nil
		},
	})

	r := chi.NewRouter()
	handler.Mount(r, "/_piko/actions")

	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, httptest.NewRequest(http.MethodPost, "/_piko/actions/cached.action", nil))
	assert.Equal(t, http.StatusOK, w1.Code)
	assert.Empty(t, w1.Header().Get("X-Action-Cache"))

	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest(http.MethodPost, "/_piko/actions/cached.action", nil))
	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Equal(t, 2, invocations, "action should be invoked every time without cache")
}

type cacheableAction struct{}

func (*cacheableAction) CacheConfig() *daemon_domain.CacheConfig {
	return &daemon_domain.CacheConfig{TTL: time.Minute}
}

type mockCaptchaService struct {
	verifyErr error
	enabled   bool
}

func (m *mockCaptchaService) Verify(_ context.Context, _, _, _ string) error {
	return m.verifyErr
}

func (m *mockCaptchaService) VerifyWithScore(_ context.Context, _, _, _ string, _ float64) (*captcha_dto.VerifyResponse, error) {
	return nil, m.verifyErr
}

func (m *mockCaptchaService) VerifyWithProvider(_ context.Context, _, _, _, _ string, _ float64) (*captcha_dto.VerifyResponse, error) {
	return nil, m.verifyErr
}

func (m *mockCaptchaService) IsEnabled() bool {
	return m.enabled
}

func (*mockCaptchaService) SiteKey() string   { return "" }
func (*mockCaptchaService) ScriptURL() string { return "" }

func (*mockCaptchaService) GetDefaultProvider(_ context.Context) (captcha_domain.CaptchaProvider, error) {
	return nil, captcha_dto.ErrCaptchaDisabled
}

func (*mockCaptchaService) GetProviderByName(_ context.Context, _ string) (captcha_domain.CaptchaProvider, error) {
	return nil, captcha_dto.ErrCaptchaDisabled
}

func (*mockCaptchaService) RegisterProvider(_ context.Context, _ string, _ captcha_domain.CaptchaProvider) error {
	return nil
}

func (*mockCaptchaService) SetDefaultProvider(_ string) error { return nil }

func (*mockCaptchaService) GetProviders(_ context.Context) []string { return nil }

func (*mockCaptchaService) HasProvider(_ string) bool { return false }

func (*mockCaptchaService) ListProviders(_ context.Context) []provider_domain.ProviderInfo {
	return nil
}

func (*mockCaptchaService) HealthCheck(_ context.Context) error { return nil }

func (*mockCaptchaService) Close(_ context.Context) error { return nil }

var _ captcha_domain.CaptchaServicePort = (*mockCaptchaService)(nil)

type mockCaptchaAction struct{}

func (mockCaptchaAction) CaptchaConfig() *daemon_domain.CaptchaConfig {
	return &daemon_domain.CaptchaConfig{}
}

type mockNilCaptchaConfigAction struct{}

func (mockNilCaptchaConfigAction) CaptchaConfig() *daemon_domain.CaptchaConfig {
	return nil
}

func TestValidateCaptcha_NoInterface(t *testing.T) {
	t.Parallel()

	handler := NewActionHandler(nil, 1024, nil, security_dto.RateLimitValues{}, false, nil, nil)
	arguments := map[string]any{
		"_captcha_token": "some-token",
		"name":           "test",
	}
	request := httptest.NewRequest(http.MethodPost, "/", nil)

	err := handler.validateCaptcha(t.Context(), request, struct{}{}, arguments, "test.action")

	assert.NoError(t, err)
	assert.NotContains(t, arguments, "_captcha_token")
	assert.Contains(t, arguments, "name")
}

func TestValidateCaptcha_NilConfig(t *testing.T) {
	t.Parallel()

	handler := NewActionHandler(nil, 1024, nil, security_dto.RateLimitValues{}, false, nil, nil)
	arguments := map[string]any{
		"_captcha_token": "some-token",
	}
	request := httptest.NewRequest(http.MethodPost, "/", nil)

	err := handler.validateCaptcha(t.Context(), request, mockNilCaptchaConfigAction{}, arguments, "test.action")

	assert.NoError(t, err)
	assert.NotContains(t, arguments, "_captcha_token")
}

func TestValidateCaptcha_ServiceUnavailable(t *testing.T) {
	t.Parallel()

	handler := NewActionHandler(nil, 1024, nil, security_dto.RateLimitValues{}, false, nil, nil)
	arguments := map[string]any{
		"_captcha_token": "some-token",
	}
	request := httptest.NewRequest(http.MethodPost, "/", nil)

	err := handler.validateCaptcha(t.Context(), request, mockCaptchaAction{}, arguments, "test.action")

	require.Error(t, err)
	assert.ErrorIs(t, err, captcha_dto.ErrCaptchaDisabled)
}

func TestValidateCaptcha_ServiceDisabled(t *testing.T) {
	t.Parallel()

	captchaSvc := &mockCaptchaService{enabled: false}
	handler := NewActionHandler(nil, 1024, nil, security_dto.RateLimitValues{}, false, nil, captchaSvc)
	arguments := map[string]any{
		"_captcha_token": "some-token",
	}
	request := httptest.NewRequest(http.MethodPost, "/", nil)

	err := handler.validateCaptcha(t.Context(), request, mockCaptchaAction{}, arguments, "test.action")

	require.Error(t, err)
	assert.ErrorIs(t, err, captcha_dto.ErrCaptchaDisabled)
}

func TestValidateCaptcha_ValidToken(t *testing.T) {
	t.Parallel()

	captchaSvc := &mockCaptchaService{enabled: true, verifyErr: nil}
	handler := NewActionHandler(nil, 1024, nil, security_dto.RateLimitValues{}, false, nil, captchaSvc)
	arguments := map[string]any{
		"_captcha_token": "valid-token",
		"name":           "test",
	}
	request := httptest.NewRequest(http.MethodPost, "/", nil)

	err := handler.validateCaptcha(t.Context(), request, mockCaptchaAction{}, arguments, "test.action")

	assert.NoError(t, err)
	assert.NotContains(t, arguments, "_captcha_token")
	assert.Contains(t, arguments, "name")
}

func TestValidateCaptcha_InvalidToken(t *testing.T) {
	t.Parallel()

	captchaSvc := &mockCaptchaService{enabled: true, verifyErr: errors.New("captcha verification failed")}
	handler := NewActionHandler(nil, 1024, nil, security_dto.RateLimitValues{}, false, nil, captchaSvc)
	arguments := map[string]any{
		"_captcha_token": "bad-token",
	}
	request := httptest.NewRequest(http.MethodPost, "/", nil)

	err := handler.validateCaptcha(t.Context(), request, mockCaptchaAction{}, arguments, "test.action")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "captcha verification failed")
}

func TestActionHandler_WriteCaptchaError(t *testing.T) {
	t.Parallel()

	handler := NewActionHandler(nil, 1024, nil, security_dto.RateLimitValues{}, false, nil, nil)

	recorder := httptest.NewRecorder()

	handler.writeCaptchaError(recorder)

	assert.Equal(t, http.StatusForbidden, recorder.Code)
	assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
	body := recorder.Body.String()
	assert.Contains(t, body, "CAPTCHA_FAILED")
}
