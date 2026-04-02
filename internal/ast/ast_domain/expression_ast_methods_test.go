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

func TestExpressionGoAnnotationMethods(t *testing.T) {
	t.Parallel()

	testAnnotation := &GoGeneratorAnnotation{
		IsStatic:      true,
		Stringability: 2,
		Symbol:        &ResolvedSymbol{Name: "testSymbol"},
	}

	testCases := []struct {
		expression Expression
		name       string
	}{
		{
			name: "Identifier",
			expression: &Identifier{
				Name:             "myVar",
				RelativeLocation: Location{Line: 1, Column: 1},
			},
		},
		{
			name: "MemberExpr",
			expression: &MemberExpression{
				Base:     &Identifier{Name: "obj"},
				Property: &Identifier{Name: "prop"},
			},
		},
		{
			name: "IndexExpr",
			expression: &IndexExpression{
				Base:  &Identifier{Name: "arr"},
				Index: &IntegerLiteral{Value: 0},
			},
		},
		{
			name: "UnaryExpr",
			expression: &UnaryExpression{
				Operator: OpNot,
				Right:    &BooleanLiteral{Value: true},
			},
		},
		{
			name: "BinaryExpr",
			expression: &BinaryExpression{
				Left:     &IntegerLiteral{Value: 1},
				Operator: OpPlus,
				Right:    &IntegerLiteral{Value: 2},
			},
		},
		{
			name: "ForInExpr",
			expression: &ForInExpression{
				ItemVariable: &Identifier{Name: "item"},
				Collection:   &Identifier{Name: "items"},
			},
		},
		{
			name: "CallExpr",
			expression: &CallExpression{
				Callee: &Identifier{Name: "fn"},
				Args:   []Expression{&IntegerLiteral{Value: 1}},
			},
		},
		{
			name: "TemplateLiteral",
			expression: &TemplateLiteral{
				Parts: []TemplateLiteralPart{
					{IsLiteral: true, Literal: "Hello "},
					{IsLiteral: false, Expression: &Identifier{Name: "name"}},
				},
			},
		},
		{
			name: "StringLiteral",
			expression: &StringLiteral{
				Value: "hello",
			},
		},
		{
			name: "IntegerLiteral",
			expression: &IntegerLiteral{
				Value: 42,
			},
		},
		{
			name: "FloatLiteral",
			expression: &FloatLiteral{
				Value: 3.14,
			},
		},
		{
			name: "DecimalLiteral",
			expression: &DecimalLiteral{
				Value: "123.456",
			},
		},
		{
			name: "BigIntLiteral",
			expression: &BigIntLiteral{
				Value: "12345678901234567890",
			},
		},
		{
			name: "DateTimeLiteral",
			expression: &DateTimeLiteral{
				Value: "2024-01-15T10:30:00Z",
			},
		},
		{
			name: "DurationLiteral",
			expression: &DurationLiteral{
				Value: "1h30m",
			},
		},
		{
			name: "DateLiteral",
			expression: &DateLiteral{
				Value: "2024-01-15",
			},
		},
		{
			name: "RuneLiteral",
			expression: &RuneLiteral{
				Value: 'A',
			},
		},
		{
			name: "TimeLiteral",
			expression: &TimeLiteral{
				Value: "10:30:00",
			},
		},
		{
			name: "BooleanLiteral",
			expression: &BooleanLiteral{
				Value: true,
			},
		},
		{
			name:       "NilLiteral",
			expression: &NilLiteral{},
		},
		{
			name: "ObjectLiteral",
			expression: &ObjectLiteral{
				Pairs: map[string]Expression{
					"key": &StringLiteral{Value: "value"},
				},
			},
		},
		{
			name: "TernaryExpr",
			expression: &TernaryExpression{
				Condition:  &BooleanLiteral{Value: true},
				Consequent: &StringLiteral{Value: "yes"},
				Alternate:  &StringLiteral{Value: "no"},
			},
		},
		{
			name: "ArrayLiteral",
			expression: &ArrayLiteral{
				Elements: []Expression{&IntegerLiteral{Value: 1}},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			t.Run("initially nil annotation", func(t *testing.T) {
				ann := tc.expression.GetGoAnnotation()
				assert.Nil(t, ann)
			})

			t.Run("set and get annotation", func(t *testing.T) {
				tc.expression.SetGoAnnotation(testAnnotation)
				ann := tc.expression.GetGoAnnotation()

				require.NotNil(t, ann)
				assert.True(t, ann.IsStatic)
				assert.Equal(t, 2, ann.Stringability)
				require.NotNil(t, ann.Symbol)
				assert.Equal(t, "testSymbol", ann.Symbol.Name)
			})

			t.Run("set nil annotation", func(t *testing.T) {
				tc.expression.SetGoAnnotation(nil)
				ann := tc.expression.GetGoAnnotation()
				assert.Nil(t, ann)
			})
		})
	}
}

