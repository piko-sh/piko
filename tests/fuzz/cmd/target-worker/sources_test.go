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

package main

import (
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/compiler/compiler_domain"
	"piko.sh/piko/internal/sfcparser"
	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/tests/fuzz/internal/generate"
)

var (
	formatterSeeds   = [...]uint64{0, 1, 23, 55}
	formatterRecipes = []struct{ feature, name, marker string }{
		{feature: "formatter-timeline", name: "components/fuzz-counter.pkc", marker: "<piko:timeline"},
		{feature: "formatter-event", name: "components/fuzz-counter.pkc", marker: "p-event:item-picked.stop="},
		{feature: "formatter-native-event", name: "components/fuzz-counter.pkc", marker: "p-on:click.prevent.once="},
		{feature: "formatter-memo", name: "components/fuzz-counter.pkc", marker: "p-memo="},
		{feature: "formatter-slot-fill", name: "pages/main.pk", marker: "p-slot="},
	}
)

func FuzzFormatter(f *testing.F) {
	for _, seed := range formatterSeeds {
		savedCase := generate.New("formatter", seed, 0)
		for index := range sourceNames(savedCase) {
			f.Add(uint8(index), savedCase.Data)
		}
	}
	f.Fuzz(func(t *testing.T, which uint8, data []byte) {
		savedCase := generate.Replay("formatter", 0, 0, data[:min(len(data), 1024)])
		require.NoError(t, run(t.Context(), new(oneSource(savedCase, which))))
	})
}

func TestFormatterTimelineConservation(t *testing.T) {
	const template = `<template name="cue-player" enable="animation"><b p-ref="title" p-timeline:hidden>title</b><b p-ref="caption" p-timeline:anchor="title">caption</b></template>`
	const first = `<piko:timeline><piko:at time="1s"><piko:show ref="title"/><piko:hide ref="caption"/></piko:at></piko:timeline>`
	const second = `<piko:timeline media="(min-width: 600px)"><piko:at time="2s"><piko:show ref="caption"/></piko:at></piko:timeline>`
	const source = template + first + second
	before := parseSource(t, source)
	for _, tc := range []struct{ name, output, property string }{
		{name: "equivalent time spelling", output: strings.Replace(source, `"1s"`, `"1000ms"`, 1), property: ""},
		{name: "missing block", output: template + first, property: "timeline-block-count"},
		{name: "reordered blocks", output: template + second + first, property: "timeline-attributes"},
		{name: "changed viewport", output: strings.Replace(source, "600px", "640px", 1), property: "timeline-attributes"},
		{name: "missing marker", output: strings.Replace(source, " p-timeline:hidden", "", 1), property: "timeline-markers"},
		{name: "changed anchor", output: strings.Replace(source, `p-timeline:anchor="title"`, `p-timeline:anchor="caption"`, 1), property: "timeline-markers"},
		{name: "relocated marker", output: strings.Replace(strings.Replace(source, " p-timeline:hidden", "", 1), " p-timeline:anchor", " p-timeline:hidden p-timeline:anchor", 1), property: "timeline-markers"},
		{name: "changed action", output: strings.Replace(source, "piko:hide", "piko:show", 1), property: "timeline-actions"},
		{name: "reordered actions", output: strings.Replace(source, `<piko:show ref="title"/><piko:hide ref="caption"/>`, `<piko:hide ref="caption"/><piko:show ref="title"/>`, 1), property: "timeline-actions"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			requireProperty(t, checkFormattedTemplate(before, parseSource(t, tc.output)), tc.property)
		})
	}
}

