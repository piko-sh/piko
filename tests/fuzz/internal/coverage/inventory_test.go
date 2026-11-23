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

package coverage

import (
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/json"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/tests/fuzz/internal/generate"
	"piko.sh/piko/tests/fuzz/internal/runner"
)

func TestGeneratedDirectiveCoverage(t *testing.T) {
	inventory := New("components", "every", 7, 0)
	for index := range uint64(100) {
		require.NoError(t, inventory.Add(t.Context(), new(generate.New("components", 7, index)), true), "case %d", index)
	}
	report := inventory.Result()
	assert.Len(t, report.Directives, len(ast_domain.DirectiveNameToType))
	assert.GreaterOrEqual(t, report.Server.DistinctTemplateShapes, 40)
	assert.GreaterOrEqual(t, report.Client.DistinctTemplateShapes, 4)
	assert.Equal(t, ProgramCounts{Cases: 1, Requests: report.Programs["pages/arithmetic.pk"].Requests, ExpectedOutputs: 486, MatchedOutputs: 486, SourceOutputIDs: 18}, report.Programs["pages/arithmetic.pk"])
	for _, name := range []string{"p-if", "p-else-if", "p-else", "p-for", "p-text", "p-bind"} {
		assert.NotZero(t, lookup(t, report, name).Server.OracleEligibleCases, "lost server production %s", name)
	}
	for _, name := range []string{"p-if", "p-for", "p-key", "p-model", "p-on", "p-text", "p-show", "p-ref", "p-class", "p-style", "p-memo", "p-event", "p-timeline", "p-html", "p-else-if", "p-else", "p-bind", "p-context"} {
		assert.NotZero(t, lookup(t, report, name).Client.OracleEligibleCases, "lost client production %s", name)
	}
	assert.Zero(t, lookup(t, report, "p-key").Server.OracleEligibleCases)
	assert.Contains(t, report.Server.SourceOnly, "p-key")
	for _, op := range []casefile.ClientOp{casefile.SetInput, casefile.SetLabel, casefile.ReverseRows, casefile.RemoveRow, casefile.ModifiedClick, casefile.ParentClick, casefile.ToggleCheck, casefile.SelectOption, casefile.ResetControls} {
		assert.Equal(t, 100, report.ClientActions[op].RecipeCases, "shared operation %s", op)
	}
	for op, counts := range report.ClientActions {
		assert.NotZero(t, counts.RecipeCases, "lost %s recipe", op)
		assert.Equal(t, counts.RecipeCases, counts.OracleEligibleCases, "lost %s selector or oracle", op)
	}
}

func TestResultIsIsolatedFromLaterAdds(t *testing.T) {
	inventory := New("components", "every", 7, 0)
	require.NoError(t, inventory.Add(t.Context(), new(generate.New("components", 7, 0)), true))
	first := inventory.Result()
	snapshot, err := json.Marshal(first)
	require.NoError(t, err)
	require.NoError(t, inventory.Add(t.Context(), new(generate.New("components", 7, 1)), true))
	after, err := json.Marshal(first)
	require.NoError(t, err)
	assert.JSONEq(t, string(snapshot), string(after), "later Add mutated a returned report")
	assert.Equal(t, 2, inventory.Result().Cases)
}

func TestAddRejectsMixedProfilesAndOverflow(t *testing.T) {
	inventory := New("components", "off", 7, 0)
	fuzzCase := generate.New("components", 7, 0)
	require.NoError(t, inventory.Add(t.Context(), &fuzzCase, false))
	other := fuzzCase
	other.ProfileSHA256 = "different"
	require.ErrorIs(t, inventory.Add(t.Context(), &other, false), ErrMixedProfiles)
	inventory.Cases = MaxCases
	require.ErrorIs(t, inventory.Add(t.Context(), &fuzzCase, false), ErrInventoryFull)
}

func TestAddReportsTemplateDiagnostics(t *testing.T) {
	fuzzCase := casefile.Case{Target: "pages", Files: map[string]string{
		"pages/broken.pk": `<template><div p-if="(("></div></template>`,
	}}
	err := New("pages", "off", 0, 0).Add(t.Context(), &fuzzCase, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "pages/broken.pk")
}