func TestTransformIdentifiers(t *testing.T) {
	t.Parallel()

	toUpper := func(s string) string {
		return "TRANSFORMED_" + s
	}

	t.Run("Identifier", func(t *testing.T) {
		t.Parallel()

		original := &Identifier{
			Name:             "myVar",
			RelativeLocation: Location{Line: 1, Column: 5},
			SourceLength:     5,
			GoAnnotations:    &GoGeneratorAnnotation{IsStatic: true},
		}

		result := original.TransformIdentifiers(toUpper)
		transformed, ok := result.(*Identifier)
		require.True(t, ok)

		assert.Equal(t, "TRANSFORMED_myVar", transformed.Name)
		assert.Equal(t, original.RelativeLocation, transformed.RelativeLocation)
		assert.Equal(t, original.SourceLength, transformed.SourceLength)
		assert.Same(t, original.GoAnnotations, transformed.GoAnnotations)
	})

	t.Run("MemberExpr", func(t *testing.T) {
		t.Parallel()

		original := &MemberExpression{
			Base:             &Identifier{Name: "obj"},
			Property:         &Identifier{Name: "prop"},
			Optional:         true,
			Computed:         false,
			RelativeLocation: Location{Line: 1, Column: 1},
			SourceLength:     10,
			GoAnnotations:    &GoGeneratorAnnotation{IsStatic: true},
		}

		result := original.TransformIdentifiers(toUpper)
		transformed, ok := result.(*MemberExpression)
		require.True(t, ok)

		base, ok := transformed.Base.(*Identifier)
		require.True(t, ok)
		assert.Equal(t, "TRANSFORMED_obj", base.Name)
		prop, ok := transformed.Property.(*Identifier)
		require.True(t, ok)
		assert.Equal(t, "TRANSFORMED_prop", prop.Name)
		assert.True(t, transformed.Optional)
		assert.False(t, transformed.Computed)
		assert.Same(t, original.GoAnnotations, transformed.GoAnnotations)
	})

	t.Run("IndexExpr", func(t *testing.T) {
		t.Parallel()

		original := &IndexExpression{
			Base:             &Identifier{Name: "arr"},
			Index:            &Identifier{Name: "index"},
			Optional:         true,
			RelativeLocation: Location{Line: 1, Column: 1},
			GoAnnotations:    &GoGeneratorAnnotation{},
		}

		result := original.TransformIdentifiers(toUpper)
		transformed, ok := result.(*IndexExpression)
		require.True(t, ok)

		transformedBase, ok := transformed.Base.(*Identifier)
		require.True(t, ok)
		assert.Equal(t, "TRANSFORMED_arr", transformedBase.Name)
		transformedIndex, ok := transformed.Index.(*Identifier)
		require.True(t, ok)
		assert.Equal(t, "TRANSFORMED_index", transformedIndex.Name)
		assert.True(t, transformed.Optional)
	})

	t.Run("UnaryExpr", func(t *testing.T) {
		t.Parallel()

		original := &UnaryExpression{
			Operator:         OpNot,
			Right:            &Identifier{Name: "flag"},
			RelativeLocation: Location{Line: 1, Column: 1},
			GoAnnotations:    &GoGeneratorAnnotation{},
		}

		result := original.TransformIdentifiers(toUpper)
		transformed, ok := result.(*UnaryExpression)
		require.True(t, ok)

		assert.Equal(t, OpNot, transformed.Operator)
		transformedRight, ok := transformed.Right.(*Identifier)
		require.True(t, ok)
		assert.Equal(t, "TRANSFORMED_flag", transformedRight.Name)
	})

	t.Run("BinaryExpr", func(t *testing.T) {
		t.Parallel()

		original := &BinaryExpression{
			Left:             &Identifier{Name: "a"},
			Operator:         OpPlus,
			Right:            &Identifier{Name: "b"},
			RelativeLocation: Location{Line: 1, Column: 1},
			GoAnnotations:    &GoGeneratorAnnotation{},
		}

		result := original.TransformIdentifiers(toUpper)
		transformed, ok := result.(*BinaryExpression)
		require.True(t, ok)

		transformedLeft, ok := transformed.Left.(*Identifier)
		require.True(t, ok)
		assert.Equal(t, "TRANSFORMED_a", transformedLeft.Name)
		assert.Equal(t, OpPlus, transformed.Operator)
		transformedRight, ok := transformed.Right.(*Identifier)
		require.True(t, ok)
		assert.Equal(t, "TRANSFORMED_b", transformedRight.Name)
	})

	t.Run("ForInExpr transforms collection but not loop variables", func(t *testing.T) {
		t.Parallel()

		original := &ForInExpression{
			IndexVariable:    &Identifier{Name: "index"},
			ItemVariable:     &Identifier{Name: "item"},
			Collection:       &Identifier{Name: "items"},
			RelativeLocation: Location{Line: 1, Column: 1},
			GoAnnotations:    &GoGeneratorAnnotation{},
		}

		result := original.TransformIdentifiers(toUpper)
		transformed, ok := result.(*ForInExpression)
		require.True(t, ok)

		assert.Equal(t, "index", transformed.IndexVariable.Name)
		assert.Equal(t, "item", transformed.ItemVariable.Name)

		transformedCollection, ok := transformed.Collection.(*Identifier)
		require.True(t, ok)
		assert.Equal(t, "TRANSFORMED_items", transformedCollection.Name)
	})

	t.Run("CallExpr", func(t *testing.T) {
		t.Parallel()

		original := &CallExpression{
			Callee: &Identifier{Name: "myFunc"},
			Args: []Expression{
				&Identifier{Name: "arg1"},
				&Identifier{Name: "arg2"},
			},
			RelativeLocation: Location{Line: 1, Column: 1},
			GoAnnotations:    &GoGeneratorAnnotation{},
		}

		result := original.TransformIdentifiers(toUpper)
		transformed, ok := result.(*CallExpression)
		require.True(t, ok)

		transformedCallee, ok := transformed.Callee.(*Identifier)
		require.True(t, ok)
		assert.Equal(t, "TRANSFORMED_myFunc", transformedCallee.Name)
		assert.Len(t, transformed.Args, 2)
		transformedArg0, ok := transformed.Args[0].(*Identifier)
		require.True(t, ok)
		assert.Equal(t, "TRANSFORMED_arg1", transformedArg0.Name)
		transformedArg1, ok := transformed.Args[1].(*Identifier)
		require.True(t, ok)
		assert.Equal(t, "TRANSFORMED_arg2", transformedArg1.Name)
	})

	t.Run("TemplateLiteral", func(t *testing.T) {
		t.Parallel()

		original := &TemplateLiteral{
			Parts: []TemplateLiteralPart{
				{IsLiteral: true, Literal: "Hello "},
				{IsLiteral: false, Expression: &Identifier{Name: "name"}},
				{IsLiteral: true, Literal: "!"},
			},
			RelativeLocation: Location{Line: 1, Column: 1},
			GoAnnotations:    &GoGeneratorAnnotation{},
		}

		result := original.TransformIdentifiers(toUpper)
		transformed, ok := result.(*TemplateLiteral)
		require.True(t, ok)

		assert.Len(t, transformed.Parts, 3)

		assert.True(t, transformed.Parts[0].IsLiteral)
		assert.Equal(t, "Hello ", transformed.Parts[0].Literal)

		assert.False(t, transformed.Parts[1].IsLiteral)
		transformedPartExpr, ok := transformed.Parts[1].Expression.(*Identifier)
		require.True(t, ok)
		assert.Equal(t, "TRANSFORMED_name", transformedPartExpr.Name)

		assert.True(t, transformed.Parts[2].IsLiteral)
		assert.Equal(t, "!", transformed.Parts[2].Literal)
	})

	t.Run("literals return copy with same values", func(t *testing.T) {
		t.Parallel()

		literalCases := []struct {
			original   Expression
			checkValue func(t *testing.T, result Expression)
			name       string
		}{
			{
				name:     "StringLiteral",
				original: &StringLiteral{Value: "hello", GoAnnotations: &GoGeneratorAnnotation{IsStatic: true}},
				checkValue: func(t *testing.T, result Expression) {
					assert.Equal(t, "hello", result.(*StringLiteral).Value)
					assert.NotNil(t, result.(*StringLiteral).GoAnnotations)
				},
			},
			{
				name:     "IntegerLiteral",
				original: &IntegerLiteral{Value: 42, GoAnnotations: &GoGeneratorAnnotation{}},
				checkValue: func(t *testing.T, result Expression) {
					assert.Equal(t, int64(42), result.(*IntegerLiteral).Value)
				},
			},
			{
				name:     "FloatLiteral",
				original: &FloatLiteral{Value: 3.14, GoAnnotations: &GoGeneratorAnnotation{}},
				checkValue: func(t *testing.T, result Expression) {
					assert.Equal(t, 3.14, result.(*FloatLiteral).Value)
				},
			},
			{
				name:     "DecimalLiteral",
				original: &DecimalLiteral{Value: "123.456", GoAnnotations: &GoGeneratorAnnotation{}},
				checkValue: func(t *testing.T, result Expression) {
					assert.Equal(t, "123.456", result.(*DecimalLiteral).Value)
				},
			},
			{
				name:     "BigIntLiteral",
				original: &BigIntLiteral{Value: "12345678901234567890", GoAnnotations: &GoGeneratorAnnotation{}},
				checkValue: func(t *testing.T, result Expression) {
					assert.Equal(t, "12345678901234567890", result.(*BigIntLiteral).Value)
				},
			},
			{
				name:     "DateTimeLiteral",
				original: &DateTimeLiteral{Value: "2024-01-15T10:30:00Z", GoAnnotations: &GoGeneratorAnnotation{}},
				checkValue: func(t *testing.T, result Expression) {
					assert.Equal(t, "2024-01-15T10:30:00Z", result.(*DateTimeLiteral).Value)
				},
			},
			{
				name:     "DurationLiteral",
				original: &DurationLiteral{Value: "1h30m", GoAnnotations: &GoGeneratorAnnotation{}},
				checkValue: func(t *testing.T, result Expression) {
					assert.Equal(t, "1h30m", result.(*DurationLiteral).Value)
				},
			},
			{
				name:     "DateLiteral",
				original: &DateLiteral{Value: "2024-01-15", GoAnnotations: &GoGeneratorAnnotation{}},
				checkValue: func(t *testing.T, result Expression) {
					assert.Equal(t, "2024-01-15", result.(*DateLiteral).Value)
				},
			},
			{
				name:     "RuneLiteral",
				original: &RuneLiteral{Value: 'A', GoAnnotations: &GoGeneratorAnnotation{}},
				checkValue: func(t *testing.T, result Expression) {
					assert.Equal(t, 'A', result.(*RuneLiteral).Value)
				},
			},
			{
				name:     "TimeLiteral",
				original: &TimeLiteral{Value: "10:30:00", GoAnnotations: &GoGeneratorAnnotation{}},
				checkValue: func(t *testing.T, result Expression) {
					assert.Equal(t, "10:30:00", result.(*TimeLiteral).Value)
				},
			},
			{
				name:     "BooleanLiteral",
				original: &BooleanLiteral{Value: true, GoAnnotations: &GoGeneratorAnnotation{}},
				checkValue: func(t *testing.T, result Expression) {
					assert.True(t, result.(*BooleanLiteral).Value)
				},
			},
			{
				name:     "NilLiteral",
				original: &NilLiteral{GoAnnotations: &GoGeneratorAnnotation{}},
				checkValue: func(t *testing.T, result Expression) {
					_, ok := result.(*NilLiteral)
					assert.True(t, ok)
				},
			},
		}

		for _, tc := range literalCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				result := tc.original.TransformIdentifiers(toUpper)
				tc.checkValue(t, result)

				assert.NotSame(t, tc.original, result)
			})
		}
	})

	t.Run("TernaryExpr", func(t *testing.T) {
		t.Parallel()

		original := &TernaryExpression{
			Condition:        &Identifier{Name: "cond"},
			Consequent:       &Identifier{Name: "then"},
			Alternate:        &Identifier{Name: "else"},
			RelativeLocation: Location{Line: 1, Column: 1},
			GoAnnotations:    &GoGeneratorAnnotation{},
		}

		result := original.TransformIdentifiers(toUpper)
		transformed, ok := result.(*TernaryExpression)
		require.True(t, ok)

		transformedCond, ok := transformed.Condition.(*Identifier)
		require.True(t, ok)
		assert.Equal(t, "TRANSFORMED_cond", transformedCond.Name)
		transformedThen, ok := transformed.Consequent.(*Identifier)
		require.True(t, ok)
		assert.Equal(t, "TRANSFORMED_then", transformedThen.Name)
		transformedElse, ok := transformed.Alternate.(*Identifier)
		require.True(t, ok)
		assert.Equal(t, "TRANSFORMED_else", transformedElse.Name)
	})

	t.Run("ArrayLiteral", func(t *testing.T) {
		t.Parallel()

		original := &ArrayLiteral{
			Elements: []Expression{
				&Identifier{Name: "a"},
				&Identifier{Name: "b"},
			},
			RelativeLocation: Location{Line: 1, Column: 1},
			GoAnnotations:    &GoGeneratorAnnotation{},
		}

		result := original.TransformIdentifiers(toUpper)
		transformed, ok := result.(*ArrayLiteral)
		require.True(t, ok)

		assert.Len(t, transformed.Elements, 2)
		transformedEl0, ok := transformed.Elements[0].(*Identifier)
		require.True(t, ok)
		assert.Equal(t, "TRANSFORMED_a", transformedEl0.Name)
		transformedEl1, ok := transformed.Elements[1].(*Identifier)
		require.True(t, ok)
		assert.Equal(t, "TRANSFORMED_b", transformedEl1.Name)
	})

	t.Run("ObjectLiteral", func(t *testing.T) {
		t.Parallel()

		original := &ObjectLiteral{
			Pairs: map[string]Expression{
				"key1": &Identifier{Name: "val1"},
				"key2": &Identifier{Name: "val2"},
			},
			RelativeLocation: Location{Line: 1, Column: 1},
			GoAnnotations:    &GoGeneratorAnnotation{},
		}

		result := original.TransformIdentifiers(toUpper)
		transformed, ok := result.(*ObjectLiteral)
		require.True(t, ok)

		assert.Contains(t, transformed.Pairs, "key1")
		assert.Contains(t, transformed.Pairs, "key2")

		transformedVal1, ok := transformed.Pairs["key1"].(*Identifier)
		require.True(t, ok)
		assert.Equal(t, "TRANSFORMED_val1", transformedVal1.Name)
		transformedVal2, ok := transformed.Pairs["key2"].(*Identifier)
		require.True(t, ok)
		assert.Equal(t, "TRANSFORMED_val2", transformedVal2.Name)
	})

	t.Run("deeply nested expression", func(t *testing.T) {
		t.Parallel()

		original := &CallExpression{
			Callee: &IndexExpression{
				Base: &MemberExpression{
					Base:     &Identifier{Name: "foo"},
					Property: &Identifier{Name: "bar"},
				},
				Index: &Identifier{Name: "baz"},
			},
			Args: []Expression{&Identifier{Name: "qux"}},
		}

		result := original.TransformIdentifiers(toUpper)
		transformed, ok := result.(*CallExpression)
		require.True(t, ok)

		indexExpr, ok := transformed.Callee.(*IndexExpression)
		require.True(t, ok)
		memberExpr, ok := indexExpr.Base.(*MemberExpression)
		require.True(t, ok)

		memberBase, ok := memberExpr.Base.(*Identifier)
		require.True(t, ok)
		assert.Equal(t, "TRANSFORMED_foo", memberBase.Name)
		memberProp, ok := memberExpr.Property.(*Identifier)
		require.True(t, ok)
		assert.Equal(t, "TRANSFORMED_bar", memberProp.Name)
		indexExprIndex, ok := indexExpr.Index.(*Identifier)
		require.True(t, ok)
		assert.Equal(t, "TRANSFORMED_baz", indexExprIndex.Name)
		transformedArg, ok := transformed.Args[0].(*Identifier)
		require.True(t, ok)
		assert.Equal(t, "TRANSFORMED_qux", transformedArg.Name)
	})
}

