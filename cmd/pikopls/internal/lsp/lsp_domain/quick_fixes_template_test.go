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
	"testing"

	"piko.sh/piko/internal/ast/ast_domain"
)

func TestFindComponentTagAtPosition(t *testing.T) {
	testCases := []struct {
		name      string
		wantTag   string
		rootNodes []*ast_domain.TemplateNode
		line      uint32
		wantNil   bool
	}{
		{
			name:      "nil root nodes returns nil",
			rootNodes: nil,
			line:      0,
			wantNil:   true,
		},
		{
			name:      "empty root nodes returns nil",
			rootNodes: []*ast_domain.TemplateNode{},
			line:      0,
			wantNil:   true,
		},
		{
			name: "finds node at matching line",
			rootNodes: []*ast_domain.TemplateNode{
				newTestNode("div", 3, 1),
			},
			line:    2,
			wantTag: "div",
		},
		{
			name: "no match at wrong line",
			rootNodes: []*ast_domain.TemplateNode{
				newTestNode("div", 3, 1),
			},
			line:    10,
			wantNil: true,
		},
		{
			name: "finds nested child node",
			rootNodes: func() []*ast_domain.TemplateNode {
				parent := newTestNode("div", 1, 1)
				child := newTestNode("span", 3, 5)
				parent.Children = append(parent.Children, child)
				return []*ast_domain.TemplateNode{parent}
			}(),
			line:    2,
			wantTag: "span",
		},
		{
			name: "skips nodes without tag name",
			rootNodes: []*ast_domain.TemplateNode{
				{
					Location: ast_domain.Location{Line: 3, Column: 1},
					TagName:  "",
				},
			},
			line:    2,
			wantNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := findComponentTagAtPosition(tc.rootNodes, tc.line, 0)

			if tc.wantNil {
				if got != nil {
					t.Errorf("expected nil, got node with tag %q", got.TagName)
				}
				return
			}

			if got == nil {
				t.Fatal("expected non-nil node")
			}

			if got.TagName != tc.wantTag {
				t.Errorf("TagName = %q, want %q", got.TagName, tc.wantTag)
			}
		})
	}
}

func TestCalculateTagEndPosition(t *testing.T) {
	testCases := []struct {
		node            *ast_domain.TemplateNode
		name            string
		content         string
		wantLine        uint32
		wantChar        uint32
		wantSelfClosing bool
		wantFound       bool
	}{
		{
			name:      "nil node returns not found",
			content:   "<div>",
			node:      nil,
			wantFound: false,
		},
		{
			name:    "node with zero line returns not found",
			content: "<div>",
			node: &ast_domain.TemplateNode{
				TagName:  "div",
				Location: ast_domain.Location{Line: 0, Column: 1},
			},
			wantFound: false,
		},
		{
			name:            "simple opening tag",
			content:         "<div>content</div>",
			node:            newTestNode("div", 1, 1),
			wantLine:        0,
			wantChar:        4,
			wantSelfClosing: false,
			wantFound:       true,
		},
		{
			name:            "self-closing tag",
			content:         "<MyComponent />",
			node:            newTestNode("MyComponent", 1, 1),
			wantLine:        0,
			wantChar:        13,
			wantSelfClosing: true,
			wantFound:       true,
		},
		{
			name:            "multi-line tag finds closing bracket",
			content:         "<MyComponent\n  title=\"hello\"\n>",
			node:            newTestNode("MyComponent", 1, 1),
			wantLine:        2,
			wantChar:        0,
			wantSelfClosing: false,
			wantFound:       true,
		},
		{
			name:            "multi-line self-closing tag",
			content:         "<MyComponent\n  title=\"hello\"\n/>",
			node:            newTestNode("MyComponent", 1, 1),
			wantLine:        2,
			wantChar:        0,
			wantSelfClosing: true,
			wantFound:       true,
		},
		{
			name:      "node line beyond content returns not found",
			content:   "<div>",
			node:      newTestNode("div", 100, 1),
			wantFound: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotLine, gotChar, gotSelfClosing, gotFound := calculateTagEndPosition([]byte(tc.content), tc.node)

			if gotFound != tc.wantFound {
				t.Fatalf("found = %v, want %v", gotFound, tc.wantFound)
			}

			if !tc.wantFound {
				return
			}

			if gotLine != tc.wantLine {
				t.Errorf("line = %d, want %d", gotLine, tc.wantLine)
			}
			if gotChar != tc.wantChar {
				t.Errorf("character = %d, want %d", gotChar, tc.wantChar)
			}
			if gotSelfClosing != tc.wantSelfClosing {
				t.Errorf("isSelfClosing = %v, want %v", gotSelfClosing, tc.wantSelfClosing)
			}
		})
	}
}
