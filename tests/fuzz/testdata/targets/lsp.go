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

package lsp_stress_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf16"

	protocol "github.com/politepixels/golang-language-server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeneratedPikoCase(t *testing.T) {
	token := os.Getenv("PIKO_FUZZ_CASE_TOKEN")
	require.NotEmpty(t, token)
	h := newStressHarness(t)
	uri := h.fileURI("pages/home.pk")
	const base = `<template>
  <div>Generated editor document</div>
</template>
<script type="application/x-go">
package main

import (
  "piko.sh/piko"
  left "testcase_lsp_stress/partials/left/badge.pk"
  right "testcase_lsp_stress/partials/right/badge.pk"
)

type Response struct { Welcome string }
func Render(_ *piko.RequestData, _ piko.NoProps) (Response, piko.Metadata, error) {
  return Response{Welcome:"hello"}, piko.Metadata{}, nil
}
</script>
`

	for _, side := range []string{"left", "right"} {
		dir := filepath.Join(h.srcDir, "partials", side)
		require.NoError(t, os.MkdirAll(dir, 0700))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "badge.pk"), []byte("<template><span>"+side+"</span></template>"), 0600))
	}
	end := strings.Index(base, "</template>")
	makeVersion := func(label, side string) string {
		return base[:end] + "<p>🙂 🐦 " + label + " {{ state.Welcome }}</p><piko:partial is=\"" + side + "\" />\n" + base[end:]
	}
	checkDefinition := func(client *stressClient, uri protocol.DocumentURI, doc, side string) []protocol.Location {
		t.Helper()
		offset := strings.Index(doc, "<piko:partial") + len("<piko:")
		before := doc[:offset]
		pos := protocol.Position{Line: uint32(strings.Count(before, "\n")), Character: uint32(len(utf16.Encode([]rune(before[strings.LastIndex(before, "\n")+1:]))))}
		loc, err := client.Definition(t.Context(), uri, pos)
		require.NoError(t, err)
		expected := h.fileURI("partials/" + side + "/badge.pk")
		if len(loc) != 1 || loc[0].URI != expected || loc[0].Range != (protocol.Range{}) {
			require.FailNow(t, fmt.Sprintf("PIKO_CHECK cross-file-definition: cursor %+v expected %s, got %+v", pos, expected, loc))
		}
		return loc
	}
	compare := func(edited bool) ([]protocol.Diagnostic, []protocol.Location) {
		final := makeVersion(token, "left")
		initial := final
		if edited {
			initial = base
		}
		docURI := uri
		filename := filepath.Join(h.srcDir, "pages", "home.pk")
		dest := filepath.Join(h.srcDir, "pages", "moved-"+token+".pk")
		destURI := h.fileURI("pages/moved-" + token + ".pk")
		if !edited {
			filename = dest
			docURI = destURI
		}
		require.NoError(t, os.WriteFile(filename, []byte(initial), 0600))
		client, closeSession := h.startSession()
		defer closeSession()
		require.NoError(t, client.DidOpen(t.Context(), docURI, initial))
		require.True(t, client.WaitForAnalysisComplete(docURI, analysisTimeout))
		if edited {
			for i := range 2 + int(token[0])%4 {
				side := []string{"right", "left"}[i%2]
				editedDoc := makeVersion(fmt.Sprintf("%s-%d", token, i), side)
				require.NoError(t, client.DidChange(t.Context(), uri, int32(i+2), editedDoc))
				require.True(t, client.WaitForAnalysisComplete(uri, analysisTimeout))
				checkDefinition(client, uri, editedDoc, side)
			}
			require.NoError(t, client.DidChange(t.Context(), uri, 10, final))
			require.True(t, client.WaitForAnalysisComplete(uri, analysisTimeout))
		}

		if edited {
			require.NoError(t, client.conn.Notify(t.Context(), protocol.MethodTextDocumentDidClose, &protocol.DidCloseTextDocumentParams{TextDocument: protocol.TextDocumentIdentifier{URI: uri}}))
			require.NoError(t, os.Rename(filename, dest))
			require.NoError(t, os.WriteFile(dest, []byte(final), 0600))
			require.NoError(t, client.conn.Notify(t.Context(), protocol.MethodWorkspaceDidChangeWatchedFiles, &protocol.DidChangeWatchedFilesParams{Changes: []*protocol.FileEvent{{URI: uri, Type: protocol.FileChangeTypeDeleted}, {URI: destURI, Type: protocol.FileChangeTypeCreated}}}))
			docURI = destURI
			require.NoError(t, client.DidOpen(t.Context(), docURI, final))
			require.True(t, client.WaitForAnalysisComplete(docURI, analysisTimeout))
		}
		locations := checkDefinition(client, docURI, final, "left")
		require.Empty(t, client.GetErrors())
		return client.GetDiagnostics(docURI), locations
	}
	componentPath := "components/edited-" + token + ".pkc"
	componentURI := h.fileURI(componentPath)
	component := `<template name="edited-field"><button p-on:click="bump()">{{ state.count }}</button></template>
<script lang="ts">const state={count:1}; function bump(){state.count+=1;}</script>`
	require.NoError(t, os.MkdirAll(filepath.Dir(filepath.Join(h.srcDir, componentPath)), 0700))
	require.NoError(t, os.WriteFile(filepath.Join(h.srcDir, componentPath), []byte(component), 0600))
	componentDiagnostics := func(edited bool) []protocol.Diagnostic {
		client, stop := h.startSession()
		defer stop()
		client.recordSentVersion(componentURI, 1)
		require.NoError(t, client.conn.Notify(t.Context(), protocol.MethodTextDocumentDidOpen, &protocol.DidOpenTextDocumentParams{TextDocument: protocol.TextDocumentItem{URI: componentURI, LanguageID: "pkc", Version: 1, Text: component}}))
		require.True(t, client.WaitForAnalysisComplete(componentURI, analysisTimeout))
		if edited {
			broken := strings.Replace(component, "state.count }}", "state.missingCount }}", 1)
			require.NoError(t, client.DidChange(t.Context(), componentURI, 2, broken))
			require.True(t, client.WaitForAnalysisComplete(componentURI, analysisTimeout))
			found := false
			for _, d := range client.GetDiagnostics(componentURI) {
				if d.Source == "piko" && strings.Contains(d.Message, "missingCount") {
					found = true
				}
			}
			require.True(t, found, "PIKO_CHECK component-invalid: missing state property diagnostic, got %v", client.GetDiagnostics(componentURI))
			require.NoError(t, client.DidChange(t.Context(), componentURI, 3, component))
			require.True(t, client.WaitForAnalysisComplete(componentURI, analysisTimeout))
		}
		return client.GetDiagnostics(componentURI)
	}
	repaired := componentDiagnostics(true)
	coldComponent := componentDiagnostics(false)
	require.ElementsMatch(t, coldComponent, repaired, "PIKO_CHECK component-repair: fresh diagnostics")

	warm, defs := compare(true)
	cold, coldDefs := compare(false)
	if !assert.ElementsMatch(t, cold, warm) {
		require.FailNow(t, "PIKO_CHECK diagnostics: cold and edited document diagnostics differ")
	}
	require.ElementsMatch(t, coldDefs, defs)
}
