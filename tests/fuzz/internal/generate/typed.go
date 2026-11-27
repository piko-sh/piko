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
	"math/big"
	"strconv"
	"strings"
	"time"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

const (
	// typedBaseCents is the lowest generated price in hundredths.
	typedBaseCents = 101

	// typedCentsStep is the price step in hundredths per counter value.
	typedCentsStep = 17

	// centsPerUnit is the number of hundredths in one whole price unit.
	centsPerUnit = 100

	// typedBigShift is the bit shift of the smallest generated large integer.
	typedBigShift = 72

	// typedBaseDay is the earliest generated day of February 2024.
	typedBaseDay = 11

	// typedShiftDays is the day offset applied to the date in negated modes.
	typedShiftDays = 31

	// typedDay is the length of one calendar day in the generated dates.
	typedDay = 24 * time.Hour

	// typedBigAddend is the large-integer literal the template adds.
	typedBigAddend = 7

	// typedDurationMinutes is the total of the template's duration literals in minutes.
	typedDurationMinutes = 75

	// typedOther is the conditional fallback text and the alternate scoped view.
	typedOther = "other"

	// typedPageTemplate formats the typed-value page from its outputs, label, date parts,
	// price and large integer.
	typedPageTemplate = `<template><main id="fuzz-program">%s</main></template>
<script type="application/x-go">
package main
import (
 "time"
 "piko.sh/piko"
 "piko.sh/piko/wdk/maths"
)
type User struct {Name string}
type Response struct {Values map[string]any; User *User; Full bool; Label string; Price maths.Decimal; Large maths.BigInt; Day time.Time; Step time.Duration}
func Render(r *piko.RequestData,props piko.NoProps)(Response,piko.Metadata,error) {
 values:=map[string]any{}
 var user *User
 mode:=r.QueryParam("mode")
 label:=%q
 switch mode {case "nil":values["key"]=nil;case "empty":values["key"]="";user=&User{};case "full":values["key"]=label;user=&User{Name:label}}
 day:=time.Date(%d,%d,%d,0,0,0,0,time.UTC)
 price,large:=%q,%q
 if mode=="nil" || mode=="empty" {price="-"+price;large="-"+large;day=day.AddDate(0,0,31)}
 return Response{Values:values,User:user,Full:mode=="full",Label:label,Price:maths.NewDecimalFromString(price),` +
		`Large:maths.NewBigIntFromString(large),Day:day,Step:24*time.Hour},piko.Metadata{},nil
}
</script>
`

	// scopedPageTemplate formats the scoped-binding page from its two fragments.
	scopedPageTemplate = `<template><main id="fuzz-program">
<section p-context="'first'"><output data-fuzz-output="left" data-fuzz-check-key="true" p-key="'detail'">left</output></section>
<section p-context="'second'"><output data-fuzz-output="right" data-fuzz-check-key="true" p-key="'detail'">right</output></section>
<div id="rich-fragment" p-html="state.Fragment"></div></main></template>
<script type="application/x-go">
package main
import "piko.sh/piko"
type Response struct{ Fragment string }
func Render(r *piko.RequestData, props piko.NoProps)(Response,piko.Metadata,error) {
 fragment:=%q
 if r.QueryParam("view")=="other" {fragment=%q}
 return Response{Fragment:fragment},piko.Metadata{},nil
}
</script>`
)

var (
	// typedFields lists independently checked typed-value fields.
	typedFields = []struct{ id, expression string }{
		{id: "lookup", expression: "state.Values['key'] ?? 'fallback'"},
		{id: "member", expression: "state.User?.Name"},
		{id: "choice", expression: "state.Full ? state.Label : 'other'"},
		{id: "decimal", expression: "state.Price + 0.01d"},
		{id: "decimal-order", expression: "state.Price > 1d ? 'above' : 'below'"},
		{id: "big", expression: "state.Large + 7n"},
		{id: "big-order", expression: "state.Large > 9007199254740993n ? 'above' : 'below'"},
		{id: "date", expression: "state.Day.Format('2006-01-02')"},
		{id: "date-order", expression: "state.Day < d'2024-03-01' ? 'before' : 'after'"},
		{id: "date-add", expression: "(state.Day + state.Step).Format('2006-01-02')"},
		{id: "duration", expression: "du'30m' + du'45m'"},
	}

	// typedModes lists generated typed-value variants.
	typedModes = []string{"missing", "nil", "empty", "full"}
)

