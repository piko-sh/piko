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
	"context"
	"testing"

	"go.lsp.dev/protocol"
	"piko.sh/piko/internal/ast/ast_domain"
)

func TestIsPositionInRange(t *testing.T) {
	testCases := []struct {
		name     string
		position protocol.Position
		r        protocol.Range
		expected bool
	}{
		{
			name:     "position at range start",
			position: protocol.Position{Line: 1, Character: 5},
			r: protocol.Range{
				Start: protocol.Position{Line: 1, Character: 5},
				End:   protocol.Position{Line: 1, Character: 10},
			},
			expected: true,
		},
		{
			name:     "position at range end",
			position: protocol.Position{Line: 1, Character: 10},
			r: protocol.Range{
				Start: protocol.Position{Line: 1, Character: 5},
				End:   protocol.Position{Line: 1, Character: 10},
			},
			expected: true,
		},
		{
			name:     "position inside range same line",
			position: protocol.Position{Line: 1, Character: 7},
			r: protocol.Range{
				Start: protocol.Position{Line: 1, Character: 5},
				End:   protocol.Position{Line: 1, Character: 10},
			},
			expected: true,
		},
		{
			name:     "position before range start",
			position: protocol.Position{Line: 1, Character: 3},
			r: protocol.Range{
				Start: protocol.Position{Line: 1, Character: 5},
				End:   protocol.Position{Line: 1, Character: 10},
			},
			expected: false,
		},
		{
			name:     "position after range end",
			position: protocol.Position{Line: 1, Character: 15},
			r: protocol.Range{
				Start: protocol.Position{Line: 1, Character: 5},
				End:   protocol.Position{Line: 1, Character: 10},
			},
			expected: false,
		},
		{
			name:     "position on line before range",
			position: protocol.Position{Line: 0, Character: 5},
			r: protocol.Range{
				Start: protocol.Position{Line: 1, Character: 5},
				End:   protocol.Position{Line: 1, Character: 10},
			},
			expected: false,
		},
		{
			name:     "position on line after range",
			position: protocol.Position{Line: 2, Character: 5},
			r: protocol.Range{
				Start: protocol.Position{Line: 1, Character: 5},
				End:   protocol.Position{Line: 1, Character: 10},
			},
			expected: false,
		},
		{
			name:     "multi-line range position in middle line",
			position: protocol.Position{Line: 2, Character: 0},
			r: protocol.Range{
				Start: protocol.Position{Line: 1, Character: 5},
				End:   protocol.Position{Line: 3, Character: 10},
			},
			expected: true,
		},
		{
			name:     "multi-line range position on start line after start char",
			position: protocol.Position{Line: 1, Character: 8},
			r: protocol.Range{
				Start: protocol.Position{Line: 1, Character: 5},
				End:   protocol.Position{Line: 3, Character: 10},
			},
			expected: true,
		},
		{
			name:     "multi-line range position on end line before end char",
			position: protocol.Position{Line: 3, Character: 5},
			r: protocol.Range{
				Start: protocol.Position{Line: 1, Character: 5},
				End:   protocol.Position{Line: 3, Character: 10},
			},
			expected: true,
		},
		{
			name:     "multi-line range position on end line after end char",
			position: protocol.Position{Line: 3, Character: 15},
			r: protocol.Range{
				Start: protocol.Position{Line: 1, Character: 5},
				End:   protocol.Position{Line: 3, Character: 10},
			},
			expected: false,
		},
		{
			name:     "zero position in zero range",
			position: protocol.Position{Line: 0, Character: 0},
			r: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 0},
			},
			expected: true,
		},
		{
			name:     "single character range",
			position: protocol.Position{Line: 0, Character: 5},
			r: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 5},
				End:   protocol.Position{Line: 0, Character: 6},
			},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isPositionInRange(tc.position, tc.r)
			if result != tc.expected {
				t.Errorf("isPositionInRange(%v, %v) = %v, want %v", tc.position, tc.r, result, tc.expected)
			}
		})
	}
}

