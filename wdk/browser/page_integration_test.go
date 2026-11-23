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
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/wdk/browser/internal/browser_provider_chromedp"
	"piko.sh/safedisk"
)

const (
	fixtureDocument = `<!doctype html>
<html><head><title>Fixture</title></head>
<body>
<h1 id="title" class="big" data-role="heading" style="color: rgb(255, 0, 0)">Hello</h1>
<p id="hidden" style="display:none">secret</p>
<input id="name" value="">
<input id="check" type="checkbox">
<input id="upload" type="file">
<button id="enabled" onclick="this.dataset.clicks = (Number(this.dataset.clicks || 0) + 1)">on</button>
<button id="disabled" disabled>off</button>
<button id="ask" onclick="this.dataset.answer = String(prompt('name?'))">ask</button>
<button id="confirm" onclick="this.dataset.answer = String(confirm('sure?'))">confirm</button>
<form id="form" onsubmit="event.preventDefault(); document.body.dataset.submitted = 'yes'"><input name="q"></form>
<div id="edit" contenteditable="true">abc<span id="tail">def</span></div>
<div id="scroller" style="height:40px;overflow:auto"><div style="height:400px">tall</div></div>
<div id="drag" draggable="true" style="width:40px;height:40px;background:#00f">d</div>
<div id="drop" style="width:80px;height:80px;background:#0f0">drop</div>
<iframe id="frame" srcdoc="<button id='inner' onclick='this.textContent=&quot;pressed&quot;'>in</button><input id='field'>"></iframe>
<script>console.log('fixture ready'); console.warn('fixture warning');</script>
</body></html>`
)

type recordingTB struct {
	testing.TB
	failures  []string
	mu        sync.Mutex
	expecting atomic.Bool
}

func (r *recordingTB) Helper() {}

func (r *recordingTB) Error(args ...any) { r.record(fmt.Sprint(args...)) }

func (r *recordingTB) Errorf(format string, args ...any) { r.record(fmt.Sprintf(format, args...)) }

func (r *recordingTB) Fatal(args ...any) { r.fatal(fmt.Sprint(args...)) }

func (r *recordingTB) Fatalf(format string, args ...any) { r.fatal(fmt.Sprintf(format, args...)) }

func (r *recordingTB) fatal(message string) {
	if !r.expecting.Load() {
		r.TB.Fatal(message)
	}
	r.record(message)
	runtime.Goexit()
}

func (r *recordingTB) record(message string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.failures = append(r.failures, message)
}

func (r *recordingTB) attempt(step func()) []string {
	r.expecting.Store(true)
	defer r.expecting.Store(false)
	done := make(chan struct{})
	go func() {
		defer close(done)
		step()
	}()
	<-done
	return r.take()
}

func (r *recordingTB) take() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	failures := r.failures
	r.failures = nil
	return failures
}

type recordedStep struct {
	action string
	failed bool
}

type recordingRunner struct {
	steps  []recordedStep
	before int
	waits  int
	closed bool
}

func (*recordingRunner) Start(string) error { return nil }

func (r *recordingRunner) BeforeStep(string, string) { r.before++ }

func (r *recordingRunner) AfterStep(action, _ string, failed bool, _ time.Duration) {
	r.steps = append(r.steps, recordedStep{action: action, failed: failed})
}

func (r *recordingRunner) WaitForContinue() { r.waits++ }

func (r *recordingRunner) Close() { r.closed = true }

func fixtureHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/second", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprint(w, `<!doctype html><title>Second</title><p id="second">second page</p>`)
	})
	mux.HandleFunc("/api", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Seen-Auth", r.Header.Get("Authorization"))
		_, _ = fmt.Fprintf(w, `{"agent":%q,"custom":%q}`, r.UserAgent(), r.Header.Get("X-Custom"))
	})
	mux.HandleFunc("/download", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Content-Disposition", "attachment; filename=report.txt")
		_, _ = fmt.Fprint(w, "report body")
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("X-Fixture", "yes")
		_, _ = fmt.Fprint(w, fixtureDocument)
	})
	return mux
}

