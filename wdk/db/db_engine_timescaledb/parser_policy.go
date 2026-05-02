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

// Policy and management calls produce a MutationAlterTableAlterColumn
// mutation tagged with TIMESCALE_POLICY_OP=<canonical-op>. The choice
// of mutation kind matches parser_hypertable.go's annotation form for
// SELECT create_hypertable: each call adjusts an existing relation's
// metadata rather than creating new schema state. Reusing
// MutationAlterTableAlterColumn keeps the catalogue dispatch table
// uniform (the alter handler already accepts EngineSpecific-only
// updates) and avoids introducing a new MutationKind that downstream
// consumers would have to handle in lockstep. The TIMESCALE_POLICY_OP
// tag carries the operation name so consumers can route on intent
// without re-classifying the kind themselves.

package db_engine_timescaledb

import (
	"fmt"

	"piko.sh/piko/internal/querier/querier_dto"
	"piko.sh/piko/wdk/db/db_engine_postgres"
)

var (
	// policyOperationNames maps each policy/management statement kind to the canonical
	// TimescaleDB function name it represents. Used by parsePolicyCall to populate
	// TIMESCALE_POLICY_OP and by the per-kind dispatch tables so downstream consumers can
	// route on intent without re-classifying the kind themselves.
	policyOperationNames = map[db_engine_postgres.StatementKind]string{
		statementKindAddCompressionPolicy:            "add_compression_policy",
		statementKindRemoveCompressionPolicy:         "remove_compression_policy",
		statementKindAddColumnstorePolicy:            "add_columnstore_policy",
		statementKindRemoveColumnstorePolicy:         "remove_columnstore_policy",
		statementKindAddRetentionPolicy:              "add_retention_policy",
		statementKindRemoveRetentionPolicy:           "remove_retention_policy",
		statementKindAddContinuousAggregatePolicy:    "add_continuous_aggregate_policy",
		statementKindRemoveContinuousAggregatePolicy: "remove_continuous_aggregate_policy",
		statementKindAddReorderPolicy:                "add_reorder_policy",
		statementKindRemoveReorderPolicy:             "remove_reorder_policy",
		statementKindAddJob:                          "add_job",
		statementKindAlterJob:                        "alter_job",
		statementKindDeleteJob:                       "delete_job",
		statementKindRunJob:                          "run_job",
		statementKindAddDimension:                    "add_dimension",
		statementKindSetChunkTimeInterval:            "set_chunk_time_interval",
		statementKindSetIntegerNowFunc:               "set_integer_now_func",
		statementKindEnableChunkSkipping:             "enable_chunk_skipping",
		statementKindDisableChunkSkipping:            "disable_chunk_skipping",
	}

	// jobOperationKinds names the kinds whose first positional argument is a job_id.
	//
	// These kinds take a job_id rather than a table reference. add_job is the registration
	// form which takes a procedure name first, while the others take an integer job_id.
	// Keeping the membership test as a small map avoids a long case list at the call site.
	jobOperationKinds = map[db_engine_postgres.StatementKind]bool{
		statementKindAlterJob:  true,
		statementKindDeleteJob: true,
		statementKindRunJob:    true,
	}
)

