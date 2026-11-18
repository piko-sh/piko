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
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/json"
	"piko.sh/piko/internal/binder"
	"piko.sh/piko/internal/daemon/daemon_domain"
	"piko.sh/piko/internal/daemon/daemon_dto"
	"piko.sh/piko/internal/security/security_dto"
)

const (
	batchSaturationTimeout = 10 * time.Second
)

type metadataAction struct {
	daemon_dto.ActionMetadata
}

func batchRequestFor(parallel bool, names ...string) *http.Request {
	entries := make([]string, 0, len(names))
	for _, name := range names {
		entries = append(entries, fmt.Sprintf(`{"name":%q}`, name))
	}
	body := fmt.Sprintf(`{"actions":[%s],"parallel":%t}`, strings.Join(entries, ","), parallel)
	request := httptest.NewRequest(http.MethodPost, "/_piko/actions/_batch", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	return request
}

func registerMetadataAction(handler *ActionHandler, name string, invoke func(action any) (any, error)) {
	handler.Register(ActionHandlerEntry{
		Name:   name,
		Method: http.MethodPost,
		Create: func() any { return &metadataAction{} },
		Invoke: func(_ context.Context, action any, _ map[string]any) (any, error) {
			return invoke(action)
		},
	})
}

func TestHandleBatch_AppliesResponseMetadataFromEveryEntry(t *testing.T) {
	t.Parallel()

	handler := NewActionHandler(nil, 1024*1024, nil, security_dto.RateLimitValues{}, false, nil, nil)

	for _, name := range []string{"first", "second"} {
		registerMetadataAction(handler, name, func(action any) (any, error) {
			metadata, ok := action.(*metadataAction)
			if !ok {
				return nil, errors.New("action was not a metadataAction")
			}
			metadata.Response().SetCookie(&http.Cookie{Name: "pp_" + name, Value: name})
			metadata.Response().AddHeader("X-Batch-Entry", name)

			return name, nil
		})
	}

	router := chi.NewRouter()
	handler.Mount(router, "/_piko/actions")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, batchRequestFor(false, "first", "second"))

	require.Equal(t, http.StatusOK, recorder.Code)

	cookies := recorder.Result().Cookies()
	require.Len(t, cookies, 2, "both entries set a cookie and both must reach the response")
	assert.Equal(t, "pp_first", cookies[0].Name)
	assert.Equal(t, "pp_second", cookies[1].Name)

	assert.Equal(t, []string{"first", "second"}, recorder.Header().Values("X-Batch-Entry"),
		"headers must arrive in request order")
}

func TestHandleBatch_AppliesResponseMetadataFromAFailingEntry(t *testing.T) {
	t.Parallel()

	handler := NewActionHandler(nil, 1024*1024, nil, security_dto.RateLimitValues{}, false, nil, nil)

	registerMetadataAction(handler, "failing", func(action any) (any, error) {
		metadata, ok := action.(*metadataAction)
		if !ok {
			return nil, errors.New("action was not a metadataAction")
		}
		metadata.Response().SetCookie(&http.Cookie{Name: "pp_session", Value: "cleared"})

		return nil, errors.New("action failed")
	})

	router := chi.NewRouter()
	handler.Mount(router, "/_piko/actions")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, batchRequestFor(false, "failing"))

	cookies := recorder.Result().Cookies()
	require.Len(t, cookies, 1, "a cookie set before the action failed must still reach the response")
	assert.Equal(t, "pp_session", cookies[0].Name)
}

func TestExecuteBatchActions_ParallelPreservesRequestOrder(t *testing.T) {
	t.Parallel()

	handler := NewActionHandler(nil, 1024*1024, nil, security_dto.RateLimitValues{}, false, nil, nil)

	const entries = 3
	arrived := make(chan struct{}, entries)
	proceed := make(chan struct{})

	for _, name := range []string{"slow", "quick", "middling"} {
		registerMetadataAction(handler, name, func(_ any) (any, error) {
			arrived <- struct{}{}
			<-proceed

			return name, nil
		})
	}

	router := chi.NewRouter()
	handler.Mount(router, "/_piko/actions")
	recorder := httptest.NewRecorder()

	go func() {
		for range entries {
			<-arrived
		}
		close(proceed)
	}()

	router.ServeHTTP(recorder, batchRequestFor(true, "slow", "quick", "middling"))

	require.Equal(t, http.StatusOK, recorder.Code,
		"all three entries must be in flight together, or none of them can finish")

	var response daemon_dto.BatchActionResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Len(t, response.Results, 3)
	assert.Equal(t, "slow", response.Results[0].Name)
	assert.Equal(t, "quick", response.Results[1].Name)
	assert.Equal(t, "middling", response.Results[2].Name)
}

