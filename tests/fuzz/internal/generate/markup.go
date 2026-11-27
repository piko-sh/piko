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
	"strconv"
	"strings"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

const (
	// markupPagePath is the case file path of the markup page source.
	markupPagePath = "pages/markup.pk"

	// markupNotes selects the board's notes text area.
	markupNotes = "#notes"

	// markupOpener selects the board's dialog opener button.
	markupOpener = "#opener"

	// markupPagePrefix holds the markup page source before its literal text.
	markupPagePrefix = `<template><main><header><h1>Stock</h1><nav aria-label="Stock pages"><a href="/markup">Stock</a></nav></header>
<article aria-labelledby="story"><h2 data-fuzz-element="story" id="story">Description</h2><p>See <em>fresh</em> <strong>stock</strong>.</p>` +
		`<blockquote><p data-fuzz-element="citation">A &amp; B</p></blockquote>
<dl><dt>SKU</dt><dd>Model</dd></dl><figure id="pin-figure"><picture><source media="(max-width: 1px)" srcset="/markup-icon.svg">` +
		`<img alt="Pin" src="/markup-icon.svg"></picture><figcaption>Pin icon</figcaption></figure>
<pre id="literal"><code>`

	// markupPageSuffix holds the markup page source after its literal text.
	markupPageSuffix = `</code></pre>
<table id="implicit"><tr><td data-fuzz-element="cell">Only</td></tr></table>
<section id="special"><piko:a href="/markup">Stock link</piko:a><piko:element is="aside"><span>Dynamic note</span></piko:element>
<piko:img src="@/lib/markup-icon.svg" alt="Asset pin"></piko:img><piko:picture src="@/lib/markup-icon.svg" alt="Responsive pin"></piko:picture>` +
		`<piko:svg src="@/lib/markup-icon.svg" aria-label="Inline pin"></piko:svg></section>
<markup-board></markup-board></article><footer><small>End</small></footer></main></template>`

	// boardHost holds the markup fixture's component host.
	boardHost = "markup-board"

	// markupComponent holds the markup fixture's component source.
	markupComponent = `<template name="markup-board"><section>
<button id="reverse" p-on:click="reverse()">Reverse</button>
<table id="grid"><caption>Stock</caption><colgroup><col><col></colgroup><thead><tr><th scope="col">Item</th><th scope="col">Qty</th></tr></thead>
<tbody><tr p-for="item in state.items" p-key="item.id" :data-id="item.id"><th scope="row">{{ item.name }}</th><td>{{ item.amount }}</td></tr></tbody>
<tfoot><tr><th scope="row">Rows</th><td>{{ state.items.length }}</td></tr></tfoot></table>
<svg viewBox="0 0 30 30"><defs><path id="pin" d="M0 0L3 3"/></defs>
<g p-for="item in state.items" p-key="item.id" :data-id="item.id"><use href="#pin"/></g>
<foreignObject width="20" height="15"><div id="legend">Legend</div></foreignObject></svg>
<canvas id="swatch" width="4" height="4" aria-label="Colour swatch" p-ref="surface"></canvas><button id="paint" p-on:click="paint()">Paint</button>
<details id="more"><summary id="toggle">Details</summary><p>Stock detail</p></details>
<label for="notes">Notes</label><form id="notes-form"><textarea id="notes" name="notes">draft</textarea><button type="reset" id="reset">Reset</button></form>
<button id="opener" p-on:click="openDialog()">Open</button><dialog id="popup" p-ref="popup"><button id="close" autofocus p-on:click="closeDialog()">Close</button></dialog>
</section></template>
<script lang="ts">
const state={items:[{id:"a",name:%s,amount:%d},{id:"b",name:%s,amount:%d}]};
function paint(){const ctx=this.refs.surface.getContext("2d");ctx.fillStyle="#124578";ctx.fillRect(0,0,4,4);}
function reverse(){state.items=[...state.items].reverse();}
function openDialog(){this.refs.popup.showModal();}
function closeDialog(){this.refs.popup.close();}
</script>`
)

