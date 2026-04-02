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

func TestCalculateExpressionRange(t *testing.T) {
	testCases := []struct {
		name             string
		baseLocation     ast_domain.Location
		relativeLocation ast_domain.Location
		sourceLen        int
		expected         protocol.Range
	}{
		{
			name:             "simple identifier at start",
			baseLocation:     ast_domain.Location{Line: 1, Column: 1},
			relativeLocation: ast_domain.Location{Line: 0, Column: 0},
			sourceLen:        5,
			expected: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 5},
			},
		},
		{
			name:             "identifier with offset",
			baseLocation:     ast_domain.Location{Line: 5, Column: 10},
			relativeLocation: ast_domain.Location{Line: 0, Column: 0},
			sourceLen:        3,
			expected: protocol.Range{
				Start: protocol.Position{Line: 4, Character: 9},
				End:   protocol.Position{Line: 4, Character: 12},
			},
		},
		{
			name:             "expression with relative location on same line",
			baseLocation:     ast_domain.Location{Line: 2, Column: 5},
			relativeLocation: ast_domain.Location{Line: 1, Column: 3},
			sourceLen:        4,
			expected: protocol.Range{
				Start: protocol.Position{Line: 1, Character: 6},
				End:   protocol.Position{Line: 1, Character: 10},
			},
		},
		{
			name:             "multi-line relative location",
			baseLocation:     ast_domain.Location{Line: 1, Column: 1},
			relativeLocation: ast_domain.Location{Line: 3, Column: 6},
			sourceLen:        6,
			expected: protocol.Range{
				Start: protocol.Position{Line: 2, Character: 5},
				End:   protocol.Position{Line: 2, Character: 11},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			expression := &ast_domain.Identifier{
				Name:             "test",
				RelativeLocation: tc.relativeLocation,
				SourceLength:     tc.sourceLen,
			}

			result := calculateExpressionRange(expression, tc.baseLocation)

			if result.Start.Line != tc.expected.Start.Line {
				t.Errorf("Start.Line = %d, want %d", result.Start.Line, tc.expected.Start.Line)
			}
			if result.Start.Character != tc.expected.Start.Character {
				t.Errorf("Start.Character = %d, want %d", result.Start.Character, tc.expected.Start.Character)
			}
			if result.End.Line != tc.expected.End.Line {
				t.Errorf("End.Line = %d, want %d", result.End.Line, tc.expected.End.Line)
			}
			if result.End.Character != tc.expected.End.Character {
				t.Errorf("End.Character = %d, want %d", result.End.Character, tc.expected.End.Character)
			}
		})
	}
}

