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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

const (
	intakeTestPage = `<script type="module" src="/pk/actions.gen.js"></script>
<form data-case="quota" action="/_piko/actions/intake.Submit" method="post"><input name="label" value="Quota"><input name="_captcha_token" value="ticket"></form>
<script>window.piko = {form: {data: form => new FormData(form)}};</script>`
	intakeTestModule = `const post = (name, body) => fetch('/_piko/actions/intake.' + name, {method: 'POST', body});
export const action = {intake: {
  Read: () => ({call: async () => (await post('Read', '{}')).json()}),
  Submit: fields => ({call: async () => {
    const response = await post('Submit', fields);
    if (!response.ok) { const failure = new Error('refused'); failure.status = response.status; throw failure; }
    return response.json();
  }})
}};`
)

func TestDriveIntakeRecordsRejectedSubmissionEvidence(t *testing.T) {
	requireChromium(t)
	clearRecipeVariables(t)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		var err error
		switch request.URL.Path {
		case "/intake":
			http.SetCookie(writer, &http.Cookie{Name: "_piko_csrf_token", Value: "bound", Path: "/"})
			writer.Header().Set("Content-Type", "text/html")
			_, err = fmt.Fprint(writer, intakeTestPage)
		case "/pk/actions.gen.js":
			writer.Header().Set("Content-Type", "text/javascript")
			_, err = fmt.Fprint(writer, intakeTestModule)
		case "/_piko/actions/intake.Read":
			writer.Header().Set("Content-Type", "application/json")
			_, err = fmt.Fprint(writer, `{"checks":1,"writes":0,"events":["read"]}`)
		case "/_piko/actions/intake.Submit":
			http.Error(writer, "forbidden", http.StatusForbidden)
		default:
			http.NotFound(writer, request)
		}
		assert.NoError(t, err)
	}))
	defer server.Close()
	origin, err := url.Parse(server.URL)
	require.NoError(t, err)
	ctx, session := openTestBrowser(t, intakeBudget()+10*time.Second)
	out := driveIntake(ctx, session, origin)
	require.Empty(t, out.DriverError)
	assert.Equal(t, "/intake", out.Page)
	assert.Equal(t, "/pk/actions.gen.js", out.Module)
	assert.Equal(t, "/_piko/actions/intake.Submit", out.FormAction)
	assert.Equal(t, "Quota", out.Label)
	assert.Equal(t, "ticket", out.Ticket)
	assert.True(t, out.Rejected)
	assert.Equal(t, http.StatusForbidden, out.Status)
	assert.Equal(t, "http", out.ErrorKind)
	require.NotNil(t, out.Before)
	require.NotNil(t, out.After)
	assert.Equal(t, *out.Before, *out.After)
	want := []casefile.FormShimBrowserRequest{
		{Method: http.MethodPost, Path: "/_piko/actions/intake.Read", Status: http.StatusOK, BindingCookie: true},
		{Method: http.MethodPost, Path: "/_piko/actions/intake.Submit", Status: http.StatusForbidden, BindingCookie: true},
		{Method: http.MethodPost, Path: "/_piko/actions/intake.Read", Status: http.StatusOK, BindingCookie: true},
	}
	assert.Equal(t, want, out.Requests)
}
