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
	"slices"
	"strconv"
	"strings"

	"piko.sh/fuzzkit/tape"
	"piko.sh/safeconv"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

const (
	// counterComponentPath is the generated counter component's source path.
	counterComponentPath = "components/fuzz-counter.pkc"

	// targetFormatter names the formatter target.
	targetFormatter = "formatter"

	// clientStepChoices bounds the drawn counter increment.
	clientStepChoices = 3

	// clientImportOdds is the denominator of the odds of a locally imported increment.
	clientImportOdds = 3

	// clientVariants is the number of base interaction recipes.
	clientVariants = 16

	// clientRecipeIndexes is the number of leading component cases that enumerate recipes.
	clientRecipeIndexes = 64

	// timelineFamily selects the animation timeline family.
	timelineFamily = 3

	// formatterTimelineBase is the formatter timeline's minimum media width in pixels.
	formatterTimelineBase = 600

	// formatterTimelineStep is the formatter timeline's media width step in pixels.
	formatterTimelineStep = 10

	// clientSwapEntries is the recipe bit that swaps the input and attribute entries.
	clientSwapEntries = 2

	// clientSwapRows is the recipe bit that swaps row reversal and removal.
	clientSwapRows = 4

	// clientSwapClicks is the recipe bit that swaps the modified and parent clicks.
	clientSwapClicks = 8

	// clientRowOne is the first row of the component's keyed list.
	clientRowOne = "one"

	// clientRowTwo is the second row of the component's keyed list.
	clientRowTwo = "two"

	// clientMiddleChoice is the select control's initial option.
	clientMiddleChoice = "middle"

	// sequenceControls holds the multi-step recipe's input and click controls.
	sequenceControls = `<input id="client-input" p-model="state.label"><button id="client-reverse" p-on:click="reverseRows">reverse</button>` +
		`<button id="client-remove" p-on:click="removeRow">remove</button><div id="client-parent" p-on:click="parentClick">` +
		`<button id="client-modified" p-on:click.prevent.stop="modifiedClick">modified</button></div>` +
		`<output id="client-handled">{{ state.handled }}</output><output id="client-bubbled">{{ state.bubbled }}</output>`

	// sequenceForm holds the multi-step recipe's checkbox and select controls.
	sequenceForm = `<input type="checkbox" id="client-check" p-model="state.agreed"><output id="client-agreed">{{ state.agreed ? 'on' : 'off' }}</output>` +
		`<select id="client-select" p-model="state.choice"><option value="first">First</option><option value="middle">Middle</option>` +
		`<option value="last">Last</option></select><output id="client-choice">{{ state.choice }}</output>` +
		`<button id="client-reset" p-on:click="resetControls">reset</button>`

	// sequenceMethods holds the multi-step recipe's script functions before the reset.
	sequenceMethods = `
function reverseRows() { state.items.reverse(); }
function removeRow() { state.items.shift(); }
function modifiedClick() { state.handled++; }
function parentClick() { state.bubbled++; }`

	// componentTemplate formats the counter component from its body, imports, state fields,
	// connection hook, increment statements and methods.
	componentTemplate = "<template name=\"fuzz-counter\"><div>%s</div></template>\n<script lang=\"ts\">\n%s\n" +
		"const state = { %s };\n%s\nfunction increment() { %s }%s\n</script>\n"
)

// clientModel selects the generated component's structure and interactions.
type clientModel struct {
	// instances is the number of sibling component instances.
	instances int

	// step is the component counter increment.
	step int

	// importStep moves the increment function into a local import.
	importStep bool

	// attributes enables reflected component attributes.
	attributes bool

	// list enables the keyed component list.
	list bool

	// conditional enables conditional component content.
	conditional bool

	// connected enables the component connection callback.
	connected bool

	// sequence enables the multi-step component recipe.
	sequence bool

	// variant selects the base interaction recipe.
	variant int

	// family selects the advanced component family.
	family int
}