// parsePolicyCall parses the shared shape of a SELECT-form policy or management call.
//
// The call shape is:
//
//	SELECT <function>(arg1 [, arg2] [, named => value] ...)
//
// The first positional argument is the target hypertable for most shapes, while the job
// operations (alter_job, delete_job, run_job) take a job_id instead, which is exposed as
// TIMESCALE_JOB_ID. Subsequent arguments are captured opaquely as TIMESCALE_POLICY_ARGS
// so downstream consumers can replay the original call without needing to re-tokenise.
//
// The mutation is MutationAlterTableAlterColumn so the catalogue does not refuse the call
// as a duplicate CREATE TABLE, since the targeted relation was created earlier in the
// migration. TIMESCALE_POLICY_OP carries the canonical operation name extracted from
// policyOperationNames.
//
// The work is split across small helpers. parsePolicyCall consumes SELECT, the operation
// keyword and the opening paren, dispatches to parsePolicyArguments for the body, then
// flushes the trailing tokens. parsePolicyArguments drives the first-argument extraction,
// applies it to the mutation, and either closes the call or delegates the tail capture.
// extractFirstPolicyArgument reads the single leading token (string, identifier, or
// number). applyFirstPolicyArgument routes the first argument into TIMESCALE_TABLE,
// TIMESCALE_JOB_ID or TIMESCALE_JOB_PROC. applyAddDimensionSecondArgument decodes the
// second argument for add_dimension calls (legacy column or by_range/by_hash).
// captureRemainingCallArguments gathers the opaque tail into TIMESCALE_POLICY_ARGS. This
// split keeps each step under the per-function size guideline and lets each helper test
// its own argument shape in isolation rather than re-exercising the entire chain.
//
// Takes p (db_engine_postgres.ParserContext) which drives the parse.
// Takes kind (db_engine_postgres.StatementKind) which selects the canonical operation
// name and disambiguates the first argument shape.
//
// Returns *querier_dto.CatalogueMutation describing the call.
// Returns error on parse failure.
func parsePolicyCall(p db_engine_postgres.ParserContext, kind db_engine_postgres.StatementKind) (*querier_dto.CatalogueMutation, error) {
	operation := policyOperationNames[kind]
	if operation == "" {
		return nil, fmt.Errorf("timescaledb: policy kind %d has no registered operation name", kind)
	}

	p.MustKeyword("SELECT")
	if !p.MatchKeyword(operation) {
		return nil, fmt.Errorf("timescaledb: expected %s at position %d", operation, p.CurrentToken().Position())
	}
	if p.CurrentToken().Kind() != db_engine_postgres.TokenLeftParen {
		return nil, fmt.Errorf("timescaledb: expected '(' after %s at position %d", operation, p.CurrentToken().Position())
	}
	openParenPosition := p.CurrentToken().Position()
	p.Advance()

	mutation, parseErr := parsePolicyArguments(p, kind, operation, openParenPosition)
	if parseErr != nil {
		return nil, parseErr
	}

	p.ConsumeRemainder()
	return mutation, nil
}

// parsePolicyArguments captures the first positional argument and opaque tail of a
// policy/management call into the returned mutation. add_job has a procedure name first;
// the *_job operations take a job_id; add_dimension has the table reference followed by
// an optional dimension builder; everything else takes a hypertable reference.
//
// Takes p (db_engine_postgres.ParserContext) which is positioned just inside the opening
// paren.
// Takes kind (db_engine_postgres.StatementKind) which classifies the expected
// first-argument shape.
// Takes operation (string) which is the canonical function name used for error wrapping.
// Takes openParenPosition (int) which is the byte offset of the `(` used to attribute
// unterminated-call diagnostics.
//
// Returns *querier_dto.CatalogueMutation with EngineSpecific populated.
// Returns error on parse failure.
func parsePolicyArguments(
	p db_engine_postgres.ParserContext,
	kind db_engine_postgres.StatementKind,
	operation string,
	openParenPosition int,
) (*querier_dto.CatalogueMutation, error) {
	mutation := &querier_dto.CatalogueMutation{
		Kind: querier_dto.MutationAlterTableAlterColumn,
		EngineSpecific: map[string]string{
			"TIMESCALE_POLICY_OP": operation,
		},
	}

	firstArg, firstErr := extractFirstPolicyArgument(p)
	if firstErr != nil {
		return nil, fmt.Errorf("%s: first argument: %w", operation, firstErr)
	}
	applyFirstPolicyArgument(mutation, kind, firstArg)

	if !consumeArgumentSeparator(p) {
		if closeErr := expectCallClose(p, operation, openParenPosition); closeErr != nil {
			return nil, closeErr
		}
		return mutation, nil
	}

	if kind == statementKindAddDimension {
		if dimensionErr := applyAddDimensionSecondArgument(p, mutation, operation); dimensionErr != nil {
			return nil, dimensionErr
		}
		if !consumeArgumentSeparator(p) {
			if closeErr := expectCallClose(p, operation, openParenPosition); closeErr != nil {
				return nil, closeErr
			}
			return mutation, nil
		}
	}

	extras, extrasErr := captureRemainingCallArguments(p, openParenPosition)
	if extrasErr != nil {
		return nil, fmt.Errorf("%s: %w", operation, extrasErr)
	}
	if extras != "" {
		mutation.EngineSpecific["TIMESCALE_POLICY_ARGS"] = extras
	}
	return mutation, nil
}

