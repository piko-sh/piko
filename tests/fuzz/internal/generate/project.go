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

// Package generate turns bounded fuzzkit choices into executable Piko sources.
package generate

import (
	"crypto/rand"
	"crypto/sha256"
	"embed"
	"fmt"
	"html"
	"io/fs"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"piko.sh/fuzzkit/tape"
	"piko.sh/safeconv"

	"piko.sh/piko/internal/gosource"
	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/tests/fuzz/internal/fixturego"
)

const (
	// Version identifies the source generator version.
	Version = "piko-project/6"

	// Profile identifies the structural generation profile.
	Profile = "small-project/6"

	// missingPageOutput holds the expected missing-route page source.
	missingPageOutput = `<section id="fuzz-program"><output data-fuzz-output="status">404</output></section>`

	// targetPages names the pages target.
	targetPages = "pages"

	// countChoices bounds the drawn initial counter.
	countChoices = 19

	// partialOdds is the denominator of the odds of a shared partial.
	partialOdds = 3

	// clientGlueOdds is the denominator of the odds of client-side page code.
	clientGlueOdds = 4

	// rowChoices bounds the drawn number of display rows.
	rowChoices = 8

	// followupChoices is the number of follow-up route families, including none.
	followupChoices = 7

	// reservedDraws is the number of choices reserved after the base project draws.
	reservedDraws = 32

	// reservedDrawRange is the range of each reserved choice.
	reservedDrawRange = 256

	// formHistories is the number of form-workflow history variants.
	formHistories = 4

	// nativeEvenIndex is the stream position of native fuzz inputs with an even first byte.
	nativeEvenIndex = 32

	// nativeChoiceLimit bounds the native fuzz choice bytes replayed.
	nativeChoiceLimit = 1024

	// graphPeriod is the case-position period of shared graph and typed pages.
	graphPeriod = 4

	// followupRouteParam selects the route-parameter follow-up.
	followupRouteParam = 1

	// followupRedirect selects the client-redirect follow-up.
	followupRedirect = 2

	// followupTypedError selects the typed not-found follow-up.
	followupTypedError = 3

	// followupUnmatched selects the unmatched-route follow-up.
	followupUnmatched = 4

	// followupSharedPartial selects the shared-partial follow-up.
	followupSharedPartial = 5

	// followupTranslation selects the page-translation follow-up.
	followupTranslation = 6

	// followupScript opens a follow-up page's Go script.
	followupScript = "<script type=\"application/x-go\">\npackage main\nimport \"piko.sh/piko\"\n"

	// followupSlugRender renders the route parameter.
	followupSlugRender = "type Response struct { Slug string }\n" +
		"func Render(r *piko.RequestData, props piko.NoProps)(Response,piko.Metadata,error){return Response{Slug:r.PathParam(\"slug\")},piko.Metadata{},nil}\n</script>"

	// followupRedirectRender redirects to the main page.
	followupRedirectRender = "func Render(r *piko.RequestData, props piko.NoProps)(piko.NoResponse,piko.Metadata,error){" +
		"return piko.NoResponse{},piko.Metadata{ClientRedirect:\"/main\",RedirectStatus:303},nil}\n</script>"

	// followupMissingRender returns a typed not-found error.
	followupMissingRender = "func Render(r *piko.RequestData, props piko.NoProps)(piko.NoResponse,piko.Metadata,error){" +
		"return piko.NoResponse{},piko.Metadata{}, &piko.NotFoundError{Resource:\"item\", ID:\"missing\"}}\n</script>"

	// followupBadgeRender renders the shared badge partial.
	followupBadgeRender = "type Props struct {Label string}\n" +
		"func Render(r *piko.RequestData,props Props)(piko.NoResponse,piko.Metadata,error){return piko.NoResponse{},piko.Metadata{},nil}\n</script>"

	// followupBadgePage holds the script of a page using the shared badge partial.
	followupBadgePage = "<script type=\"application/x-go\">\npackage main\nimport (\n\"piko.sh/piko\"\nbadge \"" + casefile.Module + "/partials/badge.pk\"\n)\n" +
		"func Render(r *piko.RequestData, props piko.NoProps)(piko.NoResponse,piko.Metadata,error){return piko.NoResponse{},piko.Metadata{},nil}\n</script>"

	// followupEmptyRender renders a page with no response data.
	followupEmptyRender = "func Render(r *piko.RequestData, props piko.NoProps)(piko.NoResponse,piko.Metadata,error){return piko.NoResponse{},piko.Metadata{},nil}\n</script>"

	// mainPageTemplate formats the main page from its content, imports, title expression,
	// visibility and row literals.
	mainPageTemplate = `<template>%s</template>
<script type="application/x-go">
package main
import (
 "piko.sh/piko"
 %s
)
type Row struct { ID int; Label string }
type Response struct { Title string; Show bool; Rows []Row }
func Render(r *piko.RequestData, props piko.NoProps) (Response,piko.Metadata,error) {
 title := %s
 return Response{Title:title,Show:%t,Rows:[]Row{%s}},piko.Metadata{Title:title},nil
}
</script>
`
)

