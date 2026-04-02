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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/protocol"
)

func testScenarioMultiFile(t *testing.T) {
	t.Parallel()

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

	for i := range 30 {
		f := files[i%len(files)]
		modified := generateModifiedTemplate(f.content, i)
		require.NoError(t, client.DidChange(ctx, f.uri, int32(i+2), modified),
			"DidChange %d should not fail", i)
	}

	for _, f := range files {
		require.True(t, client.WaitForAnalysisComplete(f.uri, analysisTimeout),
			"analysis should complete for %s", f.uri)
	}

	assert.Empty(t, client.GetErrors(), "no protocol errors during multi-file edits")
}