func TestIsRangeSmaller(t *testing.T) {
	testCases := []struct {
		name     string
		r1       protocol.Range
		r2       protocol.Range
		expected bool
	}{
		{
			name: "r1 smaller same line",
			r1: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 5},
			},
			r2: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 10},
			},
			expected: true,
		},
		{
			name: "r1 larger same line",
			r1: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 10},
			},
			r2: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 5},
			},
			expected: false,
		},
		{
			name: "equal ranges",
			r1: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 5},
			},
			r2: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 5},
			},
			expected: false,
		},
		{
			name: "r1 single line r2 multi line",
			r1: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 100},
			},
			r2: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 1, Character: 10},
			},
			expected: true,
		},
		{
			name: "both multi line r1 fewer lines",
			r1: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 1, Character: 5},
			},
			r2: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 2, Character: 5},
			},
			expected: true,
		},
		{
			name: "empty ranges",
			r1: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 0},
			},
			r2: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 0},
			},
			expected: false,
		},
		{
			name: "r1 empty r2 non-empty",
			r1: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 0},
			},
			r2: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 1},
			},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isRangeSmaller(tc.r1, tc.r2)
			if result != tc.expected {
				t.Errorf("isRangeSmaller(%v, %v) = %v, want %v", tc.r1, tc.r2, result, tc.expected)
			}
		})
	}
}

func TestAstRangeToLSPRange(t *testing.T) {
	testCases := []struct {
		name     string
		astRange ast_domain.Range
		expected protocol.Range
	}{
		{
			name: "standard conversion 1-based to 0-based",
			astRange: ast_domain.Range{
				Start: ast_domain.Location{Line: 1, Column: 1},
				End:   ast_domain.Location{Line: 1, Column: 10},
			},
			expected: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 9},
			},
		},
		{
			name: "multi-line range",
			astRange: ast_domain.Range{
				Start: ast_domain.Location{Line: 5, Column: 3},
				End:   ast_domain.Location{Line: 10, Column: 8},
			},
			expected: protocol.Range{
				Start: protocol.Position{Line: 4, Character: 2},
				End:   protocol.Position{Line: 9, Character: 7},
			},
		},
		{
			name: "synthetic start returns empty range",
			astRange: ast_domain.Range{
				Start: ast_domain.Location{Line: 0, Column: 0},
				End:   ast_domain.Location{Line: 5, Column: 5},
			},
			expected: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 0},
			},
		},
		{
			name: "synthetic end uses start",
			astRange: ast_domain.Range{
				Start: ast_domain.Location{Line: 3, Column: 5},
				End:   ast_domain.Location{Line: 0, Column: 0},
			},
			expected: protocol.Range{
				Start: protocol.Position{Line: 2, Character: 4},
				End:   protocol.Position{Line: 2, Character: 4},
			},
		},
		{
			name: "large line and column numbers",
			astRange: ast_domain.Range{
				Start: ast_domain.Location{Line: 1000, Column: 500},
				End:   ast_domain.Location{Line: 2000, Column: 800},
			},
			expected: protocol.Range{
				Start: protocol.Position{Line: 999, Character: 499},
				End:   protocol.Position{Line: 1999, Character: 799},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := astRangeToLSPRange(tc.astRange)
			if result != tc.expected {
				t.Errorf("astRangeToLSPRange(%v) = %v, want %v", tc.astRange, result, tc.expected)
			}
		})
	}
}

func TestIsNodeFromDocument(t *testing.T) {
	testCases := []struct {
		name     string
		node     *ast_domain.TemplateNode
		docPath  string
		expected bool
	}{
		{
			name: "nil GoAnnotations returns true",
			node: &ast_domain.TemplateNode{
				GoAnnotations: nil,
			},
			docPath:  "/path/to/document.pk",
			expected: true,
		},
		{
			name: "nil OriginalSourcePath returns true",
			node: &ast_domain.TemplateNode{
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalSourcePath: nil,
				},
			},
			docPath:  "/path/to/document.pk",
			expected: true,
		},
		{
			name: "matching path returns true",
			node: &ast_domain.TemplateNode{
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalSourcePath: new("/path/to/document.pk"),
				},
			},
			docPath:  "/path/to/document.pk",
			expected: true,
		},
		{
			name: "non-matching path returns false",
			node: &ast_domain.TemplateNode{
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalSourcePath: new("/path/to/other.pk"),
				},
			},
			docPath:  "/path/to/document.pk",
			expected: false,
		},
		{
			name: "empty path matches empty",
			node: &ast_domain.TemplateNode{
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalSourcePath: new(""),
				},
			},
			docPath:  "",
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isNodeFromDocument(tc.node, tc.docPath)
			if result != tc.expected {
				t.Errorf("isNodeFromDocument() = %v, want %v", result, tc.expected)
			}
		})
	}
}

