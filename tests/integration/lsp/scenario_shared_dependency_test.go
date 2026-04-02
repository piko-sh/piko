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
)

func testScenarioSharedDependency(t *testing.T) {
	t.Parallel()

	harness := newStressHarness(t)
	client, cleanup := harness.startSession()
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	homeURI := harness.fileURI("pages/home.pk")
	aboutURI := harness.fileURI("pages/about.pk")
	navURI := harness.fileURI("partials/nav.pk")

	require.NoError(t, client.DidOpen(ctx, homeURI, harness.readFile("pages/home.pk")))
	require.True(t, client.WaitForAnalysisComplete(homeURI, analysisTimeout))

	require.NoError(t, client.DidOpen(ctx, aboutURI, harness.readFile("pages/about.pk")))
	require.True(t, client.WaitForAnalysisComplete(aboutURI, analysisTimeout))

	require.NoError(t, client.DidOpen(ctx, navURI, harness.readFile("partials/nav.pk")))
	require.True(t, client.WaitForAnalysisComplete(navURI, analysisTimeout))

	modifiedNav := generateModifiedTemplate(harness.readFile("partials/nav.pk"), 1)
	require.NoError(t, client.DidChange(ctx, navURI, 2, modifiedNav))
	require.True(t, client.WaitForAnalysisComplete(navURI, analysisTimeout),
		"nav analysis should complete after modification")

	require.NoError(t, client.DidChange(ctx, homeURI, 2, harness.readFile("pages/home.pk")))
	require.True(t, client.WaitForAnalysisComplete(homeURI, analysisTimeout),
		"home should re-analyse after partial change")

	require.NoError(t, client.DidChange(ctx, aboutURI, 2, harness.readFile("pages/about.pk")))
	require.True(t, client.WaitForAnalysisComplete(aboutURI, analysisTimeout),
		"about should re-analyse after partial change")

	assert.Empty(t, client.GetErrors(), "no protocol errors during shared dependency modification")
}