var (
	// generatorFiles embeds every generator source and template so the generator profile
	// changes whenever any of them changes.
	//
	//go:embed *.go *.tmpl
	generatorFiles embed.FS

	// ProfileSHA256 identifies all embedded generator and model sources.
	//
	// When the embedded generator sources cannot be listed or read it holds an "unavailable:
	// " marker with a random suffix, so two binaries with the same failure never compare
	// equal. An unreadable Go fixture tree panics at package initialisation instead, before
	// this value is computed.
	ProfileSHA256 = profileDigest()

	// targetEmitters adds each target's own sources and expectations after the shared main
	// page, follow-up and extras.
	targetEmitters = map[string]func(*casefile.Case, Project) error{
		"auth-contracts":           addAuthContracts,
		"form-shim-contracts":      addFormShimContracts,
		"route-contracts":          infallible(addRouteContracts),
		"generated-artifacts":      infallible(addArtefacts),
		"form-workflow":            addFormWorkflow,
		"typed-catalog":            infallible(addTypedCatalogue),
		"catalog-rebuild":          infallible(addTypedCatalogue),
		"source-diagnostics":       infallible(addDiagnostics),
		"source-equivalence":       infallible(addGuardsAndEquivalence),
		"wasm":                     infallible(addEquivalence),
		"markup":                   infallible(addMarkup),
		"firefox-markup":           infallible(addMarkup),
		"shipped-components":       infallible(addShipped),
		"checkout-workflow":        addCheckout,
		"partial-components":       addPartialComponents,
		"browser-events":           infallible(addBrowserEvents),
		"browser-lifecycle":        infallible(addBrowserLifecycle),
		"collection-routes":        infallible(addCollectionRoutes),
		"content-editions":         addCollectionEditions,
		"http-actions":             infallible(addHTTPAction),
		"incremental":              infallible(addIncrementalHistory),
		"go-topology":              infallible(addGoTopology),
		"embedded-deploy":          infallible(addEmbeddedDeployment),
		"browser-element-reuse":    infallible(addBrowserElementReuse),
		"browser-lifecycle-inline": infallible(addBrowserLifecycleInline),
	}

	// labels is the bounded Unicode label vocabulary.
	labels = []string{"Hello", "A & B", "café", "C", "quote \"here\"", "<text>", "東京", "R&D", "two  spaces", "'single'", "π", "A", "duplicate"}
)

// Project holds the drawn structure of a generated application.
type Project struct {
	// graph holds the generated partial graph.
	graph *partialGraph

	// program holds the generated expression and control-flow model.
	program *program

	// Wrapper is the optional enclosing element.
	Wrapper string

	// Title is the page title.
	Title string

	// size selects structural generation bounds.
	size string

	// Rows holds the generated display labels.
	Rows []string

	// Count is the initial component counter.
	Count int

	// formHistory selects a form-workflow history.
	formHistory int

	// Followup selects a follow-up route family.
	Followup int

	// skipTypedFields selects typed-value fields omitted during shrinking.
	skipTypedFields uint32

	// skipTypedModes selects typed-value variants omitted during shrinking.
	skipTypedModes uint32

	// Query enables a query-dependent response.
	Query bool

	// omitScoped removes scoped-binding pages during shrinking.
	omitScoped bool

	// Slot enables projected component content.
	Slot bool

	// ClientGlue enables client-side page code.
	ClientGlue bool

	// Style enables generated styles.
	Style bool

	// omitMatrix removes the fixed arithmetic matrix during shrinking.
	omitMatrix bool

	// omitTyped removes typed-value pages during shrinking.
	omitTyped bool

	// TextBinding selects a text binding in the component.
	TextBinding bool

	// Partial enables a shared partial.
	Partial bool

	// Loop enables looped content.
	Loop bool

	// omitCache removes cache checks during shrinking.
	omitCache bool

	// omitUpload removes upload checks during shrinking.
	omitUpload bool

	// omitActors removes actor-isolation checks during shrinking.
	omitActors bool

	// omitCancellation removes stream-cancellation checks during shrinking.
	omitCancellation bool

	// skipRefusals selects action refusals omitted during shrinking.
	skipRefusals uint8

	// Show selects conditional content visibility.
	Show bool
}

