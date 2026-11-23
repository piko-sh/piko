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
	"slices"
	"strconv"
	"strings"

	"piko.sh/fuzzkit/tape"
	"piko.sh/piko/tests/fuzz/internal/casefile"
)

const (
	// graphDir is the saved shared-partial source directory.
	graphDir = "partials/graph/"

	// graphPage names the partial graph's entry page file.
	graphPage = "pages/partial-graph.pk"

	// graphPackage names the application-local graph helper package path.
	graphPackage = "pkg/graphmodel"

	// graphHelper names the partial graph's helper source file.
	graphHelper = graphPackage + "/label.go"

	// graphFormatter names the graph's formatting helper source file.
	graphFormatter = graphPackage + "/format.go"

	// graphSharedPackage names the transitive shared helper package path.
	graphSharedPackage = "pkg/graphtext"

	// graphSharedHelper names the transitive shared helper source file.
	graphSharedHelper = graphSharedPackage + "/text.go"

	// graphProps holds the shared-partial props declaration.
	graphProps = "type Props struct { Site string; Label string }\n"

	// graphAlpha names the first partial branch.
	graphAlpha = "alpha"

	// graphBeta names the diamond's second partial branch.
	graphBeta = "beta"

	// graphRelay names the shared relay partial.
	graphRelay = "relay"

	// graphScheduledIndexes is the number of leading cases that enumerate graph motifs.
	graphScheduledIndexes = 32

	// graphScheduledPeriod is the case-position period of each scheduled motif.
	graphScheduledPeriod = 4

	// graphMotifMask selects the motif bits from the scheduled value.
	graphMotifMask = 7

	// graphDiamondBit is the motif bit that adds the diamond branch.
	graphDiamondBit = 1

	// graphRelayBit is the motif bit that routes branches through the relay.
	graphRelayBit = 2

	// graphReverseBit is the motif bit that reverses the call order.
	graphReverseBit = 4

	// graphLeafTemplate holds the leaf partial's template markup.
	graphLeafTemplate = `<output :data-fuzz-output="props.Site" :data-value="props.Label" p-text="state.Text"></output>`

	// graphRender holds the shared-partial render function.
	graphRender = "func Render(r *piko.RequestData, props Props)(Response,piko.Metadata,error){return Response{"
)

// partialGraph selects an acyclic graph of generated partials.
type partialGraph struct {
	// word is the generated leaf text.
	word string

	// diamond adds a second branch sharing the leaf.
	diamond bool

	// relay routes each branch through a shared relay partial.
	relay bool

	// reverse reverses the page's call order.
	reverse bool

	// editLeaf selects the leaf edit in the rebuild history.
	editLeaf bool

	// editHelper selects the helper edit in the rebuild history.
	editHelper bool
}

// partialCall identifies a branch invocation in a generated page.
type partialCall struct {
	// site identifies the call site in the generated page.
	site string

	// branch names the invoked partial branch.
	branch string
}

// scheduled schedules complete graph variants before tape-driven cases.
//
// Takes seed (uint64) which selects the scheduled graph motif.
// Takes index (uint64) which identifies the case's stream position.
//
// Returns partialGraph which holds the selected acyclic shared-partial graph.
func (g partialGraph) scheduled(seed, index uint64) partialGraph {
	if index < graphScheduledIndexes {
		motif := (seed ^ (index / graphScheduledPeriod)) & graphMotifMask
		g.diamond = motif&graphDiamondBit != 0
		g.relay = motif&graphRelayBit != 0
		g.reverse = motif&graphReverseBit != 0
	}
	return g
}

// calls lists partial invocations in document order.
//
// Returns []partialCall which lists partial call values in their recorded order.
func (g partialGraph) calls() []partialCall {
	calls := []partialCall{{site: "alpha-one", branch: graphAlpha}, {site: "alpha-two", branch: graphAlpha}}
	if g.diamond {
		calls = []partialCall{{site: "alpha-one", branch: graphAlpha}, {site: "beta-one", branch: graphBeta}, {site: "alpha-two", branch: graphAlpha}, {site: "beta-two", branch: graphBeta}}
	}
	if g.reverse {
		slices.Reverse(calls)
	}
	return calls
}

// child selects a branch's leaf or shared relay.
//
// Takes branch (string) which identifies the invoked partial branch.
//
// Returns string which identifies the leaf or relay invoked by the branch.
func (g partialGraph) child(branch string) string {
	if g.relay && branch != graphRelay {
		return graphRelay
	}
	return "leaf"
}

