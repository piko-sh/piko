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

package render_domain

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	qt "github.com/valyala/quicktemplate"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/registry/registry_dto"
)

func TestRenderPikoPicture_BasicRendering(t *testing.T) {
	rctx := NewTestRenderContextBuilder().Build()
	ro := NewTestOrchestratorBuilder().Build()

	node := &ast_domain.TemplateNode{
		TagName: "piko:picture",
		Attributes: []ast_domain.HTMLAttribute{
			{Name: "src", Value: "github.com/example/assets/hero.jpg"},
			{Name: "alt", Value: "Hero image"},
		},
	}

	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	err := renderPikoPicture(ro, node, qw, rctx)
	require.NoError(t, err)

	output := buffer.String()
	assert.Contains(t, output, "<picture>")
	assert.Contains(t, output, "</picture>")
	assert.Contains(t, output, "<img")
	assert.Contains(t, output, `src="/_piko/assets/github.com/example/assets/hero.jpg"`)
	assert.Contains(t, output, `alt="Hero image"`)
	assert.Contains(t, output, "/>")
}

func TestRenderPikoPicture_MissingSrc(t *testing.T) {
	rctx := NewTestRenderContextBuilder().Build()
	ro := NewTestOrchestratorBuilder().Build()

	node := &ast_domain.TemplateNode{
		TagName: "piko:picture",
		Attributes: []ast_domain.HTMLAttribute{
			{Name: "alt", Value: "Missing source"},
			{Name: "class", Value: "placeholder"},
		},
	}

	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	err := renderPikoPicture(ro, node, qw, rctx)
	require.NoError(t, err)

	output := buffer.String()
	assert.Contains(t, output, "<picture>")
	assert.Contains(t, output, "</picture>")
	assert.Contains(t, output, "<img")
	assert.Contains(t, output, `alt="Missing source"`)
	assert.Contains(t, output, `class="placeholder"`)
	assert.NotContains(t, output, "src=")
}

func TestRenderPikoPicture_WithSingleFormatAndWidths(t *testing.T) {
	rctx := NewTestRenderContextBuilder().Build()
	ro := NewTestOrchestratorBuilder().Build()

	node := &ast_domain.TemplateNode{
		TagName: "piko:picture",
		Attributes: []ast_domain.HTMLAttribute{
			{Name: "src", Value: "github.com/example/assets/hero.jpg"},
			{Name: "widths", Value: "640, 1280"},
			{Name: "sizes", Value: "100vw"},
			{Name: "alt", Value: "Hero"},
		},
	}

	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	err := renderPikoPicture(ro, node, qw, rctx)
	require.NoError(t, err)

	output := buffer.String()

	assert.Contains(t, output, "<picture>")
	assert.Contains(t, output, "</picture>")

	assert.Contains(t, output, `<source type="image/webp"`)
	assert.Contains(t, output, `image_w640_webp 640w`)
	assert.Contains(t, output, `image_w1280_webp 1280w`)

	assert.Contains(t, output, `alt="Hero"`)
	assert.Contains(t, output, `sizes="100vw"`)
}

func TestRenderPikoPicture_MultiFormat(t *testing.T) {
	rctx := NewTestRenderContextBuilder().Build()
	ro := NewTestOrchestratorBuilder().Build()

	node := &ast_domain.TemplateNode{
		TagName: "piko:picture",
		Attributes: []ast_domain.HTMLAttribute{
			{Name: "src", Value: "github.com/example/assets/hero.jpg"},
			{Name: "widths", Value: "640, 1280"},
			{Name: "formats", Value: "avif, webp"},
			{Name: "sizes", Value: "100vw"},
			{Name: "alt", Value: "Hero"},
			{Name: "class", Value: "hero-img"},
		},
	}

	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	err := renderPikoPicture(ro, node, qw, rctx)
	require.NoError(t, err)

	output := buffer.String()

	assert.Contains(t, output, `<source type="image/avif"`)
	assert.Contains(t, output, `<source type="image/webp"`)

	assert.Contains(t, output, `image_w640_avif 640w`)
	assert.Contains(t, output, `image_w1280_avif 1280w`)

	assert.Contains(t, output, `image_w640_webp 640w`)
	assert.Contains(t, output, `image_w1280_webp 1280w`)

	assert.Contains(t, output, `alt="Hero"`)
	assert.Contains(t, output, `class="hero-img"`)

	assert.Contains(t, output, `sizes="100vw"`)
}

