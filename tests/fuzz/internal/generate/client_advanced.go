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
	"errors"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

const (
	// advancedTrue is the modelled text of a true advanced state value.
	advancedTrue = "true"

	// advancedFalse is the modelled text of a false advanced state value.
	advancedFalse = "false"

	// advancedBranch names the modelled conditional-branch state.
	advancedBranch = "branch"

	// advancedFocus names the modelled editor-focus state.
	advancedFocus = "focus"

	// advancedOpen is the modelled class and phase of the open panel.
	advancedOpen = "open"

	// advancedOpenMarkup is the modelled rendering of the open panel's markup.
	advancedOpenMarkup = "1|strong|open|Open & ready"

	// advancedFamilies is the number of advanced component families.
	advancedFamilies = 4

	// timelineChoices bounds the drawn timeline threshold in seconds.
	timelineChoices = 4

	// formsNumberBase is the smallest number typed into the forms family's number input.
	formsNumberBase = 20

	// visualBranches holds the selected visual component subprogram.
	visualBranches = `<section id="extra-output" :data-phase="state.panel ? 'open' : 'shut'" p-bind:data-note="state.draft">
<div id="extra-branches"><span p-if="state.draft == ''">pristine</span><span p-else-if="state.panel">editing</span><span p-else>hidden</span></div>
<div id="extra-markup" p-html="state.panel ? state.openMarkup : state.closedMarkup"></div>
<ul id="extra-context"><li p-for="row in state.scoped" p-context="row.scope" p-key="'shared'">{{ row.scope }}</li></ul>
</section>`

	// formsTemplate holds the forms family's additional template markup.
	formsTemplate = `<fieldset id="extra-forms"><input type="radio" name="colour" id="extra-red" value="red" p-model="state.colour">` +
		`<input type="radio" name="colour" id="extra-blue" value="blue" p-model="state.colour"><output id="extra-colour">{{ state.colour }}</output>` +
		`<select multiple size="3" id="extra-many" p-model="state.many"><option value="a">A</option><option value="b">B</option><option value="c">C</option></select>` +
		`<output id="extra-chosen">{{ state.many.join(',') }}</output><input type="number" id="extra-number" p-model="state.amount">` +
		`<output id="extra-amount">{{ state.amount }}</output></fieldset>`

	// visualTemplate holds the visual family's additional template markup.
	visualTemplate = `<div id="extra-visual" p-class="{ open: state.panel, shut: !state.panel }" p-style="{opacity:state.panel ? '1' : '0.4'}">` +
		`<div id="extra-panel" p-show="state.panel"><input id="extra-editor" p-ref="editor" p-model="state.draft"></div>` +
		`<button id="extra-focus" p-on:click="focusEditor">edit</button><button id="extra-toggle" p-on:click="togglePanel">toggle</button>` +
		`<ul id="extra-memo"><li class="b" p-for="row in state.memoRows" p-key="row.id" p-memo="row.version">{{ row.label }}</li></ul>` +
		visualBranches + `</div>`

	// visualFields holds the visual family's additional state fields.
	visualFields = `panel:true as boolean, draft:"" as string, openMarkup:"<strong data-kind='open'>Open &amp; ready</strong>", ` +
		`closedMarkup:"<em data-kind='closed'>Closed</em>", scoped:[{scope:"alpha"},{scope:"beta"}], ` +
		`memoRows:[{id:1,version:1,label:"a"},{id:2,version:1,label:"b"}],`

	// visualMethods holds the visual family's additional script functions.
	visualMethods = `
function focusEditor() { this.refs.editor.focus(); }
function togglePanel() { state.panel = !state.panel; state.scoped.reverse(); ` +
		`if (!state.panel) { state.memoRows = [{id:1,version:2,label:"changed"},{id:2,version:1,label:"buffered"}]; } else { state.memoRows[1].version++; } }`

	// nestedLeaf holds the nested family's child component source.
	nestedLeaf = `<template name="fuzz-leaf"><button id="leaf-emit" p-on:click="sendPick">pick</button><output id="leaf-tally">{{ state.total }}</output></template>
<script lang="ts">
const state = {total:0 as number};
pkc.onConnected(() => { pkc.dataset.mounts=String(Number(pkc.dataset.mounts || 0)+1); });
pkc.onDisconnected(() => { pkc.dataset.unmounts=String(Number(pkc.dataset.unmounts || 0)+1); });
function sendPick() { state.total++; this.dispatchEvent(new CustomEvent("item-picked",{detail:{value:state.total},bubbles:true,composed:true})); }
</script>`
)

