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

package ast_adapters

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
)

func TestEncodeDecodeAST_Identifier(t *testing.T) {
	testCases := []struct {
		identifier *ast_domain.Identifier
		expected   *ast_domain.Identifier
		name       string
	}{
		{
			name: "simple identifier",
			identifier: &ast_domain.Identifier{
				Name:             "userName",
				RelativeLocation: ast_domain.Location{Line: 1, Column: 5},
				SourceLength:     8,
			},
			expected: &ast_domain.Identifier{
				Name:             "userName",
				RelativeLocation: ast_domain.Location{Line: 1, Column: 5},
				SourceLength:     8,
			},
		},
		{
			name: "single character identifier",
			identifier: &ast_domain.Identifier{
				Name:             "x",
				RelativeLocation: ast_domain.Location{Line: 0, Column: 0},
				SourceLength:     1,
			},
			expected: &ast_domain.Identifier{
				Name:             "x",
				RelativeLocation: ast_domain.Location{Line: 0, Column: 0},
				SourceLength:     1,
			},
		},
		{
			name: "identifier with underscores",
			identifier: &ast_domain.Identifier{
				Name:             "user_profile_data",
				RelativeLocation: ast_domain.Location{Line: 10, Column: 20},
				SourceLength:     17,
			},
			expected: &ast_domain.Identifier{
				Name:             "user_profile_data",
				RelativeLocation: ast_domain.Location{Line: 10, Column: 20},
				SourceLength:     17,
			},
		},
		{
			name: "identifier with numbers",
			identifier: &ast_domain.Identifier{
				Name:             "item123",
				RelativeLocation: ast_domain.Location{Line: 5, Column: 15},
				SourceLength:     7,
			},
			expected: &ast_domain.Identifier{
				Name:             "item123",
				RelativeLocation: ast_domain.Location{Line: 5, Column: 15},
				SourceLength:     7,
			},
		},
		{
			name: "unicode identifier",
			identifier: &ast_domain.Identifier{
				Name:             "变量名",
				RelativeLocation: ast_domain.Location{Line: 2, Column: 4},
				SourceLength:     9,
			},
			expected: &ast_domain.Identifier{
				Name:             "变量名",
				RelativeLocation: ast_domain.Location{Line: 2, Column: 4},
				SourceLength:     9,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			original := &ast_domain.TemplateAST{
				RootNodes: []*ast_domain.TemplateNode{
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "div",
						DynamicAttributes: []ast_domain.DynamicAttribute{
							{
								Name:       "data-id",
								Expression: tc.identifier,
							},
						},
					},
				},
			}

			decoded := mustRoundTrip(t, original)

			require.Len(t, decoded.RootNodes, 1)
			require.Len(t, decoded.RootNodes[0].DynamicAttributes, 1)

			decodedIdent, ok := decoded.RootNodes[0].DynamicAttributes[0].Expression.(*ast_domain.Identifier)
			require.True(t, ok, "expected Identifier type")
			assert.Equal(t, tc.expected.Name, decodedIdent.Name)
			assert.Equal(t, tc.expected.RelativeLocation, decodedIdent.RelativeLocation)
			assert.Equal(t, tc.expected.SourceLength, decodedIdent.SourceLength)
		})
	}
}

func TestEncodeDecodeAST_StringLiteral(t *testing.T) {
	testCases := []struct {
		literal  *ast_domain.StringLiteral
		expected *ast_domain.StringLiteral
		name     string
	}{
		{
			name: "simple string",
			literal: &ast_domain.StringLiteral{
				Value:            "hello world",
				RelativeLocation: ast_domain.Location{Line: 1, Column: 5},
				SourceLength:     13,
			},
			expected: &ast_domain.StringLiteral{
				Value:            "hello world",
				RelativeLocation: ast_domain.Location{Line: 1, Column: 5},
				SourceLength:     13,
			},
		},
		{
			name: "empty string",
			literal: &ast_domain.StringLiteral{
				Value:            "",
				RelativeLocation: ast_domain.Location{Line: 0, Column: 0},
				SourceLength:     2,
			},
			expected: &ast_domain.StringLiteral{
				Value:            "",
				RelativeLocation: ast_domain.Location{Line: 0, Column: 0},
				SourceLength:     2,
			},
		},
		{
			name: "string with special characters",
			literal: &ast_domain.StringLiteral{
				Value:            "line1\nline2\ttab",
				RelativeLocation: ast_domain.Location{Line: 2, Column: 10},
				SourceLength:     20,
			},
			expected: &ast_domain.StringLiteral{
				Value:            "line1\nline2\ttab",
				RelativeLocation: ast_domain.Location{Line: 2, Column: 10},
				SourceLength:     20,
			},
		},
		{
			name: "string with unicode",
			literal: &ast_domain.StringLiteral{
				Value:            "こんにちは世界",
				RelativeLocation: ast_domain.Location{Line: 5, Column: 3},
				SourceLength:     23,
			},
			expected: &ast_domain.StringLiteral{
				Value:            "こんにちは世界",
				RelativeLocation: ast_domain.Location{Line: 5, Column: 3},
				SourceLength:     23,
			},
		},
		{
			name: "string with escaped quotes",
			literal: &ast_domain.StringLiteral{
				Value:            `he said "hello"`,
				RelativeLocation: ast_domain.Location{Line: 1, Column: 0},
				SourceLength:     19,
			},
			expected: &ast_domain.StringLiteral{
				Value:            `he said "hello"`,
				RelativeLocation: ast_domain.Location{Line: 1, Column: 0},
				SourceLength:     19,
			},
		},
		{
			name: "string with backslash",
			literal: &ast_domain.StringLiteral{
				Value:            `C:\path\to\file`,
				RelativeLocation: ast_domain.Location{Line: 3, Column: 8},
				SourceLength:     17,
			},
			expected: &ast_domain.StringLiteral{
				Value:            `C:\path\to\file`,
				RelativeLocation: ast_domain.Location{Line: 3, Column: 8},
				SourceLength:     17,
			},
		},
		{
			name: "string with emoji",
			literal: &ast_domain.StringLiteral{
				Value:            "Hello 🌍🚀",
				RelativeLocation: ast_domain.Location{Line: 1, Column: 1},
				SourceLength:     18,
			},
			expected: &ast_domain.StringLiteral{
				Value:            "Hello 🌍🚀",
				RelativeLocation: ast_domain.Location{Line: 1, Column: 1},
				SourceLength:     18,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			original := &ast_domain.TemplateAST{
				RootNodes: []*ast_domain.TemplateNode{
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "div",
						DynamicAttributes: []ast_domain.DynamicAttribute{
							{
								Name:       "value",
								Expression: tc.literal,
							},
						},
					},
				},
			}

			decoded := mustRoundTrip(t, original)

			require.Len(t, decoded.RootNodes, 1)
			require.Len(t, decoded.RootNodes[0].DynamicAttributes, 1)

			decodedLit, ok := decoded.RootNodes[0].DynamicAttributes[0].Expression.(*ast_domain.StringLiteral)
			require.True(t, ok, "expected StringLiteral type")
			assert.Equal(t, tc.expected.Value, decodedLit.Value)
			assert.Equal(t, tc.expected.RelativeLocation, decodedLit.RelativeLocation)
			assert.Equal(t, tc.expected.SourceLength, decodedLit.SourceLength)
		})
	}
}

