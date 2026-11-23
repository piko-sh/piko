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
	"bytes"
	"mime/multipart"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/json"

	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/tests/fuzz/internal/textlimit"
	"piko.sh/piko/wdk/browser"
)

func TestFormPostedFields(t *testing.T) {
	got := workflowFields("application/json", []byte(`{"_csrf_ephemeral_token":"transport","repeat":["one","two"],"label":"hello"}`))
	assert.Equal(t, `{"label":"hello","repeat":["one","two"]}`, got)
}

func TestFormPostedFieldsReportUnreadableBodies(t *testing.T) {
	for _, testCase := range []struct {
		name        string
		contentType string
		body        string
		want        string
	}{
		{name: "oversized", contentType: "application/json", body: strings.Repeat(" ", formBodyLimit+1), want: "oversized"},
		{name: "invalid content type", contentType: "", body: "{}", want: "invalid content type"},
		{name: "invalid json", contentType: "application/json", body: "{", want: "invalid json"},
		{name: "unterminated multipart", contentType: "multipart/form-data; boundary=x", body: "--x\r\nContent-Disposition: form-data; name=\"label\"\r\n\r\nvalue", want: "unreadable multipart"},
		{name: "invalid multipart", contentType: "multipart/form-data; boundary=x", body: "--x\r\nbroken header\r\n\r\n", want: "invalid multipart"},
		{name: "other media", contentType: "text/plain", body: "x", want: "text/plain"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.want, workflowFields(testCase.contentType, []byte(testCase.body)))
		})
	}
}

func TestFormMultipartFieldsMarkOversizedValues(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("label", strings.Repeat("é", formValueLimit)))
	require.NoError(t, writer.WriteField("repeat", "one"))
	require.NoError(t, writer.WriteField("_csrf_ephemeral_token", "transport"))
	require.NoError(t, writer.Close())
	var fields map[string][]string
	require.NoError(t, json.Unmarshal([]byte(workflowFields(writer.FormDataContentType(), body.Bytes())), &fields))
	require.Len(t, fields["label"], 1)
	assert.True(t, strings.HasSuffix(fields["label"][0], textlimit.Marker))
	assert.LessOrEqual(t, len(fields["label"][0]), formValueLimit+len(textlimit.Marker))
	assert.Equal(t, []string{"one"}, fields["repeat"])
	assert.NotContains(t, fields, "_csrf_ephemeral_token")
}

func TestFormRequestsPendingKeepsObservedOrder(t *testing.T) {
	calls := &requests{origin: "http://loopback.invalid", index: make(map[browser.RequestID]int), contentTypes: make(map[browser.RequestID]string)}
	ids := []browser.RequestID{"c", "a", "b", "d"}
	for _, id := range ids {
		calls.observe(browser.NetworkEvent{Kind: browser.RequestStarted, ID: id, URL: calls.origin + "/_piko/actions/task." + string(id), Method: "POST", ContentType: "application/json"})
	}
	calls.observe(browser.NetworkEvent{Kind: browser.RequestStarted, ID: "ignored", URL: "http://elsewhere.invalid/_piko/actions/x", Method: "POST"})
	out, order, contentTypes, err := calls.pending()
	require.NoError(t, err)
	assert.Equal(t, ids, order)
	require.Len(t, out, len(ids))
	for index, id := range ids {
		assert.Equal(t, "/_piko/actions/task."+string(id), out[index].Path)
		assert.Equal(t, "application/json", contentTypes[id])
	}
	out, order, _, err = calls.pending()
	require.NoError(t, err)
	assert.Empty(t, out)
	assert.Empty(t, order)
}

func TestFormRequestsReportOverflow(t *testing.T) {
	calls := &requests{origin: "http://loopback.invalid", index: make(map[browser.RequestID]int), contentTypes: make(map[browser.RequestID]string)}
	for index := range formRequestLimit + 1 {
		calls.observe(browser.NetworkEvent{Kind: browser.RequestStarted, ID: browser.RequestID(strconv.Itoa(index)), URL: calls.origin + "/_piko/partials/list", Method: "GET"})
	}
	_, _, _, err := calls.pending()
	require.ErrorIs(t, err, errFormRequestOverflow)
}

func TestFormWorkflowBudgetGrowsWithSteps(t *testing.T) {
	one := &casefile.FormWorkflow{Steps: []casefile.FormCommand{{Op: "reset"}}}
	two := &casefile.FormWorkflow{Steps: []casefile.FormCommand{{Op: "reset"}, {Op: "reset"}}}
	assert.Equal(t, formStepBudget, formWorkflowBudget(two, false)-formWorkflowBudget(one, false))
	assert.Equal(t, coverageBudget, formWorkflowBudget(one, true)-formWorkflowBudget(one, false))
}

func TestFormRecipeRejectsInvalidInput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "observation.json")
	require.ErrorIs(t, formProbe(t.Context(), "http://127.0.0.1", path, strings.Repeat("x", formRecipeLimit+1)), errInvalidFormRecipe)
	require.ErrorIs(t, formProbe(t.Context(), "http://127.0.0.1", path, "{"), errInvalidFormRecipe)
}

func TestFormStartupFailureReceipt(t *testing.T) {
	directory := t.TempDir()
	badParent := filepath.Join(directory, "file")
	require.NoError(t, os.WriteFile(badParent, []byte("not a directory"), 0o600))
	t.Setenv("TMPDIR", badParent)
	t.Setenv(envJSCoverage, "")
	t.Setenv(envBrowserTemporary, "")
	path := filepath.Join(directory, "observation.json")
	const recipe = `{"Steps":[{"Op":"reset"}]}`
	require.NoError(t, formProbe(t.Context(), "http://127.0.0.1", path, recipe))
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var out casefile.FormWorkflowObservation
	require.NoError(t, json.Unmarshal(data, &out))
	assert.Equal(t, -1, out.FailedStep)
	assert.True(t, strings.HasPrefix(out.DriverError, "opening workflow:"), out.DriverError)
}

func TestFormStartupFailureReceiptIgnoresCoverageMode(t *testing.T) {
	directory := t.TempDir()
	badParent := filepath.Join(directory, "file")
	require.NoError(t, os.WriteFile(badParent, []byte("not a directory"), 0o600))
	t.Setenv("TMPDIR", badParent)
	t.Setenv(envJSCoverage, filepath.Join(directory, "coverage.json"))
	t.Setenv(envBrowserTemporary, "")
	path := filepath.Join(directory, "observation.json")

	err := formProbe(t.Context(), "http://127.0.0.1", path, `{"Steps":[{"Op":"reset"}]}`)

	require.NoError(t, err)
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var out casefile.FormWorkflowObservation
	require.NoError(t, json.Unmarshal(data, &out))
	assert.True(t, strings.HasPrefix(out.DriverError, "opening workflow:"), out.DriverError)
}
