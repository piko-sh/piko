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
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	qt "github.com/valyala/quicktemplate"
	"piko.sh/piko/internal/ast/ast_domain"
)

func TestRenderPikoSvg_LoadsSVGFromRegistry(t *testing.T) {
	testCases := []struct {
		name           string
		svgID          string
		svgInnerHTML   string
		svgViewBox     string
		expectedSymbol bool
	}{
		{
			name:           "loads SVG with simple path",
			svgID:          "icon-home",
			svgInnerHTML:   `<path d="M10 20v-6h4v6h5v-8h3L12 3 2 12h3v8z"/>`,
			svgViewBox:     "0 0 24 24",
			expectedSymbol: true,
		},
		{
			name:           "loads SVG with multiple elements",
			svgID:          "icon-menu",
			svgInnerHTML:   `<rect x="3" y="6" width="18" height="2"/><rect x="3" y="11" width="18" height="2"/><rect x="3" y="16" width="18" height="2"/>`,
			svgViewBox:     "0 0 24 24",
			expectedSymbol: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockReg := newTestRegistryBuilder().
				withSVG(tc.svgID, tc.svgInnerHTML, ast_domain.HTMLAttribute{Name: "viewBox", Value: tc.svgViewBox}).
				build()

			rctx := NewTestRenderContextBuilder().
				WithRegistry(mockReg).
				Build()

			node := &ast_domain.TemplateNode{
				TagName: "piko:svg",
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "src", Value: tc.svgID},
				},
			}

			var buffer bytes.Buffer
			qw := qt.AcquireWriter(&buffer)
			defer qt.ReleaseWriter(qw)

			ro := NewTestOrchestratorBuilder().
				WithRegistry(mockReg).
				Build()

			err := renderPikoSvg(ro, node, qw, rctx)
			require.NoError(t, err)

			output := buffer.String()

			assert.Contains(t, output, "<svg")
			assert.Contains(t, output, "<use")
			assert.Contains(t, output, "#"+tc.svgID)
			assert.Contains(t, output, "</svg>")

			if tc.expectedSymbol {
				found := false
				for i := range rctx.requiredSvgSymbols {
					if rctx.requiredSvgSymbols[i].id == tc.svgID {
						found = true
						break
					}
				}
				assert.True(t, found, "SVG symbol should be collected")
			}
		})
	}
}

func TestRenderPikoSvg_HandlesErrors(t *testing.T) {
	t.Run("missing src attribute writes error comment", func(t *testing.T) {
		rctx := NewTestRenderContextBuilder().Build()

		node := &ast_domain.TemplateNode{
			TagName:    "piko:svg",
			Attributes: []ast_domain.HTMLAttribute{},
		}

		var buffer bytes.Buffer
		qw := qt.AcquireWriter(&buffer)
		defer qt.ReleaseWriter(qw)

		ro := NewTestOrchestratorBuilder().Build()

		err := renderPikoSvg(ro, node, qw, rctx)
		require.NoError(t, err)

		output := buffer.String()
		assert.Contains(t, output, "<!--")
		assert.Contains(t, output, "missing")
	})

	t.Run("SVG not found with error writes error comment", func(t *testing.T) {
		mockReg := newTestRegistryBuilder().
			withSVGError(errors.New("SVG not found")).
			build()

		rctx := NewTestRenderContextBuilder().
			WithRegistry(mockReg).
			Build()

		node := &ast_domain.TemplateNode{
			TagName: "piko:svg",
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "src", Value: "non-existent-svg"},
			},
		}

		var buffer bytes.Buffer
		qw := qt.AcquireWriter(&buffer)
		defer qt.ReleaseWriter(qw)

		ro := NewTestOrchestratorBuilder().
			WithRegistry(mockReg).
			Build()

		err := renderPikoSvg(ro, node, qw, rctx)
		require.NoError(t, err)

		output := buffer.String()
		assert.Contains(t, output, "<!--")
	})

	t.Run("registry error is handled gracefully", func(t *testing.T) {
		mockReg := newTestRegistryBuilder().
			withSVGError(errors.New("registry error")).
			build()

		rctx := NewTestRenderContextBuilder().
			WithRegistry(mockReg).
			Build()

		node := &ast_domain.TemplateNode{
			TagName: "piko:svg",
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "src", Value: "some-svg"},
			},
		}

		var buffer bytes.Buffer
		qw := qt.AcquireWriter(&buffer)
		defer qt.ReleaseWriter(qw)

		ro := NewTestOrchestratorBuilder().
			WithRegistry(mockReg).
			Build()

		err := renderPikoSvg(ro, node, qw, rctx)
		require.NoError(t, err)

		output := buffer.String()
		assert.Contains(t, output, "<!--")
	})
}

func TestRenderPikoSvg_MergesAttributes(t *testing.T) {
	testCases := []struct {
		name           string
		svgAttrs       []ast_domain.HTMLAttribute
		nodeAttrs      []ast_domain.HTMLAttribute
		expectedOutput []string
		notExpected    []string
	}{
		{
			name: "merges class attributes",
			svgAttrs: []ast_domain.HTMLAttribute{
				{Name: "class", Value: "svg-icon"},
				{Name: "viewBox", Value: "0 0 24 24"},
			},
			nodeAttrs: []ast_domain.HTMLAttribute{
				{Name: "src", Value: "test-svg"},
				{Name: "class", Value: "custom-class"},
			},
			expectedOutput: []string{"svg-icon", "custom-class"},
		},
		{
			name: "excludes src attribute from output",
			svgAttrs: []ast_domain.HTMLAttribute{
				{Name: "viewBox", Value: "0 0 24 24"},
			},
			nodeAttrs: []ast_domain.HTMLAttribute{
				{Name: "src", Value: "test-svg"},
				{Name: "class", Value: "custom-class"},
			},
			expectedOutput: []string{"custom-class"},
			notExpected:    []string{`src="`},
		},
		{
			name: "preserves viewBox from SVG",
			svgAttrs: []ast_domain.HTMLAttribute{
				{Name: "viewBox", Value: "0 0 100 100"},
			},
			nodeAttrs: []ast_domain.HTMLAttribute{
				{Name: "src", Value: "test-svg"},
			},
			expectedOutput: []string{`viewBox="0 0 100 100"`},
		},
		{
			name: "user attributes override SVG attributes",
			svgAttrs: []ast_domain.HTMLAttribute{
				{Name: "fill", Value: "black"},
			},
			nodeAttrs: []ast_domain.HTMLAttribute{
				{Name: "src", Value: "test-svg"},
				{Name: "fill", Value: "currentColor"},
			},
			expectedOutput: []string{`fill="currentColor"`},
			notExpected:    []string{`fill="black"`},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			svgData := &ParsedSvgData{
				InnerHTML:  `<path d="M0 0"/>`,
				Attributes: tc.svgAttrs,
			}
			mockReg := &MockRegistryPort{
				GetAssetRawSVGFunc: func(_ context.Context, assetID string) (*ParsedSvgData, error) {
					if assetID == "test-svg" {
						return svgData, nil
					}
					return nil, nil
				},
			}

			rctx := NewTestRenderContextBuilder().
				WithRegistry(mockReg).
				Build()

			node := &ast_domain.TemplateNode{
				TagName:    "piko:svg",
				Attributes: tc.nodeAttrs,
			}

			var buffer bytes.Buffer
			qw := qt.AcquireWriter(&buffer)
			defer qt.ReleaseWriter(qw)

			ro := NewTestOrchestratorBuilder().
				WithRegistry(mockReg).
				Build()

			err := renderPikoSvg(ro, node, qw, rctx)
			require.NoError(t, err)

			output := buffer.String()
			for _, expected := range tc.expectedOutput {
				assert.Contains(t, output, expected, "output should contain %q", expected)
			}
			for _, notExpected := range tc.notExpected {
				assert.NotContains(t, output, notExpected, "output should not contain %q", notExpected)
			}
		})
	}
}

func TestRenderPikoSvg_CachesResults(t *testing.T) {
	mockReg := newTestRegistryBuilder().
		withSVG("cached-svg", `<path d="M0 0"/>`, ast_domain.HTMLAttribute{Name: "viewBox", Value: "0 0 24 24"}).
		build()

	rctx := NewTestRenderContextBuilder().
		WithRegistry(mockReg).
		Build()

	node := &ast_domain.TemplateNode{
		TagName: "piko:svg",
		Attributes: []ast_domain.HTMLAttribute{
			{Name: "src", Value: "cached-svg"},
			{Name: "class", Value: "icon"},
		},
	}

	ro := NewTestOrchestratorBuilder().
		WithRegistry(mockReg).
		Build()

	var buf1 bytes.Buffer
	qw1 := qt.AcquireWriter(&buf1)
	err := renderPikoSvg(ro, node, qw1, rctx)
	qt.ReleaseWriter(qw1)
	require.NoError(t, err)

	cacheSize := len(rctx.mergedAttrsCache)
	assert.Greater(t, cacheSize, 0, "cache should have entries after first render")

	var buf2 bytes.Buffer
	qw2 := qt.AcquireWriter(&buf2)
	err = renderPikoSvg(ro, node, qw2, rctx)
	qt.ReleaseWriter(qw2)
	require.NoError(t, err)

	assert.Equal(t, cacheSize, len(rctx.mergedAttrsCache), "cache size should remain same for cache hit")

	assert.Equal(t, buf1.String(), buf2.String(), "cached output should match original")
}

