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

// generatedColumnPass detects INSERT/UPDATE statements that attempt to write to generated
// (computed) columns. SQLite rejects these at runtime, so we report them at compile time.
type generatedColumnPass struct {
	// catalogue holds the schema state for column lookups.
	catalogue *querier_dto.Catalogue
}

// Analyse detects writes to generated columns.
//
// Takes context (*diagnosticContext) which holds the query and analysis state.
//
// Returns []querier_dto.SourceError which holds any diagnostics for generated column
// writes.
func (p *generatedColumnPass) Analyse(context *diagnosticContext) []querier_dto.SourceError {
	var diagnostics []querier_dto.SourceError

	if context.RawAnalysis == nil {
		return nil
	}

	for _, reference := range context.RawAnalysis.ParameterReferences {
		if reference.Context != querier_dto.ParameterContextAssignment {
			continue
		}
		if reference.ColumnReference == nil {
			continue
		}

		column := findCatalogueColumn(p.catalogue, reference.ColumnReference.TableAlias, reference.ColumnReference.ColumnName)
		if column == nil || !column.IsGenerated {
			continue
		}

		var parameterLabel string
		if reference.Name != "" {
			parameterLabel = fmt.Sprintf("%q", reference.Name)
		} else {
			parameterLabel = fmt.Sprintf("%d", reference.Number)
		}
		diagnostics = append(diagnostics, querier_dto.SourceError{
			Filename: context.Filename,
			Line:     context.Query.Line,
			Column:   1,
			Message: fmt.Sprintf(
				"parameter %s assigns to generated column %q which cannot be written to",
				parameterLabel, reference.ColumnReference.ColumnName,
			),
			Severity: querier_dto.SeverityWarning,
			Code:     querier_dto.CodeGeneratedColumn,
		})
	}

	return diagnostics
}
