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
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"piko.sh/piko/internal/sfcparser"
)

func TestReassembleSFC_TemplateOnly(t *testing.T) {
	t.Run("simple template", func(t *testing.T) {
		sfcResult := &sfcparser.ParseResult{
			Template: "<div>Hello</div>",
		}
		formattedTemplate := "<div>\n  Hello\n</div>"

		result := reassembleSFC(sfcResult, formattedTemplate, nil, nil, nil)

		assert.Contains(t, result, "<template>")
		assert.Contains(t, result, "</template>")
		assert.Contains(t, result, "  <div>")
		assert.Contains(t, result, "    Hello")
		assert.Contains(t, result, "  </div>")
	})

	t.Run("empty template", func(t *testing.T) {
		sfcResult := &sfcparser.ParseResult{
			Template: "",
		}
		formattedTemplate := ""

		result := reassembleSFC(sfcResult, formattedTemplate, nil, nil, nil)

		assert.Equal(t, "", result)
	})

	t.Run("template with multiple lines", func(t *testing.T) {
		sfcResult := &sfcparser.ParseResult{
			Template: "<div><p>Line 1</p><p>Line 2</p></div>",
		}
		formattedTemplate := "<div>\n  <p>Line 1</p>\n  <p>Line 2</p>\n</div>"

		result := reassembleSFC(sfcResult, formattedTemplate, nil, nil, nil)

		lines := strings.Split(result, "\n")

		assert.Contains(t, result, "  <div>")
		assert.Contains(t, result, "    <p>Line 1</p>")
		assert.Contains(t, result, "    <p>Line 2</p>")
		assert.Contains(t, result, "  </div>")

		assert.Greater(t, len(lines), 3)
	})
}

func TestReassembleSFC_TemplateAndScript(t *testing.T) {
	t.Run("template with Go script", func(t *testing.T) {
		sfcResult := &sfcparser.ParseResult{
			Template: "<div>Hello</div>",
			Scripts: []sfcparser.Script{
				{
					Attributes: map[string]string{
						"type": "application/x-go",
					},
					Content: "package main\n\nfunc Render() {}",
				},
			},
		}
		formattedTemplate := "<div>\n  Hello\n</div>"
		formattedScripts := sfcResult.Scripts

		result := reassembleSFC(sfcResult, formattedTemplate, formattedScripts, nil, nil)

		assert.Contains(t, result, "<template>")
		assert.Contains(t, result, "</template>")

		assert.Contains(t, result, `<script type="application/x-go">`)
		assert.Contains(t, result, "package main")
		assert.Contains(t, result, "func Render() {}")
		assert.Contains(t, result, "</script>")

		templateIndex := strings.Index(result, "</template>")
		scriptIndex := strings.Index(result, "<script")
		assert.Less(t, templateIndex, scriptIndex)
	})

	t.Run("script with multiple attributes", func(t *testing.T) {
		sfcResult := &sfcparser.ParseResult{
			Scripts: []sfcparser.Script{
				{
					Attributes: map[string]string{
						"type": "application/x-go",
						"lang": "go",
					},
					Content: "package main",
				},
			},
		}
		formattedScripts := sfcResult.Scripts

		result := reassembleSFC(sfcResult, "", formattedScripts, nil, nil)

		assert.Contains(t, result, "<script")
		assert.Contains(t, result, `type="application/x-go"`)
		assert.Contains(t, result, `lang="go"`)
		assert.Contains(t, result, "</script>")
	})

	t.Run("script with boolean attribute", func(t *testing.T) {
		sfcResult := &sfcparser.ParseResult{
			Scripts: []sfcparser.Script{
				{
					Attributes: map[string]string{
						"async": "",
					},
					Content: "console.log('test')",
				},
			},
		}
		formattedScripts := sfcResult.Scripts

		result := reassembleSFC(sfcResult, "", formattedScripts, nil, nil)

		assert.Contains(t, result, "<script async>")
		assert.NotContains(t, result, `async=""`)
	})

	t.Run("script with empty content", func(t *testing.T) {
		sfcResult := &sfcparser.ParseResult{
			Scripts: []sfcparser.Script{
				{
					Attributes: map[string]string{
						"type": "application/x-go",
					},
					Content: "",
				},
			},
		}
		formattedScripts := sfcResult.Scripts

		result := reassembleSFC(sfcResult, "", formattedScripts, nil, nil)

		assert.Contains(t, result, `<script type="application/x-go">`)
		assert.Contains(t, result, "</script>")

		assert.Contains(t, result, ">\n</script>")
	})

	t.Run("script content ends with newline", func(t *testing.T) {
		sfcResult := &sfcparser.ParseResult{
			Scripts: []sfcparser.Script{
				{
					Content: "package main\n",
				},
			},
		}
		formattedScripts := sfcResult.Scripts

		result := reassembleSFC(sfcResult, "", formattedScripts, nil, nil)

		assert.Contains(t, result, "package main\n</script>")
		assert.NotContains(t, result, "package main\n\n</script>")
	})
}