func newFixturePage(t *testing.T) (*Page, *recordingTB, *httptest.Server) {
	t.Helper()
	if !chromeAvailable() {
		t.Skip("no local Chrome or Chromium executable")
	}
	server := httptest.NewServer(fixtureHandler())
	t.Cleanup(server.Close)
	owner, err := browser_provider_chromedp.NewBrowser(browser_provider_chromedp.BrowserOptions{Headless: true})
	require.NoError(t, err)
	t.Cleanup(owner.Close)
	sourceDirectory := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(sourceDirectory, "upload.txt"), []byte("upload"), 0o600))
	source, err := safedisk.NewNoOpSandbox(sourceDirectory, safedisk.ModeReadOnly)
	require.NoError(t, err)
	t.Cleanup(func() { _ = source.Close() })
	options := defaultHarnessOptions()
	options.outputDir = t.TempDir()
	options.interactive = false
	harness := &Harness{browser: owner, serverURL: server.URL, srcSandbox: source, opts: options}
	recorder := &recordingTB{TB: t}
	page, err := harness.newPage(recorder)
	require.NoError(t, err)
	t.Cleanup(page.Close)
	return page, recorder, server
}

func expirePage(t *testing.T, page *Page) {
	t.Helper()
	expired, cancel := context.WithDeadline(page.incognitoPage.Ctx, time.Now().Add(-time.Second))
	t.Cleanup(cancel)
	page.incognitoPage.Ctx = expired
}

func cancelPage(t *testing.T, page *Page) {
	t.Helper()
	cancelled, cancel := context.WithCancel(page.incognitoPage.Ctx)
	cancel()
	page.incognitoPage.Ctx = cancelled
}

type pageStep struct {
	run  func()
	name string
}

func requireEachStepFails(t *testing.T, recorder *recordingTB, steps []pageStep) {
	t.Helper()
	for _, step := range steps {
		start := time.Now()
		failures := recorder.attempt(step.run)
		require.NotEmpty(t, failures, "%s reported no failure", step.name)
		assert.Contains(t, failures[0], step.name, "failure message names the call")
		assert.Less(t, time.Since(start), 3*time.Second, "%s did not fail promptly", step.name)
	}
}

func TestPageNavigationAndQueries(t *testing.T) {
	page, recorder, server := newFixturePage(t)
	runner := &recordingRunner{}
	page.interactiveRunner = runner

	page.Navigate("/").WaitFor("#title").WaitForText("#title", "Hello").WaitStable()
	page.WaitForVisible("#title", WithTimeout(time.Second)).WaitForNotVisible("#hidden")
	page.WaitForEnabled("#enabled").WaitForDisabled("#disabled").WaitForNotPresent("#absent")
	require.Empty(t, recorder.take())
	assert.Equal(t, "Fixture", page.Title())
	assert.Equal(t, server.URL+"/", page.URL())

	assert.Equal(t, "heading", page.GetAttribute("#title", "data-role"))
	assert.Empty(t, page.GetAttribute("#title", "data-missing"))
	assert.Equal(t, "title", page.GetAttributes("#title")["id"])
	page.SetAttribute("#title", "data-state", "set")
	assert.Equal(t, "set", page.GetAttribute("#title", "data-state"))
	page.RemoveAttribute("#title", "data-state")
	assert.Empty(t, page.GetAttribute("#title", "data-state"))
	assert.Positive(t, page.GetDimensions("#drop").Width)

	page.Eval("document.body.dataset.flag = 'on'")
	assert.Equal(t, "on", page.EvalReturn("document.body.dataset.flag"))
	assert.Equal(t, "resolved", page.EvalReturn("Promise.resolve('resolved')"))
	page.ScrollIntoView("#drop").Wait(time.Millisecond)
	assert.Contains(t, page.CaptureDOM("#title"), "Hello")
	require.Empty(t, recorder.take())

	page.DragAndDrop("#drag", "#drop").DragTo("#drag", 5, 5).DragByOffset("#drag", 1, 1).DragAndDropHTML5("#drag", "#drop")
	require.Empty(t, recorder.take())

	page.Navigate("/second").WaitFor("#second").Reload().Stop()
	assert.Equal(t, "Second", page.Title())
	require.Empty(t, recorder.take())

	assert.Positive(t, runner.before)
	assert.Equal(t, runner.before, runner.waits, "every action waits for the runner")
	assert.Contains(t, runner.steps, recordedStep{action: "Navigate", failed: false})
	require.Len(t, recorder.attempt(func() { page.WaitFor("#absent") }), 1)
	assert.Contains(t, runner.steps, recordedStep{action: "WaitFor", failed: true})
}

func TestPageWaitsReportTimeouts(t *testing.T) {
	page, recorder, _ := newFixturePage(t)
	page.Navigate("/")
	short := WithTimeout(100 * time.Millisecond)
	requireEachStepFails(t, recorder, []pageStep{
		{name: "WaitForVisible", run: func() { page.WaitForVisible("#hidden", short) }},
		{name: "WaitForNotVisible", run: func() { page.WaitForNotVisible("#title", short) }},
		{name: "WaitForEnabled", run: func() { page.WaitForEnabled("#disabled", short) }},
		{name: "WaitForDisabled", run: func() { page.WaitForDisabled("#enabled", short) }},
		{name: "WaitForNotPresent", run: func() { page.WaitForNotPresent("#title", short) }},
	})
}

