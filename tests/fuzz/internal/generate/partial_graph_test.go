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
	"maps"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/tests/fuzz/internal/runner"
)

func TestPartialGraphMotifs(t *testing.T) {
	for bits := range 8 {
		graph := partialGraph{diamond: bits&1 != 0, relay: bits&2 != 0, reverse: bits&4 != 0, word: "elm"}
		c := casefile.Case{Files: map[string]string{}, Request: "/main", Expected: casefile.Expected{Status: http.StatusOK, Title: "test", Tag: "fuzz-counter"}}
		emitPartialGraph(&c, graph)
		requireParseableCase(t, c)
		wantFiles := 7
		sites := []string{"alpha-one", "alpha-two"}
		if graph.diamond {
			wantFiles++
			sites = []string{"alpha-one", "beta-one", "alpha-two", "beta-two"}
		}
		if graph.relay {
			wantFiles++
		}
		if graph.reverse {
			slices.Reverse(sites)
		}
		require.Len(t, c.Files, wantFiles, "graph %d", bits)
		require.Len(t, c.Probes, 2, "graph %d", bits)
		previous := -1
		for _, site := range sites {
			offset := strings.Index(c.Files[graphPage], `server.site="`+site+`"`)
			require.Greater(t, offset, previous, "graph %d source call order %s", bits, site)
			previous = offset
		}
		child := "leaf"
		if graph.relay {
			child = "relay"
			require.Contains(t, c.Files[graphDir+"relay.pk"], graph.partialImport("leaf"), "no relay edge")
		}
		branches := []string{"alpha"}
		if graph.diamond {
			branches = append(branches, "beta")
		}
		for _, name := range branches {
			wrapper := c.Files[graph.sourcePath(name)]
			require.Contains(t, c.Files[graphPage], graph.partialImport(name), "graph %d missing edge through %s", bits, name)
			require.Contains(t, wrapper, graph.partialImport(child))
			require.Contains(t, wrapper, `:server.site="props.Site"`)
			require.Contains(t, wrapper, `:server.label="state.Label"`)
		}
		if graph.diamond {
			for _, name := range []string{"alpha", "beta"} {
				file := graphDir + name + "/branch.pk"
				require.Contains(t, c.Files[graphPage], name+` "example.org/pikofuzzcase/`+file+`"`, "diamond %d collapsed the %s alias", bits, name)
				require.Contains(t, c.Files[file], `Label:props.Label+"/`+name+`"`, "diamond %d collapsed the %s semantics", bits, name)
			}
		}
		for position, request := range []string{"first", "second"} {
			probe := c.Probes[position]
			require.Equal(t, "/partial-graph?run="+request, probe.Path)
			require.True(t, probe.Program)
			require.Len(t, probe.Output, len(sites)+1)
			caption := probe.Output[0]
			assert.Equal(t, "caption", caption.ID)
			assert.Equal(t, request+"-elm", caption.Text)
			assert.True(t, caption.HasValue)
			assert.Equal(t, caption.Text, caption.Value)
			for offset, site := range sites {
				branch, _, _ := strings.Cut(site, "-")
				label := request + "-elm/" + branch
				if graph.relay {
					label += "/relay"
				}
				want := casefile.ProgramOutput{ID: site, Text: label + "@" + site, Value: label, HasValue: true}
				assert.Equal(t, want, probe.Output[offset+1], "graph %d output %d", bits, offset)
			}
		}
	}
}

func TestPartialGraphScheduleReplayAndShrinking(t *testing.T) {
	for index := range uint64(16) {
		c := New("pages", 7, index)
		_, present := c.Files[graphPage]
		require.Equal(t, index%4 == 0, present, "incorrect deterministic schedule at %d", index)
		replayed := Replay(c.Target, c.Seed, c.Index, c.Data)
		require.Equal(t, c.Hash(), replayed.Hash())
		if present {
			shrunk := ShrinkProject(&c, func(candidate *casefile.Case) bool { return candidate.Files[graphPage] == "" })
			require.Empty(t, shrunk.Files[graphPage], "could not remove whole graph")
			for _, probe := range shrunk.Probes {
				require.False(t, strings.HasPrefix(probe.Path, "/partial-graph"), "orphaned oracle %s", probe.Path)
			}
		}
	}
	for _, target := range []string{"http-actions", "components", "binder"} {
		c := New(target, 7, 0)
		assert.NotContains(t, c.Files, graphPage, target)
	}
}

