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

package querier_domain

import (
	"fmt"
	"regexp"

	"piko.sh/piko/internal/querier/querier_dto"
)

var (
	// conflictDoNothingPattern matches an ON CONFLICT ... DO NOTHING action across any
	// interior whitespace and newlines.
	conflictDoNothingPattern = regexp.MustCompile(`(?is)\bON\s+CONFLICT\b.*?\bDO\s+NOTHING\b`)
)

// commandOutputPass validates that the query command is consistent with the output
// columns. SELECT-style commands (one, many, stream) must produce columns; exec-style
// commands must not.
type commandOutputPass struct{}

// Analyse checks command and output column consistency.
//
// Takes context (*diagnosticContext) which holds the query and analysis state.
//
// Returns []querier_dto.SourceError which holds any diagnostics for command/output
// mismatches.
func (*commandOutputPass) Analyse(context *diagnosticContext) []querier_dto.SourceError {
	var diagnostics []querier_dto.SourceError

	switch context.Query.Command { //nolint:exhaustive // exhaustive case-set intentionally partial; missing entries are no-ops
	case querier_dto.QueryCommandOne, querier_dto.QueryCommandMany, querier_dto.QueryCommandStream:
		if len(context.Query.OutputColumns) == 0 {
			diagnostics = append(diagnostics, querier_dto.SourceError{
				Filename: context.Filename,
				Line:     context.Query.Line,
				Column:   1,
				Message: fmt.Sprintf(
					"query %q uses command %q but produces no output columns",
					context.Query.Name, commandName(context.Query.Command),
				),
				Severity: querier_dto.SeverityWarning,
				Code:     querier_dto.CodeCommandOutputMismatch,
			})
		}

	case querier_dto.QueryCommandExec, querier_dto.QueryCommandExecResult, querier_dto.QueryCommandExecRows,
		querier_dto.QueryCommandAsyncExec:
		if len(context.Query.OutputColumns) > 0 {
			diagnostics = append(diagnostics, querier_dto.SourceError{
				Filename: context.Filename,
				Line:     context.Query.Line,
				Column:   1,
				Message: fmt.Sprintf(
					"query %q uses command %q but produces %d output columns",
					context.Query.Name, commandName(context.Query.Command), len(context.Query.OutputColumns),
				),
				Severity: querier_dto.SeverityWarning,
				Code:     querier_dto.CodeCommandOutputMismatch,
			})
		}
	}

	if diagnostic := conflictDoNothingReturningDiagnostic(context); diagnostic != nil {
		diagnostics = append(diagnostics, *diagnostic)
	}
	if diagnostic := optionalMisuseDiagnostic(context); diagnostic != nil {
		diagnostics = append(diagnostics, *diagnostic)
	}

	return diagnostics
}

// conflictDoNothingReturningDiagnostic warns (Q046) when a command:one query pairs ON
// CONFLICT DO NOTHING with RETURNING, because a conflict skips the insert and the scan
// returns the driver no-rows sentinel. It returns nil when optional: true handles the
// zero row, or when the pattern does not match.
//
// Takes context (*diagnosticContext) which holds the query and raw analysis.
//
// Returns *querier_dto.SourceError which is the warning, or nil when it does not apply.
func conflictDoNothingReturningDiagnostic(context *diagnosticContext) *querier_dto.SourceError {
	if context.Query.Command != querier_dto.QueryCommandOne || context.Query.Optional ||
		context.RawAnalysis == nil || !context.RawAnalysis.HasReturning ||
		!conflictDoNothingPattern.MatchString(context.Query.SQL) {
		return nil
	}

	return &querier_dto.SourceError{
		Filename: context.Filename,
		Line:     context.Query.Line,
		Column:   1,
		Message: fmt.Sprintf(
			"query %q uses command %q with ON CONFLICT DO NOTHING and RETURNING; a conflict skips the "+
				"insert so RETURNING yields no row and the call returns the driver no-rows sentinel "+
				"(sql.ErrNoRows, or pgx.ErrNoRows for native pgx). Add optional: true to return "+
				"(row, false, nil), use command \"exec\"/\"execrows\", or handle errors.Is with that sentinel.",
			context.Query.Name, commandName(context.Query.Command),
		),
		Severity: querier_dto.SeverityWarning,
		Code:     querier_dto.CodeConflictDoNothingReturning,
	}
}

// optionalMisuseDiagnostic reports (Q047, error) when optional: true is misplaced. It
// returns nil when optional is unset or valid.
//
// Takes context (*diagnosticContext) which holds the query.
//
// Returns *querier_dto.SourceError which is the error, or nil when it does not apply.
func optionalMisuseDiagnostic(context *diagnosticContext) *querier_dto.SourceError {
	reason := optionalMisuseReason(context.Query)
	if reason == "" {
		return nil
	}

	return &querier_dto.SourceError{
		Filename: context.Filename,
		Line:     context.Query.Line,
		Column:   1,
		Message: fmt.Sprintf(
			"query %q sets optional: true but %s. Remove optional or use a static command:one.",
			context.Query.Name, reason,
		),
		Severity: querier_dto.SeverityError,
		Code:     querier_dto.CodeOptionalNonOneCommand,
	}
}

// optionalMisuseReason returns a reason when optional: true is set where the flag has no
// meaning, or an empty string when optional is unset or valid. The flag applies only to a
// static command:one, where a zero row becomes (row, false, nil); any other command, or a
// dynamic runtime builder or dynamic predicate parameters, is rejected.
//
// Takes query (*querier_dto.AnalysedQuery) whose command and dynamic flags are inspected.
//
// Returns string which is the misuse reason, or empty when optional is unset or valid.
func optionalMisuseReason(query *querier_dto.AnalysedQuery) string {
	if !query.Optional {
		return ""
	}

	switch {
	case query.Command != querier_dto.QueryCommandOne:
		return fmt.Sprintf("optional applies only to command:one, not command %q", commandName(query.Command))
	case query.DynamicRuntime:
		return "optional is not supported with dynamic: runtime; the builder's One() terminal already " +
			"returns the no-rows sentinel"
	case query.IsDynamic:
		return "optional is not supported on a query with optional predicate parameters"
	default:
		return ""
	}
}
