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
	"math/rand/v2"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/protocol"
)

func testScenarioSustainedLoad(t *testing.T) {
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
		{harness.fileURI("partials/nav.pk"), harness.readFile("partials/nav.pk")},
	}

	for _, f := range files {
		require.NoError(t, client.DidOpen(ctx, f.uri, f.content))
		require.True(t, client.WaitForAnalysisComplete(f.uri, analysisTimeout),
			"initial analysis should complete for %s", f.uri)
	}

	const loadDuration = 30 * time.Second
	deadline := time.Now().Add(loadDuration)

	var (
		editCount atomic.Int64
		reqCount  atomic.Int64
		errCount  atomic.Int64
		versions  = make([]atomic.Int32, len(files))
		wg        sync.WaitGroup
	)

	for i := range versions {
		versions[i].Store(1)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		rng := rand.New(rand.NewPCG(42, 0))

		for time.Now().Before(deadline) {
			idx := rng.IntN(len(files))
			f := files[idx]
			v := versions[idx].Add(1)
			modified := generateModifiedTemplate(f.content, int(v))

			if err := client.DidChange(ctx, f.uri, v, modified); err != nil {
				errCount.Add(1)
				return
			}
			editCount.Add(1)

			time.Sleep(time.Duration(rng.IntN(50)) * time.Millisecond)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		rng := rand.New(rand.NewPCG(99, 0))

		pos := protocol.Position{Line: 3, Character: 10}

		for time.Now().Before(deadline) {
			idx := rng.IntN(len(files))
			f := files[idx]

			reqCtx, reqCancel := context.WithTimeout(ctx, requestTimeout)

			switch rng.IntN(3) {
			case 0:
				_, err := client.Completion(reqCtx, f.uri, pos)
				if err != nil {
					errCount.Add(1)
				}
			case 1:
				_, err := client.Hover(reqCtx, f.uri, pos)
				if err != nil {
					errCount.Add(1)
				}
			case 2:
				_, err := client.Definition(reqCtx, f.uri, pos)
				if err != nil {
					errCount.Add(1)
				}
			}

			reqCancel()
			reqCount.Add(1)

			time.Sleep(time.Duration(rng.IntN(100)) * time.Millisecond)
		}
	}()

	wg.Wait()

	t.Logf("Sustained load completed: %d edits, %d requests, %d request errors",
		editCount.Load(), reqCount.Load(), errCount.Load())

	time.Sleep(2 * time.Second)

	assert.Empty(t, client.GetErrors(), "no protocol errors during sustained load")
}
