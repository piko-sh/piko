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
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSessionReadsBrowserInitiatedForm(t *testing.T) {
	if testing.Short() {
		t.Skip("starts a local browser")
	}
	if !chromeAvailable() {
		t.Skip("no local Chrome or Chromium executable")
	}
	received := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			http.SetCookie(w, &http.Cookie{Name: "form-session", Value: "local-fixture", Path: "/", HttpOnly: true})
			w.Header().Set("Content-Type", "text/html")
			_, _ = fmt.Fprint(w, `<input id="field"><script>
const field=document.querySelector("#field"),inputs=[];let keys=0;
field.addEventListener("keydown",()=>keys++); field.addEventListener("input",e=>inputs.push(e.inputType));
</script>`)
		case "/capture":
			body, err := io.ReadAll(io.LimitReader(r.Body, 257))
			assert.NoError(t, err)
			received <- string(body)
			w.WriteHeader(http.StatusCreated)
			_, _ = fmt.Fprint(w, "local receipt")
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	ctx, cancel := context.WithTimeoutCause(t.Context(), 20*time.Second, errors.New("network observation test budget exceeded"))
	defer cancel()
	options := DefaultBrowserOptions()
	options.SocketDir = t.TempDir()
	session, err := OpenSession(ctx, options)
	require.NoError(t, err)
	t.Cleanup(func() { _ = session.Close() })
	events := make(chan NetworkEvent, 128)
	stop, err := session.ObserveNetwork(ctx, func(event NetworkEvent) {
		select {
		case events <- event:
		case <-ctx.Done():
		}
	})
	require.NoError(t, err)
	defer stop()
	require.NoError(t, session.NavigateURL(ctx, server.URL))
	require.NoError(t, session.Evaluate(ctx, "field.focus()", nil))
	require.NoError(t, session.InsertText(ctx, "é + 🍒"))
	var edit struct {
		Value  string
		Keys   int
		Inputs []string
	}
	require.NoError(t, session.Evaluate(ctx, "({Value:field.value,Keys:keys,Inputs:inputs.slice()})", &edit))
	assert.Equal(t, "é + 🍒", edit.Value)
	assert.Zero(t, edit.Keys)
	assert.Equal(t, []string{"insertText"}, edit.Inputs)
	require.NoError(t, session.Evaluate(ctx, "field.select()", nil))
	require.NoError(t, session.PressKey(ctx, "Backspace"))
	require.NoError(t, session.Evaluate(ctx, "({Value:field.value,Keys:keys,Inputs:inputs.slice()})", &edit))
	assert.Empty(t, edit.Value)
	assert.Equal(t, 1, edit.Keys)
	assert.Equal(t, []string{"insertText", "deleteContentBackward"}, edit.Inputs)
	const form = "label=local+input"
	require.NoError(t, session.Await(ctx, `fetch("/capture",{method:"POST",body:new URLSearchParams({label:"local input"})}).then(r=>r.text())`, nil))
	select {
	case body := <-received:
		assert.Equal(t, form, body)
	case <-ctx.Done():
		require.FailNow(t, "server did not receive the form", context.Cause(ctx))
	}
	require.NoError(t, session.Evaluate(ctx, "setTimeout(()=>{throw new Error('fixture uncaught exception')},0); true", nil))
	ids := map[RequestID]bool{}
	var request NetworkEvent
	var status int
	var exception bool
	for request.ID == "" || !ids[request.ID] || status == 0 || !exception {
		select {
		case event := <-events:
			switch event.Kind {
			case RequestStarted:
				if event.URL == server.URL+"/capture" {
					request = event
				}
			case RequestCookies:
				for _, cookie := range event.Cookies {
					if cookie.Name == "form-session" && !cookie.Blocked {
						ids[event.ID] = true
					}
				}
			case ResponseReceived:
				if event.URL == server.URL+"/capture" && event.ID == request.ID {
					status = event.Status
				}
			case JavaScriptException:
				exception = strings.Contains(event.Exception, "fixture uncaught exception")
			}
		case <-ctx.Done():
			require.FailNowf(t, "missing browser notification", "request=%+v status=%d cookie=%t exception=%t", request, status, ids[request.ID], exception)
		}
	}
	assert.Equal(t, "script", request.Initiator)
	assert.Equal(t, "POST", request.Method)
	assert.True(t, strings.HasPrefix(request.ContentType, "application/x-www-form-urlencoded"), request.ContentType)
	assert.Equal(t, http.StatusCreated, status)
	post, err := session.RequestPostData(ctx, request.ID)
	require.NoError(t, err)
	assert.Equal(t, form, string(post))
	body, err := session.ResponseBody(ctx, request.ID)
	require.NoError(t, err)
	assert.Equal(t, "local receipt", string(body))
}

func TestObserverStopKeepsTab(t *testing.T) {
	if testing.Short() {
		t.Skip("starts a local browser")
	}
	if !chromeAvailable() {
		t.Skip("no local Chrome or Chromium executable")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, `<p id="page">%s</p><script>setTimeout(()=>{window.ready=true},50)</script>`, r.URL.Path)
	}))
	t.Cleanup(server.Close)
	ctx, cancel := context.WithTimeoutCause(t.Context(), 20*time.Second, errors.New("observation restart test budget exceeded"))
	defer cancel()
	options := DefaultBrowserOptions()
	options.SocketDir = t.TempDir()
	session, err := OpenSession(ctx, options)
	require.NoError(t, err)
	t.Cleanup(func() { _ = session.Close() })

	first, err := session.ObserveNetwork(ctx, func(NetworkEvent) {})
	require.NoError(t, err)
	second := make(chan NetworkEvent, 64)
	stopSecond, err := session.ObserveNetwork(ctx, func(event NetworkEvent) {
		select {
		case second <- event:
		default:
		}
	})
	require.NoError(t, err)
	defer stopSecond()
	require.NoError(t, session.NavigateURL(ctx, server.URL+"/first"))
	first()

	status, _, err := session.NavigateResponse(ctx, server.URL+"/second")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, status)
	require.NoError(t, session.WaitForExpression(ctx, "window.ready === true", 5*time.Second))
	var text string
	require.NoError(t, session.Evaluate(ctx, `document.querySelector("#page").textContent`, &text))
	assert.Equal(t, "/second", text)

	for {
		select {
		case event := <-second:
			if event.Kind == ResponseReceived && strings.HasSuffix(event.URL, "/second") {
				return
			}
		case <-ctx.Done():
			require.FailNow(t, "a remaining observer stopped receiving events", context.Cause(ctx))
		}
	}
}