func TestExecuteBatchActions_ParallelBoundsConcurrency(t *testing.T) {
	t.Parallel()

	handler := NewActionHandler(nil, 1024*1024, nil, security_dto.RateLimitValues{}, false, nil, nil)

	const entries = 24

	var inFlight, peak atomic.Int64
	var saturateOnce sync.Once
	saturated := make(chan struct{})

	names := make([]string, 0, entries)
	for i := range entries {
		name := fmt.Sprintf("entry%d", i)
		names = append(names, name)
		registerMetadataAction(handler, name, func(_ any) (any, error) {
			current := inFlight.Add(1)
			for {
				observed := peak.Load()
				if current <= observed || peak.CompareAndSwap(observed, current) {
					break
				}
			}

			if current >= int64(defaultParallelBatchWorkers) {
				saturateOnce.Do(func() { close(saturated) })
			}

			select {
			case <-saturated:
			case <-time.After(batchSaturationTimeout):
			}

			inFlight.Add(-1)

			return nil, nil
		})
	}

	router := chi.NewRouter()
	handler.Mount(router, "/_piko/actions")
	router.ServeHTTP(httptest.NewRecorder(), batchRequestFor(true, names...))

	assert.Equal(t, int64(defaultParallelBatchWorkers), peak.Load(),
		"a parallel batch must fill its worker pool and never exceed it")
}

func TestExecuteBatchActions_IsolatesAPanicToItsOwnEntry(t *testing.T) {
	t.Parallel()

	for _, parallel := range []bool{false, true} {
		t.Run(fmt.Sprintf("parallel=%t", parallel), func(t *testing.T) {
			handler := NewActionHandler(nil, 1024*1024, nil, security_dto.RateLimitValues{}, false, nil, nil)

			registerMetadataAction(handler, "panicking", func(_ any) (any, error) {
				panic("action exploded")
			})
			registerMetadataAction(handler, "healthy", func(_ any) (any, error) {
				return "fine", nil
			})

			router := chi.NewRouter()
			handler.Mount(router, "/_piko/actions")
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, batchRequestFor(parallel, "panicking", "healthy"))

			require.Equal(t, http.StatusOK, recorder.Code,
				"a panicking entry must not take down the whole batch response")

			var response daemon_dto.BatchActionResponse
			require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
			require.Len(t, response.Results, 2)
			assert.GreaterOrEqual(t, response.Results[0].Status, http.StatusInternalServerError)
			assert.Equal(t, http.StatusOK, response.Results[1].Status)
			assert.Equal(t, "fine", response.Results[1].Data)
		})
	}
}

func TestExecuteBatchActions_ParallelAppliesResponseMetadataInRequestOrder(t *testing.T) {
	t.Parallel()

	handler := NewActionHandler(nil, 1024*1024, nil, security_dto.RateLimitValues{}, false, nil, nil)

	finished := map[string]chan struct{}{
		"gamma": make(chan struct{}),
		"beta":  make(chan struct{}),
	}
	waitsFor := map[string]chan struct{}{
		"beta":  finished["gamma"],
		"alpha": finished["beta"],
	}

	for _, name := range []string{"alpha", "beta", "gamma"} {
		registerMetadataAction(handler, name, func(action any) (any, error) {
			if predecessor, waits := waitsFor[name]; waits {
				<-predecessor
			}

			metadata, ok := action.(*metadataAction)
			if !ok {
				return nil, errors.New("action was not a metadataAction")
			}
			metadata.Response().SetCookie(&http.Cookie{Name: "pp_" + name, Value: name})
			metadata.Response().AddHeader("X-Batch-Entry", name)

			if signal, signals := finished[name]; signals {
				close(signal)
			}

			return name, nil
		})
	}

	router := chi.NewRouter()
	handler.Mount(router, "/_piko/actions")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, batchRequestFor(true, "alpha", "beta", "gamma"))

	require.Equal(t, http.StatusOK, recorder.Code)

	assert.Equal(t, []string{"alpha", "beta", "gamma"}, recorder.Header().Values("X-Batch-Entry"),
		"metadata must be applied in request order, not in the order the entries finished")

	cookies := recorder.Result().Cookies()
	require.Len(t, cookies, 3)
	assert.Equal(t, "pp_alpha", cookies[0].Name)
	assert.Equal(t, "pp_beta", cookies[1].Name)
	assert.Equal(t, "pp_gamma", cookies[2].Name)
}

