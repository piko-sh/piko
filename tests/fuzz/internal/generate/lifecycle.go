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
	// lifecycleLayoutImport imports the layout partial a journey's pages share.
	lifecycleLayoutImport = `layout "` + casefile.Module + `/partials/journey-layout.pk"`

	// lifecycleLayout formats the layout partial a journey's pages share from the links its
	// navigation renders. SPA navigation between the pages morphs its root in place.
	lifecycleLayout = `<template><div id="journey-layout"><nav id="journey-nav">%s</nav>
<div id="journey-content"><piko:slot></piko:slot></div></div></template>
<script type="application/x-go">
package main
import "piko.sh/piko"
func Render(r *piko.RequestData, props piko.NoProps) (piko.NoResponse,piko.Metadata,error) {
 return piko.NoResponse{},piko.Metadata{},nil
}
</script>`

	// lifecycleGoScript formats a journey page's Go script from its imports and title.
	lifecycleGoScript = `
<script type="application/x-go">
package main
import (
 "piko.sh/piko"
 %s
)
func Render(r *piko.RequestData, props piko.NoProps) (piko.NoResponse,piko.Metadata,error) {
 return piko.NoResponse{},piko.Metadata{Title:%q},nil
}
</script>`

	// lifecyclePageScript formats a page script from its declarations and the expression
	// that finds its page's counters. The script counts its lifecycle callbacks and, while
	// connected, the clicks on its page's p-ref element.
	lifecyclePageScript = `
<script lang="ts">%[1]s
pk.onConnected(() => {
  const page = %[2]s;
  if (!page) return;
  page.connected++;
  const bound = pk.refs.bound as HTMLElement | undefined;
  const click = () => { page.clicks++; };
  bound?.addEventListener("click", click);
  pk.onCleanup(() => { page.cleaned++; bound?.removeEventListener("click", click); });
});
pk.onDisconnected(() => {
  const page = %[2]s;
  if (page) page.disconnected++;
});
</script>`

	// lifecycleCountersFunction formats the top-level function a page script declares to
	// find its page's counters, from the expression that finds them.
	lifecycleCountersFunction = `
function pageCounters() {
  return %s;
}`

	// lifecycleWatch is the component whose connections, cleanups and asynchronous
	// completions the editor journey observes.
	lifecycleWatch = `<template name="fuzz-watch"><output id="received">{{ state.received }}</output></template>
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

	// reusePager is a pager that, like a paginator bound to its listing's search box, reads
	// its configuration and binds to its filter input when it connects, and unbinds when it
	// disconnects.
	reusePager = `<template name="listing-pager"><output id="pager-owner">{{ state.owner }}</output></template>
<script lang="ts">
const state = {owner:"" as string, filter_input:"" as string};
pkc.onConnected(() => {
  const log = window.__journey;
  if (!log) return;
  log.connected++;
  const owner = state.owner;
  const filter = document.querySelector(state.filter_input);
  const react = () => { log.deliveries++; log.applied.push(owner); };
  filter?.addEventListener("change", react);
  pkc.onCleanup(() => { log.cleaned++; filter?.removeEventListener("change", react); });
});
</script>`
)

// addBrowserLifecycle adds source and expected observations for component and page script
// lifecycle interactions. Its page scripts declare a top-level function.
//
// Takes c (*casefile.Case) which receives generated sources and expected results.
// Takes p (Project) which holds the drawn application model.
func addBrowserLifecycle(c *casefile.Case, p Project) {
	addEditorJourney(c, editorInput(c, p))
}

// addBrowserLifecycleInline adds the editor journey with page scripts that have no
// function declarations.
//
// Takes c (*casefile.Case) which receives generated sources and expected results.
// Takes p (Project) which holds the drawn application model.
func addBrowserLifecycleInline(c *casefile.Case, p Project) {
	input := editorInput(c, p)
	input.InlineScripts = true
	addEditorJourney(c, input)
	c.Features = append(c.Features, "inline-page-script")
}

// editorInput draws the editor journey's input from the case position and project.
//
// Takes c (*casefile.Case) which holds the case's seed and index.
// Takes p (Project) which holds the drawn application model.
//
// Returns casefile.LifecycleInput which selects the draft, completion order and link
// placement, with page scripts that declare a top-level function.
func editorInput(c *casefile.Case, p Project) casefile.LifecycleInput {
	return casefile.LifecycleInput{
		Draft:         p.Title + " " + strconv.Itoa(p.Count),
		Reverse:       (c.Seed+c.Index)%2 != 0,
		LayoutLinks:   (c.Seed+c.Index)/2%2 != 0,
		InlineScripts: false,
		ElementReuse:  false,
	}
}

// addEditorJourney adds the editor journey's sources and expected observations.
//
// The journey and away pages share a layout partial and each has a page script, so SPA
// navigation between them morphs the layout's root in place while the page scripts must
// still disconnect on every leave and connect on every visit.
//
// Takes c (*casefile.Case) which receives generated sources and expected results.
// Takes input (casefile.LifecycleInput) which selects the journey's variant.
func addEditorJourney(c *casefile.Case, input casefile.LifecycleInput) {
	c.Expected.Client.Lifecycle = casefile.ExpectedLifecycle(input)
	c.Features = append(c.Features, "dirty-spa-navigation", "history-back", "public-partial-refresh", "detached-async-completion",
		"listener-cleanup", "layout-partial", "page-script-lifecycle", "piko-a-navigation", "history-forward")
	nav, toAway, toJourney := journeyLinks(input)("journey", "/journey", "away", "/journey-away")
	c.Files["partials/journey-layout.pk"] = fmt.Sprintf(lifecycleLayout, nav)
	c.Files["pages/journey.pk"] = `<template><piko:partial is="layout"><section id="journey">
