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
	"net/url"
	"strconv"
)

const (
	// partialEditedScoreBonus is added to the selected row's score once it is edited.
	partialEditedScoreBonus = 10

	// partialRestoredIdentity is the component identity of the restored sibling row.
	partialRestoredIdentity = 3

	// partialRowIcon is the expected rendered icon reference of every row.
	partialRowIcon = "#example.org/pikofuzzcase/lib/record-icons/arrow.svg|0 0 10 10|0 0 10 10|1,1|9,9|11314"
)

// PartialComponentsInput selects values for the partial-component journey.
type PartialComponentsInput struct {
	// Labels holds the fixture's row labels.
	Labels [2]string

	// Edited is the replacement label applied to the selected row.
	Edited string

	// Query is the detail route's search query.
	Query string

	// Locale selects the page locale.
	Locale string

	// Score is the row's score.
	Score int

	// EditSecond selects the row edited by the workflow.
	EditSecond bool

	// ReloadModes enables reload-mode coverage.
	ReloadModes bool `json:",omitempty"`

	// ReverseReload reverses the reload-mode execution order.
	ReverseReload bool `json:",omitempty"`
}

// PartialComponentRow records one rendered row and its component identity.
type PartialComponentRow struct {
	// Key identifies the row.
	Key string

	// Label is the row's displayed label.
	Label string

	// Shadow is the label rendered inside the row component's shadow root.
	Shadow string

	// Caption is the row's displayed caption.
	Caption string

	// Link is the row's detail link.
	Link string

	// Name is the detail link's accessible name.
	Name string

	// Icon is the row's rendered icon reference.
	Icon string

	// Score is the row's displayed score.
	Score int

	// ShadowScore is the score rendered inside the row component's shadow root.
	ShadowScore int

	// Count is the row's click count.
	Count int

	// HostIdentity is the row component host's identity marker.
	HostIdentity int

	// RowIdentity is the row element's identity marker.
	RowIdentity int
}

// PartialComponentState records the partial-component page after one step.
type PartialComponentState struct {
	// Reload holds reload-mode evidence.
	Reload *PartialReloadState `json:",omitempty"`

	// Step names the journey step.
	Step string

	// Path is the browser route.
	Path string

	// Locale is the page locale.
	Locale string

	// Log records the handled row clicks.
	Log string

	// Detail is the detail page's rendered summary.
	Detail string

	// Rows holds the ordered row values.
	Rows []PartialComponentRow

	// Revision is the rendered persistence revision.
	Revision int

	// ActionRevision is the revision reported by the last action, or -1 when not applicable.
	ActionRevision int
}

// PartialComponentsTrace records the partial-component journey.
type PartialComponentsTrace struct {
	// DriverError records the browser driver's failure.
	DriverError string `json:",omitempty"`

	// States holds the ordered interaction snapshots.
	States []PartialComponentState

	// Input holds the supplied interaction values.
	Input PartialComponentsInput
}

// partialJourney tracks expected page state across partial-component interactions.
type partialJourney struct {
	// out receives the modelled page states.
	out *PartialComponentsTrace

	// keys identifies the two fixture rows.
	keys [2]string

	// labels holds the rows' current labels.
	labels [2]string

	// log records the handled row clicks.
	log string

	// order lists the displayed rows by fixture index.
	order []int

	// in selects the partial-component journey values.
	in PartialComponentsInput

	// identities holds the rows' current component identities.
	identities [2]int

	// counts holds the rows' click counts.
	counts [2]int

	// selected is the fixture index of the edited row.
	selected int

	// revision is the rendered persistence revision.
	revision int
}

// ListPath formats the partial-component list route.
//
// Returns string which is the list route for the selected locale.
func (in PartialComponentsInput) ListPath() string {
	if in.Locale == "fr" {
		return "/fr/record-list"
	}
	return "/record-list"
}

// DetailPath formats the selected row's detail route.
//
// Takes key (string) which identifies the requested field or row.
//
// Returns string which is the selected row's detail route.
func (in PartialComponentsInput) DetailPath(key string) string {
	prefix := ""
	if in.Locale == "fr" {
		prefix = "/fr"
	}
	return prefix + "/record-detail?key=" + key + "&q=" + url.QueryEscape(in.Query) + "#summary"
}

