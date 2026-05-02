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

package db_engine_clickhouse

import (
	"fmt"

	"piko.sh/piko/internal/querier/querier_dto"
)

// parseCreateFunction handles `CREATE FUNCTION [IF NOT EXISTS] name AS lambda`.
//
// ClickHouse user-defined functions are lambda expressions over scalar arguments; the
// lambda body is structurally parsed into a querier_dto.Expression so the shared
// function-body analyser can infer the function's ReturnType.
//
// The signature's ReturnType is initialised to TypeCategoryUnknown so the analyser can
// detect that no engine-specific type has been declared and fold in its inferred type. A
// fallback handles inputs without `AS` so such migrations still parse, in which case the
// function is registered with no BodyExpression and downstream consumers see ReturnType
// of TypeCategoryUnknown.
//
// Returns *querier_dto.CatalogueMutation which describes the create-function mutation.
// Returns error when the identifier or lambda body fails to parse.
func (p *parser) parseCreateFunction() (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword(keywordCreate)
	p.skipCreatePrefixesInParser()
	p.mustKeyword("FUNCTION")
	p.matchIfNotExists()

	name, err := p.parseIdentifierOrKeyword()
	if err != nil {
		return nil, err
	}
	_ = p.matchOnCluster()

	if !p.matchKeyword("AS") {
		p.consumeRemainder()
		return &querier_dto.CatalogueMutation{
			Kind: querier_dto.MutationCreateFunction,
			FunctionSignature: &querier_dto.FunctionSignature{
				Name:       name,
				ReturnType: querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown},
			},
		}, nil
	}

	body, parameters, lambdaErr := p.parseCreateFunctionLambdaBody()
	if lambdaErr != nil {
		return nil, fmt.Errorf("create function %q body: %w", name, lambdaErr)
	}
	p.consumeRemainder()

	return &querier_dto.CatalogueMutation{
		Kind: querier_dto.MutationCreateFunction,
		FunctionSignature: &querier_dto.FunctionSignature{
			Name:           name,
			BodyExpression: body,
			BodyParameters: parameters,
			ReturnType:     querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown},
		},
	}, nil
}

// parseDropFunction handles `DROP FUNCTION [IF EXISTS] name`.
//
// Returns the catalogue mutation that captures the function name so the catalogue builder
// can detach the matching signature, or an error when the identifier is missing.
func (p *parser) parseDropFunction() (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword(keywordDrop)
	p.mustKeyword("FUNCTION")
	p.matchIfExists()
	name, err := p.parseIdentifierOrKeyword()
	if err != nil {
		return nil, err
	}
	_ = p.matchOnCluster()
	return &querier_dto.CatalogueMutation{
		Kind: querier_dto.MutationDropFunction,
		FunctionSignature: &querier_dto.FunctionSignature{
			Name: name,
		},
	}, nil
}

// parseCreateFunctionLambdaBody parses the lambda body of a `CREATE FUNCTION name AS
// lambda` statement and produces a structured querier_dto.Expression suitable for
// downstream catalogue-builder analysis. The cursor must be positioned at the first token
// of the lambda (either the single parameter identifier or the opening paren of a
// parameter tuple).
//
// On success it returns the body expression and the parameter list in lexical order. On
// failure it returns a non-nil error which the sole caller wraps and returns, so callers
// must check err before inspecting the results.
//
// Returns querier_dto.Expression which is the body expression tree.
// Returns []string which is the lexical parameter list, possibly empty for `() -> body`.
// Returns error when the input is malformed, such as a missing arrow or no body.
func (p *parser) parseCreateFunctionLambdaBody() (querier_dto.Expression, []string, error) {
	parameters, paramErr := p.consumeLambdaParameterList()
	if paramErr != nil {
		return nil, nil, paramErr
	}
	if p.current().kind != tokenArrow {
		return nil, nil, fmt.Errorf("expected '->' at position %d", p.current().position)
	}
	p.advance()
	body, bodyErr := p.parseLambdaBodyExpression()
	if bodyErr != nil {
		return nil, nil, bodyErr
	}
	return body, parameters, nil
}

// consumeLambdaParameterList consumes the parameter list that introduces a lambda.
//
// It accepts a bare identifier (`x -> body`) or a parenthesised tuple (`(a, b) -> body`);
// the empty tuple `()` yields a nil parameter slice. An error is returned when the head
// token is neither an identifier nor a left paren.
//
// Returns []string which holds the parameter names in declaration order.
// Returns error when the parameter list cannot be parsed.
func (p *parser) consumeLambdaParameterList() ([]string, error) {
	if p.current().kind == tokenIdentifier {
		name := p.current().value
		p.advance()
		return []string{name}, nil
	}
	if p.current().kind != tokenLeftParen {
		return nil, fmt.Errorf("expected lambda parameter list at position %d", p.current().position)
	}
	p.advance()
	var parameters []string
	for p.current().kind != tokenRightParen && !p.atEnd() {
		if p.current().kind != tokenIdentifier {
			return nil, fmt.Errorf("expected parameter name at position %d", p.current().position)
		}
		parameters = append(parameters, p.current().value)
		p.advance()
		if p.current().kind == tokenComma {
			p.advance()
			continue
		}
		break
	}
	if p.current().kind != tokenRightParen {
		return nil, fmt.Errorf("expected ')' at position %d", p.current().position)
	}
	p.advance()
	return parameters, nil
}