<form id="editor"><input id="draft" name="draft"></form>
<piko:partial is="list"></piko:partial>
<fuzz-watch id="first"></fuzz-watch><fuzz-watch id="second"></fuzz-watch>
<button type="button" id="journey-bound" p-ref="bound">Bound</button>` + toAway + `
<a href="/journey-away">Leave editor</a></section></piko:partial></template>` +
		fmt.Sprintf(lifecycleGoScript, lifecycleLayoutImport+"\n list \""+casefile.Module+"/partials/journey-list.pk\"", "Editor") +
		journeyPageScript(input)("journey")
	c.Files["partials/journey-list.pk"] = `<template public><div id="journey-list"><span id="retained-row" p-key="'row'">saved</span></div></template>`
	c.Files["pages/journey-away.pk"] = `<template><piko:partial is="layout"><main id="journey-away">Editor closed
<button type="button" id="away-bound" p-ref="bound">Bound</button>` + toJourney + `</main></piko:partial></template>` +
		fmt.Sprintf(lifecycleGoScript, lifecycleLayoutImport, "Away") + journeyPageScript(input)("away")
	c.Files["components/fuzz-watch.pkc"] = lifecycleWatch
	c.Probes = append(c.Probes, textProbe("/journey", "saved"), textProbe("/journey-away", "Editor closed"))
}

// addBrowserElementReuse adds source and expected observations for the element-reuse
// journey.
//
// Two listings share a layout partial and have the same structure, with a filter input
// and a pager component bound to it. SPA navigation between them can morph one listing's
// elements into the other's, which must not leave the pager bound with the previous
// listing's configuration.
//
// Takes c (*casefile.Case) which receives generated sources and expected results.
func addBrowserElementReuse(c *casefile.Case, _ Project) {
	input := casefile.LifecycleInput{Draft: "", Reverse: false, LayoutLinks: (c.Seed+c.Index)%2 != 0, InlineScripts: false, ElementReuse: true}
	c.Expected.Client.Lifecycle = casefile.ExpectedLifecycle(input)
	c.Features = append(c.Features, "morph-element-reuse", "layout-partial", "piko-a-navigation", "history-back", "listener-cleanup")
	first, second := casefile.ReuseFirstListing, casefile.ReuseSecondListing
	nav, toSecond, toFirst := journeyLinks(input)(first, casefile.ReuseListingPath(first), second, casefile.ReuseListingPath(second))
	c.Files["partials/journey-layout.pk"] = fmt.Sprintf(lifecycleLayout, nav)
	c.Files["pages/reuse-"+first+".pk"] = reuseListing(first, toSecond)
	c.Files["pages/reuse-"+second+".pk"] = reuseListing(second, toFirst)
	c.Files["components/listing-pager.pkc"] = reusePager
	c.Probes = append(c.Probes, textProbe(casefile.ReuseListingPath(first), "Listing "+first), textProbe(casefile.ReuseListingPath(second), "Listing "+second))
}

// journeyPageScript selects how the journey's page scripts are written.
//
// Takes input (casefile.LifecycleInput) which selects the journey's variant.
//
// Returns func(string) string which renders a page's script from the name of its
// counters.
func journeyPageScript(input casefile.LifecycleInput) func(key string) string {
	if input.InlineScripts {
		return inlinePageScript
	}
	return functionPageScript
}

// functionPageScript renders a journey page's script that finds its counters through a
// top-level function.
//
// Takes key (string) which names the page's counters in the journey's event log.
//
// Returns string which is the page script source.
func functionPageScript(key string) string {
	return fmt.Sprintf(lifecyclePageScript, fmt.Sprintf(lifecycleCountersFunction, pageCountersExpression(key)), "pageCounters()")
}

// inlinePageScript renders a journey page's script with no function declarations.
//
// Takes key (string) which names the page's counters in the journey's event log.
//
// Returns string which is the page script source.
func inlinePageScript(key string) string {
	return fmt.Sprintf(lifecyclePageScript, "", pageCountersExpression(key))
}

// pageCountersExpression builds the expression that finds a page's counters.
//
// Takes key (string) which names the page's counters in the journey's event log.
//
// Returns string which is the JavaScript expression.
func pageCountersExpression(key string) string {
	return "window.__journey?.pages." + key
}

// reuseListing renders an element-reuse listing page. Every listing has the same
// structure, and only its text, the filter's id and the pager's attributes differ.
//
// Takes listing (string) which names the listing.
// Takes link (string) which is the `<piko:a>` link the page renders itself, or empty.
//
// Returns string which is the page source.
func reuseListing(listing, link string) string {
	return fmt.Sprintf(`<template><piko:partial is="layout"><section class="listing">
