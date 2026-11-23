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
	"html"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/json"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

func TestFormDisabledControlAndDurableCoverageReceipt(t *testing.T) {
	requireChromium(t)
	clearRecipeVariables(t)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "text/html")
		_, err := fmt.Fprint(writer, `<form id="workflow-form"><span id="workflow-label"></span><input disabled name="ignored" value="omit">
			<button type="reset" id="workflow-reset">Reset</button></form><output id="workflow-status" data-revision="0"></output>
			<div partial_src="/_piko/partials/task-list"></div><script>
			const host=document.querySelector('#workflow-label');host.validity={valid:false};
			host.attachShadow({mode:'open'}).innerHTML='<input id="native-label" required>';
			</script>`)
		assert.NoError(t, err)
	}))
	defer server.Close()
	ctx, session, closeBrowser, err := openBrowser(t.Context(), 15*time.Second)
	require.NoError(t, err)
	closeOnce := sync.OnceFunc(closeBrowser)
	t.Cleanup(closeOnce)
	require.NoError(t, session.NavigateURL(ctx, server.URL))
	for _, testCase := range []struct {
		action  string
		present bool
	}{
		{action: "true", present: true},
		{action: "document.querySelector('[name=ignored]').disabled=false", present: false},
		{action: "document.querySelector('[name=ignored]').disabled=true;document.querySelector('[name=ignored]').value='different'", present: false},
		{action: "document.querySelector('[name=ignored]').remove()", present: false},
	} {
		var actual casefile.FormState
		require.NoError(t, runBrowser(ctx, session, evaluate(testCase.action, nil), evaluate(formSnapshot, &actual)), testCase.action)
		assert.Equal(t, testCase.present, actual.DisabledControlPresent, testCase.action)
	}
	closeOnce()
	directory := t.TempDir()
	path := filepath.Join(directory, "counter.json")
	t.Setenv(envWorkflowDatabase, filepath.Join(directory, "missing.sqlite"))
	t.Setenv(envJSCoverage, filepath.Join(directory, "coverage.json"))
	err = formProbe(t.Context(), server.URL, path, `{"Steps":[{"Op":"reset"}]}`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "collecting runtime coverage")
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var actual casefile.FormWorkflowObservation
	require.NoError(t, json.Unmarshal(data, &actual))
	assert.Zero(t, actual.FailedStep)
	assert.True(t, strings.Contains(actual.DriverError, "direct SQLite"), actual.DriverError)
	assert.True(t, actual.Initial.DisabledControlPresent)
}

const (
	workflowTestPage = `<form id="workflow-form"><div id="workflow-label"></div><input disabled name="ignored" value="omit">
<button type="button" id="workflow-create">Create</button><button type="reset" id="workflow-reset">Reset</button></form>
<output id="workflow-status" data-revision="0"></output><ul partial_src="/_piko/partials/task-list">%s</ul><script>
const form = document.getElementById('workflow-form'), host = document.getElementById('workflow-label');
const status = document.getElementById('workflow-status'), list = document.querySelector('[partial_src]');
host.attachShadow({mode:'open'}).innerHTML = '<input id="native-label" required>';
const input = host.shadowRoot.querySelector('#native-label');
Object.defineProperty(host, 'validity', {get: () => input.validity});
form.addEventListener('reset', () => { input.value = ''; });
const call = async (name, body, headers) => {
  const response = await fetch('/_piko/actions/tasks.' + name, {method: 'POST', body, headers});
  const row = await response.json();
  Object.assign(status.dataset, {id: String(row.id), label: row.label, done: String(row.done)});
  list.innerHTML = await (await fetch('/_piko/partials/task-list')).text();
  status.dataset.revision = String(Number(status.dataset.revision) + 1);
};
document.getElementById('workflow-create').onclick = () => {
  if (!input.value) { host.dispatchEvent(new Event('invalid')); return; }
  call('Create', JSON.stringify({label: input.value}), {'Content-Type': 'application/json'});
};
list.addEventListener('click', event => {
  const row = event.target.closest('[data-fuzz-task]'), fields = new FormData();
  fields.set('id', row.dataset.fuzzTask);
  call(event.target.hasAttribute('data-toggle') ? 'Toggle' : 'Delete', fields);
});
</script>`
)

type workflowTestTask struct {
	label string
	id    int
	done  bool
}

type workflowTestApp struct {
	t        *testing.T
	database string
	tasks    []workflowTestTask
	mu       sync.Mutex
}

func (app *workflowTestApp) rows() string {
	var rows strings.Builder
	for _, task := range app.tasks {
		fmt.Fprintf(&rows, `<li data-fuzz-task="%d" data-done="%t"><span data-task-label>%s</span><button data-toggle>Toggle</button>`+
			`<button data-delete>Delete</button></li>`, task.id, task.done, html.EscapeString(task.label))
	}
	return rows.String()
}

func (app *workflowTestApp) store(query string, arguments ...any) {
	app.t.Helper()
	const write = `import sqlite3,sys
with sqlite3.connect(sys.argv[1]) as db: db.execute(sys.argv[2], tuple(sys.argv[3:]))`
	argv := []string{"-I", "-c", write, app.database, query}
	for _, argument := range arguments {
		argv = append(argv, fmt.Sprint(argument))
	}
	output, err := exec.CommandContext(app.t.Context(), "python3", argv...).CombinedOutput()
	assert.NoError(app.t, err, string(output))
}

