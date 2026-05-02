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

func TestNullableParamRecordsNullability(t *testing.T) {
	t.Parallel()

	analysis := analyse(t, "INSERT INTO t (id, pid, label) VALUES ({id:UInt64}, {pid:Nullable(Int64)}, {label:Nullable(String)})")
	require.Len(t, analysis.ParameterReferences, 3)

	type castInfo struct {
		nullable   bool
		engineName string
	}
	byName := map[string]castInfo{}
	for index := range analysis.ParameterReferences {
		reference := analysis.ParameterReferences[index]
		require.NotNilf(t, reference.CastType, "param %q must have a cast type", reference.Name)
		byName[reference.Name] = castInfo{nullable: reference.CastType.Nullable, engineName: reference.CastType.EngineName}
	}

	assert.False(t, byName["id"].nullable, "UInt64 is not nullable")
	assert.True(t, byName["pid"].nullable, "Nullable(Int64) must record nullability")
	assert.True(t, byName["label"].nullable, "Nullable(String) must record nullability")
	assert.Equal(t, "Int64", byName["pid"].engineName, "Nullable(Int64) unwraps to its inner type")
}