func TestGraphOracleRejectsCrossSiteAndStaleRequest(t *testing.T) {
	c := casefile.Case{Files: map[string]string{}, Expected: casefile.Expected{Status: http.StatusOK, Title: "Piko Page"}}
	emitPartialGraph(&c, partialGraph{diamond: true, relay: true, word: "elm"})
	observe := func() runner.Observation {
		observation := runner.Observation{Status: http.StatusOK, MetadataTitle: "Piko Page", Heading: "Piko Page", ContentType: "text/html"}
		for _, probe := range c.Probes {
			observation.Probes = append(observation.Probes, runner.ProbeObservation{Path: probe.Path, Status: probe.Status, Program: probe.Program, Output: slices.Clone(probe.Output)})
		}
		return observation
	}
	require.NoError(t, runner.Check(&c, new(observe())))
	corruptions := map[string]func(*runner.Observation){
		"swapped call sites": func(observation *runner.Observation) {
			outputs := observation.Probes[0].Output
			outputs[0], outputs[1] = outputs[1], outputs[0]
		},
		"lost relay": func(observation *runner.Observation) {
			output := &observation.Probes[0].Output[1]
			output.Value = strings.TrimSuffix(output.Value, "/relay")
		},
		"stale query": func(observation *runner.Observation) {
			observation.Probes[1].Output = slices.Clone(observation.Probes[0].Output)
		},
		"lost call": func(observation *runner.Observation) {
			observation.Probes[0].Output = observation.Probes[0].Output[1:]
		},
	}
	for name, corrupt := range corruptions {
		observation := observe()
		corrupt(&observation)
		err := runner.Check(&c, &observation)
		require.Error(t, err, name)
		assert.True(t, strings.HasPrefix(err.Error(), "probe/program: /partial-graph"), "%s: %v", name, err)
	}
}

func TestPageWindowCoversAllEightPartialGraphs(t *testing.T) {
	for _, seed := range []uint64{0, 7, 99} {
		sources := map[string]bool{}
		for index := range uint64(32) {
			c := New("pages", seed, index)
			source := c.Files[graphPage]
			if index%4 != 0 {
				require.Empty(t, source, "unexpected partial graph at %d", index)
				continue
			}
			markup, _, _ := strings.Cut(source, "</template>")
			sources[markup+c.Files[graphDir+"alpha.pk"]+c.Files[graphDir+"alpha/branch.pk"]] = true
		}
		assert.Len(t, sources, 8, "seed %d unique graph structures", seed)
	}
}

func TestGraphRebuildUsesWarmEditSources(t *testing.T) {
	variants := map[string]bool{}
	for index := range uint64(32) {
		state := index % 4
		base, snapshot := New("incremental", 7, index), New("graph-rebuild", 7, index)
		leaf, helper := base.Files[graphDir+"leaf.pk"], base.Files[graphSharedHelper]
		if state < 2 {
			leaf = strings.Replace(leaf, "Text:props.Label", `Text:"edited:"+props.Label`, 1)
		}
		if state == 0 || state == 2 {
			helper = strings.Replace(helper, "return request +", `return request + "-changed" +`, 1)
		}
		require.Equal(t, leaf, snapshot.Files[graphDir+"leaf.pk"], "index %d", index)
		require.Equal(t, helper, snapshot.Files[graphSharedHelper], "index %d", index)
		require.Equal(t, base.Files[graphHelper], snapshot.Files[graphHelper], "index %d", index)
		require.Equal(t, base.Files[graphFormatter], snapshot.Files[graphFormatter], "index %d", index)
		require.Equal(t, base.Files[graphPage], snapshot.Files["pages/twin.pk"], "index %d", index)
		markup, _, _ := strings.Cut(snapshot.Files[graphPage], "</template>")
		branch := snapshot.Files[graphDir+"alpha.pk"] + snapshot.Files[graphDir+"alpha/branch.pk"]
		variants[markup+branch+leaf+helper] = true
		requireParseableCase(t, snapshot)
		found := map[string]bool{}
		for _, probe := range snapshot.Probes {
			if !strings.HasPrefix(probe.Path, "/partial-graph?") && !strings.HasPrefix(probe.Path, "/twin?") {
				continue
			}
			found[probe.Path] = true
			for _, output := range probe.Output {
				assert.Equal(t, state == 0 || state == 2, strings.Contains(output.Value, "-changed-"), "index %d %+v", index, output)
				assert.Equal(t, state < 2 && output.ID != "caption", strings.HasPrefix(output.Text, "edited:"), "index %d %+v", index, output)
			}
		}
		require.Len(t, found, 4, "index %d", index)
		replayed := Replay(snapshot.Target, 7, index, snapshot.Data)
		require.Equal(t, snapshot.Hash(), replayed.Hash())
	}
	assert.Len(t, variants, 32)
}

