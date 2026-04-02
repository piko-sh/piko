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

	"go.lsp.dev/protocol"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
)

func TestIsSpecialTag(t *testing.T) {
	document := newTestDocumentBuilder().Build()

	testCases := []struct {
		name     string
		tagName  string
		expected bool
	}{
		{name: "template is special", tagName: "template", expected: true},
		{name: "script is special", tagName: "script", expected: true},
		{name: "style is special", tagName: "style", expected: true},
		{name: "slot is special", tagName: "slot", expected: true},
		{name: "div is not special", tagName: "div", expected: false},
		{name: "empty string is not special", tagName: "", expected: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := document.isSpecialTag(tc.tagName)
			if result != tc.expected {
				t.Errorf("isSpecialTag(%q) = %v, want %v", tc.tagName, result, tc.expected)
			}
		})
	}
}

func TestIsComponentTag(t *testing.T) {
	document := newTestDocumentBuilder().Build()

	testCases := []struct {
		node     *ast_domain.TemplateNode
		name     string
		expected bool
	}{
		{
			name:     "nil node returns false",
			node:     nil,
			expected: false,
		},
		{
			name: "node with is attribute returns true",
			node: func() *ast_domain.TemplateNode {
				n := newTestNode("div", 1, 0)
				addAttribute(n, "is", "my-component")
				return n
			}(),
			expected: true,
		},
		{
			name: "node with PartialInfo annotation returns true",
			node: func() *ast_domain.TemplateNode {
				n := newTestNode("div", 1, 0)
				n.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
					PartialInfo: &ast_domain.PartialInvocationInfo{
						InvocationKey: "test-key",
						PartialAlias:  "test-alias",
					},
				}
				return n
			}(),
			expected: true,
		},
		{
			name:     "plain node returns false",
			node:     newTestNode("div", 1, 0),
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := document.isComponentTag(tc.node)
			if result != tc.expected {
				t.Errorf("isComponentTag() = %v, want %v", result, tc.expected)
			}
		})
	}
}

func TestGetLinkedEditingRanges_NilAnnotation(t *testing.T) {
	testCases := []struct {
		document *document
		name     string
	}{
		{
			name: "nil AnnotationResult returns nil",
			document: newTestDocumentBuilder().
				WithURI("file:///test.html").
				WithAnnotationResult(nil).
				Build(),
		},
		{
			name: "nil AnnotatedAST returns nil",
			document: newTestDocumentBuilder().
				WithURI("file:///test.html").
				WithAnnotationResult(&annotator_dto.AnnotationResult{
					AnnotatedAST: nil,
				}).
				Build(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			position := protocol.Position{Line: 0, Character: 0}

			result, err := tc.document.GetLinkedEditingRanges(position)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if result != nil {
				t.Errorf("expected nil result, got %+v", result)
			}
		})
	}
}

func TestGetLinkedEditingRanges_PositivePath(t *testing.T) {

	divNode := newTestNode("div", 1, 1)

	ast := newTestAnnotatedAST(divNode)

	document := newTestDocumentBuilder().
		WithURI("file:///test.pk").
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: ast,
		}).
		Build()

	position := protocol.Position{Line: 0, Character: 1}

	result, err := document.GetLinkedEditingRanges(position)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil LinkedEditingRanges, got nil")
	}
	if len(result.Ranges) != 2 {
		t.Fatalf("expected 2 ranges, got %d", len(result.Ranges))
	}

	openRange := result.Ranges[0]
	if openRange.Start.Line != 0 {
		t.Errorf("opening range start line = %d, want 0", openRange.Start.Line)
	}

	closeRange := result.Ranges[1]
	if closeRange.Start.Line != 0 {
		t.Errorf("closing range start line = %d, want 0", closeRange.Start.Line)
	}

	if openRange.Start.Character == closeRange.Start.Character &&
		openRange.End.Character == closeRange.End.Character {
		t.Error("opening and closing ranges should differ in character positions")
	}
}