<h1 class="listing-title">Listing %[1]s</h1><input id="%[1]s-filter" class="listing-filter" name="filter">
<listing-pager owner="%[1]s" filter_input="#%[1]s-filter"></listing-pager>%[2]s
</section></piko:partial></template>`, listing, link) + fmt.Sprintf(lifecycleGoScript, lifecycleLayoutImport, "Listing "+listing)
}

// journeyLinks selects where the `<piko:a>` links between a journey's two pages sit.
//
// Takes input (casefile.LifecycleInput) which selects the journey's variant.
//
// Returns func(...) (string, string, string) which places the links and returns the
// layout's navigation, the link the first page renders and the link the second page
// renders.
func journeyLinks(input casefile.LifecycleInput) func(first, firstPath, second, secondPath string) (nav, toSecond, toFirst string) {
	if input.LayoutLinks {
		return layoutLinks
	}
	return pageLinks
}

// layoutLinks places the `<piko:a>` links between a journey's two pages in the shared
// layout's navigation.
//
// Takes first (string) which names the first page.
// Takes firstPath (string) which is the first page's route.
// Takes second (string) which names the second page.
// Takes secondPath (string) which is the second page's route.
//
// Returns nav (string) which is the layout's navigation, holding both links.
// Returns toSecond (string) which is empty, as the first page renders no link.
// Returns toFirst (string) which is empty, as the second page renders no link.
func layoutLinks(first, firstPath, second, secondPath string) (nav, toSecond, toFirst string) {
	return pikoLink(first, firstPath) + pikoLink(second, secondPath), "", ""
}

// pageLinks places the `<piko:a>` links between a journey's two pages in each page's own
// content.
//
// Takes first (string) which names the first page.
// Takes firstPath (string) which is the first page's route.
// Takes second (string) which names the second page.
// Takes secondPath (string) which is the second page's route.
//
// Returns nav (string) which is empty, as the layout renders no link.
// Returns toSecond (string) which is the link the first page renders itself.
// Returns toFirst (string) which is the link the second page renders itself.
func pageLinks(first, firstPath, second, secondPath string) (nav, toSecond, toFirst string) {
	return "", pikoLink(second, secondPath), pikoLink(first, firstPath)
}

// pikoLink renders the `<piko:a>` link to a journey page.
//
// Takes name (string) which names the page and the link's id.
// Takes path (string) which is the page's route.
//
// Returns string which is the link markup.
func pikoLink(name, path string) string {
	return `<piko:a href="` + path + `" id="link-` + name + `">` + name + `</piko:a>`
}
