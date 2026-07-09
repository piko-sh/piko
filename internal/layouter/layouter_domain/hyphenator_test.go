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

package layouter_domain

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_ParsesPatterns(t *testing.T) {
	h := NewHyphenator("hy1p\n.al3t\n")
	require.NotNil(t, h, "expected non-nil Hyphenator")
	require.NotNil(t, h.trie, "expected non-nil trie root")
}

func TestNew_SkipsCommentsAndEmptyLines(t *testing.T) {
	patterns := "% this is a comment\n\nhy1p\n% another comment\n"
	h := NewHyphenator(patterns)
	require.NotNil(t, h, "expected non-nil Hyphenator")
}

func TestHyphenate_ShortWords(t *testing.T) {
	h := DefaultRegistry().Get("en-us")

	for _, word := range []string{"", "a", "ab", "abc", "abcd"} {
		points := h.Hyphenate(word)
		assert.Nil(t, points, "Hyphenate(%q) should be nil", word)
	}
}

func TestHyphenate_KnownWords(t *testing.T) {
	h := DefaultRegistry().Get("en-us")

	tests := []struct {
		word     string
		wantMin  int
		wantText string
	}{

		{word: "hyphenation", wantMin: 2},

		{word: "algorithm", wantMin: 1},

		{word: "computer", wantMin: 1},
	}

	for _, tt := range tests {
		points := h.Hyphenate(tt.word)
		assert.GreaterOrEqual(t, len(points), tt.wantMin,
			"Hyphenate(%q) returned %d points %v, want at least %d",
			tt.word, len(points), points, tt.wantMin)
	}
}

func TestHyphenate_UpperCase(t *testing.T) {
	h := DefaultRegistry().Get("en-us")
	lower := h.Hyphenate("hyphenation")
	upper := h.Hyphenate("HYPHENATION")
	mixed := h.Hyphenate("Hyphenation")

	require.NotEmpty(t, lower, "expected break points for 'hyphenation'")
	assert.Len(t, upper, len(lower), "uppercase should give same points as lowercase")
	assert.Len(t, mixed, len(lower), "mixed case should give same points as lowercase")
}

func TestHyphenate_BreakPointBounds(t *testing.T) {
	h := DefaultRegistry().Get("en-us")

	word := "international"
	points := h.Hyphenate(word)
	runes := []rune(word)
	for _, p := range points {
		assert.GreaterOrEqual(t, p, h.leftMin, "break point %d violates leftMin %d", p, h.leftMin)
		assert.LessOrEqual(t, p, len(runes)-h.rightMin,
			"break point %d violates rightMin %d (word len %d)", p, h.rightMin, len(runes))
	}
}

func TestInsertSoftHyphens(t *testing.T) {
	h := DefaultRegistry().Get("en-us")
	result := h.InsertSoftHyphens("hyphenation")
	assert.Contains(t, result, "\u00AD", "InsertSoftHyphens('hyphenation') expected soft hyphens")

	cleaned := strings.ReplaceAll(result, "\u00AD", "")
	assert.Equal(t, "hyphenation", cleaned, "after removing soft hyphens")
}

func TestInsertSoftHyphens_NoBreaks(t *testing.T) {
	h := DefaultRegistry().Get("en-us")

	result := h.InsertSoftHyphens("cat")
	assert.Equal(t, "cat", result)
}

func TestInsertSoftHyphens_EmptyString(t *testing.T) {
	h := DefaultRegistry().Get("en-us")
	result := h.InsertSoftHyphens("")
	assert.Equal(t, "", result)
}

func TestInsertSoftHyphens_PreservesOriginalCase(t *testing.T) {
	h := DefaultRegistry().Get("en-us")
	result := h.InsertSoftHyphens("Hyphenation")
	cleaned := strings.ReplaceAll(result, "\u00AD", "")
	assert.Equal(t, "Hyphenation", cleaned, "case not preserved")
}

func TestRegistry_Get_DefaultsToEnUS(t *testing.T) {
	r := DefaultRegistry()
	h1 := r.Get("")
	h2 := r.Get("en-us")
	assert.Equal(t, h2, h1, "empty language should return same hyphenator as en-us")
}

func TestRegistry_Get_NormalisesVariants(t *testing.T) {
	r := DefaultRegistry()
	enUS := r.Get("en-us")
	for _, lang := range []string{"en", "en-gb", "EN-US", "En"} {
		h := r.Get(lang)
		assert.Equal(t, enUS, h, "Get(%q) returned different hyphenator than en-us", lang)
	}
}

func TestRegistry_Get_UnsupportedFallsBack(t *testing.T) {
	r := DefaultRegistry()
	enUS := r.Get("en-us")
	h := r.Get("xx-unknown")
	assert.Equal(t, enUS, h, "unsupported language should fall back to en-us")
}

func TestHyphenate_SimplePattern(t *testing.T) {

	h := NewHyphenator("ab1c")

	points := h.Hyphenate("xabcy")
	assert.Nil(t, points, "Hyphenate('xabcy') should be nil (rightMin violation)")

	points = h.Hyphenate("xabcyz")
	assert.Equal(t, []int{3}, points, "Hyphenate('xabcyz') with pattern 'ab1c'")
}