// drawClient replays the component choices following the base project draws.
//
// Takes data ([]byte) which holds the replayable choice bytes.
//
// Returns clientModel which holds the drawn component structure.
func drawClient(data []byte) clientModel {
	t := tape.Replay(&tape.Recording{Data: data})
	draw(t)
	instances := 1 + safeconv.Uint64ToInt(t.Uint(2))
	step := 1 + safeconv.Uint64ToInt(t.Uint(clientStepChoices))
	importStep := t.Bool(1, clientImportOdds)
	attributes, list := t.Bool(1, 2), t.Bool(1, 2)
	conditional, connected := t.Bool(1, 2), t.Bool(1, 2)
	return clientModel{
		instances:   instances,
		step:        step,
		importStep:  importStep,
		attributes:  attributes,
		list:        list,
		conditional: conditional,
		connected:   connected,
		sequence:    false,
		variant:     safeconv.Uint64ToInt(t.Uint(clientVariants)),
		family:      safeconv.Uint64ToInt(t.Uint(advancedFamilies)),
	}
}

// emitComponent adds component sources and expectations and returns host markup.
//
// Takes c (*casefile.Case) which receives generated sources and expected results.
// Takes p (Project) which holds the drawn application model.
//
// Returns string which is the markup instantiating the generated component.
// Returns error which reports an unrepresentable interaction model, or nil on success.
func emitComponent(c *casefile.Case, p Project) (string, error) {
	model := drawClient(c.Data)
	extra := selectFixture(c, p, &model)
	wanted, hosts := clientInstances(p, model, extra)
	if model.sequence {
		if err := modelClientSteps(&wanted, p, model.variant); err != nil {
			return "", err
		}
		if err := appendFixtureSteps(&wanted, extra); err != nil {
			return "", err
		}
		c.Features = append(c.Features, "client-extra-"+extra.name,
			"client-model-input", "client-attribute-reflection", "client-keyed-reorder-remove", "client-event-modifiers", "client-multistep", "client-model-checkbox", "client-model-select")
	}
	c.Expected.Client = &wanted
	c.Files[counterComponentPath] = componentSource(c, model, p)
	if extra.name != "" {
		c.Files[counterComponentPath] = extendComponent(c.Files[counterComponentPath], extra)
	}
	if c.Target == "syntax" && model.connected {
		c.Files[counterComponentPath] = strings.Replace(c.Files[counterComponentPath], `<template name="fuzz-counter"`, "<template", 1)
		c.Features = append(c.Features, "filename-component-name")
	}
	if model.instances > 1 {
		c.Features = append(c.Features, "sibling-instances")
	}
	return hosts, nil
}

// selectFixture selects the advanced component extension for the target, if any.
//
// Takes c (*casefile.Case) which receives generated sources and features.
// Takes p (Project) which holds the drawn application model.
// Takes model (*clientModel) which holds the drawn component structure and is adjusted
// for the components target.
//
// Returns clientFixture which holds the selected extension, or the zero fixture.
func selectFixture(c *casefile.Case, p Project, model *clientModel) clientFixture {
	var extra clientFixture
	if c.Target == "components" {
		model.sequence, model.list, model.attributes = true, true, true
		model.instances = 2

		if c.Index < clientRecipeIndexes {
			model.variant = safeconv.Uint64ToInt((c.Seed ^ c.Index) & (clientVariants - 1))
			model.family = safeconv.Uint64ToInt((c.Seed + c.Index + c.Index/clientVariants) & (advancedFamilies - 1))
		}
		extra = advancedFixture(c, p, model.variant, model.family)
		c.Features = append(c.Features, fmt.Sprintf("client-recipe-%02d", model.variant))
	}
	if c.Target == targetFormatter && model.family == timelineFamily {
		extra = advancedFixture(c, p, model.variant, model.family)
		extra.template += `<span id="extra-caption" p-ref="caption" p-timeline:anchor="headline" p-timeline:hidden>caption</span>`
		width := formatterTimelineBase + formatterTimelineStep*p.Count
		extra.timeline += fmt.Sprintf(`<piko:timeline media="(min-width: %dpx)"><piko:at time="500ms"><piko:hide ref="headline"/><piko:show ref="caption"/></piko:at></piko:timeline>`+"\n", width)
		c.Features = append(c.Features, "formatter-timeline")
	}
	return extra
}

