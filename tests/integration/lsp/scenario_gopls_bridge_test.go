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

package lsp_stress_test

import (
	"context"
	"strings"
	"testing"
	"time"

	protocol "github.com/politepixels/golang-language-server"
	"github.com/stretchr/testify/require"
)

const (
	pollInterval = 250 * time.Millisecond
	negativeGoplsWait = 10 * time.Second
)

func bridgeArgs(goplsPath string) []string {
	return []string{"--gopls-bridge=true", "--gopls-path=" + goplsPath}
}

func newBridgeHarness(t *testing.T) *stressHarness {
	t.Helper()
	return newStressHarnessForFixture(t, "gopls_bridge_project", t.TempDir())
}

func testBridgeGoBlockHover(t *testing.T) {
	goplsPath := requireGopls(t)
	t.Parallel()

	harness := newBridgeHarness(t)
	client, cleanup := harness.startSessionWithOptions(sessionOptions{Args: bridgeArgs(goplsPath)})
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	indexURI := harness.fileURI("pages/index.pk")
	indexContent := harness.readFile("pages/index.pk")
	require.NoError(t, client.DidOpen(ctx, indexURI, indexContent))

	printlnPosition := findPosition(t, indexContent, "fmt.Println", "Println")

	require.Eventually(t, func() bool {
		hover, err := client.Hover(ctx, indexURI, printlnPosition)
		return err == nil && hoverContains(hover, "func fmt.Println")
	}, analysisTimeout, pollInterval, "gopls hover on fmt.Println never arrived")

	require.Empty(t, client.GetErrors())
}

func testBridgeGoBlockCompletion(t *testing.T) {
	goplsPath := requireGopls(t)
	t.Parallel()

	harness := newBridgeHarness(t)
	client, cleanup := harness.startSessionWithOptions(sessionOptions{Args: bridgeArgs(goplsPath)})
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	indexURI := harness.fileURI("pages/index.pk")
	indexContent := harness.readFile("pages/index.pk")
	require.NoError(t, client.DidOpen(ctx, indexURI, indexContent))

	memberPosition := findPosition(t, indexContent, "fmt.Println", "Println")

	require.Eventually(t, func() bool {
		list, err := client.Completion(ctx, indexURI, memberPosition)
		return err == nil && completionHasItem(list, "Println")
	}, analysisTimeout, pollInterval, "gopls completion for fmt members never arrived")

	require.Empty(t, client.GetErrors())
}

func testBridgeGoBlockDefinition(t *testing.T) {
	goplsPath := requireGopls(t)
	t.Parallel()

	harness := newBridgeHarness(t)
	client, cleanup := harness.startSessionWithOptions(sessionOptions{Args: bridgeArgs(goplsPath)})
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	indexURI := harness.fileURI("pages/index.pk")
	indexContent := harness.readFile("pages/index.pk")
	require.NoError(t, client.DidOpen(ctx, indexURI, indexContent))

	printlnPosition := findPosition(t, indexContent, "fmt.Println", "Println")

	var definitions []protocol.Location
	require.Eventually(t, func() bool {
		located, err := client.Definition(ctx, indexURI, printlnPosition)
		if err != nil || !locationInGoFile(located, "print.go") {
			return false
		}
		definitions = located
		return true
	}, analysisTimeout, pollInterval, "definition of fmt.Println never resolved to stdlib print.go")

	require.NotEmpty(t, definitions)
	require.Empty(t, client.GetErrors())
}