func TestReassembleSFC_AllBlocks(t *testing.T) {
	t.Run("complete SFC with all block types", func(t *testing.T) {
		sfcResult := &sfcparser.ParseResult{
			Template: "<div>Content</div>",
			Scripts: []sfcparser.Script{
				{
					Attributes: map[string]string{
						"type": "application/x-go",
					},
					Content: "package main",
				},
			},
			Styles: []sfcparser.Style{
				{
					Content: ".container { color: red; }",
				},
			},
			I18nBlocks: []sfcparser.I18nBlock{
				{
					Attributes: map[string]string{
						"locale": "en",
					},
					Content: `{"greeting": "Hello"}`,
				},
			},
		}
		formattedTemplate := "<div>\n  Content\n</div>"
		formattedScripts := sfcResult.Scripts
		formattedStyles := sfcResult.Styles
		formattedI18nBlocks := sfcResult.I18nBlocks

		result := reassembleSFC(sfcResult, formattedTemplate, formattedScripts, formattedStyles, formattedI18nBlocks)

		assert.Contains(t, result, "<template>")
		assert.Contains(t, result, "</template>")
		assert.Contains(t, result, "<script")
		assert.Contains(t, result, "</script>")
		assert.Contains(t, result, "<style>")
		assert.Contains(t, result, "</style>")
		assert.Contains(t, result, "<i18n")
		assert.Contains(t, result, "</i18n>")

		templateIndex := strings.Index(result, "</template>")
		scriptIndex := strings.Index(result, "<script")
		styleIndex := strings.Index(result, "<style")
		i18nIndex := strings.Index(result, "<i18n")

		assert.Less(t, templateIndex, scriptIndex)
		assert.Less(t, scriptIndex, styleIndex)
		assert.Less(t, styleIndex, i18nIndex)
	})
}

func TestReassembleSFC_MultipleBlocks(t *testing.T) {
	t.Run("multiple script blocks", func(t *testing.T) {
		sfcResult := &sfcparser.ParseResult{
			Scripts: []sfcparser.Script{
				{
					Attributes: map[string]string{
						"type": "application/x-go",
					},
					Content: "// Go script",
				},
				{
					Attributes: map[string]string{
						"type": "text/javascript",
					},
					Content: "// JS script",
				},
			},
		}
		formattedScripts := sfcResult.Scripts

		result := reassembleSFC(sfcResult, "", formattedScripts, nil, nil)

		assert.Contains(t, result, "// Go script")
		assert.Contains(t, result, "// JS script")

		scriptCount := strings.Count(result, "<script")
		assert.Equal(t, 2, scriptCount)
	})

	t.Run("multiple style blocks", func(t *testing.T) {
		sfcResult := &sfcparser.ParseResult{
			Styles: []sfcparser.Style{
				{
					Attributes: map[string]string{
						"scoped": "",
					},
					Content: ".scoped { color: blue; }",
				},
				{
					Content: ".global { color: red; }",
				},
			},
		}

		result := reassembleSFC(sfcResult, "", nil, sfcResult.Styles, nil)

		assert.Contains(t, result, ".scoped { color: blue; }")
		assert.Contains(t, result, ".global { color: red; }")

		styleCount := strings.Count(result, "<style")
		assert.Equal(t, 2, styleCount)
	})

	t.Run("multiple i18n blocks", func(t *testing.T) {
		sfcResult := &sfcparser.ParseResult{
			I18nBlocks: []sfcparser.I18nBlock{
				{
					Attributes: map[string]string{
						"locale": "en",
					},
					Content: `{"hello": "Hello"}`,
				},
				{
					Attributes: map[string]string{
						"locale": "fr",
					},
					Content: `{"hello": "Bonjour"}`,
				},
			},
		}

		result := reassembleSFC(sfcResult, "", nil, nil, sfcResult.I18nBlocks)

		assert.Contains(t, result, `{"hello": "Hello"}`)
		assert.Contains(t, result, `{"hello": "Bonjour"}`)
		assert.Contains(t, result, `locale="en"`)
		assert.Contains(t, result, `locale="fr"`)

		i18nCount := strings.Count(result, "<i18n")
		assert.Equal(t, 2, i18nCount)
	})
}

