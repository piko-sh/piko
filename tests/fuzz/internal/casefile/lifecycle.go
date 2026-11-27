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

const (
	// lifecycleSecondOwner names the second asynchronous request owner.
	lifecycleSecondOwner = "second"

	// lifecycleJourneyPath is the route of the editor journey's page.
	lifecycleJourneyPath = "/journey"

	// lifecycleAwayPath is the route of the page the editor journey leaves for.
	lifecycleAwayPath = "/journey-away"

	// lifecycleWatchers is the number of watch components the journey page renders.
	lifecycleWatchers = 2
)

// LifecycleInput controls the user's draft, the order of two asynchronous completions in
// a single browser document, where the journey's SPA links sit and which journey runs.
type LifecycleInput struct {
	// Draft is the user's unsaved text.
	Draft string

	// Reverse reverses the asynchronous completion order.
	Reverse bool

	// LayoutLinks places the `<piko:a>` links between the journey's pages in the shared
	// layout's navigation rather than in each page's own content.
	LayoutLinks bool

	// InlineScripts writes the page scripts as top-level statements with no function
	// declarations.
	InlineScripts bool

	// ElementReuse selects the element-reuse journey between two structurally identical
	// listing pages in place of the editor journey.
	ElementReuse bool
}

// PageScriptCounts records how often one page script's lifecycle callbacks ran.
type PageScriptCounts struct {
	// Connected counts pk.onConnected calls, which is one per visit to the page.
	Connected int

	// Disconnected counts pk.onDisconnected calls, which is one per leave of the page.
	Disconnected int

	// Cleaned counts the cleanups registered through pk.onCleanup that ran.
	Cleaned int
}

// LifecycleState records public observations at a named point in the journey.
type LifecycleState struct {
	// Step names the journey point.
	Step string

	// Path is the browser route.
	Path string

	// Draft is the user's unsaved text.
	Draft string

	// Focus names the focused control.
	Focus string

	// Applied names the request owners whose result was applied, or in the element-reuse
	// journey the listings whose configuration handled a filter change.
	Applied string

	// ActiveValue is the active request's displayed value.
	ActiveValue string

	// Prompts counts leave prompts.
	Prompts int

	// Connected counts component connections.
	Connected int

	// Cleaned counts component cleanups.
	Cleaned int

	// Deliveries counts subscription deliveries.
	Deliveries int

	// Journey counts the journey page script's lifecycle callbacks.
	Journey PageScriptCounts

	// Away counts the away page script's lifecycle callbacks.
	Away PageScriptCounts

	// Clicks counts the clicks the journey page script's bound element delivered.
	Clicks int

	// SameDocument reports whether the browser document was retained.
	SameDocument bool

	// SameRow reports whether the row element identity was retained.
	SameRow bool
}

// LifecycleTrace records a component's navigation and cleanup journey.
type LifecycleTrace struct {
	// FailedStep identifies the last attempted step on failure.
	FailedStep string `json:",omitempty"`

	// DriverError records the browser driver's failure.
	DriverError string `json:",omitempty"`

	// Input holds the supplied interaction values.
	Input LifecycleInput

	// States holds the ordered interaction snapshots.
	States []LifecycleState
}

// lifecycleModel accumulates the expected states of a lifecycle journey.
type lifecycleModel struct {
	// trace receives each recorded state.
	trace *LifecycleTrace

	// state is the state the next recorded step observes.
	state LifecycleState

	// watchers counts the watch components attached to the page shown.
	watchers int
}

// record appends the current state under a step name.
//
// Takes step (string) which names the journey point.
func (model *lifecycleModel) record(step string) {
	model.state.Step = step
	model.trace.States = append(model.trace.States, model.state)
}

// mountJourney models the first load of the journey page. Its watchers and page script
// connect, and its keyed row is the one later reloads must keep.
func (model *lifecycleModel) mountJourney() {
	model.state.Connected, model.watchers = lifecycleWatchers, lifecycleWatchers
	model.state.Journey.visit()
	model.state.SameRow = true
}

