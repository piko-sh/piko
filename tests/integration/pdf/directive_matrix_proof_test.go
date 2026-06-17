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

//go:build integration

package pdf_test

import (
	"image"
	_ "image/png"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	matrixGoldenDir = "testdata/038_partial_directive_matrix/golden"
)

func TestPartialDirectiveMatrix_PdfMatchesBrowser(t *testing.T) {
	pikoPDF := filepath.Join(matrixGoldenDir, "golden.pdf")
	browserPDF := filepath.Join(matrixGoldenDir, "comparison.pdf")

	if _, err := os.Stat(pikoPDF); err != nil {
		t.Skipf("golden.pdf not present, run the case with -update first: %v", err)
	}
	if _, err := os.Stat(browserPDF); err != nil {
		t.Skipf("comparison.pdf not present, run the case with -update first: %v", err)
	}

	pikoText := pdfToText(t, pikoPDF)
	browserText := pdfToText(t, browserPDF)

	textChecks := []struct {
		scenario string
		want     string
	}{
		{"p-html, single element (block C)", "bold and italic spans"},
		{"p-html, inside a loop (blocks D & E)", "Alpha — bold emphasis"},
		{"p-html, directly in the PDF template (block F)", "no partial involved"},
		{"flex nested-inline content on one line (block I)", "Experience"},
	}
	for _, c := range textChecks {

		require.Contains(t, browserText, c.want, "test self-check failed: browser PDF is missing %q, fix the expectation", c.want)
		assert.Contains(t, pikoText, c.want, "REGRESSION [%s]: Piko PDF is missing %q which the browser renders", c.scenario, c.want)
	}

	t.Run("svg painted", func(t *testing.T) {
		if _, err := exec.LookPath("pdftoppm"); err != nil {
			t.Skip("pdftoppm (poppler-utils) not installed")
		}
		browserHasBlob := hasGreenBlob(t, browserPDF)
		pikoHasBlob := hasGreenBlob(t, pikoPDF)
		require.True(t, browserHasBlob, "test self-check failed: browser PDF has no solid green shape, fix the expectation")
		assert.True(t, pikoHasBlob, "REGRESSION [inline <svg> and <piko:svg> must paint (blocks K & L)]: "+
			"the browser paints solid green dots; Piko's PDF has none")
	})
}

func pdfToText(t *testing.T, path string) string {
	t.Helper()
	bin, err := exec.LookPath("pdftotext")
	if err != nil {
		t.Skip("pdftotext (poppler-utils) not installed")
	}
	out, err := exec.Command(bin, "-layout", path, "-").Output()
	require.NoError(t, err, "pdftotext %s", path)
	return string(out)
}

func hasGreenBlob(t *testing.T, pdfPath string) bool {
	t.Helper()
	dir := t.TempDir()
	prefix := filepath.Join(dir, "page")
	out, err := exec.Command("pdftoppm", "-png", "-r", "150", pdfPath, prefix).CombinedOutput()
	require.NoError(t, err, "pdftoppm %s\n%s", pdfPath, out)
	pages, _ := filepath.Glob(prefix + "*.png")
	for _, p := range pages {
		f, err := os.Open(p)
		require.NoError(t, err)
		img, _, err := image.Decode(f)
		f.Close()
		require.NoError(t, err)
		b := img.Bounds()
		for y := b.Min.Y; y < b.Max.Y; y++ {
			run := 0
			for x := b.Min.X; x < b.Max.X; x++ {
				r, g, bl, _ := img.At(x, y).RGBA()
				if r>>8 < 90 && g>>8 > 140 && bl>>8 < 110 {
					run++
					if run >= 20 {
						return true
					}
				} else {
					run = 0
				}
			}
		}
	}
	return false
}
