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

package browser

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/wdk/browser/internal/browser_provider_chromedp"
)

func sessionOnTab(tab context.Context) *Session {
	return &Session{page: &browser_provider_chromedp.IncognitoPage{Ctx: tab}}
}

func TestOperationCarriesCallerCause(t *testing.T) {
	session := sessionOnTab(t.Context())
	cause := errors.New("caller stopped")
	caller, cancel := context.WithCancelCause(t.Context())
	operation, release := session.operation(caller)
	defer release()

	cancel(cause)

	select {
	case <-operation.Done():
	case <-time.After(5 * time.Second):
		require.FailNow(t, "operation outlived its caller")
	}
	require.ErrorIs(t, context.Cause(operation), cause)
}

func TestOperationOnDoneCallerEndsAtOnce(t *testing.T) {
	session := sessionOnTab(t.Context())
	cause := errors.New("already over")
	caller, cancel := context.WithCancelCause(t.Context())
	cancel(cause)

	operation, release := session.operation(caller)
	defer release()

	require.Error(t, operation.Err())
	require.ErrorIs(t, context.Cause(operation), cause)
}

func TestOperationEndsWithTheTab(t *testing.T) {
	tabCause := errors.New("tab closed")
	tab, closeTab := context.WithCancelCause(t.Context())
	session := sessionOnTab(tab)
	operation, release := session.operation(t.Context())
	defer release()

	closeTab(tabCause)

	require.ErrorIs(t, context.Cause(operation), tabCause)
}

func TestOperationReportsEarliestDeadline(t *testing.T) {
	later := time.Now().Add(time.Hour)
	sooner := time.Now().Add(time.Minute)
	tests := []struct {
		tabDeadline  time.Time
		callDeadline time.Time
		want         time.Time
		name         string
		wantOK       bool
	}{
		{name: "neither", tabDeadline: time.Time{}, callDeadline: time.Time{}, want: time.Time{}, wantOK: false},
		{name: "caller only", tabDeadline: time.Time{}, callDeadline: sooner, want: sooner, wantOK: true},
		{name: "tab only", tabDeadline: sooner, callDeadline: time.Time{}, want: sooner, wantOK: true},
		{name: "caller sooner", tabDeadline: later, callDeadline: sooner, want: sooner, wantOK: true},
		{name: "tab sooner", tabDeadline: sooner, callDeadline: later, want: sooner, wantOK: true},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			tab := withOptionalDeadline(t, t.Context(), testCase.tabDeadline)
			caller := withOptionalDeadline(t, t.Context(), testCase.callDeadline)
			operation, release := sessionOnTab(tab).operation(caller)
			defer release()

			deadline, ok := operation.Deadline()

			assert.Equal(t, testCase.wantOK, ok)
			assert.True(t, testCase.want.Equal(deadline), "deadline %v, want %v", deadline, testCase.want)
			derived, cancel := context.WithTimeoutCause(operation, 2*time.Hour, errors.New("unused"))
			defer cancel()
			derivedDeadline, derivedOK := derived.Deadline()
			assert.True(t, derivedOK)
			if ok {
				assert.True(t, testCase.want.Equal(derivedDeadline), "a longer derived timeout keeps the operation deadline")
			}
		})
	}
}

func TestOperationReleaseLeavesCallerAndTabRunning(t *testing.T) {
	tab, closeTab := context.WithCancel(t.Context())
	defer closeTab()
	caller, cancel := context.WithCancel(t.Context())
	defer cancel()
	operation, release := sessionOnTab(tab).operation(caller)

	release()

	require.ErrorIs(t, operation.Err(), context.Canceled)
	require.NoError(t, caller.Err())
	require.NoError(t, tab.Err())
}

func TestActionContextUsesTheOperation(t *testing.T) {
	session := sessionOnTab(t.Context())
	session.serverURL = "http://127.0.0.1:1"
	action, release := session.actionCtx(t.Context())

	assert.Equal(t, session.serverURL, action.ServerURL)
	require.NoError(t, action.Ctx.Err())
	release()
	require.Error(t, action.Ctx.Err())
}

func TestCloseWithoutPageIsIdempotent(t *testing.T) {
	session := &Session{}

	require.NoError(t, session.Close())
	require.NoError(t, session.Close())
}

func withOptionalDeadline(t *testing.T, parent context.Context, deadline time.Time) context.Context {
	t.Helper()
	if deadline.IsZero() {
		return parent
	}
	ctx, cancel := context.WithDeadlineCause(parent, deadline, errors.New("test deadline"))
	t.Cleanup(cancel)
	return ctx
}
