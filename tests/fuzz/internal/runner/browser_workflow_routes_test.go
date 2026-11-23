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
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

func newTestTally(selected bool) *workflowRouteTally {
	return &workflowRouteTally{first: nil, routes: map[string]BrowserRouteEvidence{}, incomplete: map[string]bool{}, selected: selected}
}

func TestWorkflowRouteTallyKeepsTheFirstNavigationFailure(t *testing.T) {
	tally := newTestTally(true)
	tally.record("/dom", "/elsewhere", true)
	workflow := &casefile.FormWorkflow{Steps: []casefile.FormCommand{{Op: "create", Value: "one"}}}

	tally.recordFormWorkflow(workflow, &casefile.FormWorkflowObservation{})
	tally.recordEvents(&casefile.EventsTrace{}, &casefile.ClientTransition{Events: &casefile.EventsTrace{}})

	mismatch, ok := errors.AsType[routeMismatch](tally.first)
	require.True(t, ok)
	assert.Equal(t, "/dom/navigation", mismatch.subject)
}

func TestWorkflowRouteTallyIgnoresMissingLocationsWithoutABrowser(t *testing.T) {
	tally := newTestTally(false)
	workflow := &casefile.FormWorkflow{Steps: []casefile.FormCommand{{Op: "create", Value: "one"}}}

	tally.recordFormWorkflow(workflow, &casefile.FormWorkflowObservation{})
	tally.recordEvents(&casefile.EventsTrace{}, &casefile.ClientTransition{Events: &casefile.EventsTrace{}})

	assert.NoError(t, tally.first)
}

func TestWorkflowRouteTallyReportsAMissingWorkflowLocation(t *testing.T) {
	tally := newTestTally(true)
	workflow := &casefile.FormWorkflow{Steps: []casefile.FormCommand{{Op: "create", Value: "one"}}}

	tally.recordFormWorkflow(workflow, &casefile.FormWorkflowObservation{})

	mismatch, ok := errors.AsType[routeMismatch](tally.first)
	require.True(t, ok)
	assert.Equal(t, "/workflow/navigation", mismatch.subject)
}
