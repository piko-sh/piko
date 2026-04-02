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

package layouter_dto

// Defines configuration types for the layouter module including page geometry,
// font settings, and top-level layout configuration.

// LayoutConfig holds all configuration for a layout operation.
type LayoutConfig struct {
	// DefaultFontFamily is the font family used when no font-family is
	// specified in CSS. Defaults to "serif" if empty.
	DefaultFontFamily string

	// Stylesheets contains additional CSS strings to apply after the
	// user-agent stylesheet and before inline styles.
	Stylesheets []string

	// Page defines the page geometry (dimensions and margins).
	Page PageConfig

	// DefaultFontSize is the root font size in points, defaulting to
	// 12.0 if zero and serving as the basis for rem unit resolution.
	DefaultFontSize float64

	// DefaultLineHeight is the unitless line-height multiplier. Defaults
	// to 1.2 if zero.
	DefaultLineHeight float64
}

// PageConfig defines the physical dimensions and margins of a page in points.
type PageConfig struct {
	// Width is the total page width in points.
	Width float64

	// Height is the total page height in points. Ignored when
	// AutoHeight is true.
	Height float64

	// MarginTop is the top margin in points.
	MarginTop float64

	// MarginRight is the right margin in points.
	MarginRight float64

	// MarginBottom is the bottom margin in points.
	MarginBottom float64

	// MarginLeft is the left margin in points.
	MarginLeft float64

	// AutoHeight disables pagination and sizes the page height to
	// fit all content. Width is still required.
	AutoHeight bool
}

// ContentAreaWidth returns the available width inside the page margins.
//
// Returns float64 which is the width minus left and right margins.
func (p PageConfig) ContentAreaWidth() float64 {
	return p.Width - p.MarginLeft - p.MarginRight
}

// ContentAreaHeight returns the available height inside the page margins.
//
// Returns float64 which is the height minus top and bottom margins.
func (p PageConfig) ContentAreaHeight() float64 {
	return p.Height - p.MarginTop - p.MarginBottom
}

// FontConfig holds font loading configuration for the layout engine.
type FontConfig struct {
	// FontFiles lists font files to load for text rendering.
	FontFiles []FontFile

	// FallbackFamilies lists font families to try when the primary font
	// lacks a glyph.
	FallbackFamilies []string
}

// FontFile describes a single font file to load.
type FontFile struct {
	// Family is the CSS font-family name this file provides.
	Family string

	// Path is the filesystem path to the font file.
	Path string

	// Style is the font style ("normal" or "italic").
	Style string

	// Weight is the CSS font-weight value (100-900).
	Weight int
}
