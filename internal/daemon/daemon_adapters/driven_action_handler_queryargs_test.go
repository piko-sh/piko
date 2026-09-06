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
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/security/security_dto"
	pikobinder "piko.sh/piko/wdk/binder"
)

func TestQueryArguments(t *testing.T) {
	t.Run("a parameter given once stays scalar", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/?repo=prick&limit=50", nil)

		assert.Equal(t, map[string]any{"repo": "prick", "limit": "50"}, queryArguments(request))
	})

	t.Run("a repeated parameter becomes a slice", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/?ref=main&ref=next", nil)

		assert.Equal(t, map[string]any{"ref": []any{"main", "next"}}, queryArguments(request))
	})

	t.Run("no query string yields an empty map, never nil", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/", nil)

		arguments := queryArguments(request)

		require.NotNil(t, arguments)
		assert.Empty(t, arguments)
	})

	t.Run("the ephemeral CSRF token is transport plumbing, not input", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet,
			"/?repo=prick&"+csrfEphemeralTokenKey+"=abc123", nil)

		arguments := queryArguments(request)

		assert.Equal(t, "prick", arguments["repo"])
		assert.NotContains(t, arguments, csrfEphemeralTokenKey,
			"binding the framework's own token would expose it to the action's input")
	})

	t.Run("a pathological query is capped", func(t *testing.T) {
		values := url.Values{}
		for i := range maxQueryArgumentKeys * 4 {
			values.Set("k"+strconv.Itoa(i), "v")
		}
		request := httptest.NewRequest(http.MethodGet, "/?"+values.Encode(), nil)

		assert.Len(t, queryArguments(request), maxQueryArgumentKeys)
	})
}

var (
	errRejectedInput = errors.New("input rejected")
)

type aliasStreamInput struct {
	Repo  string `json:"repo" validate:"required"`
	Limit int    `json:"limit"`
}

func TestSSEGetAliasBindsRequiredInputFromTheQueryString(t *testing.T) {

	bind := func(bound *aliasStreamInput) func(context.Context, any, map[string]any) error {
		return func(ctx context.Context, _ any, arguments map[string]any) error {
			return pikobinder.BindMap(ctx, bound,
				pikobinder.ActionInputSource(arguments, "input"),
				pikobinder.IgnoreUnknownKeys(true),
				pikobinder.WithDocumentScaleLimits(),
				pikobinder.WithValidation(true),
			)
		}
	}

	serve := func(t *testing.T, target string) (*aliasStreamInput, *httptest.ResponseRecorder) {
		t.Helper()

		var bound aliasStreamInput

		handler := NewActionHandler(nil, 1<<20, nil, security_dto.RateLimitValues{}, false, nil, nil)
		handler.Register(ActionHandlerEntry{
			Name:        "stream.Events",
			Method:      http.MethodPost,
			HasSSE:      true,
			SSEGetAlias: true,
			Create:      func() any { return &struct{}{} },
			Invoke:      func(context.Context, any, map[string]any) (any, error) { return nil, nil },
			Bind:        bind(&bound),
		})

		router := chi.NewRouter()
		handler.Mount(router, "/_piko/actions")

		request := httptest.NewRequest(http.MethodGet, target, nil)
		request.Header.Set("Accept", "text/event-stream")
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)

		return &bound, recorder
	}

	t.Run("the query satisfies a required field", func(t *testing.T) {
		bound, recorder := serve(t, "/_piko/actions/stream.Events?repo=prick&limit=50")

		assert.NotEqual(t, http.StatusUnprocessableEntity, recorder.Code,
			"a required field must be satisfiable over the alias an EventSource has to use")
		assert.Equal(t, "prick", bound.Repo)
		assert.Equal(t, 50, bound.Limit,
			"the binder's converters turn the query's strings into the declared types")
	})

	t.Run("an absent parameter is left absent, not fabricated", func(t *testing.T) {

		bound, _ := serve(t, "/_piko/actions/stream.Events?limit=50")

		assert.Empty(t, bound.Repo, "a parameter the query never carried must stay zero")
		assert.Equal(t, 50, bound.Limit)
	})

	t.Run("a bind failure still refuses the stream", func(t *testing.T) {
		handler := NewActionHandler(nil, 1<<20, nil, security_dto.RateLimitValues{}, false, nil, nil)
		handler.Register(ActionHandlerEntry{
			Name:        "stream.Events",
			Method:      http.MethodPost,
			HasSSE:      true,
			SSEGetAlias: true,
			Create:      func() any { return &struct{}{} },
			Invoke: func(context.Context, any, map[string]any) (any, error) {
				t.Error("the action ran despite its input failing to bind")

				return nil, nil
			},
			Bind: func(context.Context, any, map[string]any) error {
				return errRejectedInput
			},
		})

		router := chi.NewRouter()
		handler.Mount(router, "/_piko/actions")

		request := httptest.NewRequest(http.MethodGet, "/_piko/actions/stream.Events?repo=prick", nil)
		request.Header.Set("Accept", "text/event-stream")
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)

		assert.NotEqual(t, http.StatusOK, recorder.Code,
			"carrying input over the alias must not make its rejection unreportable")
	})
}