// choice draws a label from the bounded source vocabulary.
//
// Takes t (tape.Source) which supplies bounded deterministic choices.
//
// Returns string which is the drawn label.
func choice(t tape.Source) string { return labels[t.Pool(len(labels))] }

// draw draws the application's base structural choices.
//
// Takes t (tape.Source) which supplies bounded deterministic choices.
//
// Returns Project which holds the drawn application model.
func draw(t tape.Source) Project {
	var p Project
	p.Title = choice(t)
	p.Count = safeconv.Uint64ToInt(t.Uint(countChoices))
	p.Show, p.Loop, p.Query = t.Bool(1, 2), t.Bool(1, 2), t.Bool(1, 2)
	p.Partial, p.Style, p.ClientGlue = t.Bool(1, partialOdds), t.Bool(1, 2), t.Bool(1, clientGlueOdds)
	p.Slot, p.TextBinding = t.Bool(1, 2), t.Bool(1, 2)
	wrappers := []string{"", "section", "article", "div"}
	p.Wrapper = wrappers[t.Uint(uint64(len(wrappers)))]
	for n := t.Uint(rowChoices); n > 0; n-- {
		p.Rows = append(p.Rows, choice(t))
	}
	p.Followup = t.Pool(followupChoices)
	return p
}

// New generates a deterministic case at the requested stream position.
//
// The generator panics if case emission fails. Use Build to receive an error.
//
// Takes target (string) which selects the generated target.
// Takes seed (uint64) which initialises the deterministic stream.
// Takes index (uint64) which selects the case position.
//
// Returns casefile.Case which holds the generated sources, recorded choices and
// independent expectations.
func New(target string, seed, index uint64) casefile.Case {
	return NewSized(target, seed, index, "")
}

// NewSized generates a case under the selected structural bounds.
//
// Takes target (string) which selects the subsystem under test.
// Takes seed (uint64) which initialises the deterministic choice stream.
// Takes index (uint64) which identifies the case's stream position.
// Takes size (string) which selects structural generation bounds.
//
// Returns casefile.Case which holds the generated sources, recorded choices and
// independent expectations.
//
// Panics if the generator cannot emit the case; use Build to receive the error.
func NewSized(target string, seed, index uint64, size string) casefile.Case {
	c, err := Build(target, seed, index, size)
	if err != nil {
		panic(err)
	}
	return c
}

// Build generates a case under the selected structural bounds.
//
// Takes target (string) which selects the subsystem under test.
// Takes seed (uint64) which initialises the deterministic choice stream.
// Takes index (uint64) which identifies the case's stream position.
// Takes size (string) which selects structural generation bounds.
//
// Returns casefile.Case which holds the generated sources, recorded choices and
// independent expectations.
// Returns error which reports a generator defect that prevented emission, or nil on
// success.
func Build(target string, seed, index uint64, size string) (casefile.Case, error) {
	source := tape.New(tape.Mix(seed, index))
	project := drawProject(source, target, seed, index, size)
	c, err := emit(project, target, seed, index, source.Recording().Data)
	if err != nil {
		return casefile.Case{}, fmt.Errorf("generating %s seed %d index %d: %w", target, seed, index, err)
	}
	return c, nil
}

// drawProject draws the target's model under the saved structural bounds.
//
// Takes t (tape.Source) which supplies bounded deterministic choices.
// Takes target (string) which selects the subsystem under test.
// Takes seed (uint64) which initialises the deterministic choice stream.
// Takes index (uint64) which identifies the case's stream position.
// Takes size (string) which selects structural generation bounds.
//
// Returns Project which holds the drawn application model.
func drawProject(t tape.Source, target string, seed, index uint64, size string) Project {
	p := draw(t)
	p.size = size
	if size == "tiny" {
		p.Rows = p.Rows[:min(2, len(p.Rows))]
	}
	for range reservedDraws {
		t.Uint(reservedDrawRange)
	}
	if generatesProgram(target) {
		model := drawProgram(t, limitsFor(size))
		p.program = &model
	}
	if target == targetPages || target == "incremental" || target == "graph-rebuild" {
		model := drawPartialGraph(t).scheduled(seed, index)
		p.graph = &model
	}
	if target == "form-workflow" {
		p.formHistory = safeconv.Uint64ToInt(t.Uint(formHistories))
		if index < formHistories {
			p.formHistory = safeconv.Uint64ToInt(index)
		}
	}
	return p
}

