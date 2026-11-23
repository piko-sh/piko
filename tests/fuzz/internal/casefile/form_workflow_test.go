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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormWorkflowHistoriesAndSmallerCases(t *testing.T) {
	for _, second := range []bool{false, true} {
		workflow := NewFormWorkflow("first", "second", second)
		history, err := workflow.Model()
		require.NoError(t, err)
		remaining := FormRow{ID: 1, Label: "first", Done: true}
		if second {
			remaining = FormRow{ID: 2, Label: "second", Done: true}
		}
		require.Len(t, history, 8)
		assert.Empty(t, history[7].Rows)
		assert.Equal(t, 5, history[7].Revision)
		assert.Equal(t, "first", history[0].Input)
		assert.Empty(t, history[1].Input)
		assert.Equal(t, 1, history[2].InvalidEvents)
		assert.Equal(t, []FormRow{remaining}, history[6].Rows)
		assert.Equal(t, 4, history[6].Revision)
		for _, shorter := range ReduceFormWorkflow(workflow) {
			assert.Less(t, len(shorter.Steps), len(workflow.Steps), "non-decreasing recipe")
			_, err := shorter.Model()
			require.NoError(t, err)
		}
	}
}

func TestFormWorkflowModelRejectsInvalidHistories(t *testing.T) {
	testCases := []struct {
		name     string
		workflow *FormWorkflow
		message  string
	}{
		{name: "nil", workflow: nil, message: "form operations"},
		{name: "unknown operation", workflow: &FormWorkflow{Steps: []FormCommand{{Op: "launch"}}}, message: "unknown form operation"},
		{name: "absent row", workflow: &FormWorkflow{Steps: []FormCommand{{Op: "delete", ID: 3}}}, message: "absent task"},
		{name: "control text", workflow: &FormWorkflow{Steps: []FormCommand{{Op: "edit", Value: "a\nb"}}}, message: "invalid text"},
		{name: "missing label", workflow: &FormWorkflow{Steps: []FormCommand{{Op: "create"}}}, message: "missing create label"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := testCase.workflow.Model()
			require.ErrorContains(t, err, testCase.message)
		})
	}
}