// editAndRequest models the edits, the cancelled leave and the detached asynchronous
// completions of the first visit to the journey page.
//
// Takes input (LifecycleInput) which selects draft text and completion order.
func (model *lifecycleModel) editAndRequest(input LifecycleInput) {
	model.state.Draft, model.state.Focus = input.Draft, "draft"
	model.record("edit-and-refresh")
	model.state.Prompts++
	model.record("cancel-navigation")
	model.state.Deliveries = lifecycleWatchers
	model.record("start-requests")
	model.state.Cleaned++
	model.watchers--
	model.record("detach-first")
	for _, owner := range lifecycleOrder(input.Reverse) {
		if owner == lifecycleSecondOwner {
			model.state.Applied, model.state.ActiveValue = lifecycleSecondOwner, lifecycleSecondOwner
		}
		model.record("release-" + owner)
	}
}

// revisitJourney models SPA round trips through the `<piko:a>` links and then through
// browser history, clicking the journey page's bound element after each return.
func (model *lifecycleModel) revisitJourney() {
	model.leaveJourney()
	model.record("link-away")
	model.returnToJourney()
	model.state.Clicks++
	model.record("link-back-and-click")
	model.leaveJourney()
	model.record("history-back")
	model.returnToJourney()
	model.state.Clicks++
	model.record("history-forward-and-click")
}

// leaveJourney models SPA navigation from the journey page to the away page. The journey
// page's watchers and page script disconnect and the away page script connects.
func (model *lifecycleModel) leaveJourney() {
	model.state.Path, model.state.Focus, model.state.Draft = lifecycleAwayPath, "", ""
	model.state.ActiveValue, model.state.SameRow = "", false
	model.state.Cleaned += model.watchers
	model.watchers = 0
	model.state.Journey.leave()
	model.state.Away.visit()
}

// returnToJourney models SPA navigation from the away page back to the journey page,
// which renders its watchers again and connects a fresh page script instance.
func (model *lifecycleModel) returnToJourney() {
	model.state.Path = lifecycleJourneyPath
	model.state.Connected += lifecycleWatchers
	model.watchers = lifecycleWatchers
	model.state.Away.leave()
	model.state.Journey.visit()
}

// visit records a page script's onConnected for a new visit to its page.
func (counts *PageScriptCounts) visit() {
	counts.Connected++
}

// leave records a page script's onDisconnected and its cleanup when SPA navigation leaves
// its page.
func (counts *PageScriptCounts) leave() {
	counts.Disconnected++
	counts.Cleaned++
}

// ExpectedLifecycle models the selected navigation and cleanup journey.
//
// Takes input (LifecycleInput) which selects the journey, draft text, completion order
// and link placement.
//
// Returns *LifecycleTrace which holds the expected navigation, request and cleanup
// states.
func ExpectedLifecycle(input LifecycleInput) *LifecycleTrace {
	if input.ElementReuse {
		return expectedElementReuse(input)
	}
	return expectedEditorJourney(input)
}

// expectedEditorJourney models the editor journey. The journey has edits, a cancelled
// leave and detached asynchronous completions, then SPA visits between two page scripts
// that share one layout.
//
// Takes input (LifecycleInput) which selects draft text and completion order.
//
// Returns *LifecycleTrace which holds the expected states.
func expectedEditorJourney(input LifecycleInput) *LifecycleTrace {
	model := newLifecycleModel(input, lifecycleJourneyPath)
	model.mountJourney()
	model.record("mount")
	model.state.Clicks++
	model.record("click-bound")
	model.editAndRequest(input)
	model.state.Prompts++
	model.leaveJourney()
	model.record("navigate-away")
	model.returnToJourney()
	model.state.Deliveries += lifecycleWatchers
	model.record("back-and-dispatch")
	model.state.Clicks++
	model.record("click-after-back")
	model.revisitJourney()
	return model.trace
}

// newLifecycleModel starts a journey model on its first page.
//
// Takes input (LifecycleInput) which is recorded with the trace.
// Takes path (string) which is the route of the journey's first page.
//
// Returns *lifecycleModel which holds an empty trace and the initial state.
func newLifecycleModel(input LifecycleInput, path string) *lifecycleModel {
	model := &lifecycleModel{trace: new(LifecycleTrace), state: LifecycleState{}, watchers: 0}
	model.trace.Input = input
	model.state.Path, model.state.SameDocument = path, true
	return model
}

// lifecycleOrder selects the asynchronous completion order.
//
// Takes reverse (bool) which reverses the asynchronous completion order.
//
// Returns []string which lists the request owners in completion order.
func lifecycleOrder(reverse bool) []string {
	if reverse {
		return []string{lifecycleSecondOwner, "first"}
	}
	return []string{"first", lifecycleSecondOwner}
}
