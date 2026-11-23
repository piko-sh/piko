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
	"piko.sh/piko/tests/fuzz/internal/casefile"
)

// addReloadModes adds source and expected observations for reload-mode interactions.
//
// Takes c (*casefile.Case) which receives generated sources and expected results.
func addReloadModes(c *casefile.Case) {
	c.Files["pages/reload-history.pk"] = `<template>
<main><h1>Partial reload history</h1>
  <refresh-owner id="reload-owner">
    <piko:partial is="sample" stage="initial"></piko:partial>
  </refresh-owner>
  <refresh-owner id="reload-sibling">Sibling</refresh-owner>
</main>
</template>
<script type="application/x-go">
package main

import sample "example.org/pikofuzzcase/partials/reload-sample.pk"
</script>`
	c.Files["partials/reload-sample.pk"] = `<template public>
<section id="reload-sample" :data-stage="props.Stage" :data-preserved="props.Stage">
  <label for="reload-input">Draft</label>
  <input id="reload-input" :value="props.Stage">
  <output id="reload-content" p-text="props.Stage"></output>
</section>
</template>
<script type="application/x-go">
package main

import "piko.sh/piko"

type Props struct { Stage string ` + "`prop:\"stage\" query:\"stage\"`" + ` }
func Render(r *piko.RequestData, props Props) (piko.NoResponse, piko.Metadata, error) {
    return piko.NoResponse{}, piko.Metadata{}, nil
}
</script>`
	c.Files["components/refresh-owner.pkc"] = `<template name="refresh-owner">
  <slot></slot>
  <button type="button" id="merge" p-on:click="refresh('merge')">Merge</button>
  <button type="button" id="replace" p-on:click="refresh('replace')">Replace</button>
  <button type="button" id="children-only" p-on:click="refresh('children-only')">Children</button>
  <button type="button" id="attrs-only" p-on:click="refresh('attrs-only')">Attributes</button>
  <button type="button" id="bump" p-on:click="bump()">Count</button>
  <output id="reload-count">{{ state.count }}</output>
  <output id="reload-result">{{ state.result }}</output>
</template>
<script lang="ts">
const state = {count: 0, result: ""};
function bump() { state.count++; }
async function refresh(mode: string) {
    await window.piko.partials.reload("reload-sample", {
        mode,
        args: {stage: mode},
        preserveAttrs: ["data-preserved"]
    });
    state.result = mode;
}
</script>`
	c.Probes = append(c.Probes, textProbe("/reload-history", "Partial reload history"))
	c.Features = append(c.Features, "partial-reload-modes", "focused-draft-selection", "partial-root-ownership")
}
