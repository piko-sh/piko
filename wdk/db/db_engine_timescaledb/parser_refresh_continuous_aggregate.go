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

package db_engine_timescaledb

import (
	"fmt"

	"piko.sh/piko/internal/querier/querier_dto"
	"piko.sh/piko/wdk/db/db_engine_postgres"
)

// parseRefreshContinuousAggregateCall parses a CALL refresh_continuous_aggregate(name,
// start, finish) procedure invocation.
//
// TimescaleDB exposes the incremental refresh helper as a procedure (not a
// SELECT-returning function) because the refresh runs across multiple transactions;
// postgres CALL is the appropriate dispatch form. The mutation is a
// MutationAlterTableAlterColumn marker so the catalogue updates the existing
// continuous-aggregate relation rather than refusing the statement as a duplicate CREATE
// VIEW.
//
// The three captured arguments are stored verbatim in EngineSpecific
// (TIMESCALE_REFRESH_CONTINUOUS_AGGREGATE_TARGET, TIMESCALE_REFRESH_WINDOW_START,
// TIMESCALE_REFRESH_WINDOW_END) so downstream consumers can replay the call. The window
// bounds are kept opaque; both '2020-01-01' and NULL are common, and re-interpreting them
// would duplicate logic the migration runner already performs at execution time.
//
// Takes p (db_engine_postgres.ParserContext) which drives the parse.
//
// Returns *querier_dto.CatalogueMutation which describes the refresh call.
// Returns error when the statement fails to parse.
func parseRefreshContinuousAggregateCall(p db_engine_postgres.ParserContext) (*querier_dto.CatalogueMutation, error) {
	operation := funcNameRefreshContinuousAggregate
	p.MustKeyword("CALL")
	if !p.MatchKeyword(operation) {
		return nil, fmt.Errorf("expected %s at position %d", operation, p.CurrentToken().Position())
	}
	if p.CurrentToken().Kind() != db_engine_postgres.TokenLeftParen {
		return nil, fmt.Errorf("expected '(' after %s at position %d", operation, p.CurrentToken().Position())
	}
	openParenPosition := p.CurrentToken().Position()
	p.Advance()

	target, targetErr := extractFirstPolicyArgument(p)
	if targetErr != nil {
		return nil, fmt.Errorf("%s: target argument: %w", operation, targetErr)
	}
	mutation := buildRefreshContinuousAggregateMutation(target)

	if !consumeArgumentSeparator(p) {
		if closeErr := expectCallClose(p, operation, openParenPosition); closeErr != nil {
			return nil, closeErr
		}
		p.ConsumeRemainder()
		return mutation, nil
	}

	startArg, startErr := captureRefreshWindowBound(p)
	if startErr != nil {
		return nil, fmt.Errorf("%s: window-start argument: %w", operation, startErr)
	}
	mutation.EngineSpecific["TIMESCALE_REFRESH_WINDOW_START"] = startArg

	if consumeArgumentSeparator(p) {
		finishArg, finishErr := captureRefreshWindowBound(p)
		if finishErr != nil {
			return nil, fmt.Errorf("%s: window-end argument: %w", operation, finishErr)
		}
		mutation.EngineSpecific["TIMESCALE_REFRESH_WINDOW_END"] = finishArg
	}

	if closeErr := expectCallClose(p, operation, openParenPosition); closeErr != nil {
		return nil, closeErr
	}
	p.ConsumeRemainder()
	return mutation, nil
}

// buildRefreshContinuousAggregateMutation populates the catalogue mutation produced by a
// parsed CALL refresh_continuous_aggregate. The target argument is split into
// schema/table so a schema-qualified reference (`'analytics.hourly_temps'`) resolves
// through the catalogue without further work.
//
// Takes target (string) which is the first positional argument's text.
//
// Returns *querier_dto.CatalogueMutation with EngineSpecific populated.
func buildRefreshContinuousAggregateMutation(target string) *querier_dto.CatalogueMutation {
	schema, name := splitMaybeSchemaQualified(target)
	return &querier_dto.CatalogueMutation{
		Kind:       querier_dto.MutationAlterTableAlterColumn,
		SchemaName: schema,
		TableName:  name,
		EngineSpecific: map[string]string{
			"TIMESCALE_REFRESH_CONTINUOUS_AGGREGATE_TARGET": target,
			"TIMESCALE_POLICY_OP":                           funcNameRefreshContinuousAggregate,
		},
	}
}

// captureRefreshWindowBound reads a single window-bound argument: a literal (string or
// numeric), a bare identifier (commonly NULL), or a multi-token expression like `INTERVAL
// '1 day'` / `NOW() - INTERVAL ...`. The capture stops at the top-level comma or close
// paren of the call.
//
// Takes p (db_engine_postgres.ParserContext) which is positioned at the first token of
// the bound.
//
// Returns string which is the captured text (trimmed).
// Returns error when nested parens exceed maxParenDepth.
func captureRefreshWindowBound(p db_engine_postgres.ParserContext) (string, error) {
	return captureExpressionUntilBoundary(p)
}
