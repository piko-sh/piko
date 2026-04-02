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
	"bytes"
	"testing"
)

const testHTMLPDF = `<!DOCTYPE html>
<html>
<head><title>PDF Test</title></head>
<body>
<h1>PDF Test Page</h1>
<p>This is a test page for PDF generation.</p>
<p>It contains multiple paragraphs.</p>
<p>And some more content for testing.</p>
</body>
</html>`

func TestPrintToPDF(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	server := newTestServer(testHTMLPDF)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("generates PDF with default options", func(t *testing.T) {
			data, err := PrintToPDF(ctx)
			if err != nil {
				t.Fatalf("PrintToPDF() error = %v", err)
			}
			if len(data) == 0 {
				t.Error("PrintToPDF() returned empty data")
			}

			if !bytes.HasPrefix(data, []byte("%PDF")) {
				t.Error("PrintToPDF() did not return valid PDF data")
			}
		})

		t.Run("generates landscape PDF", func(t *testing.T) {
			data, err := PrintToPDFLandscape(ctx)
			if err != nil {
				t.Fatalf("PrintToPDFLandscape() error = %v", err)
			}
			if len(data) == 0 {
				t.Error("PrintToPDFLandscape() returned empty data")
			}
			if !bytes.HasPrefix(data, []byte("%PDF")) {
				t.Error("PrintToPDFLandscape() did not return valid PDF data")
			}
		})

		t.Run("generates A4 PDF", func(t *testing.T) {
			data, err := PrintToPDFA4(ctx)
			if err != nil {
				t.Fatalf("PrintToPDFA4() error = %v", err)
			}
			if len(data) == 0 {
				t.Error("PrintToPDFA4() returned empty data")
			}
			if !bytes.HasPrefix(data, []byte("%PDF")) {
				t.Error("PrintToPDFA4() did not return valid PDF data")
			}
		})

		t.Run("generates PDF without background", func(t *testing.T) {
			data, err := PrintToPDFNoBackground(ctx)
			if err != nil {
				t.Fatalf("PrintToPDFNoBackground() error = %v", err)
			}
			if len(data) == 0 {
				t.Error("PrintToPDFNoBackground() returned empty data")
			}
			if !bytes.HasPrefix(data, []byte("%PDF")) {
				t.Error("PrintToPDFNoBackground() did not return valid PDF data")
			}
		})

		t.Run("generates PDF with custom options", func(t *testing.T) {
			opts := DefaultPDFOptions()
			opts.Scale = 0.8
			opts.MarginTop = 1.0
			opts.MarginBottom = 1.0

			data, err := PrintToPDFWithOptions(ctx, opts)
			if err != nil {
				t.Fatalf("PrintToPDFWithOptions() error = %v", err)
			}
			if len(data) == 0 {
				t.Error("PrintToPDFWithOptions() returned empty data")
			}
			if !bytes.HasPrefix(data, []byte("%PDF")) {
				t.Error("PrintToPDFWithOptions() did not return valid PDF data")
			}
		})

		t.Run("generates PDF with header and footer", func(t *testing.T) {
			header := `<div style="font-size:10px;text-align:center;">Header Text</div>`
			footer := `<div style="font-size:10px;text-align:center;">Page <span class="pageNumber"></span></div>`

			data, err := PrintToPDFWithHeaderFooter(ctx, header, footer)
			if err != nil {
				t.Fatalf("PrintToPDFWithHeaderFooter() error = %v", err)
			}
			if len(data) == 0 {
				t.Error("PrintToPDFWithHeaderFooter() returned empty data")
			}
			if !bytes.HasPrefix(data, []byte("%PDF")) {
				t.Error("PrintToPDFWithHeaderFooter() did not return valid PDF data")
			}
		})

		t.Run("generates PDF with page range", func(t *testing.T) {
			data, err := PrintToPDFPageRange(ctx, "1")
			if err != nil {
				t.Fatalf("PrintToPDFPageRange() error = %v", err)
			}
			if len(data) == 0 {
				t.Error("PrintToPDFPageRange() returned empty data")
			}
			if !bytes.HasPrefix(data, []byte("%PDF")) {
				t.Error("PrintToPDFPageRange() did not return valid PDF data")
			}
		})
	})
}