func TestFormatterSlotMemoAndEventConservation(t *testing.T) {
	const fill = `<template><piko:partial is="frame"><b id="first" p-slot="caption">caption</b><b id="second">default</b></piko:partial></template>`
	const memo = `<template name="demo-list"><ul><li p-for="row in state.rows" p-key="row.id" p-memo="[row.version]">{{ row.label }}</li></ul></template>`
	const listeners = `p-event:pick.stop="choose($event)" p-event:pick.once="finish()"`
	const event = `<template name="demo-picker"><slot name="source" ` + listeners + `></slot><slot name="other"></slot></template>`
	const native = `<template><button p-on:click.prevent.once="choose">pick</button></template>`
	for _, tc := range []struct{ name, input, output, invariant string }{
		{name: "PK fill deletion", input: fill, output: strings.Replace(fill, ` p-slot="caption"`, "", 1), invariant: "slot-markers"},
		{name: "PK fill name", input: fill, output: strings.Replace(fill, `p-slot="caption"`, `p-slot="body"`, 1), invariant: "slot-markers"},
		{name: "PK fill owner", input: fill, output: strings.Replace(strings.Replace(fill, ` p-slot="caption"`, "", 1), `id="second"`, `id="second" p-slot="caption"`, 1), invariant: "slot-markers"},
		{name: "PKC memo deletion", input: memo, output: strings.Replace(memo, ` p-memo="[row.version]"`, "", 1), invariant: "memo-markers"},
		{name: "PKC memo expression", input: memo, output: strings.Replace(memo, "row.version", "row.id", 1), invariant: "memo-markers"},
		{name: "PKC event deletion", input: event, output: strings.Replace(event, listeners, "", 1), invariant: "event-markers"},
		{name: "PKC event argument", input: event, output: strings.Replace(event, ":pick.stop", ":save.stop", 1), invariant: "event-markers"},
		{name: "PKC event modifier", input: event, output: strings.Replace(event, ":pick.stop", ":pick.self", 1), invariant: "event-markers"},
		{name: "PKC event handler", input: event, output: strings.Replace(event, "choose($event)", "finish()", 1), invariant: "event-markers"},
		{name: "PKC handler order", input: event, output: strings.Replace(event, listeners, `p-event:pick.once="finish()" p-event:pick.stop="choose($event)"`, 1), invariant: "event-markers"},
		{name: "PKC event owner", input: event, output: strings.Replace(strings.Replace(event, listeners, "", 1), `name="other"`, `name="other" `+listeners, 1), invariant: "event-markers"},
		{name: "native once removed", input: native, output: strings.Replace(native, ".prevent.once", ".prevent", 1), invariant: "on-markers"},
		{name: "native modifiers reordered", input: native, output: strings.Replace(native, ".prevent.once", ".once.prevent", 1), invariant: "on-markers"},
		{name: "equivalent native spelling", input: native, output: strings.Replace(native, `p-on:click.prevent.once="choose"`, `p-on:click.prevent.once='cho&#111;se'`, 1), invariant: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			requireProperty(t, checkFormattedTemplate(parseSource(t, tc.input), parseSource(t, tc.output)), tc.invariant)
		})
	}
}

func TestGeneratedFormatterDirectiveReplay(t *testing.T) {
	counts := make(map[string]int)
	for index := range uint64(32) {
		savedCase := generate.New("formatter", 7, index)
		replay := generate.Replay(savedCase.Target, savedCase.Seed, savedCase.Index, savedCase.Data)
		require.Equal(t, savedCase.Hash(), replay.Hash(), "case %d did not replay", index)
		for _, recipe := range formatterRecipes {
			source := replay.Files[recipe.name]
			selected := slices.Contains(replay.Features, recipe.feature)
			require.Equal(t, selected, strings.Contains(source, recipe.marker), "case %d has inconsistent feature %s", index, recipe.feature)
			if !selected {
				continue
			}
			counts[recipe.feature]++
			if recipe.feature == "formatter-timeline" {
				require.Equal(t, 2, strings.Count(source, "<piko:timeline"), "case %d lost its timeline recipe", index)
				require.Contains(t, source, `p-timeline:anchor="headline"`, "case %d lost its timeline recipe", index)
			}
		}
		require.NoError(t, checkFormatter(t.Context(), nil, &replay), "case %d", index)
	}
	for _, recipe := range formatterRecipes {
		assert.NotZero(t, counts[recipe.feature], "expected selected cases for %s, got %v", recipe.feature, counts)
		assert.NotEqual(t, 32, counts[recipe.feature], "expected unselected cases for %s, got %v", recipe.feature, counts)
	}
}