func TestExecuteBatchActions_ReportsEntriesTheCallerAbandoned(t *testing.T) {
	t.Parallel()

	for _, parallel := range []bool{false, true} {
		t.Run(fmt.Sprintf("parallel=%t", parallel), func(t *testing.T) {
			t.Parallel()

			handler := NewActionHandler(nil, 1024*1024, nil, security_dto.RateLimitValues{}, false, nil, nil)

			var ran atomic.Int64
			for _, name := range []string{"first", "second"} {
				registerMetadataAction(handler, name, func(_ any) (any, error) {
					ran.Add(1)

					return name, nil
				})
			}

			ctx, cancel := context.WithCancel(t.Context())
			cancel()

			request := batchRequestFor(parallel, "first", "second")
			results, allSuccess := handler.executeBatchActions(
				ctx,
				httptest.NewRecorder(),
				request,
				[]daemon_dto.BatchActionItem{{Name: "first"}, {Name: "second"}},
				parallel,
			)

			assert.False(t, allSuccess)
			require.Len(t, results, 2)
			for index, result := range results {
				assert.Equal(t, http.StatusServiceUnavailable, result.Status,
					"an abandoned entry must be legible rather than a blank row")
				assert.Equal(t, "CANCELLED", result.Code)
				assert.NotEmpty(t, result.Name, "entry %d must still name its action", index)
			}
			assert.Zero(t, ran.Load(), "no action should run once the caller has gone away")
		})
	}
}

func TestParallelBatchWorkers_FallsBackToTheDefault(t *testing.T) {
	t.Parallel()

	handler := NewActionHandler(nil, 1024*1024, nil, security_dto.RateLimitValues{}, false, nil, nil)
	assert.Equal(t, defaultParallelBatchWorkers, handler.parallelBatchWorkers())

	handler.maxParallelBatchWorkers = 3
	assert.Equal(t, 3, handler.parallelBatchWorkers())

	handler.maxParallelBatchWorkers = -1
	assert.Equal(t, defaultParallelBatchWorkers, handler.parallelBatchWorkers(),
		"a nonsensical bound must not disable the pool")
}

func batchRequestBody(t *testing.T, count int) string {
	t.Helper()

	actions := make([]string, 0, count)
	for index := range count {
		actions = append(actions, `{"name":"a`+strconv.Itoa(index)+`","args":{}}`)
	}
	return `{"actions":[` + strings.Join(actions, ",") + `]}`
}

func TestHandleBatch_OverCapDoesNoAttribution(t *testing.T) {
	handler := &ActionHandler{maxBodyBytes: 1 << 20}

	pctx := daemon_dto.AcquirePikoRequestCtx()
	defer daemon_dto.ReleasePikoRequestCtx(pctx)

	body := batchRequestBody(t, maxBatchActions+1)
	request := httptest.NewRequest(http.MethodPost, "/_batch", strings.NewReader(body))
	request = request.WithContext(daemon_dto.WithPikoRequestCtx(request.Context(), pctx))
	recorder := httptest.NewRecorder()

	handler.handleBatch(recorder, request)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Empty(t, pctx.AnalyticsActionName, "a rejected batch is not attributed")
	assert.NotContains(t, pctx.AnalyticsProperties, "batch.actions",
		"a rejected batch does not build the joined action list")
}

