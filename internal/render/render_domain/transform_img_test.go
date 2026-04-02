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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	qt "github.com/valyala/quicktemplate"
	"piko.sh/piko/internal/assetpath"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/registry/registry_dto"
)

func TestAppendTransformedSrc_ModuleAbsolutePath(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple module path",
			input:    "github.com/example/assets/image.png",
			expected: "/_piko/assets/github.com/example/assets/image.png",
		},
		{
			name:     "nested directory structure",
			input:    "github.com/org/repo/assets/images/hero.jpg",
			expected: "/_piko/assets/github.com/org/repo/assets/images/hero.jpg",
		},
		{
			name:     "path with special chars in filename",
			input:    "github.com/example/assets/my-image_v2.png",
			expected: "/_piko/assets/github.com/example/assets/my-image_v2.png",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := string(assetpath.AppendTransformed(nil, tc.input, assetpath.DefaultServePath))
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestAppendTransformedSrc_SkipsTransformation(t *testing.T) {
	testCases := []struct {
		name  string
		input string
	}{
		{
			name:  "already transformed path",
			input: "/_piko/assets/github.com/example/assets/image.png",
		},
		{
			name:  "HTTP URL",
			input: "http://cdn.example.com/image.png",
		},
		{
			name:  "HTTPS URL",
			input: "https://cdn.example.com/image.png",
		},
		{
			name:  "protocol-relative URL",
			input: "//cdn.example.com/image.png",
		},
		{
			name:  "data URI with base64",
			input: "data:image/png;base64,iVBORw0KGgoAAAANSUhEUg...",
		},
		{
			name:  "data URI with SVG",
			input: "data:image/svg+xml,%3Csvg%3E%3C/svg%3E",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := string(assetpath.AppendTransformed(nil, tc.input, assetpath.DefaultServePath))
			assert.Equal(t, tc.input, result, "input should be returned unchanged")
		})
	}
}

func TestAppendTransformedSrc_CleansPath(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "trailing slash removed",
			input:    "github.com/example/assets/",
			expected: "/_piko/assets/github.com/example/assets",
		},
		{
			name:     "double slashes cleaned",
			input:    "github.com/example//assets/image.png",
			expected: "/_piko/assets/github.com/example/assets/image.png",
		},
		{
			name:     "dot segments cleaned",
			input:    "github.com/example/./assets/../assets/image.png",
			expected: "/_piko/assets/github.com/example/assets/image.png",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := string(assetpath.AppendTransformed(nil, tc.input, assetpath.DefaultServePath))
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestRenderPikoImg_BasicRendering(t *testing.T) {
	rctx := NewTestRenderContextBuilder().Build()
	ro := NewTestOrchestratorBuilder().Build()

	node := &ast_domain.TemplateNode{
		TagName: "piko:img",
		Attributes: []ast_domain.HTMLAttribute{
			{Name: "src", Value: "github.com/example/assets/image.png"},
			{Name: "alt", Value: "Test image"},
		},
	}

	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	err := renderPikoImg(ro, node, qw, rctx)
	require.NoError(t, err)

	output := buffer.String()
	assert.Contains(t, output, "<img")
	assert.Contains(t, output, `src="/_piko/assets/github.com/example/assets/image.png"`)
	assert.Contains(t, output, `alt="Test image"`)
	assert.Contains(t, output, "/>")
}

func TestRenderPikoImg_FiltersSpecialAttributes(t *testing.T) {
	testCases := []struct {
		name          string
		attrs         []ast_domain.HTMLAttribute
		notInOutput   []string
		shouldContain []string
	}{
		{
			name: "removes profile attribute",
			attrs: []ast_domain.HTMLAttribute{
				{Name: "src", Value: "github.com/example/image.png"},
				{Name: "profile", Value: "thumbnail"},
				{Name: "alt", Value: "test"},
			},
			notInOutput:   []string{`profile=`},
			shouldContain: []string{`alt="test"`},
		},
		{
			name: "removes densities attribute",
			attrs: []ast_domain.HTMLAttribute{
				{Name: "src", Value: "github.com/example/image.png"},
				{Name: "densities", Value: "1x,2x,3x"},
			},
			notInOutput: []string{`densities=`},
		},
		{
			name: "uses sizes attribute for responsive images",
			attrs: []ast_domain.HTMLAttribute{
				{Name: "src", Value: "github.com/example/image.png"},
				{Name: "sizes", Value: "(max-width: 600px) 100vw, 50vw"},
			},
			notInOutput:   []string{},
			shouldContain: []string{`sizes="(max-width: 600px) 100vw, 50vw"`, `srcset=`},
		},
		{
			name: "filters profile attrs but uses them for srcset",
			attrs: []ast_domain.HTMLAttribute{
				{Name: "src", Value: "github.com/example/image.png"},
				{Name: "profile", Value: "hero"},
				{Name: "densities", Value: "1x,2x"},
				{Name: "sizes", Value: "100vw"},
				{Name: "class", Value: "hero-image"},
				{Name: "width", Value: "800"},
				{Name: "height", Value: "600"},
			},
			notInOutput:   []string{`profile=`, `densities=`, `widths=`, `formats=`},
			shouldContain: []string{`class="hero-image"`, `width="800"`, `height="600"`, `sizes="100vw"`, `srcset=`},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rctx := NewTestRenderContextBuilder().Build()
			ro := NewTestOrchestratorBuilder().Build()

			node := &ast_domain.TemplateNode{
				TagName:    "piko:img",
				Attributes: tc.attrs,
			}

			var buffer bytes.Buffer
			qw := qt.AcquireWriter(&buffer)
			defer qt.ReleaseWriter(qw)

			err := renderPikoImg(ro, node, qw, rctx)
			require.NoError(t, err)

			output := buffer.String()
			for _, notExpected := range tc.notInOutput {
				assert.NotContains(t, output, notExpected)
			}
			for _, expected := range tc.shouldContain {
				assert.Contains(t, output, expected)
			}
		})
	}
}

func TestRenderPikoImg_SrcAttributeProcessing(t *testing.T) {

	rctx := NewTestRenderContextBuilder().Build()
	ro := NewTestOrchestratorBuilder().Build()

	node := &ast_domain.TemplateNode{
		TagName: "piko:img",
		Attributes: []ast_domain.HTMLAttribute{
			{Name: "src", Value: "github.com/example/image.png"},
		},
	}

	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	err := renderPikoImg(ro, node, qw, rctx)
	require.NoError(t, err)

	output := buffer.String()
	assert.Contains(t, output, `src="/_piko/assets/github.com/example/image.png"`)
}