func TestReassembleSFC_StyleBlocks(t *testing.T) {
	t.Run("style with attributes", func(t *testing.T) {
		sfcResult := &sfcparser.ParseResult{
			Styles: []sfcparser.Style{
				{
					Attributes: map[string]string{
						"scoped": "",
						"lang":   "scss",
					},
					Content: ".container { color: red; }",
				},
			},
		}

		result := reassembleSFC(sfcResult, "", nil, sfcResult.Styles, nil)

		assert.Contains(t, result, "<style")
		assert.Contains(t, result, "scoped")
		assert.Contains(t, result, `lang="scss"`)
		assert.Contains(t, result, ".container { color: red; }")
		assert.Contains(t, result, "</style>")
	})

	t.Run("style content without trailing newline", func(t *testing.T) {
		sfcResult := &sfcparser.ParseResult{
			Styles: []sfcparser.Style{
				{
					Content: ".container { color: red; }",
				},
			},
		}

		result := reassembleSFC(sfcResult, "", nil, sfcResult.Styles, nil)

		assert.Contains(t, result, ".container { color: red; }\n</style>")
	})

	t.Run("style content with trailing newline", func(t *testing.T) {
		sfcResult := &sfcparser.ParseResult{
			Styles: []sfcparser.Style{
				{
					Content: ".container { color: red; }\n",
				},
			},
		}

		result := reassembleSFC(sfcResult, "", nil, sfcResult.Styles, nil)

		assert.Contains(t, result, ".container { color: red; }\n</style>")
		assert.NotContains(t, result, ".container { color: red; }\n\n</style>")
	})

	t.Run("empty style block", func(t *testing.T) {
		sfcResult := &sfcparser.ParseResult{
			Styles: []sfcparser.Style{
				{
					Content: "",
				},
			},
		}

		result := reassembleSFC(sfcResult, "", nil, sfcResult.Styles, nil)

		assert.Contains(t, result, "<style>")
		assert.Contains(t, result, "</style>")
		assert.Contains(t, result, ">\n</style>")
	})
}