// NativeProject maps one native fuzz argument to a replayable page case.
//
// The generator panics if case emission fails.
//
// Takes data ([]byte) which holds the native fuzz choice bytes.
//
// Returns casefile.Case which holds the generated sources, recorded choices and
// independent expectations.
func NativeProject(data []byte) casefile.Case {
	index := uint64(1)
	if len(data) > 0 && data[0]%2 == 0 {
		index = nativeEvenIndex
	}
	return Replay(targetPages, 0, index, data[:min(len(data), nativeChoiceLimit)])
}

// Replay emits a case from a saved choice recording.
//
// The generator panics if case emission fails. Use BuildReplay to receive an error.
//
// Takes target (string) which selects the subsystem under test.
// Takes seed (uint64) which initialises the deterministic choice stream.
// Takes index (uint64) which identifies the case's stream position.
// Takes data ([]byte) which holds the replayable choice bytes.
//
// Returns casefile.Case which holds the generated sources, recorded choices and
// independent expectations.
func Replay(target string, seed, index uint64, data []byte) casefile.Case {
	return ReplaySized(target, seed, index, data, "")
}

// ReplaySized emits a recorded case under its saved structural bounds.
//
// Takes target (string) which selects the subsystem under test.
// Takes seed (uint64) which initialises the deterministic choice stream.
// Takes index (uint64) which identifies the case's stream position.
// Takes data ([]byte) which holds the replayable choice bytes.
// Takes size (string) which selects structural generation bounds.
//
// Returns casefile.Case which holds the generated sources, recorded choices and
// independent expectations.
//
// Panics if the generator cannot emit the case; use BuildReplay to receive the error.
func ReplaySized(target string, seed, index uint64, data []byte, size string) casefile.Case {
	c, err := BuildReplay(target, seed, index, data, size)
	if err != nil {
		panic(err)
	}
	return c
}

// BuildReplay emits a recorded case under its saved structural bounds.
//
// Takes target (string) which selects the subsystem under test.
// Takes seed (uint64) which initialises the deterministic choice stream.
// Takes index (uint64) which identifies the case's stream position.
// Takes data ([]byte) which holds the replayable choice bytes.
// Takes size (string) which selects structural generation bounds.
//
// Returns casefile.Case which holds the generated sources, recorded choices and
// independent expectations.
// Returns error which reports a generator defect that prevented emission, or nil on
// success.
func BuildReplay(target string, seed, index uint64, data []byte, size string) (casefile.Case, error) {
	source := tape.Replay(&tape.Recording{Data: data})
	c, err := emit(drawProject(source, target, seed, index, size), target, seed, index, data)
	if err != nil {
		return casefile.Case{}, fmt.Errorf("replaying %s seed %d index %d: %w", target, seed, index, err)
	}
	return c, nil
}

// ShrinkProject removes model features while preserving the requested failure.
//
// A candidate whose emission fails or panics is rejected. A panic in preserves, which
// runs the harness, is not recovered here.
//
// Takes c (*casefile.Case) which holds the saved sources, choices and expected results;
// it is not modified.
// Takes preserves (func(*casefile.Case) bool) which reports whether a candidate retains
// the required failure.
//
// Returns casefile.Case which holds the generated sources, recorded choices and
// independent expectations.
func ShrinkProject(c *casefile.Case, preserves func(*casefile.Case) bool) casefile.Case {
	p := drawProject(tape.Replay(&tape.Recording{Data: c.Data}), c.Target, c.Seed, c.Index, c.Size)
	try := func(next Project) {
		candidate, err := recoverEmit(func() (casefile.Case, error) { return emit(next, c.Target, c.Seed, c.Index, c.Data) })
		if err != nil {
			return
		}
		if candidate.Hash() != c.Hash() && preserves(&candidate) {
			p = next
			c = &candidate
		}
	}
	for _, simplify := range projectSimplifications() {
		next := p
		simplify(&next)
		try(next)
	}
	if _, ok := c.Files["pages/typed.pk"]; ok {
		shrinkTyped(&p, try)
	}
	if c.Expected.Action != nil {
		shrinkAction(&p, try)
	}
	if p.program != nil {
		shrinkProgram(&p, try)
	}
	if p.graph != nil {
		shrinkGraph(&p, try)
	}
	for index := range slices.Backward(p.Rows) {
		next := p
		next.Rows = slices.Delete(slices.Clone(p.Rows), index, index+1)
		try(next)
	}
	return *c
}

// recoverEmit runs one emission, converting a generator panic into an error.
//
// Takes emission (func() (casefile.Case, error)) which emits the candidate.
//
// Returns casefile.Case which holds the emitted candidate.
// Returns error which reports an emission failure or a recovered generator panic, or nil
// on success.
func recoverEmit(emission func() (casefile.Case, error)) (c casefile.Case, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			c, err = casefile.Case{}, fmt.Errorf("generator panic: %v", recovered)
		}
	}()
	return emission()
}

