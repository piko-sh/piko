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
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"piko.sh/fuzzkit/tape"
	"piko.sh/safeconv"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

const (
	// opNumber reads the request-selected input number.
	opNumber = "number"

	// opBias reads the generated arithmetic bias.
	opBias = "bias"

	// opLiteral is a literal integer.
	opLiteral = "literal"

	// opIndex reads an enclosing loop's index.
	opIndex = "index"

	// opID reads an enclosing loop row's identifier.
	opID = "id"

	// opNeg negates its operand.
	opNeg = "neg"

	// kindOutput is an output template node.
	kindOutput = "output"

	// kindGroup is a grouping template node.
	kindGroup = "group"

	// kindBranch is a three-arm conditional template node.
	kindBranch = "branch"

	// kindLoop is a row loop template node.
	kindLoop = "loop"

	// branchArms is the number of arms of a conditional chain.
	branchArms = 3

	// compositeOdds is the denominator of the odds of a composite expression.
	compositeOdds = 3

	// literalChoices bounds generated literal integers.
	literalChoices = 7

	// biasChoices bounds the generated arithmetic bias.
	biasChoices = 4

	// firstRowID is the first generated row's identifier.
	firstRowID = 7

	// rowIDStep is the identifier gap between generated rows.
	rowIDStep = 3

	// extendedNesting is the extended profile's template nesting bound.
	extendedNesting = 3

	// extendedWidth is the extended profile's root and row bound.
	extendedWidth = 4

	// defaultWidth is the default profile's root and row bound.
	defaultWidth = 3

	// programRightInput is the input number of the right-hand program request.
	programRightInput = 3

	// programPageTemplate formats the program page from its markup, row literals, bias and
	// title.
	programPageTemplate = `<template><main id="fuzz-program">%s</main></template>
<script type="application/x-go">
package main
import "piko.sh/piko"
type Row struct { ID int; Label string }
type Response struct { Number, Bias int; Title string; Rows []Row }
func Render(r *piko.RequestData, props piko.NoProps)(Response,piko.Metadata,error) {
    n := 0
    rows := []Row{%s}
    switch r.QueryParam("input") {
    case "left": n = -2
    case "right": n = 3
    default: rows = nil
    }
    return Response{Number:n,Bias:%d,Title:%q,Rows:rows},piko.Metadata{},nil
}
</script>
`
)

var (
	// errMalformedProgram reports a generated program model that the evaluator or emitter
	// cannot represent.
	errMalformedProgram = errors.New("malformed generated program")
)

// programLimits bounds generated expressions, nesting and rows.
type programLimits struct {
	// Nesting bounds template control-flow depth.
	Nesting int

	// Terms bounds integer expression depth.
	Terms int

	// Roots bounds the number of root template nodes.
	Roots int

	// Rows bounds the number of loop rows.
	Rows int
}

// row holds a generated row's identity and label.
type row struct {
	// label is the displayed fixture label.
	label string

	// id identifies the generated item.
	id int
}

// scope holds one enclosing loop's row and index.
type scope struct {
	// row holds the enclosing loop's current row.
	row row

	// index is the enclosing loop index.
	index int
}

// inputs supplies values to the independent program evaluator.
type inputs struct {
	// title is the generated title text.
	title string

	// rows holds the generated input rows.
	rows []row

	// number is the request-selected input number.
	number int

	// bias is the generated arithmetic bias.
	bias int
}

// integer represents an integer expression over inputs and enclosing loops.
type integer struct {
	// left holds the first or only arithmetic operand.
	left *integer

	// right holds the second arithmetic operand.
	right *integer

	// operation selects the expression operation.
	operation string

	// value is the literal value or enclosing-loop depth.
	value int
}

// templateNode represents one generated output, branch or loop.
type templateNode struct {
	// value holds the output expression.
	value *integer

	// kind selects an output, group, branch or loop node.
	kind string

	// children holds nested template nodes.
	children []templateNode

	// id identifies the generated item.
	id int

	// scope selects an enclosing loop index, or the page title when negative.
	scope int

	// textDirective selects p-text output instead of interpolation.
	textDirective bool

	// longBind selects the p-bind attribute spelling.
	longBind bool
}

// program holds generated template trees and their input values.
type program struct {
	// trees holds the root template nodes.
	trees []templateNode

	// rows holds the generated input rows.
	rows []row

	// bias is the generated arithmetic bias.
	bias int
}