func TestOnlySourceAttributesAndReachedLeavesCount(t *testing.T) {
	fuzzCase := casefile.Case{Target: "pages", Features: []string{"p-model", "p-slot", "p-else"}, Files: map[string]string{
		"pages/program.pk":      `<template><main id="fuzz-program"><div p-if="true"><output data-fuzz-output="yes" :data-value="title" p-text="counter"></output></div><div p-else><output data-fuzz-output="no" p-html="html"></output></div><!-- p-slot --></main></template><script lang="ts">const text = 'p-model';</script>`,
		"pages/unvisited.pk":    `<template><main id="fuzz-program"><output data-fuzz-output="yes" p-html="html"></output></main></template>`,
		"components/widget.pkc": `<template name="widget-x"><input p-model="value"></template>`,
	}, Probes: []casefile.Probe{{Program: true, Path: "/program?input=left", Status: http.StatusOK, Output: []casefile.ProgramOutput{{ID: "yes", Text: "1", HasValue: true, Value: "title"}}}}}
	inventory := New("pages", "off", 0, 0)
	require.NoError(t, inventory.Add(t.Context(), &fuzzCase, false))
	report := inventory.Result()
	assert.Equal(t, 1, lookup(t, report, "p-if").Server.OracleEligibleCases)
	assert.Equal(t, 1, lookup(t, report, "p-bind").Server.OracleEligibleCases)
	assert.Equal(t, 1, lookup(t, report, "p-text").Server.OracleEligibleCases)
	assert.Zero(t, lookup(t, report, "p-else").Server.OracleEligibleCases)
	assert.Equal(t, 1, lookup(t, report, "p-else").Server.SourceCases)
	assert.Equal(t, 1, lookup(t, report, "p-model").Client.SourceCases)
	assert.Zero(t, lookup(t, report, "p-model").Client.OracleEligibleCases)
	assert.Zero(t, lookup(t, report, "p-slot").Server.SourceCases)
	assert.Zero(t, lookup(t, report, "p-model").Server.SourceCases)
}

func TestRemovedControlsLoseOracleCreditEvenWithIntactFeatureLabels(t *testing.T) {
	fuzzCase := generate.New("components", 7, 0)
	path := "components/fuzz-counter.pkc"
	fuzzCase.Files[path] = strings.ReplaceAll(fuzzCase.Files[path], `p-key="item"`, "")
	fuzzCase.Files[path] = strings.ReplaceAll(fuzzCase.Files[path], "p-on:click.prevent.stop=", "p-on:click=")
	fuzzCase.Files[path] = strings.ReplaceAll(fuzzCase.Files[path], "p-model=", "data-unused=")
	inventory := New(fuzzCase.Target, "every", 7, 0)
	require.NoError(t, inventory.Add(t.Context(), &fuzzCase, true))
	report := inventory.Result()
	for _, name := range []string{"p-model", "p-key"} {
		row := lookup(t, report, name)
		assert.Zero(t, row.Client.SourceCases, name)
		assert.Zero(t, row.Client.OracleEligibleCases, name)
	}
	for _, op := range []casefile.ClientOp{casefile.ModifiedClick, casefile.SetInput, casefile.ToggleCheck, casefile.SelectOption} {
		assert.Equal(t, ActionCounts{RecipeCases: 1, OracleEligibleCases: 0}, report.ClientActions[op], op)
	}
}

func TestClaimedChecksRejectWrongObservation(t *testing.T) {
	fuzzCase := generate.New("components", 7, 0)
	base := matchingObservation(fuzzCase)
	require.NoError(t, runner.Check(&fuzzCase, &base))
	data, err := json.Marshal(base)
	require.NoError(t, err)
	tests := []observationChange{
		{name: "server if", phase: "conditional:", mutate: func(o *runner.Observation) { o.Visible = !o.Visible }},
		{name: "server for", phase: "rows:", mutate: func(o *runner.Observation) { o.Rows = append(o.Rows, "unexpected row") }},
		{name: "client click and text", phase: "browser/count:", mutate: func(o *runner.Observation) { o.Client.After[0].Count = "missing count" }},
		{name: "client for", phase: "browser/rows:", mutate: func(o *runner.Observation) { o.Client.After[0].Rows = nil }},
		{name: "client if", phase: "browser/condition:", mutate: func(o *runner.Observation) { o.Client.After[0].Visible = !o.Client.After[0].Visible }},
		{name: "keyed reorder", phase: "browser/key-identity:", mutate: func(o *runner.Observation) {
			for _, step := range o.Client.Steps {
				if step.Action.Op == casefile.ReverseRows {
					step.After[step.Action.Instance].Controls.Keys = nil
					return
				}
			}
		}},
		{name: "p-model", phase: "browser/input:", mutate: func(o *runner.Observation) {
			for _, step := range o.Client.Steps {
				if step.Action.Op == casefile.SetInput {
					step.After[step.Action.Instance].Controls.Input += "?"
					return
				}
			}
		}},
		{name: "event modifiers", phase: "browser/prevent:", mutate: func(o *runner.Observation) {
			for index, step := range o.Client.Steps {
				if step.Action.Op == casefile.ModifiedClick {
					o.Client.Steps[index].Prevented = false
					return
				}
			}
		}},
		{name: "reflected host attribute", phase: "browser/reflection:", mutate: func(o *runner.Observation) {
			for _, step := range o.Client.Steps {
				if step.Action.Op == casefile.SetLabel {
					step.After[step.Action.Instance].Controls.HostLabel += "?"
					return
				}
			}
		}},
	}
	for _, field := range []string{"id", "text", "binding", "presence"} {
		tests = append(tests, observationChange{name: "program " + field, phase: "probe/program:", mutate: func(o *runner.Observation) {
			for index, probe := range o.Probes {
				if !probe.Program || len(probe.Output) == 0 {
					continue
				}
				leaf := &o.Probes[index].Output[0]
				switch field {
				case "id":
					leaf.ID += "other"
				case "text":
					leaf.Text += "other"
				case "binding":
					leaf.Value += "other"
				case "presence":
					leaf.HasValue = !leaf.HasValue
				}
				return
			}
		}})
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			var observed runner.Observation
			require.NoError(t, json.Unmarshal(data, &observed))
			testCase.mutate(&observed)
			failures := runner.IndependentFailures(&fuzzCase, &observed)
			assert.True(t, slices.ContainsFunc(failures, func(err error) bool { return strings.HasPrefix(err.Error(), testCase.phase) }), "wanted %s, got %v", testCase.phase, failures)
		})
	}
}

