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

package sfcparser_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/sfcparser"
)

func TestParseSFC(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected sfcparser.ParseResult
		wantErr  bool
	}{
		{
			name: "01 - Full valid PKC file",
			input: `
				<template enable="form">
					<div>Hello</div>
				</template>
				<style>
					div { color: red; }
				</style>
				<script name="my-component">
					const x = 1;
				</script>
			`,
			expected: sfcparser.ParseResult{
				Template: `
					<div>Hello</div>
				`,
				TemplateAttributes: map[string]string{
					"enable": "form",
				},
				Scripts: []sfcparser.Script{
					{
						Content: `
					const x = 1;
				`,
						Attributes: map[string]string{
							"name": "my-component",
						},
					},
				},
				Styles: []sfcparser.Style{
					{
						Content: `
					div { color: red; }
				`,
						Attributes: map[string]string{},
					},
				},
				I18nBlocks: []sfcparser.I18nBlock{},
			},
		},
		{
			name: "02 - Full valid PK file with both script types",
			input: `
				<template>
					<p>Go SSR</p>
				</template>
				<style>
					p { font-weight: bold; }
				</style>
				<script type="application/x-go">
					package main
				</script>
				<script type="application/javascript">
					console.log('hello');
				</script>
			`,
			expected: sfcparser.ParseResult{
				Template: `
					<p>Go SSR</p>
				`,
				Scripts: []sfcparser.Script{
					{
						Content: `
					package main
				`,
						Attributes: map[string]string{"type": "application/x-go"},
					},
					{
						Content: `
					console.log('hello');
				`,
						Attributes: map[string]string{"type": "application/javascript"},
					},
				},
				Styles: []sfcparser.Style{
					{
						Content: `
					p { font-weight: bold; }
				`,
						Attributes: map[string]string{},
					},
				},
				I18nBlocks: []sfcparser.I18nBlock{},
			},
		},
		{
			name:  "03 - Template only",
			input: `<template><h1>Just a template</h1></template>`,
			expected: sfcparser.ParseResult{
				Template:   `<h1>Just a template</h1>`,
				Scripts:    []sfcparser.Script{},
				Styles:     []sfcparser.Style{},
				I18nBlocks: []sfcparser.I18nBlock{},
			},
		},
		{
			name:  "04 - Script only",
			input: `<script name="script-only">let a = 1;</script>`,
			expected: sfcparser.ParseResult{
				Scripts: []sfcparser.Script{
					{Content: "let a = 1;", Attributes: map[string]string{"name": "script-only"}},
				},
				Styles:     []sfcparser.Style{},
				I18nBlocks: []sfcparser.I18nBlock{},
			},
		},
		{
			name:  "05 - Style only",
			input: `<style>.class { color: blue; }</style>`,
			expected: sfcparser.ParseResult{
				Scripts: []sfcparser.Script{},
				Styles: []sfcparser.Style{
					{Content: ".class { color: blue; }", Attributes: map[string]string{}},
				},
				I18nBlocks: []sfcparser.I18nBlock{},
			},
		},
		{
			name: "06 - Multiple styles with default and aesthetic",
			input: `
				<style aesthetic="dark">body { background: #111; }</style>
				<template>Content</template>
				<style>body { font-family: sans-serif; }</style>
				<style aesthetic="light">body { background: #fff; }</style>
			`,
			expected: sfcparser.ParseResult{
				Template: "Content",
				Scripts:  []sfcparser.Script{},
				Styles: []sfcparser.Style{
					{Content: "body { background: #111; }", Attributes: map[string]string{"aesthetic": "dark"}},
					{Content: "body { font-family: sans-serif; }", Attributes: map[string]string{}},
					{Content: "body { background: #fff; }", Attributes: map[string]string{"aesthetic": "light"}},
				},
				I18nBlocks: []sfcparser.I18nBlock{},
			},
		},
		{
			name: "07 - Whitespace, newlines, and comments",
			input: `
				<!-- Component Start -->
				<template>
					<!-- Template Content -->
					<div>
						<span>Content</span>
					</div>
				</template>

				<style>
					/* Style Block */
					body { margin: 0; }
				</style>
				<!-- Comment between tags -->
				<script name="messy-component">
					// Script content
					function hello() {}
				</script>
				<!-- Component End -->
			`,
			expected: sfcparser.ParseResult{
				Template: `
					<!-- Template Content -->
					<div>
						<span>Content</span>
					</div>
				`,
				Scripts: []sfcparser.Script{
					{Content: `
					// Script content
					function hello() {}
				`, Attributes: map[string]string{"name": "messy-component"}},
				},
				Styles: []sfcparser.Style{
					{Content: `
					/* Style Block */
					body { margin: 0; }
				`, Attributes: map[string]string{}},
				},
				I18nBlocks: []sfcparser.I18nBlock{},
			},
		},
		{
			name: "08 - Empty tags",
			input: `
				<template></template>
				<script></script>
				<style></style>
			`,
			expected: sfcparser.ParseResult{
				Template:   "",
				Scripts:    []sfcparser.Script{{Content: "", Attributes: map[string]string{}}},
				Styles:     []sfcparser.Style{{Content: "", Attributes: map[string]string{}}},
				I18nBlocks: []sfcparser.I18nBlock{},
			},
		},
		{
			name:  "09 - No relevant tags",
			input: `<div><p>Just some random HTML</p></div>`,
			expected: sfcparser.ParseResult{
				Scripts:    []sfcparser.Script{},
				Styles:     []sfcparser.Style{},
				I18nBlocks: []sfcparser.I18nBlock{},
			},
		},
		{
			name:  "10 - Empty input string",
			input: ``,
			expected: sfcparser.ParseResult{
				Scripts:    []sfcparser.Script{},
				Styles:     []sfcparser.Style{},
				I18nBlocks: []sfcparser.I18nBlock{},
			},
		},
		{
			name:  "11 - Script with various attributes",
			input: `<template enable="form validation"></template><script type="application/javascript" name="my-comp"></script>`,
			expected: sfcparser.ParseResult{
				TemplateAttributes: map[string]string{
					"enable": "form validation",
				},
				Scripts: []sfcparser.Script{
					{Content: "", Attributes: map[string]string{
						"type": "application/javascript",
						"name": "my-comp",
					}},
				},
				Styles:     []sfcparser.Style{},
				I18nBlocks: []sfcparser.I18nBlock{},
			},
		},
		{
			name: "12 - Malformed HTML robustness",
			input: `
				<template>
					<div>
						<p>Hello
				</template>
				<style>
					p { color: green;
			`,
			expected: sfcparser.ParseResult{
				Template: `
					<div>
						<p>Hello
				`,
				Scripts: []sfcparser.Script{},
				Styles: []sfcparser.Style{{Content: `
					p { color: green;
			`, Attributes: map[string]string{}}},
				I18nBlocks: []sfcparser.I18nBlock{},
			},
		},
		{
			name: "13 - Nested template tag is ignored as a root block",
			input: `
				<template>
					<p>Outer template</p>
					<template>
						<p>Inner template</p>
					</template>
				</template>
			`,
			expected: sfcparser.ParseResult{
				Template: `
					<p>Outer template</p>
					<template>
						<p>Inner template</p>
					</template>
				`,
				Scripts:    []sfcparser.Script{},
				Styles:     []sfcparser.Style{},
				I18nBlocks: []sfcparser.I18nBlock{},
			},
		},
		{
			name: "14 - Attributes with no value",
			input: `
				<template>
					<div disabled></div>
				</template>
				<script name="no-value-attr"></script>
			`,
			expected: sfcparser.ParseResult{
				Template: `
					<div disabled></div>
				`,
				Scripts: []sfcparser.Script{
					{Content: "", Attributes: map[string]string{"name": "no-value-attr"}},
				},
				Styles:     []sfcparser.Style{},
				I18nBlocks: []sfcparser.I18nBlock{},
			},
		},
		{
			name: "15 - Special UTF-8 characters and entities",
			input: `
        <template>
            <div>© copyright</div>
            <div>🚀 rocket</div>
            <div>&nbsp; non-breaking space</div>
        </template>
        <style>
            /* CSS with © symbol */
            .icon::before { content: '©'; }
        </style>
        <script>
            // JS with 🚀 symbol
            const emoji = "🚀";
        </script>
    `,
			expected: sfcparser.ParseResult{
				Template: `
            <div>© copyright</div>
            <div>🚀 rocket</div>
            <div>&nbsp; non-breaking space</div>
        `,
				Styles: []sfcparser.Style{
					{
						Content: `
            /* CSS with © symbol */
            .icon::before { content: '©'; }
        `,
						Attributes: map[string]string{},
					},
				},
				Scripts: []sfcparser.Script{
					{
						Content: `
            // JS with 🚀 symbol
            const emoji = "🚀";
        `,
						Attributes: map[string]string{},
					},
				},
				I18nBlocks: []sfcparser.I18nBlock{},
			},
		},
		{
			name: "16 - Single i18n block with attributes",
			input: `<i18n lang="en" scope="global">
{ "hello": "Hello, World!" }
</i18n>`,
			expected: sfcparser.ParseResult{
				Scripts: []sfcparser.Script{},
				Styles:  []sfcparser.Style{},
				I18nBlocks: []sfcparser.I18nBlock{
					{
						Content: `
{ "hello": "Hello, World!" }
`,
						Attributes: map[string]string{"lang": "en", "scope": "global"},
					},
				},
			},
		},
		{
			name: "17 - Multiple i18n blocks",
			input: `
<i18n lang="en">{ "greeting": "Hello" }</i18n>
<i18n lang="es">{ "greeting": "Hola" }</i18n>`,
			expected: sfcparser.ParseResult{
				Scripts: []sfcparser.Script{},
				Styles:  []sfcparser.Style{},
				I18nBlocks: []sfcparser.I18nBlock{
					{Content: `{ "greeting": "Hello" }`, Attributes: map[string]string{"lang": "en"}},
					{Content: `{ "greeting": "Hola" }`, Attributes: map[string]string{"lang": "es"}},
				},
			},
		},
		{
			name: "18 - Self-closing script tag",
			input: `<template>Main</template>
<script src="./my-script.js" />`,
			expected: sfcparser.ParseResult{
				Template: "Main",
				Scripts: []sfcparser.Script{
					{Content: "", Attributes: map[string]string{"src": "./my-script.js"}},
				},
				Styles:     []sfcparser.Style{},
				I18nBlocks: []sfcparser.I18nBlock{},
			},
		},
		{
			name:  "19 - Mixed case tags are parsed correctly",
			input: `<TEMPLATE>SHOULD BE PARSED</TEMPLATE>`,
			expected: sfcparser.ParseResult{
				Template:   "SHOULD BE PARSED",
				Scripts:    []sfcparser.Script{},
				Styles:     []sfcparser.Style{},
				I18nBlocks: []sfcparser.I18nBlock{},
			},
		},
		{
			name: "20 - Attribute quoting variations",
			input: `<script name='single-quoted' type=unquoted enabled value="">
// code
</script>`,
			expected: sfcparser.ParseResult{
				Scripts: []sfcparser.Script{
					{
						Content: `
// code
`,
						Attributes: map[string]string{
							"name":    "single-quoted",
							"type":    "unquoted",
							"enabled": "",
							"value":   "",
						},
					},
				},
				Styles:     []sfcparser.Style{},
				I18nBlocks: []sfcparser.I18nBlock{},
			},
		},
		{
			name: "21 - Second template block is discarded",
			input: `
<template>First template</template>
<template>Second template, should be ignored</template>`,
			expected: sfcparser.ParseResult{
				Template:   "First template",
				Scripts:    []sfcparser.Script{},
				Styles:     []sfcparser.Style{},
				I18nBlocks: []sfcparser.I18nBlock{},
			},
		},
		{
			name: "22 - Root level text and comments are ignored",
			input: `
This text should be ignored.
<!-- This comment too -->
<template>Only this should be parsed</template>`,
			expected: sfcparser.ParseResult{
				Template:   "Only this should be parsed",
				Scripts:    []sfcparser.Script{},
				Styles:     []sfcparser.Style{},
				I18nBlocks: []sfcparser.I18nBlock{},
			},
		},
		{
			name: "23 - Script content with HTML-like strings",
			input: `<script>
const myTemplate = '<template><div></div></template>';
</script>`,
			expected: sfcparser.ParseResult{
				Scripts: []sfcparser.Script{
					{
						Content: `
const myTemplate = '<template><div></div></template>';
`,
						Attributes: map[string]string{},
					},
				},
				Styles:     []sfcparser.Style{},
				I18nBlocks: []sfcparser.I18nBlock{},
			},
		},
		{
			name:  "24 - CDATA section in script is treated as text",
			input: `<script>//<![CDATA[\nif (a < b) { alert("CDATA works"); }\n//]]></script>`,
			expected: sfcparser.ParseResult{
				Scripts: []sfcparser.Script{
					{
						Content:    `//<![CDATA[\nif (a < b) { alert("CDATA works"); }\n//]]>`,
						Attributes: map[string]string{},
					},
				},
				Styles:     []sfcparser.Style{},
				I18nBlocks: []sfcparser.I18nBlock{},
			},
		},
		{
			name:  "25 - Malformed attribute with unclosed quote",
			input: `<script name="oops > console.log("Still works"); </script>`,
			expected: sfcparser.ParseResult{
				Scripts: []sfcparser.Script{
					{
						Attributes: map[string]string{
							`name`:     `oops > console.log(`,
							`still`:    ``,
							`works");`: ``,
							`</script`: ``,
						},
						Content: "",
					},
				},
				Styles:     []sfcparser.Style{},
				I18nBlocks: []sfcparser.I18nBlock{},
			},
		},
		{
			name: "26 - Interleaved blocks of all types",
			input: `
<style>s1</style>
<script>js1</script>
<template>tmpl</template>
<i18n>i18n1</i18n>
<style>s2</style>
<script>js2</script>`,
			expected: sfcparser.ParseResult{
				Template: "tmpl",
				Styles: []sfcparser.Style{
					{Content: "s1", Attributes: map[string]string{}},
					{Content: "s2", Attributes: map[string]string{}},
				},
				Scripts: []sfcparser.Script{
					{Content: "js1", Attributes: map[string]string{}},
					{Content: "js2", Attributes: map[string]string{}},
				},
				I18nBlocks: []sfcparser.I18nBlock{
					{Content: "i18n1", Attributes: map[string]string{}},
				},
			},
		},
		{
			name: "27 - Accurate location tracking",
			input: `
<style>
  div {}
</style>
<template>
  Hello
</template>
<script>
// script
</script>`,
			expected: sfcparser.ParseResult{
				TemplateLocation:        sfcparser.Location{Line: 5, Column: 1},
				TemplateContentLocation: sfcparser.Location{Line: 5, Column: 11},
				Template:                "\n  Hello\n",
				Styles: []sfcparser.Style{
					{
						Location:        sfcparser.Location{Line: 2, Column: 1},
						ContentLocation: sfcparser.Location{Line: 2, Column: 8},
						Content:         "\n  div {}\n",
						Attributes:      map[string]string{},
					},
				},
				Scripts: []sfcparser.Script{
					{
						Location:        sfcparser.Location{Line: 8, Column: 1},
						ContentLocation: sfcparser.Location{Line: 8, Column: 9},
						Content:         "\n// script\n",
						Attributes:      map[string]string{},
					},
				},
				I18nBlocks: []sfcparser.I18nBlock{},
			},
		},
		{
			name:  "28 - Tags inside comments are ignored",
			input: `<!-- <template>This is a comment</template> -->`,
			expected: sfcparser.ParseResult{
				Scripts:    []sfcparser.Script{},
				Styles:     []sfcparser.Style{},
				I18nBlocks: []sfcparser.I18nBlock{},
			},
		},
		{
			name:  "29 - Template contains a script tag",
			input: `<template><div><script>alert("inline")</script></div></template>`,
			expected: sfcparser.ParseResult{
				Template:   `<div><script>alert("inline")</script></div>`,
				Scripts:    []sfcparser.Script{},
				Styles:     []sfcparser.Style{},
				I18nBlocks: []sfcparser.I18nBlock{},
			},
		},
		{
			name:  "29b - Template contains a style tag",
			input: `<template><div><style>.red { color: red; }</style></div></template>`,
			expected: sfcparser.ParseResult{
				Template:   `<div><style>.red { color: red; }</style></div>`,
				Scripts:    []sfcparser.Script{},
				Styles:     []sfcparser.Style{},
				I18nBlocks: []sfcparser.I18nBlock{},
			},
		},
		{
			name:  "30 - Template with self-closing custom elements",
			input: `<template><my-component/><br/></template>`,
			expected: sfcparser.ParseResult{
				Template:   `<my-component/><br/>`,
				Scripts:    []sfcparser.Script{},
				Styles:     []sfcparser.Style{},
				I18nBlocks: []sfcparser.I18nBlock{},
			},
		},
		{
			name:    "31 - File ends after unclosed start tag",
			input:   `<template><div>hello</div></template><script`,
			wantErr: false,
			expected: sfcparser.ParseResult{
				Template:   `<div>hello</div>`,
				Scripts:    []sfcparser.Script{},
				Styles:     []sfcparser.Style{},
				I18nBlocks: []sfcparser.I18nBlock{},
			},
		},
		{
			name:  "32 - Void element does not consume next tag",
			input: `<template><img src="..."></template><script>let a=1;</script>`,
			expected: sfcparser.ParseResult{
				Template:   `<img src="...">`,
				Scripts:    []sfcparser.Script{{Content: "let a=1;", Attributes: map[string]string{}}},
				Styles:     []sfcparser.Style{},
				I18nBlocks: []sfcparser.I18nBlock{},
			},
		},
		{
			name:  "33 - Only whitespace input",
			input: " \n\t \r\n ",
			expected: sfcparser.ParseResult{
				Scripts:    []sfcparser.Script{},
				Styles:     []sfcparser.Style{},
				I18nBlocks: []sfcparser.I18nBlock{},
			},
		},
		{
			name: "34 - Complex nested content in template",
			input: `<template>
    <div class="container">
        <!-- A comment -->
        <div class="nested" data-value="<>'">
            <p>Text</p>
        </div>
    </div>
</template>`,
			expected: sfcparser.ParseResult{
				Template: `
    <div class="container">
        <!-- A comment -->
        <div class="nested" data-value="<>'">
            <p>Text</p>
        </div>
    </div>
`,
				Scripts:    []sfcparser.Script{},
				Styles:     []sfcparser.Style{},
				I18nBlocks: []sfcparser.I18nBlock{},
			},
		},
		{
			name:  "35 - Script tag with special characters in attributes",
			input: `<script data-json='{"key": "value & stuff"}'></script>`,
			expected: sfcparser.ParseResult{
				Scripts: []sfcparser.Script{
					{Attributes: map[string]string{"data-json": `{"key": "value & stuff"}`}, Content: ""},
				},
				Styles:     []sfcparser.Style{},
				I18nBlocks: []sfcparser.I18nBlock{},
			},
		},
		{
			name:  "36 - UTF-8 in Template Content (CJK and Accented)",
			input: `<template><h1>你好, 世界!</h1><p>Welcome to München, enjoy a café crème brûlée.</p></template>`,
			expected: sfcparser.ParseResult{
				Template:   `<h1>你好, 世界!</h1><p>Welcome to München, enjoy a café crème brûlée.</p>`,
				Scripts:    []sfcparser.Script{},
				Styles:     []sfcparser.Style{},
				I18nBlocks: []sfcparser.I18nBlock{},
			},
		},
		{
			name:  "37 - UTF-8 Emojis in Script Content and Attribute Values",
			input: `<script title="🚀 Launch Status: OK ✨">const status = "✅"; let rocket = "🚀";</script>`,
			expected: sfcparser.ParseResult{
				Scripts: []sfcparser.Script{
					{
						Attributes: map[string]string{"title": "🚀 Launch Status: OK ✨"},
						Content:    `const status = "✅"; let rocket = "🚀";`,
					},
				},
				Styles:     []sfcparser.Style{},
				I18nBlocks: []sfcparser.I18nBlock{},
			},
		},
		{
			name: `38 - UTF-8 in Style and I18n Content (Symbols and Greek)`,
			input: `
<style>
/* Mathematical Symbols: ∑ ∆ ∞ */
.math::before { content: '∑'; }
</style>
<i18n lang="el">
{ "farewell": "αντίο κόσμε" }
</i18n>`,
			expected: sfcparser.ParseResult{
				Scripts: []sfcparser.Script{},
				Styles: []sfcparser.Style{
					{
						Content:    "\n/* Mathematical Symbols: ∑ ∆ ∞ */\n.math::before { content: '∑'; }\n",
						Attributes: map[string]string{},
					},
				},
				I18nBlocks: []sfcparser.I18nBlock{
					{
						Content:    "\n{ \"farewell\": \"αντίο κόσμε\" }\n",
						Attributes: map[string]string{"lang": "el"},
					},
				},
			},
		},
		{
			name: `39 - Extreme UTF-8 with Zalgo Text and Multi-Codepoint Emoji`,
			input: `<template><span>H̵e̴'s̶ ̷c̴o̷m̸i̴n̸g̸.</span></template>
<script>
// Family: 👨‍👩‍👧‍👦
let family = "👨‍👩‍👧‍👦";
</script>`,
			expected: sfcparser.ParseResult{
				Template: `<span>H̵e̴'s̶ ̷c̴o̷m̸i̴n̸g̸.</span>`,
				Scripts: []sfcparser.Script{
					{
						Content:    "\n// Family: 👨‍👩‍👧‍👦\nlet family = \"👨‍👩‍👧‍👦\";\n",
						Attributes: map[string]string{},
					},
				},
				Styles:     []sfcparser.Style{},
				I18nBlocks: []sfcparser.I18nBlock{},
			},
		},
		{
			name:  "40 - UTF-8 in Attribute Names (Lexer Behaviour)",
			input: `<script ✅="true" data-你好="世界">/* check */</script>`,
			expected: sfcparser.ParseResult{
				Scripts: []sfcparser.Script{
					{
						Attributes: map[string]string{
							"✅":       "true",
							"data-你好": "世界",
						},
						Content: "/* check */",
					},
				},
				Styles:     []sfcparser.Style{},
				I18nBlocks: []sfcparser.I18nBlock{},
			},
		},
		{
			name: "41 - Script content contains a closing script tag inside a string literal",
			input: `<script type="application/x-go">
package main
func main() {
    println("</script>")
}
</script>`,
			expected: sfcparser.ParseResult{
				Scripts: []sfcparser.Script{
					{
						Content:    "\npackage main\nfunc main() {\n    println(\"</script>\")\n}\n",
						Attributes: map[string]string{"type": "application/x-go"},
					},
				},
				Styles:     []sfcparser.Style{},
				I18nBlocks: []sfcparser.I18nBlock{},
			},
		},
		{
			name: "42 - Template content contains nested template tags",
			input: `<template>
    <div>
        <template>
            <p>Inner</p>
        </template>
    </div>
</template>`,
			expected: sfcparser.ParseResult{
				Template: `
    <div>
        <template>
            <p>Inner</p>
        </template>
    </div>
`,
				Scripts:    []sfcparser.Script{},
				Styles:     []sfcparser.Style{},
				I18nBlocks: []sfcparser.I18nBlock{},
			},
		},
		{
			name: "43 - Template with multiple levels of nested template tags",
			input: `
<template>
  <!-- Level 1 -->
  <div class="outer">
    <template v-if="show">
      <!-- Level 2 -->
      <p>Inner content</p>
      <template v-for="item in items">
        <!-- Level 3 -->
        <span>{{ item }}</span>
      </template>
    </template>
  </div>
</template>
<script>
// This script should be parsed separately
</script>
`,
			expected: sfcparser.ParseResult{
				Template: `
  <!-- Level 1 -->
  <div class="outer">
    <template v-if="show">
      <!-- Level 2 -->
      <p>Inner content</p>
      <template v-for="item in items">
        <!-- Level 3 -->
        <span>{{ item }}</span>
      </template>
    </template>
  </div>
`,
				Scripts: []sfcparser.Script{
					{
						Content:    "\n// This script should be parsed separately\n",
						Attributes: map[string]string{},
					},
				},
				Styles:     []sfcparser.Style{},
				I18nBlocks: []sfcparser.I18nBlock{},
			},
		},
		{
			name: "44 - Leading whitespace before any tag",
			input: `

  
<template>
	<p>Content</p>
</template>`,
			expected: sfcparser.ParseResult{
				TemplateLocation:        sfcparser.Location{Line: 4, Column: 1},
				TemplateContentLocation: sfcparser.Location{Line: 4, Column: 11},
				Template:                "\n\t<p>Content</p>\n",
				Scripts:                 []sfcparser.Script{},
				Styles:                  []sfcparser.Style{},
				I18nBlocks:              []sfcparser.I18nBlock{},
			},
		},
		{
			name: "44 - Leading whitespace before any tag",
			input: `

  
<template>
	<p>Content</p>
</template>`,
			expected: sfcparser.ParseResult{
				TemplateLocation:        sfcparser.Location{Line: 4, Column: 1},
				TemplateContentLocation: sfcparser.Location{Line: 4, Column: 11},
				Template:                "\n\t<p>Content</p>\n",
				Scripts:                 []sfcparser.Script{},
				Styles:                  []sfcparser.Style{},
				I18nBlocks:              []sfcparser.I18nBlock{},
			},
		},
		{
			name: "45 - Whitespace between all tags",
			input: `<template>
  Template Content
</template>


		<script>
		  Script Content
		</script>
  
  
<style>
  Style Content
</style>`,
			expected: sfcparser.ParseResult{
				TemplateLocation:        sfcparser.Location{Line: 1, Column: 1},
				TemplateContentLocation: sfcparser.Location{Line: 1, Column: 11},
				Template:                "\n  Template Content\n",
				Scripts: []sfcparser.Script{
					{
						Location:        sfcparser.Location{Line: 6, Column: 3},
						ContentLocation: sfcparser.Location{Line: 6, Column: 11},
						Content:         "\n\t\t  Script Content\n\t\t",
						Attributes:      map[string]string{},
					},
				},
				Styles: []sfcparser.Style{
					{
						Location:        sfcparser.Location{Line: 11, Column: 1},
						ContentLocation: sfcparser.Location{Line: 11, Column: 8},
						Content:         "\n  Style Content\n",
						Attributes:      map[string]string{},
					},
				},
			},
		},
		{
			name:  "46 - Whitespace within tag definition and attributes",
			input: `<script   name = "my-script"  lang='go'   disabled  >package main</script>`,
			expected: sfcparser.ParseResult{
				Scripts: []sfcparser.Script{
					{
						Content: "package main",
						Attributes: map[string]string{
							"name":     "my-script",
							"lang":     "go",
							"disabled": "",
						},
					},
				},
			},
		},
		{
			name:  "47 - Trailing whitespace after last tag",
			input: `<template>Final content</template>   \n\n\t  `,
			expected: sfcparser.ParseResult{
				Template: "Final content",
			},
		},
		{
			name: "48 - Whitespace padding inside content blocks",
			input: `<style>
	
	.padded {
		margin: 1em;
	}

</style>`,
			expected: sfcparser.ParseResult{
				Styles: []sfcparser.Style{
					{
						Location:        sfcparser.Location{Line: 1, Column: 1},
						ContentLocation: sfcparser.Location{Line: 1, Column: 8},
						Content:         "\n\t\n\t.padded {\n\t\tmargin: 1em;\n\t}\n\n",
						Attributes:      map[string]string{},
					},
				},
			},
		},
		{
			name:  "49 - Self-closing void template tag",
			input: `<template p-collection="posts" />`,
			expected: sfcparser.ParseResult{
				Template: "",
				TemplateAttributes: map[string]string{
					"p-collection": "posts",
				},
				Scripts:    []sfcparser.Script{},
				Styles:     []sfcparser.Style{},
				I18nBlocks: []sfcparser.I18nBlock{},
			},
		},
		{
			name:  "50 - Self-closing void style tag",
			input: `<style src="external.css" />`,
			expected: sfcparser.ParseResult{
				Scripts: []sfcparser.Script{},
				Styles: []sfcparser.Style{
					{
						Content:    "",
						Attributes: map[string]string{"src": "external.css"},
					},
				},
				I18nBlocks: []sfcparser.I18nBlock{},
			},
		},
		{
			name:  "51 - Self-closing void i18n tag",
			input: `<i18n src="translations.json" />`,
			expected: sfcparser.ParseResult{
				Scripts: []sfcparser.Script{},
				Styles:  []sfcparser.Style{},
				I18nBlocks: []sfcparser.I18nBlock{
					{
						Content:    "",
						Attributes: map[string]string{"src": "translations.json"},
					},
				},
			},
		},
		{
			name: "52 - Multiple nested second template tags",
			input: `<template>First</template>
<template>
	Second outer
	<template>Second inner</template>
</template>`,
			expected: sfcparser.ParseResult{
				Template:   "First",
				Scripts:    []sfcparser.Script{},
				Styles:     []sfcparser.Style{},
				I18nBlocks: []sfcparser.I18nBlock{},
			},
		},
		{
			name: "53 - Template with p-collection and p-provider attributes",
			input: `<template p-collection="blog-posts" p-provider="yaml">
	<div>Content</div>
</template>`,
			expected: sfcparser.ParseResult{
				Template: `
	<div>Content</div>
`,
				TemplateAttributes: map[string]string{
					"p-collection": "blog-posts",
					"p-provider":   "yaml",
				},
				Scripts:    []sfcparser.Script{},
				Styles:     []sfcparser.Style{},
				I18nBlocks: []sfcparser.I18nBlock{},
			},
		},
		{
			name: "54 - Script with all supported Go type variations",
			input: `
<script type="application/x-go">package main</script>
<script type="application/go">package foo</script>
<script lang="go">package bar</script>
<script lang="golang">package baz</script>`,
			expected: sfcparser.ParseResult{
				Scripts: []sfcparser.Script{
					{Content: "package main", Attributes: map[string]string{"type": "application/x-go"}},
					{Content: "package foo", Attributes: map[string]string{"type": "application/go"}},
					{Content: "package bar", Attributes: map[string]string{"lang": "go"}},
					{Content: "package baz", Attributes: map[string]string{"lang": "golang"}},
				},
				Styles:     []sfcparser.Style{},
				I18nBlocks: []sfcparser.I18nBlock{},
			},
		},
		{
			name: "55 - Mixed JavaScript and Go scripts",
			input: `
<script type="application/javascript">console.log('js1');</script>
<script type="application/x-go">package main</script>
<script type="module">export default {};</script>
<script lang="go">package test</script>
<script>// Default JS</script>`,
			expected: sfcparser.ParseResult{
				Scripts: []sfcparser.Script{
					{Content: "console.log('js1');", Attributes: map[string]string{"type": "application/javascript"}},
					{Content: "package main", Attributes: map[string]string{"type": "application/x-go"}},
					{Content: "export default {};", Attributes: map[string]string{"type": "module"}},
					{Content: "package test", Attributes: map[string]string{"lang": "go"}},
					{Content: "// Default JS", Attributes: map[string]string{}},
				},
				Styles:     []sfcparser.Style{},
				I18nBlocks: []sfcparser.I18nBlock{},
			},
		},
		{
			name:  "56 - Empty i18n block",
			input: `<i18n lang="en"></i18n>`,
			expected: sfcparser.ParseResult{
				Scripts: []sfcparser.Script{},
				Styles:  []sfcparser.Style{},
				I18nBlocks: []sfcparser.I18nBlock{
					{
						Content:    "",
						Attributes: map[string]string{"lang": "en"},
					},
				},
			},
		},
		{
			name: "57 - I18n with nested JSON structure",
			input: `<i18n lang="en" format="json5">
{
  "welcome": "Hello",
  "nested": {
    "key": "value"
  }
}
</i18n>`,
			expected: sfcparser.ParseResult{
				Scripts: []sfcparser.Script{},
				Styles:  []sfcparser.Style{},
				I18nBlocks: []sfcparser.I18nBlock{
					{
						Content: `
{
  "welcome": "Hello",
  "nested": {
    "key": "value"
  }
}
`,
						Attributes: map[string]string{
							"lang":   "en",
							"format": "json5",
						},
					},
				},
			},
		},
		{
			name: "57b - Script body with </script> in JS comment plus unclosed <!-- does not swallow following style block",
			input: `<script lang="ts">
// docs: ` + "`" + `<script type="module">…</script>` + "`" + ` is fine
const tokens = ['<!--', 'comment.html'];
const more = "no closer here";
</script>
<style>
.box { color: red; }
</style>`,
			expected: sfcparser.ParseResult{
				Scripts: []sfcparser.Script{
					{
						Content: `
// docs: ` + "`" + `<script type="module">…</script>` + "`" + ` is fine
const tokens = ['<!--', 'comment.html'];
const more = "no closer here";
`,
						Attributes: map[string]string{"lang": "ts"},
					},
				},
				Styles: []sfcparser.Style{
					{
						Content:    "\n.box { color: red; }\n",
						Attributes: map[string]string{},
					},
				},
				I18nBlocks: []sfcparser.I18nBlock{},
			},
		},
		{
			name: "57c - Script with </script> in comment then <!-- preserves following blocks",
			input: `<template>
	<div>hello</div>
</template>
<script>
// example: <script type="module"></script>
const a = '<!--';
const b = '<script type="module">';
</script>
<style>
.x { color: blue; }
</style>
<i18n lang="en">{"k": "v"}</i18n>`,
			expected: sfcparser.ParseResult{
				Template: `
	<div>hello</div>
`,
				Scripts: []sfcparser.Script{
					{
						Content: `
// example: <script type="module"></script>
const a = '<!--';
const b = '<script type="module">';
`,
						Attributes: map[string]string{},
					},
				},
				Styles: []sfcparser.Style{
					{
						Content:    "\n.x { color: blue; }\n",
						Attributes: map[string]string{},
					},
				},
				I18nBlocks: []sfcparser.I18nBlock{
					{
						Content:    `{"k": "v"}`,
						Attributes: map[string]string{"lang": "en"},
					},
				},
			},
		},
		{
			name: "58 - Template containing i18n tag",
			input: `<template>
	<div>
		<i18n>Translation inside template</i18n>
	</div>
</template>`,
			expected: sfcparser.ParseResult{
				Template: `
	<div>
		<i18n>Translation inside template</i18n>
	</div>
`,
				Scripts:    []sfcparser.Script{},
				Styles:     []sfcparser.Style{},
				I18nBlocks: []sfcparser.I18nBlock{},
			},
		},
		{
			name: "59 - License header comment before sections is ignored",
			input: `<!--
  Copyright 2026 PolitePixels Limited

  Licensed under the Apache License, Version 2.0 (the "License");
  you may not use this file except in compliance with the License.
  You may obtain a copy of the License at

      http://www.apache.org/licenses/LICENSE-2.0

  Unless required by applicable law or agreed to in writing, software
  distributed under the License is distributed on an "AS IS" BASIS,
  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  See the License for the specific language governing permissions and
  limitations under the License.

  This project stands against fascism, authoritarianism, and all forms of
  oppression. We built this to empower people, not to enable those who would
  strip others of their rights and dignity.
-->

<template>
	<div>Hello</div>
</template>

<script type="application/x-go">
package main
</script>

<style>
div { color: red; }
</style>`,
			expected: sfcparser.ParseResult{
				Template: `
	<div>Hello</div>
`,
				Scripts: []sfcparser.Script{
					{
						Content: `
package main
`,
						Attributes: map[string]string{
							"type": "application/x-go",
						},
					},
				},
				Styles: []sfcparser.Style{
					{
						Content: `
div { color: red; }
`,
						Attributes: map[string]string{},
					},
				},
				I18nBlocks: []sfcparser.I18nBlock{},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := sfcparser.Parse([]byte(tc.input))

			if tc.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)

			expectedTemplate := strings.TrimSpace(tc.expected.Template)
			actualTemplate := strings.TrimSpace(result.Template)
			assert.Equal(t, expectedTemplate, actualTemplate, "Template content mismatch")
			if tc.expected.TemplateLocation.Line > 0 {
				assert.Equal(t, tc.expected.TemplateLocation, result.TemplateLocation, "Template location mismatch")
				assert.Equal(t, tc.expected.TemplateContentLocation, result.TemplateContentLocation, "TemplateContentLocation mismatch")
			}

			require.Len(t, result.Scripts, len(tc.expected.Scripts), "Number of script blocks mismatch")
			for i, expectedScript := range tc.expected.Scripts {
				actualScript := result.Scripts[i]
				assert.Equal(t, expectedScript.Attributes, actualScript.Attributes, "Script attributes mismatch for script #%d", i)
				assert.Equal(t, strings.TrimSpace(expectedScript.Content), strings.TrimSpace(actualScript.Content), "Script content mismatch for script #%d", i)
				if expectedScript.Location.Line > 0 {
					assert.Equal(t, expectedScript.Location, actualScript.Location, "Script location mismatch for script #%d", i)
					assert.Equal(t, expectedScript.ContentLocation, actualScript.ContentLocation, "Script ContentLocation mismatch for script #%d", i)
				}
			}

			require.Len(t, result.Styles, len(tc.expected.Styles), "Number of style blocks mismatch")
			for i, expectedStyle := range tc.expected.Styles {
				actualStyle := result.Styles[i]
				assert.Equal(t, expectedStyle.Attributes, actualStyle.Attributes, "Style attributes mismatch for style #%d", i)
				assert.Equal(t, strings.TrimSpace(expectedStyle.Content), strings.TrimSpace(actualStyle.Content), "Style content mismatch for style #%d", i)
				if expectedStyle.Location.Line > 0 {
					assert.Equal(t, expectedStyle.Location, actualStyle.Location, "Style location mismatch for style #%d", i)
					assert.Equal(t, expectedStyle.ContentLocation, actualStyle.ContentLocation, "Style ContentLocation mismatch for style #%d", i)
				}
			}

			require.Len(t, result.I18nBlocks, len(tc.expected.I18nBlocks), "Number of i18n blocks mismatch")
			for i, expectedBlock := range tc.expected.I18nBlocks {
				actualBlock := result.I18nBlocks[i]
				assert.Equal(t, expectedBlock.Attributes, actualBlock.Attributes, "I18n attributes mismatch for block #%d", i)
				assert.Equal(t, strings.TrimSpace(expectedBlock.Content), strings.TrimSpace(actualBlock.Content), "I18n content mismatch for block #%d", i)
				if expectedBlock.Location.Line > 0 {
					assert.Equal(t, expectedBlock.Location, actualBlock.Location, "I18n location mismatch for block #%d", i)
					assert.Equal(t, expectedBlock.ContentLocation, actualBlock.ContentLocation, "I18n ContentLocation mismatch for block #%d", i)
				}
			}
		})
	}
}
