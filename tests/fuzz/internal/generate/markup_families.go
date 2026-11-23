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
	"html"
	"strings"

	"piko.sh/safeconv"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

const (
	// markupIndexChoices is the number of family windows selected by case index.
	markupIndexChoices = 4

	// markupSeedChoices bounds the seed's offset into the family list.
	markupSeedChoices = 16
)

var (
	// markupFamilies lists legal markup fragments and their measured leaves.
	markupFamilies = []markupFamily{
		{id: "definition", fragment: `<dl><dt>Term</dt><dd #>$</dd></dl>`, parent: "dl", tag: "dd"},
		{id: "quotation", fragment: `<figure><blockquote><p>Quote</p></blockquote><figcaption #>$</figcaption></figure>`, parent: "figure", tag: "figcaption"},
		{id: "ruby", fragment: `<p><ruby>字<rp>(</rp><rt #>$</rt><rp>)</rp></ruby></p>`, parent: "ruby", tag: "rt"},
		{id: "edits", fragment: `<p><del>Old</del><ins #>$</ins></p>`, parent: "p", tag: "ins"},
		{id: "description", fragment: `<aside><address><span #>$</span></address></aside>`, parent: "address", tag: "span"},
		{id: "keyboard", fragment: `<p><samp>Key</samp><kbd #>$</kbd></p>`, parent: "p", tag: "kbd"},
		{id: "time", fragment: `<p><time datetime="2024-02-29" #>$</time></p>`, parent: "p", tag: "time"},
		{id: "form", fragment: `<form><fieldset><legend #>$</legend><label>Quantity<input type="number" value="2"></label></fieldset></form>`, parent: "fieldset", tag: "legend"},
		{id: "options", fragment: `<label>Pick<select><optgroup label="Group"><option selected #>$</option></optgroup></select></label>`, parent: "optgroup", tag: "option"},
		{id: "meter", fragment: `<p><meter min="0" max="8" value="3" #>$</meter><progress value="1" max="4">Quarter</progress></p>`, parent: "p", tag: "meter"},
		{id: "bidi", fragment: `<p><bdi #>$</bdi><wbr><bdo dir="rtl">abc</bdo></p>`, parent: "p", tag: "bdi"},
		{id: "navigation", fragment: `<nav aria-label="Shelf"><ol><li><a href="/markup" #>$</a></li></ol></nav>`, parent: "li", tag: "a"},
		{id: "table", fragment: `<table><caption>Season</caption><tbody><tr><th scope="row">Q1</th><td #>$</td></tr></tbody></table>`, parent: "tr", tag: "td"},
		{id: "group", fragment: `<hgroup><h3>Group</h3><p #>$</p></hgroup>`, parent: "hgroup", tag: "p"},
		{
			id:       "multimedia",
			fragment: `<figure><audio preload="none" controls></audio><video muted playsinline preload="none"></video><figcaption #>$</figcaption></figure>`,
			parent:   "figure",
			tag:      "figcaption",
		},
		{id: "text", fragment: `<p><dfn>Term</dfn><abbr title="Units">u</abbr><mark #>$</mark><sub>1</sub><sup>2</sup></p>`, parent: "p", tag: "mark"},
	}
)

// markupFamily defines one markup fragment and its enclosing element.
type markupFamily struct {
	// id identifies the markup family.
	id string

	// fragment holds the legal markup with "#" marking the measured element's identifier and
	// "$" marking its text.
	fragment string

	// parent is the measured element's expected parent tag.
	parent string

	// tag is the measured element's expected tag.
	tag string
}

// addMarkupFamilies adds source and expected observations for markup-family fragments.
//
// Takes c (*casefile.Case) which receives generated sources and expected results.
// Takes value (string) which supplies the text used in each generated fragment.
func addMarkupFamilies(c *casefile.Case, value string) {
	source := c.Files["pages/markup.pk"]
	var block strings.Builder
	step := domStep("content-models", "inspect", "", "", "")

	const width = 4
	start := safeconv.Uint64ToInt((c.Index%markupIndexChoices)*width + c.Seed%markupSeedChoices)
	for offset := range width {
		family := markupFamilies[(start+offset)%len(markupFamilies)]
		id := "leaf-" + family.id

		label := "Case " + family.id + " " + strings.Join(strings.Fields(value), " ")
		block.WriteString(strings.NewReplacer("#", `id="`+id+`"`, "$", html.EscapeString(label)).Replace(family.fragment))
		reads := []struct{ operation, want string }{
			{operation: "parent", want: family.parent},
			{operation: "tag", want: family.tag},
			{operation: "text", want: label},
		}
		for _, read := range reads {
			field := casefile.DOMField{ID: family.id + "-" + read.operation, Query: domQuery("#"+id, read.operation), Want: []string{read.want}}
			step.Fields = append(step.Fields, field)
		}
		if family.id == "multimedia" {
			field := casefile.DOMField{ID: "media-defaults", Query: domQuery("figure audio, figure video", "media"), Want: []string{"true/true/none", "true/false/none"}}
			step.Fields = append(step.Fields, field)
		}
		c.Features = append(c.Features, "html-family-"+family.id)
	}
	c.Files["pages/markup.pk"] = strings.Replace(source, "</article>", block.String()+"</article>", 1)
	c.Expected.DOM.Steps = append(c.Expected.DOM.Steps, step)
}
