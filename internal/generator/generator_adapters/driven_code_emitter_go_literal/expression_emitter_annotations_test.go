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

package driven_code_emitter_go_literal

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"piko.sh/piko/internal/ast/ast_domain"
)

func TestGetAnnotationFromOperatorExpr(t *testing.T) {
	t.Parallel()

	expectedAnn := &ast_domain.GoGeneratorAnnotation{
		Stringability: 42,
	}

	testCases := []struct {
		expression ast_domain.Expression
		name       string
		wantNil    bool
	}{
		{
			name: "BinaryExpr with GoAnnotations",
			expression: &ast_domain.BinaryExpression{
				Operator:      ast_domain.OpPlus,
				Left:          &ast_domain.IntegerLiteral{Value: 1},
				Right:         &ast_domain.IntegerLiteral{Value: 2},
				GoAnnotations: expectedAnn,
			},
			wantNil: false,
		},
		{
			name: "UnaryExpr with GoAnnotations",
			expression: &ast_domain.UnaryExpression{
				Operator:      ast_domain.OpNeg,
				Right:         &ast_domain.IntegerLiteral{Value: 1},
				GoAnnotations: expectedAnn,
			},
			wantNil: false,
		},
		{
			name: "TernaryExpr with GoAnnotations",
			expression: &ast_domain.TernaryExpression{
				Condition:     &ast_domain.BooleanLiteral{Value: true},
				Consequent:    &ast_domain.IntegerLiteral{Value: 1},
				Alternate:     &ast_domain.IntegerLiteral{Value: 2},
				GoAnnotations: expectedAnn,
			},
			wantNil: false,
		},
		{
			name: "MemberExpr with GoAnnotations",
			expression: &ast_domain.MemberExpression{
				Base:          &ast_domain.Identifier{Name: "obj"},
				Property:      &ast_domain.Identifier{Name: "field"},
				GoAnnotations: expectedAnn,
			},
			wantNil: false,
		},
		{
			name: "IndexExpr with GoAnnotations",
			expression: &ast_domain.IndexExpression{
				Base:          &ast_domain.Identifier{Name: "arr"},
				Index:         &ast_domain.IntegerLiteral{Value: 0},
				GoAnnotations: expectedAnn,
			},
			wantNil: false,
		},
		{
			name: "CallExpr with GoAnnotations",
			expression: &ast_domain.CallExpression{
				Callee:        &ast_domain.Identifier{Name: "fn"},
				GoAnnotations: expectedAnn,
			},
			wantNil: false,
		},
		{
			name:       "unsupported type returns nil",
			expression: &ast_domain.StringLiteral{Value: "hello"},
			wantNil:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := getAnnotationFromOperatorExpr(tc.expression)

			if tc.wantNil {
				assert.Nil(t, result, "Expected nil for %s", tc.name)
			} else {
				assert.NotNil(t, result, "Expected non-nil annotation for %s", tc.name)
				assert.Same(t, expectedAnn, result, "Should return the exact same annotation pointer")
			}
		})
	}
}

func TestGetEffectiveKeyExpression(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		want ast_domain.Expression
		node *ast_domain.TemplateNode
		name string
	}{
		{
			name: "nil GoAnnotations returns node Key",
			node: &ast_domain.TemplateNode{
				GoAnnotations: nil,
				Key:           &ast_domain.StringLiteral{Value: "structural-key"},
			},
			want: &ast_domain.StringLiteral{Value: "structural-key"},
		},
		{
			name: "nil EffectiveKeyExpression returns node Key",
			node: &ast_domain.TemplateNode{
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					EffectiveKeyExpression: nil,
				},
				Key: &ast_domain.StringLiteral{Value: "structural-key"},
			},
			want: &ast_domain.StringLiteral{Value: "structural-key"},
		},
		{
			name: "EffectiveKeyExpression overrides node Key",
			node: func() *ast_domain.TemplateNode {
				overrideExpr := &ast_domain.Identifier{Name: "dynamicKey"}
				return &ast_domain.TemplateNode{
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						EffectiveKeyExpression: overrideExpr,
					},
					Key: &ast_domain.StringLiteral{Value: "structural-key"},
				}
			}(),
			want: &ast_domain.Identifier{Name: "dynamicKey"},
		},
		{
			name: "nil Key with nil GoAnnotations returns nil",
			node: &ast_domain.TemplateNode{
				GoAnnotations: nil,
				Key:           nil,
			},
			want: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := getEffectiveKeyExpression(tc.node)

			if tc.want == nil {
				assert.Nil(t, result)
			} else {
				assert.Equal(t, tc.want, result)
			}
		})
	}
}