// clientFixture holds a component extension and its expected interaction.
type clientFixture struct {
	// initial holds initial public component state.
	initial map[string]string

	// name identifies the selected component family.
	name string

	// template holds the family's additional template markup.
	template string

	// fields holds the family's additional state fields.
	fields string

	// methods holds the family's additional script functions.
	methods string

	// attributes holds the family's additional component attributes.
	attributes string

	// timeline holds the family's timeline section.
	timeline string

	// commands lists the ordered browser actions.
	commands []casefile.ClientAction
}

// advancedFixture builds an extended component recipe and its independent expected
// states.
//
// Takes c (*casefile.Case) which receives generated sources and expected results.
// Takes p (Project) which holds the drawn application model.
// Takes variant (int) which selects the base component recipe.
// Takes family (int) which selects the advanced component family.
//
// Returns clientFixture which holds component sections and their modelled interactions.
func advancedFixture(c *casefile.Case, p Project, variant, family int) clientFixture {
	user := variant % 2
	switch family {
	case 0:
		return formsFixture(user, p.Count)
	case 1:
		return visualFixture(user, p.Count)
	case 2:
		c.Expected.AdditionalComponents = []string{"fuzz-leaf"}
		c.Files["components/fuzz-leaf.pkc"] = nestedLeaf
		return nestedFixture(user)
	default:
		return timelineFixture(user, p.Count)
	}
}

// formsFixture builds the radio, multi-select and number-input family.
//
// Takes user (int) which selects the interacting component instance.
// Takes count (int) which seeds the typed number.
//
// Returns clientFixture which holds the family's sections and interactions.
func formsFixture(user, count int) clientFixture {
	number := strconv.Itoa(formsNumberBase + count)
	return clientFixture{
		initial:    map[string]string{"radio": "red", "radio-model": "red", "radio-values": "red,blue", "multi": "c", "multi-model": "c", "number": "7", "number-model": "7"},
		name:       "forms",
		template:   formsTemplate,
		fields:     `colour:"red" as string, many:["c"], amount:7 as number,`,
		methods:    "",
		attributes: "",
		timeline:   "",
		commands: []casefile.ClientAction{
			clientAction(casefile.ChooseRadio, user, ""), clientAction(casefile.SelectMany, user, ""), clientAction(casefile.TypeNumber, user, number),
		},
	}
}

// visualFixture builds the class, style, ref, memo and branch family.
//
// Takes user (int) which selects the interacting component instance.
// Takes count (int) which seeds the edited draft.
//
// Returns clientFixture which holds the family's sections and interactions.
func visualFixture(user, count int) clientFixture {
	return clientFixture{
		initial: map[string]string{
			advancedBranch: "pristine", "phase-attr": advancedOpen, "note-attr": "", "markup": advancedOpenMarkup, "context-order": "alpha,beta",
			"context-identity": advancedTrue, "shown": advancedTrue, "class": advancedOpen, "opacity": "1", "value": "", advancedFocus: advancedFalse,
			"editor-identity": advancedTrue, "memo": "a,b", "row-identity": advancedTrue, "memo-stable": "b",
		},
		name:       "visual",
		template:   visualTemplate,
		fields:     visualFields,
		methods:    visualMethods,
		attributes: "",
		timeline:   "",
		commands: []casefile.ClientAction{
			clientAction(casefile.RefEdit, user, "edit-"+strconv.Itoa(count)), clientAction(casefile.HideMemo, user, ""), clientAction(casefile.ShowRefocus, user, ""),
		},
	}
}

// nestedFixture builds the nested child-event family.
//
// Takes user (int) which selects the interacting component instance.
//
// Returns clientFixture which holds the family's sections and interactions.
func nestedFixture(user int) clientFixture {
	return clientFixture{
		initial:  map[string]string{"leaf-count": "0", "parent-picked": "0", "hooks": "1/0", "leaf-identity": advancedTrue},
		name:     "nested",
		template: `<div id="extra-nested"><slot name="nested-leaf" p-event:item-picked="onPicked($event)"></slot><output id="extra-picked">{{ state.picked }}</output></div>`,
		fields:   `picked:0 as number,`,
		methods: `
function onPicked(event) { state.picked = event.detail.value; }`,
		attributes: "",
		timeline:   "",
		commands: []casefile.ClientAction{
			clientAction(casefile.ChildPick, user, ""), clientAction(casefile.RemountChild, user, ""), clientAction(casefile.ChildPick, user, ""),
		},
	}
}

