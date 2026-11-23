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

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

func TestDOMSelectorQuotesHostAndTarget(t *testing.T) {
	document, err := domSelector("", "#a'b")
	require.NoError(t, err)
	assert.Equal(t, `document.querySelector("#a'b")`, document)
	shadow, err := domSelector(`x-"host"`, "#target")
	require.NoError(t, err)
	assert.Equal(t, `document.querySelector("x-\"host\"")?.shadowRoot?.querySelector("#target")`, shadow)
}

func TestDOMJourneyRecordsInvalidPlanWithoutBrowser(t *testing.T) {
	trace := domJourney(t.Context(), nil, "http://127.0.0.1", casefile.DOMPlan{Path: "https://elsewhere.invalid/"}, nil)
	assert.Equal(t, "chromium", trace.Engine)
	assert.Empty(t, trace.States)
	assert.Contains(t, trace.DriverError, "invalid DOM scenario")

	plan := casefile.DOMPlan{Path: "/dom", ReadyElement: "#ready", Steps: []casefile.DOMStep{
		{ID: "mount", Op: "inspect", Fields: []casefile.DOMField{{ID: "text", Query: casefile.DOMQuery{Selector: "#ready", Read: "text"}}}},
	}}
	trace = domJourney(t.Context(), nil, "://missing-scheme", plan, nil)
	assert.Contains(t, trace.DriverError, "missing protocol scheme")
}

func TestRunDOMStepRejectsUnsupportedKey(t *testing.T) {
	_, err := runDOMStep(t.Context(), nil, casefile.DOMStep{ID: "key", Op: "key", Target: "#field", Value: "F1"}, nil)
	require.ErrorContains(t, err, `unsupported key "F1"`)
}

func TestDOMBudgetCountsSteps(t *testing.T) {
	assert.Equal(t, domReadyWait+2*domStepBudget, domBudget(casefile.DOMPlan{Steps: make([]casefile.DOMStep, 2)}))
}
