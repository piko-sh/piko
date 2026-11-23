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

// PartialReloadState records component and form state across reload modes.
type PartialReloadState struct {
	// RootStage is the reload mode last applied to the root component.
	RootStage string

	// ChildStage is the reload mode last applied to the child component.
	ChildStage string

	// LiveOnly is the client-only state that survives the reload, or empty when replaced.
	LiveOnly string

	// Preserved is the explicitly preserved state value.
	Preserved string

	// Draft is the unsaved input text.
	Draft string

	// Default is the input's default value.
	Default string

	// Selection records the input's selection start, end and direction.
	Selection string

	// Result is the rendered reload result.
	Result string

	// RootRetained reports whether the root element identity was retained.
	RootRetained bool

	// InputRetained reports whether the input element identity was retained.
	InputRetained bool

	// Focused reports whether the input kept focus.
	Focused bool

	// SiblingRetained reports whether the sibling element identity was retained.
	SiblingRetained bool

	// SiblingUntouched reports whether the sibling's state was left untouched.
	SiblingUntouched bool

	// Slotted reports whether the slotted content was preserved.
	Slotted bool
}

// ReloadOrder lists the saved reload modes in execution order.
//
// Returns []string which lists reload modes in execution order.
func (in PartialComponentsInput) ReloadOrder() []string {
	if in.ReverseReload {
		return []string{"attrs-only", "children-only", "replace", "merge"}
	}
	return []string{"merge", "replace", "children-only", "attrs-only"}
}

// expectedReloadModes models component and form state after each reload mode.
//
// Takes in (PartialComponentsInput) which selects the partial-component journey values.
//
// Returns []PartialComponentState which lists partial component state values in their
// recorded order.
func expectedReloadModes(in PartialComponentsInput) []PartialComponentState {
	order := in.ReloadOrder()
	states := make([]PartialComponentState, 0, len(order))
	root, child := "initial", "initial"
	for _, mode := range order {
		if mode != "children-only" {
			root = mode
		}
		if mode != "attrs-only" {
			child = mode
		}
		live := "local-" + mode
		if mode == "replace" {
			live = ""
		}
		state := partialState("reload-"+mode, "/reload-history", "", 0)
		state.Reload = &PartialReloadState{
			RootStage: root, ChildStage: child, Default: child,
			LiveOnly: live, Preserved: "keep-" + mode,
			Draft: "abc-" + in.Edited, Selection: "1:3:backward", Result: mode,
			RootRetained: true, InputRetained: true, Focused: true,
			SiblingRetained: true, SiblingUntouched: true, Slotted: true,
		}
		states = append(states, state)
	}
	return states
}
