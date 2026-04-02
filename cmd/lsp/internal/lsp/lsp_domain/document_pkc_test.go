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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/protocol"
)

const testPKCContent = `<template name="test-comp">
<div class="wrapper">
<span p-text="state.message"></span>
<button p-on:click="handleClick">Click</button>
<input _ref="myInput" />
</div>
</template>

<script lang="ts">
const state = {
  message: "hello" as string,
  count: 0 as number,
  active: true as boolean
};

function handleClick(event) {
  state.count++;
}

function reset() {
  state.count = 0;
}
</script>

<style>
.wrapper {
  padding: 1rem;
}
</style>`

func newTestPKCDoc() *document {
	return &document{
		Content: []byte(testPKCContent),
		URI:     "file:///test.pkc",
	}
}

func TestIsPKCFile(t *testing.T) {
	tests := []struct {
		name     string
		uri      protocol.DocumentURI
		expected bool
	}{
		{name: "pkc file", uri: "file:///test.pkc", expected: true},
		{name: "pk file", uri: "file:///test.pk", expected: false},
		{name: "empty uri", uri: "", expected: false},
		{name: "pkc in subdirectory", uri: "file:///path/to/comp.pkc", expected: true},
		{name: "pkc-like but not pkc", uri: "file:///test.pkcc", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			document := &document{URI: tt.uri}
			assert.Equal(t, tt.expected, document.isPKCFile())
		})
	}
}

func TestPKCMetadata_StateProperties(t *testing.T) {
	document := newTestPKCDoc()
	meta := document.getPKCMetadata()
	require.NotNil(t, meta)

	assert.Equal(t, 3, len(meta.StateProperties))

	message := meta.StateProperties["message"]
	require.NotNil(t, message)
	assert.Equal(t, "string", message.JSType)

	count := meta.StateProperties["count"]
	require.NotNil(t, count)
	assert.Equal(t, "number", count.JSType)

	active := meta.StateProperties["active"]
	require.NotNil(t, active)
	assert.Equal(t, "boolean", active.JSType)
}

func TestPKCMetadata_Functions(t *testing.T) {
	document := newTestPKCDoc()
	meta := document.getPKCMetadata()
	require.NotNil(t, meta)

	assert.Equal(t, 2, len(meta.Functions))

	handleClick := meta.Functions["handleClick"]
	require.NotNil(t, handleClick)
	assert.Equal(t, []string{"event"}, handleClick.ParamNames)

	resetFunction := meta.Functions["reset"]
	require.NotNil(t, resetFunction)
	assert.Empty(t, resetFunction.ParamNames)
}

func TestPKCMetadata_ArrowFunctions(t *testing.T) {
	content := `<template>
<button p-on:click="handleClick">Click</button>
</template>

<script lang="ts" name="arrow-comp">
const state = {
  count: 0 as number
};

const handleClick = (ev) => {
  state.count++;
};

const reset = () => {
  state.count = 0;
};

const processData = function(data) {
  return data;
};
</script>`

	document := &document{
		Content: []byte(content),
		URI:     "file:///test.pkc",
	}

	meta := document.getPKCMetadata()
	require.NotNil(t, meta)

	assert.Equal(t, 3, len(meta.Functions), "expected 3 functions (2 arrow + 1 function expr)")

	handleClick := meta.Functions["handleClick"]
	require.NotNil(t, handleClick, "arrow function handleClick not found")
	assert.Equal(t, []string{"ev"}, handleClick.ParamNames)

	resetFunction := meta.Functions["reset"]
	require.NotNil(t, resetFunction, "arrow function reset not found")
	assert.Empty(t, resetFunction.ParamNames)

	processFunction := meta.Functions["processData"]
	require.NotNil(t, processFunction, "function expression processData not found")
	assert.Equal(t, []string{"data"}, processFunction.ParamNames)
}

func TestPKCNavigation_ArrowHandler(t *testing.T) {
	content := `<template>
<button p-on:click="handleClick">Click</button>
</template>

<script lang="ts" name="arrow-comp">
const state = {};

const handleClick = (ev) => {
  console.log(ev);
};
</script>`

	document := &document{
		Content: []byte(content),
		URI:     "file:///test.pkc",
	}

	position := protocol.Position{Line: 1, Character: 25}
	locations, err := document.GetPKDefinition(context.Background(), position)
	require.NoError(t, err)
	require.NotEmpty(t, locations, "expected go-to-definition for arrow function handler")

	assert.Equal(t, protocol.DocumentURI("file:///test.pkc"), locations[0].URI)
}

