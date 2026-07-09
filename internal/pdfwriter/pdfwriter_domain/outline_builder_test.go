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

package pdfwriter_domain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOutlineBuilder_NoEntriesProducesZero(t *testing.T) {
	ob := NewOutlineBuilder()
	writer := &PdfDocumentWriter{}
	writer.WriteHeader()

	result := ob.WriteObjects(context.Background(), writer, []int{1})

	assert.Zero(t, result, "expected 0 for empty outline")
}

func TestOutlineBuilder_HasEntries(t *testing.T) {
	ob := NewOutlineBuilder()

	assert.False(t, ob.HasEntries(), "expected HasEntries to be false when empty")

	ob.AddEntry(OutlineEntry{Title: "Chapter 1", Level: 1, PageIndex: 0, YPosition: 800})

	assert.True(t, ob.HasEntries(), "expected HasEntries to be true after adding entry")
}

func TestOutlineBuilder_SingleEntry(t *testing.T) {
	ob := NewOutlineBuilder()
	ob.AddEntry(OutlineEntry{Title: "Introduction", Level: 1, PageIndex: 0, YPosition: 800})

	writer := &PdfDocumentWriter{}
	writer.WriteHeader()
	page_number := writer.AllocateObject()
	writer.WriteObject(page_number, "<< /Type /Page >>")

	root := ob.WriteObjects(context.Background(), writer, []int{page_number})

	require.NotZero(t, root, "expected non-zero outline root")

	output := string(writer.Bytes())
	assert.Contains(t, output, "/Type /Outlines")
	assert.Contains(t, output, "/Title (Introduction)")
}

func TestOutlineBuilder_NestedHeadings(t *testing.T) {
	ob := NewOutlineBuilder()
	ob.AddEntry(OutlineEntry{Title: "Chapter 1", Level: 1, PageIndex: 0, YPosition: 800})
	ob.AddEntry(OutlineEntry{Title: "Section 1.1", Level: 2, PageIndex: 0, YPosition: 600})
	ob.AddEntry(OutlineEntry{Title: "Section 1.2", Level: 2, PageIndex: 0, YPosition: 400})
	ob.AddEntry(OutlineEntry{Title: "Chapter 2", Level: 1, PageIndex: 1, YPosition: 800})

	tree := ob.buildTree()

	require.Len(t, tree, 2, "expected 2 root nodes")
	assert.Equal(t, "Chapter 1", tree[0].entry.Title, "expected first root to be 'Chapter 1'")
	assert.Len(t, tree[0].children, 2, "expected 2 children under Chapter 1")
	assert.Equal(t, "Chapter 2", tree[1].entry.Title, "expected second root to be 'Chapter 2'")
	assert.Empty(t, tree[1].children, "expected no children under Chapter 2")
}

func TestOutlineBuilder_DeeplyNested(t *testing.T) {
	ob := NewOutlineBuilder()
	ob.AddEntry(OutlineEntry{Title: "H1", Level: 1, PageIndex: 0, YPosition: 800})
	ob.AddEntry(OutlineEntry{Title: "H2", Level: 2, PageIndex: 0, YPosition: 700})
	ob.AddEntry(OutlineEntry{Title: "H3", Level: 3, PageIndex: 0, YPosition: 600})

	tree := ob.buildTree()

	require.Len(t, tree, 1, "expected 1 root node")
	require.Len(t, tree[0].children, 1, "expected 1 child under H1")
	require.Len(t, tree[0].children[0].children, 1, "expected 1 child under H2")
	assert.Equal(t, "H3", tree[0].children[0].children[0].entry.Title, "expected H3 nested under H2")
}

func TestOutlineBuilder_LevelJump(t *testing.T) {
	ob := NewOutlineBuilder()
	ob.AddEntry(OutlineEntry{Title: "H1", Level: 1, PageIndex: 0, YPosition: 800})
	ob.AddEntry(OutlineEntry{Title: "H3", Level: 3, PageIndex: 0, YPosition: 700})
	ob.AddEntry(OutlineEntry{Title: "H1 again", Level: 1, PageIndex: 0, YPosition: 600})

	tree := ob.buildTree()

	require.Len(t, tree, 2, "expected 2 root nodes")
	assert.Len(t, tree[0].children, 1, "expected 1 child under first H1")
}

func TestOutlineBuilder_EscapesTitleParentheses(t *testing.T) {
	ob := NewOutlineBuilder()
	ob.AddEntry(OutlineEntry{Title: "Title (with parens)", Level: 1, PageIndex: 0, YPosition: 800})

	writer := &PdfDocumentWriter{}
	writer.WriteHeader()
	page_number := writer.AllocateObject()
	writer.WriteObject(page_number, "<< /Type /Page >>")

	ob.WriteObjects(context.Background(), writer, []int{page_number})

	output := string(writer.Bytes())
	assert.Contains(t, output, `Title \(with parens\)`, "expected escaped parentheses in output")
}

func TestOutlineBuilder_DeepNestingTerminates(t *testing.T) {

	root := &outlineNode{entry: OutlineEntry{Title: "root", PageIndex: 0}}
	current := root
	const nesting = maxOutlineDepth + 64
	for range nesting {
		child := &outlineNode{entry: OutlineEntry{Title: "child", PageIndex: 0}, parent: current}
		current.children = []*outlineNode{child}
		current = child
	}

	ob := NewOutlineBuilder()
	writer := &PdfDocumentWriter{}
	writer.WriteHeader()
	page_number := writer.AllocateObject()
	writer.WriteObject(page_number, "<< /Type /Page >>")
	rootNumber := writer.AllocateObject()

	first, last, count := ob.writeChildren(context.Background(), writer, []*outlineNode{root}, rootNumber, []int{page_number}, 0)

	require.NotZero(t, first, "expected non-zero first child object number")
	require.NotZero(t, last, "expected non-zero last child object number")
	require.Positive(t, count, "expected positive visible item count")
	assert.LessOrEqual(t, count, nesting, "expected count capped by depth limit")
}

func TestHeadingLevel(t *testing.T) {
	tests := []struct {
		tag   string
		level int
	}{
		{tag: "h1", level: 1},
		{tag: "h2", level: 2},
		{tag: "h3", level: 3},
		{tag: "h4", level: 4},
		{tag: "h5", level: 5},
		{tag: "h6", level: 6},
		{tag: "div", level: 0},
		{tag: "p", level: 0},
		{tag: "span", level: 0},
		{tag: "", level: 0},
	}

	for _, tt := range tests {
		result := headingLevel(tt.tag)
		assert.Equal(t, tt.level, result, "headingLevel(%q)", tt.tag)
	}
}