func TestRenderPikoSvg_CollectsSymbols(t *testing.T) {
	mockReg := newTestRegistryBuilder().
		withSVG("icon-a", `<path d="A"/>`, ast_domain.HTMLAttribute{Name: "viewBox", Value: "0 0 24 24"}).
		withSVG("icon-b", `<path d="B"/>`, ast_domain.HTMLAttribute{Name: "viewBox", Value: "0 0 24 24"}).
		build()

	rctx := NewTestRenderContextBuilder().
		WithRegistry(mockReg).
		Build()

	ro := NewTestOrchestratorBuilder().
		WithRegistry(mockReg).
		Build()

	node1 := &ast_domain.TemplateNode{
		TagName: "piko:svg",
		Attributes: []ast_domain.HTMLAttribute{
			{Name: "src", Value: "icon-a"},
		},
	}

	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	err := renderPikoSvg(ro, node1, qw, rctx)
	qt.ReleaseWriter(qw)
	require.NoError(t, err)

	node2 := &ast_domain.TemplateNode{
		TagName: "piko:svg",
		Attributes: []ast_domain.HTMLAttribute{
			{Name: "src", Value: "icon-b"},
		},
	}

	buffer.Reset()
	qw = qt.AcquireWriter(&buffer)
	err = renderPikoSvg(ro, node2, qw, rctx)
	qt.ReleaseWriter(qw)
	require.NoError(t, err)

	existsA, existsB := false, false
	for i := range rctx.requiredSvgSymbols {
		switch rctx.requiredSvgSymbols[i].id {
		case "icon-a":
			existsA = true
		case "icon-b":
			existsB = true
		}
	}
	assert.True(t, existsA, "icon-a should be collected")
	assert.True(t, existsB, "icon-b should be collected")
	assert.Len(t, rctx.requiredSvgSymbols, 2, "should have exactly 2 symbols")
}

func TestRenderPikoSvg_DeduplicatesSymbols(t *testing.T) {
	mockReg := newTestRegistryBuilder().
		withSVG("icon-dup", `<path d="M0 0"/>`, ast_domain.HTMLAttribute{Name: "viewBox", Value: "0 0 24 24"}).
		build()

	rctx := NewTestRenderContextBuilder().
		WithRegistry(mockReg).
		Build()

	ro := NewTestOrchestratorBuilder().
		WithRegistry(mockReg).
		Build()

	node := &ast_domain.TemplateNode{
		TagName: "piko:svg",
		Attributes: []ast_domain.HTMLAttribute{
			{Name: "src", Value: "icon-dup"},
		},
	}

	for range 5 {
		var buffer bytes.Buffer
		qw := qt.AcquireWriter(&buffer)
		err := renderPikoSvg(ro, node, qw, rctx)
		qt.ReleaseWriter(qw)
		require.NoError(t, err)
	}

	assert.Len(t, rctx.requiredSvgSymbols, 1, "symbol should be deduplicated")
}

func TestHashUserAttrs_EmptyReturnsZero(t *testing.T) {
	result := hashUserAttrs(nil)
	assert.Equal(t, uint64(0), result)

	result = hashUserAttrs([]ast_domain.HTMLAttribute{})
	assert.Equal(t, uint64(0), result)
}

func TestHashUserAttrs_SameAttrsOrderIndependent(t *testing.T) {
	attrs1 := []ast_domain.HTMLAttribute{
		{Name: "class", Value: "foo"},
		{Name: "id", Value: "bar"},
		{Name: "style", Value: "color:red"},
	}

	attrs2 := []ast_domain.HTMLAttribute{
		{Name: "style", Value: "color:red"},
		{Name: "class", Value: "foo"},
		{Name: "id", Value: "bar"},
	}

	hash1 := hashUserAttrs(attrs1)
	hash2 := hashUserAttrs(attrs2)

	assert.Equal(t, hash1, hash2, "hash should be independent of attribute order")
}

func TestHashUserAttrs_DifferentAttrsProduceDifferentHash(t *testing.T) {
	attrs1 := []ast_domain.HTMLAttribute{
		{Name: "class", Value: "foo"},
	}

	attrs2 := []ast_domain.HTMLAttribute{
		{Name: "class", Value: "bar"},
	}

	hash1 := hashUserAttrs(attrs1)
	hash2 := hashUserAttrs(attrs2)

	assert.NotEqual(t, hash1, hash2, "different attributes should produce different hashes")
}

func TestHashUserAttrs_SmallPath(t *testing.T) {

	attrs := []ast_domain.HTMLAttribute{
		{Name: "a", Value: "1"},
		{Name: "b", Value: "2"},
		{Name: "c", Value: "3"},
		{Name: "d", Value: "4"},
	}

	result := hashUserAttrs(attrs)
	assert.NotEqual(t, uint64(0), result)

	result2 := hashUserAttrs(attrs)
	assert.Equal(t, result, result2)
}

func TestHashUserAttrs_LargePath(t *testing.T) {

	attrs := make([]ast_domain.HTMLAttribute, 12)
	for i := range 12 {
		attrs[i] = ast_domain.HTMLAttribute{
			Name:  string(rune('a' + i)),
			Value: string(rune('0' + i)),
		}
	}

	result := hashUserAttrs(attrs)
	assert.NotEqual(t, uint64(0), result)

	attrs2 := make([]ast_domain.HTMLAttribute, 12)
	for i := 11; i >= 0; i-- {
		attrs2[11-i] = attrs[i]
	}

	result2 := hashUserAttrs(attrs2)
	assert.Equal(t, result, result2, "large attribute hash should be order independent")
}

func TestSortAttrsInPlace_SortsCorrectly(t *testing.T) {
	attrs := []ast_domain.HTMLAttribute{
		{Name: "zebra", Value: "z"},
		{Name: "apple", Value: "a"},
		{Name: "middle", Value: "m"},
	}

	sortAttrsInPlace(attrs)

	assert.Equal(t, "apple", attrs[0].Name)
	assert.Equal(t, "middle", attrs[1].Name)
	assert.Equal(t, "zebra", attrs[2].Name)
}

func TestSortAttrsInPlace_HandlesSingleElement(t *testing.T) {
	attrs := []ast_domain.HTMLAttribute{
		{Name: "only", Value: "one"},
	}

	sortAttrsInPlace(attrs)

	assert.Equal(t, "only", attrs[0].Name)
}

func TestSortAttrsInPlace_HandlesEmpty(t *testing.T) {
	attrs := []ast_domain.HTMLAttribute{}
	sortAttrsInPlace(attrs)
	assert.Empty(t, attrs)
}

func TestSortAttrsInPlace_AlreadySorted(t *testing.T) {
	attrs := []ast_domain.HTMLAttribute{
		{Name: "a", Value: "1"},
		{Name: "b", Value: "2"},
		{Name: "c", Value: "3"},
	}

	sortAttrsInPlace(attrs)

	assert.Equal(t, "a", attrs[0].Name)
	assert.Equal(t, "b", attrs[1].Name)
	assert.Equal(t, "c", attrs[2].Name)
}

func TestSortAttrsInPlace_ReverseSorted(t *testing.T) {
	attrs := []ast_domain.HTMLAttribute{
		{Name: "c", Value: "3"},
		{Name: "b", Value: "2"},
		{Name: "a", Value: "1"},
	}

	sortAttrsInPlace(attrs)

	assert.Equal(t, "a", attrs[0].Name)
	assert.Equal(t, "b", attrs[1].Name)
	assert.Equal(t, "c", attrs[2].Name)
}

func TestSortInsertion_SortsCorrectly(t *testing.T) {
	keys := []string{"zebra", "apple", "middle", "banana"}
	sortInsertion(&keys, 4)

	assert.Equal(t, []string{"apple", "banana", "middle", "zebra"}, keys)
}

func TestSortInsertion_HandlesPartialSlice(t *testing.T) {
	keys := []string{"c", "a", "b", "", ""}
	sortInsertion(&keys, 3)

	assert.Equal(t, "a", keys[0])
	assert.Equal(t, "b", keys[1])
	assert.Equal(t, "c", keys[2])
	assert.Equal(t, "", keys[3])
}

func TestSortInsertion_SingleElement(t *testing.T) {
	keys := []string{"only"}
	sortInsertion(&keys, 1)

	assert.Equal(t, "only", keys[0])
}

func TestSortInsertion_TwoElements(t *testing.T) {
	keys := []string{"b", "a"}
	sortInsertion(&keys, 2)

	assert.Equal(t, "a", keys[0])
	assert.Equal(t, "b", keys[1])
}

func TestStringTokeniser_BasicTokenisation(t *testing.T) {
	tok := newStringTokeniser("hello world foo")

	tokens := []string{}
	for tok.Next() {
		tokens = append(tokens, tok.Token())
	}

	assert.Equal(t, []string{"hello", "world", "foo"}, tokens)
}

func TestStringTokeniser_EmptyString(t *testing.T) {
	tok := newStringTokeniser("")

	assert.False(t, tok.Next())
	assert.Empty(t, tok.Token())
}

func TestStringTokeniser_OnlyWhitespace(t *testing.T) {
	tok := newStringTokeniser("   \t\n   ")

	assert.False(t, tok.Next())
}

