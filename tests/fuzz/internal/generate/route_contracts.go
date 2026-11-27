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
	"net/http"
	"strconv"
	"strings"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

const (
	// expressionRemainderDivisor is the divisor of the modelled template remainder
	// expression.
	expressionRemainderDivisor = 3

	// bulletinDirectory is the bulletin partials' source directory.
	bulletinDirectory = "partials/bulletin/"

	// bulletinTrace is the response header each bulletin middleware appends to.
	bulletinTrace = "X-Edition-Trace"

	// bulletinBase holds the bulletin layout partial with its slot fallbacks.
	bulletinBase = `<template><article><header><piko:slot name="caption"><h2 data-fuzz-element="caption">Editorial fallback</h2></piko:slot></header>` +
		`<section><piko:slot><p data-fuzz-element="copy">Empty edition</p></piko:slot></section></article></template>`

	// bulletinLeftTemplate holds the lead card's template, which fills the layout.
	bulletinLeftTemplate = `<piko:partial is="frame"><h2 p-slot="caption" data-fuzz-element="caption">Lead edition</h2>` +
		`<p data-fuzz-element="copy" p-text="props.Label"></p></piko:partial>`

	// bulletinLeftScript holds the lead card's script.
	bulletinLeftScript = `type Props struct {Label string}
func Render(_ *piko.RequestData,_ Props)(piko.NoResponse,piko.Metadata,error){return piko.NoResponse{},piko.Metadata{},nil}`

	// bulletinRightScript holds the side card's script.
	bulletinRightScript = `func Render(_ *piko.RequestData,_ piko.NoProps)(piko.NoResponse,piko.Metadata,error){return piko.NoResponse{},piko.Metadata{},nil}`

	// bulletinPageScript holds the bulletin page's middleware and render script.
	bulletinPageScript = `type Response struct{Label string}
func outer(next http.Handler)http.Handler{return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){` +
		`w.Header().Add("X-Edition-Trace","outer");next.ServeHTTP(w,r)})}
func inner(next http.Handler)http.Handler{return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){` +
		`w.Header().Add("X-Edition-Trace","inner");if r.URL.Query().Get("move")=="yes"{http.Redirect(w,r,"/bulletins/featured",303);return};next.ServeHTTP(w,r)})}
func Middlewares()[]func(http.Handler)http.Handler{return []func(http.Handler)http.Handler{outer,inner}}
func Render(r *piko.RequestData,_ piko.NoProps)(Response,piko.Metadata,error){
 slug:=r.PathParam("slug")
 if r.QueryParam("archived")=="yes"{return Response{},piko.Metadata{},piko.Forbidden("Archived "+slug)}
 href:="https://journal.example/en/"+slug
 return Response{Label:"Report "+slug},piko.Metadata{Title:"Edition "+slug,Description:"Daily bulletin",Language:"en-GB",RobotsRule:"index,follow",CanonicalURL:href,` +
		`AlternateLinks:[]map[string]string{{"hreflang":"fr","href":"https://journal.example/fr/"+slug},{"hreflang":"en-GB","href":href}},` +
		`OGTags:[]piko.OGTag{{Property:"og:title",Content:"Edition "+slug}}},nil
}`

	// bulletinArchiveScript holds the catch-all archive page's script.
	bulletinArchiveScript = `type Response struct{Path string}
 func Render(r *piko.RequestData,_ piko.NoProps)(Response,piko.Metadata,error){return Response{Path:r.PathParam("rest")},piko.Metadata{},nil}`

	// bulletinErrorScript holds the forbidden error page's script.
	bulletinErrorScript = `type Response struct{Message string}
 func Render(r *piko.RequestData,_ piko.NoProps)(Response,piko.Metadata,error){ctx:=piko.GetErrorContext(r);return Response{Message:ctx.Message},` +
		`piko.Metadata{Title:"Archived edition",Language:"en-GB",CanonicalURL:"https://journal.example/archived",RobotsRule:"noindex",` +
		`OGTags:[]piko.OGTag{{Property:"og:title",Content:"Archived edition"}}},nil}`
)