// evaluate models partial calls independently of source emission.
//
// Takes request (string) which is the generated request path.
//
// Returns []casefile.ProgramOutput which lists program output values in their recorded
// order.
func (g partialGraph) evaluate(request string) []casefile.ProgramOutput {
	if g.editHelper {
		request += "-changed"
	}
	label := request + "-" + g.word
	calls := g.calls()
	out := make([]casefile.ProgramOutput, 0, 1+len(calls))
	out = append(out, valuedOutput("caption", label, label))
	for _, call := range calls {
		var path strings.Builder
		path.WriteString(request + "-" + g.word)
		for node := call.branch; node != "leaf"; node = g.child(node) {
			path.WriteString("/" + node)
		}
		text := path.String() + "@" + call.site
		if g.editLeaf {
			text = "edited:" + text
		}
		out = append(out, valuedOutput(call.site, text, path.String()))
	}
	return out
}

// sourcePath finds the generated source path for a named partial.
//
// Takes name (string) which identifies the generated partial.
//
// Returns string which is the partial's generated source path.
func (g partialGraph) sourcePath(name string) string {
	if g.diamond && (name == graphAlpha || name == graphBeta) {
		return graphDir + name + "/branch.pk"
	}
	return graphDir + name + ".pk"
}

// partialImport formats an import of a generated partial.
//
// Takes name (string) which identifies the generated partial.
//
// Returns string which is the partial import declaration.
func (g partialGraph) partialImport(name string) string {
	return fmt.Sprintf("%s %q", name, casefile.Module+"/"+g.sourcePath(name))
}

// drawPartialGraph draws an acyclic shared-partial graph.
//
// Takes t (tape.Source) which supplies bounded deterministic choices.
//
// Returns partialGraph which holds the selected acyclic shared-partial graph.
func drawPartialGraph(t tape.Source) partialGraph {
	diamond, relay, reverse := t.Bool(1, 2), t.Bool(1, 2), t.Bool(1, 2)
	words := []string{"oak", "elm", "cedar"}
	return partialGraph{word: words[t.Pool(len(words))], diamond: diamond, relay: relay, reverse: reverse, editLeaf: false, editHelper: false}
}

// graphSFC assembles a generated single-file component.
//
// Takes template (string) which holds the template markup.
// Takes imports (string) which holds the source import declarations.
// Takes script (string) which holds the component's script body.
//
// Returns string which is the assembled single-file component source.
func graphSFC(template, imports, script string) string {
	return fmt.Sprintf(`<template>%s</template>
<script type="application/x-go">
package main
import (
    "piko.sh/piko"
    %s
)
%s
</script>
`, template, imports, script)
}

// emitPartialGraph adds shared-partial sources and independent output expectations.
//
// Takes c (*casefile.Case) which receives generated sources and expected results.
// Takes g (partialGraph) which holds the drawn shared-partial graph.
func emitPartialGraph(c *casefile.Case, g partialGraph) {
	var body, imports strings.Builder
	body.WriteString(`<main id="fuzz-program"><piko:partial is="frame"><div p-slot="payload">`)
	for _, call := range g.calls() {
		fmt.Fprintf(&body, `<piko:partial is=%q server.site=%q :server.label="state.Label"></piko:partial>`, call.branch, call.site)
	}
	body.WriteString(`</div><output p-slot="caption" data-fuzz-output="caption" :data-value="state.Label" p-text="state.Label"></output></piko:partial></main>`)
	c.Files[graphDir+"frame.pk"] = `<template><header><piko:slot name="caption"></piko:slot></header><section><piko:slot name="payload"></piko:slot></section></template>`
	emitGraphHelpers(c, g)
	imports.WriteString(g.partialImport("frame") + "\n")
	fmt.Fprintf(&imports, "graphmodel %q\n", casefile.Module+"/"+graphPackage)
	nodes := []string{graphAlpha}
	if g.diamond {
		nodes = append(nodes, graphBeta)
	}
	for _, node := range nodes {
		imports.WriteString(g.partialImport(node) + "\n")
	}
	c.Files[graphPage] = graphSFC(body.String(), imports.String(), `type Response struct{ Label string }
func Render(r *piko.RequestData, props piko.NoProps)(Response,piko.Metadata,error){
    return Response{Label:graphmodel.Label(r.QueryParam("run"))},piko.Metadata{},nil
}`)
	if g.relay {
		nodes = append(nodes, graphRelay)
	}
	for _, node := range nodes {
		child := g.child(node)
		template := fmt.Sprintf(`<section><piko:partial is=%q :server.site="props.Site" :server.label="state.Label"></piko:partial></section>`, child)
		script := graphProps + "type Response struct{ Label string }\n" + graphRender + "Label:props.Label+" + strconv.Quote("/"+node) + "},piko.Metadata{},nil}"
		c.Files[g.sourcePath(node)] = graphSFC(template, g.partialImport(child), script)
	}
	expression := `props.Label+marks.Site(props.Site)`
	if g.editLeaf {
		expression = `"edited:"+` + expression
	}
	leafScript := graphProps + "type Response struct{Text string}\n" + graphRender + "Text:" + expression + "},piko.Metadata{},nil}"
	c.Files[graphDir+"leaf.pk"] = graphSFC(graphLeafTemplate, fmt.Sprintf("marks %q", casefile.Module+"/"+graphSharedPackage), leafScript)
	addGraphProbes(c, g)
	addGraphFeatures(c, g)
}

