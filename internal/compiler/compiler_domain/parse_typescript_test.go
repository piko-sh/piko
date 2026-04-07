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
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTypeScriptStrict_DuplicateDeclarations(t *testing.T) {
	t.Parallel()
	parser := NewTypeScriptParser()

	t.Run("returns error for duplicate let declarations", func(t *testing.T) {
		t.Parallel()
		source := `let v = 1; let v = 2;`

		ast, err := parser.ParseTypeScriptStrict(source, "test.ts")

		require.Error(t, err)
		assert.Nil(t, ast)
		assert.Contains(t, err.Error(), "has already been declared")
		assert.Contains(t, err.Error(), "line 1")
	})

	t.Run("returns error for duplicate const declarations", func(t *testing.T) {
		t.Parallel()
		source := `const x = 1; const x = 2;`

		ast, err := parser.ParseTypeScriptStrict(source, "test.ts")

		require.Error(t, err)
		assert.Nil(t, ast)
		assert.Contains(t, err.Error(), "has already been declared")
	})

	t.Run("returns error for duplicate class declarations", func(t *testing.T) {
		t.Parallel()

		source := `let Foo = 1; let Foo = 2;`

		ast, err := parser.ParseTypeScriptStrict(source, "test.ts")

		require.Error(t, err)
		assert.Nil(t, ast)
		assert.Contains(t, err.Error(), "has already been declared")
	})

	t.Run("includes filename in error message", func(t *testing.T) {
		t.Parallel()
		source := `let v = 1; let v = 2;`

		_, err := parser.ParseTypeScriptStrict(source, "my-component.ts")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "my-component.ts")
	})

	t.Run("reports multiple errors joined", func(t *testing.T) {
		t.Parallel()
		source := `let a = 1; let a = 2; let b = 3; let b = 4;`

		_, err := parser.ParseTypeScriptStrict(source, "test.ts")

		require.Error(t, err)

		assert.Contains(t, err.Error(), `"a"`)
		assert.Contains(t, err.Error(), `"b"`)
	})

	t.Run("includes line and column in multi-line errors", func(t *testing.T) {
		t.Parallel()
		source := "let x = 1;\nlet y = 2;\nlet x = 3;\n"

		_, err := parser.ParseTypeScriptStrict(source, "test.ts")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "line 3")
	})

	t.Run("includes source line and caret in diagnostic", func(t *testing.T) {
		t.Parallel()
		source := "let count = 0;\nlet count = 1;\n"

		_, err := parser.ParseTypeScriptStrict(source, "test.ts")

		require.Error(t, err)
		errText := err.Error()

		assert.Contains(t, errText, "let count = 1;")

		assert.Contains(t, errText, "^")

		assert.Contains(t, errText, "2 |")
	})
}

func TestParseTypeScriptStrict_ValidCode(t *testing.T) {
	t.Parallel()
	parser := NewTypeScriptParser()

	t.Run("succeeds for valid TypeScript", func(t *testing.T) {
		t.Parallel()
		source := `
class MyElement extends PPElement {
	count: number = 0;
	increment() { this.count++; }
}
`

		ast, err := parser.ParseTypeScriptStrict(source, "test.ts")

		require.NoError(t, err)
		require.NotNil(t, ast)
	})

	t.Run("succeeds for empty source", func(t *testing.T) {
		t.Parallel()

		ast, err := parser.ParseTypeScriptStrict("", "test.ts")

		require.NoError(t, err)
		require.NotNil(t, ast)
	})
}

func TestParseTypeScript_Lenient(t *testing.T) {
	t.Parallel()
	parser := NewTypeScriptParser()

	t.Run("ignores duplicate declarations in lenient mode", func(t *testing.T) {
		t.Parallel()
		source := `let v = 1; let v = 2;`

		ast, err := parser.ParseTypeScript(source, "snippet.ts")

		require.NoError(t, err)
		require.NotNil(t, ast)
	})

	t.Run("ignores super outside class in lenient mode", func(t *testing.T) {
		t.Parallel()
		source := `{ super(); }`

		ast, err := parser.ParseTypeScript(source, "snippet.ts")

		require.NoError(t, err)
		require.NotNil(t, ast)
	})
}

func TestCompileSFC_DuplicateDeclarationError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("fails when script contains duplicate declarations", func(t *testing.T) {
		t.Parallel()
		rawSFC := []byte(`<script>
let v = 1;
let v = 2;
</script>
<template name="bad-component"><div>Test</div></template>`)

		artefact, err := compileSFC(ctx, "bad-component.pkc", rawSFC, "", nil)

		require.Error(t, err)
		assert.Nil(t, artefact)
		assert.Contains(t, err.Error(), "has already been declared")
	})

	t.Run("error propagates through processJavaScript", func(t *testing.T) {
		t.Parallel()
		rawSFC := []byte(`<script>
const x = "hello";
const x = "world";
</script>
<template name="dup-test"><div>Test</div></template>`)

		artefact, err := compileSFC(ctx, "dup-test.pkc", rawSFC, "", nil)

		require.Error(t, err)
		assert.Nil(t, artefact)
		assert.Contains(t, err.Error(), "javascript")
	})

	t.Run("valid SFC still compiles successfully", func(t *testing.T) {
		t.Parallel()
		rawSFC := []byte(`<script>
class GoodComponentElement extends PPElement {
	count = 0;
}
</script>
<template name="good-component"><div>Works</div></template>`)

		artefact, err := compileSFC(ctx, "good-component.pkc", rawSFC, "", nil)

		require.NoError(t, err)
		require.NotNil(t, artefact)
		assert.Equal(t, "good-component", artefact.TagName)
	})
}