// applyAddDimensionSecondArgument decodes the second argument of an add_dimension call.
//
// The argument is either a literal column name (legacy positional form) or a
// "by_range(...)" or "by_hash(...)" builder. Both shapes set TIMESCALE_TIME_COLUMN, while
// the builder form additionally records TIMESCALE_DIMENSION_BUILDER so downstream
// consumers can distinguish range partitioning from hash partitioning.
//
// Takes p (db_engine_postgres.ParserContext) which is positioned at the second argument.
// Takes mutation (*querier_dto.CatalogueMutation) which receives the extracted keys.
// Takes operation (string) which is the canonical function name used for error wrapping.
//
// Returns error on a malformed second argument.
func applyAddDimensionSecondArgument(
	p db_engine_postgres.ParserContext,
	mutation *querier_dto.CatalogueMutation,
	operation string,
) error {
	dimension, dimensionErr := extractDimensionArgument(p)
	if dimensionErr != nil {
		return fmt.Errorf("%s: dimension argument: %w", operation, dimensionErr)
	}
	mutation.EngineSpecific["TIMESCALE_TIME_COLUMN"] = dimension.column
	if dimension.builder != "" {
		mutation.EngineSpecific["TIMESCALE_DIMENSION_BUILDER"] = dimension.builder
	}
	return nil
}

// expectCallClose consumes the closing `)` of a policy call when the argument list ended
// without a trailing comma. Reports a clear error referencing openParenPosition when the
// close is missing so callers can locate the unterminated call in the source.
//
// Takes p (db_engine_postgres.ParserContext) which is positioned where the close paren is
// expected.
// Takes operation (string) which is the canonical function name for the diagnostic.
// Takes openParenPosition (int) which is the byte offset of the opening paren.
//
// Returns error when the close paren is missing.
func expectCallClose(p db_engine_postgres.ParserContext, operation string, openParenPosition int) error {
	if p.CurrentToken().Kind() != db_engine_postgres.TokenRightParen {
		return fmt.Errorf("%s: unterminated call opened at position %d", operation, openParenPosition)
	}
	p.Advance()
	return nil
}

// applyFirstPolicyArgument stores the first argument in the appropriate EngineSpecific
// key for the kind: TIMESCALE_JOB_ID for the bare-job operations, TIMESCALE_TABLE for
// everything else. The mutation's SchemaName and TableName are populated for non-job
// kinds so catalogue lookups continue to resolve the target relation.
//
// Takes mutation (*querier_dto.CatalogueMutation) which receives the metadata.
// Takes kind (db_engine_postgres.StatementKind) which selects the key.
// Takes arg (string) which is the captured first-argument text.
func applyFirstPolicyArgument(
	mutation *querier_dto.CatalogueMutation,
	kind db_engine_postgres.StatementKind,
	arg string,
) {
	if jobOperationKinds[kind] {
		mutation.EngineSpecific["TIMESCALE_JOB_ID"] = arg
		return
	}
	if kind == statementKindAddJob {
		mutation.EngineSpecific["TIMESCALE_JOB_PROC"] = arg
		return
	}
	mutation.EngineSpecific["TIMESCALE_TABLE"] = arg
	schema, table := splitMaybeSchemaQualified(arg)
	mutation.SchemaName = schema
	mutation.TableName = table
}

// extractFirstPolicyArgument reads a single string literal, bare identifier or numeric
// literal argument.
//
// The token is consumed and its text returned. Anything else (an open paren, an operator,
// or EOF) is rejected so a malformed call surfaces with a clear error.
//
// Only the single token at the cursor is captured, so a schema-qualified job procedure
// passed as a bare identifier triple ("schema . proc") is not assembled into a dotted
// name here. Callers that need schema-qualified routing rely on string-literal forms
// ("'schema.proc'") which the tokeniser emits as a single TokenString, and the downstream
// splitMaybeSchemaQualified call in applyFirstPolicyArgument then splits the dot for the
// hypertable branch. The job-proc branch stores the captured text verbatim in
// TIMESCALE_JOB_PROC and does not split, because TimescaleDB stores the procedure
// reference as a single name in its job-registry catalogue.
//
// Takes p (db_engine_postgres.ParserContext) which is positioned at the candidate first
// token.
//
// Returns string which is the captured text.
// Returns error when the cursor token is not a recognised argument shape.
func extractFirstPolicyArgument(p db_engine_postgres.ParserContext) (string, error) {
	tok := p.CurrentToken()
	switch tok.Kind() {
	case db_engine_postgres.TokenString, db_engine_postgres.TokenIdentifier, db_engine_postgres.TokenNumber:
		p.Advance()

		if castErr := consumeOptionalCast(p); castErr != nil {
			return "", castErr
		}
		return tok.Value(), nil
	}
	return "", fmt.Errorf("expected literal or identifier at position %d", tok.Position())
}