func TestGetSourceLength(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		expression   Expression
		name         string
		sourceLength int
	}{
		{
			name:         "Identifier",
			expression:   &Identifier{Name: "myVar", SourceLength: 5},
			sourceLength: 5,
		},
		{
			name:         "MemberExpr",
			expression:   &MemberExpression{Base: &Identifier{Name: "a"}, Property: &Identifier{Name: "b"}, SourceLength: 3},
			sourceLength: 3,
		},
		{
			name:         "IndexExpr",
			expression:   &IndexExpression{Base: &Identifier{Name: "arr"}, Index: &IntegerLiteral{Value: 0}, SourceLength: 6},
			sourceLength: 6,
		},
		{
			name:         "UnaryExpr",
			expression:   &UnaryExpression{Operator: OpNot, Right: &Identifier{Name: "x"}, SourceLength: 2},
			sourceLength: 2,
		},
		{
			name:         "BinaryExpr",
			expression:   &BinaryExpression{Left: &IntegerLiteral{Value: 1}, Operator: OpPlus, Right: &IntegerLiteral{Value: 2}, SourceLength: 5},
			sourceLength: 5,
		},
		{
			name:         "ForInExpr",
			expression:   &ForInExpression{ItemVariable: &Identifier{Name: "x"}, Collection: &Identifier{Name: "xs"}, SourceLength: 8},
			sourceLength: 8,
		},
		{
			name:         "CallExpr",
			expression:   &CallExpression{Callee: &Identifier{Name: "fn"}, SourceLength: 4},
			sourceLength: 4,
		},
		{
			name:         "TemplateLiteral",
			expression:   &TemplateLiteral{Parts: []TemplateLiteralPart{{IsLiteral: true, Literal: "hi"}}, SourceLength: 4},
			sourceLength: 4,
		},
		{
			name:         "StringLiteral",
			expression:   &StringLiteral{Value: "test", SourceLength: 6},
			sourceLength: 6,
		},
		{
			name:         "IntegerLiteral",
			expression:   &IntegerLiteral{Value: 123, SourceLength: 3},
			sourceLength: 3,
		},
		{
			name:         "FloatLiteral",
			expression:   &FloatLiteral{Value: 3.14, SourceLength: 4},
			sourceLength: 4,
		},
		{
			name:         "DecimalLiteral",
			expression:   &DecimalLiteral{Value: "123.456", SourceLength: 8},
			sourceLength: 8,
		},
		{
			name:         "BigIntLiteral",
			expression:   &BigIntLiteral{Value: "123", SourceLength: 4},
			sourceLength: 4,
		},
		{
			name:         "DateTimeLiteral",
			expression:   &DateTimeLiteral{Value: "2024-01-15T10:30:00Z", SourceLength: 24},
			sourceLength: 24,
		},
		{
			name:         "DurationLiteral",
			expression:   &DurationLiteral{Value: "1h30m", SourceLength: 10},
			sourceLength: 10,
		},
		{
			name:         "DateLiteral",
			expression:   &DateLiteral{Value: "2024-01-15", SourceLength: 14},
			sourceLength: 14,
		},
		{
			name:         "RuneLiteral",
			expression:   &RuneLiteral{Value: 'A', SourceLength: 4},
			sourceLength: 4,
		},
		{
			name:         "TimeLiteral",
			expression:   &TimeLiteral{Value: "10:30:00", SourceLength: 12},
			sourceLength: 12,
		},
		{
			name:         "BooleanLiteral",
			expression:   &BooleanLiteral{Value: true, SourceLength: 4},
			sourceLength: 4,
		},
		{
			name:         "NilLiteral",
			expression:   &NilLiteral{SourceLength: 3},
			sourceLength: 3,
		},
		{
			name:         "ObjectLiteral",
			expression:   &ObjectLiteral{Pairs: map[string]Expression{}, SourceLength: 2},
			sourceLength: 2,
		},
		{
			name:         "TernaryExpr",
			expression:   &TernaryExpression{Condition: &BooleanLiteral{Value: true}, Consequent: &IntegerLiteral{Value: 1}, Alternate: &IntegerLiteral{Value: 2}, SourceLength: 15},
			sourceLength: 15,
		},
		{
			name:         "ArrayLiteral",
			expression:   &ArrayLiteral{Elements: []Expression{}, SourceLength: 2},
			sourceLength: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.sourceLength, tc.expression.GetSourceLength())
		})
	}
}