func TestNativeFormatterSeedsCoverDirectives(t *testing.T) {
	covered := make(map[string]bool)
	for _, seed := range formatterSeeds {
		savedCase := generate.New("formatter", seed, 0)
		for _, recipe := range formatterRecipes {
			if strings.Contains(savedCase.Files[recipe.name], recipe.marker) {
				covered[recipe.feature] = true
			}
		}
	}
	for _, recipe := range formatterRecipes {
		assert.True(t, covered[recipe.feature], "native formatter seed corpus is missing %s", recipe.feature)
	}
}

func TestPreformattedTextConservation(t *testing.T) {
	original, err := preformattedText("<pre>left  one\n  <b>right</b></pre>")
	require.NoError(t, err)
	assert.Equal(t, []string{"left  one\n  right"}, original)

	indented, err := preformattedText("<pre>left  one\n    <b>right</b></pre>")
	require.NoError(t, err)
	assert.NotEqual(t, original, indented)
}

func TestFormatterDistinguishesLiteralTextFromMarkup(t *testing.T) {
	content := func(source string) []string {
		t.Helper()
		snapshot, err := templateContent(source)
		require.NoError(t, err)
		return snapshot.content
	}
	want := content("<ul><li>&lt;text&gt;</li></ul>")
	assert.Equal(t, want, content("<ul>\n  <li>&lt;text&gt;</li>\n</ul>"), "layout indentation changed the semantic projection")
	assert.NotEqual(t, want, content("<ul><li><text></text></li></ul>"), "literal text changed to markup without being detected")
}

func TestSameTranslations(t *testing.T) {
	assert.True(t, sameTranslations(`{"a":1,"b":2}`, `{"b":2, "a":1}`))
	assert.False(t, sameTranslations(`{"a":1}`, `{"a":1.0}`))
	assert.False(t, sameTranslations(`{"a":1}`, `{"a":`))
}

func TestSourceSemanticControls(t *testing.T) {
	const template = `<template><p>Buy <em>two</em> today.</p><button disabled aria-label="Accept" :title="state.Hint" p-on:click.prevent.once="save">Save</button></template>`
	const translations = `<i18n lang="json">{"en":{"save":"Save","tip":"Keep"}}</i18n>`
	const source = template + translations
	model := func(input string) []observedNode {
		t.Helper()
		tree, err := ast_domain.Parse(t.Context(), parseSource(t, input).Template, "controls.pk", nil)
		require.NoError(t, err)
		return projection(tree.RootNodes)
	}
	for _, tc := range []struct {
		name, from, to, invariant string
		differentAST              bool
	}{
		{name: "attribute quote", from: `aria-label="Accept"`, to: `aria-label='Accept'`, invariant: "", differentAST: false},
		{name: "boolean presence", from: " disabled", to: "", invariant: "template-attributes", differentAST: true},
		{name: "ordinary value", from: `aria-label="Accept"`, to: `aria-label="Reject"`, invariant: "template-attributes", differentAST: true},
		{name: "binding expression", from: "state.Hint", to: "state.Other", invariant: "template-attributes", differentAST: true},
		{name: "remove modifier", from: ".prevent.once", to: ".prevent", invariant: "on-markers", differentAST: true},
		{name: "reorder modifiers", from: ".prevent.once", to: ".once.prevent", invariant: "on-markers", differentAST: true},
		{name: "inline leading separator", from: "Buy <em>", to: "Buy<em>", invariant: "inline-space", differentAST: true},
		{name: "inline trailing separator", from: "</em> today", to: "</em>today", invariant: "inline-space", differentAST: true},
		{name: "translation order", from: `"save":"Save","tip":"Keep"`, to: `"tip":"Keep", "save":"Save"`, invariant: "", differentAST: false},
		{name: "translation content", from: `"tip":"Keep"`, to: `"tip":"Discard"`, invariant: "i18n-content", differentAST: false},
		{name: "translation language", from: `lang="json"`, to: `lang="yaml"`, invariant: "i18n-attributes", differentAST: false},
		{name: "lost translations", from: translations, to: "", invariant: "i18n-blocks", differentAST: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			changed := strings.Replace(source, tc.from, tc.to, 1)
			requireProperty(t, checkFormattedTemplate(parseSource(t, source), parseSource(t, changed)), tc.invariant)
			assert.Equal(t, tc.differentAST, !reflect.DeepEqual(model(source), model(changed)))
		})
	}
}