// emitGraphHelpers adds the graph's local and transitive Go helper packages.
//
// Takes c (*casefile.Case) which receives the helper sources.
// Takes g (partialGraph) which holds the drawn shared-partial graph.
func emitGraphHelpers(c *casefile.Case, g partialGraph) {
	alias := "phrase"
	if g.reverse {
		alias = "words"
	}
	c.Files[graphHelper] = fmt.Sprintf(`package graphmodel

func Label(request string) string {
	return formatLabel(request, %q)
}
`, g.word)
	c.Files[graphFormatter] = fmt.Sprintf(`package graphmodel

import %s %q

func formatLabel(request, word string) string {
	return %s.Prefix(request) + word
}
`, alias, casefile.Module+"/"+graphSharedPackage, alias)
	prefix := `request + "-"`
	if g.editHelper {
		prefix = `request + "-changed" + "-"`
	}
	c.Files[graphSharedHelper] = fmt.Sprintf(`package graphtext

func Prefix(request string) string {
	return %s
}

func Site(site string) string {
	return "@" + site
}
`, prefix)
}

// addGraphProbes adds the graph's requests and their independently modelled outputs.
//
// Takes c (*casefile.Case) which receives the probes and rebuild sources.
// Takes g (partialGraph) which holds the drawn shared-partial graph.
func addGraphProbes(c *casefile.Case, g partialGraph) {
	routes := []string{"/partial-graph"}
	if c.Target == "graph-rebuild" {
		c.Files["pages/twin.pk"] = c.Files[graphPage]
		c.Files["pages/independent.pk"] = `<template><main id="fuzz-program"><output data-fuzz-output="independent">constant</output></main></template>`
		c.Probes = append(c.Probes, programProbe("/independent", programOutput("independent", "constant")))
		routes = append(routes, "/twin")
		c.Features = append(c.Features, fmt.Sprintf("graph-snapshot-leaf-%t-helper-%t", g.editLeaf, g.editHelper))
	}
	for _, route := range routes {
		for _, request := range []string{"first", "second"} {
			c.Probes = append(c.Probes, programProbe(route+"?run="+request, g.evaluate(request)...))
		}
	}
}

// addGraphFeatures records the graph's motif, path and order features.
//
// Takes c (*casefile.Case) which receives the features.
// Takes g (partialGraph) which holds the drawn shared-partial graph.
func addGraphFeatures(c *casefile.Case, g partialGraph) {
	path, order := "direct", "forward"
	if g.relay {
		path = graphRelay
	}
	if g.reverse {
		order = "reverse"
	}
	motif := "chain"
	if g.diamond {
		motif = "diamond"
		c.Features = append(c.Features, "aliased-duplicate-partial-names")
	}
	c.Features = append(c.Features, "partial-graph-shared-props", "named-slot-projection", "local-go-import", "multi-file-go-package",
		"transitive-go-import", "shared-go-package", "partial-graph-"+motif+"-"+path+"-"+order)
	if g.diamond {
		c.Features = append(c.Features, "partial-graph-diamond")
	} else {
		c.Features = append(c.Features, "partial-graph-chain")
	}
}