func TestNodeFindState_CheckOpeningTag(t *testing.T) {
	testCases := []struct {
		node        *ast_domain.TemplateNode
		name        string
		position    protocol.Position
		expectMatch bool
		expectInTag bool
	}{
		{
			name: "synthetic opening tag returns false",
			node: &ast_domain.TemplateNode{
				OpeningTagRange: ast_domain.Range{
					Start: ast_domain.Location{Line: 0, Column: 0},
					End:   ast_domain.Location{Line: 1, Column: 10},
				},
			},
			position:    protocol.Position{Line: 0, Character: 5},
			expectMatch: false,
			expectInTag: false,
		},
		{
			name: "position in opening tag",
			node: &ast_domain.TemplateNode{
				OpeningTagRange: ast_domain.Range{
					Start: ast_domain.Location{Line: 1, Column: 1},
					End:   ast_domain.Location{Line: 1, Column: 20},
				},
			},
			position:    protocol.Position{Line: 0, Character: 5},
			expectMatch: true,
			expectInTag: true,
		},
		{
			name: "position outside opening tag",
			node: &ast_domain.TemplateNode{
				OpeningTagRange: ast_domain.Range{
					Start: ast_domain.Location{Line: 1, Column: 1},
					End:   ast_domain.Location{Line: 1, Column: 10},
				},
			},
			position:    protocol.Position{Line: 5, Character: 0},
			expectMatch: false,
			expectInTag: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			state := &nodeFindState{}
			result := state.checkOpeningTag(tc.node, tc.position)
			if result != tc.expectMatch {
				t.Errorf("checkOpeningTag() returned %v, want %v", result, tc.expectMatch)
			}
			if state.foundInTag != tc.expectInTag {
				t.Errorf("foundInTag = %v, want %v", state.foundInTag, tc.expectInTag)
			}
			if tc.expectMatch && state.bestMatch != tc.node {
				t.Errorf("bestMatch not set correctly")
			}
		})
	}
}

func TestNodeFindState_CheckNodeRange(t *testing.T) {
	testCases := []struct {
		initialState   *nodeFindState
		node           *ast_domain.TemplateNode
		name           string
		position       protocol.Position
		expectNewMatch bool
	}{
		{
			name:         "position in range updates best match",
			initialState: &nodeFindState{},
			node: &ast_domain.TemplateNode{
				NodeRange: ast_domain.Range{
					Start: ast_domain.Location{Line: 1, Column: 1},
					End:   ast_domain.Location{Line: 1, Column: 20},
				},
			},
			position:       protocol.Position{Line: 0, Character: 5},
			expectNewMatch: true,
		},
		{
			name:         "position outside range does not update",
			initialState: &nodeFindState{},
			node: &ast_domain.TemplateNode{
				NodeRange: ast_domain.Range{
					Start: ast_domain.Location{Line: 1, Column: 1},
					End:   ast_domain.Location{Line: 1, Column: 10},
				},
			},
			position:       protocol.Position{Line: 5, Character: 0},
			expectNewMatch: false,
		},
		{
			name: "smaller range replaces larger range",
			initialState: &nodeFindState{
				bestMatch: &ast_domain.TemplateNode{TagName: "outer"},
				bestRange: protocol.Range{
					Start: protocol.Position{Line: 0, Character: 0},
					End:   protocol.Position{Line: 0, Character: 100},
				},
			},
			node: &ast_domain.TemplateNode{
				TagName: "inner",
				NodeRange: ast_domain.Range{
					Start: ast_domain.Location{Line: 1, Column: 1},
					End:   ast_domain.Location{Line: 1, Column: 20},
				},
			},
			position:       protocol.Position{Line: 0, Character: 5},
			expectNewMatch: true,
		},
		{
			name: "larger range does not replace smaller range",
			initialState: &nodeFindState{
				bestMatch: &ast_domain.TemplateNode{TagName: "inner"},
				bestRange: protocol.Range{
					Start: protocol.Position{Line: 0, Character: 0},
					End:   protocol.Position{Line: 0, Character: 10},
				},
			},
			node: &ast_domain.TemplateNode{
				TagName: "outer",
				NodeRange: ast_domain.Range{
					Start: ast_domain.Location{Line: 1, Column: 1},
					End:   ast_domain.Location{Line: 1, Column: 100},
				},
			},
			position:       protocol.Position{Line: 0, Character: 5},
			expectNewMatch: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			originalMatch := tc.initialState.bestMatch
			tc.initialState.checkNodeRange(tc.node, tc.position)

			if tc.expectNewMatch {
				if tc.initialState.bestMatch != tc.node {
					t.Errorf("expected bestMatch to be updated to new node")
				}
			} else {
				if tc.initialState.bestMatch != originalMatch {
					t.Errorf("expected bestMatch to remain unchanged")
				}
			}
		})
	}
}

