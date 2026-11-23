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

func TestEventFlagsRotate(t *testing.T) {
	assert.Equal(t, 1, EventFlags(EventsInput{Rotate: 1}, 0))
	assert.Equal(t, 0, EventFlags(EventsInput{Rotate: 1}, 3))
}

func TestExpectedEventsModelsBothOwners(t *testing.T) {
	trace := ExpectedEvents(EventsInput{Label: "tick", Rotate: 0, Delta: 2})

	require.Len(t, trace.States, 1+2+1+2*eventLeafSteps)
	first, last := trace.States[0], trace.States[len(trace.States)-1]
	assert.Equal(t, "mount", first.Step)
	assert.Equal(t, 1, first.Initialisations)
	assert.Equal(t, "right", last.Owner)
	assert.Equal(t, 1, last.Left)
	assert.Equal(t, 1, last.Right)
	assert.Equal(t, 2, last.PK)
	assert.Equal(t, 2*eventLeafSteps, last.Local)
	assert.Equal(t, "tick#20", last.Payload)
	assert.Equal(t, "right-flags-3", last.Step)
}