func TestRenderPikoImg_NoSrcAttribute(t *testing.T) {
	rctx := NewTestRenderContextBuilder().Build()
	ro := NewTestOrchestratorBuilder().Build()

	node := &ast_domain.TemplateNode{
		TagName: "piko:img",
		Attributes: []ast_domain.HTMLAttribute{
			{Name: "alt", Value: "Missing source"},
			{Name: "class", Value: "placeholder"},
		},
	}

	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	err := renderPikoImg(ro, node, qw, rctx)
	require.NoError(t, err)

	output := buffer.String()
	assert.Contains(t, output, "<img")
	assert.Contains(t, output, `alt="Missing source"`)
	assert.Contains(t, output, `class="placeholder"`)
	assert.NotContains(t, output, "src=")
}

func TestRenderPikoImg_IsSelfClosing(t *testing.T) {
	rctx := NewTestRenderContextBuilder().Build()
	ro := NewTestOrchestratorBuilder().Build()

	node := &ast_domain.TemplateNode{
		TagName: "piko:img",
		Attributes: []ast_domain.HTMLAttribute{
			{Name: "src", Value: "github.com/example/image.png"},
		},
	}

	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	err := renderPikoImg(ro, node, qw, rctx)
	require.NoError(t, err)

	output := buffer.String()

	assert.True(t, output[len(output)-2:] == "/>" || output[len(output)-3:] == " />")
	assert.NotContains(t, output, "</img>")
}

func TestRenderPikoImg_PreservesAttributeOrder(t *testing.T) {
	rctx := NewTestRenderContextBuilder().Build()
	ro := NewTestOrchestratorBuilder().Build()

	node := &ast_domain.TemplateNode{
		TagName: "piko:img",
		Attributes: []ast_domain.HTMLAttribute{
			{Name: "class", Value: "first"},
			{Name: "src", Value: "github.com/example/image.png"},
			{Name: "alt", Value: "second"},
			{Name: "id", Value: "third"},
		},
	}

	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	err := renderPikoImg(ro, node, qw, rctx)
	require.NoError(t, err)

	output := buffer.String()

	assert.Contains(t, output, `src="/_piko/assets/github.com/example/image.png"`)
	assert.Contains(t, output, `class="first"`)
	assert.Contains(t, output, `alt="second"`)
	assert.Contains(t, output, `id="third"`)
}

func TestRenderPikoImg_ExternalURLUnchanged(t *testing.T) {
	rctx := NewTestRenderContextBuilder().Build()
	ro := NewTestOrchestratorBuilder().Build()

	node := &ast_domain.TemplateNode{
		TagName: "piko:img",
		Attributes: []ast_domain.HTMLAttribute{
			{Name: "src", Value: "https://cdn.example.com/image.png"},
		},
	}

	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	err := renderPikoImg(ro, node, qw, rctx)
	require.NoError(t, err)

	output := buffer.String()

	assert.Contains(t, output, `src="https://cdn.example.com/image.png"`)
	assert.NotContains(t, output, "/_piko/assets")
}

func TestRenderPikoImg_ProfileDensitiesSizesFiltering(t *testing.T) {

	testCases := []struct {
		name           string
		attributeName  string
		attributeValue string
		shouldBeOutput bool
	}{
		{name: "profile is filtered", attributeName: "profile", attributeValue: "large", shouldBeOutput: false},
		{name: "densities is filtered", attributeName: "densities", attributeValue: "2x", shouldBeOutput: false},
		{name: "sizes is preserved", attributeName: "sizes", attributeValue: "50vw", shouldBeOutput: true},
		{name: "widths is filtered", attributeName: "widths", attributeValue: "320,640", shouldBeOutput: false},
		{name: "formats is filtered", attributeName: "formats", attributeValue: "webp,avif", shouldBeOutput: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rctx := NewTestRenderContextBuilder().Build()
			ro := NewTestOrchestratorBuilder().Build()

			node := &ast_domain.TemplateNode{
				TagName: "piko:img",
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "src", Value: "github.com/example/image.png"},
					{Name: tc.attributeName, Value: tc.attributeValue},
				},
			}

			var buffer bytes.Buffer
			qw := qt.AcquireWriter(&buffer)
			defer qt.ReleaseWriter(qw)

			err := renderPikoImg(ro, node, qw, rctx)
			require.NoError(t, err)

			output := buffer.String()

			if tc.shouldBeOutput {
				assert.Contains(t, output, "sizes=", "sizes attribute should be in output")
				assert.Contains(t, output, tc.attributeValue, "sizes value should be in output")
			} else {
				assert.NotContains(t, output, tc.attributeName+"=", "attribute should be filtered from output")
			}
		})
	}
}

func TestRenderPikoImg_WithEventDirectives(t *testing.T) {
	mockCSRF := newTestCSRFMockWithTokens("eph", []byte("action"))

	rctx := NewTestRenderContextBuilder().
		WithCSRFService(mockCSRF).
		WithHTTPRequest(testHTTPRequest()).
		Build()
	ro := NewTestOrchestratorBuilder().
		WithCSRFService(mockCSRF).
		Build()

	node := &ast_domain.TemplateNode{
		TagName: "piko:img",
		Attributes: []ast_domain.HTMLAttribute{
			{Name: "src", Value: "github.com/example/image.png"},
		},
		OnEvents: map[string][]ast_domain.Directive{
			"click": {{RawExpression: "handleClick"}},
		},
	}

	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	err := renderPikoImg(ro, node, qw, rctx)
	require.NoError(t, err)

	output := buffer.String()
	assert.Contains(t, output, `p-on:click="handleClick"`)
}

