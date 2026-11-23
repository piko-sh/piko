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

package generate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

func TestShippedPlanHasIndependentRegisteredFamilies(t *testing.T) {
	c := New("shipped-components", 9, 0)
	plan, err := c.Expected.DOM.DriverPlan()
	require.NoError(t, err)
	require.Len(t, plan.Steps, 19)
	for _, step := range plan.Steps {
		for _, field := range step.Fields {
			require.Nil(t, field.Want, "driver received an independent expectation")
		}
	}
	for _, viewport := range []string{"wide", "narrow"} {
		step := casefile.DOMStep{ID: "size", Op: "viewport", Value: viewport, Fields: []casefile.DOMField{{ID: "frame", Query: casefile.DOMQuery{Selector: "#frame", Read: "tag"}}}}
		_, err := (casefile.DOMPlan{Path: "/shipped", ReadyElement: "#frame", Steps: []casefile.DOMStep{step}}).DriverPlan()
		require.NoError(t, err)
	}
	modified := *c.Expected.DOM
	modified.Steps = []casefile.DOMStep{modified.Steps[13]}
	modified.Steps[0].Value = "12345"
	_, err = modified.DriverPlan()
	require.Error(t, err, "out-of-range viewport accepted")
	assert.Contains(t, c.Files["pages/shipped.pk"], "Component gallery")
}
