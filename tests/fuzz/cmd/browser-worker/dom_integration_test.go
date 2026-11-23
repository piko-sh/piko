//go:build integration

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

package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/json"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

const (
	domTestPage = `<main><input id="field" value="ab"><button id="go" onclick="this.textContent='clicked'">go</button>
<div id="host"></div></main><script>
document.getElementById('host').attachShadow({mode:'open'}).innerHTML='<span id="inner">shadow</span>';
</script>`
)

func newDOMServer(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/dom" {
			http.NotFound(writer, request)
			return
		}
		writer.Header().Set("Content-Type", "text/html")
		_, err := fmt.Fprint(writer, domTestPage)
		assert.NoError(t, err)
	}))
	t.Cleanup(server.Close)
	return server
}

func domTestPlan() casefile.DOMPlan {
	field := func(id, host, selector, read string) casefile.DOMField {
		return casefile.DOMField{ID: id, Query: casefile.DOMQuery{Host: host, Selector: selector, Read: read}}
	}
	return casefile.DOMPlan{Path: "/dom", ReadyHost: "#host", ReadyElement: "#inner", Steps: []casefile.DOMStep{
		{ID: "mount", Op: "inspect", Fields: []casefile.DOMField{field("shadow", "#host", "#inner", "text"), field("tag", "", "#go", "tag")}},
		{ID: "narrow", Op: "viewport", Value: "narrow", Fields: []casefile.DOMField{field("visible", "", "#go", "visible")}},
		{ID: "press", Op: "click", Target: "#go", Fields: []casefile.DOMField{field("label", "", "#go", "text")}},
		{ID: "focus", Op: "focus", Target: "#field", Fields: []casefile.DOMField{field("focused", "", "#field", "focus")}},
		{ID: "end", Op: "key", Target: "#field", Value: "End", Fields: []casefile.DOMField{field("caret", "", "#field", "selection")}},
		{ID: "type", Op: "type", Target: "#field", Value: "Z", Fields: []casefile.DOMField{field("value", "", "#field", "value")}},
	}}
}

func assertDOMTestStates(t *testing.T, states []casefile.DOMState) {
	t.Helper()
	require.Len(t, states, 6)
	want := []map[string][]string{
		{"shadow": {"shadow"}, "tag": {"button"}},
		{"visible": {"true"}},
		{"label": {"clicked"}},
		{"focused": {"true"}},
		{"caret": {"2/2"}},
		{"value": {"abZ"}},
	}
	for index, state := range states {
		assert.Equal(t, "/dom", state.Path)
		assert.Equal(t, want[index], state.Values, state.Step)
	}
}

func TestDOMJourneyRunsEveryOperationInChromium(t *testing.T) {
	requireChromium(t)
	clearRecipeVariables(t)
	server := newDOMServer(t)
	ctx, session := openTestBrowser(t, 30*time.Second)
	trace := domJourney(ctx, session, server.URL, domTestPlan(), await(settleScript, nil))
	require.Empty(t, trace.DriverError)
	assert.Equal(t, "chromium", trace.Engine)
	assert.NotEmpty(t, trace.Version)
	assertDOMTestStates(t, trace.States)

	plan := domTestPlan()
	plan.Steps[2].Target = "#absent"
	failed := domJourney(ctx, session, server.URL, plan, await(settleScript, nil))
	assert.Len(t, failed.States, 2)
	assert.Contains(t, failed.DriverError, "press: ")
}

func TestFirefoxDOMModeWritesReceipt(t *testing.T) {
	if _, err := exec.LookPath("firefox"); err != nil {
		t.Skip("Firefox unavailable")
	}
	clearRecipeVariables(t)
	server := newDOMServer(t)
	raw, err := json.Marshal(domTestPlan())
	require.NoError(t, err)
	output := filepath.Join(t.TempDir(), "firefox.json")
	require.Zero(t, execute(t.Context(), []string{server.URL, "--firefox-dom", output, string(raw)}))
	data, err := os.ReadFile(output)
	require.NoError(t, err)
	var trace casefile.DOMTrace
	require.NoError(t, json.Unmarshal(data, &trace))
	require.Empty(t, trace.DriverError)
	assert.Equal(t, "firefox", trace.Engine)
	assert.NotEmpty(t, trace.Version)
	assertDOMTestStates(t, trace.States)
}