// addMarkup adds source and expected observations for public markup interactions.
//
// Takes c (*casefile.Case) which receives generated sources and expected results.
// Takes p (Project) which holds the drawn application model.
func addMarkup(c *casefile.Case, p Project) {
	label := "Item " + strconv.Itoa(p.Count)
	second := label
	if p.Show {
		second += " B"
	}
	line := fmt.Sprintf("left  %d\n  right", p.Count)
	c.Files[markupPagePath] = markupPagePrefix + html.EscapeString(line) + markupPageSuffix
	c.Files["lib/markup-icon.svg"] = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 4 4"><circle cx="2" cy="2" r="2"/></svg>`
	assets := strings.NewReplacer(`"/markup-icon.svg"`, `"/_piko/assets/`+casefile.Module+`/lib/markup-icon.svg"`, "@/lib/", casefile.Module+"/lib/")
	c.Files[markupPagePath] = assets.Replace(c.Files[markupPagePath])
	c.Files["components/markup-board.pkc"] = fmt.Sprintf(markupComponent, strconv.Quote(label), p.Count, strconv.Quote(second), p.Count+1)
	c.Files["pages/main.pk"] = strings.Replace(c.Files["pages/main.pk"], "</main>", "<markup-board hidden></markup-board></main>", 1)
	c.Expected.AdditionalComponents = append(c.Expected.AdditionalComponents, boardHost)
	contract := responseContract(nil,
		catalogueElement("title", "h1", "Stock"),
		catalogueElement("story", "h2", "Description", "id", "story"),
		catalogueElement("citation", "p", "A & B"),
		catalogueElement("cell", "td", "Only"),
	)
	c.Probes = append(c.Probes, semanticProbe("/markup", contract))
	c.Files[markupPagePath] = strings.Replace(c.Files[markupPagePath], "<h1>", "<h1 data-fuzz-element=\"title\">", 1)
	c.Features = append(c.Features, "markup-table", "markup-svg", "markup-controls", "markup-dialog", "markup-pre")
	c.Expected.DOM = markupExpectations(label, second, line, p.Count)
	addMarkupFamilies(c, p.Title)
}

// markupExpectations models the markup journey's public DOM values.
//
// Takes first (string) which is the first generated row label.
// Takes second (string) which is the second generated row label.
// Takes line (string) which is the expected literal source-text content.
// Takes quantity (int) which is the first table row's quantity.
//
// Returns *casefile.DOMPlan which holds the markup journey and its expected DOM values.
func markupExpectations(first, second, line string, quantity int) *casefile.DOMPlan {
	read := func(id, selector, property string, want ...string) casefile.DOMField {
		return markupField(id, boardHost, selector, property, "", want...)
	}
	mount := markupTable(first, second, quantity, false)
	mount = append(mount,
		markupField("implicit-body", "", "#implicit tr", "parent", "", "tbody"),
		markupField("source-text", "", "#literal code", domText, "", line),
		markupField("inline-text", "", "[data-fuzz-element=citation]", domText, "", "A & B"),
		markupField("caption", "", "#pin-figure figcaption", domText, "", "Pin icon"),
		read("initial-note", markupNotes, "value", "draft"),
		read("initial-disclosure", "#more", domOpen, "false"),
		markupField("dynamic-tag", "", "#special aside span", domText, "", "Dynamic note"),
		markupField("link-target", "", "#special a", "attr", "href", "/markup"),
		markupField("img", "", "#special img", "attr", "alt", "Asset pin", "Responsive pin"),
		markupField("picture-parent", "", "#special picture img", "parent", "", "picture"),
		markupField("inline-svg", "", "#special svg", "namespace", "", "http://www.w3.org/2000/svg"),
		markupField("image-load", "", "#pin-figure picture img", "image", "", domTrue),
		read("empty-pixel", "#swatch", "pixel", "0,0,0,0"),
	)
	steps := []casefile.DOMStep{
		domStep("mount", "inspect", "", "", "", mount...),
		domStep("reverse", domClick, boardHost, "#reverse", "", markupTable(first, second, quantity, true)...),
		domStep("toggle", domClick, boardHost, "#toggle", "", read("disclosure", "#more", domOpen, domTrue)),
		domStep("note-end", "key", boardHost, markupNotes, "End", read("caret", markupNotes, "selection", "5/5")),
		domStep("edit", "type", boardHost, markupNotes, "Z",
			read("note-focus", markupNotes, domFocus, domTrue), read("note-value", markupNotes, "value", "draftZ"), read("caret", markupNotes, "selection", "6/6")),
		domStep("reset", domClick, boardHost, "#reset", "",
			read("note-value", markupNotes, "value", "draft"), read("form-data", "#notes-form", "form", `[["notes","draft"]]`)),
		domStep("opener-focus", domFocus, boardHost, markupOpener, "", read(domFocus, markupOpener, domFocus, domTrue)),
		domStep("dialog", domClick, boardHost, markupOpener, "", read("dialog-open", "#popup", domOpen, domTrue), read("close-focus", "#close", domFocus, domTrue)),
		domStep("dismiss", "key", boardHost, "#close", "Escape",
			read("dialog-closed", "#popup", domOpen, "false"), read("return-focus", markupOpener, domFocus, domTrue)),
		domStep("canvas", domClick, boardHost, "#paint", "", read("painted-pixel", "#swatch", "pixel", "18,69,120,255")),
	}
	return &casefile.DOMPlan{Path: "/markup", ReadyHost: boardHost, ReadyElement: "#reverse", Steps: steps}
}