func TestExtractPikoImgAttrs(t *testing.T) {
	testCases := []struct {
		name       string
		wantSrc    string
		wantSizes  string
		wantWidths string
		attrs      []ast_domain.HTMLAttribute
		hasProfile bool
	}{
		{
			name: "extracts src only",
			attrs: []ast_domain.HTMLAttribute{
				{Name: "src", Value: "image.png"},
				{Name: "alt", Value: "test"},
			},
			wantSrc:    "image.png",
			hasProfile: false,
		},
		{
			name: "extracts all profile attributes",
			attrs: []ast_domain.HTMLAttribute{
				{Name: "src", Value: "image.png"},
				{Name: "sizes", Value: "100vw"},
				{Name: "densities", Value: "1x,2x"},
				{Name: "formats", Value: "webp,avif"},
				{Name: "widths", Value: "320,640"},
			},
			wantSrc:    "image.png",
			wantSizes:  "100vw",
			wantWidths: "320,640",
			hasProfile: true,
		},
		{

			name: "lowercase attribute names from parser",
			attrs: []ast_domain.HTMLAttribute{
				{Name: "src", Value: "image.png"},
				{Name: "sizes", Value: "50vw"},
			},
			wantSrc:    "image.png",
			wantSizes:  "50vw",
			hasProfile: true,
		},
		{
			name:       "empty attributes",
			attrs:      []ast_domain.HTMLAttribute{},
			wantSrc:    "",
			hasProfile: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			node := &ast_domain.TemplateNode{
				Attributes: tc.attrs,
			}
			result := extractPikoImgAttrs(node)
			assert.Equal(t, tc.wantSrc, result.src)
			assert.Equal(t, tc.wantSizes, result.sizes)
			assert.Equal(t, tc.wantWidths, result.widths)
			assert.Equal(t, tc.hasProfile, result.hasProfile())
		})
	}
}

func TestExtractPikoImgAttrs_DynamicSrc(t *testing.T) {
	dw := ast_domain.GetDirectWriter()
	dw.SetName("src")
	dw.AppendString("dynamic/image.png")

	node := &ast_domain.TemplateNode{
		Attributes:       []ast_domain.HTMLAttribute{},
		AttributeWriters: []*ast_domain.DirectWriter{dw},
	}

	result := extractPikoImgAttrs(node)
	assert.Equal(t, "dynamic/image.png", result.src)
}

func TestPikoImgAttrs_ToAssetProfile(t *testing.T) {
	testCases := []struct {
		name       string
		attrs      pikoImgAttrs
		wantWidths []int
		wantNil    bool
	}{
		{
			name:    "no profile attributes returns nil",
			attrs:   pikoImgAttrs{src: "image.png"},
			wantNil: true,
		},
		{
			name: "with sizes creates profile",
			attrs: pikoImgAttrs{
				src:   "image.png",
				sizes: "100vw",
			},
			wantNil: false,
		},
		{
			name: "with widths parses correctly",
			attrs: pikoImgAttrs{
				src:    "image.png",
				widths: "320, 640, 1280",
			},
			wantNil:    false,
			wantWidths: []int{320, 640, 1280},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.attrs.toAssetProfile()
			if tc.wantNil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				if tc.wantWidths != nil {
					assert.Equal(t, tc.wantWidths, result.Widths)
				}
			}
		})
	}
}

func TestSortProfileKeys(t *testing.T) {
	testCases := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "empty slice",
			input:    []string{},
			expected: []string{},
		},
		{
			name:     "single element",
			input:    []string{"a"},
			expected: []string{"a"},
		},
		{
			name:     "two elements already sorted",
			input:    []string{"a", "b"},
			expected: []string{"a", "b"},
		},
		{
			name:     "two elements reverse order",
			input:    []string{"b", "a"},
			expected: []string{"a", "b"},
		},
		{
			name:     "three elements",
			input:    []string{"c", "a", "b"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "four elements (insertion sort)",
			input:    []string{"d", "b", "c", "a"},
			expected: []string{"a", "b", "c", "d"},
		},
		{
			name:     "five elements",
			input:    []string{"e", "c", "a", "d", "b"},
			expected: []string{"a", "b", "c", "d", "e"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			input := make([]string, len(tc.input))
			copy(input, tc.input)
			sortProfileKeys(&input, len(input))
			assert.Equal(t, tc.expected, input)
		})
	}
}

func TestWriteSrcsetAttribute_CacheHit(t *testing.T) {
	rctx := NewTestRenderContextBuilder().Build()

	profiles := []registry_dto.NamedProfile{
		{Name: "w320", Profile: registry_dto.DesiredProfile{ResultingTags: registry_dto.TagsFromMap(map[string]string{"width": "320"})}},
	}
	artefact := &registry_dto.ArtefactMeta{
		ID:              "test-artefact",
		DesiredProfiles: profiles,
	}

	actualHash := hashDesiredProfiles(profiles)
	cacheKey := srcsetCacheKey{
		artefactID:  "test-artefact",
		profileHash: actualHash,
	}
	rctx.srcsetCache[cacheKey] = "pre-cached-value.png 320w"

	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	writeSrcsetAttribute(qw, artefact, "/_piko/assets/image.png", rctx)

	output := buffer.String()
	assert.Contains(t, output, "srcset=")
	assert.Contains(t, output, "pre-cached-value.png 320w", "should use cached value")
}

func TestWriteSrcsetAttribute_CacheMiss(t *testing.T) {
	rctx := NewTestRenderContextBuilder().Build()

	profiles := []registry_dto.NamedProfile{
		{Name: "w640", Profile: registry_dto.DesiredProfile{ResultingTags: registry_dto.TagsFromMap(map[string]string{"width": "640"})}},
	}
	artefact := &registry_dto.ArtefactMeta{
		ID:              "new-artefact",
		DesiredProfiles: profiles,
	}

	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	writeSrcsetAttribute(qw, artefact, "/_piko/assets/image.png", rctx)

	output := buffer.String()
	assert.Contains(t, output, "srcset=")
	assert.Contains(t, output, "640w")

	actualHash := hashDesiredProfiles(profiles)
	cacheKey := srcsetCacheKey{
		artefactID:  "new-artefact",
		profileHash: actualHash,
	}
	assert.NotEmpty(t, rctx.srcsetCache[cacheKey], "srcset should be cached after miss")
}

func TestWriteSrcsetAttribute_EmptyProfiles(t *testing.T) {
	rctx := NewTestRenderContextBuilder().Build()

	artefact := &registry_dto.ArtefactMeta{
		ID:              "empty-artefact",
		DesiredProfiles: []registry_dto.NamedProfile{},
	}

	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	writeSrcsetAttribute(qw, artefact, "/_piko/assets/image.png", rctx)

	output := buffer.String()

	assert.Empty(t, output, "empty profiles should not produce srcset attribute")
}

