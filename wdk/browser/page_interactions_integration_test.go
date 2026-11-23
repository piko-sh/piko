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
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPageInteractions(t *testing.T) {
	page, recorder, _ := newFixturePage(t)
	page.Navigate("/")

	page.Click("#enabled")
	assert.Equal(t, "1", page.GetAttribute("#enabled", "data-clicks"))
	page.DoubleClick("#enabled").Hover("#title").RightClick("#title")
	require.Empty(t, recorder.take())

	page.Fill("#name", "hello")
	assert.Equal(t, "hello", page.EvalReturn("document.querySelector('#name').value"))
	page.Clear("#name").Focus("#name").Type("abcd")
	assert.Equal(t, "abcd", page.EvalReturn("document.querySelector('#name').value"))
	page.SetCursor("#edit", 1)
	assert.Equal(t, 1, page.GetCursorOffset("#edit"))
	page.Select("#edit", 1, 4)
	start, end := page.GetSelectionRange("#edit")
	assert.Equal(t, [2]int{1, 4}, [2]int{start, end})
	page.SelectAll("#edit")
	assert.Equal(t, "abcdef", page.EvalReturn("String(window.getSelection())"))
	page.Press("End").PressAndHold("Shift").Release("Shift").Blur("#name")
	assert.Equal(t, false, page.EvalReturn("document.activeElement === document.querySelector('#name')"))

	page.Check("#check")
	assert.Equal(t, true, page.EvalReturn("document.querySelector('#check').checked"))
	page.Uncheck("#check")
	assert.Equal(t, false, page.EvalReturn("document.querySelector('#check').checked"))

	page.SetFiles("#upload", "upload.txt")
	assert.Equal(t, "upload.txt", page.EvalReturn("document.querySelector('#upload').files[0].name"))
	page.Scroll("#scroller", 50)
	assert.InDelta(t, 50, page.EvalReturn("document.querySelector('#scroller').scrollTop"), 1)

	page.PlaceCursorInElement("#edit", "#tail").CollapseToStart().CollapseToEnd()
	require.Empty(t, recorder.take())

	page.Fill("input[name=q]", "term").Submit("input[name=q]")
	require.Eventually(t, func() bool { return strings.HasSuffix(page.URL(), "?q=term") }, 5*time.Second, 20*time.Millisecond)

	failures := recorder.attempt(func() { page.SetFiles("#upload", "missing.txt") })
	require.Len(t, failures, 1)
	assert.Contains(t, failures[0], "file not found")
}

func TestPageInteractionsReportFailures(t *testing.T) {
	page, recorder, _ := newFixturePage(t)
	cancelPage(t, page)
	requireEachStepFails(t, recorder, []pageStep{
		{name: "DoubleClick", run: func() { page.DoubleClick("#title") }},
		{name: "Click", run: func() { page.Click("#title") }},
		{name: "Hover", run: func() { page.Hover("#title") }},
		{name: "RightClick", run: func() { page.RightClick("#title") }},
		{name: "Fill", run: func() { page.Fill("#name", "x") }},
		{name: "Clear", run: func() { page.Clear("#name") }},
		{name: "Submit", run: func() { page.Submit("#form") }},
		{name: "Uncheck", run: func() { page.Uncheck("#check") }},
		{name: "Check", run: func() { page.Check("#check") }},
		{name: "Focus", run: func() { page.Focus("#name") }},
		{name: "Blur", run: func() { page.Blur("#name") }},
		{name: "SetFiles", run: func() { page.SetFiles("#upload", "upload.txt") }},
		{name: "Scroll", run: func() { page.Scroll("#scroller", 1) }},
		{name: "Press", run: func() { page.Press("Tab") }},
		{name: "Type", run: func() { page.Type("text") }},
		{name: "PressAndHold", run: func() { page.PressAndHold("Shift") }},
		{name: "Release", run: func() { page.Release("Shift") }},
		{name: "SetCursor", run: func() { page.SetCursor("#name", 0) }},
		{name: "SelectAll", run: func() { page.SelectAll("#name") }},
		{name: "Select", run: func() { page.Select("#name", 0, 1) }},
		{name: "CollapseToStart", run: func() { page.CollapseToStart() }},
		{name: "CollapseToEnd", run: func() { page.CollapseToEnd() }},
		{name: "PlaceCursorInElement", run: func() { page.PlaceCursorInElement("#edit", "#tail") }},
		{name: "GetCursorOffset", run: func() { page.GetCursorOffset("#name") }},
		{name: "GetSelectionRange", run: func() { page.GetSelectionRange("#name") }},
	})
}
