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

package campaign

import (
	"context"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/json"
	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/tests/fuzz/internal/generate"
	"piko.sh/piko/tests/fuzz/internal/runner"
)

func TestReducePreservesTheFailure(t *testing.T) {
	dir := t.TempDir()
	c := generate.New("pages", 7, 0)
	c.Data = append(c.Data, byte(87))
	failed := func(c *casefile.Case) Result {
		result := Result{Target: c.Target, Mode: "interp", Index: c.Index, Kind: "expectation", Phase: "title"}
		switch {
		case strings.Contains(string(c.Data), "W"):
		case len(c.Data) == 0:
			result.Kind = "resource"
		default:
			result.Phase = "rows"
		}
		return result
	}
	h := Harness{Budget: testBudget{}, trial: func(_ context.Context, c *casefile.Case, _, _ string) Result { return failed(c) }}
	require.NoError(t, casefile.Write(filepath.Join(dir, "original"), &c))
	before := failed(&c)
	require.NoError(t, saveJSON(filepath.Join(dir, "original", "original.json"), before))
	out := filepath.Join(dir, "small")
	_, err := h.Reduce(t.Context(), filepath.Join(dir, "original"), out, &c, Options{Mode: "both", Browser: "off"})
	require.NoError(t, err)
	small, err := casefile.Read(out)
	require.NoError(t, err)
	assert.Equal(t, []byte{87}, small.Data)
	assert.Equal(t, before.Fingerprint(), failed(&small).Fingerprint())
}

func TestReduceRefusesAnExistingDestination(t *testing.T) {
	out := t.TempDir()
	_, err := Harness{Budget: testBudget{}}.Reduce(t.Context(), t.TempDir(), out, new(generate.New("pages", 7, 0)), Options{Mode: "both", Browser: "off"})
	require.ErrorIs(t, err, errReductionExists)
}

func TestReduceReplaysAndReducesArchivedSource(t *testing.T) {
	c := generate.New("pages", 7, 1)
	c.Generator = "retired-generator/1"
	c.ProfileSHA256 = "retired-profile"
	const path = "pages/from-archive.pk"
	const source = "<template><output>archived observation</output></template>"
	c.Files[path] = source
	dir := filepath.Join(t.TempDir(), "failure")
	require.NoError(t, casefile.Write(dir, &c))
	replayed, err := casefile.Read(dir)
	require.NoError(t, err)
	want := Result{Kind: "expectation", Target: "pages", Mode: "compiled", Phase: "probe/program", Subject: path}
	var selectionErrors []string
	h := Harness{Budget: testBudget{}, trial: func(_ context.Context, candidate *casefile.Case, mode, browser string) Result {
		if mode != "compiled" || browser != "off" || candidate.Generator != c.Generator || candidate.ProfileSHA256 != c.ProfileSHA256 {
			selectionErrors = append(selectionErrors, mode+" "+browser+" "+candidate.Generator+" "+candidate.ProfileSHA256)
		}
		if candidate.Files[path] != source {
			other := want
			other.Subject = "pages/another-output.pk"
			return other
		}
		return want
	}}
	assert.True(t, want.matches(h.RunOne(t.Context(), &replayed, "compiled", "off")))
	require.NoError(t, saveJSON(filepath.Join(dir, "original.json"), want))
	callerHash := replayed.Hash()
	out := filepath.Join(t.TempDir(), "minimal")
	_, err = h.Reduce(t.Context(), dir, out, &replayed, Options{Mode: "compiled", Browser: "off"})
	require.NoError(t, err)
	assert.Empty(t, selectionErrors, "lost archived selection or identity")
	assert.Equal(t, callerHash, replayed.Hash(), "candidate file deletion mutated the caller's archived source map")
	small, err := casefile.Read(out)
	require.NoError(t, err)
	assert.Equal(t, source, small.Files[path])
	assert.Equal(t, c.Generator, small.Generator)
	assert.True(t, want.matches(h.RunOne(t.Context(), &small, "compiled", "off")))
	original, err := casefile.Read(dir)
	require.NoError(t, err)
	assert.Equal(t, c.Hash(), original.Hash())
	assert.Less(t, len(small.Files), len(original.Files))
}

