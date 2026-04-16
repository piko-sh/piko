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

package layouter_domain

import (
	"math"
	"strconv"
	"strings"
)

const (
	// percentageDivisor is the divisor for converting
	// percentage values to a 0-1 scale.
	percentageDivisor = 100.0

	// calcPrefixLength is the byte length of the "calc("
	// prefix string.
	calcPrefixLength = len("calc(")
)

// CalcNodeType identifies the kind of node in a calc expression AST.
type CalcNodeType int

const (
	// CalcNodeNumber is a plain number with no unit.
	CalcNodeNumber CalcNodeType = iota

	// CalcNodeLength is a number with a CSS length unit (px, pt, em, etc.).
	CalcNodeLength

	// CalcNodePercentage is a percentage value.
	CalcNodePercentage

	// CalcNodeAdd is an addition operation.
	CalcNodeAdd

	// CalcNodeSubtract is a subtraction operation.
	CalcNodeSubtract

	// CalcNodeMultiply is a multiplication operation.
	CalcNodeMultiply

	// CalcNodeDivide is a division operation.
	CalcNodeDivide
)

// calcExpression represents a node in a parsed calc() expression tree.
type calcExpression struct {
	// Left is the left operand for binary operator nodes.
	Left *calcExpression

	// Right is the right operand for binary operator nodes.
	Right *calcExpression

	// Unit is the CSS length unit (e.g. "px", "em", "rem").
	Unit string

	// Value is the numeric value of this leaf node.
	Value float64

	// Type is the kind of node in the expression tree.
	Type CalcNodeType
}

// resolveCalc evaluates the calc expression tree recursively.
//
// Takes context (ResolutionContext) which provides font
// sizes and viewport dimensions for unit resolution.
// Takes containingBlockSize (float64) which is the size
// of the containing block for resolving percentages.
//
// Returns the computed result in points.
func (expression *calcExpression) resolveCalc(context ResolutionContext, containingBlockSize float64) float64 {
	if expression == nil {
		return 0
	}

	switch expression.Type {
	case CalcNodeNumber:
		return expression.Value
	case CalcNodePercentage:
		return expression.Value / percentageDivisor * containingBlockSize
	case CalcNodeLength:
		return resolveCalcLength(expression.Value, expression.Unit, context)
	case CalcNodeAdd:
		return expression.Left.resolveCalc(context, containingBlockSize) +
			expression.Right.resolveCalc(context, containingBlockSize)
	case CalcNodeSubtract:
		return expression.Left.resolveCalc(context, containingBlockSize) -
			expression.Right.resolveCalc(context, containingBlockSize)
	case CalcNodeMultiply:
		return expression.Left.resolveCalc(context, containingBlockSize) *
			expression.Right.resolveCalc(context, containingBlockSize)
	case CalcNodeDivide:
		divisor := expression.Right.resolveCalc(context, containingBlockSize)
		if divisor == 0 {
			return 0
		}
		return expression.Left.resolveCalc(context, containingBlockSize) / divisor
	default:
		return 0
	}
}

// viewportUnitResolvers maps CSS viewport unit suffixes to their resolution functions.
var viewportUnitResolvers = map[string]func(ResolutionContext) float64{
	"vw": func(context ResolutionContext) float64 {
		return context.ViewportWidth / percentageDivisor
	},
	"vh": func(context ResolutionContext) float64 {
		return context.ViewportHeight / percentageDivisor
	},
	"vmin": func(context ResolutionContext) float64 {
		return math.Min(context.ViewportWidth, context.ViewportHeight) / percentageDivisor
	},
	"vmax": func(context ResolutionContext) float64 {
		return math.Max(context.ViewportWidth, context.ViewportHeight) / percentageDivisor
	},
}

// resolveCalcLength converts a numeric value with a CSS unit
// to points using the given resolution context.
//
// Takes value (float64) which is the numeric magnitude.
// Takes unit (string) which is the CSS unit suffix
// (e.g. "px", "em", "rem").
// Takes context (ResolutionContext) which provides font
// sizes and viewport dimensions for unit conversion.
//
// Returns float64 which is the resolved length in points.
func resolveCalcLength(value float64, unit string, context ResolutionContext) float64 {
	switch unit {
	case "px":
		return value * PixelsToPoints
	case "em":
		return value * context.ParentFontSize
	case "rem":
		return value * context.RootFontSize
	case "cm":
		return value * CentimetresToPoints
	case "mm":
		return value * MillimetresToPoints
	case "in":
		return value * InchesToPoints
	case "pc":
		return value * PicasToPoints
	default:
		if resolver, ok := viewportUnitResolvers[unit]; ok {
			return value * resolver(context)
		}
		return value
	}
}

// parseCalc parses a CSS calc() expression into an AST.
//
// Takes expression (string) which is the inner content
// of a calc() expression without the surrounding
// "calc(" and ")".
//
// Returns the parsed expression tree, or nil if parsing
// fails.
func parseCalc(expression string) *calcExpression {
	expression = strings.TrimSpace(expression)
	if expression == "" {
		return nil
	}

	parser := &calcParser{input: expression, position: 0}
	result := parser.parseAdditive()
	if result == nil || parser.position < len(parser.input) {
		remaining := strings.TrimSpace(parser.input[parser.position:])
		if remaining != "" {
			return nil
		}
	}
	return result
}

// calcParser holds the state for recursive-descent parsing
// of a calc() expression string.
type calcParser struct {
	// input is the expression string being parsed.
	input string

	// position is the current byte offset in the input.
	position int
}

// skipWhitespace advances past any space characters at the
// current position.
func (parser *calcParser) skipWhitespace() {
	for parser.position < len(parser.input) && parser.input[parser.position] == ' ' {
		parser.position++
	}
}