// page records the list page as it renders after one journey step.
//
// Takes step (string) which names the journey step.
func (journey *partialJourney) page(step string) {
	in := journey.in
	state := partialState(step, in.ListPath(), in.Locale, journey.revision)
	state.ActionRevision = journey.revision
	state.Log = journey.log
	for _, i := range journey.order {
		score := in.Score + i
		if journey.revision >= 1 && i == journey.selected {
			score += partialEditedScoreBonus
		}
		state.Rows = append(state.Rows, partialRow(in, journey.keys[i], journey.labels[i], score, journey.counts[i], journey.identities[i]))
	}
	journey.out.States = append(journey.out.States, state)
}

// click records one click on a row.
//
// Takes i (int) which is the clicked row's fixture index.
func (journey *partialJourney) click(i int) {
	journey.counts[i]++
	journey.log += journey.keys[i] + "=" + journey.labels[i] + ":" + strconv.Itoa(journey.counts[i]) + ";"
}

// ExpectedPartialComponents models the partial-component journey.
//
// Takes in (PartialComponentsInput) which selects the partial-component journey values.
//
// Returns *PartialComponentsTrace which holds expected row, reload and navigation states.
func ExpectedPartialComponents(in PartialComponentsInput) *PartialComponentsTrace {
	selected := 0
	if in.EditSecond {
		selected = 1
	}
	other := 1 - selected
	journey := &partialJourney{
		out:        &PartialComponentsTrace{Input: in, States: nil, DriverError: ""},
		in:         in,
		keys:       [2]string{"amber", "blue"},
		labels:     in.Labels,
		log:        "",
		order:      []int{0, 1},
		identities: [2]int{1, 2},
		counts:     [2]int{},
		selected:   selected,
		revision:   0,
	}
	journey.page("mount")
	journey.click(selected)
	journey.page("select")
	journey.revision++
	journey.labels[selected] = in.Edited
	journey.page("edit")
	journey.click(selected)
	journey.page("select-edited")
	journey.revision++
	journey.order = []int{1, 0}
	journey.page("reorder")
	journey.click(other)
	journey.page("select-sibling")
	journey.revision++
	journey.order = []int{selected}
	journey.page("remove")
	journey.page("detached-event")
	journey.revision++
	journey.order = []int{1, 0}
	journey.identities[other] = partialRestoredIdentity
	journey.counts[other] = 0
	journey.page("restore")
	journey.click(other)
	journey.page("select-restored")
	key := journey.keys[other]
	detail := partialState("detail", in.DetailPath(key), in.Locale, journey.revision)
	detail.ActionRevision = -1
	detail.Detail = fmt.Sprintf("%s / %s / %s", key, journey.labels[other], in.Query)
	journey.out.States = append(journey.out.States, detail)

	journey.page("back")
	if in.ReloadModes {
		journey.out.States = append(journey.out.States, expectedReloadModes(in)...)
	}
	return journey.out
}

// partialState builds a partial-component page state without rows or evidence.
//
// Takes step (string) which names the journey step.
// Takes path (string) which is the browser route.
// Takes locale (string) which is the page locale.
// Takes revision (int) which is the rendered persistence revision.
//
// Returns PartialComponentState which holds the step identity and zero evidence.
func partialState(step, path, locale string, revision int) PartialComponentState {
	return PartialComponentState{
		Step:           step,
		Path:           path,
		Locale:         locale,
		Log:            "",
		Detail:         "",
		Revision:       revision,
		ActionRevision: 0,
		Rows:           nil,
		Reload:         nil,
	}
}

// partialRow models one rendered partial-component row.
//
// Takes in (PartialComponentsInput) which selects the detail route's locale and query.
// Takes key (string) which identifies the row.
// Takes label (string) which is the row's displayed label.
// Takes score (int) which is the row's displayed score.
// Takes count (int) which is the row's click count.
// Takes identity (int) which is the row's component and element identity.
//
// Returns PartialComponentRow which holds the expected rendered row.
func partialRow(in PartialComponentsInput, key, label string, score, count, identity int) PartialComponentRow {
	return PartialComponentRow{
		Key:          key,
		Label:        label,
		Shadow:       label,
		Caption:      key + " / " + label,
		Link:         in.DetailPath(key),
		Name:         "Open " + key,
		Icon:         partialRowIcon,
		Score:        score,
		ShadowScore:  score,
		Count:        count,
		HostIdentity: identity,
		RowIdentity:  identity,
	}
}
