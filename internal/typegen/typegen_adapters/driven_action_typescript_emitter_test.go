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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/annotator/annotator_dto"
)

func newTestActionSpecs() []annotator_dto.ActionSpec {
	return []annotator_dto.ActionSpec{
		{
			Name:           "customer.create",
			TSFunctionName: "customerCreate",
			Description:    "Creates a new customer",
			CallParams: []annotator_dto.ParamSpec{
				{
					Name:     "input",
					GoType:   "CreateInput",
					TSType:   "CreateInput",
					JSONName: "input",
					Struct: &annotator_dto.TypeSpec{
						Name: "CreateInput",
						Fields: []annotator_dto.FieldSpec{
							{Name: "Email", TSType: "string", JSONName: "email"},
							{Name: "Name", TSType: "string", JSONName: "name"},
						},
					},
				},
			},
			ReturnType: &annotator_dto.TypeSpec{
				Name: "CustomerResponse",
				Fields: []annotator_dto.FieldSpec{
					{Name: "ID", TSType: "number", JSONName: "id"},
					{Name: "Email", TSType: "string", JSONName: "email"},
				},
			},
		},
		{
			Name:           "order.submit",
			TSFunctionName: "orderSubmit",
			Description:    "Submits an order",
		},
	}
}

func TestNewActionTypeScriptEmitter(t *testing.T) {
	t.Parallel()
	emitter := NewActionTypeScriptEmitter()
	require.NotNil(t, emitter)
}

func TestActionTypeScriptEmitter_EmitTypeScript(t *testing.T) {
	t.Parallel()
	emitter := NewActionTypeScriptEmitter()

	t.Run("generates complete output", func(t *testing.T) {
		t.Parallel()
		specs := newTestActionSpecs()

		output, err := emitter.EmitTypeScript(context.Background(), specs)
		require.NoError(t, err)

		result := string(output)
		requireValidTypeScript(t, result)
		assert.Contains(t, result, "DO NOT EDIT")
		assert.Contains(t, result, "import { ActionBuilder")
		assert.Contains(t, result, "registerActionFunction")
		assert.Contains(t, result, "export interface CreateInput")
		assert.Contains(t, result, "email: string")
		assert.Contains(t, result, "export interface CustomerResponse")
		assert.Contains(t, result, "export function customerCreate")
		assert.Contains(t, result, "export function orderSubmit")
		assert.Contains(t, result, "export const action")
		assert.Contains(t, result, "registerActionFunction('customer.create', customerCreate)")
		assert.Contains(t, result, "registerActionFunction('order.submit', orderSubmit)")
	})

	t.Run("empty specs produce minimal output", func(t *testing.T) {
		t.Parallel()
		output, err := emitter.EmitTypeScript(context.Background(), []annotator_dto.ActionSpec{})
		require.NoError(t, err)

		result := string(output)
		assert.Contains(t, result, "DO NOT EDIT")
		assert.Contains(t, result, "import { ActionBuilder")
		assert.NotContains(t, result, "export interface")
		assert.NotContains(t, result, "export function")
		assert.NotContains(t, result, "registerActionFunction(")
	})

	t.Run("generates JSDoc from description", func(t *testing.T) {
		t.Parallel()
		specs := newTestActionSpecs()

		output, err := emitter.EmitTypeScript(context.Background(), specs)
		require.NoError(t, err)

		result := string(output)
		assert.Contains(t, result, "/**")
		assert.Contains(t, result, "Creates a new customer")
		assert.Contains(t, result, " */")
	})

	t.Run("generates namespace grouping", func(t *testing.T) {
		t.Parallel()
		specs := newTestActionSpecs()

		output, err := emitter.EmitTypeScript(context.Background(), specs)
		require.NoError(t, err)

		result := string(output)
		assert.Contains(t, result, "customer: {")
		assert.Contains(t, result, "order: {")
	})
}

func TestToLowerCamelCase(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		input string
		want  string
	}{
		{name: "PascalCase", input: "CustomerCreate", want: "customerCreate"},
		{name: "already lower", input: "create", want: "create"},
		{name: "single char", input: "A", want: "a"},
		{name: "empty string", input: "", want: ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, toLowerCamelCase(tc.input))
		})
	}
}

func TestActionTSBuildParams(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		want   string
		params []annotator_dto.ParamSpec
	}{
		{
			name: "empty params",
			want: "",
		},
		{
			name: "single required param",
			params: []annotator_dto.ParamSpec{
				{JSONName: "input", TSType: "CreateInput"},
			},
			want: "input: CreateInput",
		},
		{
			name: "single optional param",
			params: []annotator_dto.ParamSpec{
				{JSONName: "note", TSType: "string", Optional: true},
			},
			want: "note?: string",
		},
		{
			name: "multiple params",
			params: []annotator_dto.ParamSpec{
				{JSONName: "input", TSType: "CreateInput"},
				{JSONName: "note", TSType: "string", Optional: true},
			},
			want: "input: CreateInput, note?: string",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, actionTSBuildParams(tc.params))
		})
	}
}

