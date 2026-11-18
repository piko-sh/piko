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
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/daemon/daemon_domain"
	"piko.sh/piko/internal/daemon/daemon_dto"
	"piko.sh/piko/internal/security/security_dto"
	"piko.sh/piko/internal/spamdetect/spamdetect_domain"
	"piko.sh/piko/internal/spamdetect/spamdetect_dto"
)

type stubSpamDetectService struct {
	spamdetect_domain.DisabledSpamDetectService
	analyseErr     error
	result         *spamdetect_dto.AnalysisResult
	lastSubmission *spamdetect_dto.Submission
	analyseCalls   int
	enabled        bool
}

func (s *stubSpamDetectService) IsEnabled(context.Context) bool { return s.enabled }

func (s *stubSpamDetectService) Analyse(
	_ context.Context,
	submission *spamdetect_dto.Submission,
	_ *spamdetect_dto.Schema,
) (*spamdetect_dto.AnalysisResult, error) {
	s.analyseCalls++
	s.lastSubmission = submission
	if s.analyseErr != nil {
		return nil, s.analyseErr
	}
	return s.result, nil
}

type spamProtectedAction struct {
	schema *spamdetect_dto.Schema
	daemon_dto.ActionMetadata
}

func (a *spamProtectedAction) SpamSchema() *spamdetect_dto.Schema { return a.schema }

type configurableSpamAction struct {
	config *daemon_domain.SpamConfig
	spamProtectedAction
}

func (a *configurableSpamAction) SpamConfig() *daemon_domain.SpamConfig { return a.config }

func newSpamSchema(t *testing.T) *spamdetect_dto.Schema {
	t.Helper()

	schema, err := spamdetect_dto.NewSchema(
		spamdetect_dto.TextField("message", spamdetect_dto.SignalGibberish),
		spamdetect_dto.EmailField("email", spamdetect_dto.SignalBlocklist),
		spamdetect_dto.Honeypot("website"),
		spamdetect_dto.Timing("loaded_at"),
		spamdetect_dto.Meta("form", "contact"),
		spamdetect_dto.CaptureHeader("Accept-Language"),
	)
	require.NoError(t, err)
	return schema
}

func newSpamHandler(service spamdetect_domain.SpamDetectServicePort) *ActionHandler {
	handler := NewActionHandler(nil, 1<<20, nil, security_dto.RateLimitValues{}, false, nil, nil)
	handler.spamdetectService = service
	return handler
}

func newSpamProtectedAction(t *testing.T) *spamProtectedAction {
	t.Helper()

	action := &spamProtectedAction{schema: newSpamSchema(t)}
	action.SetRequest(&daemon_dto.RequestMetadata{})
	return action
}

func TestValidateSpamDetect_SkipsWhenNotApplicable(t *testing.T) {
	t.Parallel()

	enabled := &stubSpamDetectService{enabled: true, result: &spamdetect_dto.AnalysisResult{Score: 1, Threshold: 0.5}}

	testCases := []struct {
		service spamdetect_domain.SpamDetectServicePort
		action  any
		name    string
	}{
		{name: "action without a schema method", service: enabled, action: &struct{}{}},
		{name: "action with a nil schema", service: enabled, action: &spamProtectedAction{schema: nil}},
		{name: "no service configured", service: nil, action: newSpamProtectedAction(t)},
		{name: "service disabled", service: &stubSpamDetectService{enabled: false}, action: newSpamProtectedAction(t)},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			handler := newSpamHandler(testCase.service)
			request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/action", nil)

			err := handler.validateSpamDetect(t.Context(), request, testCase.action, map[string]any{}, "contact.send")

			assert.NoError(t, err)
		})
	}
}

func TestValidateSpamDetect_ComparesTheScoreWithTheThreshold(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		config    *daemon_domain.SpamConfig
		wantErr   error
		name      string
		score     float64
		threshold float64
	}{
		{name: "score under the threshold is allowed", score: 0.2, threshold: 0.5},
		{name: "score at the threshold is rejected", score: 0.5, threshold: 0.5, wantErr: spamdetect_dto.ErrSpamDetected},
		{
			name:      "an action threshold overrides the service threshold",
			score:     0.6,
			threshold: 0.5,
			config:    &daemon_domain.SpamConfig{ScoreThreshold: 0.9},
		},
		{
			name:      "a zero action threshold keeps the service threshold",
			score:     0.6,
			threshold: 0.5,
			config:    &daemon_domain.SpamConfig{ScoreThreshold: 0},
			wantErr:   spamdetect_dto.ErrSpamDetected,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := &stubSpamDetectService{
				enabled: true,
				result:  &spamdetect_dto.AnalysisResult{Score: testCase.score, Threshold: testCase.threshold},
			}
			handler := newSpamHandler(service)
			request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/action", nil)

			var action any = newSpamProtectedAction(t)
			if testCase.config != nil {
				configurable := &configurableSpamAction{spamProtectedAction: *newSpamProtectedAction(t), config: testCase.config}
				action = configurable
			}

			err := handler.validateSpamDetect(t.Context(), request, action, map[string]any{"message": "hello"}, "contact.send")

			if testCase.wantErr != nil {
				assert.ErrorIs(t, err, testCase.wantErr)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, 1, service.analyseCalls)
		})
	}
}

