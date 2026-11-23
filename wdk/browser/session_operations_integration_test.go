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
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestOpenSessionBrowserOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("launches Chrome")
	}
	if !chromeAvailable() {
		t.Skip("no local Chrome or Chromium executable")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/start":
			http.Redirect(w, r, "/app", http.StatusFound)
			return
		case "/missing":
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprint(w, `<!doctype html><script>window.observedStartup = window.early === 42</script><input id="name"><input id="check" type="checkbox"><select id="list" multiple><option>a</option><option>b</option><option>c</option></select><button id="go" onclick="this.dataset.clicked=event.isTrusted">go</button>`)
	}))
	defer server.Close()
	ctx, cancel := context.WithTimeoutCause(t.Context(), 20*time.Second, errors.New("browser operations test budget exceeded"))
	defer cancel()

	socketDir, err := os.MkdirTemp("", "wdk-sock-")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(socketDir) }()
	profilesRoot := t.TempDir()
	if runtime.GOOS != "windows" {
		t.Setenv("TMPDIR", profilesRoot)
	}
	opts := DefaultBrowserOptions()
	opts.SocketDir = socketDir
	s, err := OpenSession(ctx, opts)
	require.NoError(t, err)
	defer s.Close()
	profiles, err := filepath.Glob(filepath.Join(os.TempDir(), "piko-chromedp-*"))
	require.NoError(t, err)
	var ownedProfile string
	if runtime.GOOS != "windows" {
		require.Len(t, profiles, 1)
		ownedProfile = profiles[0]
	}
	if runtime.GOOS == "linux" {
		socket, err := os.Readlink(filepath.Join(ownedProfile, "SingletonSocket"))
		require.NoError(t, err)
		relative, err := filepath.Rel(socketDir, socket)
		require.NoError(t, err)
		require.True(t, filepath.IsLocal(relative), "socket %q outside %q", socket, socketDir)
	}
	require.NoError(t, s.ScriptOnNewDocument(ctx, "window.early = 42"))
	status, landed, err := s.NavigateResponse(ctx, server.URL+"/start")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, status)
	require.Equal(t, server.URL+"/app", landed)
	require.NoError(t, s.WaitForExpression(ctx, "window.observedStartup", 0))
	var value string
	require.NoError(t, s.Await(ctx, "Promise.resolve('ready')", &value))
	require.Equal(t, "ready", value)
	require.ErrorContains(t, s.Await(ctx, "Promise.reject(new Error('ordinary rejection'))", &value), "ordinary rejection")
	require.NoError(t, s.Click(ctx, "#go"))
	require.NoError(t, s.WaitForExpression(ctx, "document.querySelector('#go').dataset.clicked === 'true'", time.Second))
	require.NoError(t, s.Click(ctx, "#name"))
	require.NoError(t, s.Key(ctx, "HomeΔ🦊👩‍🔬 ਇ"))
	require.NoError(t, s.Evaluate(ctx, "document.querySelector('#name').value", &value))
	require.Equal(t, "HomeΔ🦊👩‍🔬 ਇ", value)
	require.NoError(t, s.Evaluate(ctx, "document.querySelector('#check').focus()", nil))
	require.NoError(t, s.PressKey(ctx, "Space"))
	require.NoError(t, s.WaitForExpression(ctx, "document.querySelector('#check').checked", time.Second))
	require.NoError(t, s.Evaluate(ctx, "document.querySelector('#list').focus()", nil))
	require.NoError(t, s.PressKey(ctx, "Home"))
	require.NoError(t, s.PressKey(ctx, "ArrowDown", "Shift"))
	require.NoError(t, s.WaitForExpression(ctx, "Array.from(document.querySelector('#list').selectedOptions, o => o.text).join(',') === 'a,b'", time.Second))
	require.NoError(t, s.Evaluate(ctx, "document.cookie = 'session=ordinary; path=/'; sessionStorage.setItem('onlyTab', 'original')", nil))
	tab, err := s.NewTab(ctx)
	require.NoError(t, err)
	defer tab.Close()
	require.NoError(t, tab.NavigateURL(ctx, server.URL+"/sibling"))
	require.NoError(t, tab.Evaluate(ctx, "document.cookie", &value))
	require.Contains(t, value, "session=ordinary")
	require.NoError(t, tab.WaitForExpression(ctx, "!sessionStorage.getItem('onlyTab')", time.Second))
	status, landed, err = tab.NavigateResponse(ctx, server.URL+"/missing")
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, status)
	require.Equal(t, server.URL+"/missing", landed)
	status, landed, err = tab.NavigateResponse(ctx, "about:blank")
	require.NoError(t, err)
	require.Zero(t, status)
	require.Equal(t, "about:blank", landed)
	require.NoError(t, tab.Close())
	shortCause := errors.New("short step budget exceeded")
	short, release := context.WithTimeoutCause(ctx, 70*time.Millisecond, shortCause)
	defer release()
	start := time.Now()
	require.ErrorIs(t, s.WaitForExpression(short, "false", 0), shortCause)
	require.Less(t, time.Since(start), 2*time.Second)
	for range 10 {
		budget, stop := context.WithTimeoutCause(ctx, 30*time.Millisecond, shortCause)
		require.ErrorIs(t, s.WaitForExpression(budget, "false", 0), shortCause)
		stop()
		require.ErrorIs(t, s.WaitForExpression(ctx, "false", 30*time.Millisecond), errWaitTimeout)
	}
	require.Error(t, s.Evaluate(short, "1", nil))
	require.NoError(t, s.Evaluate(ctx, "window.location.pathname + '/' + sessionStorage.getItem('onlyTab')", &value))
	require.Equal(t, "/app/original", value)
	h := &Harness{browser: s.owner, serverURL: server.URL}
	harnessTab, err := h.NewSession()
	require.NoError(t, err)
	require.NoError(t, harnessTab.Navigate(ctx, "/app"))
	require.NoError(t, harnessTab.Close())
	require.NoError(t, s.Evaluate(ctx, "window.observedStartup", nil))
	cancel()
	require.Error(t, s.Evaluate(t.Context(), "1", nil), "the session ends with the context that opened it")
	if ownedProfile != "" {
		require.Eventually(t, func() bool {
			_, err := os.Stat(ownedProfile)
			return errors.Is(err, fs.ErrNotExist)
		}, 4*time.Second, 10*time.Millisecond)
	}
	require.NoError(t, s.Close())
	profiles, err = filepath.Glob(filepath.Join(profilesRoot, "piko-chromedp-*"))
	require.NoError(t, err)
	require.Empty(t, profiles, "closing the session removes its profile")
}

func chromeAvailable() bool {
	for _, name := range []string{"google-chrome", "chromium", "chromium-browser"} {
		if _, err := exec.LookPath(name); err == nil {
			return true
		}
	}
	return false
}