// addTypedPage adds source and expected observations for typed-value routes.
//
// Takes c (*casefile.Case) which receives generated sources and expected results.
// Takes p (Project) which holds the drawn application model.
func addTypedPage(c *casefile.Case, p Project) {
	cents := typedBaseCents + typedCentsStep*p.Count
	price := fmt.Sprintf("%d.%02d", cents/centsPerUnit, cents%centsPerUnit)
	large := new(big.Int).Lsh(big.NewInt(1), uint(typedBigShift+p.Count))
	large.Add(large, big.NewInt(int64(len(p.Rows))))
	date := time.Date(2024, 2, typedBaseDay+p.Count, 0, 0, 0, 0, time.UTC)
	var body strings.Builder
	for position, field := range typedFields {
		if p.skipTypedFields&(1<<position) != 0 {
			continue
		}
		fmt.Fprintf(&body, `<output data-fuzz-output="%s" p-text="%s"></output>`, field.id, html.EscapeString(field.expression))
	}
	c.Files["pages/typed.pk"] = fmt.Sprintf(typedPageTemplate, body.String(), p.Title, date.Year(), date.Month(), date.Day(), price, large.String())
	for modeIndex, mode := range typedModes {
		if p.skipTypedModes&(1<<modeIndex) != 0 {
			continue
		}
		probe := programProbe("/typed?mode=" + mode)
		probe.BrowserProgram = true
		for position, value := range typedExpectations(p.Title, mode, cents, large, date) {
			if p.skipTypedFields&(1<<position) != 0 {
				continue
			}
			probe.Output = append(probe.Output, programOutput(typedFields[position].id, value))
		}
		c.Probes = append(c.Probes, probe)
	}
	c.Features = append(c.Features, "typed-optionals", "typed-map-lookup", "typed-decimal", "typed-bigint", "typed-temporal")
}

// typedExpectations models every typed field's rendered text for one mode.
//
// Takes title (string) which is the generated label.
// Takes mode (string) which selects the typed-value variant.
// Takes cents (int) which is the generated price in hundredths.
// Takes large (*big.Int) which is the generated large integer and is not modified.
// Takes date (time.Time) which is the generated date.
//
// Returns []string which holds the expected text of each typed field in order.
func typedExpectations(title, mode string, cents int, large *big.Int, date time.Time) []string {
	lookup, member, choice := "fallback", "", typedOther
	switch mode {
	case "empty":
		lookup = ""
	case "full":
		lookup, member, choice = title, title, title
	}
	wantCents, wantBig, wantDate := cents, new(big.Int).Set(large), date
	order, dayOrder := "above", "before"
	if mode == "nil" || mode == "empty" {
		wantCents = -wantCents
		wantBig.Neg(wantBig)
		wantDate = date.Add(typedShiftDays * typedDay)
		order, dayOrder = "below", "after"
	}
	decimal := strconv.FormatFloat(float64(wantCents+1)/centsPerUnit, 'f', -1, 64)
	bigSum := wantBig.Add(wantBig, big.NewInt(typedBigAddend)).String()
	return []string{
		lookup, member, choice, decimal, order, bigSum, order,
		wantDate.Format(time.DateOnly), dayOrder, wantDate.Add(typedDay).Format(time.DateOnly), (typedDurationMinutes * time.Minute).String(),
	}
}

// addScopedPage adds source and expected observations for scoped-binding routes.
//
// Takes c (*casefile.Case) which receives generated sources and expected results.
// Takes p (Project) which holds the drawn application model.
func addScopedPage(c *casefile.Case, p Project) {
	fragment := `<output data-fuzz-output="markup" data-value="rich">` + html.EscapeString(p.Title) + `</output>`
	c.Files["pages/scoped.pk"] = fmt.Sprintf(scopedPageTemplate, fragment, `<output data-fuzz-output="markup" data-value="alternate"><strong>other</strong></output>`)
	for _, view := range []string{"first", typedOther} {
		left := programOutput("left", "left")
		left.ScopedKey, left.FullKey, left.Owner = "first/detail", "first:0.detail", "main#fuzz-program/section"
		right := programOutput("right", "right")
		right.ScopedKey, right.FullKey, right.Owner = "second/detail", "second:0.detail", "main#fuzz-program/section"
		markup := valuedOutput("markup", p.Title, "rich")
		markup.Owner = "main#fuzz-program/div#rich-fragment"
		if view == typedOther {
			markup.Text, markup.Value = typedOther, "alternate"
		}
		probe := programProbe("/scoped?view="+view, left, right, markup)
		probe.BrowserProgram = true
		probe.ProgramRoot = "main"
		c.Probes = append(c.Probes, probe)
	}
	c.Features = append(c.Features, "server-key-namespaces", "server-html-fragment")
}
