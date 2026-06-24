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
	"sync"
	"testing"

	protocol "github.com/politepixels/golang-language-server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testScenarioMultiFileStorm(t *testing.T) {
	t.Parallel()

	const editsPerFile = 40

	harness := newStressHarness(t)
	client, cleanup := harness.startSession()
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	type fileInfo struct {
		uri     protocol.DocumentURI
		content string
	}

	files := []fileInfo{
		{harness.fileURI("pages/home.pk"), harness.readFile("pages/home.pk")},
		{harness.fileURI("pages/about.pk"), harness.readFile("pages/about.pk")},
		{harness.fileURI("pages/contact.pk"), harness.readFile("pages/contact.pk")},
	}

	for _, f := range files {
		require.NoError(t, client.DidOpen(ctx, f.uri, f.content))
		require.True(t, client.WaitForAnalysisComplete(f.uri, analysisTimeout),
			"initial analysis should complete for %s", f.uri)
	}

	var editors sync.WaitGroup
	for _, f := range files {
		editors.Go(func() {
			for edit := range editsPerFile {

				version := int32(edit + 2)
				modified := generateModifiedTemplate(f.content, edit)
				if err := client.DidChange(ctx, f.uri, version, modified); err != nil {
					assert.NoError(t, err, "DidChange for %s edit %d should not fail", f.uri, edit)
					return
				}
			}
		})
	}
	editors.Wait()

	for _, f := range files {
		require.True(t, client.WaitForAnalysisComplete(f.uri, analysisTimeout),
			"analysis should complete for %s after the concurrent edit storm", f.uri)
	}

	assert.Empty(t, client.GetErrors(), "no protocol errors during the concurrent multi-file storm")
}
