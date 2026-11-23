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

func TestExpectedLifecycleFollowsTheCompletionOrder(t *testing.T) {
	forward := ExpectedLifecycle(LifecycleInput{Draft: "note"})
	reverse := ExpectedLifecycle(LifecycleInput{Draft: "note", Reverse: true})

	steps := func(trace *LifecycleTrace) []string {
		names := make([]string, 0, len(trace.States))
		for _, state := range trace.States {
			names = append(names, state.Step)
		}
		return names
	}
	assert.Equal(t, []string{"mount", "edit-and-refresh", "cancel-navigation", "start-requests", "detach-first", "release-first", "release-" + lifecycleSecondOwner, "navigate-away", "back-and-dispatch"}, steps(forward))
	assert.Equal(t, "release-"+lifecycleSecondOwner, steps(reverse)[5])
	assert.Equal(t, "note", forward.States[1].Draft)
	last := forward.States[len(forward.States)-1]
	assert.Equal(t, "/journey", last.Path)
	assert.Equal(t, lifecycleReturnConnections, last.Connected)
	assert.Equal(t, 4, last.Deliveries)
	assert.False(t, last.SameRow)
}