func TestValidateSpamDetect_HandlesAnalysisErrors(t *testing.T) {
	t.Parallel()

	unexpected := errors.New("detector backend unreachable")

	testCases := []struct {
		analyseErr error
		wantErr    error
		name       string
		wantNil    bool
	}{
		{name: "disabled service allows the submission", analyseErr: spamdetect_dto.ErrSpamDetectDisabled, wantNil: true},
		{name: "no matching detectors allows the submission", analyseErr: spamdetect_dto.ErrNoMatchingDetectors, wantNil: true},
		{name: "every detector failing rejects the submission", analyseErr: spamdetect_dto.ErrAllDetectorsFailed, wantErr: spamdetect_dto.ErrSpamDetected},
		{name: "any other failure is returned wrapped", analyseErr: unexpected, wantErr: unexpected},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			handler := newSpamHandler(&stubSpamDetectService{enabled: true, analyseErr: testCase.analyseErr})
			request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/action", nil)

			err := handler.validateSpamDetect(t.Context(), request, newSpamProtectedAction(t), map[string]any{}, "contact.send")

			if testCase.wantNil {
				assert.NoError(t, err)
				return
			}
			assert.ErrorIs(t, err, testCase.wantErr)
		})
	}
}

func TestValidateSpamDetect_RecordsTheScoreOnTheAction(t *testing.T) {
	t.Parallel()

	service := &stubSpamDetectService{
		enabled: true,
		result: &spamdetect_dto.AnalysisResult{
			Score:       0.3,
			Threshold:   0.5,
			FormReasons: []string{"submitted quickly"},
			FieldResults: []spamdetect_dto.FieldResult{
				{Key: "message", Reasons: []string{"looks random"}, Score: 0.4},
				{Key: "email", Score: 0},
			},
		},
	}
	handler := newSpamHandler(service)
	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/action", nil)
	action := newSpamProtectedAction(t)

	require.NoError(t, handler.validateSpamDetect(t.Context(), request, action, map[string]any{}, "contact.send"))

	requestMeta := action.Request()
	require.NotNil(t, requestMeta.SpamScore)
	assert.InDelta(t, 0.3, *requestMeta.SpamScore, 0.0001)
	assert.Equal(t, []string{"submitted quickly", "looks random"}, requestMeta.SpamReasons)
	assert.Equal(t, map[string]float64{"message": 0.4}, requestMeta.SpamFieldScores)
}

func TestRecordSpamScore_IgnoresMissingTargets(t *testing.T) {
	t.Parallel()

	t.Run("nil result", func(t *testing.T) {
		t.Parallel()

		action := newSpamProtectedAction(t)
		recordSpamScore(action, nil)
		assert.Nil(t, action.Request().SpamScore)
	})

	t.Run("action without request metadata", func(t *testing.T) {
		t.Parallel()

		assert.NotPanics(t, func() {
			recordSpamScore(&struct{}{}, &spamdetect_dto.AnalysisResult{Score: 1})
		})
	})

	t.Run("action whose request metadata is not set", func(t *testing.T) {
		t.Parallel()

		action := &spamProtectedAction{}
		assert.NotPanics(t, func() {
			recordSpamScore(action, &spamdetect_dto.AnalysisResult{Score: 1})
		})
		assert.Nil(t, action.Request())
	})

	t.Run("clean result leaves reasons and field scores unset", func(t *testing.T) {
		t.Parallel()

		action := newSpamProtectedAction(t)
		recordSpamScore(action, &spamdetect_dto.AnalysisResult{Score: 0})

		require.NotNil(t, action.Request().SpamScore)
		assert.Nil(t, action.Request().SpamReasons)
		assert.Nil(t, action.Request().SpamFieldScores)
	})
}

func TestBuildSpamSubmission(t *testing.T) {
	t.Parallel()

	loadedAt := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	schema := newSpamSchema(t)
	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/action", nil)
	request.RemoteAddr = "198.51.100.4:5555"
	request.Header.Set("User-Agent", "test-agent")
	request.Header.Set("Referer", "https://example.com/contact")
	request.Header.Set("Accept-Language", "en-GB")

	arguments := map[string]any{
		"message":   "hello there",
		"email":     42,
		"website":   "http://spam.example",
		"loaded_at": strconv.FormatInt(loadedAt.UnixMilli(), 10),
	}

	submission := buildSpamSubmission(request, arguments, schema, "contact.send")

	assert.Equal(t, "hello there", submission.Fields["message"].Value)
	assert.Empty(t, submission.Fields["email"].Value, "a value that is not a string is read as empty")
	assert.Equal(t, "http://spam.example", submission.HoneypotValue)
	assert.True(t, loadedAt.Equal(submission.FormLoadedAt))
	assert.Equal(t, "198.51.100.4:5555", submission.RemoteIP, "the peer address is used when no client IP was resolved")
	assert.Equal(t, "test-agent", submission.UserAgent)
	assert.Equal(t, "https://example.com/contact", submission.PageURL)
	assert.Equal(t, "contact.send", submission.ActionName)
	assert.Equal(t, map[string]string{"form": "contact"}, submission.Metadata)
	assert.Equal(t, map[string]string{"Accept-Language": "en-GB"}, submission.Headers)
	assert.NotContains(t, arguments, "website", "the honeypot is removed before binding")
	assert.NotContains(t, arguments, "loaded_at", "the timing token is removed before binding")
	assert.Contains(t, arguments, "message")
}