func TestHasPassedPropAtPosition(t *testing.T) {
	testCases := []struct {
		node     *ast_domain.TemplateNode
		name     string
		position protocol.Position
		expected bool
	}{
		{
			name: "nil GoAnnotations returns false",
			node: &ast_domain.TemplateNode{
				GoAnnotations: nil,
			},
			position: protocol.Position{Line: 0, Character: 5},
			expected: false,
		},
		{
			name: "nil PartialInfo returns false",
			node: &ast_domain.TemplateNode{
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					PartialInfo: nil,
				},
			},
			position: protocol.Position{Line: 0, Character: 5},
			expected: false,
		},
		{
			name: "matching prop position returns true",
			node: &ast_domain.TemplateNode{
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					PartialInfo: &ast_domain.PartialInvocationInfo{
						PassedProps: map[string]ast_domain.PropValue{
							"testProp": {
								NameLocation: ast_domain.Location{Line: 1, Column: 5},
							},
						},
					},
				},
			},
			position: protocol.Position{Line: 0, Character: 10},
			expected: true,
		},
		{
			name: "non-matching line returns false",
			node: &ast_domain.TemplateNode{
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					PartialInfo: &ast_domain.PartialInvocationInfo{
						PassedProps: map[string]ast_domain.PropValue{
							"testProp": {
								NameLocation: ast_domain.Location{Line: 1, Column: 5},
							},
						},
					},
				},
			},
			position: protocol.Position{Line: 5, Character: 10},
			expected: false,
		},
		{
			name: "position before prop column returns false",
			node: &ast_domain.TemplateNode{
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					PartialInfo: &ast_domain.PartialInvocationInfo{
						PassedProps: map[string]ast_domain.PropValue{
							"testProp": {
								NameLocation: ast_domain.Location{Line: 1, Column: 10},
							},
						},
					},
				},
			},
			position: protocol.Position{Line: 0, Character: 3},
			expected: false,
		},
		{
			name: "synthetic location returns false",
			node: &ast_domain.TemplateNode{
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					PartialInfo: &ast_domain.PartialInvocationInfo{
						PassedProps: map[string]ast_domain.PropValue{
							"testProp": {
								NameLocation: ast_domain.Location{Line: 0, Column: 0},
							},
						},
					},
				},
			},
			position: protocol.Position{Line: 0, Character: 5},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := hasPassedPropAtPosition(context.Background(), tc.node, tc.position)
			if result != tc.expected {
				t.Errorf("hasPassedPropAtPosition() = %v, want %v", result, tc.expected)
			}
		})
	}
}

func TestHasBindDirectiveAtPosition(t *testing.T) {
	testCases := []struct {
		binds    map[string]*ast_domain.Directive
		name     string
		position protocol.Position
		expected bool
	}{
		{
			name:     "nil map returns false",
			binds:    nil,
			position: protocol.Position{Line: 0, Character: 5},
			expected: false,
		},
		{
			name:     "empty map returns false",
			binds:    map[string]*ast_domain.Directive{},
			position: protocol.Position{Line: 0, Character: 5},
			expected: false,
		},
		{
			name: "nil directive in map returns false",
			binds: map[string]*ast_domain.Directive{
				"test": nil,
			},
			position: protocol.Position{Line: 0, Character: 5},
			expected: false,
		},
		{
			name: "synthetic directive returns false",
			binds: map[string]*ast_domain.Directive{
				"test": {
					AttributeRange: ast_domain.Range{
						Start: ast_domain.Location{Line: 0, Column: 0},
						End:   ast_domain.Location{Line: 1, Column: 10},
					},
				},
			},
			position: protocol.Position{Line: 0, Character: 5},
			expected: false,
		},
		{
			name: "position in directive range returns true",
			binds: map[string]*ast_domain.Directive{
				"test": {
					AttributeRange: ast_domain.Range{
						Start: ast_domain.Location{Line: 1, Column: 1},
						End:   ast_domain.Location{Line: 1, Column: 20},
					},
				},
			},
			position: protocol.Position{Line: 0, Character: 5},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := hasBindDirectiveAtPosition(context.Background(), tc.binds, tc.position)
			if result != tc.expected {
				t.Errorf("hasBindDirectiveAtPosition() = %v, want %v", result, tc.expected)
			}
		})
	}
}

