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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractFunctionParams_BasicFunction(t *testing.T) {
	t.Parallel()
	source := `function foo(a: string, b: number): void {}`
	result := ExtractFunctionParams(source)
	require.Contains(t, result, "foo")
	params := result["foo"]
	require.Len(t, params, 2)
	assert.Equal(t, "a", params[0].Name)
	assert.Equal(t, "string", params[0].TypeName)
	assert.Equal(t, categoryString, params[0].Category)
	assert.Equal(t, "b", params[1].Name)
	assert.Equal(t, "number", params[1].TypeName)
	assert.Equal(t, categoryNumber, params[1].Category)
}

func TestExtractFunctionParams_ArrowFunction(t *testing.T) {
	t.Parallel()
	source := `const foo = (x: boolean): void => {}`
	result := ExtractFunctionParams(source)
	require.Contains(t, result, "foo")
	params := result["foo"]
	require.Len(t, params, 1)
	assert.Equal(t, "x", params[0].Name)
	assert.Equal(t, "boolean", params[0].TypeName)
	assert.Equal(t, categoryBoolean, params[0].Category)
}

func TestExtractFunctionParams_AsyncFunction(t *testing.T) {
	t.Parallel()
	source := `async function bar(x: string): Promise<void> {}`
	result := ExtractFunctionParams(source)
	require.Contains(t, result, "bar")
	params := result["bar"]
	require.Len(t, params, 1)
	assert.Equal(t, "x", params[0].Name)
	assert.Equal(t, categoryString, params[0].Category)
}

func TestExtractFunctionParams_AsyncArrowFunction(t *testing.T) {
	t.Parallel()
	source := `const baz = async (y: number): Promise<void> => {}`
	result := ExtractFunctionParams(source)
	require.Contains(t, result, "baz")
	params := result["baz"]
	require.Len(t, params, 1)
	assert.Equal(t, "y", params[0].Name)
	assert.Equal(t, categoryNumber, params[0].Category)
}

func TestExtractFunctionParams_OptionalParam(t *testing.T) {
	t.Parallel()
	source := `function f(a: string, b?: number): void {}`
	result := ExtractFunctionParams(source)
	require.Contains(t, result, "f")
	params := result["f"]
	require.Len(t, params, 2)
	assert.False(t, params[0].Optional)
	assert.True(t, params[1].Optional)
	assert.Equal(t, categoryNumber, params[1].Category)
}

func TestExtractFunctionParams_DefaultValueMarksOptional(t *testing.T) {
	t.Parallel()
	source := `function openLightbox(e, slideIndex = 0) { console.log(e, slideIndex); }`
	result := ExtractFunctionParams(source)
	require.Contains(t, result, "openLightbox")
	params := result["openLightbox"]
	require.Len(t, params, 2)
	assert.Equal(t, "e", params[0].Name)
	assert.False(t, params[0].Optional)
	assert.Equal(t, "slideIndex", params[1].Name)
	assert.True(t, params[1].Optional)
}

func TestExtractFunctionParams_TypedDefaultValueMarksOptional(t *testing.T) {
	t.Parallel()
	source := `function greetUser(name: string, greeting: string = "Hello", punctuation: string = "!") {}`
	result := ExtractFunctionParams(source)
	require.Contains(t, result, "greetUser")
	params := result["greetUser"]
	require.Len(t, params, 3)
	assert.Equal(t, "name", params[0].Name)
	assert.False(t, params[0].Optional)
	assert.Equal(t, "greeting", params[1].Name)
	assert.True(t, params[1].Optional)
	assert.Equal(t, "string", params[1].TypeName)
	assert.Equal(t, "punctuation", params[2].Name)
	assert.True(t, params[2].Optional)
}

func TestExtractFunctionParams_RestParam(t *testing.T) {
	t.Parallel()
	source := `function f(...items: string[]): void {}`
	result := ExtractFunctionParams(source)
	require.Contains(t, result, "f")
	params := result["f"]
	require.Len(t, params, 1)
	assert.True(t, params[0].IsRest)
	assert.Equal(t, "items", params[0].Name)
	assert.Equal(t, "string[]", params[0].TypeName)
	assert.Equal(t, categoryObject, params[0].Category)
}

