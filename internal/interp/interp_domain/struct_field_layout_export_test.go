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

package interp_domain

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStructLayoutTableExportRoundTrip(t *testing.T) {
	t.Parallel()

	source := &CompiledFunction{
		structLayoutTable: []structFieldLayout{
			{
				Offset:       0,
				TypeIndex:    7,
				Path:         [4]uint8{0, 0, 0, 0},
				PathLength:   1,
				Kind:         uint8(reflect.Int64),
				RegisterKind: uint8(registerInt),
				Flags:        0,
			},
			{
				Offset:       16,
				TypeIndex:    7,
				Path:         [4]uint8{2, 1, 0, 0},
				PathLength:   2,
				Kind:         uint8(reflect.String),
				RegisterKind: uint8(registerString),
				Flags:        structFieldLayoutFlagEmbedded,
			},
			{
				Offset:       24,
				TypeIndex:    11,
				Path:         [4]uint8{3, 0, 0, 0},
				PathLength:   1,
				Kind:         uint8(reflect.Float64),
				RegisterKind: uint8(registerFloat),
				Flags:        0,
			},
		},
	}

	exported := source.StructLayoutTable()
	require.Len(t, exported, 3, "exported layoutTable preserves entry count")

	for i, entry := range exported {
		require.Equal(t, source.structLayoutTable[i].Offset, entry.Offset, "entry %d Offset", i)
		require.Equal(t, source.structLayoutTable[i].TypeIndex, entry.TypeIndex, "entry %d TypeIndex", i)
		require.Equal(t, source.structLayoutTable[i].Path, entry.Path, "entry %d Path", i)
		require.Equal(t, source.structLayoutTable[i].PathLength, entry.PathLength, "entry %d PathLength", i)
		require.Equal(t, source.structLayoutTable[i].Kind, entry.Kind, "entry %d Kind", i)
		require.Equal(t, source.structLayoutTable[i].RegisterKind, entry.RegisterKind, "entry %d RegisterKind", i)
		require.Equal(t, source.structLayoutTable[i].Flags, entry.Flags, "entry %d Flags", i)
	}

	rebuilt := makeStructLayoutTableFromData(exported)
	require.Equal(t, source.structLayoutTable, rebuilt,
		"makeStructLayoutTableFromData round trip restores byte-identical layoutTable")
}

func TestStructLayoutTableExportEmpty(t *testing.T) {
	t.Parallel()

	source := &CompiledFunction{}
	exported := source.StructLayoutTable()
	require.Empty(t, exported, "empty source produces empty exported table")

	rebuilt := makeStructLayoutTableFromData(exported)
	require.Empty(t, rebuilt, "empty exported table produces empty rebuilt table")
}