func TestHasEventDirectiveAtPosition(t *testing.T) {
	testCases := []struct {
		onEvents map[string][]ast_domain.Directive
		name     string
		position protocol.Position
		expected bool
	}{
		{
			name:     "nil map returns false",
			onEvents: nil,
			position: protocol.Position{Line: 0, Character: 5},
			expected: false,
		},
		{
			name:     "empty map returns false",
			onEvents: map[string][]ast_domain.Directive{},
			position: protocol.Position{Line: 0, Character: 5},
			expected: false,
		},
		{
			name: "empty slice returns false",
			onEvents: map[string][]ast_domain.Directive{
				"click": {},
			},
			position: protocol.Position{Line: 0, Character: 5},
			expected: false,
		},
		{
			name: "synthetic directive returns false",
			onEvents: map[string][]ast_domain.Directive{
				"click": {
					{
						AttributeRange: ast_domain.Range{
							Start: ast_domain.Location{Line: 0, Column: 0},
							End:   ast_domain.Location{Line: 1, Column: 10},
						},
					},
				},
			},
			position: protocol.Position{Line: 0, Character: 5},
			expected: false,
		},
		{
			name: "position in event directive returns true",
			onEvents: map[string][]ast_domain.Directive{
				"click": {
					{
						AttributeRange: ast_domain.Range{
							Start: ast_domain.Location{Line: 1, Column: 1},
							End:   ast_domain.Location{Line: 1, Column: 20},
						},
					},
				},
			},
			position: protocol.Position{Line: 0, Character: 5},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := hasEventDirectiveAtPosition(context.Background(), tc.onEvents, tc.position)
			if result != tc.expected {
				t.Errorf("hasEventDirectiveAtPosition() = %v, want %v", result, tc.expected)
			}
		})
	}
}

func TestCheckAttrRangeMatch(t *testing.T) {
	testCases := []struct {
		name          string
		attributeName string
		attributeType string
		r             ast_domain.Range
		position      protocol.Position
		expected      bool
	}{
		{
			name:          "position in range returns true",
			attributeName: "title",
			attributeType: "static attr",
			r: ast_domain.Range{
				Start: ast_domain.Location{Line: 1, Column: 1},
				End:   ast_domain.Location{Line: 1, Column: 20},
			},
			position: protocol.Position{Line: 0, Character: 5},
			expected: true,
		},
		{
			name:          "position outside range returns false",
			attributeName: "title",
			attributeType: "static attr",
			r: ast_domain.Range{
				Start: ast_domain.Location{Line: 1, Column: 1},
				End:   ast_domain.Location{Line: 1, Column: 10},
			},
			position: protocol.Position{Line: 5, Character: 0},
			expected: false,
		},
		{
			name:          "synthetic range returns false",
			attributeName: "title",
			attributeType: "dynamic attr",
			r: ast_domain.Range{
				Start: ast_domain.Location{Line: 0, Column: 0},
				End:   ast_domain.Location{Line: 1, Column: 10},
			},
			position: protocol.Position{Line: 0, Character: 5},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := checkAttrRangeMatch(context.Background(), tc.attributeName, &tc.r, tc.position, tc.attributeType)
			if result != tc.expected {
				t.Errorf("checkAttrRangeMatch() = %v, want %v", result, tc.expected)
			}
		})
	}
}