func TestHashDesiredProfiles(t *testing.T) {
	testCases := []struct {
		name     string
		profiles []registry_dto.NamedProfile
		wantZero bool
	}{
		{
			name:     "empty profiles returns zero",
			profiles: []registry_dto.NamedProfile{},
			wantZero: true,
		},
		{
			name:     "nil profiles returns zero",
			profiles: nil,
			wantZero: true,
		},
		{
			name: "profiles with width",
			profiles: []registry_dto.NamedProfile{
				{Name: "w320", Profile: registry_dto.DesiredProfile{ResultingTags: registry_dto.TagsFromMap(map[string]string{"width": "320"})}},
			},
			wantZero: false,
		},
		{
			name: "profiles with density",
			profiles: []registry_dto.NamedProfile{
				{Name: "1x", Profile: registry_dto.DesiredProfile{ResultingTags: registry_dto.TagsFromMap(map[string]string{"density": "1x"})}},
			},
			wantZero: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := hashDesiredProfiles(tc.profiles)
			if tc.wantZero {
				assert.Equal(t, uint64(0), result)
			} else {
				assert.NotEqual(t, uint64(0), result)
			}
		})
	}
}

func TestHashDesiredProfiles_Deterministic(t *testing.T) {
	profiles := []registry_dto.NamedProfile{
		{Name: "w320", Profile: registry_dto.DesiredProfile{ResultingTags: registry_dto.TagsFromMap(map[string]string{"width": "320"})}},
		{Name: "w640", Profile: registry_dto.DesiredProfile{ResultingTags: registry_dto.TagsFromMap(map[string]string{"width": "640"})}},
	}

	hash1 := hashDesiredProfiles(profiles)
	hash2 := hashDesiredProfiles(profiles)
	assert.Equal(t, hash1, hash2)
}

func TestAppendSrcset(t *testing.T) {
	testCases := []struct {
		name     string
		profiles []registry_dto.NamedProfile
		baseURL  string
		contains []string
	}{
		{
			name:     "empty profiles",
			profiles: []registry_dto.NamedProfile{},
			baseURL:  "/_piko/assets/image.png",
			contains: []string{},
		},
		{
			name: "width-based profiles",
			profiles: []registry_dto.NamedProfile{
				{Name: "w320", Profile: registry_dto.DesiredProfile{ResultingTags: registry_dto.TagsFromMap(map[string]string{"width": "320"})}},
				{Name: "w640", Profile: registry_dto.DesiredProfile{ResultingTags: registry_dto.TagsFromMap(map[string]string{"width": "640"})}},
			},
			baseURL:  "/_piko/assets/image.png",
			contains: []string{"320w", "640w", "?v=w320", "?v=w640"},
		},
		{
			name: "density-based profiles",
			profiles: []registry_dto.NamedProfile{
				{Name: "1x", Profile: registry_dto.DesiredProfile{ResultingTags: registry_dto.TagsFromMap(map[string]string{"density": "1x"})}},
				{Name: "2x", Profile: registry_dto.DesiredProfile{ResultingTags: registry_dto.TagsFromMap(map[string]string{"density": "2x"})}},
			},
			baseURL:  "/_piko/assets/image.png",
			contains: []string{"1x", "2x", "?v=1x", "?v=2x"},
		},
		{
			name: "skips profiles without width or density",
			profiles: []registry_dto.NamedProfile{
				{Name: "empty", Profile: registry_dto.DesiredProfile{ResultingTags: registry_dto.TagsFromMap(map[string]string{"format": "webp"})}},
			},
			baseURL:  "/_piko/assets/image.png",
			contains: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var buffer []byte
			buffer = appendSrcset(buffer, tc.profiles, tc.baseURL)
			result := string(buffer)

			for _, expected := range tc.contains {
				assert.Contains(t, result, expected)
			}
		})
	}
}