// clientInstances models each component instance before and after its first click.
//
// Takes p (Project) which holds the drawn application model.
// Takes model (clientModel) which holds the drawn component structure.
// Takes extra (clientFixture) which holds the selected advanced extension.
//
// Returns casefile.ClientTransition which holds the modelled instance states.
// Returns string which is the markup instantiating every instance.
func clientInstances(p Project, model clientModel, extra clientFixture) (casefile.ClientTransition, string) {
	wanted := casefile.ClientTransition{}
	var hosts strings.Builder
	for instance := range model.instances {
		before := clientBefore(p, model, extra, instance)
		hosts.WriteString("<fuzz-counter")
		if model.attributes {
			fmt.Fprintf(&hosts, ` label="%s"`, before.Label)
		}
		hosts.WriteByte('>')
		if extra.name == "nested" {
			hosts.WriteString(`<fuzz-leaf slot="nested-leaf"></fuzz-leaf>`)
		}
		if p.Slot {
			fmt.Fprintf(&hosts, `<span slot="caption">%s</span>`, before.Slot)
		}
		hosts.WriteString("</fuzz-counter>")
		wanted.Before = append(wanted.Before, before)
		after := before
		if instance == 0 {
			after = clientAfter(p, model, before)
		}
		wanted.After = append(wanted.After, after)
	}
	return wanted, hosts.String()
}

// clientBefore models one component instance's state before any interaction.
//
// Takes p (Project) which holds the drawn application model.
// Takes model (clientModel) which holds the drawn component structure.
// Takes extra (clientFixture) which holds the selected advanced extension.
// Takes instance (int) which is the instance's zero-based position.
//
// Returns casefile.ClientInstance which holds the modelled initial state.
func clientBefore(p Project, model clientModel, extra clientFixture, instance int) casefile.ClientInstance {
	var before casefile.ClientInstance
	before.Count = strconv.Itoa(p.Count)
	if model.attributes {
		before.Label = fmt.Sprintf("instance-%d", instance+1)
	}
	if p.Slot {
		before.Slot = fmt.Sprintf("caption-%d", instance+1)
	}
	if model.list {
		before.Rows = []string{clientRowOne, clientRowTwo}
	}
	if model.conditional {
		before.Visible = p.Show
	}
	if model.connected {
		before.Lifecycle = "ready"
	}
	if model.sequence {
		before.Advanced = extra.initial
		before.Rows = []string{"zero", clientRowOne, clientRowTwo}
		before.Controls = &casefile.ClientControls{
			Form:      formState(p.Show, clientMiddleChoice),
			Input:     before.Label,
			HostLabel: before.Label,
			Handled:   "0",
			Bubbled:   "0",
			Keys:      rowKeys(len(before.Rows)),
		}
	}
	return before
}

// clientAfter models the first instance's state after its increment click.
//
// Takes p (Project) which holds the drawn application model.
// Takes model (clientModel) which holds the drawn component structure.
// Takes before (casefile.ClientInstance) which holds the state before the click.
//
// Returns casefile.ClientInstance which holds the modelled state after the click.
func clientAfter(p Project, model clientModel, before casefile.ClientInstance) casefile.ClientInstance {
	after := before
	after.Count = strconv.Itoa(p.Count + model.step)
	if model.list {
		after.Rows = []string{clientRowOne, clientRowTwo, "three"}
		if model.sequence {
			controls := *before.Controls
			after.Rows = []string{"zero", clientRowOne, clientRowTwo, "three"}
			controls.Keys = rowKeys(len(after.Rows))
			after.Controls = &controls
		}
	}
	if model.conditional {
		after.Visible = !before.Visible
	}
	return after
}

