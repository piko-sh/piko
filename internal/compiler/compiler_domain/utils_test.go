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
	"piko.sh/piko/internal/esbuild/helpers"
	"piko.sh/piko/internal/esbuild/js_ast"
)

func TestBase36(t *testing.T) {
	tests := []struct {
		name     string
		expected string
		input    int
	}{
		{
			name:     "zero",
			input:    0,
			expected: "0",
		},
		{
			name:     "single digit",
			input:    5,
			expected: "5",
		},
		{
			name:     "ten becomes a",
			input:    10,
			expected: "a",
		},
		{
			name:     "35 becomes z",
			input:    35,
			expected: "z",
		},
		{
			name:     "36 becomes 10",
			input:    36,
			expected: "10",
		},
		{
			name:     "37 becomes 11",
			input:    37,
			expected: "11",
		},
		{
			name:     "100 in base36",
			input:    100,
			expected: "2s",
		},
		{
			name:     "1000 in base36",
			input:    1000,
			expected: "rs",
		},
		{
			name:     "negative number",
			input:    -5,
			expected: "-5",
		},
		{
			name:     "negative larger number",
			input:    -100,
			expected: "-2s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := base36(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNodeKeyGenerator_NextKeyBase36(t *testing.T) {
	gen := &nodeKeyGenerator{}
	assert.Equal(t, "0", gen.nextKeyBase36())
	assert.Equal(t, "1", gen.nextKeyBase36())

	for i := 2; i < 10; i++ {
		gen.nextKeyBase36()
	}

	assert.Equal(t, "a", gen.nextKeyBase36())
}

func TestNodeKeyGenerator_Isolation(t *testing.T) {
	gen1 := &nodeKeyGenerator{}
	gen2 := &nodeKeyGenerator{}
	assert.Equal(t, "0", gen1.nextKeyBase36())
	assert.Equal(t, "0", gen2.nextKeyBase36())
	assert.Equal(t, "1", gen1.nextKeyBase36())
	assert.Equal(t, "1", gen2.nextKeyBase36())
}

func TestNormaliseWhitespace(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no whitespace changes needed",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "tabs to spaces",
			input:    "hello\tworld",
			expected: "hello world",
		},
		{
			name:     "newlines to spaces",
			input:    "hello\nworld",
			expected: "hello world",
		},
		{
			name:     "carriage returns to spaces",
			input:    "hello\rworld",
			expected: "hello world",
		},
		{
			name:     "multiple spaces to single",
			input:    "hello    world",
			expected: "hello world",
		},
		{
			name:     "mixed whitespace",
			input:    "hello\t\n\r   world",
			expected: "hello world",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only whitespace",
			input:    "   \t\n\r   ",
			expected: " ",
		},
		{
			name:     "windows line endings",
			input:    "line1\r\nline2",
			expected: "line1 line2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normaliseWhitespace(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildClassName_Utils(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple kebab case",
			input:    "my-component",
			expected: "MyComponentElement",
		},
		{
			name:     "pp prefix",
			input:    "pp-button",
			expected: "PpButtonElement",
		},
		{
			name:     "single word",
			input:    "simple",
			expected: "SimpleElement",
		},
		{
			name:     "multiple dashes",
			input:    "my-super-component",
			expected: "MySuperComponentElement",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "Element",
		},
		{
			name:     "only dashes",
			input:    "---",
			expected: "Element",
		},
		{
			name:     "leading dash",
			input:    "-component",
			expected: "ComponentElement",
		},
		{
			name:     "trailing dash",
			input:    "component-",
			expected: "ComponentElement",
		},
		{
			name:     "numbers in name",
			input:    "my-component-2",
			expected: "MyComponent2Element",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildClassName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEventRecord(t *testing.T) {
	t.Run("native event", func(t *testing.T) {
		record := eventRecord{
			EventID:  "click_1",
			IsNative: true,
		}
		assert.Equal(t, "click_1", record.EventID)
		assert.True(t, record.IsNative)
	})

	t.Run("custom event", func(t *testing.T) {
		record := eventRecord{
			EventID:  "custom_2",
			IsNative: false,
		}
		assert.Equal(t, "custom_2", record.EventID)
		assert.False(t, record.IsNative)
	})
}
func TestCreateStaticGetterFunction_ContentSurvivesTheJavaScriptRoundTrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		css  string
	}{
		{
			name: "icon font escape keeps its backslash",
			css:  `.icon:before{content:"\eb1c"}`,
		},
		{
			name: "escaped identifier selector survives",
			css:  `.w-1\/2{width:50%}`,
		},
		{
			name: "tailwind style variant selector survives",
			css:  `.md\:flex{display:flex}`,
		},
		{
			name: "trailing backslash does not run the string to the end",
			css:  `.a:before{content:"\\"}.b{color:red}`,
		},
		{
			name: "dollar brace is inert rather than a live substitution",
			css:  `.a:before{content:"${danger}"}`,
		},
		{
			name: "backtick is carried through",
			css:  ".a:before{content:\"`\"}",
		},
		{
			name: "literal non-ascii character survives",
			css:  `.a:before{content:"café"}`,
		},
		{
			name: "em dash escape survives",
			css:  `.a:before{content:"\2014"}`,
		},
		{
			name: "bell escape survives",
			css:  `.a:before{content:"\7"}`,
		},
		{
			name: "double quote inside the stylesheet survives",
			css:  `.a[data-x="y"]{color:red}`,
		},
		{
			name: "newline inside the stylesheet survives",
			css:  ".a{color:red}\n.b{color:blue}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			property := createStaticGetterFunction("css", tt.css)
			require.NotNil(t, property)

			str, isStr := property.Key.Data.(*js_ast.EString)
			require.True(t, isStr)
			assert.Equal(t, "css", helpers.UTF16ToString(str.Value))
			assert.Equal(t, js_ast.PropertyGetter, property.Kind)
			assert.True(t, property.Flags.Has(js_ast.PropertyIsStatic))

			fn, isFn := property.ValueOrNil.Data.(*js_ast.EFunction)
			require.True(t, isFn)
			require.Len(t, fn.Fn.Body.Block.Stmts, 1)
			ret, isReturn := fn.Fn.Body.Block.Stmts[0].Data.(*js_ast.SReturn)
			require.True(t, isReturn)
			value, isValue := ret.ValueOrNil.Data.(*js_ast.EString)
			require.True(t, isValue, "content must be carried as a string literal, never a template")
			assert.Equal(t, tt.css, helpers.UTF16ToString(value.Value),
				"the getter must carry the stylesheet byte for byte")
		})
	}
}
