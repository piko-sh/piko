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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/querier/querier_dto"
)

func TestSliceCommandValidationPass(t *testing.T) {
	t.Parallel()
	pass := &sliceCommandValidationPass{}

	t.Run("slice with batch produces Q017 error", func(t *testing.T) {
		t.Parallel()
		context := &diagnosticContext{
			Filename: "test.sql",
			Query: &querier_dto.AnalysedQuery{
				Name:    "BatchInsertWithSlice",
				Line:    1,
				Command: querier_dto.QueryCommandBatch,
				Parameters: []querier_dto.QueryParameter{
					{Number: 1, Name: "ids", IsSlice: true},
				},
			},
		}
		diagnostics := pass.Analyse(context)
		require.Len(t, diagnostics, 1)
		assert.Equal(t, querier_dto.CodeSliceBatchCopyFrom, diagnostics[0].Code)
		assert.Equal(t, querier_dto.SeverityError, diagnostics[0].Severity)
	})

	t.Run("slice with copyfrom produces Q017 error", func(t *testing.T) {
		t.Parallel()
		context := &diagnosticContext{
			Filename: "test.sql",
			Query: &querier_dto.AnalysedQuery{
				Name:    "CopyWithSlice",
				Line:    1,
				Command: querier_dto.QueryCommandCopyFrom,
				Parameters: []querier_dto.QueryParameter{
					{Number: 1, Name: "ids", IsSlice: true},
				},
			},
		}
		diagnostics := pass.Analyse(context)
		require.Len(t, diagnostics, 1)
		assert.Equal(t, querier_dto.CodeSliceBatchCopyFrom, diagnostics[0].Code)
	})

	t.Run("slice with dynamic runtime produces Q018 error", func(t *testing.T) {
		t.Parallel()
		context := &diagnosticContext{
			Filename: "test.sql",
			Query: &querier_dto.AnalysedQuery{
				Name:           "DynamicRuntimeWithSlice",
				Line:           1,
				Command:        querier_dto.QueryCommandMany,
				DynamicRuntime: true,
				Parameters: []querier_dto.QueryParameter{
					{Number: 1, Name: "statuses", IsSlice: true},
				},
			},
		}
		diagnostics := pass.Analyse(context)
		require.Len(t, diagnostics, 1)
		assert.Equal(t, querier_dto.CodeSliceDynamicRuntime, diagnostics[0].Code)
	})

	t.Run("slice with sortable is allowed", func(t *testing.T) {
		t.Parallel()
		context := &diagnosticContext{
			Filename: "test.sql",
			Query: &querier_dto.AnalysedQuery{
				Name:    "SortableWithSlice",
				Line:    1,
				Command: querier_dto.QueryCommandMany,
				Parameters: []querier_dto.QueryParameter{
					{Number: 1, Name: "statuses", IsSlice: true},
					{Number: 2, Name: "order_by", Kind: querier_dto.ParameterDirectiveSortable},
				},
			},
		}
		diagnostics := pass.Analyse(context)
		assert.Empty(t, diagnostics)
	})

	t.Run("slice with many command produces no error", func(t *testing.T) {
		t.Parallel()
		context := &diagnosticContext{
			Filename: "test.sql",
			Query: &querier_dto.AnalysedQuery{
				Name:    "FetchByStatuses",
				Line:    1,
				Command: querier_dto.QueryCommandMany,
				Parameters: []querier_dto.QueryParameter{
					{Number: 1, Name: "statuses", IsSlice: true},
				},
			},
		}
		diagnostics := pass.Analyse(context)
		assert.Empty(t, diagnostics)
	})

	t.Run("no slice parameters produces no error", func(t *testing.T) {
		t.Parallel()
		context := &diagnosticContext{
			Filename: "test.sql",
			Query: &querier_dto.AnalysedQuery{
				Name:    "BatchInsert",
				Line:    1,
				Command: querier_dto.QueryCommandBatch,
				Parameters: []querier_dto.QueryParameter{
					{Number: 1, Name: "id", Kind: querier_dto.ParameterDirectiveParam},
				},
			},
		}
		diagnostics := pass.Analyse(context)
		assert.Empty(t, diagnostics)
	})

	t.Run("slice with optional is allowed", func(t *testing.T) {
		t.Parallel()
		context := &diagnosticContext{
			Filename: "test.sql",
			Query: &querier_dto.AnalysedQuery{
				Name:      "FilterWithSlice",
				Line:      1,
				Command:   querier_dto.QueryCommandMany,
				IsDynamic: true,
				Parameters: []querier_dto.QueryParameter{
					{Number: 1, Name: "statuses", IsSlice: true},
					{Number: 2, Name: "priority", IsOptional: true},
				},
			},
		}
		diagnostics := pass.Analyse(context)
		assert.Empty(t, diagnostics)
	})

	t.Run("slice with limit is allowed", func(t *testing.T) {
		t.Parallel()
		context := &diagnosticContext{
			Filename: "test.sql",
			Query: &querier_dto.AnalysedQuery{
				Name:      "LimitedSlice",
				Line:      1,
				Command:   querier_dto.QueryCommandMany,
				IsDynamic: true,
				Parameters: []querier_dto.QueryParameter{
					{Number: 1, Name: "statuses", IsSlice: true},
					{Number: 2, Name: "page_size", Context: querier_dto.ParameterContextLimit},
				},
			},
		}
		diagnostics := pass.Analyse(context)
		assert.Empty(t, diagnostics)
	})
}