func TestVisitExpressionTree(t *testing.T) {
	t.Run("nil expression", func(t *testing.T) {
		var visited []string
		visitExpressionTree(nil, func(e ast_domain.Expression) {
			visited = append(visited, e.String())
		})
		if len(visited) != 0 {
			t.Errorf("expected no visits for nil, got %d", len(visited))
		}
	})

	t.Run("single identifier", func(t *testing.T) {
		identifier := &ast_domain.Identifier{Name: "foo"}
		var visited []string
		visitExpressionTree(identifier, func(e ast_domain.Expression) {
			visited = append(visited, e.String())
		})
		if len(visited) != 1 {
			t.Errorf("expected 1 visit, got %d", len(visited))
		}
		if visited[0] != "foo" {
			t.Errorf("expected 'foo', got %q", visited[0])
		}
	})

	t.Run("member expression", func(t *testing.T) {
		expression := &ast_domain.MemberExpression{
			Base:     &ast_domain.Identifier{Name: "obj"},
			Property: &ast_domain.Identifier{Name: "prop"},
		}
		var visited []string
		visitExpressionTree(expression, func(e ast_domain.Expression) {
			visited = append(visited, e.String())
		})

		if len(visited) != 3 {
			t.Errorf("expected 3 visits, got %d: %v", len(visited), visited)
		}
	})

	t.Run("binary expression", func(t *testing.T) {
		expression := &ast_domain.BinaryExpression{
			Left:     &ast_domain.Identifier{Name: "a"},
			Operator: "+",
			Right:    &ast_domain.Identifier{Name: "b"},
		}
		var visited []string
		visitExpressionTree(expression, func(e ast_domain.Expression) {
			visited = append(visited, e.String())
		})

		if len(visited) != 3 {
			t.Errorf("expected 3 visits, got %d: %v", len(visited), visited)
		}
	})

	t.Run("call expression with arguments", func(t *testing.T) {
		expression := &ast_domain.CallExpression{
			Callee: &ast_domain.Identifier{Name: "func"},
			Args: []ast_domain.Expression{
				&ast_domain.Identifier{Name: "arg1"},
				&ast_domain.Identifier{Name: "arg2"},
			},
		}
		var visited []string
		visitExpressionTree(expression, func(e ast_domain.Expression) {
			visited = append(visited, e.String())
		})

		if len(visited) != 4 {
			t.Errorf("expected 4 visits, got %d: %v", len(visited), visited)
		}
	})

	t.Run("ternary expression", func(t *testing.T) {
		expression := &ast_domain.TernaryExpression{
			Condition:  &ast_domain.Identifier{Name: "cond"},
			Consequent: &ast_domain.Identifier{Name: "then"},
			Alternate:  &ast_domain.Identifier{Name: "else"},
		}
		var visited []string
		visitExpressionTree(expression, func(e ast_domain.Expression) {
			visited = append(visited, e.String())
		})

		if len(visited) != 4 {
			t.Errorf("expected 4 visits, got %d: %v", len(visited), visited)
		}
	})

	t.Run("index expression", func(t *testing.T) {
		expression := &ast_domain.IndexExpression{
			Base:  &ast_domain.Identifier{Name: "arr"},
			Index: &ast_domain.IntegerLiteral{Value: 0},
		}
		var visited []string
		visitExpressionTree(expression, func(e ast_domain.Expression) {
			visited = append(visited, e.String())
		})

		if len(visited) != 3 {
			t.Errorf("expected 3 visits, got %d: %v", len(visited), visited)
		}
	})

	t.Run("unary expression", func(t *testing.T) {
		expression := &ast_domain.UnaryExpression{
			Operator: "!",
			Right:    &ast_domain.Identifier{Name: "x"},
		}
		var visited []string
		visitExpressionTree(expression, func(e ast_domain.Expression) {
			visited = append(visited, e.String())
		})

		if len(visited) != 2 {
			t.Errorf("expected 2 visits, got %d: %v", len(visited), visited)
		}
	})

	t.Run("object literal", func(t *testing.T) {
		expression := &ast_domain.ObjectLiteral{
			Pairs: map[string]ast_domain.Expression{
				"key": &ast_domain.Identifier{Name: "value"},
			},
		}
		var visited []string
		visitExpressionTree(expression, func(e ast_domain.Expression) {
			visited = append(visited, e.String())
		})

		if len(visited) != 2 {
			t.Errorf("expected 2 visits, got %d: %v", len(visited), visited)
		}
	})

	t.Run("array literal", func(t *testing.T) {
		expression := &ast_domain.ArrayLiteral{
			Elements: []ast_domain.Expression{
				&ast_domain.Identifier{Name: "a"},
				&ast_domain.Identifier{Name: "b"},
			},
		}
		var visited []string
		visitExpressionTree(expression, func(e ast_domain.Expression) {
			visited = append(visited, e.String())
		})

		if len(visited) != 3 {
			t.Errorf("expected 3 visits, got %d: %v", len(visited), visited)
		}
	})

	t.Run("for-in expression", func(t *testing.T) {
		expression := &ast_domain.ForInExpression{
			ItemVariable: &ast_domain.Identifier{Name: "item"},
			Collection:   &ast_domain.Identifier{Name: "items"},
		}
		var visited []string
		visitExpressionTree(expression, func(e ast_domain.Expression) {
			visited = append(visited, e.String())
		})

		if len(visited) != 3 {
			t.Errorf("expected 3 visits, got %d: %v", len(visited), visited)
		}
	})

	t.Run("for-in expression with index", func(t *testing.T) {
		expression := &ast_domain.ForInExpression{
			IndexVariable: &ast_domain.Identifier{Name: "index"},
			ItemVariable:  &ast_domain.Identifier{Name: "item"},
			Collection:    &ast_domain.Identifier{Name: "items"},
		}
		var visited []string
		visitExpressionTree(expression, func(e ast_domain.Expression) {
			visited = append(visited, e.String())
		})

		if len(visited) != 4 {
			t.Errorf("expected 4 visits, got %d: %v", len(visited), visited)
		}
	})

	t.Run("linked message expression", func(t *testing.T) {
		expression := &ast_domain.LinkedMessageExpression{
			Path: &ast_domain.Identifier{Name: "message.key"},
		}
		var visited []string
		visitExpressionTree(expression, func(e ast_domain.Expression) {
			visited = append(visited, e.String())
		})

		if len(visited) != 2 {
			t.Errorf("expected 2 visits, got %d: %v", len(visited), visited)
		}
	})
}

