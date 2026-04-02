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

package compiler_domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/esbuild/ast"
	"piko.sh/piko/internal/esbuild/helpers"
	"piko.sh/piko/internal/esbuild/js_ast"
	"piko.sh/piko/internal/esbuild/logger"
)

func TestTypeExtractor_BasicTypes(t *testing.T) {
	code := `
		const state = {
			count: 0 as number,
			message: "hello" as string,
			active: true as boolean
		};
	`

	parser := NewTypeScriptParser()
	tree, err := parser.ParseTypeScript(code, "test.ts")
	require.NoError(t, err)
	require.NotNil(t, tree)

	typeAssertions := ExtractTypeAssertions(code)

	extractor := NewTypeExtractor(tree, typeAssertions)
	metadata, err := extractor.ExtractMetadata()
	require.NoError(t, err)
	require.NotNil(t, metadata)

	assert.Equal(t, 3, len(metadata.StateProperties))
	assert.Equal(t, "number", metadata.StateProperties["count"].JSType)
	assert.Equal(t, "string", metadata.StateProperties["message"].JSType)
	assert.Equal(t, "boolean", metadata.StateProperties["active"].JSType)
	assert.Contains(t, metadata.BooleanProps, "active")
}

func TestTypeExtractor_Arrays(t *testing.T) {
	code := `
		const state = {
			items: [] as string[],
			numbers: [1, 2, 3] as number[]
		};
	`

	parser := NewTypeScriptParser()
	tree, err := parser.ParseTypeScript(code, "test.ts")
	require.NoError(t, err)

	typeAssertions := ExtractTypeAssertions(code)
	extractor := NewTypeExtractor(tree, typeAssertions)
	metadata, err := extractor.ExtractMetadata()
	require.NoError(t, err)

	assert.Equal(t, "array", metadata.StateProperties["items"].JSType)
	assert.Equal(t, "string", metadata.StateProperties["items"].ElementType)
	assert.Equal(t, "array", metadata.StateProperties["numbers"].JSType)
	assert.Equal(t, "number", metadata.StateProperties["numbers"].ElementType)
}

func TestTypeExtractor_InferredTypes(t *testing.T) {
	code := `
		const state = {
			count: 0,
			message: "hello",
			active: true,
			items: [],
			data: {}
		};
	`

	parser := NewTypeScriptParser()
	tree, err := parser.ParseTypeScript(code, "test.ts")
	require.NoError(t, err)

	typeAssertions := ExtractTypeAssertions(code)
	extractor := NewTypeExtractor(tree, typeAssertions)
	metadata, err := extractor.ExtractMetadata()
	require.NoError(t, err)

	assert.Equal(t, "number", metadata.StateProperties["count"].JSType)
	assert.Equal(t, "string", metadata.StateProperties["message"].JSType)
	assert.Equal(t, "boolean", metadata.StateProperties["active"].JSType)
	assert.Equal(t, "array", metadata.StateProperties["items"].JSType)
	assert.Equal(t, "object", metadata.StateProperties["data"].JSType)
}

func TestTypeExtractor_MixedTyping(t *testing.T) {
	code := `
		const state = {
			explicit: "test" as string,
			inferred: 42,
			complex: [] as string[]
		};
	`

	parser := NewTypeScriptParser()
	tree, err := parser.ParseTypeScript(code, "test.ts")
	require.NoError(t, err)

	typeAssertions := ExtractTypeAssertions(code)
	extractor := NewTypeExtractor(tree, typeAssertions)
	metadata, err := extractor.ExtractMetadata()
	require.NoError(t, err)

	assert.Equal(t, "string", metadata.StateProperties["explicit"].JSType)
	assert.Equal(t, "number", metadata.StateProperties["inferred"].JSType)
	assert.Equal(t, "array", metadata.StateProperties["complex"].JSType)
	assert.Equal(t, "string", metadata.StateProperties["complex"].ElementType)
}