func testBridgeCrossComponentResolution(t *testing.T) {
	goplsPath := requireGopls(t)
	t.Parallel()

	harness := newBridgeHarness(t)
	client, cleanup := harness.startSessionWithOptions(sessionOptions{Args: bridgeArgs(goplsPath)})
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	usesCardURI := harness.fileURI("pages/uses_card.pk")
	usesCardContent := harness.readFile("pages/uses_card.pk")
	require.NoError(t, client.DidOpen(ctx, usesCardURI, usesCardContent))

	greetingPosition := findPosition(t, usesCardContent, "card.Greeting", "Greeting")

	require.Eventually(t, func() bool {
		hover, err := client.Hover(ctx, usesCardURI, greetingPosition)
		return err == nil && hoverContains(hover, "Greeting() string")
	}, analysisTimeout, pollInterval, "gopls did not resolve card.Greeting across the satellite overlay")

	diagnostics := client.GetDiagnostics(usesCardURI)
	require.False(t, anyDiagnosticMessageContains(diagnostics, "undefined"),
		"cross-component import produced an 'undefined' diagnostic: %+v", diagnostics)

	require.Empty(t, client.GetErrors())
}

func testBridgeDiagnosticsMappedToLine(t *testing.T) {
	goplsPath := requireGopls(t)
	t.Parallel()

	harness := newBridgeHarness(t)
	client, cleanup := harness.startSessionWithOptions(sessionOptions{Args: bridgeArgs(goplsPath)})
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	brokenURI := harness.fileURI("pages/broken.pk")
	brokenContent := harness.readFile("pages/broken.pk")
	require.NoError(t, client.DidOpen(ctx, brokenURI, brokenContent))

	errorLine := findLine(brokenContent, "var deliberate int")
	require.GreaterOrEqual(t, errorLine, 0, "the deliberate type-error line is missing from the fixture")

	diagnostics, ok := client.WaitForDiagnostics(brokenURI, goplsDiagnosticOnLine(uint32(errorLine)), analysisTimeout)
	require.True(t, ok,
		"expected a gopls diagnostic reverse-mapped onto .pk line %d, got %+v", errorLine, diagnostics)

	require.Empty(t, client.GetErrors())
}

func testBridgeRewriteNoiseSuppressed(t *testing.T) {
	goplsPath := requireGopls(t)
	t.Parallel()

	harness := newBridgeHarness(t)
	client, cleanup := harness.startSessionWithOptions(sessionOptions{Args: bridgeArgs(goplsPath)})
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	brokenURI := harness.fileURI("pages/broken.pk")
	brokenContent := harness.readFile("pages/broken.pk")
	require.NoError(t, client.DidOpen(ctx, brokenURI, brokenContent))

	_, ok := client.WaitForDiagnostics(brokenURI, hasGoplsSource, analysisTimeout)
	require.True(t, ok, "gopls never published a diagnostic for broken.pk to inspect for noise")

	diagnostics := client.GetDiagnostics(brokenURI)
	for _, marker := range []string{"/piko-lsp/", "/.piko/", "source.pk.go"} {
		require.False(t, anyDiagnosticMessageContains(diagnostics, marker),
			"a diagnostic leaked the internal overlay marker %q: %+v", marker, diagnostics)
	}

	scriptCloseLine := findLine(brokenContent, "</script>")
	require.GreaterOrEqual(t, scriptCloseLine, 0)
	for index := range diagnostics {
		if isGoplsDiagnostic(diagnostics[index]) {
			require.NotEqual(t, uint32(scriptCloseLine), diagnostics[index].Range.Start.Line,
				"a gopls diagnostic was anchored on the </script> boundary line")
		}
	}

	require.Empty(t, client.GetErrors())
}

func testBridgeSpaceInWorkspacePath(t *testing.T) {
	goplsPath := requireGopls(t)
	t.Parallel()

	harness := newStressHarnessForFixture(t, "gopls_bridge_project", spacedTempDir(t))
	client, cleanup := harness.startSessionWithOptions(sessionOptions{Args: bridgeArgs(goplsPath)})
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	brokenURI := harness.fileURI("pages/broken.pk")
	brokenContent := harness.readFile("pages/broken.pk")
	require.NoError(t, client.DidOpen(ctx, brokenURI, brokenContent))

	errorLine := findLine(brokenContent, "var deliberate int")
	diagnostics, ok := client.WaitForDiagnostics(brokenURI, goplsDiagnosticOnLine(uint32(errorLine)), analysisTimeout)
	require.True(t, ok,
		"gopls diagnostic did not survive a workspace path containing a space (URI percent-encoding round-trip); got %+v", diagnostics)

	require.Empty(t, client.GetErrors())
}

