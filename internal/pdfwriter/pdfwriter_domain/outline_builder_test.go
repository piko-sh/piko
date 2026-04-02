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
	"strings"
	"testing"
)

func TestOutlineBuilder_NoEntriesProducesZero(t *testing.T) {
	ob := NewOutlineBuilder()
	writer := &PdfDocumentWriter{}
	writer.WriteHeader()

	result := ob.WriteObjects(writer, []int{1})

	if result != 0 {
		t.Errorf("expected 0 for empty outline, got %d", result)
	}
}

func TestOutlineBuilder_HasEntries(t *testing.T) {
	ob := NewOutlineBuilder()

	if ob.HasEntries() {
		t.Error("expected HasEntries to be false when empty")
	}

	ob.AddEntry(OutlineEntry{Title: "Chapter 1", Level: 1, PageIndex: 0, YPosition: 800})

	if !ob.HasEntries() {
		t.Error("expected HasEntries to be true after adding entry")
	}
}

func TestOutlineBuilder_SingleEntry(t *testing.T) {
	ob := NewOutlineBuilder()
	ob.AddEntry(OutlineEntry{Title: "Introduction", Level: 1, PageIndex: 0, YPosition: 800})

	writer := &PdfDocumentWriter{}
	writer.WriteHeader()
	page_number := writer.AllocateObject()
	writer.WriteObject(page_number, "<< /Type /Page >>")

	root := ob.WriteObjects(writer, []int{page_number})

	if root == 0 {
		t.Fatal("expected non-zero outline root")
	}

	output := string(writer.Bytes())
	if !strings.Contains(output, "/Type /Outlines") {
		t.Error("expected /Type /Outlines in output")
	}
	if !strings.Contains(output, "/Title (Introduction)") {
		t.Error("expected /Title (Introduction) in output")
	}
}

func TestOutlineBuilder_NestedHeadings(t *testing.T) {
	ob := NewOutlineBuilder()
	ob.AddEntry(OutlineEntry{Title: "Chapter 1", Level: 1, PageIndex: 0, YPosition: 800})
	ob.AddEntry(OutlineEntry{Title: "Section 1.1", Level: 2, PageIndex: 0, YPosition: 600})
	ob.AddEntry(OutlineEntry{Title: "Section 1.2", Level: 2, PageIndex: 0, YPosition: 400})
	ob.AddEntry(OutlineEntry{Title: "Chapter 2", Level: 1, PageIndex: 1, YPosition: 800})

	tree := ob.buildTree()

	if len(tree) != 2 {
		t.Fatalf("expected 2 root nodes, got %d", len(tree))
	}
	if tree[0].entry.Title != "Chapter 1" {
		t.Errorf("expected first root to be 'Chapter 1', got %q", tree[0].entry.Title)
	}
	if len(tree[0].children) != 2 {
		t.Errorf("expected 2 children under Chapter 1, got %d", len(tree[0].children))
	}
	if tree[1].entry.Title != "Chapter 2" {
		t.Errorf("expected second root to be 'Chapter 2', got %q", tree[1].entry.Title)
	}
	if len(tree[1].children) != 0 {
		t.Errorf("expected no children under Chapter 2, got %d", len(tree[1].children))
	}
}

func TestOutlineBuilder_DeeplyNested(t *testing.T) {
	ob := NewOutlineBuilder()
	ob.AddEntry(OutlineEntry{Title: "H1", Level: 1, PageIndex: 0, YPosition: 800})
	ob.AddEntry(OutlineEntry{Title: "H2", Level: 2, PageIndex: 0, YPosition: 700})
	ob.AddEntry(OutlineEntry{Title: "H3", Level: 3, PageIndex: 0, YPosition: 600})

	tree := ob.buildTree()

	if len(tree) != 1 {
		t.Fatalf("expected 1 root node, got %d", len(tree))
	}
	if len(tree[0].children) != 1 {
		t.Fatalf("expected 1 child under H1, got %d", len(tree[0].children))
	}
	if len(tree[0].children[0].children) != 1 {
		t.Fatalf("expected 1 child under H2, got %d", len(tree[0].children[0].children))
	}
	if tree[0].children[0].children[0].entry.Title != "H3" {
		t.Errorf("expected H3 nested under H2, got %q", tree[0].children[0].children[0].entry.Title)
	}
}

func TestOutlineBuilder_LevelJump(t *testing.T) {
	ob := NewOutlineBuilder()
	ob.AddEntry(OutlineEntry{Title: "H1", Level: 1, PageIndex: 0, YPosition: 800})
	ob.AddEntry(OutlineEntry{Title: "H3", Level: 3, PageIndex: 0, YPosition: 700})
	ob.AddEntry(OutlineEntry{Title: "H1 again", Level: 1, PageIndex: 0, YPosition: 600})

	tree := ob.buildTree()

	if len(tree) != 2 {
		t.Fatalf("expected 2 root nodes, got %d", len(tree))
	}
	if len(tree[0].children) != 1 {
		t.Errorf("expected 1 child under first H1, got %d", len(tree[0].children))
	}
}

func TestOutlineBuilder_EscapesTitleParentheses(t *testing.T) {
	ob := NewOutlineBuilder()
	ob.AddEntry(OutlineEntry{Title: "Title (with parens)", Level: 1, PageIndex: 0, YPosition: 800})

	writer := &PdfDocumentWriter{}
	writer.WriteHeader()
	page_number := writer.AllocateObject()
	writer.WriteObject(page_number, "<< /Type /Page >>")

	ob.WriteObjects(writer, []int{page_number})

	output := string(writer.Bytes())
	if !strings.Contains(output, `Title \(with parens\)`) {
		t.Errorf("expected escaped parentheses in output, got %q", output)
	}
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
		if result != tt.level {
			t.Errorf("headingLevel(%q) = %d, want %d", tt.tag, result, tt.level)
		}
	}
}