// markupTable models the board's table and SVG reads in either row order.
//
// Takes first (string) which is the first generated row label.
// Takes second (string) which is the second generated row label.
// Takes quantity (int) which is the first table row's quantity.
// Takes reversed (bool) which selects the reversed row order.
//
// Returns []casefile.DOMField which lists the table and SVG reads.
func markupTable(first, second string, quantity int, reversed bool) []casefile.DOMField {
	read := func(id, selector, property string, want ...string) casefile.DOMField {
		return markupField(id, boardHost, selector, property, "", want...)
	}
	attribute := func(id, selector, key string, want ...string) casefile.DOMField {
		return markupField(id, boardHost, selector, "attr", key, want...)
	}
	firstKey, secondKey := "a", "b"
	firstName, secondName := first, second
	firstQuantity, secondQuantity := strconv.Itoa(quantity), strconv.Itoa(quantity+1)
	firstIdentity, secondIdentity := "1", "2"
	if reversed {
		firstKey, secondKey = secondKey, firstKey
		firstName, secondName = secondName, firstName
		firstQuantity, secondQuantity = secondQuantity, firstQuantity
		firstIdentity, secondIdentity = secondIdentity, firstIdentity
	}
	return []casefile.DOMField{
		attribute("row-keys", "#grid tbody tr", "data-id", firstKey, secondKey),
		read("row-identity", "#grid tbody tr", "identity", firstIdentity, secondIdentity),
		read("names", "tbody th", domText, firstName, secondName),
		read("quantities", "tbody td", domText, firstQuantity, secondQuantity),
		read("cell-index", "tbody td", "cell", "1/1", "2/1"),
		read("rows", "#grid", "rows", "4"),
		attribute("svg-keys", "svg g", "data-id", firstKey, secondKey),
		read("svg-space", "svg g use", "namespace", "http://www.w3.org/2000/svg", "http://www.w3.org/2000/svg"),
		attribute("svg-ref", "svg g use", "href", "#pin", "#pin"),
		read("html-space", "#legend", "namespace", "http://www.w3.org/1999/xhtml"),
	}
}

// markupField builds one expected DOM read for the markup journey.
//
// Takes id (string) which identifies the read.
// Takes host (string) which names the component host, or empty for the page.
// Takes selector (string) which selects the read element.
// Takes read (string) which names the property read.
// Takes argument (string) which qualifies the read, such as an attribute name.
// Takes want (...string) which lists the expected values in document order.
//
// Returns casefile.DOMField which holds the read and its expected values.
func markupField(id, host, selector, read, argument string, want ...string) casefile.DOMField {
	return casefile.DOMField{ID: id, Query: casefile.DOMQuery{Host: host, Selector: selector, Read: read, Argument: argument}, Want: want}
}