// addRouteContracts adds source and expected observations for mode-specific route
// contracts.
//
// Takes c (*casefile.Case) which receives generated sources and expected results.
// Takes p (Project) which holds the drawn application model.
func addRouteContracts(c *casefile.Case, p Project) {
	addExpressionBoundaries(c, p)
	token := "edition-" + strconv.Itoa(p.Count)
	c.Files[bulletinDirectory+"base.pk"] = bulletinBase
	c.Files[bulletinDirectory+"left/card.pk"] = graphSFC(bulletinLeftTemplate, fmt.Sprintf("frame %q", casefile.Module+"/"+bulletinDirectory+"base.pk"), bulletinLeftScript)
	c.Files[bulletinDirectory+"right/card.pk"] = graphSFC(`<piko:partial is="shell"></piko:partial>`, fmt.Sprintf("shell %q", casefile.Module+"/"+bulletinDirectory+"base.pk"), bulletinRightScript)
	const body = `<main><piko:partial is="lead" :server.label="state.Label"></piko:partial><piko:partial is="side"></piko:partial></main>`
	imports := fmt.Sprintf("lead %q\nside %q\n\"net/http\"", casefile.Module+"/"+bulletinDirectory+"left/card.pk", casefile.Module+"/"+bulletinDirectory+"right/card.pk")
	c.Files["pages/bulletins/{slug}.pk"] = graphSFC(body, imports, bulletinPageScript)
	c.Files["pages/bulletins/featured.pk"] = `<template><main><h1 data-fuzz-element="feature">Featured edition</h1></main></template>`
	c.Files["pages/bulletins/archive/{rest}*.pk"] = graphSFC(`<main><output data-fuzz-element="archive" p-text="state.Path"></output></main>`, "", bulletinArchiveScript)
	c.Files["pages/!403.pk"] = graphSFC(`<main><h1 data-fuzz-element="error" p-text="state.Message"></h1></main>`, "", bulletinErrorScript)
	c.Probes = append(c.Probes, bulletinProbes(token)...)
	c.Features = append(c.Features, "middleware-chain", "route-precedence", "typed-error-head", "layout-fill-fallback", "localized-alternate-links")
}

// bulletinProbes models the bulletin routes' contracts from route inputs and editorial
// policy, independently of the page AST, HTTP registration, rendered output or preparsed
// manifest.
//
// Takes token (string) which is the generated edition identifier.
//
// Returns []casefile.Probe which lists the bulletin requests in order.
func bulletinProbes(token string) []casefile.Probe {
	chain := map[string][]string{bulletinTrace: {"outer", "inner"}}
	route := "/bulletins/" + token
	href := "https://journal.example/en/" + token
	head := map[string][]string{
		"title": {"Edition " + token}, "description": {"Daily bulletin"}, "lang": {"en-GB"}, "canonical": {href},
		"alternate/fr": {"https://journal.example/fr/" + token}, "alternate/en-GB": {href}, "robots": {"index,follow"}, "og:title": {"Edition " + token},
	}
	published := semanticProbe(route, responseContract(head,
		catalogueElement("caption", "h2", "Lead edition"), catalogueElement("copy", "p", "Report "+token),
		catalogueElement("caption", "h2", "Editorial fallback"), catalogueElement("copy", "p", "Empty edition")))
	published.Stage = "published"
	published.ResponseHeaders = chain
	moved := newProbe(route+"?move=yes", http.StatusSeeOther)
	moved.Stage = "middleware-stop"
	moved.Location = "/bulletins/featured"
	moved.ResponseHeaders = chain
	featured := semanticProbe("/bulletins/featured", responseContract(nil, catalogueElement("feature", "h1", "Featured edition")))
	featured.Stage = "literal-precedence"
	featured.ResponseHeaders = map[string][]string{bulletinTrace: nil}
	archive := semanticProbe("/bulletins/archive/2026/"+token, responseContract(nil, catalogueElement("archive", "output", "2026/"+token)))
	archive.Stage = "catch-all"
	archived := newProbe(route+"?archived=yes", http.StatusForbidden)
	archived.Stage = "typed-error"
	archived.ResponseHeaders = chain
	archived.ByMode = editionErrorPolicy(token)
	return []casefile.Probe{published, moved, featured, archive, archived}
}

