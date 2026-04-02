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

package annotator_domain

import (
	goast "go/ast"
	"testing"

	"github.com/stretchr/testify/assert"
	"piko.sh/piko/internal/ast/ast_domain"
)

func TestExtractNilGuardsFromCondition(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		expression ast_domain.Expression
		expected   []string
	}{
		{
			name:       "nil input returns nil",
			expression: nil,
			expected:   nil,
		},
		{
			name: "identifier != nil extracts identifier",
			expression: &ast_domain.BinaryExpression{
				Left:     &ast_domain.Identifier{Name: "user"},
				Right:    &ast_domain.NilLiteral{},
				Operator: ast_domain.OpNe,
			},
			expected: []string{"user"},
		},
		{
			name: "identifier !== nil (loose ne) extracts identifier",
			expression: &ast_domain.BinaryExpression{
				Left:     &ast_domain.Identifier{Name: "ptr"},
				Right:    &ast_domain.NilLiteral{},
				Operator: ast_domain.OpLooseNe,
			},
			expected: []string{"ptr"},
		},
		{
			name: "nil != identifier extracts identifier",
			expression: &ast_domain.BinaryExpression{
				Left:     &ast_domain.NilLiteral{},
				Right:    &ast_domain.Identifier{Name: "data"},
				Operator: ast_domain.OpNe,
			},
			expected: []string{"data"},
		},
		{
			name: "member expression != nil extracts full path",
			expression: &ast_domain.BinaryExpression{
				Left: &ast_domain.MemberExpression{
					Base:     &ast_domain.Identifier{Name: "user"},
					Property: &ast_domain.Identifier{Name: "profile"},
				},
				Right:    &ast_domain.NilLiteral{},
				Operator: ast_domain.OpNe,
			},
			expected: []string{"user.profile"},
		},
		{
			name: "negated equality !(expr == nil) extracts identifier",
			expression: &ast_domain.UnaryExpression{
				Operator: ast_domain.OpNot,
				Right: &ast_domain.BinaryExpression{
					Left:     &ast_domain.Identifier{Name: "item"},
					Right:    &ast_domain.NilLiteral{},
					Operator: ast_domain.OpEq,
				},
			},
			expected: []string{"item"},
		},
		{
			name: "negated equality !(nil == expr) extracts identifier",
			expression: &ast_domain.UnaryExpression{
				Operator: ast_domain.OpNot,
				Right: &ast_domain.BinaryExpression{
					Left:     &ast_domain.NilLiteral{},
					Right:    &ast_domain.Identifier{Name: "record"},
					Operator: ast_domain.OpEq,
				},
			},
			expected: []string{"record"},
		},
		{
			name: "negated loose equality !(expr ~= nil) extracts identifier",
			expression: &ast_domain.UnaryExpression{
				Operator: ast_domain.OpNot,
				Right: &ast_domain.BinaryExpression{
					Left:     &ast_domain.Identifier{Name: "value"},
					Right:    &ast_domain.NilLiteral{},
					Operator: ast_domain.OpLooseEq,
				},
			},
			expected: []string{"value"},
		},
		{
			name: "AND chain extracts multiple guards",
			expression: &ast_domain.BinaryExpression{
				Left: &ast_domain.BinaryExpression{
					Left:     &ast_domain.Identifier{Name: "a"},
					Right:    &ast_domain.NilLiteral{},
					Operator: ast_domain.OpNe,
				},
				Right: &ast_domain.BinaryExpression{
					Left:     &ast_domain.Identifier{Name: "b"},
					Right:    &ast_domain.NilLiteral{},
					Operator: ast_domain.OpNe,
				},
				Operator: ast_domain.OpAnd,
			},
			expected: []string{"a", "b"},
		},
		{
			name: "nested AND chain extracts all guards",
			expression: &ast_domain.BinaryExpression{
				Left: &ast_domain.BinaryExpression{
					Left: &ast_domain.BinaryExpression{
						Left:     &ast_domain.Identifier{Name: "x"},
						Right:    &ast_domain.NilLiteral{},
						Operator: ast_domain.OpNe,
					},
					Right: &ast_domain.BinaryExpression{
						Left:     &ast_domain.Identifier{Name: "y"},
						Right:    &ast_domain.NilLiteral{},
						Operator: ast_domain.OpNe,
					},
					Operator: ast_domain.OpAnd,
				},
				Right: &ast_domain.BinaryExpression{
					Left:     &ast_domain.Identifier{Name: "z"},
					Right:    &ast_domain.NilLiteral{},
					Operator: ast_domain.OpNe,
				},
				Operator: ast_domain.OpAnd,
			},
			expected: []string{"x", "y", "z"},
		},
		{
			name: "equality check (== nil) does not extract",
			expression: &ast_domain.BinaryExpression{
				Left:     &ast_domain.Identifier{Name: "shouldBeNil"},
				Right:    &ast_domain.NilLiteral{},
				Operator: ast_domain.OpEq,
			},
			expected: nil,
		},
		{
			name: "OR operator does not extract guards",
			expression: &ast_domain.BinaryExpression{
				Left: &ast_domain.BinaryExpression{
					Left:     &ast_domain.Identifier{Name: "a"},
					Right:    &ast_domain.NilLiteral{},
					Operator: ast_domain.OpNe,
				},
				Right: &ast_domain.BinaryExpression{
					Left:     &ast_domain.Identifier{Name: "b"},
					Right:    &ast_domain.NilLiteral{},
					Operator: ast_domain.OpNe,
				},
				Operator: ast_domain.OpOr,
			},
			expected: nil,
		},
		{
			name: "non-nil comparison does not extract",
			expression: &ast_domain.BinaryExpression{
				Left:     &ast_domain.Identifier{Name: "count"},
				Right:    &ast_domain.IntegerLiteral{Value: 0},
				Operator: ast_domain.OpNe,
			},
			expected: nil,
		},
		{
			name: "unary negation without binary does not extract",
			expression: &ast_domain.UnaryExpression{
				Operator: ast_domain.OpNot,
				Right:    &ast_domain.Identifier{Name: "flag"},
			},
			expected: nil,
		},
		{
			name: "unary negation of non-equality binary does not extract",
			expression: &ast_domain.UnaryExpression{
				Operator: ast_domain.OpNot,
				Right: &ast_domain.BinaryExpression{
					Left:     &ast_domain.Identifier{Name: "a"},
					Right:    &ast_domain.Identifier{Name: "b"},
					Operator: ast_domain.OpGt,
				},
			},
			expected: nil,
		},
		{
			name: "bare pointer identifier with annotation extracts guard",
			expression: func() ast_domain.Expression {
				id := &ast_domain.Identifier{Name: "ptr"}
				id.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
					ResolvedType: &ast_domain.ResolvedTypeInfo{
						TypeExpression: &goast.StarExpr{X: goast.NewIdent("User")},
					},
				}
				return id
			}(),
			expected: []string{"ptr"},
		},
		{
			name: "bare member expression with pointer annotation extracts guard",
			expression: func() ast_domain.Expression {
				me := &ast_domain.MemberExpression{
					Base:     &ast_domain.Identifier{Name: "user"},
					Property: &ast_domain.Identifier{Name: "profile"},
				}
				me.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
					ResolvedType: &ast_domain.ResolvedTypeInfo{
						TypeExpression: &goast.StarExpr{X: goast.NewIdent("Profile")},
					},
				}
				return me
			}(),
			expected: []string{"user.profile"},
		},
		{
			name:       "bare identifier without annotation does not extract",
			expression: &ast_domain.Identifier{Name: "unknown"},
			expected:   nil,
		},
		{
			name: "bare identifier with non-pointer type does not extract",
			expression: func() ast_domain.Expression {
				id := &ast_domain.Identifier{Name: "count"}
				id.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
					ResolvedType: &ast_domain.ResolvedTypeInfo{
						TypeExpression: goast.NewIdent("int"),
					},
				}
				return id
			}(),
			expected: nil,
		},
		{
			name: "bare identifier with nil ResolvedType does not extract",
			expression: func() ast_domain.Expression {
				id := &ast_domain.Identifier{Name: "item"}
				id.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
					ResolvedType: nil,
				}
				return id
			}(),
			expected: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := ExtractNilGuardsFromCondition(tc.expression)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestIsNilLiteral(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		expression ast_domain.Expression
		name       string
		expected   bool
	}{
		{
			name:       "NilLiteral returns true",
			expression: &ast_domain.NilLiteral{},
			expected:   true,
		},
		{
			name:       "Identifier returns false",
			expression: &ast_domain.Identifier{Name: "nil"},
			expected:   false,
		},
		{
			name:       "StringLiteral returns false",
			expression: &ast_domain.StringLiteral{Value: "nil"},
			expected:   false,
		},
		{
			name:       "IntegerLiteral zero returns false",
			expression: &ast_domain.IntegerLiteral{Value: 0},
			expected:   false,
		},
		{
			name:       "BooleanLiteral false returns false",
			expression: &ast_domain.BooleanLiteral{Value: false},
			expected:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := isNilLiteral(tc.expression)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestIsPointerTypeExpr(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		typeExpr goast.Expr
		name     string
		expected bool
	}{
		{
			name:     "StarExpr is pointer",
			typeExpr: &goast.StarExpr{X: goast.NewIdent("User")},
			expected: true,
		},
		{
			name:     "nested StarExpr is pointer",
			typeExpr: &goast.StarExpr{X: &goast.StarExpr{X: goast.NewIdent("int")}},
			expected: true,
		},
		{
			name:     "Ident is not pointer",
			typeExpr: goast.NewIdent("string"),
			expected: false,
		},
		{
			name:     "ArrayType is not pointer",
			typeExpr: &goast.ArrayType{Elt: goast.NewIdent("int")},
			expected: false,
		},
		{
			name:     "MapType is not pointer",
			typeExpr: &goast.MapType{Key: goast.NewIdent("string"), Value: goast.NewIdent("int")},
			expected: false,
		},
		{
			name:     "SelectorExpr is not pointer",
			typeExpr: &goast.SelectorExpr{X: goast.NewIdent("time"), Sel: goast.NewIdent("Time")},
			expected: false,
		},
		{
			name:     "nil is not pointer",
			typeExpr: nil,
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := isPointerTypeExpr(tc.typeExpr)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestHandleBinaryExpr(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		expression *ast_domain.BinaryExpression
		expected   []string
	}{
		{
			name: "OpNe with nil on right",
			expression: &ast_domain.BinaryExpression{
				Left:     &ast_domain.Identifier{Name: "x"},
				Right:    &ast_domain.NilLiteral{},
				Operator: ast_domain.OpNe,
			},
			expected: []string{"x"},
		},
		{
			name: "OpNe with nil on left",
			expression: &ast_domain.BinaryExpression{
				Left:     &ast_domain.NilLiteral{},
				Right:    &ast_domain.Identifier{Name: "y"},
				Operator: ast_domain.OpNe,
			},
			expected: []string{"y"},
		},
		{
			name: "OpLooseNe with nil",
			expression: &ast_domain.BinaryExpression{
				Left:     &ast_domain.Identifier{Name: "z"},
				Right:    &ast_domain.NilLiteral{},
				Operator: ast_domain.OpLooseNe,
			},
			expected: []string{"z"},
		},
		{
			name: "OpAnd recursively extracts from both sides",
			expression: &ast_domain.BinaryExpression{
				Left: &ast_domain.BinaryExpression{
					Left:     &ast_domain.Identifier{Name: "left"},
					Right:    &ast_domain.NilLiteral{},
					Operator: ast_domain.OpNe,
				},
				Right: &ast_domain.BinaryExpression{
					Left:     &ast_domain.Identifier{Name: "right"},
					Right:    &ast_domain.NilLiteral{},
					Operator: ast_domain.OpNe,
				},
				Operator: ast_domain.OpAnd,
			},
			expected: []string{"left", "right"},
		},
		{
			name: "OpEq does not extract",
			expression: &ast_domain.BinaryExpression{
				Left:     &ast_domain.Identifier{Name: "eq"},
				Right:    &ast_domain.NilLiteral{},
				Operator: ast_domain.OpEq,
			},
			expected: nil,
		},
		{
			name: "comparison without nil does not extract",
			expression: &ast_domain.BinaryExpression{
				Left:     &ast_domain.Identifier{Name: "a"},
				Right:    &ast_domain.Identifier{Name: "b"},
				Operator: ast_domain.OpNe,
			},
			expected: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var guards []string
			handleBinaryExpr(tc.expression, &guards)
			if tc.expected == nil {
				assert.Empty(t, guards)
			} else {
				assert.Equal(t, tc.expected, guards)
			}
		})
	}
}

func TestHandleUnaryExpr(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		expression *ast_domain.UnaryExpression
		expected   []string
	}{
		{
			name: "negated equality with nil on right",
			expression: &ast_domain.UnaryExpression{
				Operator: ast_domain.OpNot,
				Right: &ast_domain.BinaryExpression{
					Left:     &ast_domain.Identifier{Name: "ptr"},
					Right:    &ast_domain.NilLiteral{},
					Operator: ast_domain.OpEq,
				},
			},
			expected: []string{"ptr"},
		},
		{
			name: "negated equality with nil on left",
			expression: &ast_domain.UnaryExpression{
				Operator: ast_domain.OpNot,
				Right: &ast_domain.BinaryExpression{
					Left:     &ast_domain.NilLiteral{},
					Right:    &ast_domain.Identifier{Name: "ref"},
					Operator: ast_domain.OpEq,
				},
			},
			expected: []string{"ref"},
		},
		{
			name: "negated loose equality",
			expression: &ast_domain.UnaryExpression{
				Operator: ast_domain.OpNot,
				Right: &ast_domain.BinaryExpression{
					Left:     &ast_domain.Identifier{Name: "val"},
					Right:    &ast_domain.NilLiteral{},
					Operator: ast_domain.OpLooseEq,
				},
			},
			expected: []string{"val"},
		},
		{
			name: "non-not operator does not extract",
			expression: &ast_domain.UnaryExpression{
				Operator: ast_domain.OpNeg,
				Right: &ast_domain.BinaryExpression{
					Left:     &ast_domain.Identifier{Name: "num"},
					Right:    &ast_domain.NilLiteral{},
					Operator: ast_domain.OpEq,
				},
			},
			expected: nil,
		},
		{
			name: "not operator with non-binary does not extract",
			expression: &ast_domain.UnaryExpression{
				Operator: ast_domain.OpNot,
				Right:    &ast_domain.Identifier{Name: "flag"},
			},
			expected: nil,
		},
		{
			name: "not operator with inequality does not extract",
			expression: &ast_domain.UnaryExpression{
				Operator: ast_domain.OpNot,
				Right: &ast_domain.BinaryExpression{
					Left:     &ast_domain.Identifier{Name: "a"},
					Right:    &ast_domain.NilLiteral{},
					Operator: ast_domain.OpNe,
				},
			},
			expected: nil,
		},
		{
			name: "not operator with non-nil comparison does not extract",
			expression: &ast_domain.UnaryExpression{
				Operator: ast_domain.OpNot,
				Right: &ast_domain.BinaryExpression{
					Left:     &ast_domain.Identifier{Name: "x"},
					Right:    &ast_domain.IntegerLiteral{Value: 0},
					Operator: ast_domain.OpEq,
				},
			},
			expected: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var guards []string
			handleUnaryExpr(tc.expression, &guards)
			if tc.expected == nil {
				assert.Empty(t, guards)
			} else {
				assert.Equal(t, tc.expected, guards)
			}
		})
	}
}