func TestEncodeDecodeAST_IntegerLiteral(t *testing.T) {
	testCases := []struct {
		literal  *ast_domain.IntegerLiteral
		expected *ast_domain.IntegerLiteral
		name     string
	}{
		{
			name: "positive integer",
			literal: &ast_domain.IntegerLiteral{
				Value:            42,
				RelativeLocation: ast_domain.Location{Line: 1, Column: 5},
				SourceLength:     2,
			},
			expected: &ast_domain.IntegerLiteral{
				Value:            42,
				RelativeLocation: ast_domain.Location{Line: 1, Column: 5},
				SourceLength:     2,
			},
		},
		{
			name: "zero",
			literal: &ast_domain.IntegerLiteral{
				Value:            0,
				RelativeLocation: ast_domain.Location{Line: 0, Column: 0},
				SourceLength:     1,
			},
			expected: &ast_domain.IntegerLiteral{
				Value:            0,
				RelativeLocation: ast_domain.Location{Line: 0, Column: 0},
				SourceLength:     1,
			},
		},
		{
			name: "negative integer",
			literal: &ast_domain.IntegerLiteral{
				Value:            -123,
				RelativeLocation: ast_domain.Location{Line: 2, Column: 10},
				SourceLength:     4,
			},
			expected: &ast_domain.IntegerLiteral{
				Value:            -123,
				RelativeLocation: ast_domain.Location{Line: 2, Column: 10},
				SourceLength:     4,
			},
		},
		{
			name: "large positive integer",
			literal: &ast_domain.IntegerLiteral{
				Value:            9223372036854775807,
				RelativeLocation: ast_domain.Location{Line: 5, Column: 3},
				SourceLength:     19,
			},
			expected: &ast_domain.IntegerLiteral{
				Value:            9223372036854775807,
				RelativeLocation: ast_domain.Location{Line: 5, Column: 3},
				SourceLength:     19,
			},
		},
		{
			name: "large negative integer",
			literal: &ast_domain.IntegerLiteral{
				Value:            -9223372036854775808,
				RelativeLocation: ast_domain.Location{Line: 3, Column: 0},
				SourceLength:     20,
			},
			expected: &ast_domain.IntegerLiteral{
				Value:            -9223372036854775808,
				RelativeLocation: ast_domain.Location{Line: 3, Column: 0},
				SourceLength:     20,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			original := &ast_domain.TemplateAST{
				RootNodes: []*ast_domain.TemplateNode{
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "div",
						DynamicAttributes: []ast_domain.DynamicAttribute{
							{
								Name:       "count",
								Expression: tc.literal,
							},
						},
					},
				},
			}

			decoded := mustRoundTrip(t, original)

			require.Len(t, decoded.RootNodes, 1)
			require.Len(t, decoded.RootNodes[0].DynamicAttributes, 1)

			decodedLit, ok := decoded.RootNodes[0].DynamicAttributes[0].Expression.(*ast_domain.IntegerLiteral)
			require.True(t, ok, "expected IntegerLiteral type")
			assert.Equal(t, tc.expected.Value, decodedLit.Value)
			assert.Equal(t, tc.expected.RelativeLocation, decodedLit.RelativeLocation)
			assert.Equal(t, tc.expected.SourceLength, decodedLit.SourceLength)
		})
	}
}

func TestEncodeDecodeAST_FloatLiteral(t *testing.T) {
	testCases := []struct {
		literal  *ast_domain.FloatLiteral
		expected *ast_domain.FloatLiteral
		name     string
	}{
		{
			name: "simple float",
			literal: &ast_domain.FloatLiteral{
				Value:            3.14,
				RelativeLocation: ast_domain.Location{Line: 1, Column: 5},
				SourceLength:     4,
			},
			expected: &ast_domain.FloatLiteral{
				Value:            3.14,
				RelativeLocation: ast_domain.Location{Line: 1, Column: 5},
				SourceLength:     4,
			},
		},
		{
			name: "zero float",
			literal: &ast_domain.FloatLiteral{
				Value:            0.0,
				RelativeLocation: ast_domain.Location{Line: 0, Column: 0},
				SourceLength:     3,
			},
			expected: &ast_domain.FloatLiteral{
				Value:            0.0,
				RelativeLocation: ast_domain.Location{Line: 0, Column: 0},
				SourceLength:     3,
			},
		},
		{
			name: "negative float",
			literal: &ast_domain.FloatLiteral{
				Value:            -2.718,
				RelativeLocation: ast_domain.Location{Line: 2, Column: 10},
				SourceLength:     6,
			},
			expected: &ast_domain.FloatLiteral{
				Value:            -2.718,
				RelativeLocation: ast_domain.Location{Line: 2, Column: 10},
				SourceLength:     6,
			},
		},
		{
			name: "small float",
			literal: &ast_domain.FloatLiteral{
				Value:            0.000001,
				RelativeLocation: ast_domain.Location{Line: 5, Column: 3},
				SourceLength:     8,
			},
			expected: &ast_domain.FloatLiteral{
				Value:            0.000001,
				RelativeLocation: ast_domain.Location{Line: 5, Column: 3},
				SourceLength:     8,
			},
		},
		{
			name: "large float",
			literal: &ast_domain.FloatLiteral{
				Value:            1.7976931348623157e+308,
				RelativeLocation: ast_domain.Location{Line: 3, Column: 0},
				SourceLength:     25,
			},
			expected: &ast_domain.FloatLiteral{
				Value:            1.7976931348623157e+308,
				RelativeLocation: ast_domain.Location{Line: 3, Column: 0},
				SourceLength:     25,
			},
		},
		{
			name: "negative infinity",
			literal: &ast_domain.FloatLiteral{
				Value:            math.Inf(-1),
				RelativeLocation: ast_domain.Location{Line: 1, Column: 0},
				SourceLength:     4,
			},
			expected: &ast_domain.FloatLiteral{
				Value:            math.Inf(-1),
				RelativeLocation: ast_domain.Location{Line: 1, Column: 0},
				SourceLength:     4,
			},
		},
		{
			name: "positive infinity",
			literal: &ast_domain.FloatLiteral{
				Value:            math.Inf(1),
				RelativeLocation: ast_domain.Location{Line: 1, Column: 0},
				SourceLength:     3,
			},
			expected: &ast_domain.FloatLiteral{
				Value:            math.Inf(1),
				RelativeLocation: ast_domain.Location{Line: 1, Column: 0},
				SourceLength:     3,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			original := &ast_domain.TemplateAST{
				RootNodes: []*ast_domain.TemplateNode{
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "div",
						DynamicAttributes: []ast_domain.DynamicAttribute{
							{
								Name:       "ratio",
								Expression: tc.literal,
							},
						},
					},
				},
			}

			decoded := mustRoundTrip(t, original)

			require.Len(t, decoded.RootNodes, 1)
			require.Len(t, decoded.RootNodes[0].DynamicAttributes, 1)

			decodedLit, ok := decoded.RootNodes[0].DynamicAttributes[0].Expression.(*ast_domain.FloatLiteral)
			require.True(t, ok, "expected FloatLiteral type")
			assert.Equal(t, tc.expected.Value, decodedLit.Value)
			assert.Equal(t, tc.expected.RelativeLocation, decodedLit.RelativeLocation)
			assert.Equal(t, tc.expected.SourceLength, decodedLit.SourceLength)
		})
	}
}

func TestEncodeDecodeAST_BooleanLiteral(t *testing.T) {
	testCases := []struct {
		literal  *ast_domain.BooleanLiteral
		expected *ast_domain.BooleanLiteral
		name     string
	}{
		{
			name: "true value",
			literal: &ast_domain.BooleanLiteral{
				Value:            true,
				RelativeLocation: ast_domain.Location{Line: 1, Column: 5},
				SourceLength:     4,
			},
			expected: &ast_domain.BooleanLiteral{
				Value:            true,
				RelativeLocation: ast_domain.Location{Line: 1, Column: 5},
				SourceLength:     4,
			},
		},
		{
			name: "false value",
			literal: &ast_domain.BooleanLiteral{
				Value:            false,
				RelativeLocation: ast_domain.Location{Line: 2, Column: 10},
				SourceLength:     5,
			},
			expected: &ast_domain.BooleanLiteral{
				Value:            false,
				RelativeLocation: ast_domain.Location{Line: 2, Column: 10},
				SourceLength:     5,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			original := &ast_domain.TemplateAST{
				RootNodes: []*ast_domain.TemplateNode{
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "div",
						DynamicAttributes: []ast_domain.DynamicAttribute{
							{
								Name:       "enabled",
								Expression: tc.literal,
							},
						},
					},
				},
			}

			decoded := mustRoundTrip(t, original)

			require.Len(t, decoded.RootNodes, 1)
			require.Len(t, decoded.RootNodes[0].DynamicAttributes, 1)

			decodedLit, ok := decoded.RootNodes[0].DynamicAttributes[0].Expression.(*ast_domain.BooleanLiteral)
			require.True(t, ok, "expected BooleanLiteral type")
			assert.Equal(t, tc.expected.Value, decodedLit.Value)
			assert.Equal(t, tc.expected.RelativeLocation, decodedLit.RelativeLocation)
			assert.Equal(t, tc.expected.SourceLength, decodedLit.SourceLength)
		})
	}
}

