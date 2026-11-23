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
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

func TestComponentsFirstWindowCoversAllInteractionRecipes(t *testing.T) {
	for _, seed := range []uint64{0, 7, 99} {
		sequences := map[string]bool{}
		pairs := map[string]bool{}
		for index := range uint64(64) {
			c := New("components", seed, index)
			replayed := Replay(c.Target, c.Seed, c.Index, c.Data)
			require.Equal(t, c.Hash(), replayed.Hash(), "seed %d index %d", seed, index)
			steps := c.Expected.Client.Steps
			require.GreaterOrEqual(t, len(steps), 11, "expected 9 shared and 2 to 3 family-specific steps")
			require.LessOrEqual(t, len(steps), 12, "expected 9 shared and 2 to 3 family-specific steps")
			var sequence strings.Builder
			for _, step := range steps[:9] {
				fmt.Fprintf(&sequence, "%s/%d ", step.Action.Op, step.Action.Instance)
			}
			if index < 16 {
				sequences[sequence.String()] = true
			}
			fmt.Fprintf(&sequence, "/%s", steps[9].Action.Op)
			pairs[sequence.String()] = true
		}
		assert.Len(t, pairs, 64, "seed %d base recipe and advanced family combinations", seed)
		assert.Len(t, sequences, 16, "seed %d order and instance recipes", seed)
	}
}

func TestModelClientStepsRejectsUnrepresentableStates(t *testing.T) {
	controls := &casefile.ClientControls{Form: formState(false, "middle")}
	testCases := []struct {
		name  string
		after []casefile.ClientInstance
	}{
		{name: "missing instance", after: []casefile.ClientInstance{{Controls: controls}}},
		{name: "missing controls", after: []casefile.ClientInstance{{}, {}}},
		{name: "empty rows", after: []casefile.ClientInstance{{Controls: controls}, {Controls: controls}}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			transition := casefile.ClientTransition{After: testCase.after}
			err := modelClientSteps(&transition, Project{Title: "x"}, 0)
			require.Error(t, err)
		})
	}
}

func TestAppendFixtureStepsRejectsUnrepresentableStates(t *testing.T) {
	t.Run("no base step", func(t *testing.T) {
		require.Error(t, appendFixtureSteps(&casefile.ClientTransition{}, clientFixture{}))
	})
	t.Run("absent instance", func(t *testing.T) {
		transition := casefile.ClientTransition{Steps: []casefile.ClientStep{{After: []casefile.ClientInstance{{}}}}}
		fixture := clientFixture{commands: []casefile.ClientAction{{Op: casefile.ChildPick, Instance: 3}}}
		require.Error(t, appendFixtureSteps(&transition, fixture))
	})
	t.Run("unreadable counter", func(t *testing.T) {
		transition := casefile.ClientTransition{Steps: []casefile.ClientStep{{After: []casefile.ClientInstance{{Advanced: map[string]string{"leaf-count": "many"}}}}}}
		fixture := clientFixture{commands: []casefile.ClientAction{{Op: casefile.ChildPick, Instance: 0}}}
		require.ErrorContains(t, appendFixtureSteps(&transition, fixture), "leaf count")
	})
	t.Run("counter advances", func(t *testing.T) {
		transition := casefile.ClientTransition{Steps: []casefile.ClientStep{{After: []casefile.ClientInstance{{Advanced: map[string]string{"leaf-count": "1"}}}}}}
		fixture := clientFixture{commands: []casefile.ClientAction{{Op: casefile.ChildPick, Instance: 0}}}
		require.NoError(t, appendFixtureSteps(&transition, fixture))
		require.Len(t, transition.Steps, 2)
		assert.Equal(t, "2", transition.Steps[1].After[0].Advanced["parent-picked"])
		assert.Equal(t, "1", transition.Steps[0].After[0].Advanced["leaf-count"])
	})
}