func TestGetAnnotationFromLiteralExpr(t *testing.T) {
	t.Parallel()

	expectedAnn := &ast_domain.GoGeneratorAnnotation{
		Stringability: 1,
	}

	testCases := []struct {
		expression ast_domain.Expression
		name       string
		wantNil    bool
	}{
		{
			name: "StringLiteral",
			expression: &ast_domain.StringLiteral{
				Value:         "hello",
				GoAnnotations: expectedAnn,
			},
			wantNil: false,
		},
		{
			name: "IntegerLiteral",
			expression: &ast_domain.IntegerLiteral{
				Value:         42,
				GoAnnotations: expectedAnn,
			},
			wantNil: false,
		},
		{
			name: "FloatLiteral",
			expression: &ast_domain.FloatLiteral{
				Value:         3.14,
				GoAnnotations: expectedAnn,
			},
			wantNil: false,
		},
		{
			name: "BooleanLiteral",
			expression: &ast_domain.BooleanLiteral{
				Value:         true,
				GoAnnotations: expectedAnn,
			},
			wantNil: false,
		},
		{
			name: "NilLiteral",
			expression: &ast_domain.NilLiteral{
				GoAnnotations: expectedAnn,
			},
			wantNil: false,
		},
		{
			name: "DecimalLiteral",
			expression: &ast_domain.DecimalLiteral{
				Value:         "123.45",
				GoAnnotations: expectedAnn,
			},
			wantNil: false,
		},
		{
			name: "BigIntLiteral",
			expression: &ast_domain.BigIntLiteral{
				Value:         "99999999999999999999",
				GoAnnotations: expectedAnn,
			},
			wantNil: false,
		},
		{
			name: "DateTimeLiteral",
			expression: &ast_domain.DateTimeLiteral{
				Value:         "2024-01-01T00:00:00Z",
				GoAnnotations: expectedAnn,
			},
			wantNil: false,
		},
		{
			name: "DateLiteral",
			expression: &ast_domain.DateLiteral{
				Value:         "2024-01-01",
				GoAnnotations: expectedAnn,
			},
			wantNil: false,
		},
		{
			name: "TimeLiteral",
			expression: &ast_domain.TimeLiteral{
				Value:         "12:00:00",
				GoAnnotations: expectedAnn,
			},
			wantNil: false,
		},
		{
			name: "DurationLiteral",
			expression: &ast_domain.DurationLiteral{
				Value:         "5s",
				GoAnnotations: expectedAnn,
			},
			wantNil: false,
		},
		{
			name: "RuneLiteral",
			expression: &ast_domain.RuneLiteral{
				Value:         'A',
				GoAnnotations: expectedAnn,
			},
			wantNil: false,
		},
		{
			name:       "unsupported type returns nil",
			expression: &ast_domain.Identifier{Name: "x"},
			wantNil:    true,
		},
		{
			name: "CallExpr returns nil (not a literal)",
			expression: &ast_domain.CallExpression{
				Callee:        &ast_domain.Identifier{Name: "fn"},
				GoAnnotations: expectedAnn,
			},
			wantNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := getAnnotationFromLiteralExpr(tc.expression)

			if tc.wantNil {
				assert.Nil(t, result, "Expected nil for %s", tc.name)
			} else {
				assert.NotNil(t, result, "Expected non-nil annotation for %s", tc.name)
				assert.Same(t, expectedAnn, result, "Should return the exact same annotation pointer")
			}
		})
	}
}
