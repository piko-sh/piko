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
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	fakeModeWedge      = "wedge"
	fakeModeCrash      = "crash"
	fakeModeCrashOnce  = "crashonce"
	fakeModeOldVersion = "oldversion"
	fakeModeNullInit   = "nullinit"
)

const (
	fakeCrashMarkerEnv             = "FAKEGOPLS_CRASH_MARKER"
	fakeSyntheticDiagnosticMessage = "fakegopls synthetic diagnostic"
)

const (
	crashSettleWait  = 2 * time.Second
	wedgeHoverBudget = 15 * time.Second
)

func fakeBridgeSession(t *testing.T, mode string) (*stressHarness, *stressClient, func()) {
	t.Helper()
	fakePath := buildFakeGoplsBinary(t)
	harness := newBridgeHarness(t)
	client, cleanup := harness.startSessionWithOptions(sessionOptions{
		Args: []string{"--gopls-bridge=true", "--gopls-path=" + fakePath},
		Env:  []string{"FAKEGOPLS_MODE=" + mode},
	})
	return harness, client, cleanup
}

func testBridgeWedgedGopls(t *testing.T) {
	t.Parallel()

	harness, client, cleanup := fakeBridgeSession(t, fakeModeWedge)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	indexURI := harness.fileURI("pages/index.pk")
	indexContent := harness.readFile("pages/index.pk")
	require.NoError(t, client.DidOpen(ctx, indexURI, indexContent))
	require.True(t, client.WaitForAnalysisComplete(indexURI, requestTimeout), "piko analysis did not complete")

	printlnPosition := findPosition(t, indexContent, "fmt.Println", "Println")

	requestCtx, requestCancel := context.WithTimeout(ctx, requestTimeout)
	defer requestCancel()
	start := time.Now()
	_, err := client.Hover(requestCtx, indexURI, printlnPosition)
	elapsed := time.Since(start)

	require.NoError(t, err, "a wedged gopls must not surface a hover error")
	require.Less(t, elapsed, wedgeHoverBudget, "a wedged gopls hung the hover instead of falling back to piko")
	require.Empty(t, client.GetErrors())
}

func testBridgeCrashedGopls(t *testing.T) {
	t.Parallel()

	harness, client, cleanup := fakeBridgeSession(t, fakeModeCrash)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	indexURI := harness.fileURI("pages/index.pk")
	indexContent := harness.readFile("pages/index.pk")
	require.NoError(t, client.DidOpen(ctx, indexURI, indexContent))
	require.True(t, client.WaitForAnalysisComplete(indexURI, requestTimeout), "piko analysis did not complete")

	time.Sleep(crashSettleWait)

	printlnPosition := findPosition(t, indexContent, "fmt.Println", "Println")
	requestCtx, requestCancel := context.WithTimeout(ctx, requestTimeout)
	defer requestCancel()
	_, err := client.Hover(requestCtx, indexURI, printlnPosition)
	require.NoError(t, err, "a crashed gopls child must not surface a hover error")

	require.Empty(t, client.GetErrors())
}

func testBridgeTooOldGopls(t *testing.T) {
	t.Parallel()

	harness, client, cleanup := fakeBridgeSession(t, fakeModeOldVersion)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	brokenURI := harness.fileURI("pages/broken.pk")
	brokenContent := harness.readFile("pages/broken.pk")
	require.NoError(t, client.DidOpen(ctx, brokenURI, brokenContent))
	require.True(t, client.WaitForAnalysisComplete(brokenURI, requestTimeout), "piko analysis did not complete")

	diagnostics, ok := client.WaitForDiagnostics(brokenURI, hasGoplsSource, negativeGoplsWait)
	require.False(t, ok, "a gopls below the supported floor must not produce diagnostics: %+v", diagnostics)

	require.Empty(t, client.GetErrors())
}

func testBridgeNullInitGopls(t *testing.T) {
	t.Parallel()

	harness, client, cleanup := fakeBridgeSession(t, fakeModeNullInit)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	brokenURI := harness.fileURI("pages/broken.pk")
	brokenContent := harness.readFile("pages/broken.pk")
	require.NoError(t, client.DidOpen(ctx, brokenURI, brokenContent))
	require.True(t, client.WaitForAnalysisComplete(brokenURI, requestTimeout),
		"piko analysis did not survive a null gopls initialize result")

	diagnostics, ok := client.WaitForDiagnostics(brokenURI, hasGoplsSource, negativeGoplsWait)
	require.False(t, ok, "a null initialize result must not yield gopls diagnostics: %+v", diagnostics)

	require.Empty(t, client.GetErrors())
}

func testBridgeCrashRecovery(t *testing.T) {
	t.Parallel()

	fakePath := buildFakeGoplsBinary(t)
	marker := filepath.Join(t.TempDir(), "crash.marker")
	harness := newBridgeHarness(t)
	client, cleanup := harness.startSessionWithOptions(sessionOptions{
		Args: []string{"--gopls-bridge=true", "--gopls-path=" + fakePath},
		Env:  []string{"FAKEGOPLS_MODE=" + fakeModeCrashOnce, fakeCrashMarkerEnv + "=" + marker},
	})
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	indexURI := harness.fileURI("pages/index.pk")
	indexContent := harness.readFile("pages/index.pk")
	require.NoError(t, client.DidOpen(ctx, indexURI, indexContent))

	_, appeared := client.WaitForDiagnostics(indexURI, hasSyntheticGoplsDiagnostic, analysisTimeout)
	require.True(t, appeared, "the bridge never forwarded the fake gopls diagnostic")

	_, cleared := client.WaitForDiagnostics(indexURI, noGoplsSource, analysisTimeout)
	require.True(t, cleared, "the bridge did not clear the gopls diagnostic when the child crashed")

	require.NoError(t, client.DidChange(ctx, indexURI, 2, indexContent))
	_, recovered := client.WaitForDiagnostics(indexURI, hasSyntheticGoplsDiagnostic, analysisTimeout)
	require.True(t, recovered, "the bridge did not recover Go intelligence after respawning a healthy child")

	require.Empty(t, client.GetErrors())
}
