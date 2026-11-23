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

package casefile

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validDOMPlan() DOMPlan {
	return DOMPlan{
		Path:         "/dom",
		ReadyElement: "#root",
		Steps: []DOMStep{
			{ID: "look", Op: "inspect", Fields: []DOMField{{ID: "title", Query: DOMQuery{Selector: "h1", Read: "text"}, Want: []string{"Title"}}}},
			{ID: "press", Op: "key", Target: "#input", Value: "Enter", Fields: []DOMField{{ID: "value", Query: DOMQuery{Selector: "#input", Read: "value"}, Want: []string{"x"}}}},
		},
	}
}

func TestDriverPlanStripsExpectedValuesWithoutTouchingTheModel(t *testing.T) {
	plan := validDOMPlan()

	driver, err := plan.DriverPlan()

	require.NoError(t, err)
	require.Len(t, driver.Steps, 2)
	for _, step := range driver.Steps {
		for _, field := range step.Fields {
			assert.Nil(t, field.Want)
		}
	}
	assert.Equal(t, []string{"Title"}, plan.Steps[0].Fields[0].Want, "the saved plan must keep its expectations")
}

func TestDriverPlanRejectsInvalidPlans(t *testing.T) {
	testCases := []struct {
		change func(*DOMPlan)
		name   string
	}{
		{name: "remote path", change: func(plan *DOMPlan) { plan.Path = "https://example.test/" }},
		{name: "no readiness", change: func(plan *DOMPlan) { plan.ReadyElement = "" }},
		{name: "no steps", change: func(plan *DOMPlan) { plan.Steps = nil }},
		{name: "duplicate step", change: func(plan *DOMPlan) { plan.Steps[1].ID = "look" }},
		{name: "empty fields", change: func(plan *DOMPlan) { plan.Steps[0].Fields = nil }},
		{name: "oversized value", change: func(plan *DOMPlan) { plan.Steps[1].Value = strings.Repeat("v", maxDOMValueBytes+1) }},
		{name: "unknown key", change: func(plan *DOMPlan) { plan.Steps[1].Value = "F13" }},
		{name: "unknown operation", change: func(plan *DOMPlan) { plan.Steps[0].Op = "hover" }},
		{name: "missing target", change: func(plan *DOMPlan) { plan.Steps[1].Target = "" }},
		{name: "bad viewport", change: func(plan *DOMPlan) { plan.Steps[0].Op, plan.Steps[0].Value = "viewport", "tall" }},
		{name: "oversized selector", change: func(plan *DOMPlan) {
			plan.Steps[0].Fields[0].Query.Selector = strings.Repeat("s", maxDOMSelectorBytes+1)
		}},
		{name: "unknown read", change: func(plan *DOMPlan) { plan.Steps[0].Fields[0].Query.Read = "html" }},
		{name: "duplicate field", change: func(plan *DOMPlan) {
			plan.Steps[0].Fields = append(plan.Steps[0].Fields, plan.Steps[0].Fields[0])
		}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			plan := validDOMPlan()
			plan.Steps = append([]DOMStep(nil), plan.Steps...)
			for index := range plan.Steps {
				plan.Steps[index].Fields = append([]DOMField(nil), plan.Steps[index].Fields...)
			}
			testCase.change(&plan)

			_, err := plan.DriverPlan()

			assert.Error(t, err)
		})
	}
}

func TestViewportWidthSelectsTheNarrowViewport(t *testing.T) {
	assert.Equal(t, narrowViewportWidth, DOMStep{Value: "narrow"}.ViewportWidth())
	assert.Equal(t, wideViewportWidth, DOMStep{Value: "wide"}.ViewportWidth())
}