func TestParseIntList(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected []int
	}{
		{
			name:     "empty string",
			input:    "",
			expected: []int{},
		},
		{
			name:     "single value",
			input:    "320",
			expected: []int{320},
		},
		{
			name:     "multiple values",
			input:    "320, 640, 1280",
			expected: []int{320, 640, 1280},
		},
		{
			name:     "values with extra whitespace",
			input:    "  320  ,  640  ,  1280  ",
			expected: []int{320, 640, 1280},
		},
		{
			name:     "invalid values are skipped",
			input:    "320, invalid, 640",
			expected: []int{320, 640},
		},
		{
			name:     "all invalid returns empty",
			input:    "invalid, also-invalid",
			expected: []int{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parseIntList(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestParseCommaSeparated(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "single value",
			input:    "webp",
			expected: []string{"webp"},
		},
		{
			name:     "multiple values",
			input:    "webp, avif, png",
			expected: []string{"webp", "avif", "png"},
		},
		{
			name:     "trims whitespace",
			input:    "  webp  ,  avif  ",
			expected: []string{"webp", "avif"},
		},
		{
			name:     "skips empty parts",
			input:    "webp,,avif",
			expected: []string{"webp", "avif"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parseCommaSeparated(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestAppendTransformedSrc(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "transforms module path",
			input:    "github.com/example/image.png",
			expected: "/_piko/assets/github.com/example/image.png",
		},
		{
			name:     "skips https URLs",
			input:    "https://cdn.example.com/image.png",
			expected: "https://cdn.example.com/image.png",
		},
		{
			name:     "cleans dirty paths",
			input:    "github.com/example/../image.png",
			expected: "/_piko/assets/github.com/image.png",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var buffer []byte
			buffer = assetpath.AppendTransformed(buffer, tc.input, assetpath.DefaultServePath)
			assert.Equal(t, tc.expected, string(buffer))
		})
	}
}

func TestIsPikoImgSpecialAttr(t *testing.T) {

	specialAttrs := []string{"profile", "densities", "sizes", "formats", "widths", "variant", "cms-media"}
	normalAttrs := []string{"src", "alt", "class", "id", "width", "height"}

	for _, attr := range specialAttrs {
		t.Run("special_"+attr, func(t *testing.T) {
			assert.True(t, isPikoImgSpecialAttr(attr))
		})
	}

	for _, attr := range normalAttrs {
		t.Run("normal_"+attr, func(t *testing.T) {
			assert.False(t, isPikoImgSpecialAttr(attr))
		})
	}
}

func TestSortedProfileKeysPool(t *testing.T) {
	profiles := []registry_dto.NamedProfile{
		{Name: "a", Profile: registry_dto.DesiredProfile{}},
		{Name: "b", Profile: registry_dto.DesiredProfile{}},
		{Name: "c", Profile: registry_dto.DesiredProfile{}},
	}

	keys := getSortedProfileKeys(profiles)
	assert.Equal(t, 3, len(*keys))

	putSortedProfileKeys(keys)

	keys2 := getSortedProfileKeys(profiles)
	assert.Equal(t, 3, len(*keys2))
	putSortedProfileKeys(keys2)
}

func TestRenderPikoImg_IncrementsMetrics(t *testing.T) {
	rctx := NewTestRenderContextBuilder().Build()
	ro := NewTestOrchestratorBuilder().Build()

	node := &ast_domain.TemplateNode{
		TagName: "piko:img",
		Attributes: []ast_domain.HTMLAttribute{
			{Name: "src", Value: "github.com/example/image.png"},
		},
	}

	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	err := renderPikoImg(ro, node, qw, rctx)
	require.NoError(t, err)
	assert.Contains(t, buffer.String(), "<img")
}

func TestWriteIntAttr(t *testing.T) {
	testCases := []struct {
		name          string
		attributeName string
		expected      string
		value         int
	}{
		{name: "positive value writes attribute", attributeName: "width", value: 42, expected: " width=\"42\""},
		{name: "zero writes nothing", attributeName: "width", value: 0, expected: ""},
		{name: "negative writes nothing", attributeName: "height", value: -1, expected: ""},
		{name: "large positive value", attributeName: "height", value: 1920, expected: " height=\"1920\""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var buffer bytes.Buffer
			qw := qt.AcquireWriter(&buffer)
			defer qt.ReleaseWriter(qw)

			writeIntAttr(qw, tc.attributeName, tc.value)

			assert.Equal(t, tc.expected, buffer.String())
		})
	}
}

func TestCheckDimensionAttributes(t *testing.T) {
	testCases := []struct {
		name             string
		attributes       []ast_domain.HTMLAttribute
		attributeWriters []*ast_domain.DirectWriter
		expectWidth      bool
		expectHeight     bool
	}{
		{
			name:         "no dimensions present",
			attributes:   nil,
			expectWidth:  false,
			expectHeight: false,
		},
		{
			name: "width in static attributes",
			attributes: []ast_domain.HTMLAttribute{
				{Name: "width", Value: "100"},
			},
			expectWidth:  true,
			expectHeight: false,
		},
		{
			name: "height in static attributes",
			attributes: []ast_domain.HTMLAttribute{
				{Name: "height", Value: "200"},
			},
			expectWidth:  false,
			expectHeight: true,
		},
		{
			name: "both in static attributes",
			attributes: []ast_domain.HTMLAttribute{
				{Name: "width", Value: "100"},
				{Name: "height", Value: "200"},
			},
			expectWidth:  true,
			expectHeight: true,
		},
		{
			name: "width in DirectWriter",
			attributeWriters: []*ast_domain.DirectWriter{
				{Name: "width"},
			},
			expectWidth:  true,
			expectHeight: false,
		},
		{
			name: "height in DirectWriter",
			attributeWriters: []*ast_domain.DirectWriter{
				{Name: "height"},
			},
			expectWidth:  false,
			expectHeight: true,
		},
		{
			name: "nil DirectWriter is skipped",
			attributeWriters: []*ast_domain.DirectWriter{
				nil,
			},
			expectWidth:  false,
			expectHeight: false,
		},
		{
			name: "mixed static and DirectWriter",
			attributes: []ast_domain.HTMLAttribute{
				{Name: "width", Value: "100"},
			},
			attributeWriters: []*ast_domain.DirectWriter{
				{Name: "height"},
			},
			expectWidth:  true,
			expectHeight: true,
		},
		{
			name: "unrelated attributes do not match",
			attributes: []ast_domain.HTMLAttribute{
				{Name: "class", Value: "hero"},
				{Name: "alt", Value: "logo"},
			},
			expectWidth:  false,
			expectHeight: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			node := &ast_domain.TemplateNode{
				Attributes:       tc.attributes,
				AttributeWriters: tc.attributeWriters,
			}

			hasWidth, hasHeight := checkDimensionAttributes(node)

			assert.Equal(t, tc.expectWidth, hasWidth, "hasWidth")
			assert.Equal(t, tc.expectHeight, hasHeight, "hasHeight")
		})
	}
}

func TestGetImageMimeType(t *testing.T) {
	testCases := []struct {
		name     string
		format   string
		expected string
	}{
		{name: "webp format", format: "webp", expected: "image/webp"},
		{name: "avif format", format: "avif", expected: "image/avif"},
		{name: "png format", format: "png", expected: "image/png"},
		{name: "jpeg format", format: "jpeg", expected: "image/jpeg"},
		{name: "jpg format", format: "jpg", expected: "image/jpeg"},
		{name: "gif format", format: "gif", expected: "image/gif"},
		{name: "unknown format falls back to image prefix", format: "tiff", expected: "image/tiff"},
		{name: "empty format uses image prefix", format: "", expected: "image/"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := getImageMimeType(tc.format)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func newTestCMSMediaWrapper(media *mockCMSMedia) *cmsMediaWrapper {
	return tryCMSMediaWrapper(media)
}

func newTestVariantWrapperMap(variants map[string]*mockVariant) map[string]*variantWrapper {
	media := &mockCMSMedia{
		url:      "https://example.com/image.png",
		variants: variants,
	}
	wrapper := tryCMSMediaWrapper(media)
	if wrapper == nil {
		return nil
	}
	return wrapper.MediaVariants()
}

func TestRenderCMSMediaImg(t *testing.T) {
	testCases := []struct {
		name          string
		media         *mockCMSMedia
		attrs         *pikoImgAttrs
		nodeAttrs     []ast_domain.HTMLAttribute
		shouldContain []string
		notInOutput   []string
	}{
		{
			name: "basic CMS media renders img with src",
			media: &mockCMSMedia{
				url:     "https://cdn.example.com/photo.jpg",
				width:   1920,
				height:  1080,
				altText: "A photo",
			},
			attrs: &pikoImgAttrs{
				src: "https://cdn.example.com/photo.jpg",
			},
			shouldContain: []string{
				`<img`,
				`src="https://cdn.example.com/photo.jpg"`,
				`alt="A photo"`,
				`width="1920"`,
				`height="1080"`,
				`/>`,
			},
		},
		{
			name: "CMS media with named variant uses variant URL",
			media: &mockCMSMedia{
				url:     "https://cdn.example.com/photo.jpg",
				width:   1920,
				height:  1080,
				altText: "Variant photo",
				variants: map[string]*mockVariant{
					"thumb_200": {url: "https://cdn.example.com/thumb.jpg", width: 200, ready: true},
				},
			},
			attrs: &pikoImgAttrs{
				src:     "https://cdn.example.com/photo.jpg",
				variant: "thumb_200",
			},
			shouldContain: []string{
				`src="https://cdn.example.com/thumb.jpg"`,
			},
		},
		{
			name: "CMS media with unready variant falls back to media URL",
			media: &mockCMSMedia{
				url: "https://cdn.example.com/photo.jpg",
				variants: map[string]*mockVariant{
					"pending": {url: "https://cdn.example.com/pending.jpg", width: 400, ready: false},
				},
			},
			attrs: &pikoImgAttrs{
				src:     "https://cdn.example.com/photo.jpg",
				variant: "pending",
			},
			shouldContain: []string{
				`src="https://cdn.example.com/photo.jpg"`,
			},
			notInOutput: []string{
				`src="https://cdn.example.com/pending.jpg"`,
			},
		},
		{
			name: "CMS media with missing variant falls back to media URL",
			media: &mockCMSMedia{
				url:      "https://cdn.example.com/photo.jpg",
				variants: map[string]*mockVariant{},
			},
			attrs: &pikoImgAttrs{
				src:     "https://cdn.example.com/photo.jpg",
				variant: "nonexistent",
			},
			shouldContain: []string{
				`src="https://cdn.example.com/photo.jpg"`,
			},
		},
		{
			name: "CMS media with widths generates srcset",
			media: &mockCMSMedia{
				url:     "https://cdn.example.com/photo.jpg",
				width:   1920,
				height:  1080,
				altText: "Responsive photo",
				variants: map[string]*mockVariant{
					"w320": {url: "https://cdn.example.com/320.jpg", width: 320, ready: true},
					"w640": {url: "https://cdn.example.com/640.jpg", width: 640, ready: true},
				},
			},
			attrs: &pikoImgAttrs{
				src:    "https://cdn.example.com/photo.jpg",
				widths: "320,640",
			},
			shouldContain: []string{
				`srcset="`,
				`320w`,
				`640w`,
			},
		},
		{
			name: "CMS media with widths and sizes writes sizes attribute",
			media: &mockCMSMedia{
				url:     "https://cdn.example.com/photo.jpg",
				width:   1920,
				height:  1080,
				altText: "Sized photo",
				variants: map[string]*mockVariant{
					"w320": {url: "https://cdn.example.com/320.jpg", width: 320, ready: true},
				},
			},
			attrs: &pikoImgAttrs{
				src:    "https://cdn.example.com/photo.jpg",
				widths: "320",
				sizes:  "(max-width: 600px) 100vw, 50vw",
			},
			shouldContain: []string{
				`sizes="(max-width: 600px) 100vw, 50vw"`,
			},
		},
		{
			name:  "CMS media nil source emits warning and returns",
			media: nil,
			attrs: &pikoImgAttrs{
				src:      "some-src",
				cmsMedia: true,
			},
			shouldContain: []string{},
			notInOutput:   []string{`<img`},
		},
		{
			name: "user alt attribute overrides media alt",
			media: &mockCMSMedia{
				url:     "https://cdn.example.com/photo.jpg",
				altText: "Media alt text",
			},
			attrs: &pikoImgAttrs{
				src: "https://cdn.example.com/photo.jpg",
			},
			nodeAttrs: []ast_domain.HTMLAttribute{
				{Name: "alt", Value: "User alt text"},
			},
			shouldContain: []string{
				`alt="User alt text"`,
			},
			notInOutput: []string{
				`alt="Media alt text"`,
			},
		},
		{
			name: "CMS media with zero dimensions omits width and height",
			media: &mockCMSMedia{
				url:    "https://cdn.example.com/photo.jpg",
				width:  0,
				height: 0,
			},
			attrs: &pikoImgAttrs{
				src: "https://cdn.example.com/photo.jpg",
			},
			shouldContain: []string{
				`<img`,
			},
			notInOutput: []string{
				`width=`,
				`height=`,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rctx := NewTestRenderContextBuilder().Build()
			ro := NewTestOrchestratorBuilder().Build()

			node := &ast_domain.TemplateNode{
				TagName:    "piko:img",
				Attributes: tc.nodeAttrs,
			}

			attrs := tc.attrs
			if tc.media != nil {
				attrs.cmsMediaSource = newTestCMSMediaWrapper(tc.media)
			}

			var buffer bytes.Buffer
			qw := qt.AcquireWriter(&buffer)
			defer qt.ReleaseWriter(qw)

			err := renderCMSMediaImg(ro, node, qw, rctx, attrs)
			require.NoError(t, err)

			output := buffer.String()
			for _, expected := range tc.shouldContain {
				assert.Contains(t, output, expected)
			}
			for _, notExpected := range tc.notInOutput {
				assert.NotContains(t, output, notExpected)
			}
		})
	}
}

func TestWriteCMSMediaSrcset(t *testing.T) {
	testCases := []struct {
		name          string
		variants      map[string]*mockVariant
		widthsAttr    string
		shouldContain []string
		wantEmpty     bool
	}{
		{
			name:       "empty widths produces no output",
			variants:   map[string]*mockVariant{},
			widthsAttr: "",
			wantEmpty:  true,
		},
		{
			name:       "no matching variants produces no output",
			variants:   map[string]*mockVariant{},
			widthsAttr: "320,640",
			wantEmpty:  true,
		},
		{
			name: "matching variants with w prefix",
			variants: map[string]*mockVariant{
				"w320": {url: "https://cdn.example.com/320.jpg", width: 320, ready: true},
				"w640": {url: "https://cdn.example.com/640.jpg", width: 640, ready: true},
			},
			widthsAttr: "320,640",
			shouldContain: []string{
				`srcset="`,
				"https://cdn.example.com/320.jpg 320w",
				"https://cdn.example.com/640.jpg 640w",
			},
		},
		{
			name: "matching variants without w prefix",
			variants: map[string]*mockVariant{
				"320": {url: "https://cdn.example.com/320.jpg", width: 320, ready: true},
			},
			widthsAttr: "320",
			shouldContain: []string{
				`srcset="`,
				"https://cdn.example.com/320.jpg 320w",
			},
		},
		{
			name: "unready variants are skipped",
			variants: map[string]*mockVariant{
				"w320": {url: "https://cdn.example.com/320.jpg", width: 320, ready: true},
				"w640": {url: "https://cdn.example.com/640.jpg", width: 640, ready: false},
			},
			widthsAttr: "320,640",
			shouldContain: []string{
				"https://cdn.example.com/320.jpg 320w",
			},
		},
		{
			name: "partially matching widths only include found variants",
			variants: map[string]*mockVariant{
				"w320": {url: "https://cdn.example.com/320.jpg", width: 320, ready: true},
			},
			widthsAttr: "320,640,960",
			shouldContain: []string{
				"https://cdn.example.com/320.jpg 320w",
			},
		},
		{
			name: "invalid width values in attribute are ignored",
			variants: map[string]*mockVariant{
				"w320": {url: "https://cdn.example.com/320.jpg", width: 320, ready: true},
			},
			widthsAttr: "abc,320,xyz",
			shouldContain: []string{
				"https://cdn.example.com/320.jpg 320w",
			},
		},
		{
			name: "multiple entries separated by comma-space",
			variants: map[string]*mockVariant{
				"w320": {url: "https://cdn.example.com/320.jpg", width: 320, ready: true},
				"w640": {url: "https://cdn.example.com/640.jpg", width: 640, ready: true},
			},
			widthsAttr: "320,640",
			shouldContain: []string{
				", ",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rctx := NewTestRenderContextBuilder().Build()
			variantWrappers := newTestVariantWrapperMap(tc.variants)

			var buffer bytes.Buffer
			qw := qt.AcquireWriter(&buffer)
			defer qt.ReleaseWriter(qw)

			writeCMSMediaSrcset(qw, variantWrappers, tc.widthsAttr, rctx)

			output := buffer.String()
			if tc.wantEmpty {
				assert.Empty(t, output)
			} else {
				for _, expected := range tc.shouldContain {
					assert.Contains(t, output, expected)
				}
			}
		})
	}
}

func TestWriteAltFromMedia(t *testing.T) {
	testCases := []struct {
		name             string
		media            *mockCMSMedia
		nodeAttrs        []ast_domain.HTMLAttribute
		attributeWriters []*ast_domain.DirectWriter
		shouldContain    []string
		notInOutput      []string
		wantEmpty        bool
	}{
		{
			name: "media with alt text writes alt attribute",
			media: &mockCMSMedia{
				url:     "https://cdn.example.com/photo.jpg",
				altText: "A lovely sunset",
			},
			shouldContain: []string{
				`alt="A lovely sunset"`,
			},
		},
		{
			name: "media without alt text writes nothing",
			media: &mockCMSMedia{
				url:     "https://cdn.example.com/photo.jpg",
				altText: "",
			},
			wantEmpty: true,
		},
		{
			name: "static alt attribute on node prevents media alt",
			media: &mockCMSMedia{
				url:     "https://cdn.example.com/photo.jpg",
				altText: "Media alt",
			},
			nodeAttrs: []ast_domain.HTMLAttribute{
				{Name: "alt", Value: "User alt"},
			},
			wantEmpty: true,
		},
		{
			name: "dynamic alt attribute writer on node prevents media alt",
			media: &mockCMSMedia{
				url:     "https://cdn.example.com/photo.jpg",
				altText: "Media alt",
			},
			attributeWriters: func() []*ast_domain.DirectWriter {
				dw := ast_domain.GetDirectWriter()
				dw.SetName("alt")
				dw.AppendString("Dynamic user alt")
				return []*ast_domain.DirectWriter{dw}
			}(),
			wantEmpty: true,
		},
		{
			name: "nil DirectWriter in attribute writers is skipped",
			media: &mockCMSMedia{
				url:     "https://cdn.example.com/photo.jpg",
				altText: "Should appear",
			},
			attributeWriters: []*ast_domain.DirectWriter{nil},
			shouldContain: []string{
				`alt="Should appear"`,
			},
		},
		{
			name: "alt text with HTML special characters is escaped",
			media: &mockCMSMedia{
				url:     "https://cdn.example.com/photo.jpg",
				altText: "A photo with <special> & \"chars\"",
			},
			shouldContain: []string{
				`alt="A photo with &lt;special&gt; &amp; &quot;chars&quot;"`,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			node := &ast_domain.TemplateNode{
				Attributes:       tc.nodeAttrs,
				AttributeWriters: tc.attributeWriters,
			}
			wrapper := newTestCMSMediaWrapper(tc.media)
			require.NotNil(t, wrapper, "wrapper must not be nil for test")

			var buffer bytes.Buffer
			qw := qt.AcquireWriter(&buffer)
			defer qt.ReleaseWriter(qw)

			writeAltFromMedia(node, qw, wrapper)

			output := buffer.String()
			if tc.wantEmpty {
				assert.Empty(t, output)
			} else {
				for _, expected := range tc.shouldContain {
					assert.Contains(t, output, expected)
				}
			}
			for _, notExpected := range tc.notInOutput {
				assert.NotContains(t, output, notExpected)
			}
		})
	}
}

func TestWriteMediaDimensions(t *testing.T) {
	testCases := []struct {
		name             string
		media            *mockCMSMedia
		nodeAttrs        []ast_domain.HTMLAttribute
		attributeWriters []*ast_domain.DirectWriter
		shouldContain    []string
		notInOutput      []string
	}{
		{
			name: "writes width and height from media",
			media: &mockCMSMedia{
				url:    "https://cdn.example.com/photo.jpg",
				width:  1920,
				height: 1080,
			},
			shouldContain: []string{
				`width="1920"`,
				`height="1080"`,
			},
		},
		{
			name: "zero width omits width attribute",
			media: &mockCMSMedia{
				url:    "https://cdn.example.com/photo.jpg",
				width:  0,
				height: 600,
			},
			shouldContain: []string{
				`height="600"`,
			},
			notInOutput: []string{
				`width=`,
			},
		},
		{
			name: "zero height omits height attribute",
			media: &mockCMSMedia{
				url:    "https://cdn.example.com/photo.jpg",
				width:  800,
				height: 0,
			},
			shouldContain: []string{
				`width="800"`,
			},
			notInOutput: []string{
				`height=`,
			},
		},
		{
			name: "both zero omits both",
			media: &mockCMSMedia{
				url:    "https://cdn.example.com/photo.jpg",
				width:  0,
				height: 0,
			},
			notInOutput: []string{
				`width=`,
				`height=`,
			},
		},
		{
			name: "existing static width attribute prevents media width",
			media: &mockCMSMedia{
				url:    "https://cdn.example.com/photo.jpg",
				width:  1920,
				height: 1080,
			},
			nodeAttrs: []ast_domain.HTMLAttribute{
				{Name: "width", Value: "400"},
			},
			shouldContain: []string{
				`height="1080"`,
			},
			notInOutput: []string{
				`width="1920"`,
			},
		},
		{
			name: "existing static height attribute prevents media height",
			media: &mockCMSMedia{
				url:    "https://cdn.example.com/photo.jpg",
				width:  1920,
				height: 1080,
			},
			nodeAttrs: []ast_domain.HTMLAttribute{
				{Name: "height", Value: "300"},
			},
			shouldContain: []string{
				`width="1920"`,
			},
			notInOutput: []string{
				`height="1080"`,
			},
		},
		{
			name: "existing dynamic width writer prevents media width",
			media: &mockCMSMedia{
				url:    "https://cdn.example.com/photo.jpg",
				width:  1920,
				height: 1080,
			},
			attributeWriters: []*ast_domain.DirectWriter{
				{Name: "width"},
			},
			shouldContain: []string{
				`height="1080"`,
			},
			notInOutput: []string{
				`width="1920"`,
			},
		},
		{
			name: "both dimensions already set writes nothing",
			media: &mockCMSMedia{
				url:    "https://cdn.example.com/photo.jpg",
				width:  1920,
				height: 1080,
			},
			nodeAttrs: []ast_domain.HTMLAttribute{
				{Name: "width", Value: "400"},
				{Name: "height", Value: "300"},
			},
			notInOutput: []string{
				`width="1920"`,
				`height="1080"`,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			node := &ast_domain.TemplateNode{
				Attributes:       tc.nodeAttrs,
				AttributeWriters: tc.attributeWriters,
			}
			wrapper := newTestCMSMediaWrapper(tc.media)
			require.NotNil(t, wrapper, "wrapper must not be nil for test")

			var buffer bytes.Buffer
			qw := qt.AcquireWriter(&buffer)
			defer qt.ReleaseWriter(qw)

			writeMediaDimensions(node, qw, wrapper)

			output := buffer.String()
			for _, expected := range tc.shouldContain {
				assert.Contains(t, output, expected)
			}
			for _, notExpected := range tc.notInOutput {
				assert.NotContains(t, output, notExpected)
			}
		})
	}
}

func TestExtractSrcFromWriter_CMSMedia(t *testing.T) {
	testCases := []struct {
		name        string
		setupWriter func() *ast_domain.DirectWriter
		wantSrc     string
		wantWrapper bool
	}{
		{
			name: "single CMS media part returns media URL and wrapper",
			setupWriter: func() *ast_domain.DirectWriter {
				media := &mockCMSMedia{
					url:    "https://cdn.example.com/photo.jpg",
					width:  800,
					height: 600,
				}
				dw := ast_domain.GetDirectWriter()
				dw.SetName("src")
				dw.AppendAny(media)
				return dw
			},
			wantSrc:     "https://cdn.example.com/photo.jpg",
			wantWrapper: true,
		},
		{
			name: "single non-CMS any part returns string without wrapper",
			setupWriter: func() *ast_domain.DirectWriter {
				dw := ast_domain.GetDirectWriter()
				dw.SetName("src")
				dw.AppendAny("plain-string-value")
				return dw
			},
			wantSrc:     "plain-string-value",
			wantWrapper: false,
		},
		{
			name: "single string part returns value without wrapper",
			setupWriter: func() *ast_domain.DirectWriter {
				dw := ast_domain.GetDirectWriter()
				dw.SetName("src")
				dw.AppendString("static/image.png")
				return dw
			},
			wantSrc:     "static/image.png",
			wantWrapper: false,
		},
		{
			name: "multi-part writer returns joined string without wrapper",
			setupWriter: func() *ast_domain.DirectWriter {
				dw := ast_domain.GetDirectWriter()
				dw.SetName("src")
				dw.AppendString("path/")
				dw.AppendString("image.png")
				return dw
			},
			wantSrc:     "path/image.png",
			wantWrapper: false,
		},
		{
			name: "single nil AnyValue part returns empty string without wrapper",
			setupWriter: func() *ast_domain.DirectWriter {
				dw := ast_domain.GetDirectWriter()
				dw.SetName("src")
				dw.AppendAny(nil)
				return dw
			},
			wantSrc:     "",
			wantWrapper: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dw := tc.setupWriter()

			src, wrapper := extractSrcFromWriter(dw)

			assert.Equal(t, tc.wantSrc, src)
			if tc.wantWrapper {
				assert.NotNil(t, wrapper)
			} else {
				assert.Nil(t, wrapper)
			}
		})
	}
}

func TestTryExtractCMSMedia(t *testing.T) {
	testCases := []struct {
		part        *ast_domain.WriterPart
		name        string
		wantWrapper bool
	}{
		{
			name:        "nil part returns nil",
			part:        nil,
			wantWrapper: false,
		},
		{
			name: "non-any part type returns nil",
			part: &ast_domain.WriterPart{
				Type:        ast_domain.WriterPartString,
				StringValue: "some-string",
			},
			wantWrapper: false,
		},
		{
			name: "any part with CMS media returns wrapper",
			part: &ast_domain.WriterPart{
				Type:     ast_domain.WriterPartAny,
				AnyValue: &mockCMSMedia{url: "https://cdn.example.com/photo.jpg"},
			},
			wantWrapper: true,
		},
		{
			name: "any part with non-CMS value returns nil",
			part: &ast_domain.WriterPart{
				Type:     ast_domain.WriterPartAny,
				AnyValue: "just-a-string",
			},
			wantWrapper: false,
		},
		{
			name: "any part with nil value returns nil",
			part: &ast_domain.WriterPart{
				Type:     ast_domain.WriterPartAny,
				AnyValue: nil,
			},
			wantWrapper: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tryExtractCMSMedia(tc.part)

			if tc.wantWrapper {
				assert.NotNil(t, result)
			} else {
				assert.Nil(t, result)
			}
		})
	}
}