// parseAdditiveOperator attempts to consume a '+' or '-'
// operator surrounded by spaces.
//
// Returns CalcNodeType which is CalcNodeAdd or
// CalcNodeSubtract.
// Returns bool which is true if an operator was consumed.
func (parser *calcParser) parseAdditiveOperator() (CalcNodeType, bool) {
	parser.skipWhitespace()
	if parser.position >= len(parser.input) {
		return 0, false
	}

	character := parser.input[parser.position]
	if character != '+' && character != '-' {
		return 0, false
	}

	if parser.position > 0 && parser.input[parser.position-1] != ' ' {
		return 0, false
	}
	if parser.position+1 < len(parser.input) && parser.input[parser.position+1] != ' ' {
		return 0, false
	}

	nodeType := CalcNodeAdd
	if character == '-' {
		nodeType = CalcNodeSubtract
	}
	parser.position++
	parser.skipWhitespace()

	return nodeType, true
}

// parseAdditive parses an additive expression consisting of
// multiplicative terms separated by '+' or '-'.
//
// Returns *calcExpression which is the parsed AST node,
// or nil on failure.
func (parser *calcParser) parseAdditive() *calcExpression {
	left := parser.parseMultiplicative()
	if left == nil {
		return nil
	}

	for parser.position < len(parser.input) {
		nodeType, ok := parser.parseAdditiveOperator()
		if !ok {
			break
		}

		right := parser.parseMultiplicative()
		if right == nil {
			return nil
		}

		left = &calcExpression{Type: nodeType, Left: left, Right: right}
	}

	return left
}

// parseMultiplicative parses a multiplicative expression
// consisting of primary terms separated by '*' or '/'.
//
// Returns *calcExpression which is the parsed AST node,
// or nil on failure.
func (parser *calcParser) parseMultiplicative() *calcExpression {
	left := parser.parsePrimary()
	if left == nil {
		return nil
	}

	for parser.position < len(parser.input) {
		parser.skipWhitespace()
		if parser.position >= len(parser.input) {
			break
		}

		character := parser.input[parser.position]
		if character != '*' && character != '/' {
			break
		}

		nodeType := CalcNodeMultiply
		if character == '/' {
			nodeType = CalcNodeDivide
		}
		parser.position++
		parser.skipWhitespace()

		right := parser.parsePrimary()
		if right == nil {
			return nil
		}

		left = &calcExpression{Type: nodeType, Left: left, Right: right}
	}

	return left
}

// parseGroupedExpression parses a parenthesised
// sub-expression and consumes the closing ')'.
//
// Returns *calcExpression which is the parsed
// sub-expression, or nil on failure.
func (parser *calcParser) parseGroupedExpression() *calcExpression {
	result := parser.parseAdditive()
	parser.skipWhitespace()
	if parser.position < len(parser.input) && parser.input[parser.position] == ')' {
		parser.position++
	}
	return result
}

// parsePrimary parses a primary expression which is either
// a grouped sub-expression or a numeric value with unit.
//
// Returns *calcExpression which is the parsed primary
// node, or nil on failure.
func (parser *calcParser) parsePrimary() *calcExpression {
	parser.skipWhitespace()
	if parser.position >= len(parser.input) {
		return nil
	}

	if parser.input[parser.position] == '(' {
		parser.position++
		return parser.parseGroupedExpression()
	}

	if strings.HasPrefix(parser.input[parser.position:], "calc(") {
		parser.position += calcPrefixLength
		return parser.parseGroupedExpression()
	}

	return parser.parseValue()
}

// consumeSign advances past a leading '+' or '-' sign if
// present at the current position.
func (parser *calcParser) consumeSign() {
	if parser.position < len(parser.input) && (parser.input[parser.position] == '-' || parser.input[parser.position] == '+') {
		parser.position++
	}
}

// consumeDigits advances past a sequence of digit characters
// and decimal points at the current position.
func (parser *calcParser) consumeDigits() {
	for parser.position < len(parser.input) && ((parser.input[parser.position] >= '0' && parser.input[parser.position] <= '9') || parser.input[parser.position] == '.') {
		parser.position++
	}
}

// consumeUnit advances past a sequence of lowercase ASCII
// letters representing a CSS unit suffix.
func (parser *calcParser) consumeUnit() {
	for parser.position < len(parser.input) && parser.input[parser.position] >= 'a' && parser.input[parser.position] <= 'z' {
		parser.position++
	}
}

// parseValue parses a numeric literal optionally followed by
// a CSS unit or percentage sign.
//
// Returns *calcExpression which is the parsed leaf node,
// or nil if no valid number is found.
func (parser *calcParser) parseValue() *calcExpression {
	start := parser.position

	parser.consumeSign()

	if parser.position >= len(parser.input) || (parser.input[parser.position] < '0' || parser.input[parser.position] > '9') && parser.input[parser.position] != '.' {
		parser.position = start
		return nil
	}

	parser.consumeDigits()

	numberString := parser.input[start:parser.position]
	number, err := strconv.ParseFloat(numberString, 64)
	if err != nil {
		parser.position = start
		return nil
	}

	unitStart := parser.position
	parser.consumeUnit()

	if parser.position < len(parser.input) && parser.input[parser.position] == '%' {
		parser.position++
		return &calcExpression{Type: CalcNodePercentage, Value: number}
	}

	unit := parser.input[unitStart:parser.position]
	if unit == "" {
		return &calcExpression{Type: CalcNodeNumber, Value: number}
	}

	return &calcExpression{Type: CalcNodeLength, Value: number, Unit: unit}
}
