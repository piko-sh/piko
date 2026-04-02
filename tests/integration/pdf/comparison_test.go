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
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var updateGolden = flag.Bool("update", false, "update golden files")

func runPdfTestCase(t *testing.T, tc testCase) {
	t.Helper()

	fmt.Printf("\n--- [PDF] START: %s ---\n", tc.Name)

	harness := newPdfHarness(t, tc)
	defer harness.cleanup()

	fmt.Println("[PDF] Phase 1: Setting up test environment...")
	require.NoError(t, harness.setup(), "failed to setup harness")

	fmt.Println("[PDF] Phase 2: Building binary...")
	require.NoError(t, harness.buildBinary(), "failed to build binary")

	fmt.Println("[PDF] Phase 3: Rendering PDF via pdfwriter...")
	renderResult, err := harness.renderPdf()
	require.NoError(t, err, "failed to render PDF")
	require.NotEmpty(t, renderResult.pdfBytes, "rendered PDF is empty")
	actualPdf := renderResult.pdfBytes

	if harness.spec.Encryption != nil {
		fmt.Println("[PDF] Phase 4: Validating encryption with qpdf...")
		encryptedPath := filepath.Join(tc.Path, "golden", "encrypted.pdf")
		require.NoError(t, os.MkdirAll(filepath.Dir(encryptedPath), 0755))
		require.NoError(t, os.WriteFile(encryptedPath, actualPdf, 0644),
			"failed to write encrypted PDF")

		qpdfPath, lookErr := exec.LookPath("qpdf")
		if lookErr != nil {
			t.Skip("qpdf not installed, skipping encryption validation")
		}
		_ = qpdfPath

		decryptedPath := filepath.Join(tc.Path, "golden", "decrypted.pdf")
		decryptCmd := exec.Command("qpdf",
			"--password="+harness.spec.Encryption.UserPassword,
			"--decrypt", encryptedPath, decryptedPath)
		decryptOut, decryptErr := decryptCmd.CombinedOutput()
		require.NoError(t, decryptErr,
			"qpdf failed to decrypt with user password: %s", string(decryptOut))

		decrypted, readErr := os.ReadFile(decryptedPath)
		require.NoError(t, readErr)
		require.True(t, bytes.HasPrefix(decrypted, []byte("%PDF-")),
			"decrypted output should be a valid PDF")

		t.Logf("encryption validated: %d bytes encrypted, %d bytes decrypted",
			len(actualPdf), len(decrypted))
		fmt.Printf("--- [PDF] COMPLETE (encryption validated): %s ---\n", tc.Name)
		return
	}

	fmt.Println("[PDF] Phase 4: Starting server...")
	require.NoError(t, harness.startServer(), "failed to start server")

	fmt.Println("[PDF] Phase 5: Setting up browser...")
	require.NoError(t, harness.setupBrowser(), "failed to setup browser")

	fmt.Println("[PDF] Phase 6: Navigating to page...")
	require.NoError(t, harness.navigateToPage(), "failed to navigate to page")

	time.Sleep(500 * time.Millisecond)

	fmt.Println("[PDF] Phase 7: Printing browser comparison PDF...")
	comparisonPdf, err := harness.printBrowserPdf()
	require.NoError(t, err, "failed to print browser PDF")
	require.NotEmpty(t, comparisonPdf, "browser comparison PDF is empty")

	goldenDir := filepath.Join(tc.Path, "golden")
	goldenPath := filepath.Join(goldenDir, "golden.pdf")
	comparisonPath := filepath.Join(goldenDir, "comparison.pdf")

	if *updateGolden {
		fmt.Println("[PDF] Phase 8: Updating golden files...")
		require.NoError(t, os.MkdirAll(goldenDir, 0755), "failed to create golden directory")
		require.NoError(t, os.WriteFile(goldenPath, actualPdf, 0644), "failed to write golden file")
		require.NoError(t, os.WriteFile(comparisonPath, stripPdfTimestamps(comparisonPdf), 0644), "failed to write comparison file")
		if renderResult.layoutDump != "" {
			layoutGoldenPath := filepath.Join(goldenDir, "golden.go")
			require.NoError(t, os.WriteFile(layoutGoldenPath, []byte(renderResult.layoutDump), 0644), "failed to write layout golden")
			t.Logf("updated layout golden: %s (%d bytes)", layoutGoldenPath, len(renderResult.layoutDump))
		}
		t.Logf("updated golden file: %s (%d bytes)", goldenPath, len(actualPdf))
		t.Logf("updated comparison file: %s (%d bytes)", comparisonPath, len(comparisonPdf))
		fmt.Printf("--- [PDF] COMPLETE (updated golden): %s ---\n", tc.Name)
		return
	}

	fmt.Println("[PDF] Phase 8: Comparing against golden file...")
	expectedPdf, err := os.ReadFile(goldenPath)
	require.NoError(t, err, "failed to read golden file at %s (run with -update to generate)", goldenPath)

	if !bytes.Equal(actualPdf, expectedPdf) {
		firstDifference := findFirstDifference(actualPdf, expectedPdf)
		t.Errorf("PDF output differs from golden file\n"+
			"  actual size:   %d bytes\n"+
			"  expected size: %d bytes\n"+
			"  first diff at: byte %d\n"+
			"  golden file:   %s\n"+
			"  run with -update to regenerate",
			len(actualPdf), len(expectedPdf), firstDifference, goldenPath)
	}

	fmt.Printf("--- [PDF] COMPLETE: %s ---\n", tc.Name)
}

var pdfTimestampPattern = regexp.MustCompile(`/(CreationDate|ModDate) \(D:\d{14}[^)]*\)`)

func stripPdfTimestamps(data []byte) []byte {
	return pdfTimestampPattern.ReplaceAll(data, []byte("/$1 (D:19700101000000+00'00')"))
}

func findFirstDifference(actual, expected []byte) int {
	limit := min(len(actual), len(expected))
	for i := range limit {
		if actual[i] != expected[i] {
			return i
		}
	}
	return limit
}