func TestStringTokeniser_LeadingTrailingWhitespace(t *testing.T) {
	tok := newStringTokeniser("  hello  world  ")

	tokens := []string{}
	for tok.Next() {
		tokens = append(tokens, tok.Token())
	}

	assert.Equal(t, []string{"hello", "world"}, tokens)
}

func TestStringTokeniser_SingleToken(t *testing.T) {
	tok := newStringTokeniser("single")

	assert.True(t, tok.Next())
	assert.Equal(t, "single", tok.Token())
	assert.False(t, tok.Next())
}

func TestStringTokeniser_MultipleSpaces(t *testing.T) {
	tok := newStringTokeniser("a    b     c")

	tokens := []string{}
	for tok.Next() {
		tokens = append(tokens, tok.Token())
	}

	assert.Equal(t, []string{"a", "b", "c"}, tokens)
}

func TestSortAttributeKeys_TwoElements(t *testing.T) {
	keys := []string{"zebra", "apple", "", "", "", "", "", ""}
	sortAttributeKeys(&keys, 2)

	assert.Equal(t, "apple", keys[0])
	assert.Equal(t, "zebra", keys[1])
}

func TestSortAttributeKeys_TwoElementsAlreadySorted(t *testing.T) {
	keys := []string{"apple", "zebra", "", "", "", "", "", ""}
	sortAttributeKeys(&keys, 2)

	assert.Equal(t, "apple", keys[0])
	assert.Equal(t, "zebra", keys[1])
}

func TestSortAttributeKeys_ThreeElements(t *testing.T) {
	keys := []string{"zebra", "middle", "apple", "", "", "", "", ""}
	sortAttributeKeys(&keys, 3)

	assert.Equal(t, "apple", keys[0])
	assert.Equal(t, "middle", keys[1])
	assert.Equal(t, "zebra", keys[2])
}

func TestSortAttributeKeys_FourElements(t *testing.T) {
	keys := []string{"delta", "alpha", "charlie", "bravo", "", "", "", ""}
	sortAttributeKeys(&keys, 4)

	assert.Equal(t, "alpha", keys[0])
	assert.Equal(t, "bravo", keys[1])
	assert.Equal(t, "charlie", keys[2])
	assert.Equal(t, "delta", keys[3])
}

func TestSortAttributeKeys_FiveElements(t *testing.T) {
	keys := []string{"echo", "delta", "alpha", "charlie", "bravo", "", "", ""}
	sortAttributeKeys(&keys, 5)

	assert.Equal(t, "alpha", keys[0])
	assert.Equal(t, "bravo", keys[1])
	assert.Equal(t, "charlie", keys[2])
	assert.Equal(t, "delta", keys[3])
	assert.Equal(t, "echo", keys[4])
}

func TestSortAttributeKeys_SixElements(t *testing.T) {

	keys := []string{"foxtrot", "echo", "delta", "alpha", "charlie", "bravo", "", ""}
	sortAttributeKeys(&keys, 6)

	assert.Equal(t, "alpha", keys[0])
	assert.Equal(t, "bravo", keys[1])
	assert.Equal(t, "charlie", keys[2])
	assert.Equal(t, "delta", keys[3])
	assert.Equal(t, "echo", keys[4])
	assert.Equal(t, "foxtrot", keys[5])
}

func TestSortManual_TwoElements(t *testing.T) {
	keys := []string{"b", "a"}
	sortManual(&keys, 2)

	assert.Equal(t, "a", keys[0])
	assert.Equal(t, "b", keys[1])
}

func TestSortManual_TwoElementsAlreadySorted(t *testing.T) {
	keys := []string{"a", "b"}
	sortManual(&keys, 2)

	assert.Equal(t, "a", keys[0])
	assert.Equal(t, "b", keys[1])
}

func TestSortManual_ThreeElements(t *testing.T) {
	keys := []string{"c", "a", "b"}
	sortManual(&keys, 3)

	assert.Equal(t, "a", keys[0])
	assert.Equal(t, "b", keys[1])
	assert.Equal(t, "c", keys[2])
}

func TestSortManual_ThreeElementsReverseSorted(t *testing.T) {
	keys := []string{"c", "b", "a"}
	sortManual(&keys, 3)

	assert.Equal(t, "a", keys[0])
	assert.Equal(t, "b", keys[1])
	assert.Equal(t, "c", keys[2])
}

func TestSortManual_ThreeElementsAlreadySorted(t *testing.T) {
	keys := []string{"a", "b", "c"}
	sortManual(&keys, 3)

	assert.Equal(t, "a", keys[0])
	assert.Equal(t, "b", keys[1])
	assert.Equal(t, "c", keys[2])
}

func TestTryFastPathTwoClasses_SingleClass(t *testing.T) {

	result, ok := tryFastPathTwoClasses("single", -1)
	assert.False(t, ok)
	assert.Empty(t, result)
}

func TestTryFastPathTwoClasses_TwoDistinctClasses(t *testing.T) {
	result, ok := tryFastPathTwoClasses("class1 class2", 6)
	assert.True(t, ok)
	assert.Equal(t, "class1 class2", result)
}

func TestTryFastPathTwoClasses_TwoDuplicateClasses(t *testing.T) {
	result, ok := tryFastPathTwoClasses("foo foo", 3)
	assert.True(t, ok)
	assert.Equal(t, "foo", result)
}

func TestTryFastPathTwoClasses_MoreThanTwoClasses(t *testing.T) {
	result, ok := tryFastPathTwoClasses("a b c", 1)
	assert.False(t, ok)
	assert.Empty(t, result)
}

func TestTryFastPathTwoClasses_ZeroFirstSpaceIndex(t *testing.T) {
	result, ok := tryFastPathTwoClasses(" class", 0)
	assert.False(t, ok)
	assert.Empty(t, result)
}

func TestAppendDeduplicatedClassesToBuf_Empty(t *testing.T) {
	buffer := make([]byte, 0, 64)
	result := appendDeduplicatedClassesToBuf(buffer, "")
	assert.Empty(t, result)
}

func TestAppendDeduplicatedClassesToBuf_SingleClass(t *testing.T) {
	buffer := make([]byte, 0, 64)
	result := appendDeduplicatedClassesToBuf(buffer, "single")
	assert.Equal(t, "single", string(result))
}

func TestAppendDeduplicatedClassesToBuf_TwoClasses(t *testing.T) {
	buffer := make([]byte, 0, 64)
	result := appendDeduplicatedClassesToBuf(buffer, "foo bar")
	assert.Equal(t, "foo bar", string(result))
}

func TestAppendDeduplicatedClassesToBuf_WithDuplicates(t *testing.T) {
	buffer := make([]byte, 0, 64)
	result := appendDeduplicatedClassesToBuf(buffer, "foo bar foo")
	assert.Equal(t, "foo bar", string(result))
}

func TestAppendDeduplicatedClassesToBuf_ManyClasses(t *testing.T) {
	buffer := make([]byte, 0, 128)

	result := appendDeduplicatedClassesToBuf(buffer, "a b c d e f g a b c")
	assert.Equal(t, "a b c d e f g", string(result))
}

func TestAppendDeduplicatedClassesToBuf_AppendsToExisting(t *testing.T) {
	buffer := []byte("prefix:")
	result := appendDeduplicatedClassesToBuf(buffer, "foo bar")
	assert.Equal(t, "prefix:foo bar", string(result))
}

func TestAnalyseClassString_SingleClass(t *testing.T) {
	analysis := analyseClassString("single")
	assert.False(t, analysis.hasSeparator)
	assert.Equal(t, -1, analysis.firstSpaceIndex)
}

func TestAnalyseClassString_TwoClasses(t *testing.T) {
	analysis := analyseClassString("foo bar")
	assert.True(t, analysis.hasSeparator)
	assert.True(t, analysis.onlySpaces)
	assert.Equal(t, 3, analysis.firstSpaceIndex)
	assert.Equal(t, 2, analysis.estimatedCapacity)
}

func TestAnalyseClassString_WithTabs(t *testing.T) {
	analysis := analyseClassString("foo\tbar")
	assert.True(t, analysis.hasSeparator)
	assert.False(t, analysis.onlySpaces)
}

func TestAnalyseClassString_MultipleSpaces(t *testing.T) {
	analysis := analyseClassString("foo   bar   baz")
	assert.True(t, analysis.hasSeparator)
	assert.True(t, analysis.onlySpaces)
	assert.Equal(t, 3, analysis.firstSpaceIndex)
}

func TestRegisterSVGSymbol_AppendsToNilSlice(t *testing.T) {
	rctx := NewTestRenderContextBuilder().Build()
	rctx.requiredSvgSymbols = nil

	testData := &ParsedSvgData{InnerHTML: "<path/>"}
	registerSVGSymbol("test-icon", testData, rctx)

	require.Len(t, rctx.requiredSvgSymbols, 1)
	assert.Equal(t, "test-icon", rctx.requiredSvgSymbols[0].id)
	assert.Same(t, testData, rctx.requiredSvgSymbols[0].data)
}

