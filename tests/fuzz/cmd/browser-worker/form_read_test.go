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
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/html"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

func TestFormFreshClientReadsChangedServerRows(t *testing.T) {
	label := "&lt;label&gt;"
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Empty(t, request.Header.Get("Cookie"), "fresh client reused a cookie")
		http.SetCookie(writer, &http.Cookie{Name: "visit", Value: "one"})
		_, err := fmt.Fprintf(writer, `<form id="workflow-form"></form><ul><li data-fuzz-task="1" data-done="true"><span data-task-label>%s</span><button>Delete</button></li></ul>`, label)
		assert.NoError(t, err)
	}))
	defer server.Close()
	for _, want := range []string{"<label>", "next"} {
		rows, err := fetchFreshForm(t.Context(), server.URL)
		require.NoError(t, err)
		assert.Equal(t, []casefile.FormRow{{ID: 1, Label: want, Done: true}}, rows)
		label = "next"
	}
}

func TestFormFreshClientRejectsErrorStatusAndOversizedPage(t *testing.T) {
	for _, testCase := range []struct {
		name   string
		want   string
		body   string
		status int
	}{
		{name: "error status", status: http.StatusInternalServerError, body: strings.Repeat("x", 4096), want: "HTTP 500"},
		{name: "oversized page", status: http.StatusOK, body: strings.Repeat("x", freshFormLimit+1), want: errFreshFormTooLarge.Error()},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.WriteHeader(testCase.status)
				_, err := writer.Write([]byte(testCase.body))
				assert.NoError(t, err)
			}))
			defer server.Close()
			_, err := fetchFreshForm(t.Context(), server.URL)
			require.Error(t, err)
			assert.Contains(t, err.Error(), testCase.want)
		})
	}
}

func TestFormFreshClientReportsTimeoutCause(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer server.Close()
	expired, cancel := context.WithDeadlineCause(t.Context(), time.Now(), errFreshFormTimeout)
	defer cancel()
	_, err := fetchFreshForm(expired, server.URL)
	require.ErrorIs(t, err, errFreshFormTimeout)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestFormRowsRejectMalformedDocuments(t *testing.T) {
	for _, testCase := range []struct {
		name   string
		markup string
	}{
		{name: "no form", markup: `<ul></ul>`},
		{name: "bad identity", markup: `<form id="workflow-form"></form><li data-fuzz-task="x" data-done="true"><span data-task-label>a</span></li>`},
		{name: "bad state", markup: `<form id="workflow-form"></form><li data-fuzz-task="1" data-done="maybe"><span data-task-label>a</span></li>`},
		{name: "no label", markup: `<form id="workflow-form"></form><li data-fuzz-task="1" data-done="true"></li>`},
		{name: "too many rows", markup: `<form id="workflow-form"></form>` + strings.Repeat(`<li data-fuzz-task="1" data-done="true"><span data-task-label>a</span></li>`, formRowLimit+1)},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			document, err := html.Parse(strings.NewReader(testCase.markup))
			require.NoError(t, err)
			_, err = formRows(document)
			require.ErrorIs(t, err, errInvalidFormRows)
		})
	}
}

func TestFormSQLiteReadRequiresDatabasePath(t *testing.T) {
	t.Setenv(envWorkflowDatabase, "")
	_, err := readFormStore(t.Context())
	require.ErrorIs(t, err, errMissingWorkflowDatabase)
}

func TestFormSQLiteReadReportsTimeoutCause(t *testing.T) {
	expired, cancel := context.WithDeadlineCause(t.Context(), time.Now(), errFormStoreTimeout)
	defer cancel()
	_, err := queryFormStore(expired, filepath.Join(t.TempDir(), "tasks.sqlite"))
	require.ErrorIs(t, err, errFormStoreTimeout)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestFormSQLiteReadUsesIndependentReadOnlyConnection(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 unavailable")
	}
	path := filepath.Join(t.TempDir(), "tasks.sqlite")
	_, err := queryFormStore(t.Context(), path)
	require.Error(t, err, "missing database was treated as empty")
	assert.Contains(t, err.Error(), "unable to open database")
	const write = `import sqlite3,sys
with sqlite3.connect(sys.argv[1]) as db: db.executescript(sys.argv[2])`
	for index, query := range []string{"CREATE TABLE tasks(id INTEGER,label TEXT,done INTEGER); INSERT INTO tasks VALUES (7,'from-disk',0);", "UPDATE tasks SET done=1 WHERE id=7;"} {
		output, err := exec.CommandContext(t.Context(), "python3", "-I", "-c", write, path, query).CombinedOutput()
		require.NoError(t, err, string(output))
		rows, err := queryFormStore(t.Context(), path)
		require.NoError(t, err)
		assert.Equal(t, []casefile.FormRow{{ID: 7, Label: "from-disk", Done: index == 1}}, rows)
	}
}