func TestGetRelativeLocation(t *testing.T) {
	t.Parallel()

	expectedLocation := Location{Line: 10, Column: 5, Offset: 100}

	testCases := []struct {
		expression Expression
		name       string
	}{
		{
			name:       "Identifier",
			expression: &Identifier{Name: "x", RelativeLocation: expectedLocation},
		},
		{
			name:       "MemberExpr",
			expression: &MemberExpression{Base: &Identifier{Name: "a"}, Property: &Identifier{Name: "b"}, RelativeLocation: expectedLocation},
		},
		{
			name:       "IndexExpr",
			expression: &IndexExpression{Base: &Identifier{Name: "a"}, Index: &IntegerLiteral{Value: 0}, RelativeLocation: expectedLocation},
		},
		{
			name:       "UnaryExpr",
			expression: &UnaryExpression{Operator: OpNot, Right: &Identifier{Name: "x"}, RelativeLocation: expectedLocation},
		},
		{
			name:       "BinaryExpr",
			expression: &BinaryExpression{Left: &IntegerLiteral{Value: 1}, Operator: OpPlus, Right: &IntegerLiteral{Value: 2}, RelativeLocation: expectedLocation},
		},
		{
			name:       "ForInExpr",
			expression: &ForInExpression{ItemVariable: &Identifier{Name: "x"}, Collection: &Identifier{Name: "xs"}, RelativeLocation: expectedLocation},
		},
		{
			name:       "CallExpr",
			expression: &CallExpression{Callee: &Identifier{Name: "fn"}, RelativeLocation: expectedLocation},
		},
		{
			name:       "TemplateLiteral",
			expression: &TemplateLiteral{Parts: []TemplateLiteralPart{}, RelativeLocation: expectedLocation},
		},
		{
			name:       "StringLiteral",
			expression: &StringLiteral{Value: "test", RelativeLocation: expectedLocation},
		},
		{
			name:       "IntegerLiteral",
			expression: &IntegerLiteral{Value: 42, RelativeLocation: expectedLocation},
		},
		{
			name:       "FloatLiteral",
			expression: &FloatLiteral{Value: 3.14, RelativeLocation: expectedLocation},
		},
		{
			name:       "DecimalLiteral",
			expression: &DecimalLiteral{Value: "123.456", RelativeLocation: expectedLocation},
		},
		{
			name:       "BigIntLiteral",
			expression: &BigIntLiteral{Value: "123", RelativeLocation: expectedLocation},
		},
		{
			name:       "DateTimeLiteral",
			expression: &DateTimeLiteral{Value: "2024-01-15T10:30:00Z", RelativeLocation: expectedLocation},
		},
		{
			name:       "DurationLiteral",
			expression: &DurationLiteral{Value: "1h30m", RelativeLocation: expectedLocation},
		},
		{
			name:       "DateLiteral",
			expression: &DateLiteral{Value: "2024-01-15", RelativeLocation: expectedLocation},
		},
		{
			name:       "RuneLiteral",
			expression: &RuneLiteral{Value: 'A', RelativeLocation: expectedLocation},
		},
		{
			name:       "TimeLiteral",
			expression: &TimeLiteral{Value: "10:30:00", RelativeLocation: expectedLocation},
		},
		{
			name:       "BooleanLiteral",
			expression: &BooleanLiteral{Value: true, RelativeLocation: expectedLocation},
		},
		{
			name:       "NilLiteral",
			expression: &NilLiteral{RelativeLocation: expectedLocation},
		},
		{
			name:       "ObjectLiteral",
			expression: &ObjectLiteral{Pairs: map[string]Expression{}, RelativeLocation: expectedLocation},
		},
		{
			name:       "TernaryExpr",
			expression: &TernaryExpression{Condition: &BooleanLiteral{Value: true}, Consequent: &IntegerLiteral{Value: 1}, Alternate: &IntegerLiteral{Value: 2}, RelativeLocation: expectedLocation},
		},
		{
			name:       "ArrayLiteral",
			expression: &ArrayLiteral{Elements: []Expression{}, RelativeLocation: expectedLocation},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			loc := tc.expression.GetRelativeLocation()
			assert.Equal(t, expectedLocation.Line, loc.Line)
			assert.Equal(t, expectedLocation.Column, loc.Column)
			assert.Equal(t, expectedLocation.Offset, loc.Offset)
		})
	}
}