func TestBuildSpamSubmission_UsesTheResolvedClientIP(t *testing.T) {
	t.Parallel()

	ctx := daemon_dto.WithPikoRequestCtx(t.Context(), &daemon_dto.PikoRequestCtx{ClientIP: "203.0.113.9"})
	request := httptest.NewRequestWithContext(ctx, http.MethodPost, "/action", nil)
	schema, err := spamdetect_dto.NewSchema(spamdetect_dto.TextField("message", spamdetect_dto.SignalGibberish))
	require.NoError(t, err)

	submission := buildSpamSubmission(request, map[string]any{}, schema, "contact.send")

	assert.Equal(t, "203.0.113.9", submission.RemoteIP)
	assert.Empty(t, submission.HoneypotValue)
	assert.True(t, submission.FormLoadedAt.IsZero())
	assert.Nil(t, submission.Metadata)
	assert.Nil(t, submission.Headers)
}

func TestCaptureHeaders(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	request.Header.Set("X-Present", "yes")

	assert.Nil(t, captureHeaders(request, nil))
	assert.Nil(t, captureHeaders(request, []string{"X-Missing"}))
	assert.Equal(t, map[string]string{"X-Present": "yes"}, captureHeaders(request, []string{"X-Present", "X-Missing"}))
}

func TestCopyMetadata(t *testing.T) {
	t.Parallel()

	assert.Nil(t, copyMetadata(nil))
	assert.Nil(t, copyMetadata(map[string]string{}))

	source := map[string]string{"form": "contact"}
	copied := copyMetadata(source)
	copied["form"] = "changed"
	assert.Equal(t, "contact", source["form"], "the copy must not share storage with the schema")
}

func TestCollectSpamFieldScores(t *testing.T) {
	t.Parallel()

	assert.Nil(t, collectSpamFieldScores(&spamdetect_dto.AnalysisResult{}))
	assert.Nil(t, collectSpamFieldScores(&spamdetect_dto.AnalysisResult{
		FieldResults: []spamdetect_dto.FieldResult{{Key: "message", Score: 0}},
	}))
}

func TestParseFormTimestamp(t *testing.T) {
	t.Parallel()

	assert.True(t, parseFormTimestamp("").IsZero())
	assert.True(t, parseFormTimestamp("not-a-number").IsZero())
	assert.Equal(t, int64(1_700_000_000_000), parseFormTimestamp("1700000000000").UnixMilli())
}

func TestStringArgHelpers(t *testing.T) {
	t.Parallel()

	arguments := map[string]any{"text": "value", "number": 7}

	assert.Equal(t, "value", readStringArg(arguments, "text"))
	assert.Empty(t, readStringArg(arguments, "number"))
	assert.Empty(t, readStringArg(arguments, "missing"))
	assert.Contains(t, arguments, "text", "reading leaves the key in place")

	assert.Empty(t, extractStringArg(arguments, "number"))
	assert.Empty(t, extractStringArg(arguments, "missing"))
	assert.Equal(t, "value", extractStringArg(arguments, "text"))
	assert.Empty(t, arguments, "extracting removes the key")
}

func TestHandleRequest_RejectsSpamWithForbidden(t *testing.T) {
	t.Parallel()

	service := &stubSpamDetectService{
		enabled: true,
		result:  &spamdetect_dto.AnalysisResult{Score: 0.95, Threshold: 0.5},
	}
	handler := newSpamHandler(service)
	invoked := false
	schema := newSpamSchema(t)
	handler.Register(ActionHandlerEntry{
		Name:   "contact.send",
		Method: http.MethodPost,
		Create: func() any { return &spamProtectedAction{schema: schema} },
		Invoke: func(context.Context, any, map[string]any) (any, error) {
			invoked = true
			return nil, nil
		},
	})
	router := chi.NewRouter()
	handler.Mount(router, "/_piko/actions")

	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/_piko/actions/contact.send",
		strings.NewReader(`{"message":"buy now"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assert.False(t, invoked, "a spam submission must not reach the action")
	assert.Equal(t, http.StatusForbidden, recorder.Code)
	assert.Contains(t, recorder.Body.String(), `"code":"SPAM_DETECTED"`)
	require.NotNil(t, service.lastSubmission)
	assert.Equal(t, "buy now", service.lastSubmission.Fields["message"].Value)
}