// projectSimplifications lists the whole-project simplifications tried while shrinking.
//
// Returns []func(*Project) which lists each simplification in the order tried.
func projectSimplifications() []func(*Project) {
	return []func(*Project){
		func(p *Project) { p.graph = nil }, func(p *Project) { p.omitMatrix = true }, func(p *Project) { p.omitTyped = true },
		func(p *Project) { p.omitScoped = true }, func(p *Project) { p.omitCancellation = true }, func(p *Project) { p.program = nil },
		func(p *Project) { p.Followup = 0 }, func(p *Project) { p.Partial = false },
		func(p *Project) { p.Wrapper = "" }, func(p *Project) { p.Slot = false }, func(p *Project) { p.ClientGlue = false },
		func(p *Project) { p.Style = false }, func(p *Project) { p.Loop = false }, func(p *Project) { p.Show = false },
		func(p *Project) { p.Query = false }, func(p *Project) { p.TextBinding = false }, func(p *Project) { p.Count = 0 },
		func(p *Project) { p.Title = "a" },
	}
}

// shrinkTyped tries omitting each typed-value mode and field.
//
// Takes p (*Project) which holds the application model being simplified.
// Takes try (func(Project)) which tests each simplified model.
func shrinkTyped(p *Project, try func(Project)) {
	for index := range typedModes {
		next := *p
		next.skipTypedModes |= 1 << index
		try(next)
	}
	for index := range typedFields {
		next := *p
		next.skipTypedFields |= 1 << index
		try(next)
	}
}

// shrinkAction tries omitting each optional action behaviour and refusal.
//
// Takes p (*Project) which holds the application model being simplified.
// Takes try (func(Project)) which tests each simplified model.
func shrinkAction(p *Project, try func(Project)) {
	for _, simplify := range []func(*Project){
		func(next *Project) { next.omitCache = true },
		func(next *Project) { next.omitUpload = true },
		func(next *Project) { next.omitActors = true },
	} {
		next := *p
		simplify(&next)
		try(next)
	}
	for index := range len(keptRefusals(0)) {
		next := *p
		next.skipRefusals |= 1 << index
		try(next)
	}
}

// shrinkGraph tries removing the partial graph's relay and diamond.
//
// Takes p (*Project) which holds the application model being simplified.
// Takes try (func(Project)) which tests each simplified model.
func shrinkGraph(p *Project, try func(Project)) {
	for _, simplify := range []func(*partialGraph){
		func(g *partialGraph) { g.relay = false },
		func(g *partialGraph) { g.diamond = false },
	} {
		simpler := *p.graph
		simplify(&simpler)
		next := *p
		next.graph = &simpler
		try(next)
	}
}

// emit emits the project's sources and independently modelled expectations.
//
// Takes p (Project) which holds the drawn application model.
// Takes target (string) which selects the subsystem under test.
// Takes seed (uint64) which initialises the deterministic choice stream.
// Takes index (uint64) which identifies the case's stream position.
// Takes data ([]byte) which holds the replayable choice bytes.
//
// Returns casefile.Case which holds the generated sources, recorded choices and
// independent expectations.
// Returns error which reports a generator defect that prevented emission, or nil on
// success.
func emit(p Project, target string, seed, index uint64, data []byte) (casefile.Case, error) {
	c := newCase(p, target, seed, index, data)
	if err := emitMainPage(&c, p, target); err != nil {
		return casefile.Case{}, err
	}
	addFollowup(&c, p)
	if err := emitSharedExtras(&c, p, target, index); err != nil {
		return casefile.Case{}, err
	}
	if emitter, ok := targetEmitters[target]; ok {
		if err := emitter(&c, p); err != nil {
			return casefile.Case{}, err
		}
	}
	return c, nil
}

// newCase builds the case envelope and the main page's base expectations.
//
// Takes p (Project) which holds the drawn application model.
// Takes target (string) which selects the subsystem under test.
// Takes seed (uint64) which initialises the deterministic choice stream.
// Takes index (uint64) which identifies the case's stream position.
// Takes data ([]byte) which holds the replayable choice bytes.
//
// Returns casefile.Case which holds the case envelope without sources.
func newCase(p Project, target string, seed, index uint64, data []byte) casefile.Case {
	var expected casefile.Expected
	expected.Title = p.Title
	expected.Rows = p.Rows
	expected.Count = p.Count
	expected.Show = p.Show
	expected.Tag = "fuzz-counter"
	expected.Status = http.StatusOK
	var c casefile.Case
	c.Size = p.size
	c.Version = casefile.Version
	c.Generator = Version
	c.Profile = Profile
	c.ProfileSHA256 = ProfileSHA256
	c.BrowserSampleVersion = 1
	c.Target = target
	c.Seed = seed
	c.Index = index
	c.Data = append([]byte(nil), data...)
	c.Request = "/main"
	c.Files = map[string]string{}
	c.Features = []string{"go-render", "counter", "interpolation"}
	c.Expected = expected
	return c
}