func TestEncodeDecodeAST_NilLiteral(t *testing.T) {
	testCases := []struct {
		literal  *ast_domain.NilLiteral
		expected *ast_domain.NilLiteral
		name     string
	}{
		{
			name: "nil literal",
			literal: &ast_domain.NilLiteral{
				RelativeLocation: ast_domain.Location{Line: 1, Column: 5},
				SourceLength:     3,
			},
			expected: &ast_domain.NilLiteral{
				RelativeLocation: ast_domain.Location{Line: 1, Column: 5},
				SourceLength:     3,
			},
		},
		{
			name: "nil literal at start",
			literal: &ast_domain.NilLiteral{
				RelativeLocation: ast_domain.Location{Line: 0, Column: 0},
				SourceLength:     3,
			},
			expected: &ast_domain.NilLiteral{
				RelativeLocation: ast_domain.Location{Line: 0, Column: 0},
				SourceLength:     3,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			original := &ast_domain.TemplateAST{
				RootNodes: []*ast_domain.TemplateNode{
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "div",
						DynamicAttributes: []ast_domain.DynamicAttribute{
							{
								Name:       "data",
								Expression: tc.literal,
							},
						},
					},
				},
			}

			decoded := mustRoundTrip(t, original)

			require.Len(t, decoded.RootNodes, 1)
			require.Len(t, decoded.RootNodes[0].DynamicAttributes, 1)

			decodedLit, ok := decoded.RootNodes[0].DynamicAttributes[0].Expression.(*ast_domain.NilLiteral)
			require.True(t, ok, "expected NilLiteral type")
			assert.Equal(t, tc.expected.RelativeLocation, decodedLit.RelativeLocation)
			assert.Equal(t, tc.expected.SourceLength, decodedLit.SourceLength)
		})
	}
}

func TestEncodeDecodeAST_DecimalLiteral(t *testing.T) {
	testCases := []struct {
		literal  *ast_domain.DecimalLiteral
		expected *ast_domain.DecimalLiteral
		name     string
	}{
		{
			name: "simple decimal",
			literal: &ast_domain.DecimalLiteral{
				Value:            "123.45",
				RelativeLocation: ast_domain.Location{Line: 1, Column: 5},
				SourceLength:     8,
			},
			expected: &ast_domain.DecimalLiteral{
				Value:            "123.45",
				RelativeLocation: ast_domain.Location{Line: 1, Column: 5},
				SourceLength:     8,
			},
		},
		{
			name: "high precision decimal",
			literal: &ast_domain.DecimalLiteral{
				Value:            "3.141592653589793238462643383279502884197",
				RelativeLocation: ast_domain.Location{Line: 2, Column: 0},
				SourceLength:     43,
			},
			expected: &ast_domain.DecimalLiteral{
				Value:            "3.141592653589793238462643383279502884197",
				RelativeLocation: ast_domain.Location{Line: 2, Column: 0},
				SourceLength:     43,
			},
		},
		{
			name: "negative decimal",
			literal: &ast_domain.DecimalLiteral{
				Value:            "-99999.99999",
				RelativeLocation: ast_domain.Location{Line: 3, Column: 10},
				SourceLength:     14,
			},
			expected: &ast_domain.DecimalLiteral{
				Value:            "-99999.99999",
				RelativeLocation: ast_domain.Location{Line: 3, Column: 10},
				SourceLength:     14,
			},
		},
		{
			name: "zero decimal",
			literal: &ast_domain.DecimalLiteral{
				Value:            "0.00",
				RelativeLocation: ast_domain.Location{Line: 0, Column: 0},
				SourceLength:     6,
			},
			expected: &ast_domain.DecimalLiteral{
				Value:            "0.00",
				RelativeLocation: ast_domain.Location{Line: 0, Column: 0},
				SourceLength:     6,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			original := &ast_domain.TemplateAST{
				RootNodes: []*ast_domain.TemplateNode{
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "div",
						DynamicAttributes: []ast_domain.DynamicAttribute{
							{
								Name:       "amount",
								Expression: tc.literal,
							},
						},
					},
				},
			}

			decoded := mustRoundTrip(t, original)

			require.Len(t, decoded.RootNodes, 1)
			require.Len(t, decoded.RootNodes[0].DynamicAttributes, 1)

			decodedLit, ok := decoded.RootNodes[0].DynamicAttributes[0].Expression.(*ast_domain.DecimalLiteral)
			require.True(t, ok, "expected DecimalLiteral type")
			assert.Equal(t, tc.expected.Value, decodedLit.Value)
			assert.Equal(t, tc.expected.RelativeLocation, decodedLit.RelativeLocation)
			assert.Equal(t, tc.expected.SourceLength, decodedLit.SourceLength)
		})
	}
}

func TestEncodeDecodeAST_BigIntLiteral(t *testing.T) {
	testCases := []struct {
		literal  *ast_domain.BigIntLiteral
		expected *ast_domain.BigIntLiteral
		name     string
	}{
		{
			name: "large positive big int",
			literal: &ast_domain.BigIntLiteral{
				Value:            "123456789012345678901234567890",
				RelativeLocation: ast_domain.Location{Line: 1, Column: 5},
				SourceLength:     32,
			},
			expected: &ast_domain.BigIntLiteral{
				Value:            "123456789012345678901234567890",
				RelativeLocation: ast_domain.Location{Line: 1, Column: 5},
				SourceLength:     32,
			},
		},
		{
			name: "large negative big int",
			literal: &ast_domain.BigIntLiteral{
				Value:            "-987654321098765432109876543210",
				RelativeLocation: ast_domain.Location{Line: 2, Column: 0},
				SourceLength:     33,
			},
			expected: &ast_domain.BigIntLiteral{
				Value:            "-987654321098765432109876543210",
				RelativeLocation: ast_domain.Location{Line: 2, Column: 0},
				SourceLength:     33,
			},
		},
		{
			name: "zero big int",
			literal: &ast_domain.BigIntLiteral{
				Value:            "0",
				RelativeLocation: ast_domain.Location{Line: 0, Column: 0},
				SourceLength:     3,
			},
			expected: &ast_domain.BigIntLiteral{
				Value:            "0",
				RelativeLocation: ast_domain.Location{Line: 0, Column: 0},
				SourceLength:     3,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			original := &ast_domain.TemplateAST{
				RootNodes: []*ast_domain.TemplateNode{
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "div",
						DynamicAttributes: []ast_domain.DynamicAttribute{
							{
								Name:       "bigNumber",
								Expression: tc.literal,
							},
						},
					},
				},
			}

			decoded := mustRoundTrip(t, original)

			require.Len(t, decoded.RootNodes, 1)
			require.Len(t, decoded.RootNodes[0].DynamicAttributes, 1)

			decodedLit, ok := decoded.RootNodes[0].DynamicAttributes[0].Expression.(*ast_domain.BigIntLiteral)
			require.True(t, ok, "expected BigIntLiteral type")
			assert.Equal(t, tc.expected.Value, decodedLit.Value)
			assert.Equal(t, tc.expected.RelativeLocation, decodedLit.RelativeLocation)
			assert.Equal(t, tc.expected.SourceLength, decodedLit.SourceLength)
		})
	}
}