func TestTemplateLiteralPartClone(t *testing.T) {
	t.Parallel()

	t.Run("literal part", func(t *testing.T) {
		t.Parallel()

		original := TemplateLiteralPart{
			IsLiteral:        true,
			Literal:          "Hello World",
			Expression:       nil,
			RelativeLocation: Location{Line: 1, Column: 5},
		}

		clone := original.Clone()

		assert.True(t, clone.IsLiteral)
		assert.Equal(t, "Hello World", clone.Literal)
		assert.Nil(t, clone.Expression)
		assert.Equal(t, Location{Line: 1, Column: 5}, clone.RelativeLocation)
	})

	t.Run("expression part", func(t *testing.T) {
		t.Parallel()

		original := TemplateLiteralPart{
			IsLiteral:        false,
			Literal:          "",
			Expression:       &Identifier{Name: "name"},
			RelativeLocation: Location{Line: 1, Column: 10},
		}

		clone := original.Clone()

		assert.False(t, clone.IsLiteral)
		require.NotNil(t, clone.Expression)
		assert.Equal(t, "name", clone.Expression.(*Identifier).Name)
		assert.NotSame(t, original.Expression, clone.Expression)
	})

	t.Run("expression part with nil expression", func(t *testing.T) {
		t.Parallel()

		original := TemplateLiteralPart{
			IsLiteral:  false,
			Expression: nil,
		}

		clone := original.Clone()

		assert.False(t, clone.IsLiteral)
		assert.Nil(t, clone.Expression)
	})
}