// source emits the integer expression's source.
//
// Returns string which is the emitted integer expression.
func (e *integer) source() string {
	switch e.operation {
	case opNumber:
		return "state.Number"
	case opBias:
		return "state.Bias"
	case opLiteral:
		return strconv.Itoa(e.value)
	case opIndex:
		return fmt.Sprintf("i%d", e.value)
	case opID:
		return fmt.Sprintf("r%d.ID", e.value)
	case opNeg:
		return "(-" + e.left.source() + ")"
	default:
		return "(" + e.left.source() + " " + e.operation + " " + e.right.source() + ")"
	}
}

// validate checks that the expression tree has known operations and complete operands.
//
// Returns error which reports an absent expression, an unknown operation or a missing
// operand, or nil when the tree can be emitted.
func (e *integer) validate() error {
	if e == nil {
		return fmt.Errorf("%w: absent integer expression", errMalformedProgram)
	}
	switch e.operation {
	case opNumber, opBias, opLiteral, opIndex, opID:
		return nil
	case opNeg:
		return e.left.validate()
	case "+", "-", "*":
		if err := e.left.validate(); err != nil {
			return err
		}
		return e.right.validate()
	default:
		return fmt.Errorf("%w: unknown integer production %q", errMalformedProgram, e.operation)
	}
}

// eval evaluates the integer expression against model inputs and loop scopes.
//
// Takes in (inputs) which supplies independent evaluator inputs.
// Takes stack ([]scope) which holds the enclosing loop values.
//
// Returns int which is the independently evaluated integer value.
// Returns error which reports an unknown operation, a missing operand or a loop reference
// outside the enclosing scopes, or nil on success.
func (e *integer) eval(in inputs, stack []scope) (int, error) {
	switch e.operation {
	case opNumber:
		return in.number, nil
	case opBias:
		return in.bias, nil
	case opLiteral:
		return e.value, nil
	case opIndex, opID:
		if e.value < 0 || e.value >= len(stack) {
			return 0, fmt.Errorf("%w: loop reference %d outside %d enclosing scopes", errMalformedProgram, e.value, len(stack))
		}
		if e.operation == opIndex {
			return stack[e.value].index, nil
		}
		return stack[e.value].row.id, nil
	case opNeg:
		if e.left == nil {
			return 0, fmt.Errorf("%w: negation without an operand", errMalformedProgram)
		}
		value, err := e.left.eval(in, stack)
		return -value, err
	case "+", "-", "*":
		if e.left == nil || e.right == nil {
			return 0, fmt.Errorf("%w: %q without two operands", errMalformedProgram, e.operation)
		}
		left, err := e.left.eval(in, stack)
		if err != nil {
			return 0, err
		}
		right, err := e.right.eval(in, stack)
		if err != nil {
			return 0, err
		}
		return apply(e.operation, left, right)
	default:
		return 0, fmt.Errorf("%w: unknown integer production %q", errMalformedProgram, e.operation)
	}
}

// write emits the template node and its children.
//
// Takes builder (*strings.Builder) which receives emitted template source.
//
// Returns error which reports a branch or loop without its required children, or nil on
// success.
func (n templateNode) write(builder *strings.Builder) error {
	switch n.kind {
	case kindOutput:
		return n.writeOutput(builder)
	case kindBranch:
		return n.writeBranch(builder)
	case kindLoop:
		if len(n.children) == 0 {
			return fmt.Errorf("%w: loop %d has no body", errMalformedProgram, n.id)
		}
		fmt.Fprintf(builder, `<section p-for="(i%d, r%d) in state.Rows" p-key="r%d.ID">`, n.scope, n.scope, n.scope)
		if err := n.children[0].write(builder); err != nil {
			return err
		}
		builder.WriteString("</section>")
	case kindGroup:
		builder.WriteString("<article>")
		for _, child := range n.children {
			if err := child.write(builder); err != nil {
				return err
			}
		}
		builder.WriteString("</article>")
	}
	return nil
}

// writeBranch emits a three-arm conditional chain node.
//
// Takes builder (*strings.Builder) which receives emitted template source.
//
// Returns error which reports a branch without exactly three arms, or nil on success.
func (n templateNode) writeBranch(builder *strings.Builder) error {
	attributes := []string{`p-if="state.Number < 0"`, `p-else-if="state.Number == 0"`, "p-else"}
	if len(n.children) != len(attributes) {
		return fmt.Errorf("%w: branch %d has %d arms", errMalformedProgram, n.id, len(n.children))
	}
	for position, attribute := range attributes {
		builder.WriteString("<div " + attribute + ">")
		if err := n.children[position].write(builder); err != nil {
			return err
		}
		builder.WriteString("</div>")
	}
	return nil
}

