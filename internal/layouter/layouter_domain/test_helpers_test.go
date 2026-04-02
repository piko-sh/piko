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

type mockFontMetrics struct{}

func (m *mockFontMetrics) MeasureText(font FontDescriptor, size float64, text string, _ DirectionType) float64 {
	return float64(len([]rune(text))) * size * 0.5
}

func (m *mockFontMetrics) ShapeText(font FontDescriptor, size float64, text string, _ DirectionType) []GlyphPosition {
	positions := make([]GlyphPosition, 0, len(text))
	advance := size * 0.5
	for range text {
		positions = append(positions, GlyphPosition{XAdvance: advance})
	}
	return positions
}

func (m *mockFontMetrics) GetMetrics(font FontDescriptor, size float64) FontMetrics {
	return FontMetrics{
		Ascent:     size * 0.8,
		Descent:    size * 0.2,
		LineGap:    size * 0.1,
		CapHeight:  size * 0.7,
		XHeight:    size * 0.5,
		UnitsPerEm: 1000,
	}
}

func (m *mockFontMetrics) ResolveFallback(font FontDescriptor, character rune) FontDescriptor {
	return font
}

func (m *mockFontMetrics) SplitGraphemeClusters(text string) []string {
	clusters := make([]string, 0, len(text))
	for _, r := range text {
		clusters = append(clusters, string(r))
	}
	return clusters
}

func defaultResolutionContext() ResolutionContext {
	return ResolutionContext{
		ParentFontSize:       12,
		RootFontSize:         16,
		ContainingBlockWidth: 595.28,
		ViewportWidth:        595.28,
		ViewportHeight:       841.89,
	}
}
