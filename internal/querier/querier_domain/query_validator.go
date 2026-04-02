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

	"piko.sh/piko/internal/querier/querier_dto"
)

// queryValidator performs cross-query validation that requires visibility
// across all analysed queries. Per-query diagnostics are handled by the
// diagnosticAnalyser and its passes.
//
// Validation codes handled here:
//   - Q006: Duplicate query name across files
type queryValidator struct{}

// newQueryValidator creates a query validator.
//
// Returns *queryValidator which is ready to validate
// queries.
func newQueryValidator() *queryValidator {
	return &queryValidator{}
}

// ValidateDuplicateNames checks for duplicate query names
// across all analysed queries.
//
// Takes queries ([]*querier_dto.AnalysedQuery) which holds
// the queries to check for duplicate names.
//
// Returns []querier_dto.SourceError which holds Q006
// diagnostics for any duplicates found.
func (*queryValidator) ValidateDuplicateNames(
	queries []*querier_dto.AnalysedQuery,
) []querier_dto.SourceError {
	var diagnostics []querier_dto.SourceError
	seen := make(map[string]*querier_dto.AnalysedQuery)

	for _, query := range queries {
		if existing, exists := seen[query.Name]; exists {
			diagnostics = append(diagnostics, querier_dto.SourceError{
				Filename: query.Filename,
				Line:     query.Line,
				Column:   1,
				Message: fmt.Sprintf(
					"duplicate query name %q (first defined in %s:%d)",
					query.Name, existing.Filename, existing.Line,
				),
				Severity: querier_dto.SeverityError,
				Code:     querier_dto.CodeDuplicateQueryName,
			})
		} else {
			seen[query.Name] = query
		}
	}

	return diagnostics
}

// commandName returns the human-readable name of a query
// command for use in diagnostic messages.
//
// Takes command (querier_dto.QueryCommand) which is the
// command enum value.
//
// Returns string which is the human-readable command name.
func commandName(command querier_dto.QueryCommand) string {
	switch command {
	case querier_dto.QueryCommandOne:
		return "one"
	case querier_dto.QueryCommandMany:
		return "many"
	case querier_dto.QueryCommandExec:
		return "exec"
	case querier_dto.QueryCommandExecResult:
		return "execresult"
	case querier_dto.QueryCommandExecRows:
		return "execrows"
	case querier_dto.QueryCommandBatch:
		return "batch"
	case querier_dto.QueryCommandStream:
		return "stream"
	case querier_dto.QueryCommandCopyFrom:
		return "copyfrom"
	default:
		return "unknown"
	}
}