func testBridgeEnabledViaInitOption(t *testing.T) {
	goplsPath := requireGopls(t)
	t.Parallel()

	harness := newBridgeHarness(t)
	client, cleanup := harness.startSessionWithOptions(sessionOptions{
		Args:        []string{"--gopls-path=" + goplsPath},
		InitOptions: map[string]any{"goBridge": true},
	})
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	brokenURI := harness.fileURI("pages/broken.pk")
	brokenContent := harness.readFile("pages/broken.pk")
	require.NoError(t, client.DidOpen(ctx, brokenURI, brokenContent))

	_, ok := client.WaitForDiagnostics(brokenURI, hasGoplsSource, analysisTimeout)
	require.True(t, ok, "initializationOptions.goBridge=true did not enable the gopls bridge")

	require.Empty(t, client.GetErrors())
}

func testBridgeToggleOff(t *testing.T) {
	goplsPath := requireGopls(t)
	t.Parallel()

	harness := newBridgeHarness(t)
	client, cleanup := harness.startSessionWithOptions(sessionOptions{
		Args: []string{"--gopls-bridge=true", "--gopls-disable", "--gopls-path=" + goplsPath},
	})
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	brokenURI := harness.fileURI("pages/broken.pk")
	brokenContent := harness.readFile("pages/broken.pk")
	require.NoError(t, client.DidOpen(ctx, brokenURI, brokenContent))
	require.True(t, client.WaitForAnalysisComplete(brokenURI, requestTimeout), "piko analysis did not complete")

	diagnostics, ok := client.WaitForDiagnostics(brokenURI, hasGoplsSource, negativeGoplsWait)
	require.False(t, ok, "the gopls bridge produced diagnostics despite --gopls-disable: %+v", diagnostics)

	indexURI := harness.fileURI("pages/index.pk")
	indexContent := harness.readFile("pages/index.pk")
	require.NoError(t, client.DidOpen(ctx, indexURI, indexContent))
	printlnPosition := findPosition(t, indexContent, "fmt.Println", "Println")
	hover, err := client.Hover(ctx, indexURI, printlnPosition)
	require.NoError(t, err)
	require.False(t, hoverContains(hover, "func fmt.Println"),
		"hover returned gopls content despite --gopls-disable")

	list, completionErr := client.Completion(ctx, indexURI, printlnPosition)
	require.NoError(t, completionErr)
	require.False(t, completionHasItem(list, "Println"),
		"completion offered the gopls member Println despite --gopls-disable")

	require.Empty(t, client.GetErrors())
}

func testBridgeGoplsAbsent(t *testing.T) {
	requireGopls(t)
	t.Parallel()

	harness := newBridgeHarness(t)
	client, cleanup := harness.startSessionWithOptions(sessionOptions{
		Args: []string{"--gopls-bridge=true", "--gopls-path=/nonexistent/piko-gopls-does-not-exist"},
	})
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	brokenURI := harness.fileURI("pages/broken.pk")
	brokenContent := harness.readFile("pages/broken.pk")
	require.NoError(t, client.DidOpen(ctx, brokenURI, brokenContent))
	require.True(t, client.WaitForAnalysisComplete(brokenURI, requestTimeout),
		"piko analysis did not complete with gopls absent")

	diagnostics, ok := client.WaitForDiagnostics(brokenURI, hasGoplsSource, negativeGoplsWait)
	require.False(t, ok, "gopls produced diagnostics despite an unreachable --gopls-path: %+v", diagnostics)

	require.Empty(t, client.GetErrors(), "an unreachable gopls path produced client-visible errors")
}