func TestHasStandardDirectiveAtPosition(t *testing.T) {
	testCases := []struct {
		node     *ast_domain.TemplateNode
		name     string
		position protocol.Position
		expected bool
	}{
		{
			name:     "no directives set",
			node:     &ast_domain.TemplateNode{},
			position: protocol.Position{Line: 0, Character: 5},
			expected: false,
		},
		{
			name: "position in DirIf range",
			node: &ast_domain.TemplateNode{
				DirIf: &ast_domain.Directive{
					AttributeRange: ast_domain.Range{
						Start: ast_domain.Location{Line: 1, Column: 1},
						End:   ast_domain.Location{Line: 1, Column: 20},
					},
				},
			},
			position: protocol.Position{Line: 0, Character: 5},
			expected: true,
		},
		{
			name: "position in DirFor range",
			node: &ast_domain.TemplateNode{
				DirFor: &ast_domain.Directive{
					AttributeRange: ast_domain.Range{
						Start: ast_domain.Location{Line: 2, Column: 1},
						End:   ast_domain.Location{Line: 2, Column: 30},
					},
				},
			},
			position: protocol.Position{Line: 1, Character: 10},
			expected: true,
		},
		{
			name: "position outside all directives",
			node: &ast_domain.TemplateNode{
				DirIf: &ast_domain.Directive{
					AttributeRange: ast_domain.Range{
						Start: ast_domain.Location{Line: 1, Column: 1},
						End:   ast_domain.Location{Line: 1, Column: 10},
					},
				},
			},
			position: protocol.Position{Line: 5, Character: 0},
			expected: false,
		},
		{
			name: "synthetic directive skipped",
			node: &ast_domain.TemplateNode{
				DirShow: &ast_domain.Directive{
					AttributeRange: ast_domain.Range{
						Start: ast_domain.Location{Line: 0, Column: 0},
						End:   ast_domain.Location{Line: 1, Column: 10},
					},
				},
			},
			position: protocol.Position{Line: 0, Character: 5},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := hasStandardDirectiveAtPosition(context.Background(), tc.node, tc.position)
			if result != tc.expected {
				t.Errorf("hasStandardDirectiveAtPosition() = %v, want %v", result, tc.expected)
			}
		})
	}
}

func TestHasDirectiveAtPosition(t *testing.T) {
	testCases := []struct {
		node     *ast_domain.TemplateNode
		name     string
		position protocol.Position
		expected bool
	}{
		{
			name:     "empty node returns false",
			node:     &ast_domain.TemplateNode{},
			position: protocol.Position{Line: 0, Character: 5},
			expected: false,
		},
		{
			name: "matches standard directive",
			node: &ast_domain.TemplateNode{
				DirIf: &ast_domain.Directive{
					AttributeRange: ast_domain.Range{
						Start: ast_domain.Location{Line: 1, Column: 1},
						End:   ast_domain.Location{Line: 1, Column: 20},
					},
				},
			},
			position: protocol.Position{Line: 0, Character: 5},
			expected: true,
		},
		{
			name: "matches bind directive",
			node: &ast_domain.TemplateNode{
				Binds: map[string]*ast_domain.Directive{
					"class": {
						Arg: "class",
						AttributeRange: ast_domain.Range{
							Start: ast_domain.Location{Line: 1, Column: 1},
							End:   ast_domain.Location{Line: 1, Column: 20},
						},
					},
				},
			},
			position: protocol.Position{Line: 0, Character: 5},
			expected: true,
		},
		{
			name: "matches event directive",
			node: &ast_domain.TemplateNode{
				OnEvents: map[string][]ast_domain.Directive{
					"click": {
						{
							Arg: "click",
							AttributeRange: ast_domain.Range{
								Start: ast_domain.Location{Line: 1, Column: 1},
								End:   ast_domain.Location{Line: 1, Column: 20},
							},
						},
					},
				},
			},
			position: protocol.Position{Line: 0, Character: 5},
			expected: true,
		},
		{
			name: "no match across all directive types",
			node: &ast_domain.TemplateNode{
				DirIf: &ast_domain.Directive{
					AttributeRange: ast_domain.Range{
						Start: ast_domain.Location{Line: 1, Column: 1},
						End:   ast_domain.Location{Line: 1, Column: 10},
					},
				},
				Binds: map[string]*ast_domain.Directive{
					"class": {
						AttributeRange: ast_domain.Range{
							Start: ast_domain.Location{Line: 2, Column: 1},
							End:   ast_domain.Location{Line: 2, Column: 10},
						},
					},
				},
			},
			position: protocol.Position{Line: 10, Character: 0},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := hasDirectiveAtPosition(context.Background(), tc.node, tc.position)
			if result != tc.expected {
				t.Errorf("hasDirectiveAtPosition() = %v, want %v", result, tc.expected)
			}
		})
	}
}
