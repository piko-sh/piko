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

package typegen_adapters

import (
	"testing"
	"time"

	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/typegen/typegen_dto"
)

func TestGetBuilder_PutBuilder(t *testing.T) {
	t.Parallel()

	t.Run("GetBuilder returns non-nil", func(t *testing.T) {
		t.Parallel()
		b := GetBuilder()
		require.NotNil(t, b)
		PutBuilder(b)
	})

	t.Run("PutBuilder then GetBuilder works", func(t *testing.T) {
		t.Parallel()
		b1 := GetBuilder()
		require.NotNil(t, b1)
		PutBuilder(b1)

		b2 := GetBuilder()
		require.NotNil(t, b2)
		PutBuilder(b2)
	})
}

func TestBuildActionManifest(t *testing.T) {
	t.Parallel()

	t.Run("builds populated manifest", func(t *testing.T) {
		t.Parallel()
		manifest := newTestManifest()
		data := BuildActionManifest(manifest)
		require.NotNil(t, data)
		assert.Greater(t, len(data), 0)
	})

	t.Run("builds empty manifest", func(t *testing.T) {
		t.Parallel()
		manifest := &typegen_dto.ActionManifest{}
		data := BuildActionManifest(manifest)
		require.NotNil(t, data)
		assert.Greater(t, len(data), 0)
	})
}

func TestBuildActionManifestInto(t *testing.T) {
	t.Parallel()
	builder := flatbuffers.NewBuilder(1024)
	manifest := newTestManifest()

	data := BuildActionManifestInto(builder, manifest)
	require.NotNil(t, data)
	assert.Greater(t, len(data), 0)
}

func TestParseActionManifest(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		input   []byte
		wantNil bool
	}{
		{
			name:    "nil data returns nil",
			input:   nil,
			wantNil: true,
		},
		{
			name:    "empty data returns nil",
			input:   []byte{},
			wantNil: true,
		},
		{
			name:    "valid data parses successfully",
			input:   BuildActionManifest(newTestManifest()),
			wantNil: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := ParseActionManifest(tc.input)
			if tc.wantNil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
			}
		})
	}
}

func TestFBS_Roundtrip(t *testing.T) {
	t.Parallel()
	original := newTestManifest()

	data := BuildActionManifest(original)
	require.NotNil(t, data)

	restored := ParseActionManifest(data)
	require.NotNil(t, restored)

	assert.Equal(t, original.GeneratedAt.Unix(), restored.GeneratedAt.Unix())

	require.Len(t, restored.Actions, len(original.Actions))
	for i := range original.Actions {
		origAction := original.Actions[i]
		resAction := restored.Actions[i]

		assert.Equal(t, origAction.Name, resAction.Name)
		assert.Equal(t, origAction.TSFunctionName, resAction.TSFunctionName)
		assert.Equal(t, origAction.FilePath, resAction.FilePath)
		assert.Equal(t, origAction.StructName, resAction.StructName)
		assert.Equal(t, origAction.Method, resAction.Method)
		assert.Equal(t, origAction.ReturnType, resAction.ReturnType)
		assert.Equal(t, origAction.Documentation, resAction.Documentation)
		require.Len(t, resAction.Params, len(origAction.Params))
		for j := range origAction.Params {
			assert.Equal(t, origAction.Params[j].Name, resAction.Params[j].Name)
			assert.Equal(t, origAction.Params[j].GoType, resAction.Params[j].GoType)
			assert.Equal(t, origAction.Params[j].TSType, resAction.Params[j].TSType)
			assert.Equal(t, origAction.Params[j].JSONName, resAction.Params[j].JSONName)
			assert.Equal(t, origAction.Params[j].Optional, resAction.Params[j].Optional)
		}
	}

	require.Len(t, restored.Types, len(original.Types))
	for i := range original.Types {
		originalType := original.Types[i]
		resType := restored.Types[i]

		assert.Equal(t, originalType.Name, resType.Name)
		assert.Equal(t, originalType.PackagePath, resType.PackagePath)

		require.Len(t, resType.Fields, len(originalType.Fields))
		for j := range originalType.Fields {
			assert.Equal(t, originalType.Fields[j].Name, resType.Fields[j].Name)
			assert.Equal(t, originalType.Fields[j].GoType, resType.Fields[j].GoType)
			assert.Equal(t, originalType.Fields[j].TSType, resType.Fields[j].TSType)
			assert.Equal(t, originalType.Fields[j].JSONName, resType.Fields[j].JSONName)
			assert.Equal(t, originalType.Fields[j].Optional, resType.Fields[j].Optional)
			assert.Equal(t, originalType.Fields[j].Documentation, resType.Fields[j].Documentation)
		}
	}
}

func TestFBS_Roundtrip_EmptyManifest(t *testing.T) {
	t.Parallel()
	original := &typegen_dto.ActionManifest{
		GeneratedAt: time.Time{},
	}

	data := BuildActionManifest(original)
	require.NotNil(t, data)

	restored := ParseActionManifest(data)
	require.NotNil(t, restored)

	assert.Empty(t, restored.Actions)
	assert.Empty(t, restored.Types)
}