// parseLambdaBodyExpression parses the body expression of a CREATE FUNCTION lambda into a
// querier_dto.Expression tree.
//
// Supported shapes cover numeric and string literals, bare identifier column references
// where parameters resolve through the function-body scope chain when the downstream
// analyser runs, function calls `name(arg, ...)` and parenthesised subexpressions, and
// the binary operators `+`, `-`, `*`, `/`, `%` and `||` evaluated left-to-right because
// lambda-body type inference does not depend on precedence (the type system uses
// commonSupertype).
//
// More complex expressions fall back to UnknownExpression so the catalogue still records
// the function with a degraded ReturnType.
//
// Returns querier_dto.Expression which is the parsed body tree.
// Returns error when the cursor encounters an unexpected end of input mid-expression.
//
// The expressionDepth counter is incremented on entry and decremented on exit so a deeply
// nested lambda body cannot drive the parser past maxParseDepth recursive frames; every
// recursive route (parenthesised subexpressions and function-call arguments) loops back
// here, so guarding entry alone bounds the whole lambda-body grammar. When the cap is
// reached errAnalysisDepthExceeded is returned so the caller folds the function in with a
// degraded ReturnType.
func (p *parser) parseLambdaBodyExpression() (querier_dto.Expression, error) {
	if p.expressionDepth >= p.maxParseDepth {
		return nil, errAnalysisDepthExceeded
	}
	p.expressionDepth++
	defer func() { p.expressionDepth-- }()

	left, leftErr := p.parseLambdaBodyTerm()
	if leftErr != nil {
		return nil, leftErr
	}
	for {
		tok := p.current()
		if tok.kind != tokenOperator || !isLambdaBodyBinaryOperator(tok.value) {
			return left, nil
		}
		operator := tok.value
		p.advance()
		right, rightErr := p.parseLambdaBodyTerm()
		if rightErr != nil {
			return nil, rightErr
		}
		left = &querier_dto.BinaryOpExpression{
			Operator: operator,
			Left:     left,
			Right:    right,
		}
	}
}

// parseLambdaBodyTerm parses a single primary term of a lambda body: a literal,
// identifier, function call, or parenthesised subexpression. Identifiers followed by `(`
// form function calls; other identifiers become column references that the downstream
// analyser resolves against the parameter scope.
//
// Returns querier_dto.Expression which holds the parsed term.
// Returns error when no valid term token is present.
func (p *parser) parseLambdaBodyTerm() (querier_dto.Expression, error) {
	tok := p.current()
	switch tok.kind {
	case tokenNumber:
		p.advance()
		return literalExpressionFromNumber(tok.value), nil
	case tokenString:
		p.advance()
		return &querier_dto.LiteralExpression{TypeName: "String"}, nil
	case tokenLeftParen:
		p.advance()
		inner, innerErr := p.parseLambdaBodyExpression()
		if innerErr != nil {
			return nil, innerErr
		}
		if p.current().kind != tokenRightParen {
			return nil, fmt.Errorf("expected ')' to close CREATE FUNCTION lambda subexpression at position %d", p.current().position)
		}
		p.advance()
		return inner, nil
	case tokenIdentifier:
		name := tok.value
		p.advance()
		if p.current().kind == tokenLeftParen {
			return p.parseLambdaBodyFunctionCall(name)
		}
		return &querier_dto.ColumnRefExpression{ColumnName: name}, nil
	default:
		if p.atEnd() {
			return nil, fmt.Errorf("unexpected end of input in CREATE FUNCTION lambda body at position %d", tok.position)
		}
		return &querier_dto.UnknownExpression{}, nil
	}
}

// parseLambdaBodyFunctionCall parses the argument list of a function call inside a lambda
// body. The function name has been consumed; the cursor is on the opening paren.
//
// Takes name (string) which is the function name.
//
// Returns querier_dto.Expression which is a FunctionCallExpression carrying the parsed
// arguments.
// Returns error when an argument cannot be parsed.
func (p *parser) parseLambdaBodyFunctionCall(name string) (querier_dto.Expression, error) {
	p.advance()
	var arguments []querier_dto.Expression
	for p.current().kind != tokenRightParen && !p.atEnd() {
		argument, argErr := p.parseLambdaBodyExpression()
		if argErr != nil {
			return nil, argErr
		}
		arguments = append(arguments, argument)
		if p.current().kind == tokenComma {
			p.advance()
			continue
		}
		break
	}
	if p.current().kind == tokenRightParen {
		p.advance()
	}
	return &querier_dto.FunctionCallExpression{
		FunctionName: name,
		Arguments:    arguments,
	}, nil
}

// literalExpressionFromNumber builds a LiteralExpression whose TypeName captures the
// inferred numeric type so the resolver can normalise it through the engine's type
// system.
//
// Takes value (string) which is the raw textual literal.
//
// Returns *querier_dto.LiteralExpression which wraps the type name.
func literalExpressionFromNumber(value string) *querier_dto.LiteralExpression {
	sqlType := literalNumberType(value)
	return &querier_dto.LiteralExpression{TypeName: sqlType.EngineName}
}

// isLambdaBodyBinaryOperator reports whether the supplied operator token belongs to the
// small arithmetic / concat set the lambda body parser handles. Comparison and logical
// operators are out of scope because lambda bodies in CREATE FUNCTION are expected to
// produce values rather than predicates.
//
// Takes operator (string) which is the operator symbol from the current token.
//
// Returns bool which is true when the parser should consume the operator and an
// additional term.
func isLambdaBodyBinaryOperator(operator string) bool {
	switch operator {
	case "+", "-", "*", "/", "%", "||":
		return true
	default:
		return false
	}
}