func (app *workflowTestApp) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	app.mu.Lock()
	defer app.mu.Unlock()
	var err error
	switch request.URL.Path {
	case "/":
		writer.Header().Set("Content-Type", "text/html")
		_, err = fmt.Fprintf(writer, workflowTestPage, app.rows())
	case "/_piko/partials/task-list":
		writer.Header().Set("Content-Type", "text/html")
		_, err = fmt.Fprint(writer, app.rows())
	case "/_piko/actions/tasks.Create":
		var body struct {
			Label string `json:"label"`
		}
		assert.NoError(app.t, json.NewDecoder(request.Body).Decode(&body))
		task := workflowTestTask{id: len(app.tasks) + 1, label: body.Label}
		app.tasks = append(app.tasks, task)
		app.store("INSERT INTO tasks VALUES (?,?,0)", task.id, task.label)
		err = app.respond(writer, task)
	case "/_piko/actions/tasks.Toggle", "/_piko/actions/tasks.Delete":
		id, parseErr := strconv.Atoi(request.FormValue("id"))
		assert.NoError(app.t, parseErr)
		position := slices.IndexFunc(app.tasks, func(task workflowTestTask) bool { return task.id == id })
		if !assert.GreaterOrEqual(app.t, position, 0, "absent task %d", id) {
			http.NotFound(writer, request)
			return
		}
		task := app.tasks[position]
		if strings.HasSuffix(request.URL.Path, "Toggle") {
			task.done = !task.done
			app.tasks[position] = task
			app.store("UPDATE tasks SET done=1-done WHERE id=?", id)
		} else {
			app.tasks = slices.Delete(app.tasks, position, position+1)
			app.store("DELETE FROM tasks WHERE id=?", id)
		}
		err = app.respond(writer, task)
	default:
		http.NotFound(writer, request)
	}
	assert.NoError(app.t, err)
}

func (*workflowTestApp) respond(writer http.ResponseWriter, task workflowTestTask) error {
	writer.Header().Set("Content-Type", "application/json")
	_, err := fmt.Fprintf(writer, `{"id":%d,"label":%q,"done":%t}`, task.id, task.label, task.done)
	return err
}

func TestFormWorkflowModeRecordsEveryCommand(t *testing.T) {
	requireChromium(t)
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 unavailable")
	}
	clearRecipeVariables(t)
	directory := t.TempDir()
	app := &workflowTestApp{t: t, database: filepath.Join(directory, "tasks.sqlite")}
	app.store("CREATE TABLE tasks(id INTEGER,label TEXT,done INTEGER)")
	server := httptest.NewServer(app)
	defer server.Close()
	t.Setenv(envWorkflowDatabase, app.database)
	result := filepath.Join(directory, "workflow.json")
	recipe := `{"Steps":[{"Op":"edit","Value":"Milk"},{"Op":"create","Value":"Milk"},{"Op":"toggle","ID":1},` +
		`{"Op":"create","Value":"Tea"},{"Op":"delete","ID":1},{"Op":"reset"},{"Op":"invalid"}]}`
	require.Zero(t, execute(t.Context(), []string{server.URL, "--form-workflow", result, recipe}))

	data, err := os.ReadFile(result)
	require.NoError(t, err)
	var observed casefile.FormWorkflowObservation
	require.NoError(t, json.Unmarshal(data, &observed))
	require.Empty(t, observed.DriverError)
	assert.Equal(t, 7, observed.FailedStep)
	assert.Equal(t, "/", observed.BrowserPath)
	assert.Equal(t, "/_piko/partials/task-list", observed.PartialPath)
	assert.True(t, observed.Initial.DisabledControlPresent)
	assert.False(t, observed.Initial.Valid)
	require.Len(t, observed.Steps, 7)
	rows := func(values []casefile.FormRow) []casefile.FormRow {
		if len(values) == 0 {
			return nil
		}
		return values
	}
	for index, step := range observed.Steps {
		assert.True(t, step.FreshRead, "step %d", index)
		assert.True(t, step.StoreRead, "step %d", index)
		assert.Equal(t, rows(step.FreshRows), rows(step.StoreRows), "step %d", index)
		assert.Equal(t, rows(step.State.Rows), rows(step.FreshRows), "step %d", index)
	}
	assert.Equal(t, "Milk", observed.Steps[0].State.Input)
	assert.True(t, observed.Steps[0].State.Valid)

	created := observed.Steps[1]
	assert.Equal(t, 1, created.State.Revision)
	assert.Equal(t, &casefile.FormRow{ID: 1, Label: "Milk", Done: false}, created.State.Result)
	require.Len(t, created.Requests, 2)
	assert.Equal(t, casefile.FormRequest{Method: http.MethodPost, Path: "/_piko/actions/tasks.Create", Status: http.StatusOK,
		InputFields: `{"label":"Milk"}`, JSONResult: `{"id":1,"label":"Milk","done":false}`}, created.Requests[0])
	assert.Equal(t, "/_piko/partials/task-list", created.Requests[1].Path)

	toggled := observed.Steps[2]
	assert.Equal(t, []casefile.FormRow{{ID: 1, Label: "Milk", Done: true}}, toggled.FreshRows)
	assert.Equal(t, `{"id":["1"]}`, toggled.Requests[0].InputFields)
	assert.Equal(t, []casefile.FormRow{{ID: 2, Label: "Tea", Done: false}}, observed.Steps[4].FreshRows)
	assert.Empty(t, observed.Steps[5].State.Input)
	assert.Equal(t, 1, observed.Steps[6].State.InvalidEvents)
	assert.True(t, observed.FinalRead)
	assert.Equal(t, []casefile.FormRow{{ID: 2, Label: "Tea", Done: false}}, observed.FinalStore)
}
