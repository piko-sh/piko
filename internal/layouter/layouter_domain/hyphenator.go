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

package layouter_domain

import (
	_ "embed"
	"iter"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
)

const (
	// hyphenLeftMin is the minimum number of characters before the
	// first allowed break point.
	hyphenLeftMin = 2

	// hyphenRightMin is the minimum number of characters after the
	// last allowed break point.
	hyphenRightMin = 3

	// langEnUS is the default language tag for hyphenation.
	langEnUS = "en-us"
)

// enUSPatterns holds the embedded TeX hyphenation patterns for American English.
//
//go:embed patterns/hyph-en-us.pat.txt
var enUSPatterns string

// trieNode is a node in the pattern trie. Each node may have
// children keyed by rune and an optional values slice holding
// the interleaved digit levels from the pattern.
type trieNode struct {
	// children holds the child nodes keyed by rune.
	children map[rune]*trieNode

	// values holds the interleaved digit levels from the pattern.
	values []int
}

// newTrieNode creates an empty trie node with an initialised children map.
//
// Returns *trieNode which is the newly allocated node.
func newTrieNode() *trieNode {
	return &trieNode{children: make(map[rune]*trieNode)}
}

// Hyphenator holds a compiled trie of hyphenation patterns for a
// single language. It is safe for concurrent use after construction.
//
// The algorithm works by building a trie from TeX pattern strings
// during construction. Each pattern encodes interleaved letters and
// digit values that indicate hyphenation priority at each position.
// During lookup, the word is augmented with boundary markers (.),
// all matching patterns are overlaid (taking the maximum digit at
// each position), and odd-valued positions indicate valid break
// points. Breaks within leftMin characters from the start or
// rightMin characters from the end are suppressed.
type Hyphenator struct {
	// trie holds the root of the compiled pattern trie.
	trie *trieNode

	// leftMin holds the minimum number of characters before the first break point.
	leftMin int

	// rightMin holds the minimum number of characters after the last break point.
	rightMin int
}

// NewHyphenator creates a Hyphenator from a TeX-format pattern string.
//
// The pattern string contains one pattern per line. Comment lines
// beginning with '%' and empty lines are skipped.
//
// Takes patterns (string) which is the TeX-format pattern data.
//
// Returns *Hyphenator which is ready for concurrent use.
func NewHyphenator(patterns string) *Hyphenator {
	root := newTrieNode()
	buildTrie(root, strings.SplitSeq(patterns, "\n"))
	return &Hyphenator{
		trie:     root,
		leftMin:  hyphenLeftMin,
		rightMin: hyphenRightMin,
	}
}

// Hyphenate returns the rune indices of valid hyphenation points
// in the given word.
//
// Each index indicates a position between runes where a hyphen may
// be inserted. For example, for "hyp-hen" where the break is between
// index 3 and 4, the returned index would be 3.
//
// Takes word (string) which is the word to hyphenate.
//
// Returns []int which holds the rune indices of break points, or nil
// if the word is too short or contains no valid break points.
func (h *Hyphenator) Hyphenate(word string) []int {
	runes := []rune(strings.ToLower(word))
	if len(runes) < h.leftMin+h.rightMin {
		return nil
	}

	augmented := augmentWord(runes)
	levels := make([]int, len(augmented)+1)
	h.walkTrie(augmented, levels)
	return collectBreakPoints(levels, len(runes), h.leftMin, h.rightMin)
}

// InsertSoftHyphens returns the word with Unicode soft hyphen
// characters (U+00AD) inserted at all valid hyphenation points.
//
// Takes word (string) which is the word to insert soft hyphens into.
//
// Returns string which is the word with soft hyphens, or the original
// word unchanged if no hyphenation points are found.
func (h *Hyphenator) InsertSoftHyphens(word string) string {
	points := h.Hyphenate(word)
	if len(points) == 0 {
		return word
	}

	runes := []rune(word)
	var b strings.Builder
	b.Grow(len(word) + len(points)*utf8.RuneLen('\u00AD'))

	pointSet := make(map[int]bool, len(points))
	for _, p := range points {
		pointSet[p] = true
	}

	for i, r := range runes {
		if pointSet[i] {
			_, _ = b.WriteRune('\u00AD')
		}
		_, _ = b.WriteRune(r)
	}
	return b.String()
}

// walkTrie walks the pattern trie from every starting position in
// augmented, overlaying the maximum digit value at each position
// into levels.
//
// Takes augmented ([]rune) which is the boundary-marked word.
// Takes levels ([]int) which receives the overlaid digit values.
func (h *Hyphenator) walkTrie(augmented []rune, levels []int) {
	for i := range augmented {
		node := h.trie
		for j := i; j < len(augmented); j++ {
			child, ok := node.children[augmented[j]]
			if !ok {
				break
			}
			node = child
			overlayValues(node.values, levels, i)
		}
	}
}

// buildTrie inserts every non-empty, non-comment line into root.
//
// Takes root (*trieNode) which is the trie root to populate.
// Takes lines (iter.Seq[string]) which yields the pattern lines.
func buildTrie(root *trieNode, lines iter.Seq[string]) {
	for line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || line[0] == '%' {
			continue
		}
		insertPattern(root, line)
	}
}