func TestTemplateLiteralPartGetRelativeLocation(t *testing.T) {
	t.Parallel()

	part := TemplateLiteralPart{
		IsLiteral:        true,
		Literal:          "test",
		RelativeLocation: Location{Line: 5, Column: 10, Offset: 50},
	}

	loc := part.GetRelativeLocation()

	assert.Equal(t, 5, loc.Line)
	assert.Equal(t, 10, loc.Column)
	assert.Equal(t, 50, loc.Offset)
}

func TestTemplateLiteralClone(t *testing.T) {
	t.Parallel()

	t.Run("nil returns nil", func(t *testing.T) {
		t.Parallel()

		var tl *TemplateLiteral
		result := tl.Clone()
		assert.Nil(t, result)
	})

	t.Run("clones all parts", func(t *testing.T) {
		t.Parallel()

		original := &TemplateLiteral{
			Parts: []TemplateLiteralPart{
				{IsLiteral: true, Literal: "Hello "},
				{IsLiteral: false, Expression: &Identifier{Name: "name"}},
				{IsLiteral: true, Literal: "!"},
			},
			RelativeLocation: Location{Line: 1, Column: 1},
			SourceLength:     20,
			GoAnnotations:    &GoGeneratorAnnotation{IsStatic: true},
		}

		result := original.Clone()
		clone, ok := result.(*TemplateLiteral)
		require.True(t, ok)

		assert.Len(t, clone.Parts, 3)
		assert.True(t, clone.Parts[0].IsLiteral)
		assert.Equal(t, "Hello ", clone.Parts[0].Literal)
		assert.False(t, clone.Parts[1].IsLiteral)
		cloneExpr, ok := clone.Parts[1].Expression.(*Identifier)
		require.True(t, ok)
		assert.Equal(t, "name", cloneExpr.Name)
		assert.True(t, clone.Parts[2].IsLiteral)
		assert.Equal(t, "!", clone.Parts[2].Literal)

		assert.NotSame(t, original.Parts[1].Expression, clone.Parts[1].Expression)
		assert.NotSame(t, original.GoAnnotations, clone.GoAnnotations)
	})

	t.Run("empty parts", func(t *testing.T) {
		t.Parallel()

		original := &TemplateLiteral{
			Parts:            []TemplateLiteralPart{},
			RelativeLocation: Location{Line: 1, Column: 1},
		}

		result := original.Clone()
		clone, ok := result.(*TemplateLiteral)
		require.True(t, ok)

		assert.Empty(t, clone.Parts)
	})
}