// writeOutput emits an output node.
//
// Takes builder (*strings.Builder) which receives emitted template source.
//
// Returns error which reports a malformed output expression, or nil on success.
func (n templateNode) writeOutput(builder *strings.Builder) error {
	if err := n.value.validate(); err != nil {
		return err
	}
	label := "state.Title"
	if n.scope >= 0 {
		label = fmt.Sprintf("r%d.Label", n.scope)
	}
	bind := ":data-value"
	if n.longBind {
		bind = "p-bind:data-value"
	}
	fmt.Fprintf(builder, `<output data-fuzz-output="n%d" %s="%s"`, n.id, bind, label)
	if n.textDirective {
		fmt.Fprintf(builder, ` p-text="%s"></output>`, n.value.source())
	} else {
		fmt.Fprintf(builder, `>{{ %s }}</output>`, n.value.source())
	}
	return nil
}

// limitsFor selects structural limits for the saved size profile.
//
// Takes size (string) which selects structural generation bounds.
//
// Returns programLimits which holds the selected structural bounds.
func limitsFor(size string) programLimits {
	switch size {
	case "tiny":
		return programLimits{Nesting: 1, Terms: 1, Roots: 1, Rows: 2}
	case "extended":
		return programLimits{Nesting: extendedNesting, Terms: 2, Roots: extendedWidth, Rows: extendedWidth}
	default:
		return programLimits{Nesting: 2, Terms: 2, Roots: defaultWidth, Rows: defaultWidth}
	}
}

// generatesProgram reports whether the target includes generated control flow.
//
// Takes target (string) which selects the subsystem under test.
//
// Returns bool which reports whether the target emits typed template control flow.
func generatesProgram(target string) bool {
	switch target {
	case "pages", "components", "http-actions", "syntax", targetFormatter, "annotator", "ast":
		return true
	}
	return false
}

// drawInteger draws a typed integer expression over inputs and enclosing loops.
//
// Takes t (tape.Source) which supplies bounded deterministic choices.
// Takes depth (int) which is the remaining expression-recursion budget.
// Takes scopes (int) which counts lexically enclosing loops.
//
// Returns *integer which holds the generated typed integer expression tree.
func drawInteger(t tape.Source, depth, scopes int) *integer {
	if depth > 0 && t.Bool(2, compositeOdds) {
		operations := []string{"+", "-", "*", opNeg}
		operation := operations[t.Pool(len(operations))]
		expression := &integer{left: drawInteger(t, depth-1, scopes), right: nil, operation: operation, value: 0}
		if operation != opNeg {
			expression.right = drawInteger(t, depth-1, scopes)
		}
		return expression
	}
	options := []string{opNumber, opBias, opLiteral}
	if scopes > 0 {
		options = append(options, opIndex, opID)
	}
	expression := &integer{left: nil, right: nil, operation: options[t.Pool(len(options))], value: 0}
	switch expression.operation {
	case opLiteral:
		expression.value = safeconv.Uint64ToInt(t.Uint(literalChoices))
	case opIndex, opID:
		expression.value = t.Pool(scopes)
	}
	return expression
}

// drawNode draws a bounded output, branch or loop node.
//
// Takes t (tape.Source) which supplies bounded deterministic choices.
// Takes depth (int) which is the current template nesting depth.
// Takes scopes (int) which counts lexically enclosing loops.
// Takes serial (*int) which allocates stable output-node identifiers.
// Takes limit (programLimits) which bounds expression and template structure.
//
// Returns templateNode which holds the drawn node and its child nodes.
func drawNode(t tape.Source, depth, scopes int, serial *int, limit programLimits) templateNode {
	node := templateNode{value: nil, kind: kindOutput, children: nil, id: *serial, scope: scopes, textDirective: false, longBind: false}
	*serial++
	if depth < limit.Nesting {
		kinds := []string{kindOutput, kindGroup, kindBranch, kindLoop}
		node.kind = kinds[t.Pool(len(kinds))]
	}
	switch node.kind {
	case kindOutput:
		node.value = drawInteger(t, limit.Terms, scopes)
		node.textDirective = t.Bool(1, 2)
		node.longBind = t.Bool(1, 2)

		node.scope = t.Pool(scopes+1) - 1
	case kindLoop:
		node.children = []templateNode{drawNode(t, depth+1, scopes+1, serial, limit)}
	case kindGroup, kindBranch:
		width := branchArms
		if node.kind == kindGroup {
			width = 1 + safeconv.Uint64ToInt(t.Uint(2))
		}
		for range width {
			node.children = append(node.children, drawNode(t, depth+1, scopes, serial, limit))
		}
	}
	return node
}