func TestExtractFunctionParams_DefaultValue(t *testing.T) {
	t.Parallel()
	source := `function f(x: number = 42): void {}`
	result := ExtractFunctionParams(source)
	require.Contains(t, result, "f")
	params := result["f"]
	require.Len(t, params, 1)
	assert.Equal(t, "x", params[0].Name)
	assert.Equal(t, categoryNumber, params[0].Category)
}

func TestExtractFunctionParams_NoTypeAnnotation(t *testing.T) {
	t.Parallel()
	source := `function f(x): void {}`
	result := ExtractFunctionParams(source)
	require.Contains(t, result, "f")
	params := result["f"]
	require.Len(t, params, 1)
	assert.Equal(t, "x", params[0].Name)
	assert.Equal(t, "", params[0].TypeName)
	assert.Equal(t, categoryAny, params[0].Category)
}

func TestExtractFunctionParams_ComplexTypeIsObject(t *testing.T) {
	t.Parallel()
	source := `function f(config: Record<string, any>): void {}`
	result := ExtractFunctionParams(source)
	require.Contains(t, result, "f")
	params := result["f"]
	require.Len(t, params, 1)
	assert.Equal(t, categoryObject, params[0].Category)
}

func TestExtractFunctionParams_NullableUnionType(t *testing.T) {
	t.Parallel()
	source := `function f(x: string | null): void {}`
	result := ExtractFunctionParams(source)
	require.Contains(t, result, "f")
	params := result["f"]
	require.Len(t, params, 1)
	assert.Equal(t, categoryString, params[0].Category)
}

func TestExtractFunctionParams_MultipleFunctions(t *testing.T) {
	t.Parallel()
	source := `
function alpha(a: string): void {}
function beta(b: number): void {}
const gamma = (c: boolean): void => {}
`
	result := ExtractFunctionParams(source)
	require.Contains(t, result, "alpha")
	require.Contains(t, result, "beta")
	require.Contains(t, result, "gamma")
	assert.Equal(t, categoryString, result["alpha"][0].Category)
	assert.Equal(t, categoryNumber, result["beta"][0].Category)
	assert.Equal(t, categoryBoolean, result["gamma"][0].Category)
}

func TestExtractFunctionParams_ExportedFunction(t *testing.T) {
	t.Parallel()
	source := `export function f(x: string): void {}`
	result := ExtractFunctionParams(source)
	require.Contains(t, result, "f")
	assert.Equal(t, categoryString, result["f"][0].Category)
}

func TestExtractFunctionParams_ExportedArrow(t *testing.T) {
	t.Parallel()
	source := `export const f = (x: number): void => {}`
	result := ExtractFunctionParams(source)
	require.Contains(t, result, "f")
	assert.Equal(t, categoryNumber, result["f"][0].Category)
}

func TestExtractFunctionParams_NonFunctionConst(t *testing.T) {
	t.Parallel()
	source := `const x = 42;`
	result := ExtractFunctionParams(source)
	assert.NotContains(t, result, "x")
}

func TestExtractFunctionParams_DestructuredParam(t *testing.T) {
	t.Parallel()
	source := `function f({a, b}: Config): void {}`
	result := ExtractFunctionParams(source)
	require.Contains(t, result, "f")
	params := result["f"]
	require.Len(t, params, 1)
	assert.Equal(t, "(destructured)", params[0].Name)
	assert.Equal(t, categoryObject, params[0].Category)
}

func TestExtractFunctionParams_NoParams(t *testing.T) {
	t.Parallel()
	source := `function f(): void {}`
	result := ExtractFunctionParams(source)
	require.Contains(t, result, "f")
	assert.Empty(t, result["f"])
}

func TestExtractFunctionParams_EmptySource(t *testing.T) {
	t.Parallel()
	result := ExtractFunctionParams("")
	assert.Nil(t, result)
}

func TestExtractFunctionParams_EventParam(t *testing.T) {
	t.Parallel()
	source := `export function handleSubmit(event: Event): void {}`
	result := ExtractFunctionParams(source)
	require.Contains(t, result, "handleSubmit")
	params := result["handleSubmit"]
	require.Len(t, params, 1)
	assert.Equal(t, "event", params[0].Name)
	assert.Equal(t, "Event", params[0].TypeName)
	assert.Equal(t, categoryObject, params[0].Category)
}

