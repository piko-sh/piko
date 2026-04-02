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
	"strings"
	"sync"

	qt "github.com/valyala/quicktemplate"
	"piko.sh/piko/internal/ast/ast_domain"
)

const (
	// tagPikoA is the attribute name that marks links as processed.
	tagPikoA = "piko:a"

	// tagPikoSvg is the attribute name for the piko:svg custom element tag.
	tagPikoSvg = "piko:svg"

	// attributeSrc is the name of the HTML src attribute.
	attributeSrc = "src"

	// attributeSrcset is the name of the HTML srcset attribute.
	attributeSrcset = "srcset"

	// attributeClass is the HTML class attribute name for case-insensitive matching.
	attributeClass = "class"

	// initialThemeCSSCapacity is the initial capacity for the CSS theme builder,
	// set based on typical theme sizes.
	initialThemeCSSCapacity = 3072

	// standardSortedKeysSize is the buffer size for pooled key slices, set to
	// handle the typical number of attributes found in SVG elements.
	standardSortedKeysSize = 16

	// sortThresholdManual is the largest array size that uses manual sorting.
	sortThresholdManual = 3

	// sortThresholdInsertion is the maximum count for which insertion sort is used.
	sortThresholdInsertion = 5

	// maxFastPathClassStringLen is the maximum string length for fast-path class
	// deduplication optimisations.
	maxFastPathClassStringLen = 32

	// maxFastPathAttrCount is the maximum number of attributes that can use the
	// stack-allocated fast path for hashing.
	maxFastPathAttrCount = 8
)

// stringBuilderPool uses a sync.Pool to reuse strings.Builder objects,
// reducing allocation pressure in high-throughput rendering paths.
var stringBuilderPool = sync.Pool{
	New: func() any {
		b := &strings.Builder{}
		b.Grow(128)
		return b
	},
}

// writeErrorDiv writes an error message inside a div element with the given
// attributes. This matches the output of createErrorNode for consistent error
// display.
//
// Takes qw (*qt.Writer) which receives the HTML output.
// Takes userAttrs ([]ast_domain.HTMLAttribute) which provides attributes to
// add to the div element.
// Takes errorMessage (string) which contains the error text to display.
func writeErrorDiv(qw *qt.Writer, userAttrs []ast_domain.HTMLAttribute, errorMessage string) {
	qw.N().S("<div")

	for i := range userAttrs {
		attr := &userAttrs[i]
		qw.N().S(" ")
		qw.N().S(attr.Name)
		qw.N().S(`="`)
		qw.N().S(attr.Value)
		qw.N().S(`"`)
	}

	qw.N().S(">")
	qw.N().S(errorMessage)
	qw.N().S("</div>")
}

// toLowerIfNeeded converts a string to lowercase only if it contains
// uppercase letters. This avoids making a new string when the input is
// already lowercase.
//
// Takes s (string) which is the input string to check and convert.
//
// Returns string which is the lowercase version, or the original string if it
// is already lowercase.
func toLowerIfNeeded(s string) string {
	for i := range len(s) {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			return strings.ToLower(s)
		}
	}
	return s
}
