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

//go:build fuzzintegration

package runner

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/tests/fuzz/internal/generate"
)

var (
	errPilotTimeout       = errors.New("pilot: application run exceeded its deadline")
	errCanaryTimeout      = errors.New("visual source canaries: application run exceeded its deadline")
	errServerModesTimeout = errors.New("server modes: application run exceeded its deadline")
)

func TestPilot(t *testing.T) {
	repo, err := filepath.Abs("../../../..")
	require.NoError(t, err)
	project := generate.Replay("app", 0, 0, nil)
	rich := generate.Replay("components", 0, 0, []byte{8: 255, 13: 1, 14: 2, 15: 255, 16: 255, 17: 255, 18: 255, 19: 255})
	cases := map[string]casefile.Case{"http-actions": generate.New("http-actions", 7, 0), "pair": project, "rich-component": rich}
	for index := range uint64(128) {
		candidate := generate.New("components", 7, index)
		steps := candidate.Expected.Client.Steps
		if steps[0].Action.Op == casefile.SetLabel && steps[0].Action.Instance == 0 &&
			steps[1].Action.Op == casefile.RemoveRow && steps[2].Action.Op == casefile.ParentClick {
			cases["reordered-component"] = candidate
			break
		}
	}
	_, found := cases["reordered-component"]
	require.True(t, found, "missing alternate component recipe")

	for index := range uint64(128) {
		candidate := generate.New("components", 31, index)
		for _, feature := range candidate.Features {
			if !strings.HasPrefix(feature, "client-extra-") {
				continue
			}
			if _, exists := cases[feature]; !exists {
				candidate.Probes = nil
				cases[feature] = candidate
			}
		}
	}

	for _, fixture := range []struct {
		name, title string
		body        []string
	}{
		{name: "01_static_page", title: "Hello, Piko!", body: []string{"This is a completely static page rendered by the interpreted runner."}},
		{name: "02_simple_dynamic_page", title: "Dynamic Page Title", body: []string{"The message is: This content comes from the interpreted Go Render function.", "Request Path: /main", "<text> & more", "Label: <text> & more"}},
	} {
		path := filepath.Join(repo, "tests/integration/interpreted_runner/testdata", fixture.name, "src/pages/main.pk")
		body, err := os.ReadFile(path)
		require.NoError(t, err)
		cases[fixture.name] = casefile.Case{Files: map[string]string{"pages/main.pk": string(body)}, Request: "/main", Expected: casefile.Expected{Status: http.StatusOK, Title: fixture.title, Rows: fixture.body}}
	}
	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			var results []Observation
			for _, mode := range []string{"interp", "compiled"} {
				t.Run(mode, func(t *testing.T) {
					ctx, cancel := context.WithTimeoutCause(t.Context(), 6*time.Minute, errPilotTimeout)
					o, err := RunWithWorkers(ctx, repo, t.TempDir(), "", &testCase, mode, testCase.Expected.Tag != "" && testCase.Expected.Action == nil)
					cancel()
					require.NoError(t, err, "%s\npage: %s\nworker: %s\naction: %+v", mode, o.RawHTML, o.WorkerLog, o.Action)
					require.NoError(t, Check(&testCase, &o), "%s\nclient: %+v\npage: %s", mode, o.Client, o.RawHTML)
					results = append(results, o)
				})
			}
			if len(results) != 2 {
				return
			}
			require.NoError(t, compareObservations(&results[0], &results[1]))
		})
	}
}

func TestVisualSourceCanaries(t *testing.T) {
	repo, err := filepath.Abs("../../../..")
	require.NoError(t, err)
	testCase := generate.New("components", 31, 2)
	testCase.Probes = nil
	const file = "components/fuzz-counter.pkc"
	for _, mutation := range [][2]string{
		{`p-html="state.panel ? state.openMarkup : state.closedMarkup"`, `p-text="state.panel ? state.openMarkup : state.closedMarkup"`},
		{`:data-phase="state.panel ? 'open' : 'shut'"`, `data-phase="open"`},
		{`p-bind:data-note="state.draft"`, `data-note=""`},
		{`p-else-if="state.panel"`, `p-else-if="true"`},
		{`p-context="row.scope"`, `p-context="'one-namespace'"`},
	} {
		require.Contains(t, testCase.Files[file], mutation[0], "missing source production")
		testCase.Files[file] = strings.Replace(testCase.Files[file], mutation[0], mutation[1], 1)
	}
	ctx, cancel := context.WithTimeoutCause(t.Context(), 3*time.Minute, errCanaryTimeout)
	defer cancel()
	o, err := RunWithWorkers(ctx, repo, t.TempDir(), "", &testCase, "compiled", true)
	require.NoError(t, err)
	require.Error(t, Check(&testCase, &o), "broken component unexpectedly passed the browser oracle")
	require.NotNil(t, o.Client, "broken component unexpectedly passed the browser oracle")
	for index, step := range testCase.Expected.Client.Steps {
		if step.Action.Op != casefile.HideMemo {
			continue
		}
		instance := step.Action.Instance
		for _, field := range []string{"markup", "phase-attr", "note-attr", "branch", "context-identity"} {
			observed := o.Client.Steps[index].After[instance].Advanced[field]
			assert.NotEqual(t, step.After[instance].Advanced[field], observed, "%s retained expected observation with broken source", field)
		}
		return
	}
	require.Fail(t, "missing hide interaction")
}

