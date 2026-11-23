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
	"strings"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

const (
	// arithmeticHighOperand is the largest operand in the arithmetic matrix.
	arithmeticHighOperand = 3

	// arithmeticTemplate formats the arithmetic page from its outputs and operand triples.
	arithmeticTemplate = `<template><main id="fuzz-program"><section p-for="r in state.Triples" p-key="r.Label">%s</section></main></template>
<script type="application/x-go">
package main
import "piko.sh/piko"
type Triple struct {A,B,C int; Label string}
type Response struct {Triples []Triple}
func Render(r *piko.RequestData, props piko.NoProps)(Response,piko.Metadata,error) {
    return Response{Triples:[]Triple{%s}},piko.Metadata{},nil
}
</script>
`
)

// arithmeticShape defines one two-operator arithmetic expression.
type arithmeticShape struct {
	// id identifies the arithmetic shape.
	id string

	// outer is the operator applied last.
	outer string

	// inner is the operator applied to the nested pair.
	inner string

	// left places the nested operation on the left when true.
	left bool
}

// expression formats the arithmetic expression.
//
// Returns string which is the parenthesised expression source.
func (s arithmeticShape) expression() string {
	if s.left {
		return fmt.Sprintf("(r.A %s r.B) %s r.C", s.inner, s.outer)
	}
	return fmt.Sprintf("r.A %s (r.B %s r.C)", s.outer, s.inner)
}

// result evaluates the arithmetic shape on its operands.
//
// Takes a (int) which is the first arithmetic operand.
// Takes b (int) which is the second arithmetic operand.
// Takes c (int) which is the third arithmetic operand.
//
// Returns int which is the independently evaluated arithmetic result.
// Returns error which reports an operator outside the arithmetic vocabulary, or nil on
// success.
func (s arithmeticShape) result(a, b, c int) (int, error) {
	if s.left {
		nested, err := apply(s.inner, a, b)
		if err != nil {
			return 0, err
		}
		return apply(s.outer, nested, c)
	}
	nested, err := apply(s.inner, b, c)
	if err != nil {
		return 0, err
	}
	return apply(s.outer, a, nested)
}

// arithmeticShapes enumerates the fixed arithmetic expression matrix.
//
// Returns []arithmeticShape which lists arithmetic shape values in their recorded order.
func arithmeticShapes() []arithmeticShape {
	operators := []string{"+", "-", "*"}
	shapes := make([]arithmeticShape, 0, 2*len(operators)*len(operators))
	for outerIndex, outer := range operators {
		for innerIndex, inner := range operators {
			for side := range 2 {
				shapes = append(shapes, arithmeticShape{id: fmt.Sprintf("expr-%d%d%d", outerIndex, innerIndex, side), outer: outer, inner: inner, left: side == 0})
			}
		}
	}
	return shapes
}

// addArithmeticMatrix adds source and expected observations for the fixed arithmetic
// matrix.
//
// Takes c (*casefile.Case) which receives generated sources and expected results.
//
// Returns error which reports an operator outside the arithmetic vocabulary, or nil on
// success.
func addArithmeticMatrix(c *casefile.Case) error {
	shapes := arithmeticShapes()
	var fields, values strings.Builder
	for _, shape := range shapes {
		fmt.Fprintf(&fields, `<output data-fuzz-output="%s" :data-value="r.Label">{{ %s }}</output>`, shape.id, shape.expression())
	}

	operands := []int{-2, 0, arithmeticHighOperand}
	probe := programProbe("/arithmetic")
	for _, first := range operands {
		for _, second := range operands {
			for _, third := range operands {
				label := fmt.Sprintf("%d/%d/%d", first, second, third)
				fmt.Fprintf(&values, "{A:%d,B:%d,C:%d,Label:%q},", first, second, third, label)
				outputs, err := shapeOutputs(shapes, label, first, second, third)
				if err != nil {
					return err
				}
				probe.Output = append(probe.Output, outputs...)
			}
		}
	}
	c.Files["pages/arithmetic.pk"] = fmt.Sprintf(arithmeticTemplate, fields.String(), values.String())
	c.Probes = append(c.Probes, probe)
	c.Features = append(c.Features, "server-arithmetic-matrix")
	return nil
}

// shapeOutputs evaluates every arithmetic shape on one operand triple.
//
// Takes shapes ([]arithmeticShape) which lists the arithmetic shapes in order.
// Takes label (string) which identifies the operand triple.
// Takes first (int) which is the first arithmetic operand.
// Takes second (int) which is the second arithmetic operand.
// Takes third (int) which is the third arithmetic operand.
//
// Returns []casefile.ProgramOutput which holds one expected output per shape.
// Returns error which reports an operator outside the arithmetic vocabulary, or nil on
// success.
func shapeOutputs(shapes []arithmeticShape, label string, first, second, third int) ([]casefile.ProgramOutput, error) {
	outputs := make([]casefile.ProgramOutput, 0, len(shapes))
	for _, shape := range shapes {
		result, err := shape.result(first, second, third)
		if err != nil {
			return nil, fmt.Errorf("evaluating %s at %s: %w", shape.id, label, err)
		}
		outputs = append(outputs, valuedOutput(shape.id, strconv.Itoa(result), label))
	}
	return outputs, nil
}

// apply evaluates one supported integer operator.
//
// Takes operator (string) which selects the integer arithmetic operation.
// Takes a (int) which is the left arithmetic operand.
// Takes b (int) which is the right arithmetic operand.
//
// Returns int which is the selected arithmetic result.
// Returns error which reports an operator outside the arithmetic vocabulary, or nil on
// success.
func apply(operator string, a, b int) (int, error) {
	switch operator {
	case "+":
		return a + b, nil
	case "-":
		return a - b, nil
	case "*":
		return a * b, nil
	default:
		return 0, fmt.Errorf("%w: unknown arithmetic operator %q", errMalformedProgram, operator)
	}
}
