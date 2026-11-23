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
	"strconv"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

const (
	// shippedClock selects the animated time-display host.
	shippedClock = "#clock"

	// shippedFrame selects the time display's animated frame.
	shippedFrame = "#frame"

	// shippedPager selects the data-table pagination host.
	shippedPager = "#pager"

	// shippedSwitch selects the switch host.
	shippedSwitch = "#switch"

	// shippedSwitchRole selects the switch control inside its host.
	shippedSwitchRole = "[role=switch]"

	// shippedCard selects the card host.
	shippedCard = "#card"

	// shippedTools selects the developer widget host.
	shippedTools = "#tools"

	// shippedBefore is the time display's label before it is renamed.
	shippedBefore = "before"

	// shippedPresent names the attribute-presence read.
	shippedPresent = "present"

	// shippedChecked names the switch's checked attribute.
	shippedChecked = "checked"

	// shippedSlot names the slot-assignment read.
	shippedSlot = "slot"

	// shippedViewport names the viewport-resize interaction.
	shippedViewport = "viewport"

	// shippedIdleColour is the frame colour while its timeline is inactive.
	shippedIdleColour = "rgb(23, 24, 25)"

	// shippedPageTemplate formats the gallery page from its counter start and greeting.
	shippedPageTemplate = `<template><main><h1>Component gallery</h1>
<piko-counter id="counter" count="%d"></piko-counter>
<example-greeting id="greeting" message=%q></example-greeting>
<button id="rename" p-on:click="rename()">Change greeting</button>
<m2-data-table-pagination id="pager" total_rows="51" rows_per_page="25"></m2-data-table-pagination>
<m3e-switch id="switch"></m3e-switch>
<button id="clear-switch" p-on:click="clearSwitch()">Clear switch</button>
<piko-card id="card"><span id="card-title" slot="header">Record</span><span>Body</span></piko-card>
<button id="move-slot" p-on:click="moveSlot()">Move title</button>
<piko-dev-widget id="tools"></piko-dev-widget>
<time-display id="clock"></time-display><time-display id="sibling"></time-display>
</main></template>
<script lang="ts">
export function clearSwitch(){document.querySelector('#switch').removeAttribute('checked');}
export function moveSlot(){document.querySelector('#card-title').setAttribute('slot','footer');}
export function rename(){document.querySelector('#greeting').setAttribute('message','Updated');}
</script>`

	// shippedTimeDisplay holds the animated time-display component source.
	shippedTimeDisplay = `<template name="time-display" enable="animation">
<output id="frame" p-ref="frame">{{ state.label }}</output>
<button id="forward" p-on:click="forward()">Advance</button>
<button id="back" p-on:click="back()">Reverse</button>
<button id="rename" p-on:click="rename()">Change text</button>
</template>
<piko:timeline><piko:at time="1s"><piko:addclass ref="frame" class="active"/></piko:at></piko:timeline>
<piko:timeline media="(max-width: 600px)"><piko:at time="3s"><piko:addclass ref="frame" class="active"/></piko:at></piko:timeline>
<script lang="ts">
const state={time:0 as number, label:"before" as string};
function forward(){this.setAttribute("time","2");}
function back(){this.setAttribute("time","0");}
function rename(){state.label="after";}
</script>
<style>#frame{ --sample-colour:rgb(23, 24, 25);color:var(--sample-colour) }#frame.active{ --sample-colour:rgb(92, 93, 94) }</style>`
)

// addShipped adds source and expected observations for shipped-component interactions.
//
// Takes c (*casefile.Case) which receives generated sources and expected results.
// Takes p (Project) which holds the drawn application model.
func addShipped(c *casefile.Case, p Project) {
	count := 1 + p.Count
	label := fmt.Sprintf("Hello %d", p.Count)
	c.Files["pages/shipped.pk"] = fmt.Sprintf(shippedPageTemplate, count, label)
	c.Files["components/time-display.pkc"] = shippedTimeDisplay
	c.Expected.DOM = &casefile.DOMPlan{Path: "/shipped", ReadyHost: shippedTools, ReadyElement: ".toggle", Steps: shippedSteps(count, label)}
	c.Probes = append(c.Probes, textProbe("/shipped", "Component gallery"))
	c.Features = append(c.Features, "shipped-piko", "shipped-example", "shipped-m2", "shipped-m3e", "shipped-dev", "responsive-timeline", "reverse-seek")
}

