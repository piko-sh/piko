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
	"strconv"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

// addBrowserLifecycle adds source and expected observations for component lifecycle
// interactions.
//
// Takes c (*casefile.Case) which receives generated sources and expected results.
// Takes p (Project) which holds the drawn application model.
func addBrowserLifecycle(c *casefile.Case, p Project) {
	c.Expected.Client.Lifecycle = casefile.ExpectedLifecycle(casefile.LifecycleInput{
		Draft: p.Title + " " + strconv.Itoa(p.Count), Reverse: (c.Seed+c.Index)%2 != 0,
	})
	c.Features = append(c.Features, "dirty-spa-navigation", "history-back", "public-partial-refresh", "detached-async-completion", "listener-cleanup")
	c.Files["pages/journey.pk"] = `<template><section id="journey">
<form id="editor"><input id="draft" name="draft"></form>
<piko:partial is="list"></piko:partial>
<fuzz-watch id="first"></fuzz-watch><fuzz-watch id="second"></fuzz-watch>
<a href="/journey-away">Leave editor</a></section></template>
<script type="application/x-go">
package main
import (
 "piko.sh/piko"
 list "example.org/pikofuzzcase/partials/journey-list.pk"
)
func Render(r *piko.RequestData, props piko.NoProps) (piko.NoResponse,piko.Metadata,error) {
 return piko.NoResponse{},piko.Metadata{Title:"Editor"},nil
}
</script>`
	c.Files["partials/journey-list.pk"] = `<template public><div id="journey-list"><span id="retained-row" p-key="'row'">saved</span></div></template>`
	c.Files["pages/journey-away.pk"] = `<template><main id="journey-away">Editor closed</main></template>`
	c.Files["components/fuzz-watch.pkc"] = `<template name="fuzz-watch"><output id="received">{{ state.received }}</output></template>
<script lang="ts">
const state = {received:"" as string};
pkc.onConnected(() => {
  const node = pkc;
  const log = window.__journey;
  if (!log) return;
  log.connected++;
  let active = true;
  const listen = () => {
    log.deliveries++;
    if (log.started) return;
    const owner = node.id;
    const gate = new Promise<void>(resolve => { log.release[owner] = resolve; });
    fetch('/journey-away').then(response => {
      if (!response.ok) throw new Error("response " + response.status);
      return response.text();
    }).then(async body => {
      if (!body.includes('journey-away')) throw new Error("wrong local response");
      log.ready.push(owner);
      await gate;
      if (active) { state.received = owner; log.applied.push(owner); }
      log.finished.push(owner);
    });
  };
  window.addEventListener("journey-request",listen);
  pkc.onCleanup(() => { active = false; log.cleaned++; window.removeEventListener("journey-request",listen); });
});
</script>`
	c.Probes = append(c.Probes, textProbe("/journey", "saved"), textProbe("/journey-away", "Editor closed"))
}
