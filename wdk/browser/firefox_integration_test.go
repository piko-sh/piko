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

//go:build integration

package browser

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFirefoxSessionInputAndCancellation(t *testing.T) {
	if _, err := exec.LookPath("firefox"); err != nil {
		t.Skip("Firefox unavailable")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = fmt.Fprint(w, "<input id='field'>")
	}))
	t.Cleanup(server.Close)
	ctx, cancel := context.WithTimeoutCause(t.Context(), 30*time.Second, fmt.Errorf("firefox test budget exceeded"))
	defer cancel()
	session, err := OpenFirefoxSession(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = session.Close() })

	require.NoError(t, session.NavigateURL(ctx, server.URL))
	require.NoError(t, session.SetViewport(ctx, 811, 600))
	_, err = session.EvaluateString(ctx, "document.getElementById('field').focus(); ''")
	require.NoError(t, err)
	require.NoError(t, session.Key(ctx, "abc"))
	require.NoError(t, session.PressKey(ctx, "Home"))
	require.NoError(t, session.Key(ctx, "X"))
	result, err := session.EvaluateString(ctx, "Promise.resolve(document.getElementById('field').value + '/' + innerWidth)")
	require.NoError(t, err)
	assert.Equal(t, "Xabc/811", result)
	assert.NotEmpty(t, session.Version())

	started := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		close(started)
		_, err := session.EvaluateString(ctx, "new Promise(() => {})")
		done <- err
	}()
	<-started
	cancel()
	select {
	case err := <-done:
		require.Error(t, err, "pending evaluation completed after cancellation")
	case <-time.After(3 * time.Second):
		require.FailNow(t, "pending evaluation survived cancellation")
	}
	require.NoError(t, session.Close())
	_, err = os.Stat(session.profile)
	assert.ErrorIs(t, err, os.ErrNotExist, "private profile survived close")
}

func TestFirefoxCommandDeadline(t *testing.T) {
	if _, err := exec.LookPath("firefox"); err != nil {
		t.Skip("Firefox unavailable")
	}
	ctx, cancel := context.WithTimeoutCause(t.Context(), 30*time.Second, fmt.Errorf("firefox test budget exceeded"))
	defer cancel()
	session, err := OpenFirefoxSession(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = session.Close() })
	cause := fmt.Errorf("evaluation step budget exceeded")
	step, release := context.WithTimeoutCause(ctx, 200*time.Millisecond, cause)
	defer release()

	start := time.Now()
	_, err = session.EvaluateString(step, "new Promise(() => {})")

	require.ErrorIs(t, err, cause)
	require.ErrorIs(t, err, ErrFirefoxSessionBroken)
	assert.Less(t, time.Since(start), 5*time.Second)
	require.NoError(t, ctx.Err(), "the session context outlives the command")
	require.NoError(t, session.Close())
}