func TestFindMostSpecificExpression(t *testing.T) {
	t.Run("finds inner identifier in member expr", func(t *testing.T) {

		base := &ast_domain.Identifier{
			Name:             "obj",
			RelativeLocation: ast_domain.Location{Line: 1, Column: 1},
			SourceLength:     3,
		}
		prop := &ast_domain.Identifier{
			Name:             "prop",
			RelativeLocation: ast_domain.Location{Line: 1, Column: 5},
			SourceLength:     4,
		}
		memberExpr := &ast_domain.MemberExpression{
			Base:             base,
			Property:         prop,
			RelativeLocation: ast_domain.Location{Line: 1, Column: 1},
			SourceLength:     8,
		}

		baseLocation := ast_domain.Location{Line: 1, Column: 1}

		position := protocol.Position{Line: 0, Character: 5}

		result := findMostSpecificExpression(memberExpr, baseLocation, position)

		if result.bestMatch == nil {
			t.Fatal("expected to find a match")
		}
		identifier, ok := result.bestMatch.(*ast_domain.Identifier)
		if !ok {
			t.Fatalf("expected Identifier, got %T", result.bestMatch)
		}
		if identifier.Name != "prop" {
			t.Errorf("expected 'prop', got %q", identifier.Name)
		}
	})

	t.Run("tracks member context", func(t *testing.T) {
		base := &ast_domain.Identifier{
			Name:             "obj",
			RelativeLocation: ast_domain.Location{Line: 1, Column: 1},
			SourceLength:     3,
		}
		prop := &ast_domain.Identifier{
			Name:             "method",
			RelativeLocation: ast_domain.Location{Line: 1, Column: 5},
			SourceLength:     6,
		}
		memberExpr := &ast_domain.MemberExpression{
			Base:             base,
			Property:         prop,
			RelativeLocation: ast_domain.Location{Line: 1, Column: 1},
			SourceLength:     10,
		}

		baseLocation := ast_domain.Location{Line: 1, Column: 1}

		position := protocol.Position{Line: 0, Character: 5}

		result := findMostSpecificExpression(memberExpr, baseLocation, position)

		if result.memberContext == nil {
			t.Error("expected memberContext to be set when cursor is on property")
		}
		if result.memberContext != memberExpr {
			t.Error("memberContext should be the containing MemberExpr")
		}
	})

	t.Run("no match outside expression range", func(t *testing.T) {
		expression := &ast_domain.Identifier{
			Name:             "foo",
			RelativeLocation: ast_domain.Location{Line: 1, Column: 1},
			SourceLength:     3,
		}

		baseLocation := ast_domain.Location{Line: 1, Column: 1}

		position := protocol.Position{Line: 0, Character: 100}

		result := findMostSpecificExpression(expression, baseLocation, position)

		if result.bestRange.End.Character != maxRangeValue {
			t.Error("expected default max range when no match found")
		}
	})
}

func TestCalculateAttrSourceLen(t *testing.T) {
	testCases := []struct {
		name     string
		attr     ast_domain.HTMLAttribute
		expected int
	}{
		{
			name: "single line attribute",
			attr: ast_domain.HTMLAttribute{
				Value: "test",
				Location: ast_domain.Location{
					Line:   1,
					Column: 10,
				},
				AttributeRange: ast_domain.Range{
					Start: ast_domain.Location{Line: 1, Column: 5},
					End:   ast_domain.Location{Line: 1, Column: 16},
				},
			},
			expected: 5,
		},
		{
			name: "multi-line attribute",
			attr: ast_domain.HTMLAttribute{
				Value: "multi\nline\nvalue",
				Location: ast_domain.Location{
					Line:   1,
					Column: 10,
				},
				AttributeRange: ast_domain.Range{
					Start: ast_domain.Location{Line: 1, Column: 5},
					End:   ast_domain.Location{Line: 3, Column: 10},
				},
			},
			expected: 16,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := calculateAttrSourceLen(&tc.attr)
			if result != tc.expected {
				t.Errorf("calculateAttrSourceLen() = %d, want %d", result, tc.expected)
			}
		})
	}
}

func TestCreateSyntheticStringLiteral(t *testing.T) {
	attr := &ast_domain.HTMLAttribute{
		Name:  "class",
		Value: "container flex",
	}
	sourceLen := 14

	result := createSyntheticStringLiteral(attr, sourceLen)

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if result.Value != "container flex" {
		t.Errorf("Value = %q, want %q", result.Value, "container flex")
	}

	if result.SourceLength != 14 {
		t.Errorf("SourceLength = %d, want %d", result.SourceLength, 14)
	}

	if result.GoAnnotations == nil {
		t.Fatal("expected GoAnnotations to be set")
	}

	if result.GoAnnotations.ResolvedType == nil {
		t.Fatal("expected ResolvedType to be set")
	}

	if result.GoAnnotations.Symbol == nil {
		t.Fatal("expected Symbol to be set")
	}

	if result.GoAnnotations.Symbol.Name != "class" {
		t.Errorf("Symbol.Name = %q, want %q", result.GoAnnotations.Symbol.Name, "class")
	}
}

