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
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/json"

	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/tests/fuzz/internal/runtimecoverage"
)

const (
	counterScript = `class TestCounter extends HTMLElement {
  connectedCallback() {
    if (this.shadowRoot) return;
    const root = this.attachShadow({mode:'open'});
    root.innerHTML = '<output id="count">1</output><button id="increment">add</button><input id="client-input">' +
      '<span id="client-label"></span><ul id="client-list"><li>a</li><li>b</li></ul><button id="client-reverse">reverse</button>' +
      '<input type="checkbox" id="client-check"><select id="client-select"><option>one</option><option>two</option></select>';
    const count = root.querySelector('#count');
    root.querySelector('#increment').onclick = () => { count.textContent = String(Number(count.textContent) + 1); };
    root.querySelector('#client-reverse').onclick = () => { const list = root.querySelector('#client-list'); list.prepend(list.lastElementChild); };
    root.querySelector('#client-input').addEventListener('input', event => { root.querySelector('#client-label').textContent = event.target.value; });
  }
}
customElements.define('test-counter', TestCounter);`
)

func newCounterServer(t *testing.T) *httptest.Server {
	t.Helper()
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		var err error
		switch request.URL.Path {
		case "/":
			writer.Header().Set("Content-Type", "text/html")
			_, err = fmt.Fprint(writer, `<test-counter></test-counter><script src="/counter.js"></script>`)
		case "/counter.js":
			writer.Header().Set("Content-Type", "text/javascript")
			_, err = fmt.Fprint(writer, counterScript)
		case "/program":
			writer.Header().Set("Content-Type", "text/html")
			_, err = fmt.Fprint(writer, `<main id="fuzz-program"><output data-fuzz-output="sum" data-value="3">3</output>`+
				`<div hidden><output data-fuzz-output="secret">x</output></div></main>`)
		case "/outside":
			location, parseErr := url.Parse(server.URL)
			assert.NoError(t, parseErr)
			http.Redirect(writer, request, "http://localhost:"+location.Port()+"/program", http.StatusFound)
		default:
			http.NotFound(writer, request)
		}
		assert.NoError(t, err)
	}))
	t.Cleanup(server.Close)
	return server
}

func TestExecuteProbeDrivesActionsRoutesAndCoverage(t *testing.T) {
	requireChromium(t)
	clearRecipeVariables(t)
	server := newCounterServer(t)
	directory := t.TempDir()
	coverage := filepath.Join(directory, "coverage.json")
	t.Setenv(envJSCoverage, coverage)
	t.Setenv(envProgramPaths, `["/program","/outside","/program"]`)
	actions, err := json.Marshal([]casefile.ClientAction{
		{Op: casefile.SetInput, Value: "hello"},
		{Op: casefile.ReverseRows},
		{Op: casefile.ToggleCheck},
		{Op: casefile.SelectOption, Value: "last"},
	})
	require.NoError(t, err)
	result := filepath.Join(directory, "result.json")
	require.Zero(t, execute(t.Context(), []string{server.URL + "/", "test-counter", result, string(actions)}))

	data, err := os.ReadFile(result)
	require.NoError(t, err)
	var saved casefile.ClientTransition
	require.NoError(t, json.Unmarshal(data, &saved))
	assert.NotEmpty(t, saved.PrimaryBrowserVersion)
	require.Len(t, saved.Before, 1)
	require.Len(t, saved.After, 1)
	assert.Equal(t, "1", saved.Before[0].Count)
	assert.Equal(t, "2", saved.After[0].Count)
	require.Len(t, saved.Steps, 4)
	after := func(index int) casefile.ClientInstance {
		t.Helper()
		require.Len(t, saved.Steps[index].After, 1)
		return saved.Steps[index].After[0]
	}
	require.NotNil(t, after(0).Controls)
	assert.Equal(t, "hello", after(0).Controls.Input)
	assert.Equal(t, "hello", after(0).Label)
	assert.Equal(t, []string{"b", "a"}, after(1).Rows)
	require.NotNil(t, after(2).Controls.Form)
	assert.True(t, after(2).Controls.Form.Checked)
	assert.Equal(t, "two", after(3).Controls.Form.Choice)

	require.Len(t, saved.BrowserRoutes, 3)
	primary := saved.BrowserRoutes[0]
	assert.Equal(t, "/", primary.Path)
	assert.Equal(t, http.StatusOK, primary.Status)
	program := saved.BrowserRoutes[1]
	assert.Equal(t, "/program", program.Landed)
	assert.True(t, program.Program)
	assert.Equal(t, "main", program.ProgramRoot)
	require.Len(t, program.Output, 2)
	assert.Equal(t, "3", program.Output[0].Value)
	assert.False(t, program.Output[0].Hidden)
	assert.True(t, program.Output[1].Hidden)
	assert.Equal(t, "document landed outside the local application", saved.BrowserRoutes[2].DriverError)

	encoded, err := os.ReadFile(coverage)
	require.NoError(t, err)
	var profile runtimecoverage.Browser
	require.NoError(t, json.Unmarshal(encoded, &profile))
	require.Len(t, profile.Scripts, 1)
	assert.Equal(t, "/counter.js", profile.Scripts[0].Path)
	assert.Positive(t, profile.Functions.Covered)
}
