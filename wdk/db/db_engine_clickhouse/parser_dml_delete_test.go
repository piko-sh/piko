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
)

func TestAnalyseDelete_ExtractsWhereParameters(t *testing.T) {
	t.Parallel()

	analysis := analyse(t, "DELETE FROM events WHERE ts < {ts:DateTime64(9)}")

	assert.False(t, analysis.ReadOnly, "DELETE modifies data")
	require.Len(t, analysis.FromTables, 1)
	assert.Equal(t, "events", analysis.FromTables[0].Name)
	require.Len(t, analysis.ParameterReferences, 1, "the WHERE {ts:...} placeholder must be a parameter")
	assert.Equal(t, "ts", analysis.ParameterReferences[0].Name)
	require.NotNil(t, analysis.ParameterReferences[0].CastType)
	assert.Equal(t, "DateTime64", analysis.ParameterReferences[0].CastType.EngineName)
}

func TestAnalyseDelete_QualifiedTableAndMultipleParameters(t *testing.T) {
	t.Parallel()

	analysis := analyse(t, "DELETE FROM telemetry.events WHERE host_id = {host:UInt64} AND ts < {cutoff:DateTime}")

	require.Len(t, analysis.FromTables, 1)
	assert.Equal(t, "telemetry", analysis.FromTables[0].Schema)
	assert.Equal(t, "events", analysis.FromTables[0].Name)
	assert.Len(t, analysis.ParameterReferences, 2)
}

func TestAnalyseDelete_NoWhereHasNoParameters(t *testing.T) {
	t.Parallel()

	analysis := analyse(t, "DELETE FROM events")

	assert.False(t, analysis.ReadOnly)
	require.Len(t, analysis.FromTables, 1)
	assert.Equal(t, "events", analysis.FromTables[0].Name)
	assert.Empty(t, analysis.ParameterReferences)
}
