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
	_ "embed"
	"strings"

	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/tests/fuzz/internal/fixturego"
)

const (
	// partialReloadIndexes is the number of leading cases that enumerate reload orders.
	partialReloadIndexes = 8

	// partialReloadPeriod is the case-position period of the enumerated reload order.
	partialReloadPeriod = 4
)

var (
	// recordsList embeds source used to emit and identify the generated fixture.
	//
	//go:embed partial_components_list.pk.tmpl
	recordsList string

	// recordsRows embeds source used to emit and identify the generated fixture.
	//
	//go:embed partial_components_rows.pk.tmpl
	recordsRows string

	// recordsDetail embeds source used to emit and identify the generated fixture.
	//
	//go:embed partial_components_detail.pk.tmpl
	recordsDetail string

	// recordsRow embeds source used to emit and identify the generated fixture.
	//
	//go:embed partial_components_row.pkc.tmpl
	recordsRow string

	// recordsAction is the formatted Go source staged in the generated project.
	recordsAction = fixturego.Fixed("actions/records/advance.go")
)

// addPartialComponents adds source and expected observations for partial-component
// interactions.
//
// Takes c (*casefile.Case) which receives generated sources and expected results.
// Takes p (Project) which holds the drawn application model.
//
// Returns error which reports a stale record-board fixture edit, or nil on success.
func addPartialComponents(c *casefile.Case, p Project) error {
	input := casefile.PartialComponentsInput{
		Labels:        [2]string{p.Title + " A", p.Title + " B"},
		Edited:        p.Title + " revised",
		Query:         p.Title + " & items",
		Locale:        "en",
		Score:         1 + p.Count,
		EditSecond:    (c.Index+c.Seed)%2 != 0,
		ReloadModes:   true,
		ReverseReload: p.Followup%2 != 0,
	}
	if ((c.Index+c.Seed)/2)%2 != 0 {
		input.Locale = "fr"
	}
	if c.Index < partialReloadIndexes {
		input.ReverseReload = ((c.Index+c.Seed)/partialReloadPeriod)%2 != 0
	}
	addReloadModes(c)
	c.Expected.Client.PartialComponents = casefile.ExpectedPartialComponents(input)
	c.Files["config.json"] = `{"i18n":{"defaultLocale":"en","strategy":"prefix_except_default","locales":["en","fr"]}}`
	c.Files["pages/record-list.pk"] = recordsList
	c.Files["pages/record-detail.pk"] = recordsDetail
	c.Files["partials/record-rows.pk"] = recordsRows
	c.Files["components/record-chip.pkc"] = recordsRow
	c.Files["lib/record-icons/arrow.svg"] = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10"><path d="M1 1L9 9"/></svg>`
	selected := "amber"
	if input.EditSecond {
		selected = "blue"
	}
	store, err := recordboardStore(input, selected)
	if err != nil {
		return err
	}
	c.Files["pkg/recordboard/model.go"] = store
	c.Files["records-revision.txt"] = "0"
	c.Files["actions/records/advance.go"] = recordsAction
	for _, locale := range []string{"en", "fr"} {
		route := input
		route.Locale = locale
		contract := responseContract(map[string][]string{"lang": {locale}})
		for position, key := range []string{"amber", "blue"} {
			href := route.DetailPath(key)
			contract.Elements = append(contract.Elements, catalogueElement(key, "a", "Open "+key, "href", href))
			path, _, _ := strings.Cut(href, "#")
			detail := semanticProbe(path, responseContract(map[string][]string{"lang": {locale}}))
			detail.Text = key + " / " + input.Labels[position] + " / " + input.Query
			c.Probes = append(c.Probes, detail)
		}
		list := semanticProbe(route.ListPath(), contract)
		list.Text = "Record board"
		c.Probes = append(c.Probes, list)
	}
	c.Features = append(c.Features, "partial-owned-pkc-bindings", "partial-keyed-host-identity", "localized-record-navigation",
		"native-record-action", "partial-slot-event-recipient", "inline-record-svg")
	return nil
}