func TestSyntaxAgreesWithCompilerComponentNaming(t *testing.T) {
	compiler := compiler_domain.NewCompilerOrchestrator(nil, nil, compiler_domain.WithOrchestratorModuleName(casefile.Module))
	for _, tc := range []struct{ name, path, attribute, want string }{
		{name: "filename", path: "components/widgets/client-button.pkc", attribute: "", want: "client-button"},
		{name: "empty explicit name", path: "components/client-button.pkc", attribute: ` name=""`, want: "client-button"},
		{name: "explicit override", path: "components/ordinary.pkc", attribute: ` name="guest-button"`, want: "guest-button"},
		{name: "invalid filename", path: "components/button.pkc", attribute: "", want: ""},
		{name: "invalid explicit override", path: "components/valid-name.pkc", attribute: ` name="button"`, want: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := "<template" + tc.attribute + "><button>Pick</button></template>"
			result, compileErr := compiler.CompileSFCBytes(t.Context(), tc.path, []byte(source))
			checkErr := checkSyntax(t.Context(), nil, &casefile.Case{Files: map[string]string{tc.path: source}})
			if tc.want == "" {
				require.Error(t, compileErr)
				requireProperty(t, checkErr, "name")
				return
			}
			require.NoError(t, compileErr)
			require.NoError(t, checkErr)
			require.NotNil(t, result)
			assert.Equal(t, tc.want, result.TagName)
		})
	}
	seen := false
	for index := range uint64(32) {
		savedCase := generate.New("syntax", 7, index)
		if strings.Contains(savedCase.Files["components/fuzz-counter.pkc"], `<template>`) {
			seen = true
			require.NoError(t, checkSyntax(t.Context(), nil, &savedCase))
			break
		}
	}
	assert.True(t, seen, "filename-derived case unreachable")
}

func TestGeneratedRouteContractSources(t *testing.T) {
	savedCase := generate.New("route-contracts", 7, 0)
	require.NoError(t, checkSyntax(t.Context(), nil, &savedCase))
	for _, want := range []string{"/edition-expressions?order=forward", "/edition-expressions?order=reverse", "/bulletins/featured"} {
		assert.True(t, slices.ContainsFunc(savedCase.Probes, func(probe casefile.Probe) bool { return probe.Path == want }), "missing %s", want)
	}
	assert.LessOrEqual(t, len(savedCase.Probes), 16, "case exceeds the shared HTTP probe limit")
}

func TestEachSourceRequiresPikoSources(t *testing.T) {
	err := eachSource(&casefile.Case{Files: map[string]string{"README.md": "text"}}, func(string, string, *sfcparser.ParseResult) error {
		return nil
	})

	require.ErrorIs(t, err, errNoPikoSources)
}

func parseSource(t *testing.T, source string) *sfcparser.ParseResult {
	t.Helper()
	blocks, err := sfcparser.Parse([]byte(source))
	require.NoError(t, err)
	return blocks
}

func requireProperty(t *testing.T, err error, want string) {
	t.Helper()
	if want == "" {
		require.NoError(t, err)
		return
	}
	failure, ok := errors.AsType[*propertyError](err)
	require.True(t, ok, "got %v, want property %q", err, want)
	require.Equal(t, want, failure.ID, "got %v", err)
}
