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

	"github.com/stretchr/testify/require"
)

func TestNew_ParsesPatterns(t *testing.T) {
	h := NewHyphenator("hy1p\n.al3t\n")
	require.NotNil(t, h, "expected non-nil Hyphenator")
	if h.trie == nil {
		t.Fatal("expected non-nil trie root")
	}
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
		if points != nil {
			t.Errorf("Hyphenate(%q) = %v, want nil", word, points)
		}
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
		if len(points) < tt.wantMin {
			t.Errorf("Hyphenate(%q) returned %d points %v, want at least %d",
				tt.word, len(points), points, tt.wantMin)
		}
	}
}

func TestHyphenate_UpperCase(t *testing.T) {
	h := DefaultRegistry().Get("en-us")
	lower := h.Hyphenate("hyphenation")
	upper := h.Hyphenate("HYPHENATION")
	mixed := h.Hyphenate("Hyphenation")

	if len(lower) == 0 {
		t.Fatal("expected break points for 'hyphenation'")
	}
	if len(upper) != len(lower) {
		t.Errorf("uppercase gave %d points, lowercase gave %d", len(upper), len(lower))
	}
	if len(mixed) != len(lower) {
		t.Errorf("mixed case gave %d points, lowercase gave %d", len(mixed), len(lower))
	}
}

func TestHyphenate_BreakPointBounds(t *testing.T) {
	h := DefaultRegistry().Get("en-us")

	word := "international"
	points := h.Hyphenate(word)
	runes := []rune(word)
	for _, p := range points {
		if p < h.leftMin {
			t.Errorf("break point %d violates leftMin %d", p, h.leftMin)
		}
		if p > len(runes)-h.rightMin {
			t.Errorf("break point %d violates rightMin %d (word len %d)", p, h.rightMin, len(runes))
		}
	}
}

func TestInsertSoftHyphens(t *testing.T) {
	h := DefaultRegistry().Get("en-us")
	result := h.InsertSoftHyphens("hyphenation")
	if !strings.Contains(result, "\u00AD") {
		t.Errorf("InsertSoftHyphens('hyphenation') = %q, expected soft hyphens", result)
	}

	cleaned := strings.ReplaceAll(result, "\u00AD", "")
	if cleaned != "hyphenation" {
		t.Errorf("after removing soft hyphens got %q, want 'hyphenation'", cleaned)
	}
}

func TestInsertSoftHyphens_NoBreaks(t *testing.T) {
	h := DefaultRegistry().Get("en-us")

	result := h.InsertSoftHyphens("cat")
	if result != "cat" {
		t.Errorf("InsertSoftHyphens('cat') = %q, want 'cat'", result)
	}
}

func TestInsertSoftHyphens_EmptyString(t *testing.T) {
	h := DefaultRegistry().Get("en-us")
	result := h.InsertSoftHyphens("")
	if result != "" {
		t.Errorf("InsertSoftHyphens('') = %q, want ''", result)
	}
}

func TestInsertSoftHyphens_PreservesOriginalCase(t *testing.T) {
	h := DefaultRegistry().Get("en-us")
	result := h.InsertSoftHyphens("Hyphenation")
	cleaned := strings.ReplaceAll(result, "\u00AD", "")
	if cleaned != "Hyphenation" {
		t.Errorf("case not preserved: got %q, want 'Hyphenation'", cleaned)
	}
}

func TestRegistry_Get_DefaultsToEnUS(t *testing.T) {
	r := DefaultRegistry()
	h1 := r.Get("")
	h2 := r.Get("en-us")
	if h1 != h2 {
		t.Error("empty language should return same hyphenator as en-us")
	}
}

func TestRegistry_Get_NormalisesVariants(t *testing.T) {
	r := DefaultRegistry()
	enUS := r.Get("en-us")
	for _, lang := range []string{"en", "en-gb", "EN-US", "En"} {
		h := r.Get(lang)
		if h != enUS {
			t.Errorf("Get(%q) returned different hyphenator than en-us", lang)
		}
	}
}

func TestRegistry_Get_UnsupportedFallsBack(t *testing.T) {
	r := DefaultRegistry()
	enUS := r.Get("en-us")
	h := r.Get("xx-unknown")
	if h != enUS {
		t.Error("unsupported language should fall back to en-us")
	}
}

func TestHyphenate_SimplePattern(t *testing.T) {

	h := NewHyphenator("ab1c")

	points := h.Hyphenate("xabcy")
	if points != nil {
		t.Errorf("Hyphenate('xabcy') = %v, want nil (rightMin violation)", points)
	}

	points = h.Hyphenate("xabcyz")
	if len(points) != 1 || points[0] != 3 {
		t.Errorf("Hyphenate('xabcyz') with pattern 'ab1c' = %v, want [3]", points)
	}
}