func TestExtractFunctionParams_MixedParamsWithDefaults(t *testing.T) {
	t.Parallel()
	source := `function f(name: string, count: number = 0, active?: boolean): void {}`
	result := ExtractFunctionParams(source)
	require.Contains(t, result, "f")
	params := result["f"]
	require.Len(t, params, 3)

	assert.Equal(t, "name", params[0].Name)
	assert.Equal(t, categoryString, params[0].Category)
	assert.False(t, params[0].Optional)

	assert.Equal(t, "count", params[1].Name)
	assert.Equal(t, categoryNumber, params[1].Category)
	assert.True(t, params[1].Optional)

	assert.Equal(t, "active", params[2].Name)
	assert.Equal(t, categoryBoolean, params[2].Category)
	assert.True(t, params[2].Optional)
}

func TestClassifyTSType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected string
	}{
		{"string", categoryString},
		{"number", categoryNumber},
		{"bigint", categoryNumber},
		{"boolean", categoryBoolean},
		{"bool", categoryBoolean},
		{"any", categoryAny},
		{"unknown", categoryAny},
		{"void", categoryAny},
		{"never", categoryAny},
		{"undefined", categoryAny},
		{"null", categoryAny},
		{"", categoryAny},
		{"Event", categoryObject},
		{"HTMLElement", categoryObject},
		{"Record<string,any>", categoryObject},
		{"string[]", categoryObject},
		{"Array<number>", categoryObject},
		{"string|null", categoryString},
		{"number|undefined", categoryNumber},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, classifyTSType(tt.input))
		})
	}
}

func TestExtractFunctionParams_GenericFunction(t *testing.T) {
	t.Parallel()
	source := `function merge<T>(a: T, b: T): T { return a; }`
	result := ExtractFunctionParams(source)
	require.Contains(t, result, "merge")
	params := result["merge"]
	require.Len(t, params, 2)
	assert.Equal(t, "a", params[0].Name)
	assert.Equal(t, "T", params[0].TypeName)
	assert.Equal(t, categoryObject, params[0].Category)
}

func TestExtractFunctionParams_GenericArrow(t *testing.T) {
	t.Parallel()
	source := `const identity = <T>(x: T): T => x;`
	result := ExtractFunctionParams(source)
	require.Contains(t, result, "identity")
	params := result["identity"]
	require.Len(t, params, 1)
	assert.Equal(t, "x", params[0].Name)
}

func TestExtractFunctionParams_NestedGenerics(t *testing.T) {
	t.Parallel()
	source := `function wrap<T>(val: Promise<Map<string, T>>): void {}`
	result := ExtractFunctionParams(source)
	require.Contains(t, result, "wrap")
	params := result["wrap"]
	require.Len(t, params, 1)
	assert.Equal(t, "val", params[0].Name)
	assert.Equal(t, "Promise<Map<string,T>>", params[0].TypeName)
	assert.Equal(t, categoryObject, params[0].Category)
}

func TestExtractFunctionParams_IntersectionType(t *testing.T) {
	t.Parallel()
	source := `function f(x: A & B): void {}`
	result := ExtractFunctionParams(source)
	require.Contains(t, result, "f")
	params := result["f"]
	require.Len(t, params, 1)
	assert.Contains(t, params[0].TypeName, "&")
	assert.Equal(t, categoryObject, params[0].Category)
}

func TestExtractFunctionParams_ObjectTypeAnnotation(t *testing.T) {
	t.Parallel()
	source := `function f(x: { name: string }): void {}`
	result := ExtractFunctionParams(source)
	require.Contains(t, result, "f")
	params := result["f"]
	require.Len(t, params, 1)
	assert.Equal(t, "x", params[0].Name)
	assert.Equal(t, categoryObject, params[0].Category)
}

func TestExtractFunctionParams_StringLiteralType(t *testing.T) {
	t.Parallel()
	source := `function f(mode: "read" | "write"): void {}`
	result := ExtractFunctionParams(source)
	require.Contains(t, result, "f")
	params := result["f"]
	require.Len(t, params, 1)
	assert.Equal(t, "mode", params[0].Name)
}

func TestExtractFunctionParams_NumericLiteralType(t *testing.T) {
	t.Parallel()
	source := `function f(code: 200 | 404): void {}`
	result := ExtractFunctionParams(source)
	require.Contains(t, result, "f")
	params := result["f"]
	require.Len(t, params, 1)
	assert.Equal(t, "code", params[0].Name)
}