// timelineFixture builds the animation timeline family.
//
// Takes user (int) which selects the interacting component instance.
// Takes count (int) which seeds the timeline threshold.
//
// Returns clientFixture which holds the family's sections and interactions.
func timelineFixture(user, count int) clientFixture {
	threshold := 1 + count%timelineChoices
	return clientFixture{
		initial:    map[string]string{"timeline-hidden": advancedTrue, "timeline-class": advancedFalse, "timeline-ref": advancedTrue},
		name:       "timeline",
		template:   `<strong id="extra-timeline" p-ref="headline" p-timeline:hidden>ready</strong>`,
		fields:     `time:0 as number,`,
		methods:    "",
		attributes: ` enable="animation"`,
		timeline:   fmt.Sprintf("\n<piko:timeline><piko:at time=\"%ds\"><piko:show ref=\"headline\"/><piko:addclass ref=\"headline\" class=\"played\"/></piko:at></piko:timeline>\n", threshold),
		commands: []casefile.ClientAction{
			clientAction(casefile.SeekTimeline, user, strconv.Itoa(threshold+1)), clientAction(casefile.SeekTimeline, user, "0"),
		},
	}
}

// appendFixtureSteps appends the extended component's modelled interactions.
//
// Takes want (*casefile.ClientTransition) which receives the expected advanced-action
// steps.
// Takes fixture (clientFixture) which holds the extended component sections and model.
//
// Returns error which reports a transition without prior steps, a command naming an
// absent instance or an unreadable modelled counter, or nil on success.
func appendFixtureSteps(want *casefile.ClientTransition, fixture clientFixture) error {
	if len(want.Steps) == 0 {
		return errors.New("advanced component steps need a preceding base step")
	}
	states := want.Steps[len(want.Steps)-1].After
	for _, command := range fixture.commands {
		if command.Instance < 0 || command.Instance >= len(states) {
			return fmt.Errorf("advanced command %s targets instance %d of %d", command.Op, command.Instance, len(states))
		}
		states = slices.Clone(states)
		state := maps.Clone(states[command.Instance].Advanced)
		states[command.Instance].Advanced = state
		if err := applyAdvancedCommand(state, command); err != nil {
			return err
		}
		want.Steps = append(want.Steps, casefile.ClientStep{Action: command, Prevented: false, After: states})
	}
	return nil
}

// applyAdvancedCommand updates one instance's modelled advanced state for a command.
//
// Takes state (map[string]string) which holds the instance's advanced state and is
// updated in place.
// Takes command (casefile.ClientAction) which is the browser action to model.
//
// Returns error which reports an unreadable modelled counter, or nil on success.
func applyAdvancedCommand(state map[string]string, command casefile.ClientAction) error {
	switch command.Op {
	case casefile.ChooseRadio:
		state["radio"], state["radio-model"] = "blue", "blue"
	case casefile.SelectMany:
		state["multi"], state["multi-model"] = "a,b", "a,b"
	case casefile.TypeNumber:
		state["number"], state["number-model"] = command.Value, command.Value
	case casefile.RefEdit:
		state["value"], state[advancedFocus], state[advancedBranch], state["note-attr"] = command.Value, advancedTrue, "editing", command.Value
	case casefile.HideMemo:
		state[advancedBranch], state["phase-attr"], state["markup"], state["context-order"] = "hidden", "shut", "1|em|closed|Closed", "beta,alpha"
		state["shown"], state["class"], state["opacity"] = advancedFalse, "shut", "0.4"
		state[advancedFocus], state["memo"], state["memo-stable"] = advancedFalse, "changed,b", "marked"
	case casefile.ShowRefocus:
		state[advancedBranch], state["phase-attr"], state["markup"], state["context-order"] = "editing", advancedOpen, advancedOpenMarkup, "alpha,beta"
		state["shown"], state["class"], state["opacity"] = advancedTrue, advancedOpen, "1"
		state[advancedFocus], state["memo"], state["memo-stable"] = advancedTrue, "changed,buffered", "b"
	case casefile.ChildPick:
		count, err := strconv.Atoi(state["leaf-count"])
		if err != nil {
			return fmt.Errorf("reading modelled leaf count: %w", err)
		}
		state["leaf-count"], state["parent-picked"] = strconv.Itoa(count+1), strconv.Itoa(count+1)
	case casefile.RemountChild:
		state["hooks"] = "2/1"
	case casefile.SeekTimeline:
		state["timeline-hidden"], state["timeline-class"] = strconv.FormatBool(command.Value == "0"), strconv.FormatBool(command.Value != "0")
	default:
	}
	return nil
}

// extendComponent adds the selected fixture's component sections.
//
// Takes source (string) which holds the saved source text.
// Takes fixture (clientFixture) which holds the extended component sections and model.
//
// Returns string which is the source with the selected extension added.
func extendComponent(source string, fixture clientFixture) string {
	source = strings.Replace(source, `<template name="fuzz-counter">`, `<template name="fuzz-counter"`+fixture.attributes+">", 1)
	source = strings.Replace(source, "</div></template>", fixture.template+"</div></template>", 1)
	source = strings.Replace(source, "const state = { ", "const state = { "+fixture.fields, 1)
	return strings.Replace(source, "</script>", fixture.methods+"\n</script>", 1) + fixture.timeline
}