func TestDateLiteralClone(t *testing.T) {
	t.Parallel()

	t.Run("nil returns nil", func(t *testing.T) {
		t.Parallel()

		var dl *DateLiteral
		result := dl.Clone()
		assert.Nil(t, result)
	})

	t.Run("clones all fields", func(t *testing.T) {
		t.Parallel()

		original := &DateLiteral{
			Value:            "2024-12-25",
			RelativeLocation: Location{Line: 5, Column: 10},
			SourceLength:     14,
			GoAnnotations:    &GoGeneratorAnnotation{IsStatic: true},
		}

		result := original.Clone()
		clone, ok := result.(*DateLiteral)
		require.True(t, ok)

		assert.Equal(t, "2024-12-25", clone.Value)
		assert.Equal(t, Location{Line: 5, Column: 10}, clone.RelativeLocation)
		assert.Equal(t, 14, clone.SourceLength)
		require.NotNil(t, clone.GoAnnotations)
		assert.True(t, clone.GoAnnotations.IsStatic)
		assert.NotSame(t, original.GoAnnotations, clone.GoAnnotations)
	})

	t.Run("handles nil annotations", func(t *testing.T) {
		t.Parallel()

		original := &DateLiteral{
			Value:         "2024-01-01",
			GoAnnotations: nil,
		}

		result := original.Clone()
		clone, ok := result.(*DateLiteral)
		require.True(t, ok)

		assert.Equal(t, "2024-01-01", clone.Value)
		assert.Nil(t, clone.GoAnnotations)
	})
}

func TestTimeLiteralClone(t *testing.T) {
	t.Parallel()

	t.Run("nil returns nil", func(t *testing.T) {
		t.Parallel()

		var tl *TimeLiteral
		result := tl.Clone()
		assert.Nil(t, result)
	})

	t.Run("clones all fields", func(t *testing.T) {
		t.Parallel()

		original := &TimeLiteral{
			Value:            "14:30:00",
			RelativeLocation: Location{Line: 3, Column: 7},
			SourceLength:     12,
			GoAnnotations:    &GoGeneratorAnnotation{Stringability: 3},
		}

		result := original.Clone()
		clone, ok := result.(*TimeLiteral)
		require.True(t, ok)

		assert.Equal(t, "14:30:00", clone.Value)
		assert.Equal(t, Location{Line: 3, Column: 7}, clone.RelativeLocation)
		assert.Equal(t, 12, clone.SourceLength)
		require.NotNil(t, clone.GoAnnotations)
		assert.Equal(t, 3, clone.GoAnnotations.Stringability)
		assert.NotSame(t, original.GoAnnotations, clone.GoAnnotations)
	})
}