func TestCopyAnnotationToExpression(t *testing.T) {
	t.Run("nil expression", func(t *testing.T) {
		ann := &ast_domain.GoGeneratorAnnotation{}

		copyAnnotationToExpression(nil, ann)
	})

	t.Run("nil annotation", func(t *testing.T) {
		expression := &ast_domain.Identifier{Name: "test"}

		copyAnnotationToExpression(expression, nil)
	})

	t.Run("expression without annotation gets annotation set", func(t *testing.T) {
		expression := &ast_domain.Identifier{Name: "test"}
		ann := &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				CanonicalPackagePath: "example.com/pkg",
			},
		}

		copyAnnotationToExpression(expression, ann)

		if expression.GoAnnotations != ann {
			t.Error("expected annotation to be set on expression")
		}
	})

	t.Run("expression with annotation merges fields", func(t *testing.T) {
		existingType := &ast_domain.ResolvedTypeInfo{
			CanonicalPackagePath: "existing",
		}
		expression := &ast_domain.Identifier{
			Name: "test",
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: existingType,
			},
		}
		newSymbol := &ast_domain.ResolvedSymbol{Name: "newSymbol"}
		ann := &ast_domain.GoGeneratorAnnotation{
			Symbol: newSymbol,
		}

		copyAnnotationToExpression(expression, ann)

		if expression.GoAnnotations.ResolvedType != existingType {
			t.Error("expected existing ResolvedType to be preserved")
		}

		if expression.GoAnnotations.Symbol != newSymbol {
			t.Error("expected Symbol to be merged")
		}
	})

	t.Run("expression with nil fields gets new values", func(t *testing.T) {
		expression := &ast_domain.Identifier{
			Name:          "test",
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{},
		}
		newType := &ast_domain.ResolvedTypeInfo{CanonicalPackagePath: "new"}
		newSymbol := &ast_domain.ResolvedSymbol{Name: "sym"}
		ann := &ast_domain.GoGeneratorAnnotation{
			ResolvedType: newType,
			Symbol:       newSymbol,
		}

		copyAnnotationToExpression(expression, ann)

		if expression.GoAnnotations.ResolvedType != newType {
			t.Error("expected ResolvedType to be set")
		}
		if expression.GoAnnotations.Symbol != newSymbol {
			t.Error("expected Symbol to be set")
		}
	})
}

func TestIsPositionInAttributeValue(t *testing.T) {
	testCases := []struct {
		name           string
		position       protocol.Position
		attributeRange ast_domain.Range
		expected       bool
	}{
		{
			name:     "position inside range",
			position: protocol.Position{Line: 0, Character: 10},
			attributeRange: ast_domain.Range{
				Start: ast_domain.Location{Line: 1, Column: 5},
				End:   ast_domain.Location{Line: 1, Column: 20},
			},
			expected: true,
		},
		{
			name:     "position outside range",
			position: protocol.Position{Line: 0, Character: 25},
			attributeRange: ast_domain.Range{
				Start: ast_domain.Location{Line: 1, Column: 5},
				End:   ast_domain.Location{Line: 1, Column: 20},
			},
			expected: false,
		},
		{
			name:     "synthetic range",
			position: protocol.Position{Line: 0, Character: 10},
			attributeRange: ast_domain.Range{
				Start: ast_domain.Location{Line: 0, Column: 0},
				End:   ast_domain.Location{Line: 1, Column: 20},
			},
			expected: false,
		},
		{
			name:     "position on different line",
			position: protocol.Position{Line: 5, Character: 10},
			attributeRange: ast_domain.Range{
				Start: ast_domain.Location{Line: 1, Column: 5},
				End:   ast_domain.Location{Line: 1, Column: 20},
			},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isPositionInAttributeValue(tc.position, &tc.attributeRange)
			if result != tc.expected {
				t.Errorf("isPositionInAttributeValue() = %v, want %v", result, tc.expected)
			}
		})
	}
}

func TestCheckDirectiveAtPos(t *testing.T) {
	testCases := []struct {
		directive *ast_domain.Directive
		name      string
		position  protocol.Position
		expectOK  bool
	}{
		{
			name:      "nil directive",
			directive: nil,
			position:  protocol.Position{Line: 0, Character: 0},
			expectOK:  false,
		},
		{
			name: "synthetic range",
			directive: &ast_domain.Directive{
				AttributeRange: ast_domain.Range{
					Start: ast_domain.Location{Line: 0, Column: 0},
					End:   ast_domain.Location{Line: 1, Column: 10},
				},
			},
			position: protocol.Position{Line: 0, Character: 5},
			expectOK: false,
		},
		{
			name: "position in range",
			directive: &ast_domain.Directive{
				Expression: &ast_domain.Identifier{Name: "test"},
				Location:   ast_domain.Location{Line: 1, Column: 5},
				AttributeRange: ast_domain.Range{
					Start: ast_domain.Location{Line: 1, Column: 1},
					End:   ast_domain.Location{Line: 1, Column: 20},
				},
			},
			position: protocol.Position{Line: 0, Character: 10},
			expectOK: true,
		},
		{
			name: "position outside range",
			directive: &ast_domain.Directive{
				Expression: &ast_domain.Identifier{Name: "test"},
				AttributeRange: ast_domain.Range{
					Start: ast_domain.Location{Line: 1, Column: 1},
					End:   ast_domain.Location{Line: 1, Column: 10},
				},
			},
			position: protocol.Position{Line: 0, Character: 50},
			expectOK: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			expression, loc, ok := checkDirectiveAtPos(tc.directive, tc.position)
			if ok != tc.expectOK {
				t.Errorf("checkDirectiveAtPos() ok = %v, want %v", ok, tc.expectOK)
			}
			if tc.expectOK && tc.directive != nil {
				if expression != tc.directive.Expression {
					t.Error("expected expression to match directive expression")
				}
				if loc != tc.directive.Location {
					t.Error("expected location to match directive location")
				}
			}
		})
	}
}

