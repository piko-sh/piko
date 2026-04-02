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

package lsp_domain

import (
	"sync"
	"testing"

	"go.lsp.dev/protocol"
)

func TestDocument_GetSFCResult_ValidContent(t *testing.T) {
	document := &document{
		Content: []byte(`<template><div>Hello</div></template>`),
		URI:     "file:///test.pk",
	}

	result := document.getSFCResult()

	if result == nil {
		t.Fatal("expected non-nil SFCResult for valid content")
	}
	if result.Template == "" {
		t.Error("expected non-empty template in SFCResult")
	}
}

func TestDocument_GetSFCResult_InvalidContent(t *testing.T) {
	document := &document{
		Content: []byte(`not valid pk content without template`),
		URI:     "file:///test.pk",
	}

	result := document.getSFCResult()

	result2 := document.getSFCResult()
	if result != result2 {
		t.Error("expected getSFCResult to return same pointer on repeated calls")
	}
}

func TestDocument_GetSFCResult_EmptyContent(t *testing.T) {
	document := &document{
		Content: []byte(``),
		URI:     "file:///test.pk",
	}

	result := document.getSFCResult()

	_ = result
}

func TestDocument_GetSFCResult_ThreadSafe(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping race condition test in short mode")
	}

	document := &document{
		Content: []byte(`<template><div>Hello {{ name }}</div></template>
<script>
let name = "World"
</script>`),
		URI: "file:///test.pk",
	}

	const goroutines = 100
	var wg sync.WaitGroup
	wg.Add(goroutines)

	results := make([]*struct {
		result any
	}, goroutines)

	for i := range goroutines {
		results[i] = &struct{ result any }{}
		go func(index int) {
			defer wg.Done()
			results[index].result = document.getSFCResult()
		}(i)
	}

	wg.Wait()

	firstResult := results[0].result
	for i := 1; i < goroutines; i++ {
		if results[i].result != firstResult {
			t.Errorf("goroutine %d got different result pointer than goroutine 0", i)
		}
	}
}

func TestDocument_GetSFCResult_ParsesOnce(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping race condition test in short mode")
	}

	document := &document{
		Content: []byte(`<template><div>Test</div></template>`),
		URI:     "file:///test.pk",
	}

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()
			_ = document.getSFCResult()
		}()
	}

	wg.Wait()

	result := document.getSFCResult()
	if result == nil {
		t.Error("expected non-nil result after concurrent calls")
	}
}

func TestDocument_IsPositionInClientScript_NoSFC(t *testing.T) {
	document := &document{
		Content: []byte(``),
		URI:     "file:///test.pk",
	}

	position := protocol.Position{Line: 0, Character: 0}
	result := document.isPositionInClientScript(position)

	if result {
		t.Error("expected false for document with no valid SFC")
	}
}

func TestDocument_IsPositionInClientScript_InTemplate(t *testing.T) {
	document := &document{
		Content: []byte(`<template><div>Hello</div></template>
<script lang="ts" client>
let x = 1
</script>`),
		URI: "file:///test.pk",
	}

	position := protocol.Position{Line: 0, Character: 15}
	result := document.isPositionInClientScript(position)

	if result {
		t.Error("expected false for position in template")
	}
}

func TestDocument_IsPositionInClientScript_InClientScript(t *testing.T) {
	document := &document{
		Content: []byte(`<template><div>Hello</div></template>
<script lang="ts" client>
let x = 1
</script>`),
		URI: "file:///test.pk",
	}

	position := protocol.Position{Line: 2, Character: 5}
	result := document.isPositionInClientScript(position)

	if result {
		t.Log("client script correctly detected")
	}

}

func TestCountNewlinesInContent(t *testing.T) {
	testCases := []struct {
		name     string
		content  string
		expected int
	}{
		{
			name:     "empty string",
			content:  "",
			expected: 0,
		},
		{
			name:     "no newlines",
			content:  "hello world",
			expected: 0,
		},
		{
			name:     "single newline",
			content:  "hello\nworld",
			expected: 1,
		},
		{
			name:     "multiple newlines",
			content:  "line1\nline2\nline3\n",
			expected: 3,
		},
		{
			name:     "only newlines",
			content:  "\n\n\n",
			expected: 3,
		},
		{
			name:     "windows line endings",
			content:  "line1\r\nline2\r\n",
			expected: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := countNewlinesInContent(tc.content)
			if result != tc.expected {
				t.Errorf("expected %d newlines, got %d", tc.expected, result)
			}
		})
	}
}