func TestEncodeDecodeAST_RuneLiteral(t *testing.T) {
	testCases := []struct {
		literal  *ast_domain.RuneLiteral
		expected *ast_domain.RuneLiteral
		name     string
	}{
		{
			name: "ASCII character",
			literal: &ast_domain.RuneLiteral{
				Value:            'A',
				RelativeLocation: ast_domain.Location{Line: 1, Column: 5},
				SourceLength:     4,
			},
			expected: &ast_domain.RuneLiteral{
				Value:            'A',
				RelativeLocation: ast_domain.Location{Line: 1, Column: 5},
				SourceLength:     4,
			},
		},
		{
			name: "digit character",
			literal: &ast_domain.RuneLiteral{
				Value:            '9',
				RelativeLocation: ast_domain.Location{Line: 2, Column: 0},
				SourceLength:     4,
			},
			expected: &ast_domain.RuneLiteral{
				Value:            '9',
				RelativeLocation: ast_domain.Location{Line: 2, Column: 0},
				SourceLength:     4,
			},
		},
		{
			name: "newline character",
			literal: &ast_domain.RuneLiteral{
				Value:            '\n',
				RelativeLocation: ast_domain.Location{Line: 3, Column: 10},
				SourceLength:     5,
			},
			expected: &ast_domain.RuneLiteral{
				Value:            '\n',
				RelativeLocation: ast_domain.Location{Line: 3, Column: 10},
				SourceLength:     5,
			},
		},
		{
			name: "unicode character",
			literal: &ast_domain.RuneLiteral{
				Value:            '世',
				RelativeLocation: ast_domain.Location{Line: 4, Column: 2},
				SourceLength:     6,
			},
			expected: &ast_domain.RuneLiteral{
				Value:            '世',
				RelativeLocation: ast_domain.Location{Line: 4, Column: 2},
				SourceLength:     6,
			},
		},
		{
			name: "emoji rune",
			literal: &ast_domain.RuneLiteral{
				Value:            '🚀',
				RelativeLocation: ast_domain.Location{Line: 5, Column: 0},
				SourceLength:     7,
			},
			expected: &ast_domain.RuneLiteral{
				Value:            '🚀',
				RelativeLocation: ast_domain.Location{Line: 5, Column: 0},
				SourceLength:     7,
			},
		},
		{
			name: "null character",
			literal: &ast_domain.RuneLiteral{
				Value:            '\x00',
				RelativeLocation: ast_domain.Location{Line: 6, Column: 0},
				SourceLength:     8,
			},
			expected: &ast_domain.RuneLiteral{
				Value:            '\x00',
				RelativeLocation: ast_domain.Location{Line: 6, Column: 0},
				SourceLength:     8,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			original := &ast_domain.TemplateAST{
				RootNodes: []*ast_domain.TemplateNode{
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "div",
						DynamicAttributes: []ast_domain.DynamicAttribute{
							{
								Name:       "char",
								Expression: tc.literal,
							},
						},
					},
				},
			}

			decoded := mustRoundTrip(t, original)

			require.Len(t, decoded.RootNodes, 1)
			require.Len(t, decoded.RootNodes[0].DynamicAttributes, 1)

			decodedLit, ok := decoded.RootNodes[0].DynamicAttributes[0].Expression.(*ast_domain.RuneLiteral)
			require.True(t, ok, "expected RuneLiteral type")
			assert.Equal(t, tc.expected.Value, decodedLit.Value)
			assert.Equal(t, tc.expected.RelativeLocation, decodedLit.RelativeLocation)
			assert.Equal(t, tc.expected.SourceLength, decodedLit.SourceLength)
		})
	}
}

func TestEncodeDecodeAST_DateTimeLiteral(t *testing.T) {
	testCases := []struct {
		literal  *ast_domain.DateTimeLiteral
		expected *ast_domain.DateTimeLiteral
		name     string
	}{
		{
			name: "RFC3339 datetime",
			literal: &ast_domain.DateTimeLiteral{
				Value:            "2024-01-15T14:30:00Z",
				RelativeLocation: ast_domain.Location{Line: 1, Column: 5},
				SourceLength:     24,
			},
			expected: &ast_domain.DateTimeLiteral{
				Value:            "2024-01-15T14:30:00Z",
				RelativeLocation: ast_domain.Location{Line: 1, Column: 5},
				SourceLength:     24,
			},
		},
		{
			name: "datetime with timezone offset",
			literal: &ast_domain.DateTimeLiteral{
				Value:            "2024-06-20T09:15:30+05:30",
				RelativeLocation: ast_domain.Location{Line: 2, Column: 0},
				SourceLength:     29,
			},
			expected: &ast_domain.DateTimeLiteral{
				Value:            "2024-06-20T09:15:30+05:30",
				RelativeLocation: ast_domain.Location{Line: 2, Column: 0},
				SourceLength:     29,
			},
		},
		{
			name: "datetime with milliseconds",
			literal: &ast_domain.DateTimeLiteral{
				Value:            "2024-12-31T23:59:59.999Z",
				RelativeLocation: ast_domain.Location{Line: 3, Column: 10},
				SourceLength:     28,
			},
			expected: &ast_domain.DateTimeLiteral{
				Value:            "2024-12-31T23:59:59.999Z",
				RelativeLocation: ast_domain.Location{Line: 3, Column: 10},
				SourceLength:     28,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			original := &ast_domain.TemplateAST{
				RootNodes: []*ast_domain.TemplateNode{
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "div",
						DynamicAttributes: []ast_domain.DynamicAttribute{
							{
								Name:       "timestamp",
								Expression: tc.literal,
							},
						},
					},
				},
			}

			decoded := mustRoundTrip(t, original)

			require.Len(t, decoded.RootNodes, 1)
			require.Len(t, decoded.RootNodes[0].DynamicAttributes, 1)

			decodedLit, ok := decoded.RootNodes[0].DynamicAttributes[0].Expression.(*ast_domain.DateTimeLiteral)
			require.True(t, ok, "expected DateTimeLiteral type")
			assert.Equal(t, tc.expected.Value, decodedLit.Value)
			assert.Equal(t, tc.expected.RelativeLocation, decodedLit.RelativeLocation)
			assert.Equal(t, tc.expected.SourceLength, decodedLit.SourceLength)
		})
	}
}

func TestEncodeDecodeAST_DateLiteral(t *testing.T) {
	testCases := []struct {
		literal  *ast_domain.DateLiteral
		expected *ast_domain.DateLiteral
		name     string
	}{
		{
			name: "simple date",
			literal: &ast_domain.DateLiteral{
				Value:            "2024-01-15",
				RelativeLocation: ast_domain.Location{Line: 1, Column: 5},
				SourceLength:     14,
			},
			expected: &ast_domain.DateLiteral{
				Value:            "2024-01-15",
				RelativeLocation: ast_domain.Location{Line: 1, Column: 5},
				SourceLength:     14,
			},
		},
		{
			name: "end of year date",
			literal: &ast_domain.DateLiteral{
				Value:            "2024-12-31",
				RelativeLocation: ast_domain.Location{Line: 2, Column: 0},
				SourceLength:     14,
			},
			expected: &ast_domain.DateLiteral{
				Value:            "2024-12-31",
				RelativeLocation: ast_domain.Location{Line: 2, Column: 0},
				SourceLength:     14,
			},
		},
		{
			name: "leap year date",
			literal: &ast_domain.DateLiteral{
				Value:            "2024-02-29",
				RelativeLocation: ast_domain.Location{Line: 3, Column: 10},
				SourceLength:     14,
			},
			expected: &ast_domain.DateLiteral{
				Value:            "2024-02-29",
				RelativeLocation: ast_domain.Location{Line: 3, Column: 10},
				SourceLength:     14,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			original := &ast_domain.TemplateAST{
				RootNodes: []*ast_domain.TemplateNode{
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "div",
						DynamicAttributes: []ast_domain.DynamicAttribute{
							{
								Name:       "date",
								Expression: tc.literal,
							},
						},
					},
				},
			}

			decoded := mustRoundTrip(t, original)

			require.Len(t, decoded.RootNodes, 1)
			require.Len(t, decoded.RootNodes[0].DynamicAttributes, 1)

			decodedLit, ok := decoded.RootNodes[0].DynamicAttributes[0].Expression.(*ast_domain.DateLiteral)
			require.True(t, ok, "expected DateLiteral type")
			assert.Equal(t, tc.expected.Value, decodedLit.Value)
			assert.Equal(t, tc.expected.RelativeLocation, decodedLit.RelativeLocation)
			assert.Equal(t, tc.expected.SourceLength, decodedLit.SourceLength)
		})
	}
}

