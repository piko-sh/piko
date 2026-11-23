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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

func TestDOMWrongAndSharedControls(t *testing.T) {
	plan := &casefile.DOMPlan{Path: "/markup", ReadyElement: "table", Steps: []casefile.DOMStep{{ID: "reverse", Op: "inspect", Fields: []casefile.DOMField{{ID: "identity", Query: casefile.DOMQuery{Selector: "tr", Read: "identity"}, Want: []string{"2", "1"}}}}}}
	recipe, err := plan.DriverPlan()
	require.NoError(t, err)
	require.Empty(t, recipe.Steps[0].Fields[0].Want, "driver leaked the model")
	require.Len(t, plan.Steps[0].Fields[0].Want, 2, "driver overwrote the model")
	trace := &casefile.DOMTrace{States: []casefile.DOMState{{Step: "reverse", Path: "/markup", Values: map[string][]string{"identity": {"2", "1"}}}}}
	count, err := VerifyDOM(plan, trace)
	require.NoError(t, err)
	require.Equal(t, 1, count)
	broken := &casefile.DOMTrace{States: []casefile.DOMState{{Step: "reverse", Path: "/markup", Values: map[string][]string{"identity": {"1", "2"}}}}}
	for _, actual := range []*casefile.DOMTrace{broken, nil, {}} {
		_, err := VerifyDOM(plan, actual)
		require.Error(t, err, "missed corruption")
		if actual == broken {
			assertFailureSubject(t, err, "/markup/reverse/identity")
		}
	}
	good, bad := Observation{Client: &casefile.ClientTransition{DOM: trace}}, Observation{Client: &casefile.ClientTransition{DOM: broken}}
	require.Error(t, compareObservations(&good, &bad), "missed unilateral corruption")
	testCase := casefile.Case{Expected: casefile.Expected{DOM: plan}, BrowserSampleVersion: 1}
	for _, route := range []struct {
		observation       Observation
		visited, verified bool
	}{
		{observation: good, visited: true, verified: true},
		{observation: bad, visited: true, verified: false},
		{observation: Observation{}, visited: false, verified: false},
	} {
		results, err := BrowserRouteResults(&testCase, &route.observation, true)
		require.NoError(t, err)
		assert.Equal(t, route.visited, results["/markup"].Visited, "%v", results)
		assert.Equal(t, route.verified, results["/markup"].Verified, "%v", results)
	}
	duplicate := *plan
	duplicate.Steps = append(duplicate.Steps, duplicate.Steps[0])
	_, err = duplicate.DriverPlan()
	assert.Error(t, err, "duplicate step id")
	nonlocal := *plan
	nonlocal.Path = "//another.example/path"
	_, err = nonlocal.DriverPlan()
	assert.Error(t, err, "accepted nonlocal document")
}

func TestDOMRetainsIndependentFieldWitnesses(t *testing.T) {
	model := &casefile.DOMPlan{Path: "/inspect", Steps: []casefile.DOMStep{
		{ID: "mount", Fields: []casefile.DOMField{{ID: "label", Want: []string{"Before"}}}},
		{ID: "edit", Fields: []casefile.DOMField{{ID: "label", Want: []string{"After"}}, {ID: "id", Want: []string{"a"}}}},
		{ID: "reset", Fields: []casefile.DOMField{{ID: "label", Want: []string{"Before"}}}},
	}}
	observed := &casefile.DOMTrace{States: []casefile.DOMState{
		{Step: "mount", Path: "/inspect", Values: map[string][]string{"label": {"Before"}}},
		{Step: "edit", Path: "/inspect", Values: map[string][]string{"label": {"Before"}, "id": {"b"}}},
		{Step: "reset", Path: "/inspect", Values: map[string][]string{"label": {"After"}}},
	}}
	prefix, failures := domFailures(model, observed)
	require.Equal(t, 1, prefix)
	require.Len(t, failures, 3)
	for index, want := range []string{"/inspect/edit/label", "/inspect/edit/id", "/inspect/reset/label"} {
		assertFailureSubject(t, failures[index], want)
	}
}
