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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/safedisk"
)

func TestPageFrames(t *testing.T) {
	page, recorder, _ := newFixturePage(t)
	page.Navigate("/").WaitForElementInFrame("#frame", "#inner")

	assert.True(t, page.IsFrameLoaded("#frame"))
	assert.Equal(t, 1, page.CountFrames())
	assert.NotEmpty(t, page.GetFrames())
	page.ClickInFrame("#frame", "#inner")
	assert.Equal(t, "pressed", page.GetTextInFrame("#frame", "#inner"))
	page.FillInFrame("#frame", "#field", "typed")
	assert.Equal(t, "typed", page.EvalInFrame("#frame", "document.querySelector('#field').value"))
	assert.Contains(t, page.GetFrameDocument("#frame"), "inner")
	require.Empty(t, recorder.take())
}

func TestPageDownloads(t *testing.T) {
	page, recorder, server := newFixturePage(t)
	page.Navigate("/")
	directory := t.TempDir()
	page.SetDownloadPath(directory).TriggerDownload(server.URL+"/download", "report.txt")
	page.CreateBlobDownload("blob body", "text/plain", "blob.txt")
	require.Empty(t, recorder.take())

	require.NoError(t, os.WriteFile(filepath.Join(directory, "saved.txt"), []byte("saved body"), 0o600))
	downloads, err := safedisk.NewNoOpSandbox(directory, safedisk.ModeReadOnly)
	require.NoError(t, err)
	defer func() { _ = downloads.Close() }()
	assert.Equal(t, "saved body", string(page.GetDownloadedFile(downloads, "saved.txt")))

	failures := recorder.attempt(func() { page.GetDownloadedFile(downloads, "absent.txt") })
	require.Len(t, failures, 1)
	assert.Contains(t, failures[0], "GetDownloadedFile")
}

func TestPageFramesReportFailures(t *testing.T) {
	page, recorder, _ := newFixturePage(t)
	cancelPage(t, page)
	requireEachStepFails(t, recorder, []pageStep{
		{name: "GetFrames", run: func() { page.GetFrames() }},
		{name: "CountFrames", run: func() { page.CountFrames() }},
		{name: "ClickInFrame", run: func() { page.ClickInFrame("#frame", "#inner") }},
		{name: "FillInFrame", run: func() { page.FillInFrame("#frame", "#field", "v") }},
		{name: "GetTextInFrame", run: func() { page.GetTextInFrame("#frame", "#inner") }},
		{name: "WaitForElementInFrame", run: func() { page.WaitForElementInFrame("#frame", "#inner") }},
		{name: "GetFrameDocument", run: func() { page.GetFrameDocument("#frame") }},
		{name: "IsFrameLoaded", run: func() { page.IsFrameLoaded("#frame") }},
		{name: "EvalInFrame", run: func() { page.EvalInFrame("#frame", "1") }},
		{name: "SetDownloadPath", run: func() { page.SetDownloadPath(t.TempDir()) }},
		{name: "WaitForDownload", run: func() { page.WaitForDownload(t.TempDir(), func() {}) }},
		{name: "TriggerDownload", run: func() { page.TriggerDownload("/download", "report.txt") }},
		{name: "CreateBlobDownload", run: func() { page.CreateBlobDownload("x", "text/plain", "x.txt") }},
	})
}