func TestFindExprInDynamicAttrs(t *testing.T) {
	testCases := []struct {
		name      string
		attrs     []ast_domain.DynamicAttribute
		position  protocol.Position
		expectNil bool
	}{
		{
			name:      "empty attrs",
			attrs:     nil,
			position:  protocol.Position{Line: 0, Character: 0},
			expectNil: true,
		},
		{
			name: "position in attr range",
			attrs: []ast_domain.DynamicAttribute{
				{
					Expression: &ast_domain.Identifier{Name: "value"},
					Location:   ast_domain.Location{Line: 1, Column: 10},
					AttributeRange: ast_domain.Range{
						Start: ast_domain.Location{Line: 1, Column: 5},
						End:   ast_domain.Location{Line: 1, Column: 20},
					},
				},
			},
			position:  protocol.Position{Line: 0, Character: 10},
			expectNil: false,
		},
		{
			name: "position outside attr range",
			attrs: []ast_domain.DynamicAttribute{
				{
					Expression: &ast_domain.Identifier{Name: "value"},
					AttributeRange: ast_domain.Range{
						Start: ast_domain.Location{Line: 1, Column: 5},
						End:   ast_domain.Location{Line: 1, Column: 10},
					},
				},
			},
			position:  protocol.Position{Line: 0, Character: 50},
			expectNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			expression, _ := findExprInDynamicAttrs(tc.attrs, tc.position)
			if tc.expectNil && expression != nil {
				t.Error("expected nil expression")
			}
			if !tc.expectNil && expression == nil {
				t.Error("expected non-nil expression")
			}
		})
	}
}

func TestFindExprInStaticAttrs(t *testing.T) {
	testCases := []struct {
		name      string
		attrs     []ast_domain.HTMLAttribute
		position  protocol.Position
		expectNil bool
	}{
		{
			name:      "empty attrs",
			attrs:     nil,
			position:  protocol.Position{Line: 0, Character: 0},
			expectNil: true,
		},
		{
			name: "position in attr range",
			attrs: []ast_domain.HTMLAttribute{
				{
					Name:     "class",
					Value:    "container",
					Location: ast_domain.Location{Line: 1, Column: 10},
					AttributeRange: ast_domain.Range{
						Start: ast_domain.Location{Line: 1, Column: 5},
						End:   ast_domain.Location{Line: 1, Column: 25},
					},
				},
			},
			position:  protocol.Position{Line: 0, Character: 15},
			expectNil: false,
		},
		{
			name: "position outside attr range",
			attrs: []ast_domain.HTMLAttribute{
				{
					Name:  "class",
					Value: "container",
					AttributeRange: ast_domain.Range{
						Start: ast_domain.Location{Line: 1, Column: 5},
						End:   ast_domain.Location{Line: 1, Column: 10},
					},
				},
			},
			position:  protocol.Position{Line: 0, Character: 50},
			expectNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			expression, _ := findExprInStaticAttrs(tc.attrs, tc.position)
			if tc.expectNil && expression != nil {
				t.Error("expected nil expression")
			}
			if !tc.expectNil && expression == nil {
				t.Error("expected non-nil expression")
			}

			if !tc.expectNil && expression != nil {
				if _, ok := expression.(*ast_domain.StringLiteral); !ok {
					t.Errorf("expected StringLiteral, got %T", expression)
				}
			}
		})
	}
}

func TestFindExprInRichText(t *testing.T) {
	testCases := []struct {
		name      string
		parts     []ast_domain.TextPart
		position  protocol.Position
		expectNil bool
	}{
		{
			name:      "empty parts",
			parts:     nil,
			position:  protocol.Position{Line: 0, Character: 0},
			expectNil: true,
		},
		{
			name: "literal part only",
			parts: []ast_domain.TextPart{
				{
					IsLiteral: true,
					Literal:   "plain text",
				},
			},
			position:  protocol.Position{Line: 0, Character: 5},
			expectNil: true,
		},
		{
			name: "expression part with position match",
			parts: []ast_domain.TextPart{
				{
					IsLiteral:     false,
					RawExpression: "value",
					Expression:    &ast_domain.Identifier{Name: "value"},
					Location:      ast_domain.Location{Line: 1, Column: 10},
				},
			},
			position:  protocol.Position{Line: 0, Character: 10},
			expectNil: false,
		},
		{
			name: "expression part with position outside",
			parts: []ast_domain.TextPart{
				{
					IsLiteral:     false,
					RawExpression: "value",
					Expression:    &ast_domain.Identifier{Name: "value"},
					Location:      ast_domain.Location{Line: 1, Column: 10},
				},
			},
			position:  protocol.Position{Line: 0, Character: 100},
			expectNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			expression, _ := findExprInRichText(context.Background(), tc.parts, tc.position)
			if tc.expectNil && expression != nil {
				t.Error("expected nil expression")
			}
			if !tc.expectNil && expression == nil {
				t.Error("expected non-nil expression")
			}
		})
	}
}

