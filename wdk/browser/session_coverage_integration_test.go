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
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"unicode/utf16"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSCoverageWindowInChromium(t *testing.T) {
	if !chromeAvailable() {
		t.Skip("no local Chrome or Chromium executable")
	}
	const source = "/* 🐦 */\r\nfunction used() { return 23 }\r\nfunction unused() { return 0 }\r\n"
	mux := http.NewServeMux()
	mux.HandleFunc("/app.js", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		_, err := io.WriteString(w, source)
		assert.NoError(t, err)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, err := io.WriteString(w, `<script src="/app.js"></script><script>used(); used()</script>`)
		assert.NoError(t, err)
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	ctx, cancel := context.WithTimeoutCause(t.Context(), 20*time.Second, errors.New("coverage test budget exceeded"))
	defer cancel()
	opts := DefaultBrowserOptions()
	opts.SocketDir = t.TempDir()
	session, err := OpenSession(ctx, opts)
	require.NoError(t, err)
	t.Cleanup(func() { _ = session.Close() })

	limit := JSCoverageOptions{IncludeURL: func(raw string) bool { return raw == server.URL+"/app.js" }}
	usedStart := int64(len(utf16.Encode([]rune(source[:strings.Index(source, "function used")]))))
	for window, wantCount := range []int64{2, 1} {
		require.NoError(t, session.StartJSCoverage(ctx), "start window %d", window)
		if window == 0 {
			require.NoError(t, session.NavigateURL(ctx, server.URL))
		} else {
			require.NoError(t, session.Evaluate(ctx, "used()", nil))
		}
		reports, err := session.StopJSCoverage(ctx, limit)
		require.NoError(t, err, "stop window %d", window)
		require.Len(t, reports, 1, "window %d scripts", window)
		assert.Equal(t, fmt.Sprintf("%x", sha256.Sum256([]byte(source))), reports[0].SHA256)

		var used *JSFunctionCoverage
		for index := range reports[0].Functions {
			if reports[0].Functions[index].Name == "used" {
				used = &reports[0].Functions[index]
			}
		}
		require.NotNil(t, used, "window %d missing used: %+v", window, reports[0].Functions)
		require.NotEmpty(t, used.Ranges)
		assert.Equal(t, usedStart, used.Ranges[0].Start, "window %d", window)
		assert.Equal(t, wantCount, used.Ranges[0].Count, "window %d", window)
	}
}

func TestCoverageSkipsDebugger(t *testing.T) {
	if !chromeAvailable() {
		t.Skip("no local Chrome or Chromium executable")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, err := io.WriteString(w, `<script>function halt() { debugger; return 7 }</script><script>debugger; window.loaded = true</script>`)
		assert.NoError(t, err)
	}))
	t.Cleanup(server.Close)
	ctx, cancel := context.WithTimeoutCause(t.Context(), 20*time.Second, errors.New("debugger coverage test budget exceeded"))
	defer cancel()
	opts := DefaultBrowserOptions()
	opts.SocketDir = t.TempDir()
	session, err := OpenSession(ctx, opts)
	require.NoError(t, err)
	t.Cleanup(func() { _ = session.Close() })

	require.NoError(t, session.StartJSCoverage(ctx))
	require.NoError(t, session.NavigateURL(ctx, server.URL))
	step, release := context.WithTimeoutCause(ctx, 5*time.Second, errors.New("page paused at a debugger statement"))
	defer release()
	var loaded bool
	require.NoError(t, session.Evaluate(step, "window.loaded === true", &loaded))
	assert.True(t, loaded)
	var value int
	require.NoError(t, session.Evaluate(step, "halt()", &value))
	assert.Equal(t, 7, value)
	_, err = session.StopJSCoverage(step, JSCoverageOptions{IncludeURL: func(string) bool { return false }})
	require.NoError(t, err)
}