// evaluateNodes models template outputs in document order.
//
// Takes nodes ([]templateNode) which holds the ordered template nodes.
// Takes in (inputs) which supplies independent evaluator inputs.
// Takes stack ([]scope) which holds the enclosing loop values.
// Takes parentKey (string) which is the enclosing rendered scope key.
// Takes owner (string) which identifies the component owning the output.
// Takes out (*[]casefile.ProgramOutput) which receives modelled output leaves in document
// order.
//
// Returns error which reports a malformed node or expression, or nil on success.
func evaluateNodes(nodes []templateNode, in inputs, stack []scope, parentKey, owner string, out *[]casefile.ProgramOutput) error {
	slot := 0
	for index := range nodes {
		node := &nodes[index]
		childKey := func(offset int) string { return parentKey + ":" + strconv.Itoa(slot+offset) }
		var err error
		switch node.kind {
		case kindOutput:
			err = evaluateOutput(node, in, stack, childKey(0), owner, out)
		case kindBranch:
			arm := branchArm(in.number)
			err = evaluateBranch(node, arm, in, stack, childKey(arm), owner+"/div", out)
			slot += 2
		case kindLoop:
			err = evaluateLoop(node, in, stack, childKey(0), owner+"/section", out)
		case kindGroup:
			err = evaluateNodes(node.children, in, stack, childKey(0), owner+"/article", out)
		}
		if err != nil {
			return err
		}
		slot++
	}
	return nil
}

// evaluateOutput models one output node's rendered value and label.
//
// Takes node (*templateNode) which is the output node.
// Takes in (inputs) which supplies independent evaluator inputs.
// Takes stack ([]scope) which holds the enclosing loop values.
// Takes key (string) which is the output's rendered scope key.
// Takes owner (string) which identifies the component owning the output.
// Takes out (*[]casefile.ProgramOutput) which receives the modelled output leaf.
//
// Returns error which reports a malformed expression or label scope, or nil on success.
func evaluateOutput(node *templateNode, in inputs, stack []scope, key, owner string, out *[]casefile.ProgramOutput) error {
	label := in.title
	if node.scope >= 0 {
		if node.scope >= len(stack) {
			return fmt.Errorf("%w: output %d labels scope %d outside %d enclosing scopes", errMalformedProgram, node.id, node.scope, len(stack))
		}
		label = stack[node.scope].row.label
	}
	value, err := node.value.eval(in, stack)
	if err != nil {
		return err
	}
	output := valuedOutput(fmt.Sprintf("n%d", node.id), strconv.Itoa(value), label)
	output.FullKey = key
	output.Owner = owner
	*out = append(*out, output)
	return nil
}

// evaluateBranch models the rendered arm of a conditional chain node.
//
// Takes node (*templateNode) which is the branch node.
// Takes arm (int) which is the rendered arm's position.
// Takes in (inputs) which supplies independent evaluator inputs.
// Takes stack ([]scope) which holds the enclosing loop values.
// Takes key (string) which is the rendered arm's scope key.
// Takes owner (string) which identifies the element owning the arm.
// Takes out (*[]casefile.ProgramOutput) which receives modelled output leaves.
//
// Returns error which reports a missing arm or a malformed node, or nil on success.
func evaluateBranch(node *templateNode, arm int, in inputs, stack []scope, key, owner string, out *[]casefile.ProgramOutput) error {
	if arm >= len(node.children) {
		return fmt.Errorf("%w: branch %d has %d arms", errMalformedProgram, node.id, len(node.children))
	}
	return evaluateNodes(node.children[arm:arm+1], in, stack, key, owner, out)
}

// evaluateLoop models a loop node's outputs for every input row.
//
// Takes node (*templateNode) which is the loop node.
// Takes in (inputs) which supplies independent evaluator inputs.
// Takes stack ([]scope) which holds the enclosing loop values.
// Takes key (string) which is the loop's rendered scope key.
// Takes owner (string) which identifies the element owning the loop body.
// Takes out (*[]casefile.ProgramOutput) which receives modelled output leaves.
//
// Returns error which reports a malformed node or expression, or nil on success.
func evaluateLoop(node *templateNode, in inputs, stack []scope, key, owner string, out *[]casefile.ProgramOutput) error {
	for position, current := range in.rows {
		inner := slices.Concat(stack, []scope{{row: current, index: position}})
		if err := evaluateNodes(node.children, in, inner, key+"."+strconv.Itoa(current.id), owner, out); err != nil {
			return err
		}
	}
	return nil
}