func TestPageConsoleCapture(t *testing.T) {
	page, recorder, _ := newFixturePage(t)
	page.Navigate("/")
	require.Eventually(t, func() bool {
		return strings.Contains(strings.Join(page.ConsoleLogs(), "\n"), "fixture ready")
	}, 5*time.Second, 20*time.Millisecond)

	levels := page.ConsoleLogsWithLevel()
	require.NotEmpty(t, levels)
	assert.False(t, page.HasConsoleErrors())
	assert.Empty(t, page.ConsoleErrors())
	page.AssertConsole().HasMessage("fixture ready").HasLog("fixture ready").HasWarning("fixture warning").NoErrors()
	page.AssertNoConsoleErrors()
	require.Empty(t, recorder.take())

	page.AssertNoConsoleWarnings()
	page.AssertConsole().NoWarnings().HasError("never logged")
	assert.Len(t, recorder.take(), 3)

	page.Eval("console.error('fixture failure')")
	require.Eventually(t, page.HasConsoleErrors, 5*time.Second, 20*time.Millisecond)
	assert.NotEmpty(t, page.ConsoleErrors())
	page.AssertNoConsoleErrors()
	page.AssertConsole().NoErrors().HasLog("never logged").HasWarning("never logged").HasMessage("never logged")
	assert.Len(t, recorder.take(), 5)
	page.ClearConsole()
	assert.Empty(t, page.ConsoleLogs())
}

func TestPageMatchGolden(t *testing.T) {
	page, recorder, _ := newFixturePage(t)
	t.Chdir(t.TempDir())
	page.Navigate("/")

	t.Setenv("PIKO_UPDATE_GOLDEN", "1")
	page.MatchGolden("#title", "title")
	t.Setenv("PIKO_UPDATE_GOLDEN", "")
	page.MatchGolden("#title", "title")
	require.Empty(t, recorder.take())

	page.SetAttribute("#title", "data-changed", "yes").MatchGolden("#title", "title")
	failures := recorder.take()
	require.Len(t, failures, 1)
	assert.Contains(t, failures[0], "golden file mismatch")
}

func TestPageRecoversAnUnresponsivePage(t *testing.T) {
	page, recorder, _ := newFixturePage(t)
	stale := page.incognitoPage
	tab := stale.Ctx
	expirePage(t, page)
	page.Navigate("/")
	require.Empty(t, recorder.take())
	assert.NotSame(t, stale, page.incognitoPage)
	assert.Error(t, tab.Err(), "the replaced tab is closed")
	assert.Equal(t, "Fixture", page.Title())

	stale = page.incognitoPage
	expirePage(t, page)
	page.WaitFor("#title")
	require.Empty(t, recorder.take())
	assert.NotSame(t, stale, page.incognitoPage)
	assert.Equal(t, "Fixture", page.Title())
}

func TestPageReportsFailures(t *testing.T) {
	page, recorder, _ := newFixturePage(t)
	cancelPage(t, page)
	requireEachStepFails(t, recorder, []pageStep{
		{name: "Reload", run: func() { page.Reload() }},
		{name: "Back", run: func() { page.Back() }},
		{name: "Forward", run: func() { page.Forward() }},
		{name: "Stop", run: func() { page.Stop() }},
		{name: "Title", run: func() { page.Title() }},
		{name: "URL", run: func() { page.URL() }},
		{name: "ScrollIntoView", run: func() { page.ScrollIntoView("#title") }},
		{name: "GetAttribute", run: func() { page.GetAttribute("#title", "id") }},
		{name: "GetAttributes", run: func() { page.GetAttributes("#title") }},
		{name: "SetAttribute", run: func() { page.SetAttribute("#title", "id", "x") }},
		{name: "RemoveAttribute", run: func() { page.RemoveAttribute("#title", "id") }},
		{name: "GetDimensions", run: func() { page.GetDimensions("#title") }},
		{name: "Eval", run: func() { page.Eval("1") }},
		{name: "EvalReturn", run: func() { page.EvalReturn("1") }},
		{name: "DragAndDrop", run: func() { page.DragAndDrop("#drag", "#drop") }},
		{name: "DragTo", run: func() { page.DragTo("#drag", 1, 1) }},
		{name: "DragByOffset", run: func() { page.DragByOffset("#drag", 1, 1) }},
		{name: "DragAndDropHTML5", run: func() { page.DragAndDropHTML5("#drag", "#drop") }},
	})
}
