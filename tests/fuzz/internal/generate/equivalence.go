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
	"strconv"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

const (
	// equivalenceTemplate formats an equivalence page template from its number output
	// attributes, its output text and the partial's import alias.
	equivalenceTemplate = `<main id="fuzz-program"><output data-fuzz-output="number" %s>%s</output>` +
		`<piko:partial is=%q :server.label="state.Label"></piko:partial></main>`

	// guardsTemplate holds the short-circuit guard page template.
	guardsTemplate = `<main id="fuzz-program"><output data-fuzz-output="enabled">{{ state.User != nil && state.User.Enabled ? 'active' : 'absent' }}</output>` +
		`<output data-fuzz-output="ratio">{{ state.Divisor == 0 || state.Amount / state.Divisor >= 2 ? 'yes' : 'no' }}</output></main>`
)

// addEquivalence adds source and expected observations for equivalent route expressions.
//
// Takes c (*casefile.Case) which receives generated sources and expected results.
// Takes p (Project) which holds the drawn application model.
func addEquivalence(c *casefile.Case, p Project) {
	leaf := "partials/equivalence/badge.pk"
	c.Files[leaf] = graphSFC(`<output data-fuzz-output="partial" :data-value="props.Label">{{ props.Label }}</output>`, "", `type Props struct {Label string}
type Response struct{}
func Render(_ *piko.RequestData,_ Props)(Response,piko.Metadata,error){return Response{},piko.Metadata{},nil}`)
	const helper = "model/equivalence"
	if c.Target != "wasm" {
		c.Files[helper+"/scale.go"] = "package equivalence\nfunc Number(input int) int {return scale(input)+1}\n"
		c.Files[helper+"/bias.go"] = "package equivalence\nfunc scale(input int) int{return input*2}\n"
	}
	for _, variant := range []string{"base", "aliased", "long", "split", "changed"} {
		if c.Target == "wasm" && variant == "split" {
			continue
		}
		alias, binding, mark, expression := "badge", ":data-value", "{{ state.Value }}", "(2*n)+1"
		imports := ""
		switch variant {
		case "aliased":
			alias = "stamp"
		case "long":
			binding = "p-bind:data-value"
			mark = ""
		case "split":
			expression = "convert.Number(n)"
			imports = fmt.Sprintf("convert %q\n", casefile.Module+"/"+helper)
		case "changed":
			expression = "(2*n)+2"
		}
		attributes := binding + `="state.Label"`
		if variant == "long" {
			attributes += ` p-text="state.Value"`
		}
		template := fmt.Sprintf(equivalenceTemplate, attributes, mark, alias)
		imports += fmt.Sprintf("%s %q", alias, casefile.Module+"/"+leaf)
		code := fmt.Sprintf(`type Response struct {Value int; Label string}
func Render(_ *piko.RequestData,_ piko.NoProps)(Response,piko.Metadata,error) {
 n:=%d
 return Response{Value: %s, Label: %s},piko.Metadata{},nil
}`, p.Count, expression, strconv.Quote(p.Title))
		route := "/equivalence-" + variant
		c.Files["pages"+route+".pk"] = graphSFC(template, imports, code)
		number := 2*p.Count + 1
		if variant == "changed" {
			number++
		}
		probe := programProbe(route, valuedOutput("number", strconv.Itoa(number), p.Title), valuedOutput("partial", p.Title, p.Title))
		probe.Stage = "rewrite-" + variant
		probe.BrowserProgram = c.Target != "wasm"
		c.Probes = append(c.Probes, probe)
	}
	c.Features = append(c.Features, "source-equivalence", "source-localized-edit", "partial-alias", "bind-long-short")
}

// addGuards adds source and expected observations for short-circuit guards.
//
// Takes c (*casefile.Case) which receives generated sources and expected results.
// Takes p (Project) which holds the drawn application model.
func addGuards(c *casefile.Case, p Project) {
	c.Files["pages/guards.pk"] = graphSFC(guardsTemplate, "", fmt.Sprintf(`type User struct{Enabled bool}
type Response struct{User *User;Amount,Divisor int}
func Render(r *piko.RequestData,_ piko.NoProps)(Response,piko.Metadata,error){
 s:=Response{Amount:%d}
 switch r.QueryParam("case"){case "present":s.User=&User{Enabled:true};s.Divisor=s.Amount;case "disabled":s.User=&User{};s.Divisor=1}
 return s,piko.Metadata{},nil
}`, 2+p.Count))
	guards := []struct{ name, user, ratio string }{
		{name: "none", user: "absent", ratio: "yes"},
		{name: "present", user: "active", ratio: "no"},
		{name: "disabled", user: "absent", ratio: "yes"},
	}
	for _, guard := range guards {
		c.Probes = append(c.Probes, programProbe("/guards?case="+guard.name, programOutput("enabled", guard.user), programOutput("ratio", guard.ratio)))
	}
	c.Features = append(c.Features, "short-circuit-guard", "type-pointer-boundary")
}