func TestGraphGoHelperLivesInApplicationPackage(t *testing.T) {
	for _, target := range []string{"pages", "incremental", "graph-rebuild"} {
		c := New(target, 7, 0)
		for name := range c.Files {
			isPublicGo := strings.HasPrefix(name, "lib/") && strings.HasSuffix(name, ".go")
			require.False(t, isPublicGo, "%s: server Go source %s in Piko's public asset directory", target, name)
		}
		assert.Contains(t, c.Files["pkg/graphmodel/label.go"], "func Label(request string)", target)
		assert.Contains(t, c.Files[graphPage], `graphmodel "example.org/pikofuzzcase/pkg/graphmodel"`, target)
		assert.Contains(t, c.Files[graphPage], "graphmodel.Label(", target)
	}
}

func TestGraphSharesATransitiveApplicationPackage(t *testing.T) {
	aliases := map[string]bool{}
	for index := range uint64(32) {
		c := New("incremental", 7, index)
		alias := "phrase"
		if strings.Contains(c.Files[graphFormatter], "import words ") {
			alias = "words"
		}
		aliases[alias] = true
		require.Contains(t, c.Files[graphHelper], "return formatLabel(request,", "index %d", index)
		require.Contains(t, c.Files[graphFormatter], "import "+alias+` "example.org/pikofuzzcase/pkg/graphtext"`, "index %d", index)
		require.Contains(t, c.Files[graphFormatter], "return "+alias+".Prefix(request) + word", "index %d", index)
		require.Contains(t, c.Files[graphSharedHelper], `return request + "-"`, "index %d", index)
		require.Contains(t, c.Files[graphSharedHelper], `return "@" + site`, "index %d", index)
		require.Contains(t, c.Files[graphDir+"leaf.pk"], `marks "example.org/pikofuzzcase/pkg/graphtext"`, "index %d", index)
		require.Contains(t, c.Files[graphDir+"leaf.pk"], `marks.Site(props.Site)`, "index %d", index)
	}
	assert.Len(t, aliases, 2)
}

func TestReduceDiamondKeepingRequiredBranchAndRequest(t *testing.T) {
	c := New("pages", 7, 0)
	require.NotEmpty(t, c.Files[graphDir+"beta/branch.pk"], "missing original diamond")
	require.NotEmpty(t, c.Files[graphDir+"relay.pk"], "missing original relay")
	keeps := func(candidate *casefile.Case) bool {
		for _, probe := range candidate.Probes {
			if probe.Path == "/partial-graph?run=second" && slices.ContainsFunc(probe.Output, func(output casefile.ProgramOutput) bool { return output.ID == "alpha-one" }) {
				return true
			}
		}
		return false
	}
	small := ShrinkProject(&c, keeps)
	require.True(t, keeps(&small))
	assert.Empty(t, small.Files[graphDir+"beta/branch.pk"], "%v", slices.Sorted(maps.Keys(small.Files)))
	assert.Empty(t, small.Files[graphDir+"relay.pk"])
	assert.NotEmpty(t, small.Files[graphDir+"alpha.pk"])
	assert.NotEmpty(t, small.Files[graphDir+"leaf.pk"])
	for _, probe := range small.Probes {
		if !strings.HasPrefix(probe.Path, "/partial-graph") {
			continue
		}
		orphaned := slices.ContainsFunc(probe.Output, func(output casefile.ProgramOutput) bool { return strings.HasPrefix(output.ID, "beta-") })
		assert.False(t, orphaned, "orphaned removed call in %s", probe.Path)
	}
}