func TestPKCDiagnostics_ArrowFunctionNoFalsePositive(t *testing.T) {
	content := `<template>
<button p-on:click="handleClick">Click</button>
</template>

<script lang="ts">
const state = {};

const handleClick = (ev) => {
  console.log(ev);
};
</script>`

	document := &document{
		Content: []byte(content),
		URI:     "file:///test.pkc",
	}

	diagnostics := document.getPKCDiagnostics()
	assert.Empty(t, diagnostics, "arrow function handler should not produce unknown handler diagnostic")
}

func TestPKCMetadata_CSSClasses(t *testing.T) {
	document := newTestPKCDoc()
	meta := document.getPKCMetadata()
	require.NotNil(t, meta)

	assert.Contains(t, meta.CSSClasses, "wrapper")
}

func TestPKCMetadata_Refs(t *testing.T) {
	document := newTestPKCDoc()
	meta := document.getPKCMetadata()
	require.NotNil(t, meta)

	assert.Contains(t, meta.Refs, "myInput")
}

func TestPKCMetadata_ComponentName(t *testing.T) {
	document := newTestPKCDoc()
	meta := document.getPKCMetadata()
	require.NotNil(t, meta)

	assert.Equal(t, "test-comp", meta.ComponentName)
}

func TestPKCMetadata_NonPKCFile(t *testing.T) {
	document := &document{
		Content: []byte(testPKCContent),
		URI:     "file:///test.pk",
	}
	meta := document.getPKCMetadata()
	assert.Nil(t, meta)
}

func TestPKCMetadata_EmptyContent(t *testing.T) {
	document := &document{
		Content: []byte{},
		URI:     "file:///test.pkc",
	}
	meta := document.getPKCMetadata()
	require.NotNil(t, meta)
	assert.Empty(t, meta.StateProperties)
	assert.Empty(t, meta.Functions)
	assert.Empty(t, meta.CSSClasses)
}

func TestCheckPKCStatePropertyContext(t *testing.T) {
	document := &document{URI: "file:///test.pkc"}

	tests := []struct {
		name     string
		line     string
		cursor   int
		expected string
	}{
		{
			name:     "state.message in directive",
			line:     `p-text="state.message"`,
			cursor:   18,
			expected: "message",
		},
		{
			name:     "state.count in interpolation",
			line:     `{{ state.count }}`,
			cursor:   12,
			expected: "count",
		},
		{
			name:     "cursor at start of property",
			line:     `p-text="state.active"`,
			cursor:   14,
			expected: "active",
		},
		{
			name:     "no state prefix",
			line:     `p-text="foo.bar"`,
			cursor:   12,
			expected: "",
		},
		{
			name:     "updatestate not matched",
			line:     `p-text="updatestate.foo"`,
			cursor:   22,
			expected: "",
		},
		{
			name:     "cursor before state",
			line:     `p-text="state.message"`,
			cursor:   5,
			expected: "",
		},
		{
			name:     "cursor on dot",
			line:     `p-text="state.message"`,
			cursor:   13,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			position := protocol.Position{Line: 0, Character: uint32(tt.cursor)}
			ctx := document.checkPKCStatePropertyContext(tt.line, tt.cursor, position)

			if tt.expected == "" {
				assert.Nil(t, ctx)
			} else {
				require.NotNil(t, ctx)
				assert.Equal(t, tt.expected, ctx.Name)
				assert.Equal(t, PKDefPKCStateProperty, ctx.Kind)
			}
		})
	}
}

func TestPKCNavigation_StateProperty(t *testing.T) {
	document := newTestPKCDoc()

	position := protocol.Position{Line: 2, Character: 22}
	locations, err := document.GetPKDefinition(context.Background(), position)
	require.NoError(t, err)
	require.NotEmpty(t, locations)

	assert.Equal(t, protocol.DocumentURI("file:///test.pkc"), locations[0].URI)
}

func TestPKCNavigation_Handler(t *testing.T) {
	document := newTestPKCDoc()

	position := protocol.Position{Line: 3, Character: 25}
	locations, err := document.GetPKDefinition(context.Background(), position)
	require.NoError(t, err)
	require.NotEmpty(t, locations)

	assert.Equal(t, protocol.DocumentURI("file:///test.pkc"), locations[0].URI)
}

