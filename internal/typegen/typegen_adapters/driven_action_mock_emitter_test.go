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
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/annotator/annotator_dto"
)

func TestNewActionMockEmitter(t *testing.T) {
	t.Parallel()
	emitter := NewActionMockEmitter()
	require.NotNil(t, emitter)
}

func TestActionMockEmitter_EmitMocks(t *testing.T) {
	t.Parallel()
	emitter := NewActionMockEmitter()

	t.Run("generates complete output", func(t *testing.T) {
		t.Parallel()
		specs := newTestActionSpecs()

		output, err := emitter.EmitMocks(context.Background(), specs)
		require.NoError(t, err)

		result := string(output)
		requireValidTypeScript(t, result)
		assert.Contains(t, result, "DO NOT EDIT")
		assert.Contains(t, result, "import type { ActionError }")
		assert.Contains(t, result, "MockConfig")
		assert.Contains(t, result, "ActionMock")
		assert.Contains(t, result, "createMock")
		assert.Contains(t, result, "export const mocks")
		assert.Contains(t, result, "customerCreate:")
		assert.Contains(t, result, "resetAllMocks")
		assert.Contains(t, result, "getMockCalls")
		assert.Contains(t, result, "wasMockCalled")
	})

	t.Run("imports response types", func(t *testing.T) {
		t.Parallel()
		specs := newTestActionSpecs()

		output, err := emitter.EmitMocks(context.Background(), specs)
		require.NoError(t, err)

		result := string(output)
		assert.Contains(t, result, "import type { CustomerResponse }")
	})

	t.Run("generates error scenarios", func(t *testing.T) {
		t.Parallel()
		specs := newTestActionSpecs()

		output, err := emitter.EmitMocks(context.Background(), specs)
		require.NoError(t, err)

		result := string(output)
		assert.Contains(t, result, "notFound")
		assert.Contains(t, result, "validationError")
		assert.Contains(t, result, "serverError")
		assert.Contains(t, result, "networkError")
		assert.Contains(t, result, "unauthorized")
	})

	t.Run("empty specs produce framework only", func(t *testing.T) {
		t.Parallel()
		output, err := emitter.EmitMocks(context.Background(), []annotator_dto.ActionSpec{})
		require.NoError(t, err)

		result := string(output)
		assert.Contains(t, result, "MockConfig")
		assert.Contains(t, result, "ActionMock")
		assert.Contains(t, result, "createMock")
		assert.Contains(t, result, "export const mocks")
	})
}

func TestActionMockEmitter_GenerateDefaultValue(t *testing.T) {
	t.Parallel()
	emitter := NewActionMockEmitter()

	testCases := []struct {
		name     string
		typeSpec *annotator_dto.TypeSpec
		want     string
	}{
		{
			name: "nil type",
			want: "undefined",
		},
		{
			name:     "empty fields",
			typeSpec: &annotator_dto.TypeSpec{Name: "Empty"},
			want:     "{}",
		},
		{
			name: "type with fields",
			typeSpec: &annotator_dto.TypeSpec{
				Name: "Response",
				Fields: []annotator_dto.FieldSpec{
					{JSONName: "name", TSType: "string"},
					{JSONName: "count", TSType: "number"},
				},
			},
			want: "{ name: '', count: 0 }",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			value, ok := emitter.generateDefaultValue(actionTSTypeNames{}, tc.typeSpec)
			require.True(t, ok)
			assert.Equal(t, tc.want, value)
		})
	}
}

func TestActionMockEmitter_DefaultValueForTSType(t *testing.T) {
	t.Parallel()
	emitter := NewActionMockEmitter()

	testCases := []struct {
		name     string
		tsType   string
		want     string
		optional bool
	}{
		{name: "string", tsType: "string", want: "''"},
		{name: "number", tsType: "number", want: "0"},
		{name: "boolean", tsType: "boolean", want: "false"},
		{name: "Date", tsType: "Date", want: "new Date()"},
		{name: "File", tsType: "File", want: "new File([], 'test')"},
		{name: "Blob", tsType: "Blob", want: "new Blob()"},
		{name: "array type", tsType: "string[]", want: "[]"},
		{name: "nullable type", tsType: "Customer | null", want: "null"},
		{name: "optional field", tsType: "string", optional: true, want: "undefined"},
		{name: "unknown type", tsType: "CustomType", want: "{} as CustomType"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, emitter.defaultValueForTSType(tc.tsType, tc.optional))
		})
	}
}

func TestActionMockEmitter_CompositeReturnTypes(t *testing.T) {
	t.Parallel()
	emitter := NewActionMockEmitter()

	specs := []annotator_dto.ActionSpec{{
		Name:           "report.Fetch",
		TSFunctionName: "reportFetch",
		ReturnType:     &annotator_dto.TypeSpec{Name: "map[string]any"},
	}}

	output, err := emitter.EmitMocks(context.Background(), specs)
	require.NoError(t, err)

	result := string(output)
	requireValidTypeScript(t, result)
	assert.Contains(t, result, "createMock<Record<string, unknown>>")
	assert.NotContains(t, result, "import type { map[string]any }")
	assert.NotContains(t, result, "} from './actions';")
}

func TestActionMockEmitter_UnsupportedGoTypes(t *testing.T) {
	t.Parallel()
	emitter := NewActionMockEmitter()

	specs := []annotator_dto.ActionSpec{{
		Name:           "stream.Open",
		TSFunctionName: "streamOpen",
		ReturnType:     &annotator_dto.TypeSpec{Name: "chan int"},
	}}

	_, err := emitter.EmitMocks(context.Background(), specs)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `action "stream.Open"`)
	assert.Contains(t, err.Error(), `"chan int"`)
}

func TestActionMockEmitter_QuotesPropertyNames(t *testing.T) {
	t.Parallel()
	emitter := NewActionMockEmitter()

	specs := []annotator_dto.ActionSpec{{
		Name:           "echo.Run",
		TSFunctionName: "2faEcho",
		ReturnType: &annotator_dto.TypeSpec{
			Name: "EchoOutput",
			Fields: []annotator_dto.FieldSpec{
				{Name: "Spaced", GoType: "string", TSType: "string", JSONName: "a b"},
				{Name: "Plain", GoType: "bool", TSType: "boolean", JSONName: "plain"},
			},
		},
	}}

	output, err := emitter.EmitMocks(context.Background(), specs)
	require.NoError(t, err)

	result := string(output)
	requireValidTypeScript(t, result)
	assert.Contains(t, result, `default: { "a b": '', plain: false }`)
	assert.Contains(t, result, "_2faEcho: createMock<EchoOutput>")
}