// shippedSteps models the shipped-component journey and its expected DOM values.
//
// Takes count (int) which is the counter's starting value.
// Takes label (string) which is the greeting's initial message.
//
// Returns []casefile.DOMStep which lists the ordered journey steps.
func shippedSteps(count int, label string) []casefile.DOMStep {
	text := func(id, host, selector, want string) casefile.DOMField {
		return shippedField(id, host, selector, domText, "", want)
	}
	attribute := func(id, host, selector, key, want string) casefile.DOMField {
		return shippedField(id, host, selector, "attr", key, want)
	}
	slot := func(id, selector, want string) casefile.DOMField {
		return shippedField(id, shippedCard, selector, shippedSlot, "", want)
	}
	present := func(id, host, selector, key, want string) casefile.DOMField {
		return shippedField(id, host, selector, shippedPresent, key, want)
	}
	nextPage, lastPage := "button[aria-label='Next page']", "button[aria-label='Last page']"
	return []casefile.DOMStep{
		domStep("first", "inspect", "", "", "", text("start", "#counter", ".value", strconv.Itoa(count)), text("greeting", "#greeting", ".greeting", label)),
		domStep("counter", domClick, "#counter", ".buttons button:last-child", "", text("increment", "#counter", ".value", strconv.Itoa(count+1))),
		domStep("greeting", domClick, "", "#rename", "", text("prop-change", "#greeting", ".greeting", "Updated")),
		domStep("table-next", domClick, shippedPager, nextPage, "", text("page-range", shippedPager, ".page-range", "26\u201350 of 51")),
		domStep("table-last", domClick, shippedPager, lastPage, "",
			text("page-range", shippedPager, ".page-range", "51\u201351 of 51"), present("last-disabled", shippedPager, nextPage, "disabled", domTrue)),
		domStep("switch", domClick, shippedSwitch, shippedSwitchRole, "",
			attribute(shippedChecked, shippedSwitch, shippedSwitchRole, "aria-checked", domTrue), present("host-reflection", "", shippedSwitch, shippedChecked, domTrue)),
		domStep("switch-key", "key", shippedSwitch, shippedSwitchRole, "Enter", attribute("unchecked", shippedSwitch, shippedSwitchRole, "aria-checked", "false")),
		domStep("switch-again", domClick, shippedSwitch, shippedSwitchRole, "", present("reflected", "", shippedSwitch, shippedChecked, domTrue)),
		domStep("switch-remove", domClick, "", "#clear-switch", "",
			attribute("cleared", shippedSwitch, shippedSwitchRole, "aria-checked", "false"), present("absent", "", shippedSwitch, shippedChecked, "false")),
		domStep("slot-before", "inspect", "", "", "", slot("header", "slot[name=header]", `["Record"]`), slot("default", "slot:not([name])", `["Body"]`)),
		domStep("slot-move", domClick, "", "#move-slot", "",
			slot("old-fallback", "slot[name=header]", `["Card"]`), slot("new-recipient", "slot[name=footer]", `["Record"]`),
			shippedField("footer-trigger", shippedCard, "footer", "visible", "", domTrue), slot("default", "slot:not([name])", `["Body"]`)),
		domStep("dev-open", domClick, shippedTools, ".toggle", "", shippedField("panel-open", shippedTools, ".panel.open", "tag", "", "div")),
		domStep("dev-close", domClick, shippedTools, ".close-btn", "", shippedField("panel-closed", shippedTools, ".panel:not(.open)", "tag", "", "div")),
		domStep("wide", shippedViewport, "", "", "wide", shippedClockFields(false, shippedBefore)...),
		domStep("forward", domClick, shippedClock, "#forward", "", shippedClockFields(true, shippedBefore)...),
		domStep("narrow", shippedViewport, "", "", "narrow", shippedClockFields(false, shippedBefore)...),
		domStep("wide-again", shippedViewport, "", "", "wide", shippedClockFields(true, shippedBefore)...),
		domStep("rerender", domClick, shippedClock, "#rename", "", shippedClockFields(true, "after")...),
		domStep("reverse", domClick, shippedClock, "#back", "", shippedClockFields(false, "after")...),
	}
}

// shippedClockFields models the animated clock and its unanimated sibling.
//
// Takes active (bool) which reports whether the clock's timeline class is applied.
// Takes label (string) which is the clock's expected frame text.
//
// Returns []casefile.DOMField which lists the clock and sibling reads.
func shippedClockFields(active bool, label string) []casefile.DOMField {
	colour := shippedIdleColour
	if active {
		colour = "rgb(92, 93, 94)"
	}
	return []casefile.DOMField{
		shippedField("scoped-colour", shippedClock, shippedFrame, "style", "color", colour),
		shippedField("edited", shippedClock, shippedFrame, domText, "", label),
		shippedField("sibling-scope", "#sibling", shippedFrame, "style", "color", shippedIdleColour),
		shippedField("sibling-text", "#sibling", shippedFrame, domText, "", shippedBefore),
	}
}

// shippedField builds one expected single-valued DOM read.
//
// Takes id (string) which identifies the read.
// Takes host (string) which names the component host, or empty for the page.
// Takes selector (string) which selects the read element.
// Takes read (string) which names the property read.
// Takes argument (string) which qualifies the read, such as an attribute name.
// Takes want (string) which is the expected value.
//
// Returns casefile.DOMField which holds the read and its expected value.
func shippedField(id, host, selector, read, argument, want string) casefile.DOMField {
	return casefile.DOMField{ID: id, Query: casefile.DOMQuery{Host: host, Selector: selector, Read: read, Argument: argument}, Want: []string{want}}
}