func TestBrowserOffPreservesSourceAndSkipsAllClientOracles(t *testing.T) {
	inventory := New("components", "off", 7, 0)
	require.NoError(t, inventory.Add(t.Context(), new(generate.New("components", 7, 0)), false))
	report := inventory.Result()
	assert.Zero(t, report.BrowserSelected)
	assert.GreaterOrEqual(t, report.Client.Seen, 4)
	assert.Zero(t, report.Client.OracleEligible)
	assert.NotZero(t, report.Server.OracleEligible)
	for op, counts := range report.ClientActions {
		assert.Zero(t, counts.OracleEligibleCases, op)
		assert.LessOrEqual(t, counts.RecipeCases, 1, op)
	}
}

func TestGeneratedPageKeyHTMLAndSlotCoverage(t *testing.T) {
	inventory := New("pages", "off", 7, 0)
	for index := range uint64(2) {
		require.NoError(t, inventory.Add(t.Context(), new(generate.New("pages", 7, index)), false))
	}
	for _, name := range []string{"p-context", "p-key", "p-html", "p-slot"} {
		assert.Equal(t, 1, lookup(t, inventory.Result(), name).Server.OracleEligibleCases, name)
	}
}

type observationChange struct {
	mutate func(*runner.Observation)
	name   string
	phase  string
}

func lookup(t *testing.T, report Report, name string) Directive {
	t.Helper()
	index := slices.IndexFunc(report.Directives, func(row Directive) bool { return row.Name == name })
	require.GreaterOrEqual(t, index, 0, "unreported registered directive %s", name)
	return report.Directives[index]
}

func matchingObservation(fuzzCase casefile.Case) runner.Observation {
	expected := fuzzCase.Expected
	observation := runner.Observation{Status: expected.Status, ContentType: "text/html", Heading: expected.Title, MetadataTitle: expected.Title, Rows: expected.Rows, Visible: expected.Show, Frame: expected.Frame, Component: true, ComponentCompiled: true, ComponentAsset: "component.js", Client: expected.Client}
	client := *expected.Client
	client.BrowserRoutes = []casefile.BrowserRouteVisit{{Path: fuzzCase.Request, Landed: fuzzCase.Request, Status: expected.Status}}
	observation.Client = &client
	observation.Semantic = expected.Semantic
	observation.Mode = "interp"
	for _, probe := range fuzzCase.Probes {
		status, semantic := probe.Status, probe.Semantic
		if override, ok := probe.ByMode[observation.Mode]; ok {
			status = override.Status
			semantic = &override.Semantic
		}
		if probe.BrowserProgram {
			client.BrowserRoutes = append(client.BrowserRoutes, casefile.BrowserRouteVisit{Path: probe.Path, Landed: probe.Path, Status: status, Program: probe.Program, ProgramRoot: probe.ProgramRoot, Output: probe.Output})
		}
		observation.Probes = append(observation.Probes, runner.ProbeObservation{Path: probe.Path, Status: status, Location: probe.Location, Text: probe.Text, Program: probe.Program, ProgramRoot: probe.ProgramRoot, Semantic: semantic, Output: probe.Output})
	}
	return observation
}
