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
)

func lifecycleSteps(trace *LifecycleTrace) []string {
	names := make([]string, 0, len(trace.States))
	for _, state := range trace.States {
		names = append(names, state.Step)
	}
	return names
}

func TestExpectedLifecycleFollowsTheCompletionOrder(t *testing.T) {
	forward := ExpectedLifecycle(LifecycleInput{Draft: "note"})
	reverse := ExpectedLifecycle(LifecycleInput{Draft: "note", Reverse: true})

	assert.Equal(t, []string{"mount", "click-bound", "edit-and-refresh", "cancel-navigation", "start-requests", "detach-first",
		"release-first", "release-" + lifecycleSecondOwner, "navigate-away", "back-and-dispatch", "click-after-back",
		"link-away", "link-back-and-click", "history-back", "history-forward-and-click"}, lifecycleSteps(forward))
	assert.Equal(t, "release-"+lifecycleSecondOwner, lifecycleSteps(reverse)[6])
	assert.Equal(t, "note", forward.States[2].Draft)
	back := forward.States[9]
	assert.Equal(t, "/journey", back.Path)
	assert.Equal(t, 4, back.Connected)
	assert.Equal(t, 4, back.Deliveries)
	assert.False(t, back.SameRow)
}

func TestExpectedLifecycleCountsEveryPageScriptVisitAndLeave(t *testing.T) {
	trace := ExpectedLifecycle(LifecycleInput{Draft: "note", LayoutLinks: true})
	visits := map[string]int{"/journey": 1, "/journey-away": 0}
	leaves := map[string]int{"/journey": 0, "/journey-away": 0}
	clicks := 0
	previous := trace.States[0]
	for _, state := range trace.States {
		if state.Path != previous.Path {
			visits[state.Path]++
			leaves[previous.Path]++
		}
		if state.Clicks != clicks {
			assert.Equal(t, clicks+1, state.Clicks, "%s clicked the bound element more than once", state.Step)
			assert.Equal(t, "/journey", state.Path, state.Step)
			clicks = state.Clicks
		}
		journey := PageScriptCounts{Connected: visits["/journey"], Disconnected: leaves["/journey"], Cleaned: leaves["/journey"]}
		away := PageScriptCounts{Connected: visits["/journey-away"], Disconnected: leaves["/journey-away"], Cleaned: leaves["/journey-away"]}
		assert.Equal(t, journey, state.Journey, state.Step)
		assert.Equal(t, away, state.Away, state.Step)
		assert.True(t, state.SameDocument, state.Step)
		previous = state
	}
	assert.Equal(t, 4, visits["/journey"])
	assert.Equal(t, 3, visits["/journey-away"])
	assert.Equal(t, 4, clicks)
}