func TestReducePreservesEachFailingModeRouteActorAndStage(t *testing.T) {
	c := casefile.Case{Version: casefile.Version, Target: "pages", Expected: casefile.Expected{Status: 200}, Files: map[string]string{"pages/main.pk": "main", "partials/actor.pk": "actor", "partials/unused.pk": "unused"}}
	good := runner.Observation{Status: 200, ContentType: "text/html", MetadataTitle: "Piko Page"}
	fail := func(c *casefile.Case) Result {
		result := Result{Target: c.Target, Kind: "pass"}
		for _, mode := range []string{"interp", "compiled"} {
			one := c
			one.Probes = []casefile.Probe{{Path: "/left", Actor: "A", Stage: "first", Status: 200, Text: "left"}}
			if mode == "compiled" {
				one.Probes[0].Path = "/right"
			}
			if c.Files["partials/actor.pk"] == "" {
				one.Probes[0].Actor = "B"
				one.Probes[0].Stage = "later"
			}
			observation := good
			observation.Probes = []runner.ProbeObservation{{Path: one.Probes[0].Path, Actor: one.Probes[0].Actor, Status: 200, Text: "wrong"}}
			for _, issue := range runner.IndependentFailures(one, &observation) {
				result = failure("expectation", mode, issue, result)
			}
		}
		return result
	}
	full := fail(&c)
	require.Len(t, full.Failures, 2)
	assert.Equal(t, "/left/actor=A/stage=first/probe/text", full.Failures[0].Subject)
	assert.Equal(t, "/right/actor=A/stage=first/probe/text", full.Failures[1].Subject)
	assert.Equal(t, "interp", full.Mode)
	single := full
	single.Failures = single.Failures[:1]
	assert.False(t, full.matches(single), "secondary failure can vanish during reduction")
	assert.False(t, single.matches(full), "secondary failure can vanish during reduction")
	legacy := full
	legacy.Failures = nil
	assert.True(t, legacy.matches(full), "legacy primary failure no longer replays")
	wire, err := json.Marshal(full)
	require.NoError(t, err)
	var saved Result
	require.NoError(t, json.Unmarshal(wire, &saved))
	assert.True(t, full.matches(saved), "lost joint fingerprint on receipt round trip")
	h := Harness{Budget: testBudget{}, trial: func(_ context.Context, c *casefile.Case, _, _ string) Result { return fail(c) }}
	dir := t.TempDir()
	out := filepath.Join(dir, "reduced")
	_, err = h.Reduce(t.Context(), dir, out, &c, Options{Mode: "both", Browser: "off"})
	require.NoError(t, err)
	reduced, err := casefile.Read(out)
	require.NoError(t, err)
	assert.NotEmpty(t, reduced.Files["partials/actor.pk"])
	assert.Empty(t, reduced.Files["partials/unused.pk"])
	assert.True(t, full.matches(fail(&reduced)))
}

