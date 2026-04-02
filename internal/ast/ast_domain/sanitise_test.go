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

package ast_domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSanitiseForEncoding(t *testing.T) {
	t.Parallel()

	t.Run("nil tree returns nil", func(t *testing.T) {
		t.Parallel()

		result := SanitiseForEncoding(nil, "/base")
		assert.Nil(t, result)
	})

	t.Run("empty base path returns cloned tree without path sanitisation", func(t *testing.T) {
		t.Parallel()

		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "div",
				},
			},
			Diagnostics: []*Diagnostic{{Message: "test"}},
		}

		result := SanitiseForEncoding(tree, "")

		require.NotNil(t, result)
		assert.NotSame(t, tree, result)
		assert.Nil(t, result.Diagnostics)
		assert.Len(t, result.RootNodes, 1)
	})

	t.Run("removes diagnostics from AST", func(t *testing.T) {
		t.Parallel()

		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType:    NodeElement,
					TagName:     "div",
					Diagnostics: []*Diagnostic{{Message: "node diagnostic"}},
				},
			},
			Diagnostics: []*Diagnostic{{Message: "ast diagnostic"}},
		}

		result := SanitiseForEncoding(tree, "/base")

		require.NotNil(t, result)
		assert.Nil(t, result.Diagnostics)
		assert.Nil(t, result.RootNodes[0].Diagnostics)
	})

	t.Run("sanitises absolute paths to relative", func(t *testing.T) {
		t.Parallel()

		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "div",
					GoAnnotations: &GoGeneratorAnnotation{
						OriginalSourcePath:  new("/base/path/to/file.pkc"),
						GeneratedSourcePath: new("/base/path/to/generated.go"),
					},
				},
			},
		}

		result := SanitiseForEncoding(tree, "/base/path")

		require.NotNil(t, result)
		require.NotNil(t, result.RootNodes[0].GoAnnotations)
		require.NotNil(t, result.RootNodes[0].GoAnnotations.OriginalSourcePath)
		assert.Equal(t, "to/file.pkc", *result.RootNodes[0].GoAnnotations.OriginalSourcePath)
		require.NotNil(t, result.RootNodes[0].GoAnnotations.GeneratedSourcePath)
		assert.Equal(t, "to/generated.go", *result.RootNodes[0].GoAnnotations.GeneratedSourcePath)
	})

	t.Run("sanitises nested directive expressions", func(t *testing.T) {
		t.Parallel()

		annotation := &GoGeneratorAnnotation{
			OriginalSourcePath: new("/home/user/project/file.pkc"),
		}
		expression := &Identifier{
			Name:          "visible",
			GoAnnotations: annotation,
		}
		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "div",
					DirIf: &Directive{
						Expression: expression,
					},
				},
			},
		}

		result := SanitiseForEncoding(tree, "/home/user/project")

		require.NotNil(t, result)
		require.NotNil(t, result.RootNodes[0].DirIf)
		require.NotNil(t, result.RootNodes[0].DirIf.Expression)
		identifier, ok := result.RootNodes[0].DirIf.Expression.(*Identifier)
		require.True(t, ok)
		require.NotNil(t, identifier.GoAnnotations)
		require.NotNil(t, identifier.GoAnnotations.OriginalSourcePath)
		assert.Equal(t, "file.pkc", *identifier.GoAnnotations.OriginalSourcePath)
	})

	t.Run("sanitises dynamic attributes", func(t *testing.T) {
		t.Parallel()

		annotation := &GoGeneratorAnnotation{
			OriginalSourcePath: new("/workspace/src/template.pkc"),
		}
		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "div",
					DynamicAttributes: []DynamicAttribute{
						{
							Name:          "class",
							RawExpression: "className",
							Expression:    &Identifier{Name: "className"},
							GoAnnotations: annotation,
						},
					},
				},
			},
		}

		result := SanitiseForEncoding(tree, "/workspace/src")

		require.NotNil(t, result)
		require.Len(t, result.RootNodes[0].DynamicAttributes, 1)
		require.NotNil(t, result.RootNodes[0].DynamicAttributes[0].GoAnnotations)
		require.NotNil(t, result.RootNodes[0].DynamicAttributes[0].GoAnnotations.OriginalSourcePath)
		assert.Equal(t, "template.pkc", *result.RootNodes[0].DynamicAttributes[0].GoAnnotations.OriginalSourcePath)
	})

	t.Run("sanitises rich text parts", func(t *testing.T) {
		t.Parallel()

		annotation := &GoGeneratorAnnotation{
			OriginalSourcePath: new("/app/views/page.pkc"),
		}
		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeText,
					RichText: []TextPart{
						{
							IsLiteral:     false,
							Expression:    &Identifier{Name: "name"},
							GoAnnotations: annotation,
						},
					},
				},
			},
		}

		result := SanitiseForEncoding(tree, "/app/views")

		require.NotNil(t, result)
		require.Len(t, result.RootNodes[0].RichText, 1)
		require.NotNil(t, result.RootNodes[0].RichText[0].GoAnnotations)
		require.NotNil(t, result.RootNodes[0].RichText[0].GoAnnotations.OriginalSourcePath)
		assert.Equal(t, "page.pkc", *result.RootNodes[0].RichText[0].GoAnnotations.OriginalSourcePath)
	})

	t.Run("sanitises events", func(t *testing.T) {
		t.Parallel()

		annotation := &GoGeneratorAnnotation{
			OriginalSourcePath: new("/project/components/button.pkc"),
		}
		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "button",
					OnEvents: map[string][]Directive{
						"click": {
							{
								Expression:    &Identifier{Name: "handleClick"},
								GoAnnotations: annotation,
							},
						},
					},
				},
			},
		}

		result := SanitiseForEncoding(tree, "/project/components")

		require.NotNil(t, result)
		require.Contains(t, result.RootNodes[0].OnEvents, "click")
		require.Len(t, result.RootNodes[0].OnEvents["click"], 1)
		require.NotNil(t, result.RootNodes[0].OnEvents["click"][0].GoAnnotations)
		require.NotNil(t, result.RootNodes[0].OnEvents["click"][0].GoAnnotations.OriginalSourcePath)
		assert.Equal(t, "button.pkc", *result.RootNodes[0].OnEvents["click"][0].GoAnnotations.OriginalSourcePath)
	})

	t.Run("sanitises binds", func(t *testing.T) {
		t.Parallel()

		annotation := &GoGeneratorAnnotation{
			OriginalSourcePath: new("/root/templates/input.pkc"),
		}
		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "input",
					Binds: map[string]*Directive{
						"value": {
							Expression:    &Identifier{Name: "inputValue"},
							GoAnnotations: annotation,
						},
					},
				},
			},
		}

		result := SanitiseForEncoding(tree, "/root/templates")

		require.NotNil(t, result)
		require.Contains(t, result.RootNodes[0].Binds, "value")
		require.NotNil(t, result.RootNodes[0].Binds["value"].GoAnnotations)
		require.NotNil(t, result.RootNodes[0].Binds["value"].GoAnnotations.OriginalSourcePath)
		assert.Equal(t, "input.pkc", *result.RootNodes[0].Binds["value"].GoAnnotations.OriginalSourcePath)
	})

	t.Run("sanitises partial info props", func(t *testing.T) {
		t.Parallel()

		propAnn := &GoGeneratorAnnotation{
			OriginalSourcePath: new("/base/partial.pkc"),
		}
		partialInfo := &PartialInvocationInfo{
			PartialAlias: "MyPartial",
			PassedProps: map[string]PropValue{
				"title": {
					Expression:        &StringLiteral{Value: "Hello"},
					InvokerAnnotation: propAnn,
				},
			},
		}
		nodeAnn := &GoGeneratorAnnotation{
			PartialInfo: partialInfo,
		}
		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType:      NodeElement,
					TagName:       "my-partial",
					GoAnnotations: nodeAnn,
				},
			},
		}

		result := SanitiseForEncoding(tree, "/base")

		require.NotNil(t, result)
		require.NotNil(t, result.RootNodes[0].GoAnnotations)
		require.NotNil(t, result.RootNodes[0].GoAnnotations.PartialInfo)
		require.Contains(t, result.RootNodes[0].GoAnnotations.PartialInfo.PassedProps, "title")
	})
}