func TestVisitForInExprChildren(t *testing.T) {
	var visited []string
	visitor := func(e ast_domain.Expression) {
		if id, ok := e.(*ast_domain.Identifier); ok {
			visited = append(visited, id.Name)
		}
	}

	testCases := []struct {
		name       string
		expression *ast_domain.ForInExpression
		expected   []string
	}{
		{
			name: "all fields present",
			expression: &ast_domain.ForInExpression{
				IndexVariable: &ast_domain.Identifier{Name: "index"},
				ItemVariable:  &ast_domain.Identifier{Name: "item"},
				Collection:    &ast_domain.Identifier{Name: "items"},
			},
			expected: []string{"index", "item", "items"},
		},
		{
			name: "nil index variable",
			expression: &ast_domain.ForInExpression{
				IndexVariable: nil,
				ItemVariable:  &ast_domain.Identifier{Name: "val"},
				Collection:    &ast_domain.Identifier{Name: "list"},
			},
			expected: []string{"val", "list"},
		},
		{
			name: "nil item variable",
			expression: &ast_domain.ForInExpression{
				IndexVariable: &ast_domain.Identifier{Name: "i"},
				ItemVariable:  nil,
				Collection:    &ast_domain.Identifier{Name: "arr"},
			},
			expected: []string{"i", "arr"},
		},
		{
			name: "only collection",
			expression: &ast_domain.ForInExpression{
				IndexVariable: nil,
				ItemVariable:  nil,
				Collection:    &ast_domain.Identifier{Name: "data"},
			},
			expected: []string{"data"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			visited = nil
			visitForInExprChildren(tc.expression, visitor)
			if len(visited) != len(tc.expected) {
				t.Fatalf("visited %d nodes, want %d: %v", len(visited), len(tc.expected), visited)
			}
			for i, name := range tc.expected {
				if visited[i] != name {
					t.Errorf("visited[%d] = %q, want %q", i, visited[i], name)
				}
			}
		})
	}
}

func TestVisitExpressionChildren(t *testing.T) {
	testCases := []struct {
		name       string
		expression ast_domain.Expression
		expected   []string
	}{
		{
			name: "MemberExpr visits base and property",
			expression: &ast_domain.MemberExpression{
				Base:     &ast_domain.Identifier{Name: "obj"},
				Property: &ast_domain.Identifier{Name: "prop"},
			},
			expected: []string{"obj", "prop"},
		},
		{
			name: "CallExpr visits callee and arguments",
			expression: &ast_domain.CallExpression{
				Callee: &ast_domain.Identifier{Name: "fn"},
				Args: []ast_domain.Expression{
					&ast_domain.Identifier{Name: "a"},
					&ast_domain.Identifier{Name: "b"},
				},
			},
			expected: []string{"fn", "a", "b"},
		},
		{
			name: "BinaryExpr visits left and right",
			expression: &ast_domain.BinaryExpression{
				Left:  &ast_domain.Identifier{Name: "x"},
				Right: &ast_domain.Identifier{Name: "y"},
			},
			expected: []string{"x", "y"},
		},
		{
			name: "UnaryExpr visits right",
			expression: &ast_domain.UnaryExpression{
				Right: &ast_domain.Identifier{Name: "val"},
			},
			expected: []string{"val"},
		},
		{
			name: "TernaryExpr visits condition consequent alternate",
			expression: &ast_domain.TernaryExpression{
				Condition:  &ast_domain.Identifier{Name: "cond"},
				Consequent: &ast_domain.Identifier{Name: "yes"},
				Alternate:  &ast_domain.Identifier{Name: "no"},
			},
			expected: []string{"cond", "yes", "no"},
		},
		{
			name: "IndexExpr visits base and index",
			expression: &ast_domain.IndexExpression{
				Base:  &ast_domain.Identifier{Name: "arr"},
				Index: &ast_domain.Identifier{Name: "i"},
			},
			expected: []string{"arr", "i"},
		},
		{
			name: "TemplateLiteral visits non-literal parts",
			expression: &ast_domain.TemplateLiteral{
				Parts: []ast_domain.TemplateLiteralPart{
					{IsLiteral: true, Expression: &ast_domain.Identifier{Name: "skip"}},
					{IsLiteral: false, Expression: &ast_domain.Identifier{Name: "interp"}},
				},
			},
			expected: []string{"interp"},
		},
		{
			name: "ObjectLiteral visits values",
			expression: &ast_domain.ObjectLiteral{
				Pairs: map[string]ast_domain.Expression{
					"key": &ast_domain.Identifier{Name: "val"},
				},
			},
			expected: []string{"val"},
		},
		{
			name: "ArrayLiteral visits elements",
			expression: &ast_domain.ArrayLiteral{
				Elements: []ast_domain.Expression{
					&ast_domain.Identifier{Name: "el1"},
					&ast_domain.Identifier{Name: "el2"},
				},
			},
			expected: []string{"el1", "el2"},
		},
		{
			name:       "Identifier has no children",
			expression: &ast_domain.Identifier{Name: "leaf"},
			expected:   []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var visited []string
			visitExpressionChildren(tc.expression, func(e ast_domain.Expression) {
				if id, ok := e.(*ast_domain.Identifier); ok {
					visited = append(visited, id.Name)
				}
			})
			if len(visited) != len(tc.expected) {
				t.Fatalf("visited %d nodes, want %d: %v", len(visited), len(tc.expected), visited)
			}
			for i, name := range tc.expected {
				if visited[i] != name {
					t.Errorf("visited[%d] = %q, want %q", i, visited[i], name)
				}
			}
		})
	}
}