// addExpressionBoundaries adds source and expected observations for expression-boundary
// routes.
//
// Takes c (*casefile.Case) which receives generated sources and expected results.
// Takes p (Project) which holds the drawn application model.
func addExpressionBoundaries(c *casefile.Case, p Project) {
	const template = `<main id="fuzz-program"><section p-context="'edition'"><article p-for="row in state.Rows" p-key="row.ID"><div p-if="row.Show">FIELDS</div>` +
		`<div p-else><output data-fuzz-output="skip" :data-value="row.ID">omitted</output></div></article></section></main>`
	const record = `type Row struct{ID string;Number int;Text string;Point rune;Show bool}
type Response struct{Rows []Row}
func Render(r *piko.RequestData,_ piko.NoProps)(Response,piko.Metadata,error){
 rows:=[]Row{{"zero",0,"",'界',true},{"negative",-%d,"été",'海',true},{"skipped",1,"guest",'x',false}}
 if r.QueryParam("order")=="reverse"{rows[0],rows[1]=rows[1],rows[0]}
 return Response{Rows:rows},piko.Metadata{},nil
}`
	magnitude := p.Count + 2
	fields := []struct{ id, expression string }{
		{id: "coerce", expression: `row.Number ~= "0" ? 'same' : 'different'`},
		{id: "loose-not", expression: `row.Number !~= "0" ? 'different' : 'same'`},
		{id: "truth", expression: `~row.Text ? 'yes' : 'no'`},
		{id: "remainder", expression: "row.Number % " + strconv.Itoa(expressionRemainderDivisor)},
		{id: "template", expression: "`edition ${row.ID}: ${row.Number}`"},
		{id: "rune", expression: `row.Point == r'界' ? 'match' : 'miss'`},
		{id: "clock", expression: `(row.Number == 0 ? t'23:59:58' : t'23:59:59') < t'23:59:59' ? 'before' : 'after'`},
		{id: "instant", expression: `(row.Number == 0 ? dt'2024-03-01T00:00:00Z' : dt'2024-03-01T00:00:01Z') == dt'2024-02-29T19:00:00-05:00' ? 'same' : 'different'`},
	}
	var leaves strings.Builder
	for _, field := range fields {
		fmt.Fprintf(&leaves, `<output data-fuzz-output="%s" :data-value="row.ID" p-text="%s"></output>`, field.id, html.EscapeString(field.expression))
	}
	c.Files["pages/edition-expressions.pk"] = graphSFC(strings.Replace(template, "FIELDS", leaves.String(), 1), "", fmt.Sprintf(record, magnitude))
	for _, order := range []string{"forward", "reverse"} {
		probe := programProbe("/edition-expressions?order=" + order)
		ids := []string{"zero", "negative"}
		if order == "reverse" {
			ids[0], ids[1] = ids[1], ids[0]
		}
		for _, id := range ids {
			number := 0
			coercion, truth := "same", "no"
			point, clock, instant := "match", "before", "same"
			if id == "negative" {
				number = -magnitude
				coercion, truth = "different", "yes"
				point, clock, instant = "miss", "after", "different"
			}
			values := []string{coercion, coercion, truth, strconv.Itoa(number % expressionRemainderDivisor), "edition " + id + ": " + strconv.Itoa(number), point, clock, instant}
			for position, field := range fields {
				probe.Output = append(probe.Output, valuedOutput(field.id, values[position], id))
			}
		}
		probe.Output = append(probe.Output, valuedOutput("skip", "omitted", "skipped"))
		c.Probes = append(c.Probes, probe)
	}
	c.Features = append(c.Features, "typed-expression-boundaries", "conditional-keyed-groups")
}

// editionErrorPolicy models the intentional error-response differences between modes.
//
// Takes token (string) which is the generated edition identifier.
//
// Returns map[string]casefile.ModeResponse which maps server modes to their explicit
// error-response contracts.
func editionErrorPolicy(token string) map[string]casefile.ModeResponse {
	contracts := make(map[string]casefile.ModeResponse, 2)
	for _, mode := range []string{"compiled", "interp"} {
		message := "Archived " + token
		if mode == "interp" {
			message = "renderPage: runPage failed for pages/bulletins/{slug}.pk: " + message
		}
		head := map[string][]string{
			"title": {"Archived edition"}, "lang": {"en-GB"}, "canonical": {"https://journal.example/archived"}, "robots": {"noindex"}, "og:title": {"Archived edition"},
		}
		contracts[mode] = casefile.ModeResponse{Status: http.StatusForbidden, Semantic: *responseContract(head, catalogueElement("error", "h1", message))}
	}
	return contracts
}