func TestPKCNavigation_CSSClass(t *testing.T) {
	document := newTestPKCDoc()

	position := protocol.Position{Line: 1, Character: 14}
	locations, err := document.GetPKDefinition(context.Background(), position)
	require.NoError(t, err)
	require.NotEmpty(t, locations)

	assert.Equal(t, protocol.DocumentURI("file:///test.pkc"), locations[0].URI)
}

func TestPKCHover_StateProperty(t *testing.T) {
	document := newTestPKCDoc()

	position := protocol.Position{Line: 2, Character: 22}
	hover, err := document.GetPKHoverInfo(context.Background(), position)
	require.NoError(t, err)
	require.NotNil(t, hover)

	assert.Contains(t, hover.Contents.Value, "state.message")
	assert.Contains(t, hover.Contents.Value, "string")
}

func TestPKCHover_Handler(t *testing.T) {
	document := newTestPKCDoc()

	position := protocol.Position{Line: 3, Character: 25}
	hover, err := document.GetPKHoverInfo(context.Background(), position)
	require.NoError(t, err)
	require.NotNil(t, hover)

	assert.Contains(t, hover.Contents.Value, "handleClick")
	assert.Contains(t, hover.Contents.Value, "event")
}

func TestPKCCompletion_StateFields(t *testing.T) {
	content := `<template>
<span p-text="state."></span>
</template>

<script lang="ts">
const state = {
  message: "hello" as string,
  count: 0 as number
};
</script>`

	document := &document{
		Content: []byte(content),
		URI:     "file:///test.pkc",
	}

	position := protocol.Position{Line: 1, Character: 20}
	result, err := document.GetCompletions(context.Background(), position)
	require.NoError(t, err)
	require.NotNil(t, result)

	labels := make([]string, 0, len(result.Items))
	for _, item := range result.Items {
		labels = append(labels, item.Label)
	}

	assert.Contains(t, labels, "message")
	assert.Contains(t, labels, "count")
}

func TestPKCCompletion_Directives(t *testing.T) {
	content := `<template>
<span p-></span>
</template>

<script lang="ts">
const state = {};
</script>`

	document := &document{
		Content: []byte(content),
		URI:     "file:///test.pkc",
	}

	position := protocol.Position{Line: 1, Character: 8}
	result, err := document.GetCompletions(context.Background(), position)
	require.NoError(t, err)
	require.NotNil(t, result)

	labels := make([]string, 0, len(result.Items))
	for _, item := range result.Items {
		labels = append(labels, item.Label)
	}

	assert.Contains(t, labels, "if")
	assert.Contains(t, labels, "for")
	assert.Contains(t, labels, "on")
}

func TestPKCCompletion_EventHandlers(t *testing.T) {
	content := `<template>
<button p-on:click=""></button>
</template>

<script lang="ts">
const state = {};

function handleClick() {}
function handleSubmit() {}
</script>`

	document := &document{
		Content: []byte(content),
		URI:     "file:///test.pkc",
	}

	position := protocol.Position{Line: 1, Character: 20}
	result, err := document.GetCompletions(context.Background(), position)
	require.NoError(t, err)
	require.NotNil(t, result)

	labels := make([]string, 0, len(result.Items))
	for _, item := range result.Items {
		labels = append(labels, item.Label)
	}

	assert.Contains(t, labels, "handleClick")
	assert.Contains(t, labels, "handleSubmit")
}

func TestPKCDocumentSymbols(t *testing.T) {
	document := newTestPKCDoc()

	symbols, err := document.GetDocumentSymbols()
	require.NoError(t, err)
	require.NotEmpty(t, symbols)

	var scriptSymbol *protocol.DocumentSymbol
	for _, sym := range symbols {
		ds, ok := sym.(protocol.DocumentSymbol)
		require.True(t, ok, "expected symbol to be protocol.DocumentSymbol")
		if ds.Name == "<script>" {
			scriptSymbol = &ds
			break
		}
	}

	require.NotNil(t, scriptSymbol, "expected <script> symbol")
	require.NotEmpty(t, scriptSymbol.Children)

	childNames := make([]string, 0, len(scriptSymbol.Children))
	for _, child := range scriptSymbol.Children {
		childNames = append(childNames, child.Name)
	}

	assert.Contains(t, childNames, "state")
	assert.Contains(t, childNames, "handleClick")
	assert.Contains(t, childNames, "reset")
}

