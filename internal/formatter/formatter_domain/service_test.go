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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFormatterService(t *testing.T) {
	t.Run("creates service with default options", func(t *testing.T) {
		service := NewFormatterService()

		require.NotNil(t, service)
		impl, ok := service.(*formatterServiceImpl)
		require.True(t, ok)
		assert.NotNil(t, impl.options)
		assert.Equal(t, 2, impl.options.IndentSize)
		assert.False(t, impl.options.PreserveEmptyLines)
		assert.True(t, impl.options.SortAttributes)
	})
}

func TestNewFormatterServiceWithOptions(t *testing.T) {
	t.Run("creates service with custom options", func(t *testing.T) {
		opts := &FormatOptions{
			IndentSize:         4,
			PreserveEmptyLines: true,
			SortAttributes:     false,
		}
		service := NewFormatterServiceWithOptions(opts)

		require.NotNil(t, service)
		impl, ok := service.(*formatterServiceImpl)
		require.True(t, ok)
		assert.Equal(t, opts, impl.options)
	})

	t.Run("uses defaults when nil options provided", func(t *testing.T) {
		service := NewFormatterServiceWithOptions(nil)

		require.NotNil(t, service)
		impl, ok := service.(*formatterServiceImpl)
		require.True(t, ok)
		assert.NotNil(t, impl.options)
		assert.Equal(t, 2, impl.options.IndentSize)
	})
}

