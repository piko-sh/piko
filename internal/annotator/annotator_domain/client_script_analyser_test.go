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
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnalyseClientScript(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		expectedFunctions     map[string]ExportedFunction
		name                  string
		source                string
		sourcePath            string
		expectNil             bool
		expectDefaultExported bool
	}{
		{
			name:       "empty source returns nil",
			source:     "",
			sourcePath: "test.ts",
			expectNil:  true,
		},
		{
			name:       "named function export",
			source:     `export function handleClick() {}`,
			sourcePath: "test.ts",
			expectedFunctions: map[string]ExportedFunction{
				"handleClick": {Name: "handleClick", IsAsync: false},
			},
		},
		{
			name:       "async function export",
			source:     `export async function fetchData() {}`,
			sourcePath: "test.ts",
			expectedFunctions: map[string]ExportedFunction{
				"fetchData": {Name: "fetchData", IsAsync: true},
			},
		},
		{
			name: "multiple named function exports",
			source: `export function alpha() {}
export function beta() {}`,
			sourcePath: "test.ts",
			expectedFunctions: map[string]ExportedFunction{
				"alpha": {Name: "alpha", IsAsync: false},
				"beta":  {Name: "beta", IsAsync: false},
			},
		},
		{
			name: "re-export clause",
			source: `function foo() {}
function bar() {}
export { foo, bar }`,
			sourcePath: "test.ts",
			expectedFunctions: map[string]ExportedFunction{
				"foo": {Name: "foo", IsAsync: false},
				"bar": {Name: "bar", IsAsync: false},
			},
		},
		{
			name:       "default function export with name",
			source:     `export default function main() {}`,
			sourcePath: "test.ts",
			expectedFunctions: map[string]ExportedFunction{
				"main": {Name: "main", IsAsync: false},
			},
		},
		{
			name:       "arrow function export via const",
			source:     `export const handler = () => {}`,
			sourcePath: "test.ts",
			expectedFunctions: map[string]ExportedFunction{
				"handler": {Name: "handler", IsAsync: false},
			},
		},
		{
			name:       "async arrow function export",
			source:     `export const loader = async () => {}`,
			sourcePath: "test.ts",
			expectedFunctions: map[string]ExportedFunction{
				"loader": {Name: "loader", IsAsync: true},
			},
		},
		{
			name:              "non-function const export is not captured",
			source:            `export const value = 42`,
			sourcePath:        "test.ts",
			expectedFunctions: map[string]ExportedFunction{},
		},
		{
			name:       "non-exported function is still captured",
			source:     `function onClick() {}`,
			sourcePath: "test.ts",
			expectedFunctions: map[string]ExportedFunction{
				"onClick": {Name: "onClick", IsAsync: false},
			},
		},
		{
			name:       "non-exported const arrow is still captured",
			source:     `const submit = () => {}`,
			sourcePath: "test.ts",
			expectedFunctions: map[string]ExportedFunction{
				"submit": {Name: "submit", IsAsync: false},
			},
		},
		{
			name: "mixed exports combine named default and clause",
			source: `export function alpha() {}
export default function beta() {}
function gamma() {}
export { gamma }`,
			sourcePath: "test.ts",
			expectedFunctions: map[string]ExportedFunction{
				"alpha": {Name: "alpha", IsAsync: false},
				"beta":  {Name: "beta", IsAsync: false},
				"gamma": {Name: "gamma", IsAsync: false},
			},
		},
		{
			name:       "async default function export",
			source:     `export default async function load() {}`,
			sourcePath: "test.ts",
			expectedFunctions: map[string]ExportedFunction{
				"load": {Name: "load", IsAsync: true},
			},
		},
		{
			name:       "function expression via const",
			source:     `export const render = function() {}`,
			sourcePath: "test.ts",
			expectedFunctions: map[string]ExportedFunction{
				"render": {Name: "render", IsAsync: false},
			},
		},
		{
			name:       "async function expression via const",
			source:     `export const render = async function() {}`,
			sourcePath: "test.ts",
			expectedFunctions: map[string]ExportedFunction{
				"render": {Name: "render", IsAsync: true},
			},
		},
		{
			name:       "multiple const declarations in one statement",
			source:     `export const a = () => {}, b = async () => {}`,
			sourcePath: "test.ts",
			expectedFunctions: map[string]ExportedFunction{
				"a": {Name: "a", IsAsync: false},
				"b": {Name: "b", IsAsync: true},
			},
		},
		{
			name:              "non-function let export is not captured",
			source:            `export let counter = 0`,
			sourcePath:        "test.ts",
			expectedFunctions: map[string]ExportedFunction{},
		},
		{
			name:              "string const export is not captured",
			source:            `export const label = "hello"`,
			sourcePath:        "test.ts",
			expectedFunctions: map[string]ExportedFunction{},
		},
		{
			name:       "typescript typed function is captured",
			source:     `export function greet(name: string): void {}`,
			sourcePath: "test.ts",
			expectedFunctions: map[string]ExportedFunction{
				"greet": {
					Name:    "greet",
					IsAsync: false,
					Params: []ParamInfo{
						{Name: "name", TypeName: "string", Category: categoryString},
					},
				},
			},
		},
		{
			name:              "anonymous default export is not captured as a named function",
			source:            `export default function() {}`,
			sourcePath:        "test.ts",
			expectedFunctions: map[string]ExportedFunction{},
		},
		{
			name:              "destructured const export is not captured",
			source:            `export const { a, b } = getValues()`,
			sourcePath:        "test.ts",
			expectedFunctions: map[string]ExportedFunction{},
		},
		{
			name:              "class export is not captured as a function",
			source:            `export class MyComponent {}`,
			sourcePath:        "test.ts",
			expectedFunctions: map[string]ExportedFunction{},
		},
		{
			name:       "let function assignment is captured",
			source:     `let handler = () => {}`,
			sourcePath: "test.ts",
			expectedFunctions: map[string]ExportedFunction{
				"handler": {Name: "handler", IsAsync: false},
			},
		},
		{
			name:       "var function assignment is captured",
			source:     `var handler = function() {}`,
			sourcePath: "test.ts",
			expectedFunctions: map[string]ExportedFunction{
				"handler": {Name: "handler", IsAsync: false},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := AnalyseClientScript(tc.source, tc.sourcePath)

			if tc.expectNil {
				assert.Nil(t, result)
				return
			}

			require.NotNil(t, result)
			assert.Equal(t, tc.source, result.ScriptContent)
			assert.Equal(t, tc.sourcePath, result.SourcePath)
			assert.Equal(t, tc.expectedFunctions, result.ExportedFunctions)
		})
	}
}