// emitMainPage adds the main page, its optional partial frame and its features.
//
// Takes c (*casefile.Case) which receives generated sources and expected results.
// Takes p (Project) which holds the drawn application model.
// Takes target (string) which selects the subsystem under test.
//
// Returns error which reports an unrepresentable component model, or nil on success.
func emitMainPage(c *casefile.Case, p Project, target string) error {
	titleExpr := strconv.Quote(p.Title)
	if p.Query {
		c.Request += "?label=" + url.QueryEscape(p.Title)
		titleExpr = `r.QueryParam("label")`
		c.Features = append(c.Features, "query")
	}
	var body strings.Builder
	body.WriteString(`<main id="case-root"><h1 id="case-title">{{ state.Title }}</h1><ul id="case-list">`)
	if p.Loop {
		body.WriteString(`<li p-for="row in state.Rows" p-key="row.ID">{{ row.Label }}</li>`)
		c.Features = append(c.Features, "keyed-loop")
	} else {
		c.Expected.Rows = nil
		for _, label := range p.Rows {
			fmt.Fprintf(&body, "<li>%s</li>", html.EscapeString(label))

			c.Expected.Rows = append(c.Expected.Rows, strings.Join(strings.Fields(label), " "))
		}
	}
	body.WriteString(`</ul><p id="case-visible" p-if="state.Show">visible</p>`)
	hosts, err := emitComponent(c, p)
	if err != nil {
		return err
	}
	body.WriteString(hosts)
	body.WriteString("</main>")
	inner := body.String()
	if p.Wrapper != "" {
		inner = "<" + p.Wrapper + ">" + inner + "</" + p.Wrapper + ">"
	}
	var imports string
	if p.Partial {
		imports = `frame "` + casefile.Module + `/partials/frame.pk"`
		inner = emitFrame(c, target, inner)
	}
	var rows strings.Builder
	for position, label := range p.Rows {
		fmt.Fprintf(&rows, "{ID:%d,Label:%q},", position+1, label)
	}
	c.Files["pages/main.pk"] = fmt.Sprintf(mainPageTemplate, inner, imports, titleExpr, p.Show, rows.String())
	if p.Style {
		c.Files[counterComponentPath] += "<style>button { color: navy; }</style>\n"
		c.Files["pages/main.pk"] += "<style>#case-root { padding: 1px; }</style>\n"
		c.Features = append(c.Features, "scoped-style")
	}
	if p.ClientGlue {
		c.Files["pages/main.pk"] += `<script lang="ts">export function caseLabel(): string { return "ready"; }</script>`
		c.Features = append(c.Features, "client-script")
	}
	return nil
}

// emitFrame adds the shared frame partial and wraps the page content in it.
//
// Takes c (*casefile.Case) which receives the frame source and features.
// Takes target (string) which selects the subsystem under test.
// Takes inner (string) which is the page content placed in the frame.
//
// Returns string which is the page content wrapped in the frame partial.
func emitFrame(c *casefile.Case, target, inner string) string {
	slot := "<piko:slot></piko:slot>"
	if target == targetFormatter {
		inner += `<span p-slot="caption">caption</span>`
		slot += `<piko:slot name="caption"></piko:slot>`
		c.Features = append(c.Features, "formatter-slot-fill")
	}
	c.Files["partials/frame.pk"] = `<template><section id="case-frame">` + slot + `</section></template>`
	c.Expected.Frame = true
	c.Features = append(c.Features, "partial")
	return `<piko:partial is="frame">` + inner + "</piko:partial>"
}

// emitSharedExtras adds the graph, program, matrix and typed pages shared by targets.
//
// Takes c (*casefile.Case) which receives generated sources and expected results.
// Takes p (Project) which holds the drawn application model.
// Takes target (string) which selects the subsystem under test.
// Takes index (uint64) which identifies the case's stream position.
//
// Returns error which reports a malformed program or matrix, or nil on success.
func emitSharedExtras(c *casefile.Case, p Project, target string, index uint64) error {
	if p.graph != nil {
		emitScheduledGraph(c, *p.graph, target, index)
	}
	if p.program != nil {
		if err := emitProgram(c, p.Title, *p.program); err != nil {
			return err
		}
	}
	if err := emitMatrixPages(c, p, target, index); err != nil {
		return err
	}
	if (target == targetPages && index%graphPeriod == 1 || target == "oracle-sensitivity") && !p.omitScoped {
		addScopedPage(c, p)
	}
	if target == targetFormatter && p.Followup == 0 {
		c.Files["pages/whitespace-probe.pk"] = fmt.Sprintf("<template><pre>left  %d\n  right</pre></template>", p.Count)
		c.Features = append(c.Features, "preformatted-text")
	}
	return nil
}

