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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/typegen/typegen_dto"
)

func newTestManifest() *typegen_dto.ActionManifest {
	return &typegen_dto.ActionManifest{
		GeneratedAt: time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC),
		Actions: []typegen_dto.ActionEntry{
			{
				Name:           "customer.create",
				TSFunctionName: "customerCreate",
				FilePath:       "actions/customer/create.go",
				StructName:     "CreateAction",
				Method:         "POST",
				ReturnType:     "CustomerResponse",
				Documentation:  "Creates a new customer",
				Params: []typegen_dto.ActionParam{
					{
						Name:     "input",
						GoType:   "CreateInput",
						TSType:   "CreateInput",
						JSONName: "input",
						Optional: false,
					},
					{
						Name:     "note",
						GoType:   "*string",
						TSType:   "string | null",
						JSONName: "note",
						Optional: true,
					},
				},
			},
		},
		Types: []typegen_dto.ActionType{
			{
				Name:        "CreateInput",
				PackagePath: "myapp/actions/customer",
				Fields: []typegen_dto.ActionField{
					{
						Name:          "Email",
						GoType:        "string",
						TSType:        "string",
						JSONName:      "email",
						Documentation: "Customer email address",
						Optional:      false,
					},
					{
						Name:     "Age",
						GoType:   "int",
						TSType:   "number",
						JSONName: "age",
						Optional: true,
					},
				},
			},
		},
	}
}

func TestMarshalJSON(t *testing.T) {
	t.Parallel()

	t.Run("marshals complete manifest", func(t *testing.T) {
		t.Parallel()
		manifest := newTestManifest()

		data, err := MarshalJSON(manifest)
		require.NoError(t, err)

		output := string(data)
		assert.Contains(t, output, `"tsFunctionName"`)
		assert.Contains(t, output, `"customerCreate"`)
		assert.Contains(t, output, `"customer.create"`)
		assert.Contains(t, output, `"goType"`)
		assert.Contains(t, output, `"tsType"`)
		assert.Contains(t, output, `"jsonName"`)
		assert.Contains(t, output, `"generatedAt"`)
	})

	t.Run("marshals empty manifest", func(t *testing.T) {
		t.Parallel()
		manifest := &typegen_dto.ActionManifest{}

		data, err := MarshalJSON(manifest)
		require.NoError(t, err)

		output := string(data)
		assert.Contains(t, output, `"actions": []`)
		assert.Contains(t, output, `"types": []`)
	})
}

func TestUnmarshalJSON_InvalidJSON(t *testing.T) {
	t.Parallel()
	_, err := UnmarshalJSON([]byte("{{invalid"))
	assert.Error(t, err)
}

func TestJSON_Roundtrip(t *testing.T) {
	t.Parallel()
	original := newTestManifest()

	data, err := MarshalJSON(original)
	require.NoError(t, err)

	restored, err := UnmarshalJSON(data)
	require.NoError(t, err)

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

func TestUnixToTime(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    int64
		wantZero bool
	}{
		{
			name:  "positive timestamp",
			input: 1705312200,
		},
		{
			name:     "zero timestamp",
			input:    0,
			wantZero: true,
		},
		{
			name:     "negative timestamp",
			input:    -1,
			wantZero: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := unixToTime(tc.input)
			if tc.wantZero {
				assert.True(t, result.IsZero())
			} else {
				assert.False(t, result.IsZero())
				assert.Equal(t, tc.input, result.Unix())
			}
		})
	}
}
