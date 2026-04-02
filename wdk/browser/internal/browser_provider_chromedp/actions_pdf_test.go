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

package browser_provider_chromedp

import (
	"math"
	"testing"
)

func TestDefaultPDFOptions(t *testing.T) {
	opts := DefaultPDFOptions()

	if opts.Scale != 1.0 {
		t.Errorf("Scale = %f, expected 1.0", opts.Scale)
	}
	if opts.PaperWidth != PDFPaperWidthLetter {
		t.Errorf("PaperWidth = %v, expected %v", opts.PaperWidth, PDFPaperWidthLetter)
	}
	if opts.PaperHeight != PDFPaperHeightLetter {
		t.Errorf("PaperHeight = %v, expected %v", opts.PaperHeight, PDFPaperHeightLetter)
	}
	if opts.MarginTop != PDFMarginDefault {
		t.Errorf("MarginTop = %v, expected %v", opts.MarginTop, PDFMarginDefault)
	}
	if opts.MarginBottom != PDFMarginDefault {
		t.Errorf("MarginBottom = %v, expected %v", opts.MarginBottom, PDFMarginDefault)
	}
	if opts.MarginLeft != PDFMarginDefault {
		t.Errorf("MarginLeft = %v, expected %v", opts.MarginLeft, PDFMarginDefault)
	}
	if opts.MarginRight != PDFMarginDefault {
		t.Errorf("MarginRight = %v, expected %v", opts.MarginRight, PDFMarginDefault)
	}
	if opts.Landscape {
		t.Error("Landscape should be false")
	}
	if !opts.PrintBackground {
		t.Error("PrintBackground should be true")
	}
	if opts.PreferCSSPageSize {
		t.Error("PreferCSSPageSize should be false")
	}
	if opts.PageRanges != "" {
		t.Errorf("PageRanges = %q, expected empty string", opts.PageRanges)
	}
	if opts.HeaderTemplate != "" {
		t.Errorf("HeaderTemplate = %q, expected empty string", opts.HeaderTemplate)
	}
	if opts.FooterTemplate != "" {
		t.Errorf("FooterTemplate = %q, expected empty string", opts.FooterTemplate)
	}
}

func TestA4PDFOptions(t *testing.T) {
	opts := A4PDFOptions()

	if math.Abs(opts.PaperWidth-PDFPaperWidthA4) > 0.01 {
		t.Errorf("PaperWidth = %f, expected %f", opts.PaperWidth, PDFPaperWidthA4)
	}
	if math.Abs(opts.PaperHeight-PDFPaperHeightA4) > 0.01 {
		t.Errorf("PaperHeight = %f, expected %f", opts.PaperHeight, PDFPaperHeightA4)
	}

	if opts.Scale != 1.0 {
		t.Errorf("Scale = %f, expected 1.0", opts.Scale)
	}
	if opts.MarginTop != PDFMarginDefault {
		t.Errorf("MarginTop = %f, expected %f", opts.MarginTop, PDFMarginDefault)
	}
	if !opts.PrintBackground {
		t.Error("PrintBackground should be true")
	}
}