func TestTypeExtractor_EmptyState(t *testing.T) {
	code := `
		const other = "not state";
	`

	parser := NewTypeScriptParser()
	tree, err := parser.ParseTypeScript(code, "test.ts")
	require.NoError(t, err)

	typeAssertions := ExtractTypeAssertions(code)
	extractor := NewTypeExtractor(tree, typeAssertions)
	metadata, err := extractor.ExtractMetadata()
	require.NoError(t, err)

	assert.Equal(t, 0, len(metadata.StateProperties))
	assert.Equal(t, 0, len(metadata.BooleanProps))
}

func TestTypeExtractor_Methods(t *testing.T) {
	code := `
		const state = {
			count: 0 as number
		};

		function increment() {
			state.count++;
		}

		function decrement() {
			state.count--;
		}
	`

	parser := NewTypeScriptParser()
	tree, err := parser.ParseTypeScript(code, "test.ts")
	require.NoError(t, err)

	typeAssertions := ExtractTypeAssertions(code)
	extractor := NewTypeExtractor(tree, typeAssertions)
	metadata, err := extractor.ExtractMetadata()
	require.NoError(t, err)

	assert.Equal(t, 1, len(metadata.StateProperties))
	assert.Equal(t, 2, len(metadata.Methods))
}

func TestPropertyMetadata_GetPropType(t *testing.T) {
	tests := []struct {
		name     string
		expected string
		prop     PropertyMetadata
	}{
		{
			name:     "string type",
			prop:     PropertyMetadata{JSType: "string"},
			expected: "String",
		},
		{
			name:     "number type",
			prop:     PropertyMetadata{JSType: "number"},
			expected: "Number",
		},
		{
			name:     "boolean type",
			prop:     PropertyMetadata{JSType: "boolean"},
			expected: "Boolean",
		},
		{
			name:     "array type",
			prop:     PropertyMetadata{JSType: "array"},
			expected: "Array",
		},
		{
			name:     "array with element type",
			prop:     PropertyMetadata{JSType: "array", ElementType: "string"},
			expected: "Array<String>",
		},
		{
			name:     "map type",
			prop:     PropertyMetadata{JSType: "object", KeyType: "string", ValueType: "number"},
			expected: "Map<String,Number>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.prop.GetPropType()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPropertyMetadata_GetDefaultValue(t *testing.T) {
	tests := []struct {
		name     string
		expected string
		prop     PropertyMetadata
	}{
		{
			name:     "with initial value",
			prop:     PropertyMetadata{JSType: "string", InitialValue: `"test"`},
			expected: `"test"`,
		},
		{
			name:     "string fallback",
			prop:     PropertyMetadata{JSType: "string"},
			expected: `""`,
		},
		{
			name:     "number fallback",
			prop:     PropertyMetadata{JSType: "number"},
			expected: "0",
		},
		{
			name:     "boolean fallback",
			prop:     PropertyMetadata{JSType: "boolean"},
			expected: "false",
		},
		{
			name:     "array fallback",
			prop:     PropertyMetadata{JSType: "array"},
			expected: "[]",
		},
		{
			name:     "object fallback",
			prop:     PropertyMetadata{JSType: "object"},
			expected: "{}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.prop.GetDefaultValue()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func newTestExtractor() *typeExtractor {
	return &typeExtractor{
		ast:            &js_ast.AST{},
		typeAssertions: map[string]TypeAssertion{},
	}
}

func newTestExtractorWithSymbols(symbols []ast.Symbol) *typeExtractor {
	return &typeExtractor{
		ast: &js_ast.AST{
			Symbols: symbols,
		},
		typeAssertions: map[string]TypeAssertion{},
	}
}

func makeStringExpr(s string) js_ast.Expr {
	return js_ast.Expr{
		Data: &js_ast.EString{Value: helpers.StringToUTF16(s)},
		Loc:  logger.Loc{},
	}
}

func TestExprToString(t *testing.T) {
	t.Parallel()

	ext := newTestExtractor()

	tests := []struct {
		name       string
		expression js_ast.Expr
		expected   string
	}{
		{
			name:       "number integer",
			expression: js_ast.Expr{Data: &js_ast.ENumber{Value: 42}, Loc: logger.Loc{}},
			expected:   "42",
		},
		{
			name:       "number float",
			expression: js_ast.Expr{Data: &js_ast.ENumber{Value: 3.14}, Loc: logger.Loc{}},
			expected:   "3.14",
		},
		{
			name:       "number zero",
			expression: js_ast.Expr{Data: &js_ast.ENumber{Value: 0}, Loc: logger.Loc{}},
			expected:   "0",
		},
		{
			name:       "number negative",
			expression: js_ast.Expr{Data: &js_ast.ENumber{Value: -7}, Loc: logger.Loc{}},
			expected:   "-7",
		},
		{
			name:       "string simple",
			expression: makeStringExpr("hello"),
			expected:   `"hello"`,
		},
		{
			name:       "string empty",
			expression: makeStringExpr(""),
			expected:   `""`,
		},
		{
			name:       "string with special characters",
			expression: makeStringExpr(`line\nbreak`),
			expected:   `"line\\nbreak"`,
		},
		{
			name:       "boolean true",
			expression: js_ast.Expr{Data: &js_ast.EBoolean{Value: true}, Loc: logger.Loc{}},
			expected:   "true",
		},
		{
			name:       "boolean false",
			expression: js_ast.Expr{Data: &js_ast.EBoolean{Value: false}, Loc: logger.Loc{}},
			expected:   "false",
		},
		{
			name:       "empty array",
			expression: js_ast.Expr{Data: &js_ast.EArray{Items: nil}, Loc: logger.Loc{}},
			expected:   "[]",
		},
		{
			name: "array with single number",
			expression: js_ast.Expr{
				Data: &js_ast.EArray{Items: []js_ast.Expr{
					{Data: &js_ast.ENumber{Value: 1}, Loc: logger.Loc{}},
				}},
				Loc: logger.Loc{},
			},
			expected: "[1]",
		},
		{
			name: "array with multiple elements",
			expression: js_ast.Expr{
				Data: &js_ast.EArray{Items: []js_ast.Expr{
					{Data: &js_ast.ENumber{Value: 1}, Loc: logger.Loc{}},
					{Data: &js_ast.ENumber{Value: 2}, Loc: logger.Loc{}},
					{Data: &js_ast.ENumber{Value: 3}, Loc: logger.Loc{}},
				}},
				Loc: logger.Loc{},
			},
			expected: "[1, 2, 3]",
		},
		{
			name: "array with mixed types",
			expression: js_ast.Expr{
				Data: &js_ast.EArray{Items: []js_ast.Expr{
					{Data: &js_ast.ENumber{Value: 1}, Loc: logger.Loc{}},
					makeStringExpr("two"),
					{Data: &js_ast.EBoolean{Value: true}, Loc: logger.Loc{}},
				}},
				Loc: logger.Loc{},
			},
			expected: `[1, "two", true]`,
		},
		{
			name:       "empty object",
			expression: js_ast.Expr{Data: &js_ast.EObject{Properties: nil}, Loc: logger.Loc{}},
			expected:   "{}",
		},
		{
			name: "object with string key and number value",
			expression: js_ast.Expr{
				Data: &js_ast.EObject{Properties: []js_ast.Property{
					{
						Key:        makeStringExpr("count"),
						ValueOrNil: js_ast.Expr{Data: &js_ast.ENumber{Value: 5}, Loc: logger.Loc{}},
					},
				}},
				Loc: logger.Loc{},
			},
			expected: `{"count": 5}`,
		},
		{
			name: "object with multiple properties",
			expression: js_ast.Expr{
				Data: &js_ast.EObject{Properties: []js_ast.Property{
					{
						Key:        makeStringExpr("name"),
						ValueOrNil: makeStringExpr("alice"),
					},
					{
						Key:        makeStringExpr("age"),
						ValueOrNil: js_ast.Expr{Data: &js_ast.ENumber{Value: 30}, Loc: logger.Loc{}},
					},
				}},
				Loc: logger.Loc{},
			},
			expected: `{"name": "alice", "age": 30}`,
		},
		{
			name:       "null expression",
			expression: js_ast.Expr{Data: &js_ast.ENull{}, Loc: logger.Loc{}},
			expected:   "null",
		},
		{
			name:       "undefined expression",
			expression: js_ast.Expr{Data: &js_ast.EUndefined{}, Loc: logger.Loc{}},
			expected:   "undefined",
		},
		{
			name:       "unrecognised expression falls back to null",
			expression: js_ast.Expr{Data: &js_ast.EThis{}, Loc: logger.Loc{}},
			expected:   "null",
		},
		{
			name: "nested array inside object",
			expression: js_ast.Expr{
				Data: &js_ast.EObject{Properties: []js_ast.Property{
					{
						Key: makeStringExpr("items"),
						ValueOrNil: js_ast.Expr{
							Data: &js_ast.EArray{Items: []js_ast.Expr{
								{Data: &js_ast.ENumber{Value: 1}, Loc: logger.Loc{}},
								{Data: &js_ast.ENumber{Value: 2}, Loc: logger.Loc{}},
							}},
							Loc: logger.Loc{},
						},
					},
				}},
				Loc: logger.Loc{},
			},
			expected: `{"items": [1, 2]}`,
		},
		{
			name: "nested object inside array",
			expression: js_ast.Expr{
				Data: &js_ast.EArray{Items: []js_ast.Expr{
					{
						Data: &js_ast.EObject{Properties: []js_ast.Property{
							{
								Key:        makeStringExpr("id"),
								ValueOrNil: js_ast.Expr{Data: &js_ast.ENumber{Value: 1}, Loc: logger.Loc{}},
							},
						}},
						Loc: logger.Loc{},
					},
				}},
				Loc: logger.Loc{},
			},
			expected: `[{"id": 1}]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := ext.expressionToString(tt.expression)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBooleanToString(t *testing.T) {
	t.Parallel()

	ext := newTestExtractor()

	tests := []struct {
		name     string
		expected string
		value    bool
	}{
		{
			name:     "true value returns true",
			value:    true,
			expected: "true",
		},
		{
			name:     "false value returns false",
			value:    false,
			expected: "false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := ext.booleanToString(tt.value)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestArrayToString(t *testing.T) {
	t.Parallel()

	ext := newTestExtractor()

	tests := []struct {
		name     string
		array    *js_ast.EArray
		expected string
	}{
		{
			name:     "empty array with nil items",
			array:    &js_ast.EArray{Items: nil},
			expected: "[]",
		},
		{
			name:     "empty array with zero-length slice",
			array:    &js_ast.EArray{Items: []js_ast.Expr{}},
			expected: "[]",
		},
		{
			name: "single string element",
			array: &js_ast.EArray{Items: []js_ast.Expr{
				makeStringExpr("hello"),
			}},
			expected: `["hello"]`,
		},
		{
			name: "multiple number elements",
			array: &js_ast.EArray{Items: []js_ast.Expr{
				{Data: &js_ast.ENumber{Value: 10}, Loc: logger.Loc{}},
				{Data: &js_ast.ENumber{Value: 20}, Loc: logger.Loc{}},
				{Data: &js_ast.ENumber{Value: 30}, Loc: logger.Loc{}},
			}},
			expected: "[10, 20, 30]",
		},
		{
			name: "array with null element",
			array: &js_ast.EArray{Items: []js_ast.Expr{
				{Data: &js_ast.ENull{}, Loc: logger.Loc{}},
			}},
			expected: "[null]",
		},
		{
			name: "nested array",
			array: &js_ast.EArray{Items: []js_ast.Expr{
				{
					Data: &js_ast.EArray{Items: []js_ast.Expr{
						{Data: &js_ast.ENumber{Value: 1}, Loc: logger.Loc{}},
					}},
					Loc: logger.Loc{},
				},
			}},
			expected: "[[1]]",
		},
		{
			name: "array with boolean elements",
			array: &js_ast.EArray{Items: []js_ast.Expr{
				{Data: &js_ast.EBoolean{Value: true}, Loc: logger.Loc{}},
				{Data: &js_ast.EBoolean{Value: false}, Loc: logger.Loc{}},
			}},
			expected: "[true, false]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := ext.arrayToString(tt.array)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestObjectToString(t *testing.T) {
	t.Parallel()

	ext := newTestExtractor()

	tests := []struct {
		name     string
		object   *js_ast.EObject
		expected string
	}{
		{
			name:     "empty object with nil properties",
			object:   &js_ast.EObject{Properties: nil},
			expected: "{}",
		},
		{
			name:     "empty object with zero-length slice",
			object:   &js_ast.EObject{Properties: []js_ast.Property{}},
			expected: "{}",
		},
		{
			name: "single string-keyed property with number value",
			object: &js_ast.EObject{Properties: []js_ast.Property{
				{
					Key:        makeStringExpr("x"),
					ValueOrNil: js_ast.Expr{Data: &js_ast.ENumber{Value: 1}, Loc: logger.Loc{}},
				},
			}},
			expected: `{"x": 1}`,
		},
		{
			name: "multiple properties",
			object: &js_ast.EObject{Properties: []js_ast.Property{
				{
					Key:        makeStringExpr("a"),
					ValueOrNil: makeStringExpr("alpha"),
				},
				{
					Key:        makeStringExpr("b"),
					ValueOrNil: js_ast.Expr{Data: &js_ast.EBoolean{Value: false}, Loc: logger.Loc{}},
				},
			}},
			expected: `{"a": "alpha", "b": false}`,
		},
		{
			name: "property with non-string key is skipped",
			object: &js_ast.EObject{Properties: []js_ast.Property{
				{
					Key:        js_ast.Expr{Data: &js_ast.ENumber{Value: 42}, Loc: logger.Loc{}},
					ValueOrNil: makeStringExpr("ignored"),
				},
				{
					Key:        makeStringExpr("kept"),
					ValueOrNil: js_ast.Expr{Data: &js_ast.ENumber{Value: 1}, Loc: logger.Loc{}},
				},
			}},
			expected: `{"kept": 1}`,
		},
		{
			name: "property with null value",
			object: &js_ast.EObject{Properties: []js_ast.Property{
				{
					Key:        makeStringExpr("val"),
					ValueOrNil: js_ast.Expr{Data: &js_ast.ENull{}, Loc: logger.Loc{}},
				},
			}},
			expected: `{"val": null}`,
		},
		{
			name: "nested object value",
			object: &js_ast.EObject{Properties: []js_ast.Property{
				{
					Key: makeStringExpr("inner"),
					ValueOrNil: js_ast.Expr{
						Data: &js_ast.EObject{Properties: []js_ast.Property{
							{
								Key:        makeStringExpr("deep"),
								ValueOrNil: js_ast.Expr{Data: &js_ast.ENumber{Value: 99}, Loc: logger.Loc{}},
							},
						}},
						Loc: logger.Loc{},
					},
				},
			}},
			expected: `{"inner": {"deep": 99}}`,
		},
		{
			name: "all properties with non-string keys produces empty braces",
			object: &js_ast.EObject{Properties: []js_ast.Property{
				{
					Key:        js_ast.Expr{Data: &js_ast.ENumber{Value: 0}, Loc: logger.Loc{}},
					ValueOrNil: makeStringExpr("nope"),
				},
			}},
			expected: "{}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := ext.objectToString(tt.object)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetPropertyName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		key      js_ast.Expr
		name     string
		expected string
		symbols  []ast.Symbol
	}{
		{
			name:     "string key returns decoded value",
			symbols:  nil,
			key:      makeStringExpr("count"),
			expected: "count",
		},
		{
			name:     "empty string key returns empty string",
			symbols:  nil,
			key:      makeStringExpr(""),
			expected: "",
		},
		{
			name: "identifier key resolves via symbols table",
			symbols: []ast.Symbol{
				{OriginalName: "myProp"},
			},
			key: js_ast.Expr{
				Data: &js_ast.EIdentifier{Ref: ast.Ref{SourceIndex: 0, InnerIndex: 0}},
				Loc:  logger.Loc{},
			},
			expected: "myProp",
		},
		{
			name: "identifier key with out-of-bounds index returns empty",
			symbols: []ast.Symbol{
				{OriginalName: "only"},
			},
			key: js_ast.Expr{
				Data: &js_ast.EIdentifier{Ref: ast.Ref{SourceIndex: 0, InnerIndex: 5}},
				Loc:  logger.Loc{},
			},
			expected: "",
		},
		{
			name:    "identifier key with empty symbols table returns empty",
			symbols: nil,
			key: js_ast.Expr{
				Data: &js_ast.EIdentifier{Ref: ast.Ref{SourceIndex: 0, InnerIndex: 0}},
				Loc:  logger.Loc{},
			},
			expected: "",
		},
		{
			name:     "unrecognised key type returns empty",
			symbols:  nil,
			key:      js_ast.Expr{Data: &js_ast.ENumber{Value: 42}, Loc: logger.Loc{}},
			expected: "",
		},
		{
			name:     "boolean key type returns empty",
			symbols:  nil,
			key:      js_ast.Expr{Data: &js_ast.EBoolean{Value: true}, Loc: logger.Loc{}},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ext := newTestExtractorWithSymbols(tt.symbols)
			result := ext.getPropertyName(tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractPropertyKey(t *testing.T) {
	t.Parallel()

	ext := newTestExtractor()

	tests := []struct {
		name     string
		keyExpr  js_ast.Expr
		expected string
	}{
		{
			name:     "string key returns decoded value",
			keyExpr:  makeStringExpr("title"),
			expected: "title",
		},
		{
			name:     "empty string key returns empty string",
			keyExpr:  makeStringExpr(""),
			expected: "",
		},
		{
			name:     "string key with unicode characters",
			keyExpr:  makeStringExpr("caf\u00e9"),
			expected: "caf\u00e9",
		},
		{
			name:     "number key returns empty string",
			keyExpr:  js_ast.Expr{Data: &js_ast.ENumber{Value: 0}, Loc: logger.Loc{}},
			expected: "",
		},
		{
			name:     "boolean key returns empty string",
			keyExpr:  js_ast.Expr{Data: &js_ast.EBoolean{Value: true}, Loc: logger.Loc{}},
			expected: "",
		},
		{
			name:     "identifier key returns empty string",
			keyExpr:  js_ast.Expr{Data: &js_ast.EIdentifier{Ref: ast.Ref{}}, Loc: logger.Loc{}},
			expected: "",
		},
		{
			name:     "null key returns empty string",
			keyExpr:  js_ast.Expr{Data: &js_ast.ENull{}, Loc: logger.Loc{}},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := ext.extractPropertyKey(tt.keyExpr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExprToString_IntegrationWithParsedAST(t *testing.T) {
	t.Parallel()

	code := `
		const state = {
			count: 0 as number,
			name: "default" as string,
			active: true as boolean,
			items: [1, 2, 3] as number[],
			config: {} as object
		};
	`

	parser := NewTypeScriptParser()
	parsedAST, err := parser.ParseTypeScript(code, "test.ts")
	require.NoError(t, err)
	require.NotNil(t, parsedAST)

	typeAssertions := ExtractTypeAssertions(code)
	extractor := NewTypeExtractor(parsedAST, typeAssertions)
	metadata, err := extractor.ExtractMetadata()
	require.NoError(t, err)

	assert.Equal(t, "0", metadata.StateProperties["count"].InitialValue)
	assert.Equal(t, `"default"`, metadata.StateProperties["name"].InitialValue)
	assert.Equal(t, "true", metadata.StateProperties["active"].InitialValue)
	assert.Equal(t, "[1, 2, 3]", metadata.StateProperties["items"].InitialValue)
	assert.Equal(t, "{}", metadata.StateProperties["config"].InitialValue)
}