func TestReduceKeepsOnlyTheRequiredActorCookie(t *testing.T) {
	c := casefile.Case{Version: casefile.Version, Target: "pages", Expected: casefile.Expected{Status: 200}, Files: map[string]string{"pages/main.pk": "main"}, Probes: []casefile.Probe{
		{Actor: "editor", Stage: "setup", Path: "/setup", Cookies: map[string]string{"edition": "draft", "display": "wide"}, Status: 200, Text: "ok"},
		{Actor: "reader", Stage: "setup", Path: "/setup", Cookies: map[string]string{"edition": "public"}, Status: 200, Text: "ok"},
		{Actor: "editor", Stage: "decorate", Path: "/deco", Cookies: map[string]string{"color": "blue"}, Status: 200, Text: "ok"},
		{Actor: "editor", Stage: "revisit", Path: "/edition", Status: 200, Text: "ok"},
	}}
	original := c.Hash()
	run := func(_ context.Context, c *casefile.Case, _, _ string) Result {
		jars := map[string]map[string]string{}
		observation := runner.Observation{Mode: "interp", Status: 200, MetadataTitle: "Piko Page", ContentType: "text/html"}
		for _, probe := range c.Probes {
			if jars[probe.Actor] == nil {
				jars[probe.Actor] = map[string]string{}
			}
			maps.Copy(jars[probe.Actor], probe.Cookies)
			text := "ok"
			if probe.Path == "/edition" && jars[probe.Actor]["edition"] == "draft" {
				text = "stale draft"
			}
			observation.Probes = append(observation.Probes, runner.ProbeObservation{Actor: probe.Actor, Path: probe.Path, Status: 200, Text: text})
		}
		result := Result{Target: c.Target, Kind: "pass"}
		for _, issue := range runner.IndependentFailures(c, &observation) {
			result = failure("expectation", "interp", issue, result)
		}
		return result
	}
	h := Harness{Budget: testBudget{}, trial: run}
	before := run(t.Context(), &c, "", "")
	require.Equal(t, "/edition/actor=editor/stage=revisit/probe/text", before.Subject)
	dir := t.TempDir()
	out := filepath.Join(dir, "small")
	_, err := h.Reduce(t.Context(), dir, out, &c, Options{Mode: "interp", Browser: "off"})
	require.NoError(t, err)
	reduced, err := casefile.Read(out)
	require.NoError(t, err)
	require.Len(t, reduced.Probes, 2)
	assert.Equal(t, "setup", reduced.Probes[0].Stage)
	assert.Equal(t, "revisit", reduced.Probes[1].Stage)
	assert.Equal(t, map[string]string{"edition": "draft"}, reduced.Probes[0].Cookies)
	assert.True(t, before.matches(run(t.Context(), &reduced, "", "")))
	assert.Equal(t, original, c.Hash())
}

func TestSavedOptions(t *testing.T) {
	defaults, err := SavedOptions(t.TempDir())
	require.NoError(t, err)
	assert.Equal(t, Options{Mode: "both", Browser: "sample"}, defaults)

	valid := t.TempDir()
	want := cleanReport().Options
	require.NoError(t, saveJSON(filepath.Join(valid, "options.json"), want))
	saved, err := SavedOptions(valid)
	require.NoError(t, err)
	assert.Equal(t, want, saved)

	invalid := t.TempDir()
	broken := want
	broken.Mode = "sideways"
	require.NoError(t, saveJSON(filepath.Join(invalid, "options.json"), broken))
	_, err = SavedOptions(invalid)
	require.ErrorContains(t, err, "invalid options")

	malformed := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(malformed, "options.json"), []byte("{"), 0o600))
	_, err = SavedOptions(malformed)
	require.ErrorContains(t, err, "decoding")

	oversized := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(oversized, "options.json"), make([]byte, maxOptionsBytes+1), 0o600))
	_, err = SavedOptions(oversized)
	require.ErrorContains(t, err, "exceeds")
}

func TestPublishReductionPublishesOnceAndRefusesAnExistingBundle(t *testing.T) {
	out := filepath.Join(t.TempDir(), "nested", "minimal")
	c := generate.New("pages", 7, 0)
	options := Options{Mode: "compiled", Browser: "off"}
	final := Result{Kind: "expectation"}

	require.NoError(t, publishReduction(out, &c, &options, &final))
	assert.FileExists(t, filepath.Join(out, "options.json"))
	assert.FileExists(t, filepath.Join(out, "original.json"))

	assert.ErrorIs(t, publishReduction(out, &c, &options, &final), errReductionExists)
}