func TestIsNil(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    any
		name     string
		expected bool
	}{
		{name: "nil interface", input: nil, expected: true},
		{name: "nil pointer", input: (*TemplateNode)(nil), expected: true},
		{name: "nil slice", input: ([]int)(nil), expected: true},
		{name: "nil map", input: (map[string]int)(nil), expected: true},
		{name: "non-nil pointer", input: &TemplateNode{}, expected: false},
		{name: "non-nil slice", input: []int{1, 2}, expected: false},
		{name: "non-nil map", input: map[string]int{"a": 1}, expected: false},
		{name: "string value", input: "test", expected: false},
		{name: "int value", input: 42, expected: false},
		{name: "struct value", input: Location{Line: 1}, expected: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := isNil(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestSanitiseExpression(t *testing.T) {
	t.Parallel()

	basePath := "/project"

	t.Run("sanitises MemberExpr", func(t *testing.T) {
		t.Parallel()

		ann := &GoGeneratorAnnotation{OriginalSourcePath: new("/project/file.pkc")}
		expression := &MemberExpression{
			Base:     &Identifier{Name: "obj", GoAnnotations: ann},
			Property: &Identifier{Name: "prop"},
		}

		sanitiseExpression(expression, basePath)

		identifier, ok := expression.Base.(*Identifier)
		require.True(t, ok)
		require.NotNil(t, identifier.GoAnnotations)
		assert.Equal(t, "file.pkc", *identifier.GoAnnotations.OriginalSourcePath)
	})

	t.Run("sanitises IndexExpr", func(t *testing.T) {
		t.Parallel()

		ann := &GoGeneratorAnnotation{OriginalSourcePath: new("/project/file.pkc")}
		expression := &IndexExpression{
			Base:  &Identifier{Name: "arr", GoAnnotations: ann},
			Index: &DecimalLiteral{Value: "0"},
		}

		sanitiseExpression(expression, basePath)

		identifier, ok := expression.Base.(*Identifier)
		require.True(t, ok)
		require.NotNil(t, identifier.GoAnnotations)
		assert.Equal(t, "file.pkc", *identifier.GoAnnotations.OriginalSourcePath)
	})

	t.Run("sanitises CallExpr", func(t *testing.T) {
		t.Parallel()

		ann := &GoGeneratorAnnotation{OriginalSourcePath: new("/project/file.pkc")}
		expression := &CallExpression{
			Callee: &Identifier{Name: "fn", GoAnnotations: ann},
			Args:   []Expression{&StringLiteral{Value: "argument"}},
		}

		sanitiseExpression(expression, basePath)

		identifier, ok := expression.Callee.(*Identifier)
		require.True(t, ok)
		require.NotNil(t, identifier.GoAnnotations)
		assert.Equal(t, "file.pkc", *identifier.GoAnnotations.OriginalSourcePath)
	})

	t.Run("sanitises BinaryExpr", func(t *testing.T) {
		t.Parallel()

		ann := &GoGeneratorAnnotation{OriginalSourcePath: new("/project/file.pkc")}
		expression := &BinaryExpression{
			Left:     &Identifier{Name: "a", GoAnnotations: ann},
			Operator: "+",
			Right:    &DecimalLiteral{Value: "1"},
		}

		sanitiseExpression(expression, basePath)

		identifier, ok := expression.Left.(*Identifier)
		require.True(t, ok)
		require.NotNil(t, identifier.GoAnnotations)
		assert.Equal(t, "file.pkc", *identifier.GoAnnotations.OriginalSourcePath)
	})

	t.Run("sanitises UnaryExpr", func(t *testing.T) {
		t.Parallel()

		ann := &GoGeneratorAnnotation{OriginalSourcePath: new("/project/file.pkc")}
		expression := &UnaryExpression{
			Operator: "!",
			Right:    &Identifier{Name: "flag", GoAnnotations: ann},
		}

		sanitiseExpression(expression, basePath)

		identifier, ok := expression.Right.(*Identifier)
		require.True(t, ok)
		require.NotNil(t, identifier.GoAnnotations)
		assert.Equal(t, "file.pkc", *identifier.GoAnnotations.OriginalSourcePath)
	})

	t.Run("sanitises TernaryExpr", func(t *testing.T) {
		t.Parallel()

		ann := &GoGeneratorAnnotation{OriginalSourcePath: new("/project/file.pkc")}
		expression := &TernaryExpression{
			Condition:  &Identifier{Name: "cond", GoAnnotations: ann},
			Consequent: &StringLiteral{Value: "yes"},
			Alternate:  &StringLiteral{Value: "no"},
		}

		sanitiseExpression(expression, basePath)

		identifier, ok := expression.Condition.(*Identifier)
		require.True(t, ok)
		require.NotNil(t, identifier.GoAnnotations)
		assert.Equal(t, "file.pkc", *identifier.GoAnnotations.OriginalSourcePath)
	})

	t.Run("sanitises ForInExpr", func(t *testing.T) {
		t.Parallel()

		ann := &GoGeneratorAnnotation{OriginalSourcePath: new("/project/file.pkc")}
		expression := &ForInExpression{
			ItemVariable: &Identifier{Name: "item"},
			Collection:   &Identifier{Name: "items", GoAnnotations: ann},
		}

		sanitiseExpression(expression, basePath)

		identifier, ok := expression.Collection.(*Identifier)
		require.True(t, ok)
		require.NotNil(t, identifier.GoAnnotations)
		assert.Equal(t, "file.pkc", *identifier.GoAnnotations.OriginalSourcePath)
	})

	t.Run("sanitises ObjectLiteral", func(t *testing.T) {
		t.Parallel()

		ann := &GoGeneratorAnnotation{OriginalSourcePath: new("/project/file.pkc")}
		expression := &ObjectLiteral{
			Pairs: map[string]Expression{
				"key": &Identifier{Name: "val", GoAnnotations: ann},
			},
		}

		sanitiseExpression(expression, basePath)

		identifier, ok := expression.Pairs["key"].(*Identifier)
		require.True(t, ok)
		require.NotNil(t, identifier.GoAnnotations)
		assert.Equal(t, "file.pkc", *identifier.GoAnnotations.OriginalSourcePath)
	})

	t.Run("sanitises ArrayLiteral", func(t *testing.T) {
		t.Parallel()

		ann := &GoGeneratorAnnotation{OriginalSourcePath: new("/project/file.pkc")}
		expression := &ArrayLiteral{
			Elements: []Expression{
				&Identifier{Name: "item", GoAnnotations: ann},
			},
		}

		sanitiseExpression(expression, basePath)

		identifier, ok := expression.Elements[0].(*Identifier)
		require.True(t, ok)
		require.NotNil(t, identifier.GoAnnotations)
		assert.Equal(t, "file.pkc", *identifier.GoAnnotations.OriginalSourcePath)
	})

	t.Run("sanitises TemplateLiteral", func(t *testing.T) {
		t.Parallel()

		ann := &GoGeneratorAnnotation{OriginalSourcePath: new("/project/file.pkc")}
		expression := &TemplateLiteral{
			Parts: []TemplateLiteralPart{
				{
					IsLiteral:  false,
					Expression: &Identifier{Name: "name", GoAnnotations: ann},
				},
			},
		}

		sanitiseExpression(expression, basePath)

		identifier, ok := expression.Parts[0].Expression.(*Identifier)
		require.True(t, ok)
		require.NotNil(t, identifier.GoAnnotations)
		assert.Equal(t, "file.pkc", *identifier.GoAnnotations.OriginalSourcePath)
	})
}

func TestSanitiseItem(t *testing.T) {
	t.Parallel()

	t.Run("nil item does not panic", func(t *testing.T) {
		t.Parallel()

		assert.NotPanics(t, func() {
			sanitiseItem(nil, "/base")
		})
	})

	t.Run("typed nil pointer does not panic", func(t *testing.T) {
		t.Parallel()

		var node *TemplateNode
		assert.NotPanics(t, func() {
			sanitiseItem(node, "/base")
		})
	})
}
