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

//go:build !integration

package layouter_domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type stubFontMetrics struct{}

func (s *stubFontMetrics) MeasureText(_ FontDescriptor, _ float64, _ string, _ DirectionType) float64 {
	return 0
}

func (s *stubFontMetrics) ShapeText(_ FontDescriptor, _ float64, text string, _ DirectionType) []GlyphPosition {
	glyphs := make([]GlyphPosition, len([]rune(text)))
	for i := range glyphs {
		glyphs[i] = GlyphPosition{GlyphID: uint16(i), XAdvance: 6.0}
	}
	return glyphs
}

func (s *stubFontMetrics) GetMetrics(_ FontDescriptor, _ float64) FontMetrics {
	return FontMetrics{}
}

func (s *stubFontMetrics) ResolveFallback(_ FontDescriptor, _ rune) FontDescriptor {
	return FontDescriptor{}
}

func (s *stubFontMetrics) SplitGraphemeClusters(text string) []string {
	clusters := make([]string, 0, len(text))
	for _, r := range text {
		clusters = append(clusters, string(r))
	}
	return clusters
}

func TestSubstitutePageNumbers_ReplacesPagePlaceholder(t *testing.T) {
	box := &LayoutBox{
		Text:      "Page {page}",
		PageIndex: 2,
		Style: ComputedStyle{
			FontFamily: "NotoSans",
			FontWeight: 400,
			FontSize:   12,
		},
	}
	root := &LayoutBox{Children: []*LayoutBox{box}}

	SubstitutePageNumbers(root, 5, &stubFontMetrics{})

	assert.Equal(t, "Page 3", box.Text)
	assert.Len(t, box.Glyphs, 6)
}

func TestSubstitutePageNumbers_ReplacesPagesPlaceholder(t *testing.T) {
	box := &LayoutBox{
		Text:      "{page} of {pages}",
		PageIndex: 0,
		Style: ComputedStyle{
			FontFamily: "NotoSans",
			FontWeight: 400,
			FontSize:   12,
		},
	}
	root := &LayoutBox{Children: []*LayoutBox{box}}

	SubstitutePageNumbers(root, 10, &stubFontMetrics{})

	assert.Equal(t, "1 of 10", box.Text)
}

func TestSubstitutePageNumbers_NoPlaceholderUnchanged(t *testing.T) {
	box := &LayoutBox{
		Text:      "Hello World",
		PageIndex: 0,
		Glyphs:    []GlyphPosition{{GlyphID: 1}},
	}
	root := &LayoutBox{Children: []*LayoutBox{box}}

	SubstitutePageNumbers(root, 3, &stubFontMetrics{})

	assert.Equal(t, "Hello World", box.Text, "expected unchanged text")
	assert.Len(t, box.Glyphs, 1, "expected original glyphs")
}

func TestSubstitutePageNumbers_NestedBoxes(t *testing.T) {
	inner := &LayoutBox{
		Text:      "Page {page}",
		PageIndex: 4,
		Style:     ComputedStyle{FontSize: 12},
	}
	root := &LayoutBox{
		Children: []*LayoutBox{
			{Children: []*LayoutBox{inner}},
		},
	}

	SubstitutePageNumbers(root, 8, &stubFontMetrics{})

	assert.Equal(t, "Page 5", inner.Text)
}