func testBridgeDiagnosticsOnEditAndClear(t *testing.T) {
	goplsPath := requireGopls(t)
	t.Parallel()

	harness := newBridgeHarness(t)
	client, cleanup := harness.startSessionWithOptions(sessionOptions{Args: bridgeArgs(goplsPath)})
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	indexURI := harness.fileURI("pages/index.pk")
	validContent := harness.readFile("pages/index.pk")
	require.NoError(t, client.DidOpen(ctx, indexURI, validContent))

	const printCall = `fmt.Println("rendering index page")`
	brokenContent := strings.Replace(validContent, printCall, "var bad int = \"not an int\"\n\tfmt.Println(bad)", 1)
	require.NotEqual(t, validContent, brokenContent, "fixture changed: edit anchor %q not found", printCall)
	require.NoError(t, client.DidChange(ctx, indexURI, 2, brokenContent))

	errorLine := findLine(brokenContent, "var bad int")
	require.GreaterOrEqual(t, errorLine, 0)
	_, ok := client.WaitForDiagnostics(indexURI, goplsDiagnosticOnLine(uint32(errorLine)), analysisTimeout)
	require.True(t, ok, "gopls did not surface the type error introduced by an edit on .pk line %d", errorLine)

	require.NoError(t, client.DidChange(ctx, indexURI, 3, validContent))
	_, cleared := client.WaitForDiagnostics(indexURI, noGoplsSource, analysisTimeout)
	require.True(t, cleared, "the gopls diagnostic did not clear after the error was corrected")

	require.Empty(t, client.GetErrors())
}

func testBridgeRenameAcrossBlock(t *testing.T) {
	goplsPath := requireGopls(t)
	t.Parallel()

	harness := newBridgeHarness(t)
	client, cleanup := harness.startSessionWithOptions(sessionOptions{Args: bridgeArgs(goplsPath)})
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	indexURI := harness.fileURI("pages/index.pk")
	indexContent := harness.readFile("pages/index.pk")
	require.NoError(t, client.DidOpen(ctx, indexURI, indexContent))

	typePosition := findPosition(t, indexContent, "type IndexResponse struct", "IndexResponse")
	blockStart := findLine(indexContent, `<script type="application/x-go">`)
	blockEnd := findLine(indexContent, "</script>")

	const newName = "PageResponse"
	var edits []protocol.TextEdit
	require.Eventually(t, func() bool {
		edit, err := client.Rename(ctx, indexURI, typePosition, newName)
		if err != nil {
			return false
		}
		edits = editsForURI(edit, indexURI)
		return len(edits) >= 2
	}, analysisTimeout, pollInterval, "gopls rename of IndexResponse never produced the expected in-block edits")

	for _, edit := range edits {
		require.Equal(t, newName, edit.NewText, "a rename edit carried the wrong replacement text")
		require.True(t, rangeWithinBlock(edit.Range, blockStart, blockEnd),
			"a rename edit was not reverse-mapped into the Go block: %+v", edit)
	}

	require.Empty(t, client.GetErrors())
}

func testBridgeCompletionAutoImport(t *testing.T) {
	goplsPath := requireGopls(t)
	t.Parallel()

	harness := newBridgeHarness(t)
	client, cleanup := harness.startSessionWithOptions(sessionOptions{Args: bridgeArgs(goplsPath)})
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	autoImportURI := harness.fileURI("pages/autoimport.pk")
	autoImportContent := harness.readFile("pages/autoimport.pk")
	require.NoError(t, client.DidOpen(ctx, autoImportURI, autoImportContent))

	memberPosition := findPosition(t, autoImportContent, "strings.ToUpper", "ToUpper")
	blockStart := findLine(autoImportContent, `<script type="application/x-go">`)
	blockEnd := findLine(autoImportContent, "</script>")

	var importEdit protocol.TextEdit
	require.Eventually(t, func() bool {
		list, err := client.Completion(ctx, autoImportURI, memberPosition)
		if err != nil {
			return false
		}
		edit, found := completionImportEdit(list, `"strings"`)
		if found {
			importEdit = edit
		}
		return found
	}, analysisTimeout, pollInterval, "gopls never offered an unimported completion with a strings auto-import edit")

	require.True(t, rangeWithinBlock(importEdit.Range, blockStart, blockEnd),
		"the auto-import edit was not reverse-mapped into the Go block: %+v", importEdit)

	require.Empty(t, client.GetErrors())
}

