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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/querier/querier_dto"
)

func TestProjectionFunctionCallAttachesExpression(t *testing.T) {
	t.Parallel()

	t.Run("nested toInt64(count())", func(t *testing.T) {
		t.Parallel()
		analysis := analyse(t, "SELECT toInt64(count()) AS total FROM events")
		require.Len(t, analysis.OutputColumns, 1)
		assert.Equal(t, "total", analysis.OutputColumns[0].Name)
		outer, ok := analysis.OutputColumns[0].Expression.(*querier_dto.FunctionCallExpression)
		require.True(t, ok, "projection expression must be a FunctionCallExpression, got %T", analysis.OutputColumns[0].Expression)
		assert.Equal(t, "toInt64", outer.FunctionName)
		require.Len(t, outer.Arguments, 1)
		inner, ok := outer.Arguments[0].(*querier_dto.FunctionCallExpression)
		require.True(t, ok, "argument must be the nested count() call, got %T", outer.Arguments[0])
		assert.Equal(t, "count", inner.FunctionName)
	})

	t.Run("bare count()", func(t *testing.T) {
		t.Parallel()
		analysis := analyse(t, "SELECT count() AS n FROM events")
		require.Len(t, analysis.OutputColumns, 1)
		call, ok := analysis.OutputColumns[0].Expression.(*querier_dto.FunctionCallExpression)
		require.True(t, ok, "got %T", analysis.OutputColumns[0].Expression)
		assert.Equal(t, "count", call.FunctionName)
	})

	t.Run("single-arg cast toInt64(occurrence)", func(t *testing.T) {
		t.Parallel()
		analysis := analyse(t, "SELECT toInt64(occurrence) AS n FROM events")
		require.Len(t, analysis.OutputColumns, 1)
		call, ok := analysis.OutputColumns[0].Expression.(*querier_dto.FunctionCallExpression)
		require.True(t, ok, "got %T", analysis.OutputColumns[0].Expression)
		assert.Equal(t, "toInt64", call.FunctionName)
	})
}

func TestProjectionDirectColumnHasNoExpression(t *testing.T) {
	t.Parallel()

	analysis := analyse(t, "SELECT id FROM events")
	require.Len(t, analysis.OutputColumns, 1)
	assert.Equal(t, "id", analysis.OutputColumns[0].ColumnName)
	assert.Nil(t, analysis.OutputColumns[0].Expression)
}