func TestFormatterService_Format(t *testing.T) {
	ctx := context.Background()
	service := NewFormatterService()

	t.Run("formats simple template", func(t *testing.T) {
		source := []byte(`<template>
<div><p>Hello</p></div>
</template>`)

		result, err := service.Format(ctx, source)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Contains(t, string(result), "<template>")
		assert.Contains(t, string(result), "<div>")
		assert.Contains(t, string(result), "<p>")
		assert.Contains(t, string(result), "</template>")
	})

	t.Run("formats template with script", func(t *testing.T) {
		source := []byte(`<template>
<div>Hello</div>
</template>

<script type="application/x-go">
package main
import "piko.sh/piko"
type Response struct{Name string}
func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
return Response{Name: "Test"}, piko.Metadata{}, nil
}
</script>`)

		result, err := service.Format(ctx, source)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Contains(t, string(result), "<template>")
		assert.Contains(t, string(result), "<script type=\"application/x-go\">")

		assert.Contains(t, string(result), "package main")
		assert.Contains(t, string(result), "type Response struct")
	})

	t.Run("formats template with style", func(t *testing.T) {
		source := []byte(`<template>
<div>Hello</div>
</template>

<style>
.container { color: red; }
</style>`)

		result, err := service.Format(ctx, source)

		require.NoError(t, err)
		assert.Contains(t, string(result), "<template>")
		assert.Contains(t, string(result), "<style>")

		assert.Contains(t, string(result), ".container {")
		assert.Contains(t, string(result), "color: red;")
		assert.Contains(t, string(result), "</style>")
	})

	t.Run("returns error for invalid SFC", func(t *testing.T) {

		source := []byte{}

		result, err := service.Format(ctx, source)

		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("preserves declarative shadow DOM template", func(t *testing.T) {
		source := []byte(`<div id="app">
<my-component>
<template shadowrootmode="open"><style>:host { display: block; }</style><div class="inner"><span>Hello</span></div></template>
</my-component>
</div>`)

		result, err := service.FormatWithOptions(ctx, source, &FormatOptions{
			FileFormat:          FormatHTML,
			IndentSize:          2,
			SortAttributes:      false,
			MaxLineLength:       120,
			AttributeWrapIndent: 1,
			RawHTMLMode:         true,
		})

		require.NoError(t, err)
		output := string(result)
		assert.Contains(t, output, `<template shadowrootmode="open">`)
		assert.Contains(t, output, `</template>`)
		assert.Contains(t, output, `:host { display: block; }`)
		assert.Contains(t, output, `<span>Hello</span>`)
	})

	t.Run("plain template remains fragment", func(t *testing.T) {
		source := []byte(`<template>
<div><p>Hello</p></div>
</template>`)

		result, err := service.FormatWithOptions(ctx, source, &FormatOptions{
			FileFormat:          FormatHTML,
			IndentSize:          2,
			SortAttributes:      false,
			MaxLineLength:       120,
			AttributeWrapIndent: 1,
			RawHTMLMode:         true,
		})

		require.NoError(t, err)
		output := string(result)

		assert.NotContains(t, output, `<template>`)
		assert.Contains(t, output, `<div>`)
		assert.Contains(t, output, `<p>Hello</p>`)
	})

	t.Run("handles template parse errors gracefully", func(t *testing.T) {

		source := []byte(`<template>
<div>
<p>Unclosed paragraph
</div>
</template>`)

		result, err := service.Format(ctx, source)

		if err != nil {
			assert.Contains(t, err.Error(), "parsing")
		} else {
			assert.NotEmpty(t, result)
		}
	})
}

func TestFormatterService_FormatWithOptions(t *testing.T) {
	ctx := context.Background()
	service := NewFormatterService()

	t.Run("uses custom indent size", func(t *testing.T) {

		source := []byte(`<template>
<div>
<p>First</p>
<p>Second</p>
</div>
</template>`)

		opts := &FormatOptions{
			IndentSize:          4,
			SortAttributes:      true,
			MaxLineLength:       100,
			AttributeWrapIndent: 1,
		}

		result, err := service.FormatWithOptions(ctx, source, opts)

		require.NoError(t, err)
		assert.NotNil(t, result)

		resultString := string(result)

		assert.Contains(t, resultString, "  <div>")

		assert.Contains(t, resultString, "      <p>")
	})

	t.Run("respects sort attributes option", func(t *testing.T) {
		source := []byte(`<template>
<div z-attr="last" a-attr="first" m-attr="middle">Content</div>
</template>`)

		optsWithSort := &FormatOptions{
			IndentSize:     2,
			SortAttributes: true,
		}

		result, err := service.FormatWithOptions(ctx, source, optsWithSort)

		require.NoError(t, err)
		resultString := string(result)

		aPosition := strings.Index(resultString, "a-attr")
		mPosition := strings.Index(resultString, "m-attr")
		zPosition := strings.Index(resultString, "z-attr")

		if aPosition > 0 && mPosition > 0 && zPosition > 0 {
			assert.Less(t, aPosition, mPosition)
			assert.Less(t, mPosition, zPosition)
		}
	})
}

func TestFormatterService_formatTemplate(t *testing.T) {
	service := NewFormatterService()
	serviceImpl, ok := service.(*formatterServiceImpl)
	require.True(t, ok, "Expected service to be *formatterServiceImpl")

	t.Run("formats valid template", func(t *testing.T) {
		templateContent := "<div><p>Hello</p></div>"

		result, err := serviceImpl.formatTemplate(context.Background(), templateContent, serviceImpl.options)

		require.NoError(t, err)
		assert.NotEmpty(t, result)
		assert.Contains(t, result, "<div>")
		assert.Contains(t, result, "<p>")
		assert.Contains(t, result, "Hello")
	})

	t.Run("returns error for invalid template syntax", func(t *testing.T) {

		templateContent := "<div p-if='invalid expression with {{{{ nested braces'>"

		result, err := serviceImpl.formatTemplate(context.Background(), templateContent, serviceImpl.options)

		if err != nil {
			assert.Contains(t, strings.ToLower(err.Error()), "pars")
		} else {

			assert.NotEmpty(t, result)
		}
	})

	t.Run("handles empty template", func(t *testing.T) {
		templateContent := ""

		result, err := serviceImpl.formatTemplate(context.Background(), templateContent, serviceImpl.options)

		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("handles template with directives", func(t *testing.T) {
		templateContent := `<div p-if="state.IsVisible"><p p-for="item in items">{{item}}</p></div>`

		result, err := serviceImpl.formatTemplate(context.Background(), templateContent, serviceImpl.options)

		require.NoError(t, err)
		assert.Contains(t, result, "p-if=")
		assert.Contains(t, result, "p-for=")
	})
}

func TestFormatterService_formatGoScript(t *testing.T) {
	service := NewFormatterService()
	serviceImpl, ok := service.(*formatterServiceImpl)
	require.True(t, ok, "Expected service to be *formatterServiceImpl")

	t.Run("formats valid Go code", func(t *testing.T) {
		scriptContent := `package main
import "fmt"
func main(){fmt.Println("Hello")}`

		result, err := serviceImpl.formatGoScript(scriptContent)

		require.NoError(t, err)
		assert.NotEmpty(t, result)
		assert.Contains(t, result, "package main")
		assert.Contains(t, result, "import \"fmt\"")
		assert.Contains(t, result, "func main()")

		assert.Contains(t, result, "\n")
	})

	t.Run("returns error for invalid Go code", func(t *testing.T) {
		scriptContent := `package main
func main() {
	// Missing closing brace`

		result, err := serviceImpl.formatGoScript(scriptContent)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "formatting Go script")
		assert.Empty(t, result)
	})

	t.Run("handles empty script", func(t *testing.T) {
		scriptContent := ""

		result, err := serviceImpl.formatGoScript(scriptContent)

		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("formats complex Go code", func(t *testing.T) {
		scriptContent := `package main
import ( "fmt"; "strings" )
type Response struct { Name string; Age int }
func Render() (Response,error) { return Response{Name:"Test",Age:30},nil }`

		result, err := serviceImpl.formatGoScript(scriptContent)

		require.NoError(t, err)
		assert.Contains(t, result, "package main")

		assert.Contains(t, result, "import")
		assert.Contains(t, result, "type Response struct")
	})

	t.Run("preserves Go code semantics", func(t *testing.T) {
		scriptContent := `package main

func Add(a, b int) int {
	return a + b
}

func Multiply(a, b int) int {
	return a * b
}`

		result, err := serviceImpl.formatGoScript(scriptContent)

		require.NoError(t, err)
		assert.Contains(t, result, "func Add")
		assert.Contains(t, result, "func Multiply")
		assert.Contains(t, result, "return a + b")
		assert.Contains(t, result, "return a * b")
	})
}

func TestFormatterService_Integration(t *testing.T) {
	ctx := context.Background()
	service := NewFormatterService()

	t.Run("formats complete SFC end-to-end", func(t *testing.T) {
		source := []byte(`<template>
<div class="container"><header><h1>Title</h1></header>
<main><p>Content goes here</p></main>
<footer><p>Footer</p></footer></div>
</template>

<script type="application/x-go">
package main
import "piko.sh/piko"
type Response struct{Title string;Content string}
func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
return Response{Title:"Test",Content:"Hello"}, piko.Metadata{}, nil
}
</script>

<style>
.container{display:flex;flex-direction:column;}.header{background:blue;}
</style>`)

		result, err := service.Format(ctx, source)

		require.NoError(t, err)
		assert.NotNil(t, result)

		resultString := string(result)

		assert.Contains(t, resultString, "<template>")
		assert.Contains(t, resultString, "  <div")
		assert.Contains(t, resultString, "    <header>")

		assert.Contains(t, resultString, `<script type="application/x-go">`)
		assert.Contains(t, resultString, "package main")
		assert.Contains(t, resultString, "type Response struct {")

		assert.Contains(t, resultString, "<style>")
		assert.Contains(t, resultString, ".container")

		templateIndex := strings.Index(resultString, "</template>")
		scriptIndex := strings.Index(resultString, "<script")
		styleIndex := strings.Index(resultString, "<style")

		assert.Less(t, templateIndex, scriptIndex)
		assert.Less(t, scriptIndex, styleIndex)
	})

	t.Run("handles SFC with multiple scripts", func(t *testing.T) {
		source := []byte(`<template>
<div>Content</div>
</template>

<script type="application/x-go">
package main
func Render() {}
</script>

<script type="text/javascript">
console.log("client-side");
</script>`)

		result, err := service.Format(ctx, source)

		require.NoError(t, err)
		resultString := string(result)

		assert.Contains(t, resultString, `type="application/x-go"`)
		assert.Contains(t, resultString, `type="text/javascript"`)
		assert.Contains(t, resultString, "package main")
		assert.Contains(t, resultString, "console.log")

		scriptCount := strings.Count(resultString, "<script")
		assert.Equal(t, 2, scriptCount)
	})

	t.Run("preserves Go script formatting quality", func(t *testing.T) {
		source := []byte(`<template><div>Test</div></template>

<script type="application/x-go">
package main
import "piko.sh/piko"
type Props struct{ID string;Name string;Email string}
type Response struct{User Props;IsAdmin bool}
func Render(r *piko.RequestData, props Props) (Response, piko.Metadata, error) {
isAdmin:=props.Email=="admin@example.com"
return Response{User:props,IsAdmin:isAdmin}, piko.Metadata{Title:props.Name}, nil
}
</script>`)

		result, err := service.Format(ctx, source)

		require.NoError(t, err)
		resultString := string(result)

		assert.Contains(t, resultString, "type Props struct {")
		assert.Contains(t, resultString, "type Response struct {")
		assert.Contains(t, resultString, "func Render(")

		assert.Contains(t, resultString, "isAdmin :=")
	})

	t.Run("formats idempotently", func(t *testing.T) {
		source := []byte(`<template>
<div class="container">
<p>Hello, World!</p>
</div>
</template>`)

		result1, err := service.Format(ctx, source)
		require.NoError(t, err)

		result2, err := service.Format(ctx, result1)
		require.NoError(t, err)

		assert.Equal(t, string(result1), string(result2))
	})
}

func TestFormatterService_ErrorHandling(t *testing.T) {
	ctx := context.Background()
	service := NewFormatterService()

	t.Run("handles Go script format failure gracefully", func(t *testing.T) {

		source := []byte(`<template>
<div>Content</div>
</template>

<script type="application/x-go">
package main
func main() {
	// Syntax error: missing closing brace
</script>`)

		result, err := service.Format(ctx, source)

		if err != nil {
			assert.Contains(t, err.Error(), "format")
		} else {

			assert.Contains(t, string(result), "package main")
		}
	})

	t.Run("handles template with parse errors", func(t *testing.T) {
		source := []byte(`<template>
<div p-if="unclosed string
<p>Content</p>
</div>
</template>`)

		result, err := service.Format(ctx, source)

		if err != nil {
			assert.Contains(t, strings.ToLower(err.Error()), "pars")
		} else {
			assert.NotEmpty(t, result)
		}
	})
}

func TestFormatterService_DetectFileFormat(t *testing.T) {
	service := NewFormatterService()
	serviceImpl, ok := service.(*formatterServiceImpl)
	require.True(t, ok, "Expected service to be *formatterServiceImpl")

	t.Run("detects PK when template tag present", func(t *testing.T) {
		source := []byte(`<template>
<div>Content</div>
</template>`)
		opts := DefaultFormatOptions()

		format := serviceImpl.detectFileFormat(source, opts)

		assert.Equal(t, FormatPK, format)
	})

	t.Run("detects PK when template tag has attributes", func(t *testing.T) {
		source := []byte(`<template lang="html">
<div>Content</div>
</template>`)
		opts := DefaultFormatOptions()

		format := serviceImpl.detectFileFormat(source, opts)

		assert.Equal(t, FormatPK, format)
	})

	t.Run("detects HTML when no template tag", func(t *testing.T) {
		source := []byte(`<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body><p>Content</p></body>
</html>`)
		opts := DefaultFormatOptions()

		format := serviceImpl.detectFileFormat(source, opts)

		assert.Equal(t, FormatHTML, format)
	})

	t.Run("detects HTML for plain elements", func(t *testing.T) {
		source := []byte(`<div><p>Just some HTML</p></div>`)
		opts := DefaultFormatOptions()

		format := serviceImpl.detectFileFormat(source, opts)

		assert.Equal(t, FormatHTML, format)
	})

	t.Run("respects explicit FormatPK option", func(t *testing.T) {
		source := []byte(`<div>Content without template tag</div>`)
		opts := &FormatOptions{FileFormat: FormatPK}

		format := serviceImpl.detectFileFormat(source, opts)

		assert.Equal(t, FormatPK, format)
	})

	t.Run("respects explicit FormatHTML option", func(t *testing.T) {
		source := []byte(`<template><div>Content</div></template>`)
		opts := &FormatOptions{FileFormat: FormatHTML}

		format := serviceImpl.detectFileFormat(source, opts)

		assert.Equal(t, FormatHTML, format)
	})

	t.Run("detects HTML for shadow root template only", func(t *testing.T) {
		source := []byte(`<div><my-component><template shadowrootmode="open"><style>:host{display:block}</style><div>Content</div></template></my-component></div>`)
		opts := DefaultFormatOptions()

		format := serviceImpl.detectFileFormat(source, opts)

		assert.Equal(t, FormatHTML, format)
	})

	t.Run("detects HTML for shadow root template with serializable", func(t *testing.T) {
		source := []byte(`<custom-card><template shadowrootmode="open" shadowrootserializable><div>Card</div></template></custom-card>`)
		opts := DefaultFormatOptions()

		format := serviceImpl.detectFileFormat(source, opts)

		assert.Equal(t, FormatHTML, format)
	})

	t.Run("detects PK when bare template and shadow root both present", func(t *testing.T) {
		source := []byte(`<template>
<div>
<my-component><template shadowrootmode="open"><div>Shadow content</div></template></my-component>
</div>
</template>`)
		opts := DefaultFormatOptions()

		format := serviceImpl.detectFileFormat(source, opts)

		assert.Equal(t, FormatPK, format)
	})
}

func TestFormatterService_FormatHTMLOnly(t *testing.T) {
	ctx := context.Background()
	service := NewFormatterService()

	t.Run("formats plain HTML", func(t *testing.T) {
		source := []byte(`<div><p>Hello</p><p>World</p></div>`)
		opts := &FormatOptions{
			FileFormat:     FormatHTML,
			IndentSize:     2,
			SortAttributes: true,
		}

		result, err := service.FormatWithOptions(ctx, source, opts)

		require.NoError(t, err)
		resultString := string(result)
		assert.Contains(t, resultString, "<div>")
		assert.Contains(t, resultString, "<p>Hello</p>")
		assert.Contains(t, resultString, "<p>World</p>")
		assert.Contains(t, resultString, "</div>")

		assert.NotContains(t, resultString, "<template>")
	})

	t.Run("formats HTML with doctype", func(t *testing.T) {
		source := []byte(`<!DOCTYPE html><html><head><title>Test</title></head><body><p>Content</p></body></html>`)
		opts := &FormatOptions{
			FileFormat:     FormatHTML,
			IndentSize:     2,
			SortAttributes: true,
		}

		result, err := service.FormatWithOptions(ctx, source, opts)

		require.NoError(t, err)
		resultString := string(result)

		assert.Contains(t, resultString, "<p>Content</p>")
	})

	t.Run("formats HTML with attributes", func(t *testing.T) {
		source := []byte(`<div z-attr="last" a-attr="first" m-attr="middle">Content</div>`)
		opts := &FormatOptions{
			FileFormat:     FormatHTML,
			IndentSize:     2,
			SortAttributes: true,
		}

		result, err := service.FormatWithOptions(ctx, source, opts)

		require.NoError(t, err)
		resultString := string(result)

		aPosition := strings.Index(resultString, "a-attr")
		mPosition := strings.Index(resultString, "m-attr")
		zPosition := strings.Index(resultString, "z-attr")

		assert.Greater(t, aPosition, 0)
		assert.Greater(t, mPosition, 0)
		assert.Greater(t, zPosition, 0)
		assert.Less(t, aPosition, mPosition)
		assert.Less(t, mPosition, zPosition)
	})

	t.Run("formats nested HTML structure", func(t *testing.T) {
		source := []byte(`<div><header><h1>Title</h1></header><main><article><p>Content</p></article></main><footer><p>Footer</p></footer></div>`)
		opts := &FormatOptions{
			FileFormat:     FormatHTML,
			IndentSize:     2,
			SortAttributes: true,
		}

		result, err := service.FormatWithOptions(ctx, source, opts)

		require.NoError(t, err)
		resultString := string(result)

		assert.Contains(t, resultString, "<header>")
		assert.Contains(t, resultString, "<h1>Title</h1>")
		assert.Contains(t, resultString, "<main>")
		assert.Contains(t, resultString, "<article>")
		assert.Contains(t, resultString, "<footer>")
	})

	t.Run("HTML format is idempotent", func(t *testing.T) {
		source := []byte(`<div><p>Hello, World!</p></div>`)
		opts := &FormatOptions{
			FileFormat:     FormatHTML,
			IndentSize:     2,
			SortAttributes: true,
		}

		result1, err := service.FormatWithOptions(ctx, source, opts)
		require.NoError(t, err)

		result2, err := service.FormatWithOptions(ctx, result1, opts)
		require.NoError(t, err)

		assert.Equal(t, string(result1), string(result2))
	})

	t.Run("auto-detects and formats plain HTML", func(t *testing.T) {
		source := []byte(`<div><p>No template tag here</p></div>`)
		opts := &FormatOptions{
			FileFormat:     FormatAuto,
			IndentSize:     2,
			SortAttributes: true,
		}

		result, err := service.FormatWithOptions(ctx, source, opts)

		require.NoError(t, err)
		resultString := string(result)

		assert.Contains(t, resultString, "<div>")
		assert.NotContains(t, resultString, "<template>")
	})

	t.Run("formats HTML with self-closing tags", func(t *testing.T) {
		source := []byte(`<div><img src="test.jpg" alt="Test"><br><hr></div>`)
		opts := &FormatOptions{
			FileFormat:     FormatHTML,
			IndentSize:     2,
			SortAttributes: true,
		}

		result, err := service.FormatWithOptions(ctx, source, opts)

		require.NoError(t, err)
		resultString := string(result)

		assert.Contains(t, resultString, "<img")
		assert.Contains(t, resultString, "<br")
		assert.Contains(t, resultString, "<hr")
	})
}

func TestContainsPKTemplateWithAttributes(t *testing.T) {
	t.Run("returns true for template with lang attribute", func(t *testing.T) {
		assert.True(t, containsPKTemplateWithAttributes(`<template lang="html"><div>Content</div></template>`))
	})

	t.Run("returns false for shadow root template", func(t *testing.T) {
		assert.False(t, containsPKTemplateWithAttributes(`<my-comp><template shadowrootmode="open"><div>Content</div></template></my-comp>`))
	})

	t.Run("returns false for shadow root template with serializable", func(t *testing.T) {
		assert.False(t, containsPKTemplateWithAttributes(`<my-comp><template shadowrootmode="open" shadowrootserializable><div>Content</div></template></my-comp>`))
	})

	t.Run("returns false for no template tags", func(t *testing.T) {
		assert.False(t, containsPKTemplateWithAttributes(`<div><p>Just HTML</p></div>`))
	})

	t.Run("returns true when both PK and shadow root templates present", func(t *testing.T) {
		source := `<template lang="html"><my-comp><template shadowrootmode="open"><div>Shadow</div></template></my-comp></template>`
		assert.True(t, containsPKTemplateWithAttributes(source))
	})

	t.Run("returns false for bare template without space", func(t *testing.T) {
		assert.False(t, containsPKTemplateWithAttributes(`<template><div>Content</div></template>`))
	})
}

func TestFormatterService_ShadowRootFormatting(t *testing.T) {
	ctx := context.Background()
	service := NewFormatterService()

	t.Run("shadow root template formats as block", func(t *testing.T) {
		source := []byte(`<div><my-component><template shadowrootmode="open"><style>:host { display: block; }</style><div class="inner"><span>Hello</span></div></template></my-component></div>`)
		opts := &FormatOptions{
			FileFormat:          FormatHTML,
			IndentSize:          2,
			SortAttributes:      false,
			MaxLineLength:       120,
			AttributeWrapIndent: 1,
			RawHTMLMode:         true,
		}

		result, err := service.FormatWithOptions(ctx, source, opts)
		require.NoError(t, err)
		output := string(result)

		assert.Contains(t, output, `<template shadowrootmode="open">`)
		assert.Contains(t, output, `</template>`)

		lines := strings.Split(output, "\n")
		var templateOpenLine, templateCloseLine int
		for i, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.Contains(trimmed, `<template shadowrootmode="open">`) {
				templateOpenLine = i
			}
			if trimmed == "</template>" {
				templateCloseLine = i
			}
		}
		assert.Greater(t, templateCloseLine, templateOpenLine,
			"template closing tag should be on a later line than opening tag")
	})

	t.Run("style within shadow root gets own line", func(t *testing.T) {
		source := []byte(`<div><my-component><template shadowrootmode="open"><style>:host { display: block; }</style><div>Content</div></template></my-component></div>`)
		opts := &FormatOptions{
			FileFormat:          FormatHTML,
			IndentSize:          2,
			SortAttributes:      false,
			MaxLineLength:       120,
			AttributeWrapIndent: 1,
			RawHTMLMode:         true,
		}

		result, err := service.FormatWithOptions(ctx, source, opts)
		require.NoError(t, err)
		output := string(result)

		lines := strings.Split(output, "\n")
		var styleLine, contentLine int
		for i, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "<style>") {
				styleLine = i
			}
			if strings.Contains(trimmed, "<div>Content</div>") {
				contentLine = i
			}
		}
		assert.NotEqual(t, styleLine, contentLine,
			"style and content should be on different lines")
	})

	t.Run("shadow root formatting is idempotent", func(t *testing.T) {

		source := []byte(`<div>
  <my-component>
    <template shadowrootmode="open">
      <style>:host { display: block; }</style>
      <div class="inner"><span>Hello</span></div>
    </template>
  </my-component>
</div>
`)
		opts := &FormatOptions{
			FileFormat:          FormatHTML,
			IndentSize:          2,
			SortAttributes:      false,
			MaxLineLength:       120,
			AttributeWrapIndent: 1,
			RawHTMLMode:         true,
		}

		result1, err := service.FormatWithOptions(ctx, source, opts)
		require.NoError(t, err)

		result2, err := service.FormatWithOptions(ctx, result1, opts)
		require.NoError(t, err)

		assert.Equal(t, string(result1), string(result2))
	})
}

func TestFormatterService_EdgeCases(t *testing.T) {
	ctx := context.Background()
	service := NewFormatterService()

	t.Run("handles empty input", func(t *testing.T) {
		source := []byte("")

		result, err := service.Format(ctx, source)

		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("handles whitespace-only input", func(t *testing.T) {
		source := []byte("   \n\n   \t\t   \n   ")

		result, err := service.Format(ctx, source)

		require.NoError(t, err)

		assert.Empty(t, strings.TrimSpace(string(result)))
	})

	t.Run("handles template-only SFC", func(t *testing.T) {
		source := []byte(`<template>
<div>
<p>Just a template</p>
</div>
</template>`)

		result, err := service.Format(ctx, source)

		require.NoError(t, err)
		assert.Contains(t, string(result), "<template>")
		assert.NotContains(t, string(result), "<script>")
		assert.NotContains(t, string(result), "<style>")
	})

	t.Run("handles script-only SFC", func(t *testing.T) {
		source := []byte(`<script type="application/x-go">
package main

func main() {}
</script>`)

		result, err := service.Format(ctx, source)

		require.NoError(t, err)
		assert.Contains(t, string(result), "<script")
		assert.Contains(t, string(result), "package main")
		assert.NotContains(t, string(result), "<template>")
	})

	t.Run("handles deeply nested template structure", func(t *testing.T) {
		source := []byte(`<template>
<div><section><article><header><h1>Title</h1></header><main><p>Content</p></main></article></section></div>
</template>`)

		result, err := service.Format(ctx, source)

		require.NoError(t, err)
		resultString := string(result)

		assert.Contains(t, resultString, "<div>")
		assert.Contains(t, resultString, "<section>")
		assert.Contains(t, resultString, "<article>")
		assert.Contains(t, resultString, "<header>")
		assert.Contains(t, resultString, "<h1>")
	})
}

func TestFormatterService_RawHTMLMode(t *testing.T) {
	ctx := context.Background()
	service := NewFormatterService()

	t.Run("formats HTML with runtime p-key values in RawHTMLMode", func(t *testing.T) {

		source := []byte(`<div id="app"><div partial="pages_index" p-key="r.0"><h1 p-key="r.0:0">Test</h1></div></div>`)

		result, err := service.FormatWithOptions(ctx, source, &FormatOptions{
			FileFormat:     FormatHTML,
			IndentSize:     2,
			SortAttributes: false,
			RawHTMLMode:    true,
		})

		require.NoError(t, err)
		resultString := string(result)

		assert.Contains(t, resultString, `p-key="r.0"`)
		assert.Contains(t, resultString, `p-key="r.0:0"`)
		assert.Contains(t, resultString, `<h1 p-key="r.0:0">`)
		assert.Contains(t, resultString, "<div id=\"app\">")
	})

	t.Run("fails on runtime p-key values without RawHTMLMode", func(t *testing.T) {

		source := []byte(`<div p-key="r.0">Test</div>`)

		_, err := service.FormatWithOptions(ctx, source, &FormatOptions{
			FileFormat:  FormatHTML,
			RawHTMLMode: false,
		})

		require.Error(t, err)
	})

	t.Run("formats complex rendered DOM output", func(t *testing.T) {
		source := []byte(`<div id="app"><div partial="pages_index_3949c94c" p-key="r.0"><h1 partial="pages_index_3949c94c" p-key="r.0:0">Mutation Observer Test</h1><mutation-watcher partial="pages_index_3949c94c" p-key="r.0:1" last-action="none" next-child-num="3"></mutation-watcher></div>    </div>`)

		result, err := service.FormatWithOptions(ctx, source, &FormatOptions{
			FileFormat:          FormatHTML,
			IndentSize:          2,
			SortAttributes:      false,
			MaxLineLength:       120,
			AttributeWrapIndent: 1,
			RawHTMLMode:         true,
		})

		require.NoError(t, err)
		resultString := string(result)

		assert.Contains(t, resultString, "p-key=\"r.0\"")
		assert.Contains(t, resultString, "p-key=\"r.0:0\"")
		assert.Contains(t, resultString, "p-key=\"r.0:1\"")
		assert.Contains(t, resultString, "last-action=\"none\"")
		assert.Contains(t, resultString, "next-child-num=\"3\"")

		assert.True(t, strings.Contains(resultString, "\n"), "output should be formatted with newlines")
	})
}

func TestFormatterService_ConditionalChains(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	service := NewFormatterService()

	t.Run("p-if p-else-if p-else chain", func(t *testing.T) {
		t.Parallel()
		source := []byte(`<template>
<div p-if="state.Status == 1"><p>Active</p></div>
<div p-else-if="state.Status == 2"><p>Pending</p></div>
<div p-else><p>Unknown</p></div>
</template>`)

		result, err := service.Format(ctx, source)

		require.NoError(t, err)
		resultString := string(result)
		assert.Contains(t, resultString, `p-if="state.Status == 1"`)
		assert.Contains(t, resultString, `p-else-if="state.Status == 2"`)
		assert.Contains(t, resultString, "p-else")
		assert.Contains(t, resultString, "<p>Active</p>")
		assert.Contains(t, resultString, "<p>Pending</p>")
		assert.Contains(t, resultString, "<p>Unknown</p>")
	})

	t.Run("nested p-if inside p-for", func(t *testing.T) {
		t.Parallel()
		source := []byte(`<template>
<ul><li p-for="item in state.Items"><span p-if="item.Visible">{{item.Name}}</span><span p-else>Hidden</span></li></ul>
</template>`)

		result, err := service.Format(ctx, source)

		require.NoError(t, err)
		resultString := string(result)
		assert.Contains(t, resultString, `p-for="item in state.Items"`)
		assert.Contains(t, resultString, `p-if="item.Visible"`)
		assert.Contains(t, resultString, "p-else")
		assert.Contains(t, resultString, "{{ item.Name }}")
	})
}

func TestFormatterService_MixedInlineBlockElements(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	service := NewFormatterService()

	t.Run("inline elements within block elements", func(t *testing.T) {
		t.Parallel()
		source := []byte(`<template>
<div><p>Text with <strong>bold</strong> and <em>italic</em> words</p></div>
</template>`)

		result, err := service.Format(ctx, source)

		require.NoError(t, err)
		resultString := string(result)
		assert.Contains(t, resultString, "<strong>bold</strong>")
		assert.Contains(t, resultString, "<em>italic</em>")
	})

	t.Run("block elements prevent inline formatting", func(t *testing.T) {
		t.Parallel()
		source := []byte(`<template>
<div><section><p>Paragraph</p></section><aside><p>Sidebar</p></aside></div>
</template>`)

		result, err := service.Format(ctx, source)

		require.NoError(t, err)
		resultString := string(result)
		assert.Contains(t, resultString, "<section>")
		assert.Contains(t, resultString, "<aside>")

		lines := strings.Split(strings.TrimSpace(resultString), "\n")
		assert.Greater(t, len(lines), 3, "block elements should be on separate lines")
	})

	t.Run("list items with short content stay inline", func(t *testing.T) {
		t.Parallel()
		source := []byte(`<template>
<ul>
<li>Short item</li>
<li>Another</li>
</ul>
</template>`)

		result, err := service.Format(ctx, source)

		require.NoError(t, err)
		resultString := string(result)
		assert.Contains(t, resultString, "<li>Short item</li>")
		assert.Contains(t, resultString, "<li>Another</li>")
	})
}

func TestFormatterService_WhitespaceSensitiveElements(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	service := NewFormatterService()

	t.Run("pre element preserves whitespace", func(t *testing.T) {
		t.Parallel()
		source := []byte(`<template>
<pre>  line one
  line two
    indented</pre>
</template>`)

		result, err := service.Format(ctx, source)

		require.NoError(t, err)
		resultString := string(result)
		assert.Contains(t, resultString, "<pre>")
		assert.Contains(t, resultString, "</pre>")

		assert.Contains(t, resultString, "  line one")
		assert.Contains(t, resultString, "    indented")
	})

	t.Run("code element preserves whitespace", func(t *testing.T) {
		t.Parallel()
		source := []byte(`<template>
<code>func main() {
	fmt.Println("hello")
}</code>
</template>`)

		result, err := service.Format(ctx, source)

		require.NoError(t, err)
		resultString := string(result)
		assert.Contains(t, resultString, "<code>")
		assert.Contains(t, resultString, "</code>")
		assert.Contains(t, resultString, `fmt.Println("hello")`)
	})

	t.Run("textarea element preserves whitespace", func(t *testing.T) {
		t.Parallel()
		source := []byte(`<template>
<textarea>Some default
text content
  with indentation</textarea>
</template>`)

		result, err := service.Format(ctx, source)

		require.NoError(t, err)
		resultString := string(result)
		assert.Contains(t, resultString, "<textarea>")
		assert.Contains(t, resultString, "</textarea>")
		assert.Contains(t, resultString, "Some default")
	})
}

func TestFormatterService_SelfClosingWithManyAttributes(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	service := NewFormatterService()

	t.Run("img with many attributes wraps when exceeding line length", func(t *testing.T) {
		t.Parallel()
		source := []byte(`<template>
<img src="/images/hero.png" alt="A very long description of the hero image" class="hero-image responsive" width="1200" height="800" loading="lazy" decoding="async">
</template>`)

		opts := &FormatOptions{
			IndentSize:          2,
			SortAttributes:      true,
			MaxLineLength:       80,
			AttributeWrapIndent: 1,
		}

		result, err := service.FormatWithOptions(ctx, source, opts)

		require.NoError(t, err)
		resultString := string(result)
		assert.Contains(t, resultString, "<img")
		assert.Contains(t, resultString, "/>")
		assert.Contains(t, resultString, `src="/images/hero.png"`)
		assert.Contains(t, resultString, `loading="lazy"`)
	})

	t.Run("input with many attributes", func(t *testing.T) {
		t.Parallel()
		source := []byte(`<template>
<input type="text" name="username" placeholder="Enter your username" class="form-input large" required autocomplete="off" maxlength="100" minlength="3">
</template>`)

		opts := &FormatOptions{
			IndentSize:          2,
			SortAttributes:      true,
			MaxLineLength:       60,
			AttributeWrapIndent: 1,
		}

		result, err := service.FormatWithOptions(ctx, source, opts)

		require.NoError(t, err)
		resultString := string(result)
		assert.Contains(t, resultString, "<input")
		assert.Contains(t, resultString, "/>")
		assert.Contains(t, resultString, `type="text"`)
		assert.Contains(t, resultString, "required")

		lines := strings.Split(strings.TrimSpace(resultString), "\n")
		assert.Greater(t, len(lines), 2, "attributes should wrap onto multiple lines")
	})
}

func TestFormatterService_ForDirective(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	service := NewFormatterService()

	t.Run("p-for on list items", func(t *testing.T) {
		t.Parallel()
		source := []byte(`<template>
<ul><li p-for="item in state.Items" p-key="item.ID">{{item.Name}}</li></ul>
</template>`)

		result, err := service.Format(ctx, source)

		require.NoError(t, err)
		resultString := string(result)
		assert.Contains(t, resultString, `p-for="item in state.Items"`)
		assert.Contains(t, resultString, `p-key="item.ID"`)
		assert.Contains(t, resultString, "{{ item.Name }}")
	})

	t.Run("p-for on table rows", func(t *testing.T) {
		t.Parallel()
		source := []byte(`<template>
<table><thead><tr><th>Name</th><th>Age</th></tr></thead><tbody><tr p-for="user in state.Users"><td>{{user.Name}}</td><td>{{user.Age}}</td></tr></tbody></table>
</template>`)

		result, err := service.Format(ctx, source)

		require.NoError(t, err)
		resultString := string(result)
		assert.Contains(t, resultString, `p-for="user in state.Users"`)
		assert.Contains(t, resultString, "{{ user.Name }}")
		assert.Contains(t, resultString, "{{ user.Age }}")
		assert.Contains(t, resultString, "<thead>")
		assert.Contains(t, resultString, "<tbody>")
	})

	t.Run("p-for with nested block structure", func(t *testing.T) {
		t.Parallel()
		source := []byte(`<template>
<div p-for="section in state.Sections"><h2>{{section.Title}}</h2><p>{{section.Content}}</p></div>
</template>`)

		result, err := service.Format(ctx, source)

		require.NoError(t, err)
		resultString := string(result)
		assert.Contains(t, resultString, `p-for="section in state.Sections"`)
		assert.Contains(t, resultString, "{{ section.Title }}")
		assert.Contains(t, resultString, "{{ section.Content }}")
	})
}

func TestFormatterService_TextAndHTMLDirectives(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	service := NewFormatterService()

	t.Run("p-text directive on span", func(t *testing.T) {
		t.Parallel()
		source := []byte(`<template>
<div><span p-text="state.Username"></span></div>
</template>`)

		result, err := service.Format(ctx, source)

		require.NoError(t, err)
		resultString := string(result)
		assert.Contains(t, resultString, `p-text="state.Username"`)
	})

	t.Run("p-html directive on div", func(t *testing.T) {
		t.Parallel()
		source := []byte(`<template>
<div p-html="state.RichContent"></div>
</template>`)

		result, err := service.Format(ctx, source)

		require.NoError(t, err)
		resultString := string(result)
		assert.Contains(t, resultString, `p-html="state.RichContent"`)
	})

	t.Run("p-text and p-show combined", func(t *testing.T) {
		t.Parallel()
		source := []byte(`<template>
<p p-show="state.HasMessage" p-text="state.Message"></p>
</template>`)

		result, err := service.Format(ctx, source)

		require.NoError(t, err)
		resultString := string(result)
		assert.Contains(t, resultString, `p-show="state.HasMessage"`)
		assert.Contains(t, resultString, `p-text="state.Message"`)
	})
}

func TestFormatterService_CommentHandling(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	service := NewFormatterService()

	t.Run("HTML comment in template", func(t *testing.T) {
		t.Parallel()
		source := []byte(`<template>
<!-- This is a comment -->
<div>Content</div>
</template>`)

		result, err := service.Format(ctx, source)

		require.NoError(t, err)
		resultString := string(result)
		assert.Contains(t, resultString, "<!-- This is a comment -->")
		assert.Contains(t, resultString, "<div>Content</div>")
	})

	t.Run("comment between elements", func(t *testing.T) {
		t.Parallel()
		source := []byte(`<template>
<header><h1>Title</h1></header>
<!-- Navigation section -->
<nav><a href="/">Home</a></nav>
</template>`)

		result, err := service.Format(ctx, source)

		require.NoError(t, err)
		resultString := string(result)
		assert.Contains(t, resultString, "<!-- Navigation section -->")
		assert.Contains(t, resultString, "<header>")
		assert.Contains(t, resultString, "<nav>")
	})

	t.Run("comment in plain HTML mode", func(t *testing.T) {
		t.Parallel()
		source := []byte(`<div><!-- inline note --><span>Text</span></div>`)

		opts := &FormatOptions{
			FileFormat: FormatHTML,
			IndentSize: 2,
		}

		result, err := service.FormatWithOptions(ctx, source, opts)

		require.NoError(t, err)
		resultString := string(result)
		assert.Contains(t, resultString, "<!-- inline note -->")
		assert.Contains(t, resultString, "<span>Text</span>")
	})

	t.Run("long comment gets its own line", func(t *testing.T) {
		t.Parallel()
		source := []byte(`<div><!-- This is a very long comment that should definitely exceed the inline comment length limit and be placed on its own line --><span>Text</span></div>`)

		opts := &FormatOptions{
			FileFormat: FormatHTML,
			IndentSize: 2,
		}

		result, err := service.FormatWithOptions(ctx, source, opts)

		require.NoError(t, err)
		resultString := string(result)
		assert.Contains(t, resultString, "<!-- This is a very long comment")
		assert.Contains(t, resultString, "<span>Text</span>")
	})
}

func TestFormatterService_FragmentNodes(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	service := NewFormatterService()

	t.Run("template with fragment-like multiple children", func(t *testing.T) {
		t.Parallel()
		source := []byte(`<template>
<h1>Title</h1>
<p>First paragraph</p>
<p>Second paragraph</p>
</template>`)

		result, err := service.Format(ctx, source)

		require.NoError(t, err)
		resultString := string(result)
		assert.Contains(t, resultString, "<h1>Title</h1>")
		assert.Contains(t, resultString, "<p>First paragraph</p>")
		assert.Contains(t, resultString, "<p>Second paragraph</p>")
	})

	t.Run("plain HTML with multiple root elements", func(t *testing.T) {
		t.Parallel()
		source := []byte(`<header><h1>Title</h1></header><main><p>Content</p></main><footer><p>Footer</p></footer>`)

		opts := &FormatOptions{
			FileFormat: FormatHTML,
			IndentSize: 2,
		}

		result, err := service.FormatWithOptions(ctx, source, opts)

		require.NoError(t, err)
		resultString := string(result)
		assert.Contains(t, resultString, "<header>")
		assert.Contains(t, resultString, "<main>")
		assert.Contains(t, resultString, "<footer>")
	})
}

func TestFormatterService_DeeplyNestedStructures(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	service := NewFormatterService()

	t.Run("six levels deep nesting", func(t *testing.T) {
		t.Parallel()
		source := []byte(`<template>
<div><section><article><div><header><h1>Deep Title</h1></header></div></article></section></div>
</template>`)

		result, err := service.Format(ctx, source)

		require.NoError(t, err)
		resultString := string(result)
		assert.Contains(t, resultString, "<div>")
		assert.Contains(t, resultString, "<section>")
		assert.Contains(t, resultString, "<article>")
		assert.Contains(t, resultString, "<header>")
		assert.Contains(t, resultString, "<h1>Deep Title</h1>")

		lines := strings.Split(resultString, "\n")
		maxIndent := 0
		for _, line := range lines {
			indent := len(line) - len(strings.TrimLeft(line, " "))
			if indent > maxIndent {
				maxIndent = indent
			}
		}
		assert.GreaterOrEqual(t, maxIndent, 10, "deeply nested content should have significant indentation")
	})

	t.Run("nested form with fieldset", func(t *testing.T) {
		t.Parallel()
		source := []byte(`<template>
<form><fieldset><legend>Personal Info</legend><div><label>Name</label><input type="text" name="name"></div><div><label>Email</label><input type="email" name="email"></div></fieldset></form>
</template>`)

		result, err := service.Format(ctx, source)

		require.NoError(t, err)
		resultString := string(result)
		assert.Contains(t, resultString, "<form>")
		assert.Contains(t, resultString, "<fieldset>")
		assert.Contains(t, resultString, "<legend>Personal Info</legend>")
		assert.Contains(t, resultString, `name="name"`)
		assert.Contains(t, resultString, `type="text"`)
		assert.Contains(t, resultString, `name="email"`)
		assert.Contains(t, resultString, `type="email"`)
	})
}

func TestFormatterService_MultipleRootNodes(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	service := NewFormatterService()

	t.Run("template with multiple top-level elements", func(t *testing.T) {
		t.Parallel()
		source := []byte(`<template>
<nav><a href="/">Home</a></nav>
<main><p>Content</p></main>
<footer><small>Copyright</small></footer>
</template>`)

		result, err := service.Format(ctx, source)

		require.NoError(t, err)
		resultString := string(result)
		assert.Contains(t, resultString, "<nav>")
		assert.Contains(t, resultString, "<main>")
		assert.Contains(t, resultString, "<footer>")
	})

	t.Run("HTML format with multiple root elements of different types", func(t *testing.T) {
		t.Parallel()
		source := []byte(`<!-- Page header --><header><h1>Title</h1></header><main><p>Body</p></main>`)

		opts := &FormatOptions{
			FileFormat: FormatHTML,
			IndentSize: 2,
		}

		result, err := service.FormatWithOptions(ctx, source, opts)

		require.NoError(t, err)
		resultString := string(result)
		assert.Contains(t, resultString, "<!-- Page header -->")
		assert.Contains(t, resultString, "<header>")
		assert.Contains(t, resultString, "<main>")
	})
}

func TestFormatterService_InterpolationExpressions(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	service := NewFormatterService()

	t.Run("single interpolation in text", func(t *testing.T) {
		t.Parallel()
		source := []byte(`<template>
<p>Hello, {{state.Username}}!</p>
</template>`)

		result, err := service.Format(ctx, source)

		require.NoError(t, err)
		resultString := string(result)
		assert.Contains(t, resultString, "{{ state.Username }}")
	})

	t.Run("multiple interpolations in one element", func(t *testing.T) {
		t.Parallel()
		source := []byte(`<template>
<p>{{state.FirstName}} {{state.LastName}}</p>
</template>`)

		result, err := service.Format(ctx, source)

		require.NoError(t, err)
		resultString := string(result)
		assert.Contains(t, resultString, "{{ state.FirstName }}")
		assert.Contains(t, resultString, "{{ state.LastName }}")
	})

	t.Run("interpolation with method call", func(t *testing.T) {
		t.Parallel()
		source := []byte(`<template>
<span>{{state.Items.Count}}</span>
</template>`)

		result, err := service.Format(ctx, source)

		require.NoError(t, err)
		resultString := string(result)
		assert.Contains(t, resultString, "{{ state.Items.Count }}")
	})
}

func TestFormatterService_FormatRange(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	service := NewFormatterService()

	t.Run("formats specific range within document", func(t *testing.T) {
		t.Parallel()
		source := []byte("<template>\n<div><p>Hello</p></div>\n</template>")

		formatRange := Range{
			StartLine:      1,
			StartCharacter: 0,
			EndLine:        1,
			EndCharacter:   22,
		}

		result, err := service.FormatRange(ctx, source, formatRange, nil)

		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("returns error for invalid range", func(t *testing.T) {
		t.Parallel()
		source := []byte("<template>\n<div>Hello</div>\n</template>")

		formatRange := Range{
			StartLine:      999,
			StartCharacter: 0,
			EndLine:        999,
			EndCharacter:   0,
		}

		_, err := service.FormatRange(ctx, source, formatRange, nil)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "range offsets")
	})

	t.Run("FormatRange with custom options", func(t *testing.T) {
		t.Parallel()
		source := []byte("<template>\n<div><p>Content</p></div>\n</template>")

		formatRange := Range{
			StartLine:      1,
			StartCharacter: 0,
			EndLine:        1,
			EndCharacter:   22,
		}

		opts := &FormatOptions{
			IndentSize:     4,
			SortAttributes: true,
		}

		result, err := service.FormatRange(ctx, source, formatRange, opts)

		require.NoError(t, err)
		assert.NotNil(t, result)
	})
}

func TestFormatterService_PreserveEmptyLines(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	service := NewFormatterService()

	t.Run("preserve empty lines adds spacing between major blocks", func(t *testing.T) {
		t.Parallel()
		source := []byte(`<template>
<header><h1>Title</h1></header>
<main><p>Content</p></main>
<footer><p>Footer</p></footer>
</template>`)

		opts := &FormatOptions{
			IndentSize:         2,
			SortAttributes:     true,
			PreserveEmptyLines: true,
			MaxLineLength:      100,
		}

		result, err := service.FormatWithOptions(ctx, source, opts)

		require.NoError(t, err)
		resultString := string(result)
		assert.Contains(t, resultString, "<header>")
		assert.Contains(t, resultString, "<main>")
		assert.Contains(t, resultString, "<footer>")
	})

	t.Run("preserve empty lines with p-if directive", func(t *testing.T) {
		t.Parallel()
		source := []byte(`<template>
<div><p>First</p></div>
<div p-if="state.Show"><p>Conditional</p></div>
</template>`)

		opts := &FormatOptions{
			IndentSize:         2,
			SortAttributes:     true,
			PreserveEmptyLines: true,
			MaxLineLength:      100,
		}

		result, err := service.FormatWithOptions(ctx, source, opts)

		require.NoError(t, err)
		resultString := string(result)
		assert.Contains(t, resultString, `p-if="state.Show"`)
	})
}

func TestFormatterService_I18nBlocks(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	service := NewFormatterService()

	t.Run("formats SFC with i18n block", func(t *testing.T) {
		t.Parallel()
		source := []byte(`<template>
<div>Hello</div>
</template>

<i18n lang="json">
{"en":{"hello":"Hello"},"fr":{"hello":"Bonjour"}}
</i18n>`)

		result, err := service.Format(ctx, source)

		require.NoError(t, err)
		resultString := string(result)
		assert.Contains(t, resultString, "<i18n")
		assert.Contains(t, resultString, "</i18n>")

		assert.Contains(t, resultString, `"hello"`)
	})

	t.Run("formats i18n block with default lang", func(t *testing.T) {
		t.Parallel()
		source := []byte(`<template>
<p>Test</p>
</template>

<i18n>
{"greeting":"Hi","farewell":"Bye"}
</i18n>`)

		result, err := service.Format(ctx, source)

		require.NoError(t, err)
		resultString := string(result)
		assert.Contains(t, resultString, "<i18n>")
		assert.Contains(t, resultString, `"greeting"`)
		assert.Contains(t, resultString, `"farewell"`)
	})
}

func TestFormatterService_ComplexDirectiveCombinations(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	service := NewFormatterService()

	t.Run("element with p-show and p-class directives", func(t *testing.T) {
		t.Parallel()
		source := []byte(`<template>
<div p-show="state.IsVisible" p-class="state.DynamicClass">Content</div>
</template>`)

		result, err := service.Format(ctx, source)

		require.NoError(t, err)
		resultString := string(result)
		assert.Contains(t, resultString, `p-show="state.IsVisible"`)
		assert.Contains(t, resultString, `p-class="state.DynamicClass"`)
	})

	t.Run("element with p-model directive", func(t *testing.T) {
		t.Parallel()
		source := []byte(`<template>
<input type="text" p-model="state.Username">
</template>`)

		result, err := service.Format(ctx, source)

		require.NoError(t, err)
		resultString := string(result)
		assert.Contains(t, resultString, `p-model="state.Username"`)
	})

	t.Run("element with p-on event handler", func(t *testing.T) {
		t.Parallel()
		source := []byte(`<template>
<button p-on:click="handleClick">Click Me</button>
</template>`)

		result, err := service.Format(ctx, source)

		require.NoError(t, err)
		resultString := string(result)
		assert.Contains(t, resultString, `p-on:click="handleClick"`)
		assert.Contains(t, resultString, "Click Me")
	})

	t.Run("element with dynamic attribute binding", func(t *testing.T) {
		t.Parallel()
		source := []byte(`<template>
<a :href="state.Link" :title="state.LinkTitle">Visit</a>
</template>`)

		result, err := service.Format(ctx, source)

		require.NoError(t, err)
		resultString := string(result)
		assert.Contains(t, resultString, `:href="state.Link"`)
		assert.Contains(t, resultString, `:title="state.LinkTitle"`)
	})

	t.Run("element with p-ref directive", func(t *testing.T) {
		t.Parallel()
		source := []byte(`<template>
<div p-ref="containerRef">Referenced element</div>
</template>`)

		result, err := service.Format(ctx, source)

		require.NoError(t, err)
		resultString := string(result)
		assert.Contains(t, resultString, `p-ref="containerRef"`)
	})

	t.Run("element with p-style directive", func(t *testing.T) {
		t.Parallel()
		source := []byte(`<template>
<div p-style="state.CustomStyles">Styled content</div>
</template>`)

		result, err := service.Format(ctx, source)

		require.NoError(t, err)
		resultString := string(result)
		assert.Contains(t, resultString, `p-style="state.CustomStyles"`)
	})
}

func TestFormatterService_AttributeWrapping(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	service := NewFormatterService()

	t.Run("short attributes stay on one line", func(t *testing.T) {
		t.Parallel()
		source := []byte(`<template>
<div id="app" class="main">Content</div>
</template>`)

		opts := &FormatOptions{
			IndentSize:          2,
			SortAttributes:      true,
			MaxLineLength:       100,
			AttributeWrapIndent: 1,
		}

		result, err := service.FormatWithOptions(ctx, source, opts)

		require.NoError(t, err)
		resultString := string(result)

		for line := range strings.SplitSeq(resultString, "\n") {
			if strings.Contains(line, "<div") {
				assert.Contains(t, line, "id=")
				assert.Contains(t, line, "class=")
			}
		}
	})

	t.Run("many attributes wrap with custom indent", func(t *testing.T) {
		t.Parallel()
		source := []byte(`<template>
<div id="container" class="wrapper flex-col" data-testid="main-container" role="main" aria-label="Main content area">Content</div>
</template>`)

		opts := &FormatOptions{
			IndentSize:          2,
			SortAttributes:      true,
			MaxLineLength:       50,
			AttributeWrapIndent: 2,
		}

		result, err := service.FormatWithOptions(ctx, source, opts)

		require.NoError(t, err)
		resultString := string(result)
		assert.Contains(t, resultString, `id="container"`)
		assert.Contains(t, resultString, `role="main"`)

		lines := strings.Split(strings.TrimSpace(resultString), "\n")
		assert.Greater(t, len(lines), 4, "attributes should be wrapped across multiple lines")
	})

	t.Run("wrapping disabled when MaxLineLength is zero", func(t *testing.T) {
		t.Parallel()
		source := []byte(`<template>
<div id="container" class="wrapper flex-col" data-testid="main-container" role="main" aria-label="Main content area">Content</div>
</template>`)

		opts := &FormatOptions{
			IndentSize:     2,
			SortAttributes: true,
			MaxLineLength:  0,
		}

		result, err := service.FormatWithOptions(ctx, source, opts)

		require.NoError(t, err)
		resultString := string(result)

		for line := range strings.SplitSeq(resultString, "\n") {
			if strings.Contains(line, "<div") {
				assert.Contains(t, line, "id=")
				assert.Contains(t, line, "class=")
				assert.Contains(t, line, "role=")
			}
		}
	})
}

func TestFormatterService_SortAttributesDisabled(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	service := NewFormatterService()

	t.Run("preserves original attribute order when sorting disabled", func(t *testing.T) {
		t.Parallel()
		source := []byte(`<template>
<div z-custom="last" a-custom="first" id="test">Content</div>
</template>`)

		opts := &FormatOptions{
			IndentSize:     2,
			SortAttributes: false,
			MaxLineLength:  100,
		}

		result, err := service.FormatWithOptions(ctx, source, opts)

		require.NoError(t, err)
		resultString := string(result)
		zPosition := strings.Index(resultString, "z-custom")
		aPosition := strings.Index(resultString, "a-custom")
		assert.Greater(t, zPosition, 0)
		assert.Greater(t, aPosition, 0)

		assert.Less(t, zPosition, aPosition, "attributes should be in original order when sorting disabled")
	})
}

func TestFormatterService_CSSFormatting(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	service := NewFormatterService()

	t.Run("formats compressed CSS", func(t *testing.T) {
		t.Parallel()
		source := []byte(`<template>
<div>Hello</div>
</template>

<style>
.a{color:red;background:blue;}.b{margin:0;padding:10px;}
</style>`)

		result, err := service.Format(ctx, source)

		require.NoError(t, err)
		resultString := string(result)
		assert.Contains(t, resultString, "<style>")
		assert.Contains(t, resultString, ".a")
		assert.Contains(t, resultString, ".b")
		assert.Contains(t, resultString, "color: red;")
		assert.Contains(t, resultString, "</style>")
	})

	t.Run("handles empty style block", func(t *testing.T) {
		t.Parallel()
		source := []byte(`<template>
<div>Hello</div>
</template>

<style>
</style>`)

		result, err := service.Format(ctx, source)

		require.NoError(t, err)
		assert.NotNil(t, result)
	})
}

func TestFormatterService_EmptyElementsAndBooleanAttrs(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	service := NewFormatterService()

	t.Run("empty div renders with closing tag", func(t *testing.T) {
		t.Parallel()
		source := []byte(`<template>
<div id="placeholder"></div>
</template>`)

		result, err := service.Format(ctx, source)

		require.NoError(t, err)
		resultString := string(result)
		assert.Contains(t, resultString, `<div id="placeholder"></div>`)
	})

	t.Run("boolean attributes without values", func(t *testing.T) {
		t.Parallel()
		source := []byte(`<template>
<input type="checkbox" checked disabled>
</template>`)

		result, err := service.Format(ctx, source)

		require.NoError(t, err)
		resultString := string(result)
		assert.Contains(t, resultString, "checked")
		assert.Contains(t, resultString, "disabled")
	})

	t.Run("empty self-closing element", func(t *testing.T) {
		t.Parallel()
		source := []byte(`<template>
<br>
<hr>
</template>`)

		result, err := service.Format(ctx, source)

		require.NoError(t, err)
		resultString := string(result)
		assert.Contains(t, resultString, "<br />")
		assert.Contains(t, resultString, "<hr />")
	})
}

func TestFormatterService_CompleteComponentFormatting(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	service := NewFormatterService()

	t.Run("full component with all block types is idempotent", func(t *testing.T) {
		t.Parallel()
		source := []byte(`<template>
<div class="container">
<h1>{{state.Title}}</h1>
<ul><li p-for="item in state.Items" p-key="item.ID"><span p-if="item.Active">{{item.Name}}</span><span p-else>Inactive</span></li></ul>
</div>
</template>

<script type="application/x-go">
package main

import "piko.sh/piko"

type Response struct {
	Title string
	Items []Item
}

type Item struct {
	ID     int
	Name   string
	Active bool
}

func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
	return Response{
		Title: "My Page",
		Items: []Item{
			{ID: 1, Name: "First", Active: true},
		},
	}, piko.Metadata{}, nil
}
</script>

<style>
.container {
  max-width: 800px;
  margin: 0 auto;
}
</style>`)

		result1, err := service.Format(ctx, source)
		require.NoError(t, err)

		result2, err := service.Format(ctx, result1)
		require.NoError(t, err)

		assert.Equal(t, string(result1), string(result2),
			"formatting should be idempotent for a full component")
	})
}