func TestEncodeDecodeAST_TimeLiteral(t *testing.T) {
	testCases := []struct {
		literal  *ast_domain.TimeLiteral
		expected *ast_domain.TimeLiteral
		name     string
	}{
		{
			name: "simple time",
			literal: &ast_domain.TimeLiteral{
				Value:            "14:30:00",
				RelativeLocation: ast_domain.Location{Line: 1, Column: 5},
				SourceLength:     12,
			},
			expected: &ast_domain.TimeLiteral{
				Value:            "14:30:00",
				RelativeLocation: ast_domain.Location{Line: 1, Column: 5},
				SourceLength:     12,
			},
		},
		{
			name: "midnight",
			literal: &ast_domain.TimeLiteral{
				Value:            "00:00:00",
				RelativeLocation: ast_domain.Location{Line: 2, Column: 0},
				SourceLength:     12,
			},
			expected: &ast_domain.TimeLiteral{
				Value:            "00:00:00",
				RelativeLocation: ast_domain.Location{Line: 2, Column: 0},
				SourceLength:     12,
			},
		},
		{
			name: "time with milliseconds",
			literal: &ast_domain.TimeLiteral{
				Value:            "23:59:59.999",
				RelativeLocation: ast_domain.Location{Line: 3, Column: 10},
				SourceLength:     16,
			},
			expected: &ast_domain.TimeLiteral{
				Value:            "23:59:59.999",
				RelativeLocation: ast_domain.Location{Line: 3, Column: 10},
				SourceLength:     16,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			original := &ast_domain.TemplateAST{
				RootNodes: []*ast_domain.TemplateNode{
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "div",
						DynamicAttributes: []ast_domain.DynamicAttribute{
							{
								Name:       "time",
								Expression: tc.literal,
							},
						},
					},
				},
			}

			decoded := mustRoundTrip(t, original)

			require.Len(t, decoded.RootNodes, 1)
			require.Len(t, decoded.RootNodes[0].DynamicAttributes, 1)

			decodedLit, ok := decoded.RootNodes[0].DynamicAttributes[0].Expression.(*ast_domain.TimeLiteral)
			require.True(t, ok, "expected TimeLiteral type")
			assert.Equal(t, tc.expected.Value, decodedLit.Value)
			assert.Equal(t, tc.expected.RelativeLocation, decodedLit.RelativeLocation)
			assert.Equal(t, tc.expected.SourceLength, decodedLit.SourceLength)
		})
	}
}

func TestEncodeDecodeAST_DurationLiteral(t *testing.T) {
	testCases := []struct {
		literal  *ast_domain.DurationLiteral
		expected *ast_domain.DurationLiteral
		name     string
	}{
		{
			name: "hours duration",
			literal: &ast_domain.DurationLiteral{
				Value:            "2h",
				RelativeLocation: ast_domain.Location{Line: 1, Column: 5},
				SourceLength:     6,
			},
			expected: &ast_domain.DurationLiteral{
				Value:            "2h",
				RelativeLocation: ast_domain.Location{Line: 1, Column: 5},
				SourceLength:     6,
			},
		},
		{
			name: "minutes duration",
			literal: &ast_domain.DurationLiteral{
				Value:            "30m",
				RelativeLocation: ast_domain.Location{Line: 2, Column: 0},
				SourceLength:     7,
			},
			expected: &ast_domain.DurationLiteral{
				Value:            "30m",
				RelativeLocation: ast_domain.Location{Line: 2, Column: 0},
				SourceLength:     7,
			},
		},
		{
			name: "complex duration",
			literal: &ast_domain.DurationLiteral{
				Value:            "1h30m45s",
				RelativeLocation: ast_domain.Location{Line: 3, Column: 10},
				SourceLength:     12,
			},
			expected: &ast_domain.DurationLiteral{
				Value:            "1h30m45s",
				RelativeLocation: ast_domain.Location{Line: 3, Column: 10},
				SourceLength:     12,
			},
		},
		{
			name: "milliseconds duration",
			literal: &ast_domain.DurationLiteral{
				Value:            "500ms",
				RelativeLocation: ast_domain.Location{Line: 4, Column: 2},
				SourceLength:     9,
			},
			expected: &ast_domain.DurationLiteral{
				Value:            "500ms",
				RelativeLocation: ast_domain.Location{Line: 4, Column: 2},
				SourceLength:     9,
			},
		},
		{
			name: "nanoseconds duration",
			literal: &ast_domain.DurationLiteral{
				Value:            "100ns",
				RelativeLocation: ast_domain.Location{Line: 5, Column: 0},
				SourceLength:     9,
			},
			expected: &ast_domain.DurationLiteral{
				Value:            "100ns",
				RelativeLocation: ast_domain.Location{Line: 5, Column: 0},
				SourceLength:     9,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			original := &ast_domain.TemplateAST{
				RootNodes: []*ast_domain.TemplateNode{
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "div",
						DynamicAttributes: []ast_domain.DynamicAttribute{
							{
								Name:       "timeout",
								Expression: tc.literal,
							},
						},
					},
				},
			}

			decoded := mustRoundTrip(t, original)

			require.Len(t, decoded.RootNodes, 1)
			require.Len(t, decoded.RootNodes[0].DynamicAttributes, 1)

			decodedLit, ok := decoded.RootNodes[0].DynamicAttributes[0].Expression.(*ast_domain.DurationLiteral)
			require.True(t, ok, "expected DurationLiteral type")
			assert.Equal(t, tc.expected.Value, decodedLit.Value)
			assert.Equal(t, tc.expected.RelativeLocation, decodedLit.RelativeLocation)
			assert.Equal(t, tc.expected.SourceLength, decodedLit.SourceLength)
		})
	}
}

