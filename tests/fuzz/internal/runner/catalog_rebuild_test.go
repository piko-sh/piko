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

package runner

import (
	"context"
	"io"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/tests/fuzz/internal/generate"
)

func TestCatalogueReceiptRequiresEditUndoAndFreshComparison(t *testing.T) {
	first, second := strings.Repeat("aa", 32), strings.Repeat("bb", 32)
	valid := []CatalogRebuildStep{
		{Step: "baseline", WarmSHA256: first, CleanSHA256: first},
		{Step: "edit-card", WarmSHA256: second, CleanSHA256: second},
		{Step: "undo-card", WarmSHA256: first, CleanSHA256: first},
	}
	require.NoError(t, checkCatalogueReceipt(valid))
	changes := []func([]CatalogRebuildStep) []CatalogRebuildStep{
		func(steps []CatalogRebuildStep) []CatalogRebuildStep { return steps[:2] },
		func(steps []CatalogRebuildStep) []CatalogRebuildStep { steps[1].WarmSHA256 = first; return steps },
		func(steps []CatalogRebuildStep) []CatalogRebuildStep {
			steps[1].WarmSHA256 = first
			steps[1].CleanSHA256 = first
			return steps
		},
		func(steps []CatalogRebuildStep) []CatalogRebuildStep {
			steps[2].WarmSHA256 = second
			steps[2].CleanSHA256 = second
			return steps
		},
	}
	for _, change := range changes {
		assert.Error(t, checkCatalogueReceipt(change(slices.Clone(valid))), "accepted broken rebuild history")
	}
}

func TestCatalogueEditModelPreservesUnrelatedPage(t *testing.T) {
	testCase := generate.New("catalog-rebuild", 7, 0)
	baseHash := testCase.Hash()
	edit, err := editCatalogue(&testCase)
	require.NoError(t, err)
	require.Equal(t, baseHash, testCase.Hash(), "source edit not isolated")
	require.NotEqual(t, baseHash, edit.Hash(), "source edit not isolated")
	changed := 0
	summary := 0
	for index, probe := range edit.Probes {
		if probe.Path == "/catalog-summary" {
			summary++
			require.Equal(t, testCase.Probes[index].Semantic, probe.Semantic, "changed unaffected branch contract")
		}
		if probe.Path == "/catalog" {
			for elementIndex, element := range probe.Semantic.Elements {
				if element.Key == "buy" && strings.HasPrefix(element.Name, "Inspect ") && strings.HasPrefix(testCase.Probes[index].Semantic.Elements[elementIndex].Name, "Buy ") {
					changed++
				}
			}
		}
	}
	assert.Equal(t, 24, changed, "expected eight cards per actor")
	assert.Equal(t, 1, summary, "expected separate unaffected summary")
}

func TestWatchBudgetPreservesSemanticFailures(t *testing.T) {
	testCase := casefile.Case{Probes: []casefile.Probe{{Path: "/catalog", Semantic: &casefile.ResponseContract{Head: map[string][]string{"title": {"Expected"}}}}}}
	client := &http.Client{Transport: responseTransport(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("<title>Stale</title>"))}, nil
	})}
	o, err := awaitWatchedCatalogue(t.Context(), client, "http://local", &testCase, 20*time.Millisecond)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "head/title:")
	assert.NotErrorIs(t, err, context.DeadlineExceeded)
	require.Len(t, o.Probes, 1)
	assert.Equal(t, "<title>Stale</title>", o.Probes[0].RawHTML)
	interrupted, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = awaitWatchedCatalogue(interrupted, client, "http://local", &testCase, 20*time.Millisecond)
	assert.ErrorIs(t, err, context.Canceled, "lost outer cancellation")
}

func TestWatchedCatalogueObservationHasItsOwnBudget(t *testing.T) {
	testCase := casefile.Case{Request: "/catalog", Expected: casefile.Expected{Status: http.StatusTeapot}, Probes: []casefile.Probe{{Path: "/catalog", Semantic: &casefile.ResponseContract{Head: map[string][]string{"title": {"Expected"}}}}}}
	var deadlines []time.Time
	client := &http.Client{Transport: responseTransport(func(request *http.Request) (*http.Response, error) {
		if deadline, ok := request.Context().Deadline(); ok {
			deadlines = append(deadlines, deadline)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("<title>Expected</title>"))}, nil
	})}

	_, err := awaitWatchedCatalogue(t.Context(), client, "http://local", &testCase, time.Hour)

	require.ErrorContains(t, err, "status: got 200")

	require.GreaterOrEqual(t, len(deadlines), 2)
	assert.True(t, deadlines[len(deadlines)-1].After(deadlines[0]), "observation reused the rebuild budget")
}

func TestCatalogueDigestIsStableAndSensitive(t *testing.T) {
	observation := Observation{Status: http.StatusOK, Probes: []ProbeObservation{{Path: "/catalog", Actor: "guest", Status: http.StatusOK}}}

	first, err := catalogueDigest(&observation)
	require.NoError(t, err)
	second, err := catalogueDigest(&observation)
	require.NoError(t, err)
	assert.Equal(t, first, second)
	assert.Len(t, first, 64)

	observation.Probes[0].Status = http.StatusNotFound
	changed, err := catalogueDigest(&observation)
	require.NoError(t, err)
	assert.NotEqual(t, first, changed)
}
