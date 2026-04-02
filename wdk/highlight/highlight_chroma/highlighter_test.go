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

package highlight_chroma

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	assert.Equal(t, "dracula", config.Style)
	assert.True(t, config.WithClasses)
	assert.False(t, config.WithLineNumbers)
	assert.False(t, config.LineNumbersInTable)
	assert.Equal(t, 4, config.TabWidth)
}

func TestNewChromaHighlighter(t *testing.T) {
	t.Run("creates highlighter with default config", func(t *testing.T) {
		h := NewChromaHighlighter(DefaultConfig())

		require.NotNil(t, h)
		assert.NotNil(t, h.style)
		assert.NotNil(t, h.formatter)
	})

	t.Run("creates highlighter with custom style", func(t *testing.T) {
		config := Config{Style: "monokai", TabWidth: 4}

		h := NewChromaHighlighter(config)

		require.NotNil(t, h)
		assert.NotNil(t, h.style)
	})

	t.Run("falls back when style is empty", func(t *testing.T) {
		config := Config{}

		h := NewChromaHighlighter(config)

		require.NotNil(t, h)
		assert.NotNil(t, h.style)
	})

	t.Run("falls back for unknown style", func(t *testing.T) {
		config := Config{Style: "nonexistent-style-xyz"}

		h := NewChromaHighlighter(config)

		require.NotNil(t, h)
		assert.NotNil(t, h.style)
	})

	t.Run("defaults tab width when zero", func(t *testing.T) {
		config := Config{Style: "dracula", TabWidth: 0}

		h := NewChromaHighlighter(config)

		require.NotNil(t, h)
	})
}

func TestHighlighter_Highlight(t *testing.T) {
	h := NewChromaHighlighter(DefaultConfig())

	t.Run("highlights known language", func(t *testing.T) {
		code := `func main() { fmt.Println("hello") }`

		result := h.Highlight(code, "go")

		assert.NotEmpty(t, result)
		assert.Contains(t, result, "chroma")
		assert.NotContains(t, result, `class="language-go"`)
	})

	t.Run("handles unknown language with fallback lexer", func(t *testing.T) {
		code := "some code here"

		result := h.Highlight(code, "unknown-lang-xyz")

		assert.NotEmpty(t, result)
	})

	t.Run("highlights empty code", func(t *testing.T) {
		result := h.Highlight("", "go")

		assert.NotEmpty(t, result)
	})

	t.Run("highlights JavaScript", func(t *testing.T) {
		code := `const x = 42;`

		result := h.Highlight(code, "javascript")

		assert.NotEmpty(t, result)
		assert.Contains(t, result, "chroma")
	})

	t.Run("highlights Python", func(t *testing.T) {
		code := `definition hello(): print("world")`

		result := h.Highlight(code, "python")

		assert.NotEmpty(t, result)
		assert.Contains(t, result, "chroma")
	})
}

func TestPlainCodeBlock(t *testing.T) {
	t.Run("wraps code in pre/code tags", func(t *testing.T) {
		result := plainCodeBlock("hello world", "go")

		assert.Equal(t, `<pre><code class="language-go">hello world</code></pre>`, result)
	})

	t.Run("escapes HTML special characters", func(t *testing.T) {
		result := plainCodeBlock(`<div>"hello" & 'world'</div>`, "html")

		assert.Contains(t, result, "&lt;div&gt;")
		assert.Contains(t, result, "&amp;")
		assert.Contains(t, result, "&#34;hello&#34;")
		assert.Contains(t, result, "&#39;world&#39;")
	})

	t.Run("escapes language name", func(t *testing.T) {
		result := plainCodeBlock("code", `"><script>alert(1)</script>`)

		assert.Contains(t, result, "&lt;script&gt;")
		assert.NotContains(t, result, "<script>")
	})

	t.Run("omits language class when empty", func(t *testing.T) {
		result := plainCodeBlock("hello", "")

		assert.Equal(t, "<pre><code>hello</code></pre>", result)
	})
}