func testBridgeUnterminatedLiteralClampsAndClears(t *testing.T) {
	goplsPath := requireGopls(t)
	t.Parallel()

	harness := newBridgeHarness(t)
	client, cleanup := harness.startSessionWithOptions(sessionOptions{Args: bridgeArgs(goplsPath)})
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	indexURI := harness.fileURI("pages/index.pk")
	validContent := harness.readFile("pages/index.pk")
	require.NoError(t, client.DidOpen(ctx, indexURI, validContent))

	const printCall = `fmt.Println("rendering index page")`
	brokenContent := strings.Replace(validContent, printCall, "fmt.Println(`oops", 1)
	require.NotEqual(t, validContent, brokenContent, "fixture changed: edit anchor %q not found", printCall)
	require.NoError(t, client.DidChange(ctx, indexURI, 2, brokenContent))

	scriptCloseLine := findLine(brokenContent, "</script>")
	require.GreaterOrEqual(t, scriptCloseLine, 0)

	diagnostics, ok := client.WaitForDiagnostics(indexURI, hasGoplsSource, analysisTimeout)
	require.True(t, ok, "gopls never reported the unterminated raw string literal")

	for index := range diagnostics {
		if !isGoplsDiagnostic(diagnostics[index]) {
			continue
		}
		require.Less(t, int(diagnostics[index].Range.End.Line), scriptCloseLine,
			"a gopls diagnostic end was painted onto or past the </script> line: %+v", diagnostics[index])
	}

	require.NoError(t, client.DidChange(ctx, indexURI, 3, validContent))
	_, cleared := client.WaitForDiagnostics(indexURI, noGoplsSource, analysisTimeout)
	require.True(t, cleared, "the gopls diagnostic did not clear after the literal was terminated")

	require.Empty(t, client.GetErrors())
}

func testBridgeMultiConnectionSharedChild(t *testing.T) {
	goplsPath := requireGopls(t)
	t.Parallel()

	harness := newBridgeHarness(t)
	port, serverCleanup := harness.startTCPBridgeServer(goplsPath)
	defer serverCleanup()

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	clientA, _ := harness.dialTCPClient(port)
	brokenURI := harness.fileURI("pages/broken.pk")
	require.NoError(t, clientA.DidOpen(ctx, brokenURI, harness.readFile("pages/broken.pk")))
	_, warmed := clientA.WaitForDiagnostics(brokenURI, hasGoplsSource, analysisTimeout)
	require.True(t, warmed, "client A did not warm the shared gopls child")

	clientB, clientBCleanup := harness.dialTCPClient(port)
	defer clientBCleanup()
	indexURI := harness.fileURI("pages/index.pk")
	indexContent := harness.readFile("pages/index.pk")
	require.NoError(t, clientB.DidOpen(ctx, indexURI, indexContent))

	require.NoError(t, clientA.Close())

	printlnPosition := findPosition(t, indexContent, "fmt.Println", "Println")
	require.Eventually(t, func() bool {
		hover, err := clientB.Hover(ctx, indexURI, printlnPosition)
		return err == nil && hoverContains(hover, "func fmt.Println")
	}, analysisTimeout, pollInterval,
		"client B lost gopls intelligence after the spawning connection disconnected")

	require.Empty(t, clientB.GetErrors())
}