// emitMatrixPages adds the arithmetic matrix and typed pages on their scheduled cases.
//
// Takes c (*casefile.Case) which receives generated sources and expected results.
// Takes p (Project) which holds the drawn application model.
// Takes target (string) which selects the subsystem under test.
// Takes index (uint64) which identifies the case's stream position.
//
// Returns error which reports a malformed arithmetic matrix, or nil on success.
func emitMatrixPages(c *casefile.Case, p Project, target string, index uint64) error {
	if target != targetPages && target != "components" && target != "http-actions" {
		return nil
	}
	if index == 0 && p.size != "tiny" && !p.omitMatrix {
		if err := addArithmeticMatrix(c); err != nil {
			return err
		}
	}
	if !p.omitTyped && index%graphPeriod == 0 {
		addTypedPage(c, p)
	}
	return nil
}

// emitScheduledGraph adds the partial graph on its scheduled cases and rebuild targets.
//
// Takes c (*casefile.Case) which receives generated sources and expected results.
// Takes graph (partialGraph) which holds the drawn shared-partial graph.
// Takes target (string) which selects the subsystem under test.
// Takes index (uint64) which identifies the case's stream position.
func emitScheduledGraph(c *casefile.Case, graph partialGraph, target string, index uint64) {
	if index%graphPeriod != 0 && target != "incremental" && target != "graph-rebuild" {
		return
	}
	if target == "graph-rebuild" {
		graph.editLeaf = index%graphPeriod < 2
		graph.editHelper = index%graphPeriod == 0 || index%graphPeriod == 2
	}
	emitPartialGraph(c, graph)
}

// addFollowup adds source and expected observations for follow-up routes.
//
// Takes c (*casefile.Case) which receives generated sources and expected results.
// Takes p (Project) which holds the drawn application model.
func addFollowup(c *casefile.Case, p Project) {
	switch p.Followup {
	case followupRouteParam:
		c.Files["pages/items/{slug}.pk"] = "<template><p>{{ state.Slug }}</p></template>\n" + followupScript + followupSlugRender
		value := fmt.Sprintf("item-%d", p.Count)
		c.Probes = []casefile.Probe{textProbe("/items/"+value, value)}
		c.Features = append(c.Features, "route-param")
	case followupRedirect:
		c.Files["pages/moved.pk"] = "<template><p>moved</p></template>\n" + followupScript + followupRedirectRender
		moved := newProbe("/moved", http.StatusSeeOther)
		moved.Location = "/main"
		c.Probes = []casefile.Probe{moved}
		c.Features = append(c.Features, "redirect")
	case followupTypedError:
		c.Files["pages/missing.pk"] = "<template><p>unreachable</p></template>\n" + followupScript + followupMissingRender
		addMissingPage(c, "/missing")
		c.Features = append(c.Features, "typed-error")
	case followupUnmatched:
		addMissingPage(c, "/missing-route")
		c.Features = append(c.Features, "unmatched-route")
	case followupSharedPartial:
		addSharedPartialFollowup(c, p)
	case followupTranslation:
		addTranslationFollowup(c, p)
	}
}

// addSharedPartialFollowup adds two routes sharing one partial with different props.
//
// Takes c (*casefile.Case) which receives generated sources and expected results.
// Takes p (Project) which holds the drawn application model.
func addSharedPartialFollowup(c *casefile.Case, p Project) {
	const badge = "<template><span data-fuzz-element=\"badge\" p-text=\"props.Label\"></span></template>\n"
	c.Files["partials/badge.pk"] = badge + followupScript + followupBadgeRender
	for _, route := range []string{"left", "right"} {
		label := route + "-" + strconv.Itoa(p.Count)
		c.Files["pages/"+route+".pk"] = fmt.Sprintf("<template><piko:partial is=\"badge\" server.label=%q></piko:partial></template>\n", label) + followupBadgePage
		contract := responseContract(nil, catalogueElement("badge", "span", label))
		contract.AppText = &label
		probe := semanticProbe("/"+route, contract)
		probe.Text = label
		c.Probes = append(c.Probes, probe)
	}
	c.Features = append(c.Features, "shared-partial-props")
}