func TestExtractFunctionParams_TupleType(t *testing.T) {
	t.Parallel()
	source := `function f(x: [string, number]): void {}`
	result := ExtractFunctionParams(source)
	require.Contains(t, result, "f")
	params := result["f"]
	require.Len(t, params, 1)
	assert.Equal(t, "x", params[0].Name)
	assert.Equal(t, categoryObject, params[0].Category)
}

func TestExtractFunctionParams_NullableType(t *testing.T) {
	t.Parallel()
	source := `function f(x: string | null | undefined): void {}`
	result := ExtractFunctionParams(source)
	require.Contains(t, result, "f")
	params := result["f"]
	require.Len(t, params, 1)
	assert.Equal(t, categoryString, params[0].Category)
}

func TestExtractFunctionParams_DefaultObjectValue(t *testing.T) {
	t.Parallel()
	source := `function f(config: Options = { debug: true }): void {}`
	result := ExtractFunctionParams(source)
	require.Contains(t, result, "f")
	params := result["f"]
	require.Len(t, params, 1)
	assert.Equal(t, "config", params[0].Name)
	assert.Equal(t, categoryObject, params[0].Category)
}

func TestExtractFunctionParams_DefaultArrayValue(t *testing.T) {
	t.Parallel()
	source := `function f(items: string[] = ["a", "b"]): void {}`
	result := ExtractFunctionParams(source)
	require.Contains(t, result, "f")
	params := result["f"]
	require.Len(t, params, 1)
	assert.Equal(t, "items", params[0].Name)
}

func TestExtractFunctionParams_DefaultFunctionCallValue(t *testing.T) {
	t.Parallel()
	source := `function f(x: number = getDefault(1, 2)): void {}`
	result := ExtractFunctionParams(source)
	require.Contains(t, result, "f")
	params := result["f"]
	require.Len(t, params, 1)
	assert.Equal(t, "x", params[0].Name)
	assert.Equal(t, categoryNumber, params[0].Category)
}

func TestExtractFunctionParams_ArrayDestructuredParam(t *testing.T) {
	t.Parallel()
	source := `function f([a, b]: Config): void {}`
	result := ExtractFunctionParams(source)
	require.Contains(t, result, "f")
	params := result["f"]
	require.Len(t, params, 1)
	assert.Equal(t, "(destructured)", params[0].Name)
	assert.Equal(t, categoryObject, params[0].Category)
}

func TestExtractFunctionParams_NestedDestructuredParam(t *testing.T) {
	t.Parallel()
	source := `function f({ inner: { a, b } }: Config): void {}`
	result := ExtractFunctionParams(source)
	require.Contains(t, result, "f")
	params := result["f"]
	require.Len(t, params, 1)
	assert.Equal(t, "(destructured)", params[0].Name)
	assert.Equal(t, categoryObject, params[0].Category)
}

func TestExtractFunctionParams_GroupedParenType(t *testing.T) {
	t.Parallel()
	source := `function f(x: (string | number)): void {}`
	result := ExtractFunctionParams(source)
	require.Contains(t, result, "f")
	params := result["f"]
	require.Len(t, params, 1)
	assert.Equal(t, "x", params[0].Name)
}

func TestExtractFunctionParams_ConditionalType(t *testing.T) {
	t.Parallel()
	source := `function f(x: string extends any ? string : never): void {}`
	result := ExtractFunctionParams(source)
	require.Contains(t, result, "f")
	params := result["f"]
	require.Len(t, params, 1)
}

func TestExtractFunctionParams_QualifiedType(t *testing.T) {
	t.Parallel()
	source := `function f(x: Namespace.Type): void {}`
	result := ExtractFunctionParams(source)
	require.Contains(t, result, "f")
	params := result["f"]
	require.Len(t, params, 1)
	assert.Contains(t, params[0].TypeName, ".")
	assert.Equal(t, categoryObject, params[0].Category)
}

func TestExtractFunctionParams_LexerPanicRecovery(t *testing.T) {
	t.Parallel()

	source := "function greet(name: string) { return `hello ${name}`; }\nfunction other(x: number) {}"

	result := ExtractFunctionParams(source)

	assert.NotNil(t, result)
}

func TestStripNullableFromType_AllNullable(t *testing.T) {
	t.Parallel()
	result := stripNullableFromType("null|undefined")
	assert.Equal(t, "", result)
}

func TestStripNullableFromType_NoUnion(t *testing.T) {
	t.Parallel()
	result := stripNullableFromType("string")
	assert.Equal(t, "string", result)
}
