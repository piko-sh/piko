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

package formatter_domain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatter_Idempotency(t *testing.T) {
	testCases := []struct {
		name   string
		input  string
		format FileFormat
	}{
		{
			name: "simple PK template",
			input: `<template>
  <div><p>Hello</p></div>
</template>`,
			format: FormatPK,
		},
		{
			name: "nested elements",
			input: `<template>
  <div><ul><li>Item</li></ul></div>
</template>`,
			format: FormatPK,
		},
		{
			name: "with attributes",
			input: `<template>
  <div class="test" id="main"><p>Content</p></div>
</template>`,
			format: FormatPK,
		},
		{
			name: "with directives",
			input: `<template>
  <div p-if="state.Show">
    <p p-text="state.Text"></p>
  </div>
</template>`,
			format: FormatPK,
		},
		{
			name: "plain HTML",
			input: `<div>
  <p>Hello</p>
  <p>World</p>
</div>`,
			format: FormatHTML,
		},
		{
			name:   "HTML with attributes",
			input:  `<div class="container" id="main"><span>Text</span></div>`,
			format: FormatHTML,
		},
		{
			name: "complex nested structure",
			input: `<template>
  <section>
    <header><h1>Title</h1></header>
    <main><article><p>Content</p></article></main>
    <footer><p>Footer</p></footer>
  </section>
</template>`,
			format: FormatPK,
		},
		{
			name: "list items",
			input: `<template>
  <ul><li>Item 1</li><li>Item 2</li><li>Item 3</li></ul>
</template>`,
			format: FormatPK,
		},
		{
			name: "mixed inline and block",
			input: `<template>
  <p>Text with <strong>bold</strong> and <em>italic</em> content.</p>
</template>`,
			format: FormatPK,
		},
		{
			name: "self-closing tags",
			input: `<template>
  <div>
    <img alt="Test" src="test.jpg" />
    <br />
    <hr />
  </div>
</template>`,
			format: FormatPK,
		},
		{
			name: "HTML5 semantic elements",
			input: `<article>
  <figure>
    <img alt="Photo" src="photo.jpg" />
    <figcaption>Caption</figcaption>
  </figure>
</article>`,
			format: FormatHTML,
		},
		{
			name: "form elements",
			input: `<form>
  <label for="name">Name:</label>
  <input id="name" type="text" />
  <button type="submit">Submit</button>
</form>`,
			format: FormatHTML,
		},
	}

	formatter := NewFormatterService()
	ctx := context.Background()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			opts := &FormatOptions{
				FileFormat:          tc.format,
				IndentSize:          2,
				SortAttributes:      true,
				MaxLineLength:       100,
				AttributeWrapIndent: 1,
			}

			first, err := formatter.FormatWithOptions(ctx, []byte(tc.input), opts)
			require.NoError(t, err, "First format should succeed")

			second, err := formatter.FormatWithOptions(ctx, first, opts)
			require.NoError(t, err, "Second format should succeed")

			assert.Equal(t, string(first), string(second),
				"Formatting should be idempotent: formatting twice should produce identical output")
		})
	}
}

func TestFormatter_Idempotency_MultipleIterations(t *testing.T) {
	testCases := []struct {
		name   string
		input  string
		format FileFormat
	}{
		{
			name:   "complex PK",
			input:  `<template><div class="wrapper"><header><nav><ul><li><a href="/">Home</a></li><li><a href="/about">About</a></li></ul></nav></header><main><h1>Welcome</h1><p>Content here.</p></main></div></template>`,
			format: FormatPK,
		},
		{
			name:   "complex HTML",
			input:  `<div><header><nav><ul><li><a href="/">Home</a></li></ul></nav></header><main><article><h1>Title</h1><p>Paragraph with <strong>bold</strong> text.</p></article></main></div>`,
			format: FormatHTML,
		},
	}

	formatter := NewFormatterService()
	ctx := context.Background()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			opts := &FormatOptions{
				FileFormat:          tc.format,
				IndentSize:          2,
				SortAttributes:      true,
				MaxLineLength:       100,
				AttributeWrapIndent: 1,
			}

			current := []byte(tc.input)
			var previous []byte

			for i := range 5 {
				formatted, err := formatter.FormatWithOptions(ctx, current, opts)
				require.NoError(t, err, "Format iteration %d should succeed", i+1)

				if i > 0 {
					assert.Equal(t, string(previous), string(formatted),
						"Iteration %d should produce same result as iteration %d", i+1, i)
				}

				previous = formatted
				current = formatted
			}
		})
	}
}
