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

// sliceCommandValidationPass rejects piko.slice parameters in command and directive
// contexts where they cannot be safely expanded.
type sliceCommandValidationPass struct{}

// Analyse checks for invalid piko.slice usage.
//
// Takes validationContext (*diagnosticContext) which holds the query and analysis state.
//
// Returns []querier_dto.SourceError which holds any Q017 or Q018 diagnostics.
func (*sliceCommandValidationPass) Analyse(validationContext *diagnosticContext) []querier_dto.SourceError {
	hasSlice := false
	for i := range validationContext.Query.Parameters {
		if validationContext.Query.Parameters[i].IsSlice {
			hasSlice = true
			break
		}
	}
	if !hasSlice {
		return nil
	}

	var diagnostics []querier_dto.SourceError

	if validationContext.Query.Command == querier_dto.QueryCommandBatch ||
		validationContext.Query.Command == querier_dto.QueryCommandCopyFrom {
		diagnostics = append(diagnostics, querier_dto.SourceError{
			Filename: validationContext.Filename,
			Line:     validationContext.Query.Line,
			Column:   1,
			Message: fmt.Sprintf(
				"piko.slice cannot be used with :%s command in query %q - batch operations iterate over rows and cannot expand slice parameters",
				commandName(validationContext.Query.Command), validationContext.Query.Name,
			),
			Severity: querier_dto.SeverityError,
			Code:     querier_dto.CodeSliceBatchCopyFrom,
		})
	}

	if validationContext.Query.DynamicRuntime {
		diagnostics = append(diagnostics, querier_dto.SourceError{
			Filename: validationContext.Filename,
			Line:     validationContext.Query.Line,
			Column:   1,
			Message: fmt.Sprintf(
				"piko.slice cannot be used with piko.dynamic: runtime in query %q - the runtime builder cannot expand slice placeholders",
				validationContext.Query.Name,
			),
			Severity: querier_dto.SeverityError,
			Code:     querier_dto.CodeSliceDynamicRuntime,
		})
	}

	return diagnostics
}