// componentSource emits the component's source and records its features.
//
// Takes c (*casefile.Case) which receives generated sources and expected results.
// Takes model (clientModel) which holds the drawn component structure.
// Takes p (Project) which holds the drawn application model.
//
// Returns string which is the emitted custom-element source.
func componentSource(c *casefile.Case, model clientModel, p Project) string {
	var body, fields, changed strings.Builder
	count := `<span id="count">{{ state.count }}</span>`
	if p.TextBinding {
		count = `<span id="count" p-text="state.count"></span>`
		c.Features = append(c.Features, "p-text")
	}
	click := "p-on:click"
	if c.Target == targetFormatter && model.connected {
		click += ".prevent.once"
		c.Features = append(c.Features, "formatter-native-event")
	}
	body.WriteString(count + `<button id="increment" ` + click + `="increment">+</button>`)
	fmt.Fprintf(&fields, "count: %d as number,", p.Count)
	imports := ""
	increment := fmt.Sprintf("state.count += %d;", model.step)
	if model.importStep {
		imports = `import { incrementByStep } from '@/lib/counter-step';`
		c.Files["lib/counter-step.ts"] = fmt.Sprintf("export function incrementByStep(value: number): number { return value + %d; }\n", model.step)
		increment = "state.count = incrementByStep(state.count);"
		c.Features = append(c.Features, "component-local-import")
	}
	changed.WriteString(increment)
	if model.attributes {
		body.WriteString(`<span id="client-label">{{ state.label }}</span>`)
		fields.WriteString(`label: "default" as string,`)
		c.Features = append(c.Features, "attribute-prop")
	}
	if model.list {
		componentList(c, model, &body, &fields, &changed)
	}
	if model.conditional {
		body.WriteString(`<p id="client-visible" p-if="state.shown">visible</p>`)
		fmt.Fprintf(&fields, "shown: %t as boolean,", p.Show)
		changed.WriteString("state.shown = !state.shown;")
		c.Features = append(c.Features, "client-condition")
	}
	methods := ""
	if model.sequence {
		body.WriteString(sequenceControls)
		body.WriteString(sequenceForm)
		fmt.Fprintf(&fields, "handled: 0 as number, bubbled: 0 as number, agreed: %t as boolean, choice: \"middle\" as string,", p.Show)
		methods = sequenceMethods + fmt.Sprintf("\nfunction resetControls() { state.agreed = %t; state.choice = \"middle\"; }", p.Show)
	}
	hook := ""
	if model.connected {
		fields.WriteString(`life: "new" as string,`)
		body.WriteString(`<span id="client-life">{{ state.life }}</span>`)
		hook = `pkc.onConnected(() => { state.life = "ready"; });`
		c.Features = append(c.Features, "connected-hook")
	}
	if p.Slot {
		listener := ""
		if c.Target == targetFormatter {
			listener = ` p-event:item-picked.stop="increment()"`
			c.Features = append(c.Features, "formatter-event")
		}
		body.WriteString(`<slot name="caption"` + listener + `></slot>`)
		c.Features = append(c.Features, "slot")
	}
	return fmt.Sprintf(componentTemplate, body.String(), imports, fields.String(), hook, changed.String(), methods)
}

// componentList adds the keyed component list to the component source.
//
// Takes c (*casefile.Case) which receives generated features.
// Takes model (clientModel) which holds the drawn component structure.
// Takes body (*strings.Builder) which receives the list markup.
// Takes fields (*strings.Builder) which receives the list state field.
// Takes changed (*strings.Builder) which receives the increment's list change.
func componentList(c *casefile.Case, model clientModel, body, fields, changed *strings.Builder) {
	memo := ""
	if c.Target == targetFormatter {
		memo = ` p-memo="[item, state.count]"`
		c.Features = append(c.Features, "formatter-memo")
	}
	body.WriteString(`<ul id="client-list"><li p-for="item in state.items" p-key="item"` + memo + `>{{ item }}</li></ul>`)
	if model.sequence {
		fields.WriteString(`items: ["zero","one","two"],`)
	} else {
		fields.WriteString(`items: ["one", "two"],`)
	}
	changed.WriteString(`state.items.push("three");`)
	c.Features = append(c.Features, "client-keyed-loop")
}

// modelClientSteps models native form and keyed-list interactions.
//
// Takes wanted (*casefile.ClientTransition) which receives independently modelled
// component steps.
// Takes p (Project) which holds the drawn application model.
// Takes variant (int) which selects the base component recipe.
//
// Returns error which reports an action naming an absent instance, an instance without
// modelled controls or a removal from an empty list, or nil on success.
func modelClientSteps(wanted *casefile.ClientTransition, p Project, variant int) error {
	states := wanted.After
	for _, action := range clientActions(p, variant) {
		if action.Instance < 0 || action.Instance >= len(states) {
			return fmt.Errorf("client action %s targets instance %d of %d", action.Op, action.Instance, len(states))
		}
		states = slices.Clone(states)
		instance := &states[action.Instance]
		if instance.Controls == nil {
			return fmt.Errorf("client action %s targets instance %d without modelled controls", action.Op, action.Instance)
		}
		controls := *instance.Controls
		instance.Controls = &controls
		if err := applyClientAction(instance, &controls, action, p.Show); err != nil {
			return err
		}
		wanted.Steps = append(wanted.Steps, casefile.ClientStep{Action: action, Prevented: action.Op == casefile.ModifiedClick, After: states})
	}
	return nil
}