func TestLinkedMessageExpr_Methods(t *testing.T) {
	t.Parallel()

	t.Run("String returns at-prefixed path", func(t *testing.T) {
		t.Parallel()

		lm := &LinkedMessageExpression{
			Path: &Identifier{Name: "common.greeting"},
		}
		assert.Equal(t, "@common.greeting", lm.String())
	})

	t.Run("GetSourceLength returns stored length", func(t *testing.T) {
		t.Parallel()

		lm := &LinkedMessageExpression{
			Path:         &Identifier{Name: "message"},
			SourceLength: 12,
		}
		assert.Equal(t, 12, lm.GetSourceLength())
	})

	t.Run("TransformIdentifiers transforms path", func(t *testing.T) {
		t.Parallel()

		original := &LinkedMessageExpression{
			Path:             &Identifier{Name: "common.greeting"},
			GoAnnotations:    &GoGeneratorAnnotation{IsStatic: true},
			RelativeLocation: Location{Line: 1, Column: 5},
			SourceLength:     18,
		}

		result := original.TransformIdentifiers(func(s string) string {
			return "PREFIX_" + s
		})

		transformed, ok := result.(*LinkedMessageExpression)
		require.True(t, ok)

		pathIdent, ok := transformed.Path.(*Identifier)
		require.True(t, ok)
		assert.Equal(t, "PREFIX_common.greeting", pathIdent.Name)
		assert.Same(t, original.GoAnnotations, transformed.GoAnnotations)
		assert.Equal(t, original.RelativeLocation, transformed.RelativeLocation)
		assert.Equal(t, original.SourceLength, transformed.SourceLength)
	})

	t.Run("Clone returns deep copy", func(t *testing.T) {
		t.Parallel()

		original := &LinkedMessageExpression{
			Path:             &Identifier{Name: "key.path"},
			GoAnnotations:    &GoGeneratorAnnotation{IsStatic: true},
			RelativeLocation: Location{Line: 3, Column: 10, Offset: 50},
			SourceLength:     15,
		}

		result := original.Clone()
		clone, ok := result.(*LinkedMessageExpression)
		require.True(t, ok)

		clonedPath, ok := clone.Path.(*Identifier)
		require.True(t, ok)
		assert.Equal(t, "key.path", clonedPath.Name)
		assert.NotSame(t, original.Path, clone.Path)
		assert.NotSame(t, original.GoAnnotations, clone.GoAnnotations)
		assert.Equal(t, original.RelativeLocation, clone.RelativeLocation)
		assert.Equal(t, original.SourceLength, clone.SourceLength)
	})

	t.Run("Clone nil returns nil", func(t *testing.T) {
		t.Parallel()

		var lm *LinkedMessageExpression
		result := lm.Clone()
		assert.Nil(t, result)
	})

	t.Run("SetLocation updates location and length", func(t *testing.T) {
		t.Parallel()

		lm := &LinkedMessageExpression{
			Path: &Identifier{Name: "message"},
		}

		newLocation := Location{Line: 7, Column: 20, Offset: 150}
		lm.SetLocation(newLocation, 25)

		assert.Equal(t, newLocation, lm.GetRelativeLocation())
		assert.Equal(t, 25, lm.GetSourceLength())
	})

	t.Run("GetGoAnnotation and SetGoAnnotation", func(t *testing.T) {
		t.Parallel()

		lm := &LinkedMessageExpression{
			Path: &Identifier{Name: "message"},
		}
		assert.Nil(t, lm.GetGoAnnotation())

		ann := &GoGeneratorAnnotation{IsStatic: true}
		lm.SetGoAnnotation(ann)
		assert.Same(t, ann, lm.GetGoAnnotation())

		lm.SetGoAnnotation(nil)
		assert.Nil(t, lm.GetGoAnnotation())
	})
}

func TestForInExpr_SetLocation(t *testing.T) {
	t.Parallel()

	expression := &ForInExpression{
		ItemVariable: &Identifier{Name: "item"},
		Collection:   &Identifier{Name: "items"},
	}

	newLocation := Location{Line: 5, Column: 3, Offset: 42}
	expression.SetLocation(newLocation, 30)

	assert.Equal(t, newLocation, expression.GetRelativeLocation())
	assert.Equal(t, 30, expression.GetSourceLength())

	updatedLocation := Location{Line: 10, Column: 15, Offset: 200}
	expression.SetLocation(updatedLocation, 50)

	assert.Equal(t, updatedLocation, expression.GetRelativeLocation())
	assert.Equal(t, 50, expression.GetSourceLength())
}

func TestTemplateLiteral_SetLocation(t *testing.T) {
	t.Parallel()

	tl := &TemplateLiteral{
		Parts: []TemplateLiteralPart{
			{IsLiteral: true, Literal: "Hello "},
			{IsLiteral: false, Expression: &Identifier{Name: "name"}},
		},
	}

	newLocation := Location{Line: 2, Column: 8, Offset: 20}
	tl.SetLocation(newLocation, 18)

	assert.Equal(t, newLocation, tl.GetRelativeLocation())
	assert.Equal(t, 18, tl.GetSourceLength())

	updatedLocation := Location{Line: 12, Column: 1, Offset: 300}
	tl.SetLocation(updatedLocation, 42)

	assert.Equal(t, updatedLocation, tl.GetRelativeLocation())
	assert.Equal(t, 42, tl.GetSourceLength())
}
