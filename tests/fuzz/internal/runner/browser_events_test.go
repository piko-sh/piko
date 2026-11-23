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
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

func TestEventsBoundariesAndCorruptedObservations(t *testing.T) {
	for rotate := range 4 {
		expected := casefile.ExpectedEvents(casefile.EventsInput{Label: "local <A>", Delta: 2, Rotate: rotate})
		require.Len(t, expected.States, 12, "wrong schedule %+v", expected)
		end := expected.States[len(expected.States)-1]
		assert.Equal(t, 8, end.Local)
		assert.Equal(t, 2, end.PK)
		assert.Equal(t, 1, end.Left)
		assert.Equal(t, 1, end.Right)
		assert.Equal(t, 20, end.LeafSerial)
		assert.Equal(t, 2, end.LeftSerial)
		assert.Equal(t, 12, end.RightSerial)
		assert.Equal(t, 2, strings.Count(end.Log, "pk:leaf;"))
		for _, owner := range []string{"left", "right"} {
			for flag := range 4 {
				seen := 0
				for _, state := range expected.States {
					if state.Step == owner+"-flags-"+string(rune('0'+flag)) {
						seen++
					}
				}
				assert.Equal(t, 1, seen, "%s flag %d coverage", owner, flag)
			}
		}
		for _, mutation := range []struct {
			breakIt func(*casefile.EventState)
			field   string
		}{
			{field: "module-initialisation", breakIt: func(state *casefile.EventState) { state.Initialisations = 2 }},
			{field: "shared-module-counter", breakIt: func(state *casefile.EventState) { state.RightSerial = 2 }},
			{field: "slot-owner", breakIt: func(state *casefile.EventState) { state.SameLeaf = false }},
			{field: "listener-count", breakIt: func(state *casefile.EventState) { state.PK-- }},
			{field: "delivery-order", breakIt: func(state *casefile.EventState) {
				state.Log = strings.Replace(state.Log, "pk:leaf;", "pk:origin;", 1)
			}},
			{field: "pk-event-payload", breakIt: func(state *casefile.EventState) { state.Payload = "" }},
		} {
			corrupt := *expected
			corrupt.States = slices.Clone(expected.States)
			mutation.breakIt(&corrupt.States[11])
			err := checkEvents(&corrupt, expected)
			bad, ok := errors.AsType[eventMismatch](err)
			require.True(t, ok, "expected %s, got %v", mutation.field, err)
			assert.Equal(t, mutation.field, bad.field)
			assert.NotEmpty(t, bad.FailureSubject())
			assert.Equal(t, 11, VerifiedEventSteps(&corrupt, expected))
		}
	}
}
