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
	"fmt"
)

const (
	// eventFlagValues is the number of distinct event propagation flag values.
	eventFlagValues = 4

	// eventLeafSteps is the number of leaf events dispatched under each owner.
	eventLeafSteps = 4

	// eventBubbleFlags is the flag value at which a leaf event reaches its owner and the
	// page.
	eventBubbleFlags = 3
)

// EventsInput selects values for the component event journey.
type EventsInput struct {
	// Label is the displayed label.
	Label string

	// Rotate offsets the event propagation flags.
	Rotate int

	// Delta is the counter increment applied by each event.
	Delta int
}

// EventState records the component state after one event step.
type EventState struct {
	// Step names the event step.
	Step string

	// Owner names the component that owns the leaf.
	Owner string

	// Log records the handled event sequence.
	Log string

	// Payload is the last delivered event payload.
	Payload string

	// Initialisations counts leaf initialisations.
	Initialisations int `json:"Initializations"`

	// Local counts events handled by the leaf itself.
	Local int

	// Left counts events handled by the left owner.
	Left int

	// Right counts events handled by the right owner.
	Right int

	// PK counts events handled by the page.
	PK int

	// LeafSerial is the leaf's last event serial.
	LeafSerial int

	// LeftSerial is the left owner's last event serial.
	LeftSerial int

	// RightSerial is the right owner's last event serial.
	RightSerial int

	// SameLeaf reports whether the leaf retained its identity.
	SameLeaf bool
}

// EventsTrace records the component event journey.
type EventsTrace struct {
	// BrowserPath records the browser's final route.
	BrowserPath string `json:",omitempty"`

	// DriverError records the browser driver's failure.
	DriverError string `json:",omitempty"`

	// States holds the ordered interaction snapshots.
	States []EventState

	// Input holds the supplied interaction values.
	Input EventsInput
}

// EventFlags calculates the event journey's expected propagation flags.
//
// Takes input (EventsInput) which selects the component event journey values.
// Takes index (int) which is the zero-based item index.
//
// Returns int which is the modelled event propagation flag value.
func EventFlags(input EventsInput, index int) int { return (input.Rotate + index) % eventFlagValues }

// ExpectedEvents models the component event journey.
//
// Takes input (EventsInput) which selects the component event journey values.
//
// Returns *EventsTrace which holds the independently expected component event states.
func ExpectedEvents(input EventsInput) *EventsTrace {
	trace := &EventsTrace{BrowserPath: "", Input: input, States: nil, DriverError: ""}
	state := EventState{
		Step:            "",
		Owner:           "left",
		Log:             "",
		Payload:         "",
		Initialisations: 1,
		Local:           0,
		Left:            0,
		Right:           0,
		PK:              0,
		LeafSerial:      0,
		LeftSerial:      0,
		RightSerial:     0,
		SameLeaf:        true,
	}
	counter := 0
	add := func(name string) { state.Step = name; trace.States = append(trace.States, state) }
	add("mount")
	for ownerIndex, owner := range []string{"left", "right"} {
		if ownerIndex == 1 {
			state.Owner = owner
			add("move-right")
		}
		counter += input.Delta
		if ownerIndex == 0 {
			state.LeftSerial = counter
		} else {
			state.RightSerial = counter
		}
		add("tick-" + owner)
		for index := range eventLeafSteps {
			flags := EventFlags(input, index)
			counter += input.Delta
			state.LeafSerial = counter
			state.Local++
			state.Log += "local:origin;"
			if flags == eventBubbleFlags {
				if ownerIndex == 0 {
					state.Left++
				} else {
					state.Right++
				}
				state.PK++
				state.Log += owner + ":leaf;pk:leaf;"
				state.Payload = fmt.Sprintf("%s#%d", input.Label, counter)
			}
			add(fmt.Sprintf("%s-flags-%d", owner, flags))
		}
	}
	return trace
}
