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
	"html"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

const (
	// browserEventChoices bounds the drawn event delta and slot rotation.
	browserEventChoices = 4
)

// addBrowserEvents adds source and expected observations for component event
// interactions.
//
// Takes c (*casefile.Case) which receives generated sources and expected results.
// Takes p (Project) which holds the drawn application model.
func addBrowserEvents(c *casefile.Case, p Project) {
	input := casefile.EventsInput{Label: p.Title, Delta: 1 + p.Count%browserEventChoices, Rotate: int((c.Seed + c.Index) % browserEventChoices)}
	c.Expected.Client.Events = casefile.ExpectedEvents(input)
	c.Features = append(c.Features, "shadow-event-flags", "pk-event-owner", "slot-reassignment", "ts-module-diamond")
	c.Probes = append(c.Probes, textProbe("/events", "Event board"))
	c.Files["pages/events.pk"] = fmt.Sprintf(`<template><main id="event-page" p-event:board-signal="pageSaw($event)">
	<h1>Event board</h1><fuzz-owner id="left" delta="%d"><fuzz-emitter id="leaf" slot="board" delta="%d" payload="%s"></fuzz-emitter></fuzz-owner>
	<fuzz-owner id="right" delta="%d"></fuzz-owner><output id="event-count">0</output><output id="event-payload"></output><output id="event-log"></output>
	</main></template><script lang="ts">
	export function pageSaw(event: CustomEvent) {
		const count=document.getElementById('event-count');
		count.textContent=String(Number(count.textContent)+1);
		document.getElementById('event-log').textContent+='pk:'+event.target.id+';';
		document.getElementById('event-payload').textContent=event.detail.text;
	}
	</script>`, input.Delta, input.Delta, html.EscapeString(input.Label), input.Delta)
	c.Files["lib/event-serial.ts"] = `let serial = 0;
	globalThis.__pikoEventInits=(globalThis.__pikoEventInits || 0)+1;
	export function advance(delta: number): number { serial+=delta; return serial; }`
	for _, branch := range []string{"emitter", "owner"} {
		c.Files["lib/event-"+branch+".ts"] = `import {advance} from './event-serial.js';
		export function stamp(delta:number):number{return advance(delta);}`
	}
	c.Files["components/fuzz-emitter.pkc"] = `<template name="fuzz-emitter"><section>
	<span id="origin" p-ref="origin" p-on:board-signal="localSaw($event)">origin</span>
	<button id="emit-0" p-on:click="emitSignal(false,false)">0</button><button id="emit-1" p-on:click="emitSignal(true,false)">1</button>
	<button id="emit-2" p-on:click="emitSignal(false,true)">2</button><button id="emit-3" p-on:click="emitSignal(true,true)">3</button>
	<output id="local">{{ state.local }}</output><output id="serial">{{ state.serial }}</output></section></template>
	<script lang="ts">
	import {stamp} from '@/lib/event-emitter';
	const state={delta:1 as number,payload:"" as string,local:0 as number,serial:0 as number};
	function localSaw(event:CustomEvent){state.local++; document.getElementById('event-log').textContent+='local:'+event.target.id+';';}
	function emitSignal(bubbles:boolean,composed:boolean) {
		state.serial=stamp(state.delta);
		this.refs.origin.dispatchEvent(new CustomEvent('board-signal',{bubbles,composed,detail:{text:state.payload+'#'+state.serial}}));
	}
	</script>`
	c.Files["components/fuzz-owner.pkc"] = `<template name="fuzz-owner"><section>
	<slot name="board" p-event:board-signal="ownerSaw($event)"></slot><button id="tick" p-on:click="tick">tick</button>
	<output id="seen">{{ state.seen }}</output><output id="serial">{{ state.serial }}</output>
	</section></template><script lang="ts">
	import {stamp} from '@/lib/event-owner';
	const state={delta:1 as number,seen:0 as number,serial:0 as number};
	function tick(){state.serial=stamp(state.delta);}
	function ownerSaw(event:CustomEvent){state.seen++; document.getElementById('event-log').textContent+=this.id+':'+event.target.id+';';}
	</script>`
}