// insertPattern parses a single TeX pattern and inserts it into
// the trie.
//
// A pattern like "hy1p" means the letters are "hyp" and the values
// at positions [0,1,2,3] are [0,0,1,0]. The dot (.) represents
// a word boundary.
//
// Takes root (*trieNode) which is the trie root to insert into.
// Takes pattern (string) which is the TeX pattern string.
func insertPattern(root *trieNode, pattern string) {
	var letters []rune
	var values []int

	for _, r := range pattern {
		if r >= '0' && r <= '9' {
			for len(values) < len(letters) {
				values = append(values, 0)
			}
			values = append(values, int(r-'0'))
		} else {
			for len(values) < len(letters) {
				values = append(values, 0)
			}
			letters = append(letters, unicode.ToLower(r))
		}
	}
	for len(values) <= len(letters) {
		values = append(values, 0)
	}

	node := root
	for _, letter := range letters {
		child, ok := node.children[letter]
		if !ok {
			child = newTrieNode()
			node.children[letter] = child
		}
		node = child
	}
	node.values = values
}

// augmentWord wraps runes with boundary markers for trie lookup.
//
// Takes runes ([]rune) which is the word as a rune slice.
//
// Returns []rune which is the word with leading and trailing dot markers.
func augmentWord(runes []rune) []rune {
	augmented := make([]rune, 0, len(runes)+2)
	augmented = append(augmented, '.')
	augmented = append(augmented, runes...)
	return append(augmented, '.')
}

// overlayValues merges pattern values into levels starting at
// offset, keeping the maximum at each position.
//
// Takes values ([]int) which holds the pattern digit values.
// Takes levels ([]int) which receives the maximum values.
// Takes offset (int) which is the starting position in levels.
func overlayValues(values, levels []int, offset int) {
	for k, v := range values {
		pos := offset + k
		if pos < len(levels) && v > levels[pos] {
			levels[pos] = v
		}
	}
}

// collectBreakPoints returns rune indices where odd-valued levels
// indicate valid hyphenation points, respecting margin constraints.
//
// Takes levels ([]int) which holds the computed digit levels.
// Takes wordLen (int) which is the number of runes in the word.
// Takes leftMin (int) which is the minimum characters before the first break.
// Takes rightMin (int) which is the minimum characters after the last break.
//
// Returns []int which holds the rune indices of valid break points.
func collectBreakPoints(levels []int, wordLen, leftMin, rightMin int) []int {
	var points []int
	for i := leftMin; i <= wordLen-rightMin; i++ {
		if levels[i+1]%2 == 1 {
			points = append(points, i)
		}
	}
	return points
}

// HyphenationRegistry holds lazily-initialised hyphenators keyed
// by language tag.
type HyphenationRegistry struct {
	// hyphenators holds the cached hyphenators keyed by normalised language tag.
	hyphenators map[string]*Hyphenator

	// mu guards concurrent access to the hyphenators map.
	mu sync.Mutex
}

var defaultHyphenationRegistry = &HyphenationRegistry{
	hyphenators: make(map[string]*Hyphenator),
}

// DefaultRegistry returns the package-level registry of
// hyphenators, lazily initialising them on first use.
//
// Returns *HyphenationRegistry which is the shared registry instance.
func DefaultRegistry() *HyphenationRegistry {
	return defaultHyphenationRegistry
}

// Get returns the Hyphenator for the given language tag,
// constructing it on first access.
//
// Falls back to "en-us" for unsupported languages. The language
// tag is normalised to lowercase.
//
// Takes lang (string) which is the language tag to look up.
//
// Returns *Hyphenator which is the hyphenator for the language.
//
// Safe for concurrent use by multiple goroutines.
func (r *HyphenationRegistry) Get(lang string) *Hyphenator {
	lang = normaliseLang(lang)

	r.mu.Lock()
	defer r.mu.Unlock()

	if h, ok := r.hyphenators[lang]; ok {
		return h
	}

	patterns := patternsForLang(lang)
	if patterns == "" {
		lang = langEnUS
		if h, ok := r.hyphenators[lang]; ok {
			return h
		}
		patterns = enUSPatterns
	}

	h := NewHyphenator(patterns)
	r.hyphenators[lang] = h
	return h
}

// normaliseLang lowercases and maps English variants to en-us.
//
// Takes lang (string) which is the raw language tag.
//
// Returns string which is the normalised language tag.
func normaliseLang(lang string) string {
	lang = strings.ToLower(strings.TrimSpace(lang))
	if lang == "" {
		return langEnUS
	}
	switch lang {
	case "en", "en-gb", "en-au", "en-ca", "en-nz":
		return langEnUS
	}
	return lang
}

// patternsForLang returns the embedded pattern data for a
// language tag, or empty string if unsupported.
//
// Takes lang (string) which is the normalised language tag.
//
// Returns string which holds the pattern data, or empty if unsupported.
func patternsForLang(lang string) string {
	if lang == langEnUS {
		return enUSPatterns
	}
	return ""
}