func TestRegisterSVGSymbol_AddsToExistingSlice(t *testing.T) {
	rctx := NewTestRenderContextBuilder().Build()
	existingData := &ParsedSvgData{InnerHTML: "<circle/>"}
	rctx.requiredSvgSymbols = []svgSymbolEntry{
		{id: "existing-icon", data: existingData},
	}

	newData := &ParsedSvgData{InnerHTML: "<rect/>"}
	registerSVGSymbol("new-icon", newData, rctx)

	require.Len(t, rctx.requiredSvgSymbols, 2)
	assert.Equal(t, "existing-icon", rctx.requiredSvgSymbols[0].id)
	assert.Equal(t, "new-icon", rctx.requiredSvgSymbols[1].id)
}

func TestRegisterSVGSymbol_DeduplicatesByID(t *testing.T) {
	rctx := NewTestRenderContextBuilder().Build()
	data := &ParsedSvgData{InnerHTML: "<path/>"}

	registerSVGSymbol("icon-a", data, rctx)
	registerSVGSymbol("icon-a", data, rctx)
	registerSVGSymbol("icon-a", data, rctx)

	assert.Len(t, rctx.requiredSvgSymbols, 1)
}

func TestInitCacheIfNeeded_InitialisesNilCache(t *testing.T) {
	rctx := NewTestRenderContextBuilder().Build()
	rctx.mergedAttrsCache = nil

	initialiseCacheIfNeeded(rctx)

	require.NotNil(t, rctx.mergedAttrsCache)
}

func TestInitCacheIfNeeded_SkipsExistingCache(t *testing.T) {
	rctx := NewTestRenderContextBuilder().Build()
	existingCache := make(map[svgCacheKey]string)
	existingCache[svgCacheKey{artefactID: "existing", userAttrsHash: 123}] = "cached"
	rctx.mergedAttrsCache = existingCache

	initialiseCacheIfNeeded(rctx)

	assert.Contains(t, rctx.mergedAttrsCache, svgCacheKey{artefactID: "existing", userAttrsHash: 123})
	assert.Equal(t, "cached", rctx.mergedAttrsCache[svgCacheKey{artefactID: "existing", userAttrsHash: 123}])
}

func TestSortIndicesByAttrName_SortsCorrectly(t *testing.T) {
	attrs := []ast_domain.HTMLAttribute{
		{Name: "zebra", Value: "z"},
		{Name: "apple", Value: "a"},
		{Name: "middle", Value: "m"},
		{Name: "banana", Value: "b"},
	}
	indices := []int{0, 1, 2, 3}

	sortIndicesByAttrName(attrs, indices)

	assert.Equal(t, "apple", attrs[indices[0]].Name)
	assert.Equal(t, "banana", attrs[indices[1]].Name)
	assert.Equal(t, "middle", attrs[indices[2]].Name)
	assert.Equal(t, "zebra", attrs[indices[3]].Name)
}

func TestSortIndicesByAttrName_SingleElement(t *testing.T) {
	attrs := []ast_domain.HTMLAttribute{{Name: "only", Value: "one"}}
	indices := []int{0}

	sortIndicesByAttrName(attrs, indices)

	assert.Equal(t, []int{0}, indices)
}

func TestSortIndicesByAttrName_AlreadySorted(t *testing.T) {
	attrs := []ast_domain.HTMLAttribute{
		{Name: "alpha", Value: "1"},
		{Name: "beta", Value: "2"},
		{Name: "gamma", Value: "3"},
	}
	indices := []int{0, 1, 2}

	sortIndicesByAttrName(attrs, indices)

	assert.Equal(t, []int{0, 1, 2}, indices)
}

func TestSortIndicesByAttrName_ReverseSorted(t *testing.T) {
	attrs := []ast_domain.HTMLAttribute{
		{Name: "c", Value: "3"},
		{Name: "b", Value: "2"},
		{Name: "a", Value: "1"},
	}
	indices := []int{0, 1, 2}

	sortIndicesByAttrName(attrs, indices)

	assert.Equal(t, "a", attrs[indices[0]].Name)
	assert.Equal(t, "b", attrs[indices[1]].Name)
	assert.Equal(t, "c", attrs[indices[2]].Name)
}

func TestExtractUserAttrsOnly_FiltersCorrectly(t *testing.T) {

	attrs := []ast_domain.HTMLAttribute{
		{Name: "src", Value: "icon.svg"},
		{Name: "class", Value: "my-class"},
		{Name: "piko:svg", Value: ""},
		{Name: "id", Value: "my-icon"},
		{Name: "aria-label", Value: "Icon"},
	}

	result := extractUserAttrsOnly(attrs)

	assert.Len(t, result, 3)

	names := make([]string, len(result))
	for i, attr := range result {
		names[i] = attr.Name
	}
	assert.Contains(t, names, "class")
	assert.Contains(t, names, "id")
	assert.Contains(t, names, "aria-label")
	assert.NotContains(t, names, "src")
	assert.NotContains(t, names, "piko:svg")
}

func TestExtractUserAttrsOnly_EmptyInput(t *testing.T) {
	result := extractUserAttrsOnly([]ast_domain.HTMLAttribute{})
	assert.Empty(t, result)
}

func TestExtractUserAttrsOnly_AllFiltered(t *testing.T) {
	attrs := []ast_domain.HTMLAttribute{
		{Name: "src", Value: "icon.svg"},
		{Name: "piko:svg", Value: ""},
	}

	result := extractUserAttrsOnly(attrs)
	assert.Empty(t, result)
}

func TestEstimateLoadedAttrsSize_MultipleClassAttrs(t *testing.T) {
	attrs := []ast_domain.HTMLAttribute{
		{Name: "class", Value: "foo"},
		{Name: "class", Value: "bar"},
		{Name: "id", Value: "test"},
	}

	size, classSize := estimateLoadedAttrsSize(attrs)

	assert.Equal(t, 7, classSize)

	assert.Equal(t, 10, size)
}

func TestEstimateLoadedAttrsSize_NoClasses(t *testing.T) {
	attrs := []ast_domain.HTMLAttribute{
		{Name: "id", Value: "test"},
		{Name: "viewBox", Value: "0 0 24 24"},
	}

	size, classSize := estimateLoadedAttrsSize(attrs)

	assert.Equal(t, 0, classSize)

	expectedSize := (4 + 2 + 4) + (4 + 7 + 9)
	assert.Equal(t, expectedSize, size)
}

func TestGetSortedKeysBuffer_StandardSize(t *testing.T) {
	buffer := getSortedKeysBuffer(10)
	defer putSortedKeysBuffer(buffer)

	require.NotNil(t, buffer)
	assert.GreaterOrEqual(t, cap(*buffer), 10)
}

func TestGetSortedKeysBuffer_LargeSize(t *testing.T) {

	largeSize := standardSortedKeysSize + 10
	buffer := getSortedKeysBuffer(largeSize)

	require.NotNil(t, buffer)
	assert.Equal(t, largeSize, cap(*buffer))
}

func TestHashUserAttrsDirect_LargeAttrCount(t *testing.T) {

	attrs := make([]ast_domain.HTMLAttribute, 12)
	for i := range 12 {
		attrs[i] = ast_domain.HTMLAttribute{
			Name:  string(rune('a' + i)),
			Value: string(rune('A' + i)),
		}
	}

	hash := hashUserAttrsDirect(attrs, nil)

	assert.NotEqual(t, uint64(0), hash)
}

func TestHashUserAttrsDirect_WithSrcAttr(t *testing.T) {

	attrs := []ast_domain.HTMLAttribute{
		{Name: "src", Value: "icon.svg"},
		{Name: "class", Value: "my-class"},
	}

	hash1 := hashUserAttrsDirect(attrs, nil)

	attrs2 := []ast_domain.HTMLAttribute{
		{Name: "class", Value: "my-class"},
	}
	hash2 := hashUserAttrsDirect(attrs2, nil)

	assert.Equal(t, hash1, hash2, "hash should be same with or without src attr")
}

func TestWriteSVGWithAttrs_FormatsCorrectly(t *testing.T) {
	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	writeSVGWithAttrs(qw, ` class="icon" viewBox="0 0 24 24"`, "my-icon")

	output := buffer.String()
	assert.Contains(t, output, "<svg")
	assert.Contains(t, output, `class="icon"`)
	assert.Contains(t, output, `viewBox="0 0 24 24"`)
	assert.Contains(t, output, `<use href="#my-icon">`)
	assert.Contains(t, output, "</use></svg>")
}

func TestWriteSVGWithAttrs_EscapesArtefactID(t *testing.T) {
	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	writeSVGWithAttrs(qw, "", `icon<script>`)

	output := buffer.String()
	assert.Contains(t, output, "&lt;script&gt;")
	assert.NotContains(t, output, "<script>")
}

func TestMergeAndCacheAttrs_CachesResult(t *testing.T) {
	rctx := NewTestRenderContextBuilder().Build()
	initialiseCacheIfNeeded(rctx)

	nodeAttrs := []ast_domain.HTMLAttribute{
		{Name: "class", Value: "user-class"},
	}
	loadedAttrs := []ast_domain.HTMLAttribute{
		{Name: "viewBox", Value: "0 0 24 24"},
	}

	cacheKey := svgCacheKey{artefactID: "test-icon", userAttrsHash: 12345}

	result := mergeAndCacheAttrs(nodeAttrs, nil, loadedAttrs, cacheKey, rctx)

	assert.Contains(t, result, "user-class")
	assert.Contains(t, result, "viewBox")

	cached, exists := rctx.mergedAttrsCache[cacheKey]
	assert.True(t, exists)
	assert.Equal(t, result, cached)
}