func TestAnalyseClientScript_StoresSourceMetadata(t *testing.T) {
	t.Parallel()

	source := `export function test() {}`
	path := "components/button.ts"

	result := AnalyseClientScript(source, path)

	require.NotNil(t, result)
	assert.Equal(t, source, result.ScriptContent)
	assert.Equal(t, path, result.SourcePath)
}

func TestClientScriptExports_HasExport(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		exports  *ClientScriptExports
		lookup   string
		expected bool
	}{
		{
			name:     "nil receiver returns false",
			exports:  nil,
			lookup:   "anything",
			expected: false,
		},
		{
			name: "nil functions map returns false",
			exports: &ClientScriptExports{
				ExportedFunctions: nil,
			},
			lookup:   "anything",
			expected: false,
		},
		{
			name: "existing export returns true",
			exports: &ClientScriptExports{
				ExportedFunctions: map[string]ExportedFunction{
					"handleClick": {Name: "handleClick", IsAsync: false},
				},
			},
			lookup:   "handleClick",
			expected: true,
		},
		{
			name: "non-existing export returns false",
			exports: &ClientScriptExports{
				ExportedFunctions: map[string]ExportedFunction{
					"handleClick": {Name: "handleClick", IsAsync: false},
				},
			},
			lookup:   "nonExistent",
			expected: false,
		},
		{
			name: "empty map returns false",
			exports: &ClientScriptExports{
				ExportedFunctions: map[string]ExportedFunction{},
			},
			lookup:   "anything",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := tc.exports.HasExport(tc.lookup)

			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestClientScriptExports_ExportNames(t *testing.T) {
	t.Parallel()

	t.Run("nil receiver returns nil", func(t *testing.T) {
		t.Parallel()

		var exports *ClientScriptExports
		result := exports.ExportNames()

		assert.Nil(t, result)
	})

	t.Run("nil functions map returns nil", func(t *testing.T) {
		t.Parallel()

		exports := &ClientScriptExports{
			ExportedFunctions: nil,
		}
		result := exports.ExportNames()

		assert.Nil(t, result)
	})

	t.Run("empty functions map returns empty slice", func(t *testing.T) {
		t.Parallel()

		exports := &ClientScriptExports{
			ExportedFunctions: map[string]ExportedFunction{},
		}
		result := exports.ExportNames()

		assert.Empty(t, result)
		assert.NotNil(t, result)
	})

	t.Run("returns all export names", func(t *testing.T) {
		t.Parallel()

		exports := &ClientScriptExports{
			ExportedFunctions: map[string]ExportedFunction{
				"alpha":   {Name: "alpha", IsAsync: false},
				"beta":    {Name: "beta", IsAsync: true},
				"charlie": {Name: "charlie", IsAsync: false},
			},
		}
		result := exports.ExportNames()

		sort.Strings(result)
		assert.Equal(t, []string{"alpha", "beta", "charlie"}, result)
	})

	t.Run("single export returns single element slice", func(t *testing.T) {
		t.Parallel()

		exports := &ClientScriptExports{
			ExportedFunctions: map[string]ExportedFunction{
				"only": {Name: "only", IsAsync: false},
			},
		}
		result := exports.ExportNames()

		assert.Equal(t, []string{"only"}, result)
	})
}

func TestAnalyseClientScript_EndToEnd(t *testing.T) {
	t.Parallel()

	t.Run("full component script", func(t *testing.T) {
		t.Parallel()

		source := `
import { someUtil } from './utils';

const CONSTANT = 42;

export function handleClick() {
    console.log('clicked');
}

export async function fetchData() {
    const res = await fetch('/api/data');
    return res.json();
}

export const onSubmit = async () => {
    await someUtil();
};

function helperFunction() {
    return CONSTANT;
}
`
		result := AnalyseClientScript(source, "component.ts")

		require.NotNil(t, result)

		assert.True(t, result.HasExport("handleClick"))
		assert.False(t, result.ExportedFunctions["handleClick"].IsAsync)

		assert.True(t, result.HasExport("fetchData"))
		assert.True(t, result.ExportedFunctions["fetchData"].IsAsync)

		assert.True(t, result.HasExport("onSubmit"))
		assert.True(t, result.ExportedFunctions["onSubmit"].IsAsync)

		assert.True(t, result.HasExport("helperFunction"))
		assert.False(t, result.ExportedFunctions["helperFunction"].IsAsync)

		assert.False(t, result.HasExport("CONSTANT"))

		names := result.ExportNames()
		sort.Strings(names)
		assert.Equal(t, []string{"fetchData", "handleClick", "helperFunction", "onSubmit"}, names)
	})

	t.Run("export clause with aliased names", func(t *testing.T) {
		t.Parallel()

		source := `
function internalName() {}
export { internalName as publicName }
`
		result := AnalyseClientScript(source, "test.ts")

		require.NotNil(t, result)

		assert.True(t, result.HasExport("publicName"))

		assert.True(t, result.HasExport("internalName"))
	})

	t.Run("async non-exported function", func(t *testing.T) {
		t.Parallel()

		source := `async function backgroundTask() {}`

		result := AnalyseClientScript(source, "test.ts")

		require.NotNil(t, result)
		assert.True(t, result.HasExport("backgroundTask"))
		assert.True(t, result.ExportedFunctions["backgroundTask"].IsAsync)
	})
}