func TestReassembleSFC_I18nBlocks(t *testing.T) {
	t.Run("i18n with attributes", func(t *testing.T) {
		sfcResult := &sfcparser.ParseResult{
			I18nBlocks: []sfcparser.I18nBlock{
				{
					Attributes: map[string]string{
						"locale": "en",
						"format": "json",
					},
					Content: `{"key": "value"}`,
				},
			},
		}

		result := reassembleSFC(sfcResult, "", nil, nil, sfcResult.I18nBlocks)

		assert.Contains(t, result, "<i18n")
		assert.Contains(t, result, `locale="en"`)
		assert.Contains(t, result, `format="json"`)
		assert.Contains(t, result, `{"key": "value"}`)
		assert.Contains(t, result, "</i18n>")
	})

	t.Run("i18n content without trailing newline", func(t *testing.T) {
		sfcResult := &sfcparser.ParseResult{
			I18nBlocks: []sfcparser.I18nBlock{
				{
					Content: `{"key": "value"}`,
				},
			},
		}

		result := reassembleSFC(sfcResult, "", nil, nil, sfcResult.I18nBlocks)

		assert.Contains(t, result, `{"key": "value"}`+"\n</i18n>")
	})

	t.Run("i18n content with trailing newline", func(t *testing.T) {
		sfcResult := &sfcparser.ParseResult{
			I18nBlocks: []sfcparser.I18nBlock{
				{
					Content: "{\"key\": \"value\"}\n",
				},
			},
		}

		result := reassembleSFC(sfcResult, "", nil, nil, sfcResult.I18nBlocks)

		assert.Contains(t, result, "{\"key\": \"value\"}\n</i18n>")
		assert.NotContains(t, result, "{\"key\": \"value\"}\n\n</i18n>")
	})

	t.Run("empty i18n block", func(t *testing.T) {
		sfcResult := &sfcparser.ParseResult{
			I18nBlocks: []sfcparser.I18nBlock{
				{
					Content: "",
				},
			},
		}

		result := reassembleSFC(sfcResult, "", nil, nil, sfcResult.I18nBlocks)

		assert.Contains(t, result, "<i18n>")
		assert.Contains(t, result, "</i18n>")
		assert.Contains(t, result, ">\n</i18n>")
	})
}

func TestReassembleSFC_TemplateIndentation(t *testing.T) {
	t.Run("single line template gets indented", func(t *testing.T) {
		sfcResult := &sfcparser.ParseResult{
			Template: "<div>Hello</div>",
		}
		formattedTemplate := "<div>Hello</div>"

		result := reassembleSFC(sfcResult, formattedTemplate, nil, nil, nil)

		assert.Contains(t, result, "  <div>Hello</div>")
	})

	t.Run("multi-line template gets consistent indentation", func(t *testing.T) {
		sfcResult := &sfcparser.ParseResult{
			Template: "<div>\n  <p>Hello</p>\n</div>",
		}
		formattedTemplate := "<div>\n  <p>Hello</p>\n</div>"

		result := reassembleSFC(sfcResult, formattedTemplate, nil, nil, nil)

		assert.Contains(t, result, "  <div>")
		assert.Contains(t, result, "    <p>Hello</p>")
		assert.Contains(t, result, "  </div>")
	})

	t.Run("empty lines in template are preserved with indentation", func(t *testing.T) {
		sfcResult := &sfcparser.ParseResult{
			Template: "<div>\n\n  <p>Hello</p>\n</div>",
		}
		formattedTemplate := "<div>\n\n  <p>Hello</p>\n</div>"

		result := reassembleSFC(sfcResult, formattedTemplate, nil, nil, nil)

		lines := strings.Split(result, "\n")

		hasEmptyLine := slices.Contains(lines, "")
		assert.True(t, hasEmptyLine)
	})
}

func TestReassembleSFC_BlockSpacing(t *testing.T) {
	t.Run("blank line before script block", func(t *testing.T) {
		sfcResult := &sfcparser.ParseResult{
			Template: "<div>Hello</div>",
			Scripts: []sfcparser.Script{
				{Content: "package main"},
			},
		}
		formattedTemplate := "<div>Hello</div>"
		formattedScripts := sfcResult.Scripts

		result := reassembleSFC(sfcResult, formattedTemplate, formattedScripts, nil, nil)

		assert.Contains(t, result, "</template>\n\n<script>")
	})

	t.Run("blank line before style block", func(t *testing.T) {
		sfcResult := &sfcparser.ParseResult{
			Styles: []sfcparser.Style{
				{Content: ".container {}"},
			},
		}

		result := reassembleSFC(sfcResult, "", nil, sfcResult.Styles, nil)

		assert.True(t, strings.HasPrefix(result, "\n<style>"))
	})

	t.Run("blank line before i18n block", func(t *testing.T) {
		sfcResult := &sfcparser.ParseResult{
			I18nBlocks: []sfcparser.I18nBlock{
				{Content: "{}"},
			},
		}

		result := reassembleSFC(sfcResult, "", nil, nil, sfcResult.I18nBlocks)

		assert.True(t, strings.HasPrefix(result, "\n<i18n>"))
	})
}
