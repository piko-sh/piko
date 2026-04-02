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

package pikotest_domain

import (
	"context"
	"net/http"
	"testing"
	"time"

	"piko.sh/piko/internal/daemon/daemon_adapters"
	"piko.sh/piko/internal/daemon/daemon_dto"
	"piko.sh/piko/internal/pikotest/pikotest_dto"
)

// ActionTester provides a test harness for server actions.
// It creates action instances, injects test metadata, invokes the action,
// and returns structured results for assertion.
type ActionTester struct {
	// tb is the test context for reporting failures.
	tb testing.TB

	// entry holds the action handler entry for creating and invoking actions.
	entry daemon_adapters.ActionHandlerEntry
}

// NewActionTester creates a new ActionTester for the given action handler entry.
//
// Takes tb (testing.TB) which is the test instance for error reporting.
// Takes entry (ActionHandlerEntry) which describes the action to test.
//
// Returns *ActionTester ready for invoking and asserting.
func NewActionTester(tb testing.TB, entry daemon_adapters.ActionHandlerEntry) *ActionTester {
	tb.Helper()
	return &ActionTester{
		tb:    tb,
		entry: entry,
	}
}

// Invoke creates a new action instance, injects test metadata, and calls
// the action with the given arguments.
//
// Takes ctx (context.Context) which is the context for the action invocation,
// used to inject mock dependencies via context values.
// Takes arguments (map[string]any) which maps parameter names to values.
// Pass nil or an empty map for actions with no parameters.
//
// Returns *ActionResultView which wraps the response and error for assertions.
func (at *ActionTester) Invoke(ctx context.Context, arguments map[string]any) *ActionResultView {
	at.tb.Helper()

	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime).Milliseconds()
		TestActionDuration.Record(context.Background(), float64(duration))
		TestActionCount.Add(context.Background(), 1)
	}()

	action := at.entry.Create()

	injectTestMetadata(action)

	if arguments == nil {
		arguments = make(map[string]any)
	}

	result, err := at.entry.Invoke(ctx, action, arguments)

	fullResponse := buildTestFullResponse(action, result)

	return newActionResultView(at.tb, &pikotest_dto.ActionResult{
		Response: fullResponse,
		Err:      err,
	})
}

// injectTestMetadata injects empty request/response metadata into an action,
// mirroring what the framework does at runtime.
//
// Takes action (any) which is the action to inject metadata into.
func injectTestMetadata(action any) {
	type metadataInjector interface {
		SetRequest(request *daemon_dto.RequestMetadata)
		SetResponse(response *daemon_dto.ResponseWriter)
	}

	if injector, ok := action.(metadataInjector); ok {
		injector.SetRequest(&daemon_dto.RequestMetadata{
			Method:      http.MethodPost,
			Path:        "/test",
			Headers:     make(http.Header),
			QueryParams: make(map[string][]string),
		})
		injector.SetResponse(daemon_dto.NewResponseWriter())
	}
}

// buildTestFullResponse builds the full response from the action, including
// any helpers that were set via Response().AddHelper().
//
// Takes action (any) which is checked for a Response() method to extract
// helpers.
// Takes result (any) which provides the data to include in the response.
//
// Returns *daemon_dto.ActionFullResponse which contains the result data and
// any helpers from the action's response writer.
func buildTestFullResponse(action any, result any) *daemon_dto.ActionFullResponse {
	type responseGetter interface {
		Response() *daemon_dto.ResponseWriter
	}

	response := &daemon_dto.ActionFullResponse{
		Data: result,
	}

	if getter, ok := action.(responseGetter); ok {
		if rw := getter.Response(); rw != nil {
			response.Helpers = rw.GetHelpers()
		}
	}

	return response
}
