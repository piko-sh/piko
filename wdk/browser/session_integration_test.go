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
	"bytes"
	"context"
	"errors"
	"image"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func openFixtureSession(t *testing.T) (*Session, *httptest.Server, context.Context) {
	t.Helper()
	if !chromeAvailable() {
		t.Skip("no local Chrome or Chromium executable")
	}
	server := httptest.NewServer(fixtureHandler())
	t.Cleanup(server.Close)
	ctx, cancel := context.WithTimeoutCause(t.Context(), 30*time.Second, errors.New("session test budget exceeded"))
	t.Cleanup(cancel)
	options := DefaultBrowserOptions()
	options.SocketDir = t.TempDir()
	session, err := OpenSession(ctx, options)
	require.NoError(t, err)
	t.Cleanup(func() { _ = session.Close() })
	session.serverURL = server.URL
	return session, server, ctx
}

func TestSessionPageOperations(t *testing.T) {
	session, _, ctx := openFixtureSession(t)

	require.NoError(t, session.Navigate(ctx, "/"))
	require.NoError(t, session.WaitReady(ctx, "#title"))
	require.NoError(t, session.SetViewport(ctx, 640, 480))
	width, err := session.EvalReturn(ctx, "innerWidth")
	require.NoError(t, err)
	assert.InDelta(t, 640, width, 0)

	require.NoError(t, session.SetViewportWithScale(ctx, 320, 240, 2))
	ratio, err := session.EvalReturn(ctx, "devicePixelRatio + ':' + innerWidth")
	require.NoError(t, err)
	assert.Equal(t, "2:320", ratio)

	require.NoError(t, session.Eval(ctx, "document.title = 'changed'"))
	title, err := session.EvalReturn(ctx, "document.title")
	require.NoError(t, err)
	assert.Equal(t, "changed", title)

	screenshot, err := session.ScreenshotViewport(ctx)
	require.NoError(t, err)
	config, _, err := image.DecodeConfig(bytes.NewReader(screenshot))
	require.NoError(t, err)
	assert.Equal(t, 640, config.Width, "the capture is in device pixels")

	require.NoError(t, session.Stop(ctx))
	require.NoError(t, session.InsertText(ctx, ""))
}

func TestSessionOperationErrors(t *testing.T) {
	session, _, ctx := openFixtureSession(t)
	require.NoError(t, session.Navigate(ctx, "/"))

	err := session.WaitReadyTimeout(ctx, "#absent", 50*time.Millisecond)
	require.Error(t, err)
	_, err = session.EvalReturn(ctx, "throw new Error('evaluation refused')")
	require.ErrorContains(t, err, "evaluating script")
	require.Error(t, session.NavigateURL(ctx, "http://[::1"))
	_, _, err = session.NavigateResponse(ctx, "http://[::1")
	require.ErrorContains(t, err, "navigating to")
	_, err = session.ResponseBody(ctx, "no-such-request")
	require.ErrorContains(t, err, "reading response body of request no-such-request")
	_, err = session.RequestPostData(ctx, "no-such-request")
	require.ErrorContains(t, err, "reading post data of request no-such-request")

	cause := errors.New("caller gave up")
	cancelled, cancel := context.WithCancelCause(ctx)
	cancel(cause)
	require.ErrorIs(t, session.SetViewportWithScale(cancelled, 10, 10, 1), context.Canceled)
	require.Error(t, session.ScriptOnNewDocument(cancelled, "1"))
	require.Error(t, session.Key(cancelled, "a"))
	require.Error(t, session.StartJSCoverage(cancelled))
	_, err = session.StopJSCoverage(cancelled, JSCoverageOptions{})
	require.Error(t, err)
	_, err = session.ObserveNetwork(cancelled, func(NetworkEvent) {})
	require.ErrorContains(t, err, "enabling network observation")
	_, err = session.NewTab(cancelled)
	require.ErrorContains(t, err, "opening browser tab")

	_, err = (&Session{}).NewTab(ctx)
	require.ErrorIs(t, err, errSessionHasNoPage)

	require.NoError(t, session.Close())
	require.NoError(t, session.Close(), "closing twice is harmless")
	require.Error(t, session.Eval(ctx, "1"))
	_, err = session.EvalReturn(ctx, "1")
	require.Error(t, err)
	_, err = session.ScreenshotViewport(ctx)
	require.Error(t, err)
}