func TestPKCDocumentSymbols_StyleSection(t *testing.T) {
	document := newTestPKCDoc()

	symbols, err := document.GetDocumentSymbols()
	require.NoError(t, err)

	var styleSymbol *protocol.DocumentSymbol
	for _, sym := range symbols {
		ds, ok := sym.(protocol.DocumentSymbol)
		require.True(t, ok, "expected symbol to be protocol.DocumentSymbol")
		if ds.Name == "<style>" {
			styleSymbol = &ds
			break
		}
	}

	require.NotNil(t, styleSymbol, "expected <style> symbol")

	childNames := make([]string, 0, len(styleSymbol.Children))
	for _, child := range styleSymbol.Children {
		childNames = append(childNames, child.Name)
	}

	assert.Contains(t, childNames, ".wrapper")
}

func TestPKCDiagnostics_UnknownStateProperty(t *testing.T) {
	content := `<template>
<span p-text="state.nonexistent"></span>
</template>

<script lang="ts">
const state = {
  message: "hello" as string
};
</script>`

	document := &document{
		Content: []byte(content),
		URI:     "file:///test.pkc",
	}

	diagnostics := document.getPKCDiagnostics()
	require.NotEmpty(t, diagnostics)

	found := false
	for _, d := range diagnostics {
		if strings.Contains(d.Message, "nonexistent") {
			found = true
			break
		}
	}
	assert.True(t, found, "expected diagnostic for unknown state property 'nonexistent'")
}

func TestPKCDiagnostics_UnknownHandler(t *testing.T) {
	content := `<template>
<button p-on:click="unknownHandler">Click</button>
</template>

<script lang="ts">
const state = {};

function handleClick() {}
</script>`

	document := &document{
		Content: []byte(content),
		URI:     "file:///test.pkc",
	}

	diagnostics := document.getPKCDiagnostics()
	require.NotEmpty(t, diagnostics)

	found := false
	for _, d := range diagnostics {
		if strings.Contains(d.Message, "unknownHandler") {
			found = true
			break
		}
	}
	assert.True(t, found, "expected diagnostic for unknown handler 'unknownHandler'")
}

func TestPKCDiagnostics_NoFalsePositives(t *testing.T) {
	content := `<template>
<span p-text="state.message"></span>
<button p-on:click="handleClick">Click</button>
</template>

<script lang="ts">
const state = {
  message: "hello" as string
};

function handleClick() {}
</script>`

	document := &document{
		Content: []byte(content),
		URI:     "file:///test.pkc",
	}

	diagnostics := document.getPKCDiagnostics()
	assert.Empty(t, diagnostics)
}

func TestPKCCSSClassDefinitions(t *testing.T) {
	document := newTestPKCDoc()

	definitions := document.findCSSClassDefinitions()
	require.NotNil(t, definitions)
	assert.Contains(t, definitions, "wrapper")
}

func TestPKCCompletion_EmptyPKCFile(t *testing.T) {
	document := &document{
		Content: []byte{},
		URI:     "file:///test.pkc",
	}

	position := protocol.Position{Line: 0, Character: 0}
	result, err := document.GetCompletions(context.Background(), position)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Empty(t, result.Items)
}

func TestGetPKCTypeString(t *testing.T) {
	tests := []struct {
		name     string
		prop     pkcStateProperty
		expected string
	}{
		{
			name:     "string type",
			prop:     pkcStateProperty{JSType: "string"},
			expected: "string",
		},
		{
			name:     "array with element type",
			prop:     pkcStateProperty{JSType: "array", ElementType: "string"},
			expected: "Array<string>",
		},
		{
			name:     "array without element type",
			prop:     pkcStateProperty{JSType: "array"},
			expected: "Array",
		},
		{
			name:     "map type",
			prop:     pkcStateProperty{JSType: "object", KeyType: "string", ValueType: "number"},
			expected: "Map<string, number>",
		},
		{
			name:     "plain object",
			prop:     pkcStateProperty{JSType: "object"},
			expected: "Object",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getPKCTypeString(&tt.prop)
			assert.Equal(t, tt.expected, result)
		})
	}
}