func TestHandleBatch_WithinCapIsAttributed(t *testing.T) {
	handler := &ActionHandler{maxBodyBytes: 1 << 20}

	pctx := daemon_dto.AcquirePikoRequestCtx()
	defer daemon_dto.ReleasePikoRequestCtx(pctx)

	request := httptest.NewRequest(http.MethodPost, "/_batch",
		strings.NewReader(batchRequestBody(t, 2)))
	request = request.WithContext(daemon_dto.WithPikoRequestCtx(request.Context(), pctx))

	handler.handleBatch(httptest.NewRecorder(), request)

	assert.Equal(t, batchActionName, pctx.AnalyticsActionName)
	assert.Equal(t, "2", pctx.AnalyticsProperties["batch.count"])
	assert.Equal(t, "a0,a1", pctx.AnalyticsProperties["batch.actions"])
}

func TestExecuteBatchActions_ParallelEntriesMayRecordAnalytics(t *testing.T) {
	t.Parallel()

	handler := NewActionHandler(nil, 1024*1024, nil, security_dto.RateLimitValues{}, false, nil, nil)
	pctx := daemon_dto.AcquirePikoRequestCtx()
	defer daemon_dto.ReleasePikoRequestCtx(pctx)

	for _, name := range []string{"first", "second", "third", "fourth"} {
		registerMetadataAction(handler, name, func(action any) (any, error) {
			pctx.SetAnalyticsProperty(name, "recorded", 64)
			pctx.SetAnalyticsAction(name)

			return action, nil
		})
	}

	request := batchRequestFor(true, "first", "second", "third", "fourth")
	request = request.WithContext(daemon_dto.WithPikoRequestCtx(request.Context(), pctx))

	handler.handleBatch(httptest.NewRecorder(), request)

	for _, name := range []string{"first", "second", "third", "fourth"} {
		assert.Equal(t, "recorded", pctx.AnalyticsProperties[name],
			"every entry records its own property")
	}
}

type timedMetadataAction struct {
	daemon_dto.ActionMetadata
}

func (*timedMetadataAction) ResourceLimits() *daemon_domain.ResourceLimits {
	return &daemon_domain.ResourceLimits{Timeout: time.Minute}
}

func TestHandleBatch_MetadataCarriesTheActionTimeout(t *testing.T) {
	t.Parallel()

	handler := NewActionHandler(nil, 1024*1024, nil, security_dto.RateLimitValues{}, false, nil, nil)

	var (
		mu          sync.Mutex
		hasDeadline bool
	)
	handler.Register(ActionHandlerEntry{
		Name:   "timed",
		Method: http.MethodPost,
		Create: func() any { return &timedMetadataAction{} },
		Invoke: func(_ context.Context, action any, _ map[string]any) (any, error) {
			timed, ok := action.(*timedMetadataAction)
			if !ok {
				return nil, errors.New("action was not a timedMetadataAction")
			}

			_, deadlined := timed.Ctx().Deadline()

			mu.Lock()
			hasDeadline = deadlined
			mu.Unlock()

			return "done", nil
		},
	})

	router := chi.NewRouter()
	handler.Mount(router, "/_piko/actions")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, batchRequestFor(false, "timed"))

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.True(t, hasDeadline,
		"the action's metadata context must carry the per-call timeout on the batch path")
}

func TestHandleActionError_AnswersABindingFailureWithFieldErrors(t *testing.T) {
	t.Parallel()

	bindErr := bindQuantityForm(t, map[string]any{"Quantity": "lots", "Price": "1e400"})

	handler := NewActionHandler(nil, 1024, nil, security_dto.RateLimitValues{}, false, nil, nil)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/action", nil)
	handler.handleActionError(recorder, request, &struct{}{}, bindErr)

	require.Equal(t, http.StatusUnprocessableEntity, recorder.Code)

	var body struct {
		Errors  map[string]string `json:"errors"`
		Code    string            `json:"code"`
		Message string            `json:"message"`
		Status  int               `json:"status"`
	}
	require.NoError(t, json.ConfigDefault.Unmarshal(recorder.Body.Bytes(), &body))

	assert.Equal(t, http.StatusUnprocessableEntity, body.Status)
	assert.Equal(t, "VALIDATION_FAILED", body.Code)
	assert.Equal(t, "validation failed", body.Message)
	assert.Equal(t, map[string]string{
		"Quantity": "Enter a whole number",
		"Price":    "Number is too large or too small",
	}, body.Errors)
}

