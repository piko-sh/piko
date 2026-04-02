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

func testScenarioConcurrentCompletion(t *testing.T) {
	t.Parallel()

	harness := newStressHarness(t)
	client, cleanup := harness.startSession()
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	homeURI := harness.fileURI("pages/home.pk")
	homeContent := harness.readFile("pages/home.pk")

	require.NoError(t, client.DidOpen(ctx, homeURI, homeContent))
	require.True(t, client.WaitForAnalysisComplete(homeURI, analysisTimeout),
		"initial analysis should complete")

	modified := generateModifiedTemplate(homeContent, 1)
	require.NoError(t, client.DidChange(ctx, homeURI, 2, modified))

	reqCtx, reqCancel := context.WithTimeout(ctx, requestTimeout)
	defer reqCancel()

	result, err := client.Completion(reqCtx, homeURI, protocol.Position{
		Line:      4,
		Character: 10,
	})
	require.NoError(t, err, "completion during edit should not error or deadlock")
	require.NotNil(t, result, "completion should return a result")

	assert.Empty(t, client.GetErrors(), "no protocol errors during concurrent completion")
}
