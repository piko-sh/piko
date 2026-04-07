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
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/esbuild/helpers"
	"piko.sh/piko/internal/esbuild/js_ast"
)

func TestIsAssetTag(t *testing.T) {
	testCases := []struct {
		name     string
		tagName  string
		expected bool
	}{
		{name: "piko:img lowercase", tagName: "piko:img", expected: true},
		{name: "piko:img uppercase", tagName: "PIKO:IMG", expected: true},
		{name: "piko:img mixed case", tagName: "Piko:Img", expected: true},
		{name: "piko:svg lowercase", tagName: "piko:svg", expected: true},
		{name: "piko:svg uppercase", tagName: "PIKO:SVG", expected: true},
		{name: "piko:picture lowercase", tagName: "piko:picture", expected: true},
		{name: "piko:picture uppercase", tagName: "PIKO:PICTURE", expected: true},
		{name: "div not asset tag", tagName: "div", expected: false},
		{name: "img not asset tag", tagName: "img", expected: false},
		{name: "piko:a not asset tag", tagName: "piko:a", expected: false},
		{name: "custom-element not asset tag", tagName: "custom-element", expected: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isAssetTag(tc.tagName)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestIsPikoImg(t *testing.T) {
	assert.True(t, isPikoImg("piko:img"))
	assert.True(t, isPikoImg("PIKO:IMG"))
	assert.False(t, isPikoImg("piko:svg"))
	assert.False(t, isPikoImg("img"))
}

func TestIsPikoSvg(t *testing.T) {
	assert.True(t, isPikoSvg("piko:svg"))
	assert.True(t, isPikoSvg("PIKO:SVG"))
	assert.False(t, isPikoSvg("piko:img"))
	assert.False(t, isPikoSvg("svg"))
}

func TestTransformAssetSrc(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "module path",
			input:    "github.com/user/app/assets/hero.png",
			expected: "/_piko/assets/github.com/user/app/assets/hero.png",
		},
		{
			name:     "relative path",
			input:    "assets/logo.svg",
			expected: "/_piko/assets/assets/logo.svg",
		},
		{
			name:     "already absolute",
			input:    "/static/image.png",
			expected: "/static/image.png",
		},
		{
			name:     "http URL",
			input:    "http://example.com/image.png",
			expected: "http://example.com/image.png",
		},
		{
			name:     "https URL",
			input:    "https://example.com/image.png",
			expected: "https://example.com/image.png",
		},
		{
			name:     "data URI",
			input:    "data:image/png;base64,ABC123",
			expected: "data:image/png;base64,ABC123",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "path needing cleaning",
			input:    "assets/../images/hero.png",
			expected: "/_piko/assets/images/hero.png",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := transformAssetSrc(tc.input, "")
			assert.Equal(t, tc.expected, result)
		})
	}

	t.Run("resolves @/ alias when module name provided", func(t *testing.T) {
		result := transformAssetSrc("@/lib/images/hero.png", "testmodule")
		assert.Equal(t, "/_piko/assets/testmodule/lib/images/hero.png", result)
	})

	t.Run("keeps @/ when no module name provided", func(t *testing.T) {
		result := transformAssetSrc("@/lib/images/hero.png", "")
		assert.Equal(t, "/_piko/assets/@/lib/images/hero.png", result)
	})
}