func TestRenderPikoPicture_FallbackAutoDetection(t *testing.T) {
	testCases := []struct {
		name           string
		src            string
		expectedFormat string
	}{
		{name: "JPEG source falls back to jpg", src: "images/photo.jpg", expectedFormat: "jpg"},
		{name: "JPEG extension", src: "images/photo.jpeg", expectedFormat: "jpg"},
		{name: "PNG source falls back to png", src: "images/icon.png", expectedFormat: "png"},
		{name: "GIF source falls back to png", src: "images/anim.gif", expectedFormat: "png"},
		{name: "WebP source falls back to png", src: "images/photo.webp", expectedFormat: "png"},
		{name: "No extension falls back to jpg", src: "images/photo", expectedFormat: "jpg"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := inferFallbackFormat(tc.src)
			assert.Equal(t, tc.expectedFormat, result)
		})
	}
}

func TestRenderPikoPicture_FiltersSpecialAttributes(t *testing.T) {
	rctx := NewTestRenderContextBuilder().Build()
	ro := NewTestOrchestratorBuilder().Build()

	node := &ast_domain.TemplateNode{
		TagName: "piko:picture",
		Attributes: []ast_domain.HTMLAttribute{
			{Name: "src", Value: "github.com/example/image.jpg"},
			{Name: "profile", Value: "thumbnail"},
			{Name: "densities", Value: "1x,2x"},
			{Name: "formats", Value: "avif,webp"},
			{Name: "widths", Value: "640,1280"},
			{Name: "variant", Value: "thumb"},
			{Name: "alt", Value: "test"},
			{Name: "class", Value: "my-class"},
			{Name: "loading", Value: "lazy"},
		},
	}

	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	err := renderPikoPicture(ro, node, qw, rctx)
	require.NoError(t, err)

	output := buffer.String()

	assert.NotContains(t, output, `profile=`)
	assert.NotContains(t, output, `densities=`)
	assert.NotContains(t, output, `formats=`)
	assert.NotContains(t, output, `widths=`)
	assert.NotContains(t, output, `variant=`)

	assert.Contains(t, output, `alt="test"`)
	assert.Contains(t, output, `class="my-class"`)
	assert.Contains(t, output, `loading="lazy"`)
}

func TestRenderPikoPicture_SizesOnSourceAndImg(t *testing.T) {
	rctx := NewTestRenderContextBuilder().Build()
	ro := NewTestOrchestratorBuilder().Build()

	node := &ast_domain.TemplateNode{
		TagName: "piko:picture",
		Attributes: []ast_domain.HTMLAttribute{
			{Name: "src", Value: "github.com/example/hero.jpg"},
			{Name: "widths", Value: "640"},
			{Name: "sizes", Value: "(max-width: 600px) 100vw, 50vw"},
			{Name: "alt", Value: "Test"},
		},
	}

	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	err := renderPikoPicture(ro, node, qw, rctx)
	require.NoError(t, err)

	output := buffer.String()

	sizesCount := strings.Count(output, `sizes="(max-width: 600px) 100vw, 50vw"`)
	assert.Equal(t, 2, sizesCount, "sizes should appear on both <source> and <img>")
}