func TestHandleTruthinessCheck(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		expression ast_domain.Expression
		expected   []string
	}{
		{
			name: "pointer type with annotation extracts guard",
			expression: func() ast_domain.Expression {
				id := &ast_domain.Identifier{Name: "ptr"}
				id.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
					ResolvedType: &ast_domain.ResolvedTypeInfo{
						TypeExpression: &goast.StarExpr{X: goast.NewIdent("T")},
					},
				}
				return id
			}(),
			expected: []string{"ptr"},
		},
		{
			name: "member expr with pointer type extracts guard",
			expression: func() ast_domain.Expression {
				me := &ast_domain.MemberExpression{
					Base:     &ast_domain.Identifier{Name: "obj"},
					Property: &ast_domain.Identifier{Name: "ptr"},
				}
				me.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
					ResolvedType: &ast_domain.ResolvedTypeInfo{
						TypeExpression: &goast.StarExpr{X: goast.NewIdent("Data")},
					},
				}
				return me
			}(),
			expected: []string{"obj.ptr"},
		},
		{
			name:       "no annotation does not extract",
			expression: &ast_domain.Identifier{Name: "noAnn"},
			expected:   nil,
		},
		{
			name: "annotation with nil ResolvedType does not extract",
			expression: func() ast_domain.Expression {
				id := &ast_domain.Identifier{Name: "nilType"}
				id.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
					ResolvedType: nil,
				}
				return id
			}(),
			expected: nil,
		},
		{
			name: "non-pointer type does not extract",
			expression: func() ast_domain.Expression {
				id := &ast_domain.Identifier{Name: "notPtr"}
				id.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
					ResolvedType: &ast_domain.ResolvedTypeInfo{
						TypeExpression: goast.NewIdent("int"),
					},
				}
				return id
			}(),
			expected: nil,
		},
		{
			name: "slice type does not extract",
			expression: func() ast_domain.Expression {
				id := &ast_domain.Identifier{Name: "slice"}
				id.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
					ResolvedType: &ast_domain.ResolvedTypeInfo{
						TypeExpression: &goast.ArrayType{Elt: goast.NewIdent("int")},
					},
				}
				return id
			}(),
			expected: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var guards []string
			handleTruthinessCheck(tc.expression, &guards)
			if tc.expected == nil {
				assert.Empty(t, guards)
			} else {
				assert.Equal(t, tc.expected, guards)
			}
		})
	}
}