func TestIsUserAttr(t *testing.T) {

	testCases := []struct {
		name          string
		attributeName string
		expected      bool
	}{
		{name: "src is not user attr", attributeName: "src", expected: false},
		{name: "piko:svg is not user attr", attributeName: "piko:svg", expected: false},
		{name: "class is user attr", attributeName: "class", expected: true},
		{name: "id is user attr", attributeName: "id", expected: true},
		{name: "data-value is user attr", attributeName: "data-value", expected: true},
		{name: "viewbox is user attr", attributeName: "viewbox", expected: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isUserAttr(tc.attributeName)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestHashAttrsByIndices_DeterministicOutput(t *testing.T) {
	attrs := []ast_domain.HTMLAttribute{
		{Name: "id", Value: "test"},
		{Name: "class", Value: "my-class"},
		{Name: "viewBox", Value: "0 0 24 24"},
	}
	indices := []int{0, 1, 2}

	hash1 := hashAttrsByIndices(attrs, indices)
	hash2 := hashAttrsByIndices(attrs, indices)

	assert.Equal(t, hash1, hash2, "same input should produce same hash")
}

func TestHashAttrsByIndices_DifferentOrderDifferentHash(t *testing.T) {
	attrs := []ast_domain.HTMLAttribute{
		{Name: "a", Value: "1"},
		{Name: "b", Value: "2"},
	}

	hash1 := hashAttrsByIndices(attrs, []int{0, 1})
	hash2 := hashAttrsByIndices(attrs, []int{1, 0})

	assert.NotEqual(t, hash1, hash2, "different order should produce different hash")
}

func TestWriterValueLen(t *testing.T) {
	testCases := []struct {
		setup    func(dw *ast_domain.DirectWriter)
		name     string
		expected int
	}{
		{
			name:     "empty writer returns zero",
			setup:    func(_ *ast_domain.DirectWriter) {},
			expected: 0,
		},
		{
			name: "single string part",
			setup: func(dw *ast_domain.DirectWriter) {
				dw.AppendString("hello")
			},
			expected: 5,
		},
		{
			name: "multiple string parts",
			setup: func(dw *ast_domain.DirectWriter) {
				dw.AppendString("foo")
				dw.AppendString("bar")
			},
			expected: 6,
		},
		{
			name: "single int part",
			setup: func(dw *ast_domain.DirectWriter) {
				dw.AppendInt(12345)
			},
			expected: 5,
		},
		{
			name: "single bool true",
			setup: func(dw *ast_domain.DirectWriter) {
				dw.AppendBool(true)
			},
			expected: 4,
		},
		{
			name: "single bool false",
			setup: func(dw *ast_domain.DirectWriter) {
				dw.AppendBool(false)
			},
			expected: 5,
		},
		{
			name: "single bytes part",
			setup: func(dw *ast_domain.DirectWriter) {
				dw.AppendPooledBytes(new([]byte("rawdata")))
			},
			expected: 7,
		},
		{
			name: "mixed string and int parts",
			setup: func(dw *ast_domain.DirectWriter) {
				dw.AppendString("px-")
				dw.AppendInt(4)
			},
			expected: 4,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dw := &ast_domain.DirectWriter{Name: "test"}
			tc.setup(dw)
			result := writerValueLen(dw)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestAppendWriterAttribute(t *testing.T) {
	testCases := []struct {
		name     string
		initial  string
		setup    func(dw *ast_domain.DirectWriter)
		dwName   string
		expected string
	}{
		{
			name:    "single string value",
			initial: "",
			setup: func(dw *ast_domain.DirectWriter) {
				dw.AppendString("my-value")
			},
			dwName:   "data-id",
			expected: ` data-id="my-value"`,
		},
		{
			name:    "appends to existing buffer",
			initial: `<svg`,
			setup: func(dw *ast_domain.DirectWriter) {
				dw.AppendString("icon")
			},
			dwName:   "class",
			expected: `<svg class="icon"`,
		},
		{
			name:    "multi-part value",
			initial: "",
			setup: func(dw *ast_domain.DirectWriter) {
				dw.AppendString("prefix-")
				dw.AppendInt(42)
			},
			dwName:   "id",
			expected: ` id="prefix-42"`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dw := &ast_domain.DirectWriter{Name: tc.dwName}
			tc.setup(dw)
			buffer := []byte(tc.initial)
			result := appendWriterAttribute(buffer, dw)
			assert.Equal(t, tc.expected, string(result))
		})
	}
}

func TestHasSvgUserAttrWriterByName(t *testing.T) {
	makeWriter := func(name string) *ast_domain.DirectWriter {
		dw := &ast_domain.DirectWriter{Name: name}
		dw.AppendString("value")
		return dw
	}

	testCases := []struct {
		name     string
		search   string
		writers  []*ast_domain.DirectWriter
		expected bool
	}{
		{
			name:     "nil slice returns false",
			writers:  nil,
			search:   "class",
			expected: false,
		},
		{
			name:     "empty slice returns false",
			writers:  []*ast_domain.DirectWriter{},
			search:   "class",
			expected: false,
		},
		{
			name:     "finds matching writer",
			writers:  []*ast_domain.DirectWriter{makeWriter("class"), makeWriter("id")},
			search:   "id",
			expected: true,
		},
		{
			name:     "does not find non-existent name",
			writers:  []*ast_domain.DirectWriter{makeWriter("class")},
			search:   "id",
			expected: false,
		},
		{
			name:     "skips src writer",
			writers:  []*ast_domain.DirectWriter{makeWriter("src")},
			search:   "src",
			expected: false,
		},
		{
			name:     "skips piko:svg writer",
			writers:  []*ast_domain.DirectWriter{makeWriter("piko:svg")},
			search:   "piko:svg",
			expected: false,
		},
		{
			name:     "skips nil writers",
			writers:  []*ast_domain.DirectWriter{nil, makeWriter("class")},
			search:   "class",
			expected: true,
		},
		{
			name:     "finds class among mixed writers",
			writers:  []*ast_domain.DirectWriter{makeWriter("src"), makeWriter("piko:svg"), makeWriter("fill")},
			search:   "fill",
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := hasSvgUserAttrWriterByName(tc.writers, tc.search)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestCollectClassesToBufFromWriters(t *testing.T) {
	makeWriter := func(name, value string) *ast_domain.DirectWriter {
		dw := &ast_domain.DirectWriter{Name: name}
		dw.AppendString(value)
		return dw
	}

	testCases := []struct {
		name     string
		initial  string
		expected string
		writers  []*ast_domain.DirectWriter
	}{
		{
			name:     "nil writers returns empty",
			initial:  "",
			writers:  nil,
			expected: "",
		},
		{
			name:     "empty writers returns empty",
			initial:  "",
			writers:  []*ast_domain.DirectWriter{},
			expected: "",
		},
		{
			name:    "single class writer",
			initial: "",
			writers: []*ast_domain.DirectWriter{
				makeWriter("class", "btn"),
			},
			expected: "btn",
		},
		{
			name:    "multiple class writers",
			initial: "",
			writers: []*ast_domain.DirectWriter{
				makeWriter("class", "btn"),
				makeWriter("class", "primary"),
			},
			expected: "btn primary",
		},
		{
			name:    "skips non-class writers",
			initial: "",
			writers: []*ast_domain.DirectWriter{
				makeWriter("id", "my-id"),
				makeWriter("class", "active"),
				makeWriter("fill", "red"),
			},
			expected: "active",
		},
		{
			name:    "skips src writer",
			initial: "",
			writers: []*ast_domain.DirectWriter{
				makeWriter("src", "icon.svg"),
				makeWriter("class", "icon"),
			},
			expected: "icon",
		},
		{
			name:    "skips piko:svg writer",
			initial: "",
			writers: []*ast_domain.DirectWriter{
				makeWriter("piko:svg", ""),
				makeWriter("class", "icon"),
			},
			expected: "icon",
		},
		{
			name:    "skips nil writers",
			initial: "",
			writers: []*ast_domain.DirectWriter{
				nil,
				makeWriter("class", "valid"),
			},
			expected: "valid",
		},
		{
			name:    "appends to existing buffer content",
			initial: "existing",
			writers: []*ast_domain.DirectWriter{
				makeWriter("class", "new"),
			},
			expected: "existing new",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buffer := []byte(tc.initial)
			result := collectClassesToBufFromWriters(buffer, tc.writers)
			assert.Equal(t, tc.expected, string(result))
		})
	}
}

func TestAppendDeduplicatedClassesToBufFromBytes(t *testing.T) {
	testCases := []struct {
		name     string
		initial  string
		input    string
		expected string
	}{
		{
			name:     "empty input returns unchanged buffer",
			initial:  "",
			input:    "",
			expected: "",
		},
		{
			name:     "single class",
			initial:  "",
			input:    "btn",
			expected: "btn",
		},
		{
			name:     "two distinct classes",
			initial:  "",
			input:    "btn primary",
			expected: "btn primary",
		},
		{
			name:     "duplicate classes are deduplicated",
			initial:  "",
			input:    "btn btn",
			expected: "btn",
		},
		{
			name:     "multiple duplicates",
			initial:  "",
			input:    "a b c a b c",
			expected: "a b c",
		},
		{
			name:     "preserves first occurrence order",
			initial:  "",
			input:    "z a m z a",
			expected: "z a m",
		},
		{
			name:     "appends to existing buffer",
			initial:  "prefix:",
			input:    "foo bar",
			expected: "prefix:foo bar",
		},
		{
			name:     "handles multiple spaces between classes",
			initial:  "",
			input:    "a   b   c",
			expected: "a b c",
		},
		{
			name:     "many unique classes",
			initial:  "",
			input:    "a b c d e f g h",
			expected: "a b c d e f g h",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buffer := []byte(tc.initial)
			result := appendDeduplicatedClassesToBufFromBytes(buffer, []byte(tc.input))
			assert.Equal(t, tc.expected, string(result))
		})
	}
}

func TestCountUserAttrWriters(t *testing.T) {
	makeWriter := func(name string) *ast_domain.DirectWriter {
		dw := &ast_domain.DirectWriter{Name: name}
		dw.AppendString("v")
		return dw
	}

	testCases := []struct {
		name     string
		writers  []*ast_domain.DirectWriter
		expected int
	}{
		{
			name:     "nil slice returns zero",
			writers:  nil,
			expected: 0,
		},
		{
			name:     "empty slice returns zero",
			writers:  []*ast_domain.DirectWriter{},
			expected: 0,
		},
		{
			name:     "all user attrs",
			writers:  []*ast_domain.DirectWriter{makeWriter("class"), makeWriter("id"), makeWriter("fill")},
			expected: 3,
		},
		{
			name:     "excludes src",
			writers:  []*ast_domain.DirectWriter{makeWriter("src"), makeWriter("class")},
			expected: 1,
		},
		{
			name:     "excludes piko:svg",
			writers:  []*ast_domain.DirectWriter{makeWriter("piko:svg"), makeWriter("id")},
			expected: 1,
		},
		{
			name:     "skips nil writers",
			writers:  []*ast_domain.DirectWriter{nil, makeWriter("class"), nil},
			expected: 1,
		},
		{
			name:     "only non-user attrs returns zero",
			writers:  []*ast_domain.DirectWriter{makeWriter("src"), makeWriter("piko:svg")},
			expected: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := countUserAttrWriters(tc.writers)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestInsertionSortAttrRefs(t *testing.T) {
	testCases := []struct {
		name     string
		refs     []attributeReference
		expected []string
	}{
		{
			name:     "empty slice",
			refs:     []attributeReference{},
			expected: []string{},
		},
		{
			name:     "single element",
			refs:     []attributeReference{{name: "alpha"}},
			expected: []string{"alpha"},
		},
		{
			name:     "already sorted",
			refs:     []attributeReference{{name: "a"}, {name: "b"}, {name: "c"}},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "reverse sorted",
			refs:     []attributeReference{{name: "c"}, {name: "b"}, {name: "a"}},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "unsorted",
			refs:     []attributeReference{{name: "zebra"}, {name: "apple"}, {name: "middle"}, {name: "banana"}},
			expected: []string{"apple", "banana", "middle", "zebra"},
		},
		{
			name:     "two elements swapped",
			refs:     []attributeReference{{name: "b"}, {name: "a"}},
			expected: []string{"a", "b"},
		},
		{
			name:     "duplicate names",
			refs:     []attributeReference{{name: "x"}, {name: "a"}, {name: "x"}},
			expected: []string{"a", "x", "x"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			insertionSortAttrRefs(tc.refs)
			names := make([]string, len(tc.refs))
			for i, ref := range tc.refs {
				names[i] = ref.name
			}
			assert.Equal(t, tc.expected, names)
		})
	}
}

func TestHashSortedAttrRefs(t *testing.T) {
	t.Run("deterministic output", func(t *testing.T) {
		refs := []attributeReference{
			{name: "class", value: "icon"},
			{name: "id", value: "my-id"},
		}

		hash1 := hashSortedAttrRefs(refs)
		hash2 := hashSortedAttrRefs(refs)

		assert.Equal(t, hash1, hash2, "same input should produce same hash")
	})

	t.Run("different values produce different hash", func(t *testing.T) {
		refs1 := []attributeReference{
			{name: "class", value: "foo"},
		}
		refs2 := []attributeReference{
			{name: "class", value: "bar"},
		}

		hash1 := hashSortedAttrRefs(refs1)
		hash2 := hashSortedAttrRefs(refs2)

		assert.NotEqual(t, hash1, hash2, "different values should produce different hashes")
	})

	t.Run("different names produce different hash", func(t *testing.T) {
		refs1 := []attributeReference{
			{name: "class", value: "x"},
		}
		refs2 := []attributeReference{
			{name: "id", value: "x"},
		}

		hash1 := hashSortedAttrRefs(refs1)
		hash2 := hashSortedAttrRefs(refs2)

		assert.NotEqual(t, hash1, hash2, "different names should produce different hashes")
	})

	t.Run("empty refs produces a hash", func(t *testing.T) {
		refs := []attributeReference{}
		result := hashSortedAttrRefs(refs)

		assert.NotEqual(t, uint64(0), result)
	})

	t.Run("writer ref produces different hash than static ref", func(t *testing.T) {
		dw := &ast_domain.DirectWriter{Name: "class"}
		dw.AppendString("icon")

		refsStatic := []attributeReference{
			{name: "class", value: "icon"},
		}
		refsWriter := []attributeReference{
			{name: "class", writer: dw},
		}

		hashStatic := hashSortedAttrRefs(refsStatic)
		hashWriter := hashSortedAttrRefs(refsWriter)

		assert.Equal(t, hashStatic, hashWriter,
			"static and writer refs with same content should produce same hash")
	})
}

func TestHashSingleAttrRef(t *testing.T) {
	t.Run("static ref is deterministic", func(t *testing.T) {
		ref := attributeReference{name: "id", value: "test"}

		h1 := fnv.New64()
		hashSingleAttrRef(h1, &ref)
		sum1 := h1.Sum64()

		h2 := fnv.New64()
		hashSingleAttrRef(h2, &ref)
		sum2 := h2.Sum64()

		assert.Equal(t, sum1, sum2)
	})

	t.Run("writer ref delegates to hashWriterValue", func(t *testing.T) {
		dw := &ast_domain.DirectWriter{Name: "fill"}
		dw.AppendString("red")

		ref := attributeReference{name: "fill", writer: dw}

		h := fnv.New64()
		hashSingleAttrRef(h, &ref)
		result := h.Sum64()

		assert.NotEqual(t, uint64(0), result)
	})

	t.Run("different values produce different results", func(t *testing.T) {
		ref1 := attributeReference{name: "id", value: "alpha"}
		ref2 := attributeReference{name: "id", value: "beta"}

		h1 := fnv.New64()
		hashSingleAttrRef(h1, &ref1)

		h2 := fnv.New64()
		hashSingleAttrRef(h2, &ref2)

		assert.NotEqual(t, h1.Sum64(), h2.Sum64())
	})
}

func TestHashWriterValue(t *testing.T) {
	t.Run("single string value uses fast path", func(t *testing.T) {
		dw := &ast_domain.DirectWriter{Name: "class"}
		dw.AppendString("icon")

		h := fnv.New64()
		hashWriterValue(h, dw)
		result := h.Sum64()

		assert.NotEqual(t, uint64(0), result)
	})

	t.Run("multi-part value uses parts path", func(t *testing.T) {
		dw := &ast_domain.DirectWriter{Name: "data"}
		dw.AppendString("prefix-")
		dw.AppendInt(42)

		h := fnv.New64()
		hashWriterValue(h, dw)
		result := h.Sum64()

		assert.NotEqual(t, uint64(0), result)
	})

	t.Run("same content produces same hash", func(t *testing.T) {
		dw1 := &ast_domain.DirectWriter{Name: "class"}
		dw1.AppendString("icon")

		dw2 := &ast_domain.DirectWriter{Name: "class"}
		dw2.AppendString("icon")

		h1 := fnv.New64()
		hashWriterValue(h1, dw1)

		h2 := fnv.New64()
		hashWriterValue(h2, dw2)

		assert.Equal(t, h1.Sum64(), h2.Sum64())
	})

	t.Run("different content produces different hash", func(t *testing.T) {
		dw1 := &ast_domain.DirectWriter{Name: "class"}
		dw1.AppendString("foo")

		dw2 := &ast_domain.DirectWriter{Name: "class"}
		dw2.AppendString("bar")

		h1 := fnv.New64()
		hashWriterValue(h1, dw1)

		h2 := fnv.New64()
		hashWriterValue(h2, dw2)

		assert.NotEqual(t, h1.Sum64(), h2.Sum64())
	})
}

func TestHashWriterParts(t *testing.T) {
	t.Run("string part", func(t *testing.T) {
		dw := &ast_domain.DirectWriter{Name: "test"}
		dw.AppendString("hello")

		h := fnv.New64()
		hashWriterParts(h, dw)

		assert.NotEqual(t, uint64(0), h.Sum64())
	})

	t.Run("int part", func(t *testing.T) {
		dw := &ast_domain.DirectWriter{Name: "test"}
		dw.AppendInt(999)

		h := fnv.New64()
		hashWriterParts(h, dw)

		assert.NotEqual(t, uint64(0), h.Sum64())
	})

	t.Run("bool true part", func(t *testing.T) {
		dw := &ast_domain.DirectWriter{Name: "test"}
		dw.AppendBool(true)

		h := fnv.New64()
		hashWriterParts(h, dw)

		assert.NotEqual(t, uint64(0), h.Sum64())
	})

	t.Run("bool false part", func(t *testing.T) {
		dw := &ast_domain.DirectWriter{Name: "test"}
		dw.AppendBool(false)

		h := fnv.New64()
		hashWriterParts(h, dw)

		assert.NotEqual(t, uint64(0), h.Sum64())
	})

	t.Run("float part", func(t *testing.T) {
		dw := &ast_domain.DirectWriter{Name: "test"}
		dw.AppendFloat(3.14)

		h := fnv.New64()
		hashWriterParts(h, dw)

		assert.NotEqual(t, uint64(0), h.Sum64())
	})

	t.Run("bytes part", func(t *testing.T) {
		dw := &ast_domain.DirectWriter{Name: "test"}
		dw.AppendPooledBytes(new([]byte("raw-bytes")))

		h := fnv.New64()
		hashWriterParts(h, dw)

		assert.NotEqual(t, uint64(0), h.Sum64())
	})

	t.Run("same parts produce same hash", func(t *testing.T) {
		dw1 := &ast_domain.DirectWriter{Name: "test"}
		dw1.AppendString("abc")
		dw1.AppendInt(123)

		dw2 := &ast_domain.DirectWriter{Name: "test"}
		dw2.AppendString("abc")
		dw2.AppendInt(123)

		h1 := fnv.New64()
		hashWriterParts(h1, dw1)

		h2 := fnv.New64()
		hashWriterParts(h2, dw2)

		assert.Equal(t, h1.Sum64(), h2.Sum64())
	})

	t.Run("different parts produce different hash", func(t *testing.T) {
		dw1 := &ast_domain.DirectWriter{Name: "test"}
		dw1.AppendString("abc")

		dw2 := &ast_domain.DirectWriter{Name: "test"}
		dw2.AppendString("xyz")

		h1 := fnv.New64()
		hashWriterParts(h1, dw1)

		h2 := fnv.New64()
		hashWriterParts(h2, dw2)

		assert.NotEqual(t, h1.Sum64(), h2.Sum64())
	})

	t.Run("bool true vs false produce different hash", func(t *testing.T) {
		dw1 := &ast_domain.DirectWriter{Name: "test"}
		dw1.AppendBool(true)

		dw2 := &ast_domain.DirectWriter{Name: "test"}
		dw2.AppendBool(false)

		h1 := fnv.New64()
		hashWriterParts(h1, dw1)

		h2 := fnv.New64()
		hashWriterParts(h2, dw2)

		assert.NotEqual(t, h1.Sum64(), h2.Sum64())
	})
}

func TestHashAttrsWithWriters(t *testing.T) {
	makeWriter := func(name, value string) *ast_domain.DirectWriter {
		dw := &ast_domain.DirectWriter{Name: name}
		dw.AppendString(value)
		return dw
	}

	t.Run("deterministic for same inputs", func(t *testing.T) {
		attrs := []ast_domain.HTMLAttribute{
			{Name: "class", Value: "icon"},
			{Name: "id", Value: "test"},
		}
		indices := []int{0, 1}
		writers := []*ast_domain.DirectWriter{makeWriter("fill", "red")}

		hash1 := hashAttrsWithWriters(attrs, indices, 2, writers, 1)
		hash2 := hashAttrsWithWriters(attrs, indices, 2, writers, 1)

		assert.Equal(t, hash1, hash2)
	})

	t.Run("different attrs produce different hash", func(t *testing.T) {
		attrs1 := []ast_domain.HTMLAttribute{
			{Name: "class", Value: "icon"},
		}
		attrs2 := []ast_domain.HTMLAttribute{
			{Name: "class", Value: "button"},
		}

		hash1 := hashAttrsWithWriters(attrs1, []int{0}, 1, nil, 0)
		hash2 := hashAttrsWithWriters(attrs2, []int{0}, 1, nil, 0)

		assert.NotEqual(t, hash1, hash2)
	})

	t.Run("same attrs in different order produce same hash via sorting", func(t *testing.T) {
		attrs := []ast_domain.HTMLAttribute{
			{Name: "fill", Value: "red"},
			{Name: "class", Value: "icon"},
		}

		hash1 := hashAttrsWithWriters(attrs, []int{0, 1}, 2, nil, 0)
		hash2 := hashAttrsWithWriters(attrs, []int{1, 0}, 2, nil, 0)

		assert.Equal(t, hash1, hash2,
			"same attributes in different index order should produce same hash after sorting")
	})

	t.Run("fast path used for 16 or fewer combined attrs", func(t *testing.T) {
		attrs := make([]ast_domain.HTMLAttribute, 10)
		indices := make([]int, 10)
		for i := range 10 {
			attrs[i] = ast_domain.HTMLAttribute{
				Name:  string(rune('a' + i)),
				Value: string(rune('A' + i)),
			}
			indices[i] = i
		}
		writers := []*ast_domain.DirectWriter{
			makeWriter("w1", "v1"),
			makeWriter("w2", "v2"),
		}

		result := hashAttrsWithWriters(attrs, indices, 10, writers, 2)
		assert.NotEqual(t, uint64(0), result)
	})

	t.Run("slow path used for more than 16 combined attrs", func(t *testing.T) {
		attrs := make([]ast_domain.HTMLAttribute, 15)
		indices := make([]int, 15)
		for i := range 15 {
			attrs[i] = ast_domain.HTMLAttribute{
				Name:  fmt.Sprintf("attr%02d", i),
				Value: fmt.Sprintf("val%02d", i),
			}
			indices[i] = i
		}
		writers := make([]*ast_domain.DirectWriter, 5)
		for i := range 5 {
			writers[i] = makeWriter(fmt.Sprintf("dw%02d", i), fmt.Sprintf("dwval%02d", i))
		}

		result := hashAttrsWithWriters(attrs, indices, 15, writers, 5)
		assert.NotEqual(t, uint64(0), result)
	})
}

func TestHashAttrsWithWritersFast(t *testing.T) {
	makeWriter := func(name, value string) *ast_domain.DirectWriter {
		dw := &ast_domain.DirectWriter{Name: name}
		dw.AppendString(value)
		return dw
	}

	t.Run("static attrs only", func(t *testing.T) {
		attrs := []ast_domain.HTMLAttribute{
			{Name: "id", Value: "test"},
			{Name: "class", Value: "icon"},
		}

		result := hashAttrsWithWritersFast(attrs, []int{0, 1}, 2, nil)
		assert.NotEqual(t, uint64(0), result)
	})

	t.Run("mixed static and writer attrs", func(t *testing.T) {
		attrs := []ast_domain.HTMLAttribute{
			{Name: "id", Value: "test"},
		}
		writers := []*ast_domain.DirectWriter{makeWriter("fill", "red")}

		result := hashAttrsWithWritersFast(attrs, []int{0}, 1, writers)
		assert.NotEqual(t, uint64(0), result)
	})

	t.Run("deterministic output", func(t *testing.T) {
		attrs := []ast_domain.HTMLAttribute{
			{Name: "viewBox", Value: "0 0 24 24"},
		}
		writers := []*ast_domain.DirectWriter{makeWriter("class", "icon")}

		hash1 := hashAttrsWithWritersFast(attrs, []int{0}, 1, writers)
		hash2 := hashAttrsWithWritersFast(attrs, []int{0}, 1, writers)

		assert.Equal(t, hash1, hash2)
	})
}

func TestCollectAttrRefs(t *testing.T) {
	makeWriter := func(name, value string) *ast_domain.DirectWriter {
		dw := &ast_domain.DirectWriter{Name: name}
		dw.AppendString(value)
		return dw
	}

	t.Run("static attrs only", func(t *testing.T) {
		attrs := []ast_domain.HTMLAttribute{
			{Name: "id", Value: "test"},
			{Name: "class", Value: "icon"},
		}
		var refs [maxCombinedAttrCount]attributeReference

		n := collectAttrRefs(&refs, attrs, []int{0, 1}, 2, nil)

		assert.Equal(t, 2, n)
		assert.Equal(t, "id", refs[0].name)
		assert.Equal(t, "test", refs[0].value)
		assert.Equal(t, "class", refs[1].name)
		assert.Equal(t, "icon", refs[1].value)
	})

	t.Run("writer attrs only", func(t *testing.T) {
		writers := []*ast_domain.DirectWriter{makeWriter("fill", "red")}
		var refs [maxCombinedAttrCount]attributeReference

		n := collectAttrRefs(&refs, nil, nil, 0, writers)

		assert.Equal(t, 1, n)
		assert.Equal(t, "fill", refs[0].name)
		assert.NotNil(t, refs[0].writer)
	})

	t.Run("mixed static and writer", func(t *testing.T) {
		attrs := []ast_domain.HTMLAttribute{
			{Name: "id", Value: "test"},
		}
		writers := []*ast_domain.DirectWriter{makeWriter("class", "icon")}
		var refs [maxCombinedAttrCount]attributeReference

		n := collectAttrRefs(&refs, attrs, []int{0}, 1, writers)

		assert.Equal(t, 2, n)
		assert.Equal(t, "id", refs[0].name)
		assert.Equal(t, "class", refs[1].name)
	})

	t.Run("skips src and piko:svg writers", func(t *testing.T) {
		writers := []*ast_domain.DirectWriter{
			makeWriter("src", "icon.svg"),
			makeWriter("piko:svg", ""),
			makeWriter("class", "valid"),
		}
		var refs [maxCombinedAttrCount]attributeReference

		n := collectAttrRefs(&refs, nil, nil, 0, writers)

		assert.Equal(t, 1, n)
		assert.Equal(t, "class", refs[0].name)
	})

	t.Run("skips nil writers", func(t *testing.T) {
		writers := []*ast_domain.DirectWriter{nil, makeWriter("id", "test")}
		var refs [maxCombinedAttrCount]attributeReference

		n := collectAttrRefs(&refs, nil, nil, 0, writers)

		assert.Equal(t, 1, n)
		assert.Equal(t, "id", refs[0].name)
	})

	t.Run("respects staticCount limit", func(t *testing.T) {
		attrs := []ast_domain.HTMLAttribute{
			{Name: "a", Value: "1"},
			{Name: "b", Value: "2"},
			{Name: "c", Value: "3"},
		}
		var refs [maxCombinedAttrCount]attributeReference

		n := collectAttrRefs(&refs, attrs, []int{0, 1, 2}, 2, nil)

		assert.Equal(t, 2, n)
		assert.Equal(t, "a", refs[0].name)
		assert.Equal(t, "b", refs[1].name)
	})

	t.Run("empty inputs return zero", func(t *testing.T) {
		var refs [maxCombinedAttrCount]attributeReference

		n := collectAttrRefs(&refs, nil, nil, 0, nil)

		assert.Equal(t, 0, n)
	})
}

func TestHashAttrsWithWritersSlow(t *testing.T) {
	makeWriter := func(name, value string) *ast_domain.DirectWriter {
		dw := &ast_domain.DirectWriter{Name: name}
		dw.AppendString(value)
		return dw
	}

	t.Run("static attrs only", func(t *testing.T) {
		attrs := []ast_domain.HTMLAttribute{
			{Name: "id", Value: "test"},
			{Name: "class", Value: "icon"},
		}

		result := hashAttrsWithWritersSlow(attrs, 2, nil)
		assert.NotEqual(t, uint64(0), result)
	})

	t.Run("with writer attrs", func(t *testing.T) {
		attrs := []ast_domain.HTMLAttribute{
			{Name: "id", Value: "test"},
		}
		writers := []*ast_domain.DirectWriter{makeWriter("fill", "red")}

		result := hashAttrsWithWritersSlow(attrs, 1, writers)
		assert.NotEqual(t, uint64(0), result)
	})

	t.Run("excludes src from static attrs", func(t *testing.T) {
		attrsWithSrc := []ast_domain.HTMLAttribute{
			{Name: "src", Value: "icon.svg"},
			{Name: "class", Value: "icon"},
		}
		attrsWithoutSrc := []ast_domain.HTMLAttribute{
			{Name: "class", Value: "icon"},
		}

		hash1 := hashAttrsWithWritersSlow(attrsWithSrc, 2, nil)
		hash2 := hashAttrsWithWritersSlow(attrsWithoutSrc, 1, nil)

		assert.Equal(t, hash1, hash2,
			"src attr should be excluded so hashes should match")
	})

	t.Run("excludes src and piko:svg writers", func(t *testing.T) {
		attrs := []ast_domain.HTMLAttribute{
			{Name: "class", Value: "icon"},
		}
		writers := []*ast_domain.DirectWriter{
			makeWriter("src", "icon.svg"),
			makeWriter("piko:svg", ""),
		}

		hashWithWriters := hashAttrsWithWritersSlow(attrs, 1, writers)
		hashWithout := hashAttrsWithWritersSlow(attrs, 1, nil)

		assert.Equal(t, hashWithWriters, hashWithout,
			"non-user writers should be excluded")
	})

	t.Run("deterministic output", func(t *testing.T) {
		attrs := []ast_domain.HTMLAttribute{
			{Name: "class", Value: "icon"},
		}
		writers := []*ast_domain.DirectWriter{makeWriter("fill", "red")}

		hash1 := hashAttrsWithWritersSlow(attrs, 1, writers)
		hash2 := hashAttrsWithWritersSlow(attrs, 1, writers)

		assert.Equal(t, hash1, hash2)
	})
}

func TestEstimateMergedAttrsSizeInline(t *testing.T) {
	makeWriter := func(name, value string) *ast_domain.DirectWriter {
		dw := &ast_domain.DirectWriter{Name: name}
		dw.AppendString(value)
		return dw
	}

	testCases := []struct {
		name        string
		loadedAttrs []ast_domain.HTMLAttribute
		nodeAttrs   []ast_domain.HTMLAttribute
		writers     []*ast_domain.DirectWriter
		minExpected int
	}{
		{
			name:        "empty inputs",
			loadedAttrs: nil,
			nodeAttrs:   nil,
			writers:     nil,
			minExpected: 0,
		},
		{
			name: "loaded attrs only",
			loadedAttrs: []ast_domain.HTMLAttribute{
				{Name: "viewBox", Value: "0 0 24 24"},
			},
			nodeAttrs:   nil,
			writers:     nil,
			minExpected: 1,
		},
		{
			name:        "node attrs only",
			loadedAttrs: nil,
			nodeAttrs: []ast_domain.HTMLAttribute{
				{Name: "id", Value: "test"},
			},
			writers:     nil,
			minExpected: 1,
		},
		{
			name: "with writer attrs",
			loadedAttrs: []ast_domain.HTMLAttribute{
				{Name: "viewBox", Value: "0 0 24 24"},
			},
			nodeAttrs:   nil,
			writers:     []*ast_domain.DirectWriter{makeWriter("fill", "red")},
			minExpected: 1,
		},
		{
			name: "class attrs from writers add to class size",
			loadedAttrs: []ast_domain.HTMLAttribute{
				{Name: "class", Value: "svg-icon"},
			},
			nodeAttrs: []ast_domain.HTMLAttribute{
				{Name: "src", Value: "icon.svg"},
			},
			writers:     []*ast_domain.DirectWriter{makeWriter("class", "extra")},
			minExpected: 1,
		},
		{
			name:        "skips src and piko:svg writers",
			loadedAttrs: nil,
			nodeAttrs:   nil,
			writers: []*ast_domain.DirectWriter{
				makeWriter("src", "icon.svg"),
				makeWriter("piko:svg", ""),
				makeWriter("id", "test"),
			},
			minExpected: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := estimateMergedAttrsSizeInline(tc.loadedAttrs, tc.nodeAttrs, tc.writers)
			assert.GreaterOrEqual(t, result, tc.minExpected)
		})
	}
}

func TestAppendWriterAttrsFiltered(t *testing.T) {
	makeWriter := func(name, value string) *ast_domain.DirectWriter {
		dw := &ast_domain.DirectWriter{Name: name}
		dw.AppendString(value)
		return dw
	}

	testCases := []struct {
		name        string
		initial     string
		writers     []*ast_domain.DirectWriter
		expected    string
		notExpected []string
	}{
		{
			name:     "nil writers returns unchanged buffer",
			initial:  "<svg",
			writers:  nil,
			expected: "<svg",
		},
		{
			name:     "empty writers returns unchanged buffer",
			initial:  "<svg",
			writers:  []*ast_domain.DirectWriter{},
			expected: "<svg",
		},
		{
			name:    "appends user attribute",
			initial: "",
			writers: []*ast_domain.DirectWriter{
				makeWriter("fill", "red"),
			},
			expected: ` fill="red"`,
		},
		{
			name:    "skips class writer",
			initial: "",
			writers: []*ast_domain.DirectWriter{
				makeWriter("class", "icon"),
				makeWriter("fill", "red"),
			},
			expected:    ` fill="red"`,
			notExpected: []string{"class"},
		},
		{
			name:    "skips src writer",
			initial: "",
			writers: []*ast_domain.DirectWriter{
				makeWriter("src", "icon.svg"),
				makeWriter("id", "my-id"),
			},
			expected:    ` id="my-id"`,
			notExpected: []string{"src"},
		},
		{
			name:    "skips piko:svg writer",
			initial: "",
			writers: []*ast_domain.DirectWriter{
				makeWriter("piko:svg", ""),
				makeWriter("fill", "blue"),
			},
			expected:    ` fill="blue"`,
			notExpected: []string{"piko:svg"},
		},
		{
			name:    "skips nil writers",
			initial: "",
			writers: []*ast_domain.DirectWriter{
				nil,
				makeWriter("stroke", "black"),
			},
			expected: ` stroke="black"`,
		},
		{
			name:    "multiple valid writers",
			initial: "",
			writers: []*ast_domain.DirectWriter{
				makeWriter("fill", "red"),
				makeWriter("stroke", "black"),
				makeWriter("opacity", "0.5"),
			},
			expected: ` fill="red" stroke="black" opacity="0.5"`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buffer := []byte(tc.initial)
			result := appendWriterAttrsFiltered(buffer, tc.writers)
			assert.Equal(t, tc.expected, string(result))
			for _, ne := range tc.notExpected {
				assert.NotContains(t, string(result), ne)
			}
		})
	}
}