func TestFormatToMIMEType(t *testing.T) {
	testCases := []struct {
		format   string
		expected string
	}{
		{"avif", "image/avif"},
		{"webp", "image/webp"},
		{"jpg", "image/jpeg"},
		{"jpeg", "image/jpeg"},
		{"png", "image/png"},
		{"gif", "image/gif"},
		{"tiff", "image/tiff"},
	}

	for _, tc := range testCases {
		t.Run(tc.format, func(t *testing.T) {
			result := formatToMIMEType(tc.format)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestAppendSrcsetForFormat(t *testing.T) {
	profiles := []registry_dto.NamedProfile{
		{
			Name: "image_w640_avif",
			Profile: registry_dto.DesiredProfile{
				ResultingTags: makeTagsWithWidth("640"),
			},
		},
		{
			Name: "image_w1280_avif",
			Profile: registry_dto.DesiredProfile{
				ResultingTags: makeTagsWithWidth("1280"),
			},
		},
		{
			Name: "image_w640_webp",
			Profile: registry_dto.DesiredProfile{
				ResultingTags: makeTagsWithWidth("640"),
			},
		},
		{
			Name: "image_w1280_webp",
			Profile: registry_dto.DesiredProfile{
				ResultingTags: makeTagsWithWidth("1280"),
			},
		},
	}

	baseURL := "/_piko/assets/hero.jpg"

	t.Run("filters to avif only", func(t *testing.T) {
		result := string(appendSrcsetForFormat(nil, profiles, baseURL, "avif"))
		assert.Contains(t, result, "image_w640_avif 640w")
		assert.Contains(t, result, "image_w1280_avif 1280w")
		assert.NotContains(t, result, "webp")
	})

	t.Run("filters to webp only", func(t *testing.T) {
		result := string(appendSrcsetForFormat(nil, profiles, baseURL, "webp"))
		assert.Contains(t, result, "image_w640_webp 640w")
		assert.Contains(t, result, "image_w1280_webp 1280w")
		assert.NotContains(t, result, "avif")
	})

	t.Run("returns empty for unknown format", func(t *testing.T) {
		result := appendSrcsetForFormat(nil, profiles, baseURL, "tiff")
		assert.Empty(t, result)
	})

	t.Run("empty profiles returns empty", func(t *testing.T) {
		result := appendSrcsetForFormat(nil, nil, baseURL, "webp")
		assert.Empty(t, result)
	})
}

func TestRenderPikoPicture_NoProfile(t *testing.T) {
	rctx := NewTestRenderContextBuilder().Build()
	ro := NewTestOrchestratorBuilder().Build()

	node := &ast_domain.TemplateNode{
		TagName: "piko:picture",
		Attributes: []ast_domain.HTMLAttribute{
			{Name: "src", Value: "github.com/example/photo.jpg"},
			{Name: "alt", Value: "Photo"},
		},
	}

	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	err := renderPikoPicture(ro, node, qw, rctx)
	require.NoError(t, err)

	output := buffer.String()

	assert.Contains(t, output, "<picture>")
	assert.Contains(t, output, "</picture>")
	assert.Contains(t, output, `src="/_piko/assets/github.com/example/photo.jpg"`)

	assert.NotContains(t, output, "<source")
	assert.NotContains(t, output, "srcset=")
}

func TestRenderPikoPicture_FallbackImgSrc(t *testing.T) {
	rctx := NewTestRenderContextBuilder().Build()
	ro := NewTestOrchestratorBuilder().Build()

	node := &ast_domain.TemplateNode{
		TagName: "piko:picture",
		Attributes: []ast_domain.HTMLAttribute{
			{Name: "src", Value: "github.com/example/hero.jpg"},
			{Name: "widths", Value: "640, 1280"},
			{Name: "formats", Value: "avif, webp"},
			{Name: "sizes", Value: "100vw"},
		},
	}

	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	err := renderPikoPicture(ro, node, qw, rctx)
	require.NoError(t, err)

	output := buffer.String()

	assert.Contains(t, output, `image_w1280_jpg`)
}

func makeTagsWithWidth(width string) registry_dto.Tags {
	var tags registry_dto.Tags
	tags.SetByName("width", width)
	return tags
}