func TestTypedCataloguePilot(t *testing.T) {
	for _, index := range []uint64{1, 0} {
		checkServerModes(t, "typed-catalog", index)
	}
}

func TestInterpretedHostParityPilot(t *testing.T) {
	for _, target := range []string{"pages", "http-actions", "graph-rebuild"} {
		t.Run(target, func(t *testing.T) { checkServerModes(t, target, 1) })
	}
}

func TestCatalogueRebuildPilot(t *testing.T) { checkServerModes(t, "catalog-rebuild", 0) }

func TestLiveCatalogueWatcher(t *testing.T) {
	repo, err := filepath.Abs("../../../..")
	require.NoError(t, err)
	testCase := generate.New("catalog-rebuild", 7, 0)
	workers := t.TempDir()
	for _, variant := range []struct {
		name    string
		browser bool
	}{{name: "http", browser: false}, {name: "chromium", browser: true}} {
		t.Run(variant.name, func(t *testing.T) {
			o, err := RunWithWorkers(t.Context(), repo, t.TempDir(), workers, &testCase, "interp", variant.browser)
			for _, event := range o.CatalogWatch {
				t.Log(event)
			}
			require.NoError(t, err, "live watcher\n%s", o.WorkerLog)
			require.Contains(t, strings.Join(o.CatalogWatch, "\n"), `msg="Starting targeted rebuild"`, "missing rebuild triggered by the watched edit in receipt")
			require.NoError(t, Check(&testCase, &o))
		})
	}
}

func TestLegacyErrorHostingPilot(t *testing.T) {
	repo, err := filepath.Abs("../../../..")
	require.NoError(t, err)
	testCase := generate.New("pages", 7, 0)
	require.NotEmpty(t, testCase.Files["pages/missing.pk"], "seed 7 index 0 must still cover the typed error fixture")
	delete(testCase.Files, "pages/!404.pk")
	testCase.Probes = []casefile.Probe{{Path: "/missing", Status: http.StatusNotFound}, {Path: "/missing-route", Status: http.StatusNotFound}}
	testCase.Features = append(testCase.Features, "unmatched-route")
	var hosts []Observation
	for _, mode := range []string{"interp", "compiled"} {
		o, err := RunWithWorkers(t.Context(), repo, t.TempDir(), "", &testCase, mode, false)
		require.NoError(t, err)
		require.NoError(t, Check(&testCase, &o), "%s; probes %+v", mode, o.Probes)
		hosts = append(hosts, o)
	}
	require.NoError(t, CompareCase(&testCase, &hosts[0], &hosts[1]))
}

func checkServerModes(t *testing.T, target string, index uint64) {
	t.Helper()
	repo, err := filepath.Abs("../../../..")
	require.NoError(t, err)
	testCase := generate.New(target, 7, index)
	var observed []Observation
	for _, mode := range []string{"compiled", "interp"} {
		t.Logf("executing target %s seed 7 index %d mode %s", target, index, mode)
		ctx, cancel := context.WithTimeoutCause(t.Context(), 4*time.Minute, errServerModesTimeout)
		o, err := RunWithWorkers(ctx, repo, t.TempDir(), "", &testCase, mode, false)
		cancel()
		require.NoError(t, err, "seed 7 index %d mode %s\n%s", index, mode, o.WorkerLog)
		if err := Check(&testCase, &o); err != nil {
			for _, probe := range o.Probes {
				t.Logf("probe %s %s %d: %.2000s", probe.Actor, probe.Path, probe.Status, probe.RawHTML)
			}
			require.NoError(t, err, "seed 7 index %d mode %s\n%s", index, mode, o.WorkerLog)
		}
		if target == "catalog-rebuild" {
			var hashes []string
			for stepIndex, step := range o.CatalogRebuild {
				require.Less(t, stepIndex, 3, "invalid rebuild step %+v", step)
				require.Equal(t, []string{"baseline", "edit-card", "undo-card"}[stepIndex], step.Step, "invalid rebuild step %+v", step)
				require.Equal(t, step.CleanSHA256, step.WarmSHA256, "invalid rebuild step %+v", step)
				hashes = append(hashes, step.WarmSHA256)
			}
			require.Len(t, hashes, 3, "catalogue rebuild history: %v", hashes)
			require.Equal(t, hashes[0], hashes[2], "catalogue rebuild history: %v", hashes)
			require.NotEqual(t, hashes[0], hashes[1], "catalogue rebuild history: %v", hashes)
		}
		observed = append(observed, o)
	}
	require.NoError(t, CompareCase(&testCase, &observed[0], &observed[1]), "seed 7 index %d", index)
}
