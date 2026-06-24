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

package lsp_domain

import (
	"context"
	"testing"

	protocol "github.com/politepixels/golang-language-server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/cmd/lsp/internal/lsp/gopls_bridge"
)

func TestHandleGoplsDiagnosticsUnknownOverlayIsIgnored(t *testing.T) {
	t.Parallel()

	workspace := &workspace{}
	workspace.handleGoplsDiagnostics(context.Background(), &protocol.PublishDiagnosticsParams{
		URI:         "file:///site/piko-lsp/unknown/source.pk.go",
		Diagnostics: []protocol.Diagnostic{{Message: "boom"}},
	})

	assert.Nil(t, workspace.goplsDiagnosticsFor(testRealURI), "diagnostics for an unknown overlay are dropped")
}

func TestHandleGoplsDiagnosticsReverseMapsAndStores(t *testing.T) {
	t.Parallel()

	workspace := &workspace{}
	mapper := gopls_bridge.NewMapper(testRealURI, testPrimaryURI, 10, 1)
	workspace.registerGoplsOverlay(testPrimaryURI, testRealURI, mapper, 50)

	workspace.handleGoplsDiagnostics(context.Background(), &protocol.PublishDiagnosticsParams{
		URI: testPrimaryURI,
		Diagnostics: []protocol.Diagnostic{
			{Range: sampleRange(2, 4, 9), Message: "undeclared name"},
			{Range: sampleRange(0, 0, 3), Message: "with explicit source", Source: "vet"},
		},
	})

	stored := workspace.goplsDiagnosticsFor(testRealURI)
	require.Len(t, stored, 2)

	assert.Equal(t, uint32(11), stored[0].Range.Start.Line)
	assert.Equal(t, "gopls", stored[0].Source, "an empty source is labelled gopls")
	assert.Equal(t, uint32(9), stored[1].Range.Start.Line)
	assert.Equal(t, "vet", stored[1].Source, "an explicit source is preserved")
}

func TestHandleGoplsDiagnosticsClampsMultiLineEndToBlock(t *testing.T) {
	t.Parallel()

	workspace := &workspace{}

	mapper := gopls_bridge.NewMapper(testRealURI, testPrimaryURI, 10, 1)
	workspace.registerGoplsOverlay(testPrimaryURI, testRealURI, mapper, 5)

	workspace.handleGoplsDiagnostics(context.Background(), &protocol.PublishDiagnosticsParams{
		URI: testPrimaryURI,
		Diagnostics: []protocol.Diagnostic{
			{
				Range:   protocol.Range{Start: protocol.Position{Line: 3, Character: 2}, End: protocol.Position{Line: 5, Character: 0}},
				Message: "string literal not terminated",
			},
		},
	})

	stored := workspace.goplsDiagnosticsFor(testRealURI)
	require.Len(t, stored, 1, "an in-block diagnostic survives even when its end runs past the block")
	assert.Equal(t, uint32(12), stored[0].Range.Start.Line)
	assert.Equal(t, uint32(13), stored[0].Range.End.Line, "the end is clamped to the last in-block .pk line, off the </script> tag on line 14")
	assert.Equal(t, uint32(lineEndColumnSentinel), stored[0].Range.End.Character, "the clamped end extends to the line end via the sentinel column")
}

func TestHandleGoplsDiagnosticsWindowsAndDenyLists(t *testing.T) {
	t.Parallel()

	workspace := &workspace{}
	mapper := gopls_bridge.NewMapper(testRealURI, testPrimaryURI, 10, 1)

	workspace.registerGoplsOverlay(testPrimaryURI, testRealURI, mapper, 5)

	workspace.handleGoplsDiagnostics(context.Background(), &protocol.PublishDiagnosticsParams{
		URI: testPrimaryURI,
		Diagnostics: []protocol.Diagnostic{
			{Range: sampleRange(1, 0, 3), Message: "real Go error in the block"},
			{Range: sampleRange(9, 0, 1), Message: "gopls EOF marker past the block end"},
			{Range: sampleRange(2, 0, 3), Message: "error in the synthetic overlay " + gopls_bridge.OverlayPathMarker + "layout_hash/source.pk.go"},
		},
	})

	stored := workspace.goplsDiagnosticsFor(testRealURI)
	require.Len(t, stored, 1, "the EOF-past-block and rewrite-induced diagnostics are dropped")
	assert.Equal(t, "real Go error in the block", stored[0].Message)
	assert.Equal(t, uint32(10), stored[0].Range.Start.Line, "the surviving diagnostic is reverse-mapped")
}

func TestInvalidateGoplsDiagnostics(t *testing.T) {
	t.Parallel()

	workspace := &workspace{}
	workspace.setGoplsDiagnostics(testRealURI, []protocol.Diagnostic{{Message: "stale"}})
	workspace.invalidateGoplsDiagnostics(testRealURI)
	assert.Nil(t, workspace.goplsDiagnosticsFor(testRealURI), "an edit discards stale gopls diagnostics")
}

func TestOnGoplsChildDeathClearsCache(t *testing.T) {
	t.Parallel()

	workspace := &workspace{}
	workspace.setGoplsDiagnostics(testRealURI, []protocol.Diagnostic{{Message: "pre-crash error"}})

	workspace.onGoplsChildDeath("/module/root")

	assert.Nil(t, workspace.goplsDiagnosticsFor(testRealURI), "a gopls crash clears cached Go diagnostics")
}

func TestUnregisterGoplsOverlays(t *testing.T) {
	t.Parallel()

	workspace := &workspace{}
	mapper := gopls_bridge.NewMapper(testRealURI, testPrimaryURI, 10, 1)
	workspace.registerGoplsOverlay(testPrimaryURI, testRealURI, mapper, 50)
	workspace.registerGoplsOverlay(testSatelliteU, testRealURI, mapper, 50)
	workspace.setGoplsDiagnostics(testRealURI, []protocol.Diagnostic{{Message: "x"}})

	virtualURIs := workspace.unregisterGoplsOverlays(testRealURI)

	assert.ElementsMatch(t, []protocol.DocumentURI{testPrimaryURI, testSatelliteU}, virtualURIs,
		"every overlay belonging to the closed document is returned for closing")
	assert.Nil(t, workspace.goplsDiagnosticsFor(testRealURI), "cached diagnostics are purged")

	assert.Empty(t, workspace.unregisterGoplsOverlays(testRealURI))
}