func TestActionTSBuildArgObject(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		want   string
		params []annotator_dto.ParamSpec
	}{
		{
			name: "empty params",
			want: "{}",
		},
		{
			name: "single struct param",
			params: []annotator_dto.ParamSpec{
				{JSONName: "input", Struct: &annotator_dto.TypeSpec{Name: "Input"}},
			},
			want: "input",
		},
		{
			name: "single non-struct param",
			params: []annotator_dto.ParamSpec{
				{JSONName: "id"},
			},
			want: "{ id }",
		},
		{
			name: "multiple params",
			params: []annotator_dto.ParamSpec{
				{JSONName: "a"},
				{JSONName: "b"},
			},
			want: "{ a, b }",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, actionTSBuildArgObject(tc.params))
		})
	}
}

func TestActionTSCollectTypeSpecs(t *testing.T) {
	t.Parallel()

	t.Run("deduplicates and sorts", func(t *testing.T) {
		t.Parallel()
		specs := []annotator_dto.ActionSpec{
			{
				CallParams: []annotator_dto.ParamSpec{
					{Struct: &annotator_dto.TypeSpec{Name: "Bravo"}},
					{Struct: &annotator_dto.TypeSpec{Name: "Alpha"}},
				},
			},
			{
				CallParams: []annotator_dto.ParamSpec{
					{Struct: &annotator_dto.TypeSpec{Name: "Bravo"}},
				},
			},
		}

		result := actionTSCollectTypeSpecs(specs)
		require.Len(t, result, 2)
		assert.Equal(t, "Alpha", result[0].Name)
		assert.Equal(t, "Bravo", result[1].Name)
	})

	t.Run("empty input returns empty", func(t *testing.T) {
		t.Parallel()
		result := actionTSCollectTypeSpecs(nil)
		assert.Empty(t, result)
	})

	t.Run("no struct params returns empty", func(t *testing.T) {
		t.Parallel()
		specs := []annotator_dto.ActionSpec{
			{
				CallParams: []annotator_dto.ParamSpec{
					{Name: "id"},
				},
			},
		}
		result := actionTSCollectTypeSpecs(specs)
		assert.Empty(t, result)
	})
}

func TestActionTSCollectResponseTypes(t *testing.T) {
	t.Parallel()

	t.Run("deduplicates and sorts", func(t *testing.T) {
		t.Parallel()
		specs := []annotator_dto.ActionSpec{
			{ReturnType: &annotator_dto.TypeSpec{Name: "Zulu"}},
			{ReturnType: &annotator_dto.TypeSpec{Name: "Alpha"}},
			{ReturnType: &annotator_dto.TypeSpec{Name: "Zulu"}},
		}

		result := actionTSCollectResponseTypes(specs)
		require.Len(t, result, 2)
		assert.Equal(t, "Alpha", result[0].Name)
		assert.Equal(t, "Zulu", result[1].Name)
	})

	t.Run("nil ReturnType excluded", func(t *testing.T) {
		t.Parallel()
		specs := []annotator_dto.ActionSpec{
			{ReturnType: nil},
			{ReturnType: &annotator_dto.TypeSpec{Name: "Response"}},
		}

		result := actionTSCollectResponseTypes(specs)
		require.Len(t, result, 1)
		assert.Equal(t, "Response", result[0].Name)
	})

	t.Run("empty input returns nil", func(t *testing.T) {
		t.Parallel()
		result := actionTSCollectResponseTypes(nil)
		assert.Nil(t, result)
	})
}

func TestActionTSGroupByNamespace(t *testing.T) {
	t.Parallel()

	specs := []annotator_dto.ActionSpec{
		{Name: "customer.create"},
		{Name: "customer.update"},
		{Name: "simpleAction"},
		{Name: "order.submit"},
	}

	groups := actionTSGroupByNamespace(specs)

	assert.Len(t, groups["customer"], 2)
	assert.Len(t, groups[""], 1)
	assert.Len(t, groups["order"], 1)
	assert.Equal(t, "simpleAction", groups[""][0].Name)
}

func TestActionTSExtractFunctionName(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		tsFunctionName string
		namespace      string
		want           string
	}{
		{
			name:           "extracts suffix preserving case",
			tsFunctionName: "customerCreate",
			namespace:      "customer",
			want:           "Create",
		},
		{
			name:           "too short returns original",
			tsFunctionName: "ab",
			namespace:      "abcdef",
			want:           "ab",
		},
		{
			name:           "same length returns original",
			tsFunctionName: "customer",
			namespace:      "customer",
			want:           "customer",
		},
		{
			name:           "preserves full case of suffix",
			tsFunctionName: "orderSubmitAll",
			namespace:      "order",
			want:           "SubmitAll",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := actionTSExtractFunctionName(tc.tsFunctionName, tc.namespace)

			assert.Equal(t, tc.want, result)
		})
	}
}

func TestActionTSSortNamespaces(t *testing.T) {
	t.Parallel()

	groups := map[string][]annotator_dto.ActionSpec{
		"zebra":  {},
		"alpha":  {},
		"":       {},
		"middle": {},
	}

	result := actionTSSortNamespaces(groups)
	require.Len(t, result, 4)

	assert.True(t, strings.Compare(result[0], result[1]) <= 0)
	assert.True(t, strings.Compare(result[1], result[2]) <= 0)
	assert.True(t, strings.Compare(result[2], result[3]) <= 0)
}