// clientActions lists the base recipe's ordered browser actions.
//
// Takes p (Project) which holds the drawn application model.
// Takes variant (int) which selects the instance roles and action order.
//
// Returns []casefile.ClientAction which lists the ordered actions.
func clientActions(p Project, variant int) []casefile.ClientAction {
	first := variant % 2
	other := 1 - first
	entry := clientAction(casefile.SetInput, first, p.Title+" input")
	attribute := clientAction(casefile.SetLabel, other, p.Title+" attribute")
	reverse := clientAction(casefile.ReverseRows, first, "")
	remove := clientAction(casefile.RemoveRow, first, "")
	modified := clientAction(casefile.ModifiedClick, other, "")
	parent := clientAction(casefile.ParentClick, other, "")
	if variant&clientSwapEntries != 0 {
		entry, attribute = attribute, entry
	}
	choice := "first"
	if variant&clientSwapRows != 0 {
		reverse, remove = remove, reverse
		choice = "last"
	}
	if variant&clientSwapClicks != 0 {
		modified, parent = parent, modified
	}
	toggle := clientAction(casefile.ToggleCheck, first, "")
	selectOption := clientAction(casefile.SelectOption, first, choice)
	reset := clientAction(casefile.ResetControls, first, "")
	return []casefile.ClientAction{entry, reverse, modified, attribute, remove, parent, toggle, selectOption, reset}
}

// applyClientAction updates one instance's modelled state for a base browser action.
//
// Takes instance (*casefile.ClientInstance) which holds the acted-on instance.
// Takes controls (*casefile.ClientControls) which holds the instance's own copy of its
// controls.
// Takes action (casefile.ClientAction) which is the browser action to model.
// Takes show (bool) which is the checkbox state restored by a reset.
//
// Returns error which reports a removal from an empty list, or nil on success.
func applyClientAction(instance *casefile.ClientInstance, controls *casefile.ClientControls, action casefile.ClientAction, show bool) error {
	switch action.Op {
	case casefile.SetInput, casefile.SetLabel:
		instance.Label = action.Value
		controls.HostLabel, controls.Input = action.Value, action.Value
	case casefile.ReverseRows:
		instance.Rows, controls.Keys = slices.Clone(instance.Rows), slices.Clone(controls.Keys)
		slices.Reverse(instance.Rows)
		slices.Reverse(controls.Keys)
	case casefile.RemoveRow:
		if len(instance.Rows) == 0 || len(controls.Keys) == 0 {
			return fmt.Errorf("client action %s on instance %d with no rows", action.Op, action.Instance)
		}
		instance.Rows, controls.Keys = instance.Rows[1:], controls.Keys[1:]
	case casefile.ModifiedClick:
		controls.Handled = "1"
	case casefile.ParentClick:
		controls.Bubbled = "1"
	case casefile.ToggleCheck:
		controls.Form = formState(!controls.Form.Checked, controls.Form.Choice)
	case casefile.SelectOption:
		controls.Form = formState(controls.Form.Checked, action.Value)
	case casefile.ResetControls:
		controls.Form = formState(show, clientMiddleChoice)
	default:
	}
	return nil
}

// formState models native, rendered and reflected form values.
//
// Takes checked (bool) which is the expected checkbox state.
// Takes choice (string) which is the expected selected option.
//
// Returns *casefile.ClientForm which holds the native, rendered and reflected form
// values.
func formState(checked bool, choice string) *casefile.ClientForm {
	text := "off"
	if checked {
		text = "on"
	}
	return &casefile.ClientForm{Checked: checked, HostChecked: checked, CheckedState: text, Choice: choice, ChoiceState: choice, HostChoice: choice}
}

// rowKeys lists the one-based keys of a keyed list's rows.
//
// Takes count (int) which is the number of rows.
//
// Returns []int which holds the keys 1 to count in order.
func rowKeys(count int) []int {
	keys := make([]int, count)
	for index := range keys {
		keys[index] = index + 1
	}
	return keys
}