// addTranslationFollowup adds a page translated through its own i18n block.
//
// Takes c (*casefile.Case) which receives generated sources and expected results.
// Takes p (Project) which holds the drawn application model.
func addTranslationFollowup(c *casefile.Case, p Project) {
	value := fmt.Sprintf("Translated item %d", p.Count)
	c.Files["pages/translation.pk"] = "<template><p>{{ T(\"message\") }}</p></template>\n" + followupScript + followupEmptyRender + "\n" +
		fmt.Sprintf("<i18n lang=\"json\">{\"en\":{\"message\":%q}}</i18n>", value)
	if c.Target == targetFormatter {
		const controls = `<p>Read <em>two</em> words.</p><input aria-label="Quantity" disabled :value="'one'">`
		c.Files["pages/translation.pk"] = strings.Replace(c.Files["pages/translation.pk"], "</template>", controls+"</template>", 1)
		c.Features = append(c.Features, "formatter-inline-attributes-i18n")
	}
	c.Features = append(c.Features, "page-translation")
	c.Probes = []casefile.Probe{textProbe("/translation", value)}
}

// addMissingPage adds a missing-route request and its expected response.
//
// Takes c (*casefile.Case) which receives generated sources and expected results.
// Takes path (string) which is the missing-route request path.
func addMissingPage(c *casefile.Case, path string) {
	c.Files["pages/!404.pk"] = "<template><main><h1>Page unavailable</h1>" + missingPageOutput + "</main></template>"
	missing := programProbe(path, programOutput("status", "404"))
	missing.Status = http.StatusNotFound
	c.Probes = []casefile.Probe{missing}
}

// infallible adapts a target emitter that cannot fail to the emitter signature.
//
// Takes add (func(*casefile.Case, Project)) which adds the target's sources.
//
// Returns func(*casefile.Case, Project) error which runs add and reports no error.
func infallible(add func(*casefile.Case, Project)) func(*casefile.Case, Project) error {
	return func(c *casefile.Case, p Project) error {
		add(c, p)
		return nil
	}
}

// addGuardsAndEquivalence adds the short-circuit guards and the equivalent routes.
//
// Takes c (*casefile.Case) which receives generated sources and expected results.
// Takes p (Project) which holds the drawn application model.
func addGuardsAndEquivalence(c *casefile.Case, p Project) {
	addGuards(c, p)
	addEquivalence(c, p)
}

// addCollectionEditions adds the collection routes and their edition history.
//
// Takes c (*casefile.Case) which receives generated sources and expected results.
// Takes p (Project) which holds the drawn application model.
//
// Returns error which reports a collection probe the history cannot revise, or nil on
// success.
func addCollectionEditions(c *casefile.Case, p Project) error {
	addCollectionRoutes(c, p)
	return addContentEditions(c, p)
}

// addIncrementalHistory adds the file-edit history on every graph-period case.
//
// Takes c (*casefile.Case) which receives generated sources and expected results.
// Takes p (Project) which holds the drawn application model.
func addIncrementalHistory(c *casefile.Case, p Project) {
	if c.Index%graphPeriod == 0 {
		addFileEditHistory(c, p)
	}
}

// profileDigest hashes every embedded generator source, template and model file.
//
// Returns string which is the hexadecimal SHA-256 digest, or an "unavailable: " marker
// naming the read failure with a random suffix, so that profile comparisons fail visibly
// and no two markers compare equal.
func profileDigest() string {
	parts, err := profileSources()
	if err != nil {
		return "unavailable: " + err.Error() + " " + rand.Text()
	}
	return fmt.Sprintf("%x", sha256.Sum256([]byte(strings.Join(parts, "\x00"))))
}

// profileSources lists the generator identity inputs in a deterministic order.
//
// Returns []string which holds the fixture digests followed by each embedded generator
// file and the casefile model.
// Returns error which reports an unreadable embedded file, or nil on success.
func profileSources() ([]string, error) {
	parts := []string{gosource.SourceSHA256(), fixturego.SourceSHA256()}
	entries, err := fs.ReadDir(generatorFiles, ".")
	if err != nil {
		return nil, fmt.Errorf("reading embedded generator sources: %w", err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || strings.HasSuffix(name, "_test.go") {
			continue
		}
		data, err := fs.ReadFile(generatorFiles, name)
		if err != nil {
			return nil, fmt.Errorf("reading embedded generator source %s: %w", name, err)
		}
		parts = append(parts, name+"\x00"+string(data))
	}
	model, err := casefile.ModelSource()
	if err != nil {
		return nil, fmt.Errorf("reading casefile model: %w", err)
	}
	return append(parts, model), nil
}