// branchArm selects the conditional chain arm rendered for an input number.
//
// Takes number (int) which is the request-selected input number.
//
// Returns int which is the rendered arm: negative, zero or positive.
func branchArm(number int) int {
	switch {
	case number < 0:
		return 0
	case number > 0:
		return 2
	default:
		return 1
	}
}

// drawProgram draws template trees and their input values.
//
// Takes t (tape.Source) which supplies bounded deterministic choices.
// Takes limit (programLimits) which bounds expression and template structure.
//
// Returns program which holds the typed template trees and their inputs.
func drawProgram(t tape.Source, limit programLimits) program {
	serial := 0
	var trees []templateNode
	for remaining := 1 + safeconv.Uint64ToInt(t.Uint(safeconv.IntToUint64(limit.Roots))); remaining > 0; remaining-- {
		trees = append(trees, drawNode(t, 0, 0, &serial, limit))
	}
	bias := 1 + safeconv.Uint64ToInt(t.Uint(biasChoices))
	rows := make([]row, 1+safeconv.Uint64ToInt(t.Uint(safeconv.IntToUint64(limit.Rows))))
	for position := range rows {
		rows[position] = row{id: firstRowID + position*rowIDStep, label: choice(t)}
	}
	return program{trees: trees, bias: bias, rows: rows}
}

// shrinkProgram tries smaller typed trees while recomputing their expectations.
//
// Takes p (*Project) which holds the application model being simplified.
// Takes try (func(Project)) which tests each simplified model while retaining the same
// confirmed failure.
func shrinkProgram(p *Project, try func(Project)) {
	replace := func(model program) {
		next := *p
		next.program = &model
		try(next)
	}
	for position := range slices.Backward(p.program.trees) {
		model := *p.program
		model.trees = slices.Delete(slices.Clone(model.trees), position, position+1)
		replace(model)
	}
	for position := range slices.Backward(p.program.rows) {
		model := *p.program
		model.rows = slices.Delete(slices.Clone(model.rows), position, position+1)
		replace(model)
	}
	for position, node := range p.program.trees {
		model := *p.program
		model.trees = slices.Clone(model.trees)
		number := &integer{left: nil, right: nil, operation: opNumber, value: 0}
		model.trees[position] = templateNode{value: number, kind: kindOutput, children: nil, id: node.id, scope: -1, textDirective: false, longBind: false}
		replace(model)
	}
}

// emitProgram adds control-flow source and independently evaluated outputs.
//
// Takes c (*casefile.Case) which receives generated sources and expected results.
// Takes title (string) which is the generated page title.
// Takes model (program) which holds the generated typed program model.
//
// Returns error which reports a malformed program model, or nil on success.
func emitProgram(c *casefile.Case, title string, model program) error {
	var literal strings.Builder
	for _, current := range model.rows {
		fmt.Fprintf(&literal, "{ID:%d,Label:%q},", current.id, current.label)
	}
	var markup strings.Builder
	for _, tree := range model.trees {
		if err := tree.write(&markup); err != nil {
			return err
		}
	}
	c.Files["pages/program.pk"] = fmt.Sprintf(programPageTemplate, markup.String(), literal.String(), model.bias, title)
	requests := []struct {
		name   string
		number int
	}{
		{name: "left", number: -2},
		{name: "zero", number: 0},
		{name: "right", number: programRightInput},
	}
	for _, request := range requests {
		in := inputs{title: title, rows: model.rows, number: request.number, bias: model.bias}
		if in.number == 0 {
			in.rows = nil
		}
		probe := programProbe("/program?input=" + request.name)
		probe.ProgramRoot = "main"
		if err := evaluateNodes(model.trees, in, nil, "r.0", "main#fuzz-program", &probe.Output); err != nil {
			return err
		}
		c.Probes = append(c.Probes, probe)
	}
	c.Features = append(c.Features, "server-program")
	markers := []struct{ token, feature string }{
		{token: "p-else-if=", feature: "server-conditional-chain"},
		{token: "p-for=", feature: "server-indexed-loop"},
		{token: "(i1, r1)", feature: "server-nested-loop"},
		{token: "p-text=", feature: "server-text-directive"},
		{token: "p-bind:data-value", feature: "server-long-bind"},
		{token: ":data-value=", feature: "server-bound-attribute"},
	}
	for _, marker := range markers {
		if strings.Contains(markup.String(), marker.token) {
			c.Features = append(c.Features, marker.feature)
		}
	}
	return nil
}
