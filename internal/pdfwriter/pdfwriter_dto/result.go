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

package pdfwriter_dto

import "piko.sh/piko/internal/layouter/layouter_dto"

// PdfResult holds the output of a PDF rendering operation.
type PdfResult struct {
	// LayoutDump is an optional human-readable serialisation of the layout
	// box tree, useful for golden file debugging in integration tests.
	//
	// Empty when not requested.
	LayoutDump string

	// Content holds the rendered PDF bytes.
	Content []byte

	// PageCount is the number of pages in the generated PDF.
	PageCount int
}

// PdfConfig holds configuration for a PDF rendering operation.
type PdfConfig struct {
	// Stylesheets contains additional CSS strings to apply after the
	// user-agent stylesheet and before inline styles.
	Stylesheets []string

	// Page defines the page dimensions and margins in points.
	Page layouter_dto.PageConfig

	// DefaultFontSize is the root font size in points. Defaults to 12.0
	// if zero.
	DefaultFontSize float64

	// DefaultLineHeight is the unitless line-height multiplier. Defaults
	// to 1.2 if zero.
	DefaultLineHeight float64
}