func TestEncodeDecodeAST_ArrayLiteral(t *testing.T) {
	t.Run("empty array", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					DynamicAttributes: []ast_domain.DynamicAttribute{
						{
							Name: "items",
							Expression: &ast_domain.ArrayLiteral{
								Elements:         []ast_domain.Expression{},
								RelativeLocation: ast_domain.Location{Line: 1, Column: 5},
								SourceLength:     2,
							},
						},
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		require.Len(t, decoded.RootNodes, 1)
		require.Len(t, decoded.RootNodes[0].DynamicAttributes, 1)

		decodedLit, ok := decoded.RootNodes[0].DynamicAttributes[0].Expression.(*ast_domain.ArrayLiteral)
		require.True(t, ok, "expected ArrayLiteral type")
		assert.Empty(t, decodedLit.Elements)
		assert.Equal(t, ast_domain.Location{Line: 1, Column: 5}, decodedLit.RelativeLocation)
		assert.Equal(t, 2, decodedLit.SourceLength)
	})

	t.Run("array with single element", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					DynamicAttributes: []ast_domain.DynamicAttribute{
						{
							Name: "items",
							Expression: &ast_domain.ArrayLiteral{
								Elements: []ast_domain.Expression{
									&ast_domain.IntegerLiteral{
										Value:            42,
										RelativeLocation: ast_domain.Location{Line: 1, Column: 6},
										SourceLength:     2,
									},
								},
								RelativeLocation: ast_domain.Location{Line: 1, Column: 5},
								SourceLength:     4,
							},
						},
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		decodedLit, ok := decoded.RootNodes[0].DynamicAttributes[0].Expression.(*ast_domain.ArrayLiteral)
		require.True(t, ok, "expected ArrayLiteral type")
		require.Len(t, decodedLit.Elements, 1)

		intLit, ok := decodedLit.Elements[0].(*ast_domain.IntegerLiteral)
		require.True(t, ok, "expected IntegerLiteral element")
		assert.Equal(t, int64(42), intLit.Value)
	})

	t.Run("array with multiple mixed elements", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					DynamicAttributes: []ast_domain.DynamicAttribute{
						{
							Name: "items",
							Expression: &ast_domain.ArrayLiteral{
								Elements: []ast_domain.Expression{
									&ast_domain.IntegerLiteral{
										Value:            1,
										RelativeLocation: ast_domain.Location{Line: 1, Column: 6},
										SourceLength:     1,
									},
									&ast_domain.StringLiteral{
										Value:            "hello",
										RelativeLocation: ast_domain.Location{Line: 1, Column: 9},
										SourceLength:     7,
									},
									&ast_domain.BooleanLiteral{
										Value:            true,
										RelativeLocation: ast_domain.Location{Line: 1, Column: 18},
										SourceLength:     4,
									},
								},
								RelativeLocation: ast_domain.Location{Line: 1, Column: 5},
								SourceLength:     18,
							},
						},
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		decodedLit, ok := decoded.RootNodes[0].DynamicAttributes[0].Expression.(*ast_domain.ArrayLiteral)
		require.True(t, ok, "expected ArrayLiteral type")
		require.Len(t, decodedLit.Elements, 3)

		intLit, ok := decodedLit.Elements[0].(*ast_domain.IntegerLiteral)
		require.True(t, ok, "expected IntegerLiteral")
		assert.Equal(t, int64(1), intLit.Value)

		strLit, ok := decodedLit.Elements[1].(*ast_domain.StringLiteral)
		require.True(t, ok, "expected StringLiteral")
		assert.Equal(t, "hello", strLit.Value)

		boolLit, ok := decodedLit.Elements[2].(*ast_domain.BooleanLiteral)
		require.True(t, ok, "expected BooleanLiteral")
		assert.True(t, boolLit.Value)
	})

	t.Run("nested arrays", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					DynamicAttributes: []ast_domain.DynamicAttribute{
						{
							Name: "matrix",
							Expression: &ast_domain.ArrayLiteral{
								Elements: []ast_domain.Expression{
									&ast_domain.ArrayLiteral{
										Elements: []ast_domain.Expression{
											&ast_domain.IntegerLiteral{Value: 1},
											&ast_domain.IntegerLiteral{Value: 2},
										},
										RelativeLocation: ast_domain.Location{Line: 1, Column: 6},
									},
									&ast_domain.ArrayLiteral{
										Elements: []ast_domain.Expression{
											&ast_domain.IntegerLiteral{Value: 3},
											&ast_domain.IntegerLiteral{Value: 4},
										},
										RelativeLocation: ast_domain.Location{Line: 1, Column: 14},
									},
								},
								RelativeLocation: ast_domain.Location{Line: 1, Column: 5},
							},
						},
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		decodedLit, ok := decoded.RootNodes[0].DynamicAttributes[0].Expression.(*ast_domain.ArrayLiteral)
		require.True(t, ok)
		require.Len(t, decodedLit.Elements, 2)

		row1, ok := decodedLit.Elements[0].(*ast_domain.ArrayLiteral)
		require.True(t, ok)
		require.Len(t, row1.Elements, 2)

		row2, ok := decodedLit.Elements[1].(*ast_domain.ArrayLiteral)
		require.True(t, ok)
		require.Len(t, row2.Elements, 2)
	})
}

func TestEncodeDecodeAST_ObjectLiteral(t *testing.T) {
	t.Run("empty object", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					DynamicAttributes: []ast_domain.DynamicAttribute{
						{
							Name: "data",
							Expression: &ast_domain.ObjectLiteral{
								Pairs:            map[string]ast_domain.Expression{},
								RelativeLocation: ast_domain.Location{Line: 1, Column: 5},
								SourceLength:     2,
							},
						},
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		decodedLit, ok := decoded.RootNodes[0].DynamicAttributes[0].Expression.(*ast_domain.ObjectLiteral)
		require.True(t, ok, "expected ObjectLiteral type")
		assert.Empty(t, decodedLit.Pairs)
		assert.Equal(t, ast_domain.Location{Line: 1, Column: 5}, decodedLit.RelativeLocation)
	})

	t.Run("object with single pair", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					DynamicAttributes: []ast_domain.DynamicAttribute{
						{
							Name: "data",
							Expression: &ast_domain.ObjectLiteral{
								Pairs: map[string]ast_domain.Expression{
									"name": &ast_domain.StringLiteral{
										Value:            "Alice",
										RelativeLocation: ast_domain.Location{Line: 1, Column: 12},
										SourceLength:     7,
									},
								},
								RelativeLocation: ast_domain.Location{Line: 1, Column: 5},
								SourceLength:     20,
							},
						},
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		decodedLit, ok := decoded.RootNodes[0].DynamicAttributes[0].Expression.(*ast_domain.ObjectLiteral)
		require.True(t, ok, "expected ObjectLiteral type")
		require.Len(t, decodedLit.Pairs, 1)
		require.Contains(t, decodedLit.Pairs, "name")

		nameLit, ok := decodedLit.Pairs["name"].(*ast_domain.StringLiteral)
		require.True(t, ok)
		assert.Equal(t, "Alice", nameLit.Value)
	})

	t.Run("object with multiple pairs", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					DynamicAttributes: []ast_domain.DynamicAttribute{
						{
							Name: "user",
							Expression: &ast_domain.ObjectLiteral{
								Pairs: map[string]ast_domain.Expression{
									"name": &ast_domain.StringLiteral{
										Value:            "Bob",
										RelativeLocation: ast_domain.Location{Line: 1, Column: 12},
									},
									"age": &ast_domain.IntegerLiteral{
										Value:            30,
										RelativeLocation: ast_domain.Location{Line: 1, Column: 25},
									},
									"active": &ast_domain.BooleanLiteral{
										Value:            true,
										RelativeLocation: ast_domain.Location{Line: 1, Column: 40},
									},
								},
								RelativeLocation: ast_domain.Location{Line: 1, Column: 5},
							},
						},
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		decodedLit, ok := decoded.RootNodes[0].DynamicAttributes[0].Expression.(*ast_domain.ObjectLiteral)
		require.True(t, ok)
		require.Len(t, decodedLit.Pairs, 3)

		nameLit, ok := decodedLit.Pairs["name"].(*ast_domain.StringLiteral)
		require.True(t, ok)
		assert.Equal(t, "Bob", nameLit.Value)

		ageLit, ok := decodedLit.Pairs["age"].(*ast_domain.IntegerLiteral)
		require.True(t, ok)
		assert.Equal(t, int64(30), ageLit.Value)

		activeLit, ok := decodedLit.Pairs["active"].(*ast_domain.BooleanLiteral)
		require.True(t, ok)
		assert.True(t, activeLit.Value)
	})

	t.Run("nested objects", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					DynamicAttributes: []ast_domain.DynamicAttribute{
						{
							Name: "config",
							Expression: &ast_domain.ObjectLiteral{
								Pairs: map[string]ast_domain.Expression{
									"server": &ast_domain.ObjectLiteral{
										Pairs: map[string]ast_domain.Expression{
											"host": &ast_domain.StringLiteral{Value: "localhost"},
											"port": &ast_domain.IntegerLiteral{Value: 8080},
										},
									},
									"database": &ast_domain.ObjectLiteral{
										Pairs: map[string]ast_domain.Expression{
											"name": &ast_domain.StringLiteral{Value: "mydb"},
										},
									},
								},
								RelativeLocation: ast_domain.Location{Line: 1, Column: 5},
							},
						},
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		configLit, ok := decoded.RootNodes[0].DynamicAttributes[0].Expression.(*ast_domain.ObjectLiteral)
		require.True(t, ok)
		require.Len(t, configLit.Pairs, 2)

		serverLit, ok := configLit.Pairs["server"].(*ast_domain.ObjectLiteral)
		require.True(t, ok)
		require.Contains(t, serverLit.Pairs, "host")
		require.Contains(t, serverLit.Pairs, "port")

		dbLit, ok := configLit.Pairs["database"].(*ast_domain.ObjectLiteral)
		require.True(t, ok)
		require.Contains(t, dbLit.Pairs, "name")
	})

	t.Run("object with special key names", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					DynamicAttributes: []ast_domain.DynamicAttribute{
						{
							Name: "data",
							Expression: &ast_domain.ObjectLiteral{
								Pairs: map[string]ast_domain.Expression{
									"key-with-dash":        &ast_domain.IntegerLiteral{Value: 1},
									"key.with.dots":        &ast_domain.IntegerLiteral{Value: 2},
									"key_with_underscores": &ast_domain.IntegerLiteral{Value: 3},
									"123":                  &ast_domain.IntegerLiteral{Value: 4},
								},
								RelativeLocation: ast_domain.Location{Line: 1, Column: 5},
							},
						},
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		decodedLit, ok := decoded.RootNodes[0].DynamicAttributes[0].Expression.(*ast_domain.ObjectLiteral)
		require.True(t, ok)
		require.Len(t, decodedLit.Pairs, 4)
		require.Contains(t, decodedLit.Pairs, "key-with-dash")
		require.Contains(t, decodedLit.Pairs, "key.with.dots")
		require.Contains(t, decodedLit.Pairs, "key_with_underscores")
		require.Contains(t, decodedLit.Pairs, "123")
	})
}