func TestBuildBatchErrorResult_AnswersABindingFailureWithFieldErrors(t *testing.T) {
	t.Parallel()

	bindErr := bindQuantityForm(t, map[string]any{"Quantity": "lots"})

	handler := NewActionHandler(nil, 1024, nil, security_dto.RateLimitValues{}, false, nil, nil)
	result := handler.buildBatchErrorResult(t.Context(), "order.create", bindErr, false)

	assert.Equal(t, http.StatusUnprocessableEntity, result.Status)
	assert.Equal(t, "VALIDATION_FAILED", result.Code)
	assert.Equal(t, "validation failed", result.Error)
	assert.Equal(t, map[string]string{"Quantity": "Enter a whole number"}, result.Errors)
}

func bindQuantityForm(t *testing.T, source map[string]any) error {
	t.Helper()

	var destination struct {
		Quantity int
		Price    float64
	}
	err := binder.NewASTBinder().BindMap(t.Context(), &destination, source)
	require.Error(t, err)

	return err
}

func TestFieldErrorsFor(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		err  error
		want map[string]string
		name string
	}{
		{
			name: "validation error raised by an action",
			err:  daemon_dto.NewValidationError(map[string]string{"email": "is required"}),
			want: map[string]string{"email": "is required"},
		},
		{
			name: "binding failure",
			err:  binder.MultiError{"Quantity": errors.New("could not convert")},
			want: map[string]string{"Quantity": "Enter a valid value"},
		},
		{
			name: "validate tag failure with field messages",
			err: &binder.ValidationFailedError{
				Err:    errors.New("validator rejected the input"),
				Fields: map[string][]string{"email": {"is required", "must be an email"}},
			},
			want: map[string]string{"email": "is required, must be an email"},
		},
		{
			name: "validate tag failure without field messages",
			err:  &binder.ValidationFailedError{Err: errors.New("validator rejected the input"), Fields: nil},
			want: nil,
		},
		{
			name: "wrapped binding failure",
			err:  fmt.Errorf("binding input: %w", binder.MultiError{"Quantity": errors.New("could not convert")}),
			want: map[string]string{"Quantity": "Enter a valid value"},
		},
		{
			name: "plain error",
			err:  errors.New("database unreachable"),
			want: nil,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, testCase.want, fieldErrorsFor(testCase.err))
		})
	}
}

func TestBuildBatchErrorResult_ReportsAnInvalidStatusOnce(t *testing.T) {
	t.Parallel()

	handler := NewActionHandler(nil, 1024*1024, nil, security_dto.RateLimitValues{}, false, nil, nil)
	request, logs := newLogRecordingRequest(t, http.MethodPost, "/_piko/actions/_batch", "{}")

	result := handler.buildBatchErrorResult(request.Context(), "order.create", daemon_dto.PageError(http.StatusFound, "moved"), false)

	assert.Equal(t, http.StatusInternalServerError, result.Status)
	assert.Len(t, logs.atLevel(slog.LevelError), 1, "an invalid status must be reported exactly once")
}

func TestExecuteBatchActions_ReportsAPanicOnce(t *testing.T) {
	t.Parallel()

	handler := NewActionHandler(nil, 1024*1024, nil, security_dto.RateLimitValues{}, false, nil, nil)
	registerMetadataAction(handler, "panicking", func(_ any) (any, error) {
		panic("action exploded")
	})
	router := chi.NewRouter()
	handler.Mount(router, "/_piko/actions")
	request, logs := newLogRecordingRequest(t, http.MethodPost, "/_piko/actions/_batch",
		`{"actions":[{"name":"panicking"}],"parallel":false}`)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Len(t, logs.atLevel(slog.LevelError), 1, "a recovered panic is already logged where it was recovered")
}