func TestParseCommaSeparated(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected []string
	}{
		{name: "single value", input: "webp", expected: []string{"webp"}},
		{name: "multiple values", input: "webp,jpeg,png", expected: []string{"webp", "jpeg", "png"}},
		{name: "values with spaces", input: " webp , jpeg , png ", expected: []string{"webp", "jpeg", "png"}},
		{name: "empty string", input: "", expected: nil},
		{name: "empty values filtered", input: "webp,,png", expected: []string{"webp", "png"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parseCommaSeparated(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestParseIntList(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected []int
	}{
		{name: "single width", input: "640", expected: []int{640}},
		{name: "multiple widths", input: "320,640,1024", expected: []int{320, 640, 1024}},
		{name: "with spaces", input: " 320 , 640 , 1024 ", expected: []int{320, 640, 1024}},
		{name: "invalid values skipped", input: "320,abc,640", expected: []int{320, 640}},
		{name: "zero skipped", input: "0,320,640", expected: []int{320, 640}},
		{name: "negative skipped", input: "-100,320,640", expected: []int{320, 640}},
		{name: "empty string", input: "", expected: []int{}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parseIntList(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestBuildSrcsetValue(t *testing.T) {
	testCases := []struct {
		name     string
		baseSrc  string
		attrs    pikoImgAttrs
		expected string
	}{
		{
			name:    "widths only default format",
			baseSrc: "/_piko/assets/hero.png",
			attrs: pikoImgAttrs{
				widths: "320,640",
			},
			expected: "/_piko/assets/hero.png?v=image_w320_webp 320w, /_piko/assets/hero.png?v=image_w640_webp 640w",
		},
		{
			name:    "widths with format",
			baseSrc: "/_piko/assets/hero.png",
			attrs: pikoImgAttrs{
				widths:  "320,640",
				formats: "avif",
			},
			expected: "/_piko/assets/hero.png?v=image_w320_avif 320w, /_piko/assets/hero.png?v=image_w640_avif 640w",
		},
		{
			name:    "widths with multiple formats",
			baseSrc: "/_piko/assets/hero.png",
			attrs: pikoImgAttrs{
				widths:  "320",
				formats: "webp,avif",
			},
			expected: "/_piko/assets/hero.png?v=image_w320_webp 320w, /_piko/assets/hero.png?v=image_w320_avif 320w",
		},
		{
			name:    "densities only",
			baseSrc: "/_piko/assets/hero.png",
			attrs: pikoImgAttrs{
				densities: "1x,2x",
			},
			expected: "/_piko/assets/hero.png?v=webp@1x 1x, /_piko/assets/hero.png?v=webp@2x 2x",
		},
		{
			name:    "densities with format",
			baseSrc: "/_piko/assets/hero.png",
			attrs: pikoImgAttrs{
				densities: "1x,2x",
				formats:   "avif",
			},
			expected: "/_piko/assets/hero.png?v=avif@1x 1x, /_piko/assets/hero.png?v=avif@2x 2x",
		},
		{
			name:     "no profile attributes",
			baseSrc:  "/_piko/assets/hero.png",
			attrs:    pikoImgAttrs{},
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := buildSrcsetValue(tc.baseSrc, tc.attrs)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestExtractPikoImgAttrs(t *testing.T) {
	t.Run("extracts static src", func(t *testing.T) {
		node := &ast_domain.TemplateNode{
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "src", Value: "github.com/user/app/hero.png"},
			},
		}
		attrs := extractPikoImgAttrs(node)
		assert.Equal(t, "github.com/user/app/hero.png", attrs.source)
	})

	t.Run("extracts responsive attributes", func(t *testing.T) {
		node := &ast_domain.TemplateNode{
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "src", Value: "hero.png"},
				{Name: "sizes", Value: "100vw"},
				{Name: "densities", Value: "1x,2x"},
				{Name: "formats", Value: "webp,avif"},
				{Name: "widths", Value: "320,640,1024"},
			},
		}
		attrs := extractPikoImgAttrs(node)
		assert.Equal(t, "hero.png", attrs.source)
		assert.Equal(t, "100vw", attrs.sizes)
		assert.Equal(t, "1x,2x", attrs.densities)
		assert.Equal(t, "webp,avif", attrs.formats)
		assert.Equal(t, "320,640,1024", attrs.widths)
	})

	t.Run("extracts dynamic src", func(t *testing.T) {
		node := &ast_domain.TemplateNode{
			DynamicAttributes: []ast_domain.DynamicAttribute{
				{Name: "src", Expression: &ast_domain.Identifier{Name: "imagePath"}},
			},
		}
		attrs := extractPikoImgAttrs(node)
		assert.Empty(t, attrs.source)
		assert.NotNil(t, attrs.dynamicSource)
	})
}

func TestExtractPikoSvgAttrs(t *testing.T) {
	t.Run("extracts static src", func(t *testing.T) {
		node := &ast_domain.TemplateNode{
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "src", Value: "icons/arrow.svg"},
			},
		}
		attrs := extractPikoSvgAttrs(node)
		assert.Equal(t, "icons/arrow.svg", attrs.source)
	})

	t.Run("extracts dynamic src", func(t *testing.T) {
		node := &ast_domain.TemplateNode{
			DynamicAttributes: []ast_domain.DynamicAttribute{
				{Name: "src", Expression: &ast_domain.Identifier{Name: "iconPath"}},
			},
		}
		attrs := extractPikoSvgAttrs(node)
		assert.Empty(t, attrs.source)
		assert.NotNil(t, attrs.dynamicSource)
	})
}

func TestPikoImgAttrs_HasProfile(t *testing.T) {
	t.Run("no profile attributes", func(t *testing.T) {
		attrs := pikoImgAttrs{source: "image.png"}
		assert.False(t, attrs.hasProfile())
	})

	t.Run("has sizes", func(t *testing.T) {
		attrs := pikoImgAttrs{source: "image.png", sizes: "100vw"}
		assert.True(t, attrs.hasProfile())
	})

	t.Run("has densities", func(t *testing.T) {
		attrs := pikoImgAttrs{source: "image.png", densities: "1x,2x"}
		assert.True(t, attrs.hasProfile())
	})

	t.Run("has formats", func(t *testing.T) {
		attrs := pikoImgAttrs{source: "image.png", formats: "webp"}
		assert.True(t, attrs.hasProfile())
	})

	t.Run("has widths", func(t *testing.T) {
		attrs := pikoImgAttrs{source: "image.png", widths: "320,640"}
		assert.True(t, attrs.hasProfile())
	})
}

func TestBuildPikoImgAST(t *testing.T) {
	ctx := context.Background()

	t.Run("piko:img with static src", func(t *testing.T) {
		registry := NewRegistryContext()
		events := newEventBindingCollection(registry)

		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "piko:img",
			Key:      &ast_domain.StringLiteral{Value: "img0"},
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "src", Value: "github.com/user/app/hero.png"},
				{Name: "alt", Value: "Hero image"},
			},
		}

		keyExpr := newStringLiteral("img0")
		result, err := buildPikoImgAST(ctx, node, &nodeBuildContext{events: events}, keyExpr)
		require.NoError(t, err)

		call, ok := result.Data.(*js_ast.ECall)
		require.True(t, ok)
		dot, ok := call.Target.Data.(*js_ast.EDot)
		require.True(t, ok)
		assert.Equal(t, "el", dot.Name)

		require.Len(t, call.Args, 4)
		tagArg, ok := call.Args[0].Data.(*js_ast.EString)
		require.True(t, ok)
		assert.Equal(t, "img", helpers.UTF16ToString(tagArg.Value))
	})

	t.Run("piko:img with srcset attributes", func(t *testing.T) {
		registry := NewRegistryContext()
		events := newEventBindingCollection(registry)

		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "piko:img",
			Key:      &ast_domain.StringLiteral{Value: "img0"},
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "src", Value: "hero.png"},
				{Name: "widths", Value: "320,640"},
				{Name: "formats", Value: "webp"},
			},
		}

		keyExpr := newStringLiteral("img0")
		result, err := buildPikoImgAST(ctx, node, &nodeBuildContext{events: events}, keyExpr)
		require.NoError(t, err)
		require.NotNil(t, result.Data)
	})
}