func TestEncodeDecodeAST_TemplateLiteral(t *testing.T) {
	t.Run("simple string template", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					DynamicAttributes: []ast_domain.DynamicAttribute{
						{
							Name: "text",
							Expression: &ast_domain.TemplateLiteral{
								Parts: []ast_domain.TemplateLiteralPart{
									{
										IsLiteral:        true,
										Literal:          "Hello, World!",
										RelativeLocation: ast_domain.Location{Line: 1, Column: 6},
									},
								},
								RelativeLocation: ast_domain.Location{Line: 1, Column: 5},
								SourceLength:     15,
							},
						},
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		decodedLit, ok := decoded.RootNodes[0].DynamicAttributes[0].Expression.(*ast_domain.TemplateLiteral)
		require.True(t, ok, "expected TemplateLiteral type")
		require.Len(t, decodedLit.Parts, 1)
		assert.True(t, decodedLit.Parts[0].IsLiteral)
		assert.Equal(t, "Hello, World!", decodedLit.Parts[0].Literal)
		assert.Equal(t, ast_domain.Location{Line: 1, Column: 5}, decodedLit.RelativeLocation)
	})

	t.Run("template with interpolation", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					DynamicAttributes: []ast_domain.DynamicAttribute{
						{
							Name: "greeting",
							Expression: &ast_domain.TemplateLiteral{
								Parts: []ast_domain.TemplateLiteralPart{
									{
										IsLiteral:        true,
										Literal:          "Hello, ",
										RelativeLocation: ast_domain.Location{Line: 1, Column: 6},
									},
									{
										IsLiteral: false,
										Expression: &ast_domain.Identifier{
											Name:             "name",
											RelativeLocation: ast_domain.Location{Line: 1, Column: 15},
											SourceLength:     4,
										},
										RelativeLocation: ast_domain.Location{Line: 1, Column: 13},
									},
									{
										IsLiteral:        true,
										Literal:          "!",
										RelativeLocation: ast_domain.Location{Line: 1, Column: 20},
									},
								},
								RelativeLocation: ast_domain.Location{Line: 1, Column: 5},
								SourceLength:     17,
							},
						},
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		decodedLit, ok := decoded.RootNodes[0].DynamicAttributes[0].Expression.(*ast_domain.TemplateLiteral)
		require.True(t, ok, "expected TemplateLiteral type")
		require.Len(t, decodedLit.Parts, 3)

		assert.True(t, decodedLit.Parts[0].IsLiteral)
		assert.Equal(t, "Hello, ", decodedLit.Parts[0].Literal)

		assert.False(t, decodedLit.Parts[1].IsLiteral)
		identifier, ok := decodedLit.Parts[1].Expression.(*ast_domain.Identifier)
		require.True(t, ok)
		assert.Equal(t, "name", identifier.Name)

		assert.True(t, decodedLit.Parts[2].IsLiteral)
		assert.Equal(t, "!", decodedLit.Parts[2].Literal)
	})

	t.Run("template with multiple interpolations", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					DynamicAttributes: []ast_domain.DynamicAttribute{
						{
							Name: "message",
							Expression: &ast_domain.TemplateLiteral{
								Parts: []ast_domain.TemplateLiteralPart{
									{
										IsLiteral: true,
										Literal:   "User ",
									},
									{
										IsLiteral:  false,
										Expression: &ast_domain.Identifier{Name: "name"},
									},
									{
										IsLiteral: true,
										Literal:   " has ",
									},
									{
										IsLiteral:  false,
										Expression: &ast_domain.Identifier{Name: "count"},
									},
									{
										IsLiteral: true,
										Literal:   " items",
									},
								},
								RelativeLocation: ast_domain.Location{Line: 1, Column: 5},
							},
						},
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		decodedLit, ok := decoded.RootNodes[0].DynamicAttributes[0].Expression.(*ast_domain.TemplateLiteral)
		require.True(t, ok)
		require.Len(t, decodedLit.Parts, 5)

		assert.Equal(t, "User ", decodedLit.Parts[0].Literal)
		assert.Equal(t, "name", decodedLit.Parts[1].Expression.(*ast_domain.Identifier).Name)
		assert.Equal(t, " has ", decodedLit.Parts[2].Literal)
		assert.Equal(t, "count", decodedLit.Parts[3].Expression.(*ast_domain.Identifier).Name)
		assert.Equal(t, " items", decodedLit.Parts[4].Literal)
	})

	t.Run("template with complex expression", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					DynamicAttributes: []ast_domain.DynamicAttribute{
						{
							Name: "result",
							Expression: &ast_domain.TemplateLiteral{
								Parts: []ast_domain.TemplateLiteralPart{
									{
										IsLiteral: true,
										Literal:   "Total: ",
									},
									{
										IsLiteral: false,
										Expression: &ast_domain.BinaryExpression{
											Operator: ast_domain.OpMul,
											Left:     &ast_domain.Identifier{Name: "price"},
											Right:    &ast_domain.Identifier{Name: "quantity"},
										},
									},
								},
								RelativeLocation: ast_domain.Location{Line: 1, Column: 5},
							},
						},
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		decodedLit, ok := decoded.RootNodes[0].DynamicAttributes[0].Expression.(*ast_domain.TemplateLiteral)
		require.True(t, ok)
		require.Len(t, decodedLit.Parts, 2)

		assert.Equal(t, "Total: ", decodedLit.Parts[0].Literal)

		binExpr, ok := decodedLit.Parts[1].Expression.(*ast_domain.BinaryExpression)
		require.True(t, ok)
		assert.Equal(t, ast_domain.OpMul, binExpr.Operator)
	})

	t.Run("empty template", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					DynamicAttributes: []ast_domain.DynamicAttribute{
						{
							Name: "empty",
							Expression: &ast_domain.TemplateLiteral{
								Parts:            []ast_domain.TemplateLiteralPart{},
								RelativeLocation: ast_domain.Location{Line: 1, Column: 5},
								SourceLength:     2,
							},
						},
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		decodedLit, ok := decoded.RootNodes[0].DynamicAttributes[0].Expression.(*ast_domain.TemplateLiteral)
		require.True(t, ok)
		assert.Empty(t, decodedLit.Parts)
	})
}