func TestFindTopLevelExpressionOnNode(t *testing.T) {
	testCases := []struct {
		node      *ast_domain.TemplateNode
		name      string
		position  protocol.Position
		expectNil bool
	}{
		{
			name:      "empty node returns nil",
			node:      &ast_domain.TemplateNode{},
			position:  protocol.Position{Line: 0, Character: 5},
			expectNil: true,
		},
		{
			name: "finds expression in dynamic attribute",
			node: &ast_domain.TemplateNode{
				DynamicAttributes: []ast_domain.DynamicAttribute{
					{
						Expression: &ast_domain.Identifier{Name: "val"},
						Location:   ast_domain.Location{Line: 1, Column: 5},
						AttributeRange: ast_domain.Range{
							Start: ast_domain.Location{Line: 1, Column: 1},
							End:   ast_domain.Location{Line: 1, Column: 20},
						},
					},
				},
			},
			position:  protocol.Position{Line: 0, Character: 5},
			expectNil: false,
		},
		{
			name: "finds expression in directive",
			node: &ast_domain.TemplateNode{
				DirIf: &ast_domain.Directive{
					Expression: &ast_domain.Identifier{Name: "cond"},
					Location:   ast_domain.Location{Line: 1, Column: 10},
					AttributeRange: ast_domain.Range{
						Start: ast_domain.Location{Line: 1, Column: 1},
						End:   ast_domain.Location{Line: 1, Column: 25},
					},
				},
			},
			position:  protocol.Position{Line: 0, Character: 10},
			expectNil: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			expression, _ := findTopLevelExpressionOnNode(context.Background(), tc.node, tc.position)
			if tc.expectNil && expression != nil {
				t.Errorf("expected nil, got %T", expression)
			}
			if !tc.expectNil && expression == nil {
				t.Error("expected non-nil expression")
			}
		})
	}
}

func TestFindExprInPassedProps(t *testing.T) {
	testCases := []struct {
		node      *ast_domain.TemplateNode
		name      string
		position  protocol.Position
		expectNil bool
	}{
		{
			name:      "nil GoAnnotations returns nil",
			node:      &ast_domain.TemplateNode{},
			position:  protocol.Position{Line: 0, Character: 5},
			expectNil: true,
		},
		{
			name: "nil PartialInfo returns nil",
			node: &ast_domain.TemplateNode{
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{},
			},
			position:  protocol.Position{Line: 0, Character: 5},
			expectNil: true,
		},
		{
			name: "synthetic location skipped",
			node: &ast_domain.TemplateNode{
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					PartialInfo: &ast_domain.PartialInvocationInfo{
						PassedProps: map[string]ast_domain.PropValue{
							"name": {
								Expression:   &ast_domain.Identifier{Name: "val"},
								NameLocation: ast_domain.Location{Line: 0, Column: 0},
								Location:     ast_domain.Location{Line: 1, Column: 5},
							},
						},
					},
				},
			},
			position:  protocol.Position{Line: 0, Character: 5},
			expectNil: true,
		},
		{
			name: "wrong line returns nil",
			node: &ast_domain.TemplateNode{
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					PartialInfo: &ast_domain.PartialInvocationInfo{
						PassedProps: map[string]ast_domain.PropValue{
							"name": {
								Expression:   &ast_domain.Identifier{Name: "val"},
								NameLocation: ast_domain.Location{Line: 5, Column: 3},
								Location:     ast_domain.Location{Line: 5, Column: 3},
							},
						},
					},
				},
			},
			position:  protocol.Position{Line: 0, Character: 5},
			expectNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			expression, _ := findExprInPassedProps(context.Background(), tc.node, tc.position)
			if tc.expectNil && expression != nil {
				t.Errorf("expected nil, got %T", expression)
			}
			if !tc.expectNil && expression == nil {
				t.Error("expected non-nil expression")
			}
		})
	}
}