func TestBuildPikoSvgAST(t *testing.T) {
	ctx := context.Background()

	t.Run("piko:svg with static src", func(t *testing.T) {
		registry := NewRegistryContext()
		events := newEventBindingCollection(registry)

		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "piko:svg",
			Key:      &ast_domain.StringLiteral{Value: "svg0"},
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "src", Value: "icons/arrow.svg"},
				{Name: "class", Value: "icon"},
			},
		}

		keyExpr := newStringLiteral("svg0")
		result, err := buildPikoSvgAST(ctx, node, &nodeBuildContext{events: events}, keyExpr)
		require.NoError(t, err)

		call, ok := result.Data.(*js_ast.ECall)
		require.True(t, ok)
		dot, ok := call.Target.Data.(*js_ast.EDot)
		require.True(t, ok)
		assert.Equal(t, "el", dot.Name)

		require.Len(t, call.Args, 4)
		tagArg, ok := call.Args[0].Data.(*js_ast.EString)
		require.True(t, ok)
		assert.Equal(t, "piko-svg-inline", helpers.UTF16ToString(tagArg.Value))
	})
}

func TestIsPikoPicture(t *testing.T) {
	assert.True(t, isPikoPicture("piko:picture"))
	assert.True(t, isPikoPicture("PIKO:PICTURE"))
	assert.False(t, isPikoPicture("piko:img"))
	assert.False(t, isPikoPicture("picture"))
}