func TestEncodeDecodeAST_CombinedLiterals(t *testing.T) {
	t.Run("all literal types in single AST", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					DynamicAttributes: []ast_domain.DynamicAttribute{
						{Name: "str", Expression: &ast_domain.StringLiteral{Value: "hello"}},
						{Name: "int", Expression: &ast_domain.IntegerLiteral{Value: 42}},
						{Name: "float", Expression: &ast_domain.FloatLiteral{Value: 3.14}},
						{Name: "bool", Expression: &ast_domain.BooleanLiteral{Value: true}},
						{Name: "nil", Expression: &ast_domain.NilLiteral{}},
						{Name: "decimal", Expression: &ast_domain.DecimalLiteral{Value: "99.99"}},
						{Name: "bigint", Expression: &ast_domain.BigIntLiteral{Value: "123456789012345678901234567890"}},
						{Name: "rune", Expression: &ast_domain.RuneLiteral{Value: 'A'}},
						{Name: "datetime", Expression: &ast_domain.DateTimeLiteral{Value: "2024-01-15T14:30:00Z"}},
						{Name: "date", Expression: &ast_domain.DateLiteral{Value: "2024-01-15"}},
						{Name: "time", Expression: &ast_domain.TimeLiteral{Value: "14:30:00"}},
						{Name: "duration", Expression: &ast_domain.DurationLiteral{Value: "2h30m"}},
						{Name: "ident", Expression: &ast_domain.Identifier{Name: "myVar"}},
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		require.Len(t, decoded.RootNodes, 1)
		attrs := decoded.RootNodes[0].DynamicAttributes

		attributeMap := make(map[string]ast_domain.Expression)
		for _, attr := range attrs {
			attributeMap[attr.Name] = attr.Expression
		}

		strLit, ok := attributeMap["str"].(*ast_domain.StringLiteral)
		require.True(t, ok)
		assert.Equal(t, "hello", strLit.Value)

		intLit, ok := attributeMap["int"].(*ast_domain.IntegerLiteral)
		require.True(t, ok)
		assert.Equal(t, int64(42), intLit.Value)

		floatLit, ok := attributeMap["float"].(*ast_domain.FloatLiteral)
		require.True(t, ok)
		assert.Equal(t, 3.14, floatLit.Value)

		boolLit, ok := attributeMap["bool"].(*ast_domain.BooleanLiteral)
		require.True(t, ok)
		assert.True(t, boolLit.Value)

		_, ok = attributeMap["nil"].(*ast_domain.NilLiteral)
		require.True(t, ok)

		decLit, ok := attributeMap["decimal"].(*ast_domain.DecimalLiteral)
		require.True(t, ok)
		assert.Equal(t, "99.99", decLit.Value)

		bigLit, ok := attributeMap["bigint"].(*ast_domain.BigIntLiteral)
		require.True(t, ok)
		assert.Equal(t, "123456789012345678901234567890", bigLit.Value)

		runeLit, ok := attributeMap["rune"].(*ast_domain.RuneLiteral)
		require.True(t, ok)
		assert.Equal(t, 'A', runeLit.Value)

		dtLit, ok := attributeMap["datetime"].(*ast_domain.DateTimeLiteral)
		require.True(t, ok)
		assert.Equal(t, "2024-01-15T14:30:00Z", dtLit.Value)

		dateLit, ok := attributeMap["date"].(*ast_domain.DateLiteral)
		require.True(t, ok)
		assert.Equal(t, "2024-01-15", dateLit.Value)

		timeLit, ok := attributeMap["time"].(*ast_domain.TimeLiteral)
		require.True(t, ok)
		assert.Equal(t, "14:30:00", timeLit.Value)

		durLit, ok := attributeMap["duration"].(*ast_domain.DurationLiteral)
		require.True(t, ok)
		assert.Equal(t, "2h30m", durLit.Value)

		identLit, ok := attributeMap["ident"].(*ast_domain.Identifier)
		require.True(t, ok)
		assert.Equal(t, "myVar", identLit.Name)
	})

	t.Run("array containing all literal types", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					DynamicAttributes: []ast_domain.DynamicAttribute{
						{
							Name: "mixed",
							Expression: &ast_domain.ArrayLiteral{
								Elements: []ast_domain.Expression{
									&ast_domain.StringLiteral{Value: "text"},
									&ast_domain.IntegerLiteral{Value: 123},
									&ast_domain.FloatLiteral{Value: 1.5},
									&ast_domain.BooleanLiteral{Value: false},
									&ast_domain.NilLiteral{},
									&ast_domain.Identifier{Name: "x"},
								},
							},
						},
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		arrLit, ok := decoded.RootNodes[0].DynamicAttributes[0].Expression.(*ast_domain.ArrayLiteral)
		require.True(t, ok)
		require.Len(t, arrLit.Elements, 6)

		_, ok = arrLit.Elements[0].(*ast_domain.StringLiteral)
		require.True(t, ok)

		_, ok = arrLit.Elements[1].(*ast_domain.IntegerLiteral)
		require.True(t, ok)

		_, ok = arrLit.Elements[2].(*ast_domain.FloatLiteral)
		require.True(t, ok)

		_, ok = arrLit.Elements[3].(*ast_domain.BooleanLiteral)
		require.True(t, ok)

		_, ok = arrLit.Elements[4].(*ast_domain.NilLiteral)
		require.True(t, ok)

		_, ok = arrLit.Elements[5].(*ast_domain.Identifier)
		require.True(t, ok)
	})

	t.Run("deeply nested structure", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					DynamicAttributes: []ast_domain.DynamicAttribute{
						{
							Name: "nested",
							Expression: &ast_domain.ObjectLiteral{
								Pairs: map[string]ast_domain.Expression{
									"level1": &ast_domain.ObjectLiteral{
										Pairs: map[string]ast_domain.Expression{
											"level2": &ast_domain.ArrayLiteral{
												Elements: []ast_domain.Expression{
													&ast_domain.ObjectLiteral{
														Pairs: map[string]ast_domain.Expression{
															"level3": &ast_domain.StringLiteral{Value: "deep"},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		obj1, ok := decoded.RootNodes[0].DynamicAttributes[0].Expression.(*ast_domain.ObjectLiteral)
		require.True(t, ok)

		obj2, ok := obj1.Pairs["level1"].(*ast_domain.ObjectLiteral)
		require.True(t, ok)

		arr, ok := obj2.Pairs["level2"].(*ast_domain.ArrayLiteral)
		require.True(t, ok)
		require.Len(t, arr.Elements, 1)

		obj3, ok := arr.Elements[0].(*ast_domain.ObjectLiteral)
		require.True(t, ok)

		strLit, ok := obj3.Pairs["level3"].(*ast_domain.StringLiteral)
		require.True(t, ok)
		assert.Equal(t, "deep", strLit.Value)
	})
}

func TestEncodeDecodeAST_LiteralLocationPreservation(t *testing.T) {
	testCases := []struct {
		expression       ast_domain.Expression
		name             string
		expectedLocation ast_domain.Location
	}{
		{
			name: "string literal location",
			expression: &ast_domain.StringLiteral{
				Value:            "test",
				RelativeLocation: ast_domain.Location{Line: 10, Column: 25},
				SourceLength:     6,
			},
			expectedLocation: ast_domain.Location{Line: 10, Column: 25},
		},
		{
			name: "integer literal location",
			expression: &ast_domain.IntegerLiteral{
				Value:            42,
				RelativeLocation: ast_domain.Location{Line: 100, Column: 0},
				SourceLength:     2,
			},
			expectedLocation: ast_domain.Location{Line: 100, Column: 0},
		},
		{
			name: "float literal location",
			expression: &ast_domain.FloatLiteral{
				Value:            3.14,
				RelativeLocation: ast_domain.Location{Line: 5, Column: 50},
				SourceLength:     4,
			},
			expectedLocation: ast_domain.Location{Line: 5, Column: 50},
		},
		{
			name: "array literal location",
			expression: &ast_domain.ArrayLiteral{
				Elements:         []ast_domain.Expression{&ast_domain.IntegerLiteral{Value: 1}},
				RelativeLocation: ast_domain.Location{Line: 20, Column: 15},
				SourceLength:     3,
			},
			expectedLocation: ast_domain.Location{Line: 20, Column: 15},
		},
		{
			name: "object literal location",
			expression: &ast_domain.ObjectLiteral{
				Pairs:            map[string]ast_domain.Expression{"a": &ast_domain.IntegerLiteral{Value: 1}},
				RelativeLocation: ast_domain.Location{Line: 30, Column: 8},
				SourceLength:     7,
			},
			expectedLocation: ast_domain.Location{Line: 30, Column: 8},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			original := &ast_domain.TemplateAST{
				RootNodes: []*ast_domain.TemplateNode{
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "div",
						DynamicAttributes: []ast_domain.DynamicAttribute{
							{
								Name:       "attr",
								Expression: tc.expression,
							},
						},
					},
				},
			}

			decoded := mustRoundTrip(t, original)

			decodedExpr := decoded.RootNodes[0].DynamicAttributes[0].Expression
			assert.Equal(t, tc.expectedLocation, decodedExpr.GetRelativeLocation())
		})
	}
}