func TestFindExprInNodeDirectives(t *testing.T) {
	testCases := []struct {
		node      *ast_domain.TemplateNode
		name      string
		position  protocol.Position
		expectNil bool
	}{
		{
			name:      "empty node returns nil",
			node:      &ast_domain.TemplateNode{},
			position:  protocol.Position{Line: 0, Character: 5},
			expectNil: true,
		},
		{
			name: "finds DirIf",
			node: &ast_domain.TemplateNode{
				DirIf: &ast_domain.Directive{
					Expression: &ast_domain.Identifier{Name: "show"},
					Location:   ast_domain.Location{Line: 1, Column: 5},
					AttributeRange: ast_domain.Range{
						Start: ast_domain.Location{Line: 1, Column: 1},
						End:   ast_domain.Location{Line: 1, Column: 20},
					},
				},
			},
			position:  protocol.Position{Line: 0, Character: 5},
			expectNil: false,
		},
		{
			name: "finds bind directive",
			node: &ast_domain.TemplateNode{
				Binds: map[string]*ast_domain.Directive{
					"class": {
						Expression: &ast_domain.Identifier{Name: "cls"},
						Location:   ast_domain.Location{Line: 2, Column: 3},
						AttributeRange: ast_domain.Range{
							Start: ast_domain.Location{Line: 2, Column: 1},
							End:   ast_domain.Location{Line: 2, Column: 25},
						},
					},
				},
			},
			position:  protocol.Position{Line: 1, Character: 10},
			expectNil: false,
		},
		{
			name: "finds event directive",
			node: &ast_domain.TemplateNode{
				OnEvents: map[string][]ast_domain.Directive{
					"click": {
						{
							Expression: &ast_domain.Identifier{Name: "handler"},
							Location:   ast_domain.Location{Line: 3, Column: 5},
							AttributeRange: ast_domain.Range{
								Start: ast_domain.Location{Line: 3, Column: 1},
								End:   ast_domain.Location{Line: 3, Column: 30},
							},
						},
					},
				},
			},
			position:  protocol.Position{Line: 2, Character: 10},
			expectNil: false,
		},
		{
			name: "position outside all directives",
			node: &ast_domain.TemplateNode{
				DirIf: &ast_domain.Directive{
					Expression: &ast_domain.Identifier{Name: "cond"},
					Location:   ast_domain.Location{Line: 1, Column: 5},
					AttributeRange: ast_domain.Range{
						Start: ast_domain.Location{Line: 1, Column: 1},
						End:   ast_domain.Location{Line: 1, Column: 20},
					},
				},
			},
			position:  protocol.Position{Line: 5, Character: 10},
			expectNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			expression, _ := findExprInNodeDirectives(tc.node, tc.position)
			if tc.expectNil && expression != nil {
				t.Errorf("expected nil, got %T", expression)
			}
			if !tc.expectNil && expression == nil {
				t.Error("expected non-nil expression")
			}
		})
	}
}

func TestVisitForInExprChildrenWithContext(t *testing.T) {

	indexVar := &ast_domain.Identifier{
		Name:             "index",
		RelativeLocation: ast_domain.Location{Line: 1, Column: 1},
		SourceLength:     3,
	}
	itemVar := &ast_domain.Identifier{
		Name:             "item",
		RelativeLocation: ast_domain.Location{Line: 1, Column: 6},
		SourceLength:     4,
	}
	collection := &ast_domain.Identifier{
		Name:             "items",
		RelativeLocation: ast_domain.Location{Line: 1, Column: 14},
		SourceLength:     5,
	}

	testCases := []struct {
		forInExpr    *ast_domain.ForInExpression
		name         string
		position     protocol.Position
		expectNonNil bool
	}{
		{
			name: "cursor on collection finds expression",
			forInExpr: &ast_domain.ForInExpression{
				IndexVariable:    indexVar,
				ItemVariable:     itemVar,
				Collection:       collection,
				RelativeLocation: ast_domain.Location{Line: 1, Column: 1},
				SourceLength:     18,
			},

			position:     protocol.Position{Line: 0, Character: 14},
			expectNonNil: true,
		},
		{
			name: "cursor on item variable finds expression",
			forInExpr: &ast_domain.ForInExpression{
				IndexVariable:    indexVar,
				ItemVariable:     itemVar,
				Collection:       collection,
				RelativeLocation: ast_domain.Location{Line: 1, Column: 1},
				SourceLength:     18,
			},

			position:     protocol.Position{Line: 0, Character: 6},
			expectNonNil: true,
		},
		{
			name: "nil index variable does not panic",
			forInExpr: &ast_domain.ForInExpression{
				IndexVariable:    nil,
				ItemVariable:     itemVar,
				Collection:       collection,
				RelativeLocation: ast_domain.Location{Line: 1, Column: 1},
				SourceLength:     14,
			},
			position:     protocol.Position{Line: 0, Character: 6},
			expectNonNil: true,
		},
		{
			name: "nil item variable does not panic",
			forInExpr: &ast_domain.ForInExpression{
				IndexVariable:    indexVar,
				ItemVariable:     nil,
				Collection:       collection,
				RelativeLocation: ast_domain.Location{Line: 1, Column: 1},
				SourceLength:     18,
			},
			position:     protocol.Position{Line: 0, Character: 14},
			expectNonNil: true,
		},
		{
			name: "cursor outside range finds nothing specific",
			forInExpr: &ast_domain.ForInExpression{
				IndexVariable:    indexVar,
				ItemVariable:     itemVar,
				Collection:       collection,
				RelativeLocation: ast_domain.Location{Line: 1, Column: 1},
				SourceLength:     18,
			},
			position:     protocol.Position{Line: 5, Character: 0},
			expectNonNil: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			baseLocation := ast_domain.Location{Line: 1, Column: 1}
			result := &expressionFindResult{
				bestRange: protocol.Range{
					Start: protocol.Position{Line: 0, Character: 0},
					End:   protocol.Position{Line: maxRangeValue, Character: maxRangeValue},
				},
			}

			visitForInExprChildrenWithContext(tc.forInExpr, baseLocation, tc.position, result)

			if tc.expectNonNil && result.bestMatch == nil {
				t.Error("expected bestMatch to be set")
			}
			if !tc.expectNonNil && result.bestMatch != nil {
				t.Errorf("expected bestMatch to be nil, got %T", result.bestMatch)
			}
		})
	}
}