func TestBuildPikoPictureAST(t *testing.T) {
	ctx := context.Background()

	t.Run("piko:picture with static src", func(t *testing.T) {
		registry := NewRegistryContext()
		events := newEventBindingCollection(registry)

		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "piko:picture",
			Key:      &ast_domain.StringLiteral{Value: "pic0"},
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "src", Value: "github.com/user/app/hero.jpg"},
				{Name: "alt", Value: "Hero image"},
			},
		}

		keyExpr := newStringLiteral("pic0")
		result, err := buildPikoPictureAST(ctx, node, &nodeBuildContext{events: events}, keyExpr)
		require.NoError(t, err)

		call, ok := result.Data.(*js_ast.ECall)
		require.True(t, ok)
		dot, ok := call.Target.Data.(*js_ast.EDot)
		require.True(t, ok)
		assert.Equal(t, "el", dot.Name)

		require.Len(t, call.Args, 4)
		tagArg, ok := call.Args[0].Data.(*js_ast.EString)
		require.True(t, ok)
		assert.Equal(t, "picture", helpers.UTF16ToString(tagArg.Value))
	})

	t.Run("piko:picture with multi-format srcset", func(t *testing.T) {
		registry := NewRegistryContext()
		events := newEventBindingCollection(registry)

		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "piko:picture",
			Key:      &ast_domain.StringLiteral{Value: "pic0"},
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "src", Value: "hero.jpg"},
				{Name: "widths", Value: "640,1280"},
				{Name: "formats", Value: "avif,webp"},
				{Name: "sizes", Value: "100vw"},
			},
		}

		keyExpr := newStringLiteral("pic0")
		result, err := buildPikoPictureAST(ctx, node, &nodeBuildContext{events: events}, keyExpr)
		require.NoError(t, err)
		require.NotNil(t, result.Data)

		call, ok := result.Data.(*js_ast.ECall)
		require.True(t, ok)

		require.Len(t, call.Args, 4)
		childrenArg := call.Args[3]
		arr, ok := childrenArg.Data.(*js_ast.EArray)
		require.True(t, ok)

		assert.Equal(t, 3, len(arr.Items))
	})
}

func TestBuildAssetElementNodeAST(t *testing.T) {
	ctx := context.Background()

	t.Run("routes piko:img correctly", func(t *testing.T) {
		registry := NewRegistryContext()
		events := newEventBindingCollection(registry)

		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "piko:img",
			Key:      &ast_domain.StringLiteral{Value: "0"},
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "src", Value: "test.png"},
			},
		}

		keyExpr := newStringLiteral("0")
		result, err := buildAssetElementNodeAST(ctx, node, &nodeBuildContext{events: events}, keyExpr)
		require.NoError(t, err)
		require.NotNil(t, result.Data)
	})

	t.Run("routes piko:picture correctly", func(t *testing.T) {
		registry := NewRegistryContext()
		events := newEventBindingCollection(registry)

		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "piko:picture",
			Key:      &ast_domain.StringLiteral{Value: "0"},
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "src", Value: "test.jpg"},
			},
		}

		keyExpr := newStringLiteral("0")
		result, err := buildAssetElementNodeAST(ctx, node, &nodeBuildContext{events: events}, keyExpr)
		require.NoError(t, err)
		require.NotNil(t, result.Data)

		call, ok := result.Data.(*js_ast.ECall)
		require.True(t, ok)
		tagArg, ok := call.Args[0].Data.(*js_ast.EString)
		require.True(t, ok)
		assert.Equal(t, "picture", helpers.UTF16ToString(tagArg.Value))
	})

	t.Run("routes piko:svg correctly", func(t *testing.T) {
		registry := NewRegistryContext()
		events := newEventBindingCollection(registry)

		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "piko:svg",
			Key:      &ast_domain.StringLiteral{Value: "0"},
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "src", Value: "test.svg"},
			},
		}

		keyExpr := newStringLiteral("0")
		result, err := buildAssetElementNodeAST(ctx, node, &nodeBuildContext{events: events}, keyExpr)
		require.NoError(t, err)
		require.NotNil(t, result.Data)
	})
}

func TestBuildAssetSrcTransformCall(t *testing.T) {
	t.Run("wraps expression with piko.assets.resolve without module name", func(t *testing.T) {
		srcExpr := newIdentifier("imagePath")
		result := buildAssetSrcTransformCall(srcExpr, "")

		call, ok := result.Data.(*js_ast.ECall)
		require.True(t, ok)

		dot, ok := call.Target.Data.(*js_ast.EDot)
		require.True(t, ok)
		assert.Equal(t, "resolve", dot.Name)

		require.Len(t, call.Args, 1)
	})

	t.Run("includes module name as second argument when present", func(t *testing.T) {
		srcExpr := newIdentifier("imagePath")
		result := buildAssetSrcTransformCall(srcExpr, "testmodule")

		call, ok := result.Data.(*js_ast.ECall)
		require.True(t, ok)

		require.Len(t, call.Args, 2)

		strLit, ok := call.Args[1].Data.(*js_ast.EString)
		require.True(t, ok)
		assert.Equal(t, "testmodule", helpers.UTF16ToString(strLit.Value))
	})
}
