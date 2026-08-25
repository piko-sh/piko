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

func TestActionTSEscapeComment(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		input string
		want  string
	}{
		{name: "ordinary prose is unchanged", input: "Exports a report.", want: "Exports a report."},
		{name: "comment terminator is escaped", input: "ends a block with */ here", want: `ends a block with *\/ here`},
		{name: "every terminator is escaped", input: "*/ and */", want: `*\/ and *\/`},
		{name: "terminator at the end", input: "trailing */", want: `trailing *\/`},
		{name: "lone asterisk is kept", input: "a * b", want: "a * b"},
		{name: "lone slash is kept", input: "a / b", want: "a / b"},
		{name: "empty string", input: "", want: ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, actionTSEscapeComment(tc.input))
		})
	}
}

func TestActionTypeScriptEmitter_DescriptionCannotCloseTheJSDocBlock(t *testing.T) {
	t.Parallel()

	emitter := NewActionTypeScriptEmitter()
	output, err := emitter.EmitTypeScript(t.Context(), []annotator_dto.ActionSpec{{
		Name:           "report.Export",
		PackageName:    "report",
		PackagePath:    "app/actions/report",
		StructName:     "ExportAction",
		Description:    "Exports a report. */ globalThis.pwned = true; /*",
		TSFunctionName: "reportExport",
		ReturnType:     &annotator_dto.TypeSpec{Name: "string"},
	}})

	require.NoError(t, err)
	assert.NotContains(t, string(output), "*/ globalThis")
	assert.Contains(t, string(output), `*\/ globalThis`)
}

func TestActionTSBuildParams(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		want   string
		params []actionTSParam
	}{
		{
			name: "empty params",
			want: "",
		},
		{
			name: "single required param",
			params: []actionTSParam{
				{binding: "input", jsonName: "input", tsType: "CreateInput"},
			},
			want: "input: CreateInput",
		},
		{
			name: "single optional param",
			params: []actionTSParam{
				{binding: "note", jsonName: "note", tsType: "string", optional: true},
			},
			want: "note?: string",
		},
		{
			name: "multiple params",
			params: []actionTSParam{
				{binding: "input", jsonName: "input", tsType: "CreateInput"},
				{binding: "note", jsonName: "note", tsType: "string", optional: true},
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
		params []actionTSParam
	}{
		{
			name: "empty params",
			want: "{}",
		},
		{
			name: "single struct param",
			params: []actionTSParam{
				{binding: "input", jsonName: "input", isStruct: true},
			},
			want: "input",
		},
		{
			name: "single non-struct param",
			params: []actionTSParam{
				{binding: "id", jsonName: "id"},
			},
			want: "{ id }",
		},
		{
			name: "multiple params",
			params: []actionTSParam{
				{binding: "a", jsonName: "a"},
				{binding: "b", jsonName: "b"},
			},
			want: "{ a, b }",
		},
		{
			name: "renamed binding keeps the json key",
			params: []actionTSParam{
				{binding: "in_", jsonName: "in"},
				{binding: "b", jsonName: "b"},
			},
			want: "{ in: in_, b }",
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

	groups := actionTSGroupByNamespace(specs, actionTSAssignFunctionNames(specs, actionTSModuleScope()))

	assert.Len(t, groups["customer"], 2)
	assert.Len(t, groups[""], 1)
	assert.Len(t, groups["order"], 1)
	assert.Equal(t, "simpleAction", groups[""][0].name)
}

func TestActionTSExtractFunctionName(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		tsFunctionName string
		actionName     string
		namespace      string
		want           string
	}{
		{
			name:           "extracts suffix preserving case",
			tsFunctionName: "customerCreate",
			actionName:     "customer.create",
			namespace:      "customer",
			want:           "Create",
		},
		{
			name:           "too short falls back to the action name",
			tsFunctionName: "ab",
			actionName:     "abcdef.thing",
			namespace:      "abcdef",
			want:           "thing",
		},
		{
			name:           "same length returns original",
			tsFunctionName: "customer",
			actionName:     "customer",
			namespace:      "customer",
			want:           "customer",
		},
		{
			name:           "preserves full case of suffix",
			tsFunctionName: "orderSubmitAll",
			actionName:     "order.submitAll",
			namespace:      "order",
			want:           "SubmitAll",
		},
		{
			name:           "repaired function name falls back to the action name",
			tsFunctionName: "_2faLogin",
			actionName:     "weird.Name",
			namespace:      "weird",
			want:           "Name",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := actionTSExtractFunctionName(tc.actionName, tc.tsFunctionName, tc.namespace)

			assert.Equal(t, tc.want, result)
		})
	}
}

func TestActionTSSortNamespaces(t *testing.T) {
	t.Parallel()

	groups := map[string][]actionTSNamespaceEntry{
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

func TestActionTypeScriptEmitter_CompositeReturnTypes(t *testing.T) {
	t.Parallel()
	emitter := NewActionTypeScriptEmitter()

	testCases := []struct {
		name       string
		goType     string
		wantTSType string
	}{
		{name: "map", goType: "map[string]any", wantTSType: "Record<string, unknown>"},
		{name: "map of slices", goType: "map[string][]int", wantTSType: "Record<string, number[]>"},
		{name: "slice", goType: "[]string", wantTSType: "string[]"},
		{name: "slice of pointers", goType: "[]*string", wantTSType: "(string | null)[]"},
		{name: "byte slice", goType: "[]byte", wantTSType: "string"},
		{name: "fixed array", goType: "[3]int", wantTSType: "number[]"},
		{name: "primitive", goType: "string", wantTSType: "string"},
		{name: "empty interface", goType: "interface{}", wantTSType: "unknown"},
		{name: "unresolved named type", goType: "time.Duration", wantTSType: "unknown"},
		{name: "time", goType: "time.Time", wantTSType: "string"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			specs := []annotator_dto.ActionSpec{{
				Name:           "report.Fetch",
				TSFunctionName: "reportFetch",
				ReturnType:     &annotator_dto.TypeSpec{Name: tc.goType},
			}}

			output, err := emitter.EmitTypeScript(context.Background(), specs)
			require.NoError(t, err)

			result := string(output)
			requireValidTypeScript(t, result)
			assert.Contains(t, result, "ActionBuilder<"+tc.wantTSType+">")
			assert.NotContains(t, result, "export interface "+tc.goType)
		})
	}
}

func TestActionTypeScriptEmitter_UnsupportedGoTypes(t *testing.T) {
	t.Parallel()
	emitter := NewActionTypeScriptEmitter()

	t.Run("return type names the action and the method", func(t *testing.T) {
		t.Parallel()
		specs := []annotator_dto.ActionSpec{{
			Name:           "stream.Open",
			TSFunctionName: "streamOpen",
			ReturnType:     &annotator_dto.TypeSpec{Name: "chan int"},
		}}

		_, err := emitter.EmitTypeScript(context.Background(), specs)
		require.Error(t, err)
		assert.Contains(t, err.Error(), `action "stream.Open"`)
		assert.Contains(t, err.Error(), "Call")
		assert.Contains(t, err.Error(), `"chan int"`)
	})

	t.Run("parameter type names the parameter", func(t *testing.T) {
		t.Parallel()
		specs := []annotator_dto.ActionSpec{{
			Name:           "stream.Open",
			TSFunctionName: "streamOpen",
			CallParams: []annotator_dto.ParamSpec{{
				Name:     "handler",
				GoType:   "func(int) error",
				TSType:   "func(int) error",
				JSONName: "handler",
			}},
		}}

		_, err := emitter.EmitTypeScript(context.Background(), specs)
		require.Error(t, err)
		assert.Contains(t, err.Error(), `parameter "handler"`)
		assert.Contains(t, err.Error(), `"func(int) error"`)
	})

	t.Run("field type names the type and the field", func(t *testing.T) {
		t.Parallel()
		specs := []annotator_dto.ActionSpec{{
			Name:           "stream.Open",
			TSFunctionName: "streamOpen",
			ReturnType: &annotator_dto.TypeSpec{
				Name: "OpenOutput",
				Fields: []annotator_dto.FieldSpec{
					{Name: "Events", GoType: "chan string", TSType: "chan string", JSONName: "events"},
				},
			},
		}}

		_, err := emitter.EmitTypeScript(context.Background(), specs)
		require.Error(t, err)
		assert.Contains(t, err.Error(), `type "OpenOutput"`)
		assert.Contains(t, err.Error(), `field "Events"`)
	})
}

func TestActionTypeScriptEmitter_ReservedParameterNames(t *testing.T) {
	t.Parallel()
	emitter := NewActionTypeScriptEmitter()

	t.Run("single struct parameter is renamed and spread", func(t *testing.T) {
		t.Parallel()
		specs := []annotator_dto.ActionSpec{{
			Name:           "echo.Run",
			TSFunctionName: "echoRun",
			CallParams: []annotator_dto.ParamSpec{{
				Name:     "in",
				GoType:   "EchoInput",
				TSType:   "EchoInput",
				JSONName: "in",
				Struct: &annotator_dto.TypeSpec{
					Name:   "EchoInput",
					Fields: []annotator_dto.FieldSpec{{Name: "Text", TSType: "string", JSONName: "text"}},
				},
			}},
		}}

		output, err := emitter.EmitTypeScript(context.Background(), specs)
		require.NoError(t, err)

		result := string(output)
		requireValidTypeScript(t, result)
		assert.Contains(t, result, "export function echoRun(in_: EchoInput)")
		assert.Contains(t, result, "createActionBuilder<void>('echo.Run', in_)")
	})

	t.Run("multiple parameters keep their json keys", func(t *testing.T) {
		t.Parallel()
		specs := []annotator_dto.ActionSpec{{
			Name:           "echo.Run",
			TSFunctionName: "echoRun",
			CallParams: []annotator_dto.ParamSpec{
				{Name: "class", GoType: "string", TSType: "string", JSONName: "class"},
				{Name: "id", GoType: "int64", TSType: "number", JSONName: "id"},
			},
		}}

		output, err := emitter.EmitTypeScript(context.Background(), specs)
		require.NoError(t, err)

		result := string(output)
		requireValidTypeScript(t, result)
		assert.Contains(t, result, "export function echoRun(class_: string, id: number)")
		assert.Contains(t, result, "{ class: class_, id }")
	})

	t.Run("parameters that sanitise alike stay distinct", func(t *testing.T) {
		t.Parallel()
		specs := []annotator_dto.ActionSpec{{
			Name:           "echo.Run",
			TSFunctionName: "echoRun",
			CallParams: []annotator_dto.ParamSpec{
				{Name: "a", GoType: "string", TSType: "string", JSONName: "a-b"},
				{Name: "b", GoType: "string", TSType: "string", JSONName: "a.b"},
			},
		}}

		output, err := emitter.EmitTypeScript(context.Background(), specs)
		require.NoError(t, err)

		result := string(output)
		requireValidTypeScript(t, result)
		assert.Contains(t, result, "export function echoRun(a_b: string, a_b2: string)")
		assert.Contains(t, result, `{ "a-b": a_b, "a.b": a_b2 }`)
	})
}

func TestActionTypeScriptEmitter_QuotesPropertyNames(t *testing.T) {
	t.Parallel()
	emitter := NewActionTypeScriptEmitter()

	specs := []annotator_dto.ActionSpec{{
		Name:           "echo.Run",
		TSFunctionName: "echoRun",
		ReturnType: &annotator_dto.TypeSpec{
			Name: "EchoOutput",
			Fields: []annotator_dto.FieldSpec{
				{Name: "Spaced", GoType: "string", TSType: "string", JSONName: "a b"},
				{Name: "Hyphen", GoType: "string", TSType: "string", JSONName: "my-field"},
				{Name: "Digit", GoType: "string", TSType: "string", JSONName: "2fa"},
				{Name: "Plain", GoType: "string", TSType: "string", JSONName: "plain"},
				{Name: "Quoted", GoType: "string", TSType: "string", JSONName: `he"llo`},
			},
		},
	}}

	output, err := emitter.EmitTypeScript(context.Background(), specs)
	require.NoError(t, err)

	result := string(output)
	requireValidTypeScript(t, result)
	assert.Contains(t, result, `  "a b": string;`)
	assert.Contains(t, result, `  "my-field": string;`)
	assert.Contains(t, result, `  "2fa": string;`)
	assert.Contains(t, result, "  plain: string;")
	assert.Contains(t, result, `  "he\"llo": string;`)
}

func TestActionTypeScriptEmitter_RepairsGeneratedNames(t *testing.T) {
	t.Parallel()
	emitter := NewActionTypeScriptEmitter()

	t.Run("function name that cannot be bound", func(t *testing.T) {
		t.Parallel()
		specs := []annotator_dto.ActionSpec{{
			Name:           "auth.Login",
			TSFunctionName: "2faLogin",
		}}

		output, err := emitter.EmitTypeScript(context.Background(), specs)
		require.NoError(t, err)

		result := string(output)
		requireValidTypeScript(t, result)
		assert.Contains(t, result, "export function _2faLogin()")
		assert.Contains(t, result, "registerActionFunction('auth.Login', _2faLogin)")
		assert.Contains(t, result, "Login: _2faLogin")
	})

	t.Run("type name that is a reserved word", func(t *testing.T) {
		t.Parallel()
		specs := []annotator_dto.ActionSpec{{
			Name:           "record.Remove",
			TSFunctionName: "recordRemove",
			CallParams: []annotator_dto.ParamSpec{{
				Name:     "input",
				GoType:   "delete",
				TSType:   "delete",
				JSONName: "input",
				Struct: &annotator_dto.TypeSpec{
					Name:   "delete",
					Fields: []annotator_dto.FieldSpec{{Name: "ID", GoType: "string", TSType: "string", JSONName: "id"}},
				},
			}},
		}}

		output, err := emitter.EmitTypeScript(context.Background(), specs)
		require.NoError(t, err)

		result := string(output)
		requireValidTypeScript(t, result)
		assert.Contains(t, result, "export interface delete_ {")
		assert.Contains(t, result, "export function recordRemove(input: delete_)")
	})

	t.Run("action name that would end its string literal", func(t *testing.T) {
		t.Parallel()
		specs := []annotator_dto.ActionSpec{{
			Name:           "it's.Fine",
			TSFunctionName: "itsFine",
		}}

		output, err := emitter.EmitTypeScript(context.Background(), specs)
		require.NoError(t, err)

		result := string(output)
		requireValidTypeScript(t, result)
		assert.Contains(t, result, `registerActionFunction("it's.Fine", itsFine)`)
	})
}

func TestActionTSIsParsableTSType(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		tsType string
		want   bool
	}{
		{name: "declared identifier", tsType: "Customer", want: true},
		{name: "declared array", tsType: "Customer[]", want: true},
		{name: "generic", tsType: "Record<string, any>", want: true},
		{name: "union with null", tsType: "Customer | null", want: true},
		{name: "parenthesised union array", tsType: "(string | null)[]", want: true},
		{name: "known global", tsType: "Date", want: true},
		{name: "empty", tsType: "", want: false},
		{name: "go channel", tsType: "chan int", want: false},
		{name: "go array", tsType: "[3]int", want: false},
		{name: "go func", tsType: "func(int) error", want: false},
		{name: "go struct", tsType: "struct{}", want: false},
		{name: "reserved word", tsType: "class", want: false},
		{name: "undeclared name resolves to nothing", tsType: "Decimal", want: false},
		{name: "undeclared name inside a union", tsType: "Customer | Decimal", want: false},
	}

	names := actionTSTypeNames{
		byKey:  map[string]string{"app/dto.Customer": "Customer"},
		byName: map[string]string{"Customer": "Customer"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, names.isResolvableTSType(tc.tsType))
		})
	}
}

func TestActionTSQuote(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		input string
		want  string
	}{
		{name: "plain", input: "customer.create", want: "'customer.create'"},
		{name: "single quote", input: "it's", want: `"it's"`},
		{name: "backslash", input: `a\b`, want: `"a\\b"`},
		{name: "newline", input: "a\nb", want: `"a\nb"`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, actionTSQuote(tc.input))
		})
	}
}

func TestActionTypeScriptEmitterDeclaresBothTypesWhenPackagesShareATypeName(t *testing.T) {
	t.Parallel()

	emitter := NewActionTypeScriptEmitter()
	output, err := emitter.EmitTypeScript(t.Context(), []annotator_dto.ActionSpec{
		{
			Name: "admin.Run", TSFunctionName: "adminRun",
			PackagePath: "app/actions/admin", PackageName: "admin", StructName: "RunAction",
			ReturnType: &annotator_dto.TypeSpec{
				Name: "Output", PackagePath: "app/actions/admin",
				Fields: []annotator_dto.FieldSpec{{Name: "Rows", TSType: "number", JSONName: "rows"}},
			},
		},
		{
			Name: "public.Run", TSFunctionName: "publicRun",
			PackagePath: "app/actions/public", PackageName: "public", StructName: "RunAction",
			ReturnType: &annotator_dto.TypeSpec{
				Name: "Output", PackagePath: "app/actions/public",
				Fields: []annotator_dto.FieldSpec{{Name: "Message", TSType: "string", JSONName: "message"}},
			},
		},
	})

	require.NoError(t, err)
	result := string(output)
	requireValidTypeScript(t, result)

	assert.Contains(t, result, "export interface Output {")
	assert.Contains(t, result, "export interface Output2 {")
	assert.Contains(t, result, "rows: number")
	assert.Contains(t, result, "message: string")
	assert.Equal(t, 1, strings.Count(result, "export interface Output {"),
		"each package's type must be declared exactly once")
}

func TestActionTypeScriptEmitterKeepsTypeNamesClearOfTheRuntimeImports(t *testing.T) {
	t.Parallel()

	emitter := NewActionTypeScriptEmitter()
	output, err := emitter.EmitTypeScript(t.Context(), []annotator_dto.ActionSpec{{
		Name: "build.Make", TSFunctionName: "buildMake",
		PackagePath: "app/actions/build", PackageName: "build", StructName: "MakeAction",
		ReturnType: &annotator_dto.TypeSpec{
			Name: "ActionBuilder", PackagePath: "app/actions/build",
			Fields: []annotator_dto.FieldSpec{{Name: "OK", TSType: "boolean", JSONName: "ok"}},
		},
	}})

	require.NoError(t, err)
	result := string(output)
	requireValidTypeScript(t, result)

	assert.NotContains(t, result, "export interface ActionBuilder {",
		"a Go type called ActionBuilder would redeclare the imported runtime symbol")
	assert.Contains(t, result, "export interface ActionBuilder2 {")
}

func TestNamespaceKeyIsNotRenamedByAnUnrelatedAction(t *testing.T) {
	t.Parallel()

	emitter := NewActionTypeScriptEmitter()
	output, err := emitter.EmitTypeScript(t.Context(), []annotator_dto.ActionSpec{
		{Name: "zzaB.C", TSFunctionName: "zzaBC", PackagePath: "app/actions/zzab", PackageName: "zzab"},
		{Name: "zza.BC", TSFunctionName: "zzaBC", PackagePath: "app/actions/zza", PackageName: "zza"},
	})

	require.NoError(t, err)
	result := string(output)
	requireValidTypeScript(t, result)

	assert.Contains(t, result, "C: ", "the namespace key must come from the action name")
	assert.NotContains(t, result, "C2: ",
		"one action gaining a suffix must not rename another action's namespace key")
}

func TestNamespaceKeysStayApartWithinOneNamespace(t *testing.T) {
	t.Parallel()

	emitter := NewActionTypeScriptEmitter()
	output, err := emitter.EmitTypeScript(t.Context(), []annotator_dto.ActionSpec{
		{Name: "report.run", TSFunctionName: "run", PackagePath: "app/a", PackageName: "a"},
		{Name: "report.run", TSFunctionName: "runAgain", PackagePath: "app/b", PackageName: "b"},
	})

	require.NoError(t, err)
	result := string(output)
	requireValidTypeScript(t, result)

	assert.Contains(t, result, "run: ")
	assert.Contains(t, result, "run2: ",
		"two entries in one namespace object must not declare the same key")
}

func TestUndeclaredTSTypeDegradesToUnknown(t *testing.T) {
	t.Parallel()

	emitter := NewActionTypeScriptEmitter()
	output, err := emitter.EmitTypeScript(t.Context(), []annotator_dto.ActionSpec{{
		Name: "money.Total", TSFunctionName: "moneyTotal",
		PackagePath: "app/actions/money", PackageName: "money", StructName: "TotalAction",
		ReturnType: &annotator_dto.TypeSpec{
			Name: "Amount", PackagePath: "app/actions/money",
			Fields: []annotator_dto.FieldSpec{
				{Name: "Value", GoType: "decimal.Decimal", TSType: "Decimal", JSONName: "value"},
				{Name: "At", GoType: "time.Time", TSType: "Date", JSONName: "at"},
			},
		},
	}})

	require.NoError(t, err)
	result := string(output)
	requireValidTypeScript(t, result)

	assert.Contains(t, result, "value: unknown",
		"a type name nothing declares must degrade, not be emitted as a dangling reference")
	assert.Contains(t, result, "at: Date", "a known global stays as it is")
}

func TestGenericGoReturnTypeDegradesInsteadOfFailingTheBuild(t *testing.T) {
	t.Parallel()

	emitter := NewActionTypeScriptEmitter()
	output, err := emitter.EmitTypeScript(t.Context(), []annotator_dto.ActionSpec{{
		Name: "page.List", TSFunctionName: "pageList",
		PackagePath: "app/actions/page", PackageName: "page", StructName: "ListAction",
		ReturnType: &annotator_dto.TypeSpec{Name: "Page[User]", PackagePath: "app/actions/page"},
	}})

	require.NoError(t, err, "a generic instantiation must degrade, not abort the whole build")
	result := string(output)
	requireValidTypeScript(t, result)
	assert.Contains(t, result, "ActionBuilder<unknown>")
}

func TestActionTSStripTypeArguments(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		input string
		want  string
	}{
		{name: "plain type", input: "User", want: "User"},
		{name: "qualified type", input: "dto.User", want: "dto.User"},
		{name: "generic instantiation", input: "Page[User]", want: "Page"},
		{name: "qualified generic", input: "dto.Page[dto.User]", want: "dto.Page"},
		{name: "nested arguments", input: "Page[Map[string, int]]", want: "Page"},
		{name: "go array is left alone", input: "[3]int", want: "[3]int"},
		{name: "go slice is left alone", input: "[]int", want: "[]int"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, actionTSStripTypeArguments(tc.input))
		})
	}
}

func TestInterfaceMembersAreCorrect(t *testing.T) {
	t.Parallel()

	shared := &annotator_dto.TypeSpec{
		Name: "Shared", PackagePath: "app/dto",
		Fields: []annotator_dto.FieldSpec{
			{Name: "Keep", TSType: "string", JSONName: "keep"},
			{Name: "Secret", TSType: "string", JSONName: "-"},
			{Name: "First", TSType: "string", JSONName: "same"},
			{Name: "Second", TSType: "number", JSONName: "same"},
		},
	}

	emitter := NewActionTypeScriptEmitter()
	output, err := emitter.EmitTypeScript(t.Context(), []annotator_dto.ActionSpec{{
		Name: "dto.Round", TSFunctionName: "dtoRound",
		PackagePath: "app/actions/dto", PackageName: "dto", StructName: "RoundAction",
		CallParams: []annotator_dto.ParamSpec{
			{Name: "input", JSONName: "input", GoType: "Shared", TSType: "Shared", Struct: shared},
		},
		ReturnType: shared,
	}})

	require.NoError(t, err)
	result := string(output)
	requireValidTypeScript(t, result)

	assert.Equal(t, 1, strings.Count(result, "export interface Shared {"),
		"a type used as both a parameter and a return is declared once")
	assert.Contains(t, result, "keep: string")
	assert.NotContains(t, result, `"-"`,
		`a json:"-" field must not become a required "-" property`)
	assert.Equal(t, 1, strings.Count(result, "same"),
		"two fields sharing a JSON name must not declare the member twice")
}

func TestDeeplyNestedGoTypeIsReportedRatherThanOverflowing(t *testing.T) {
	t.Parallel()

	names := actionTSTypeNames{byKey: map[string]string{}, byName: map[string]string{}}

	shallow, ok := names.goType(strings.Repeat("[]", 8) + "string")
	require.True(t, ok)
	assert.Equal(t, strings.Repeat("string", 1)+strings.Repeat("[]", 8), shallow)

	_, ok = names.goType(strings.Repeat("[]", maxGoTypeDepth+1) + "string")
	assert.False(t, ok, "a type past the depth cap must be reported, not truncated")
}

func TestEmptyStructKeepsItsDeclaredType(t *testing.T) {
	t.Parallel()

	emitter := NewActionTypeScriptEmitter()
	output, err := emitter.EmitTypeScript(t.Context(), []annotator_dto.ActionSpec{{
		Name: "errtest.Validation", TSFunctionName: "errtestValidation",
		PackagePath: "app/actions/errtest", PackageName: "errtest", StructName: "ValidationAction",
		HasError: true,
		CallParams: []annotator_dto.ParamSpec{{
			Name: "input", JSONName: "input", GoType: "ValidationInput", TSType: "ValidationInput",
			Struct: &annotator_dto.TypeSpec{Name: "ValidationInput", PackagePath: "app/actions/errtest"},
		}},
		ReturnType: &annotator_dto.TypeSpec{Name: "ValidationResponse", PackagePath: "app/actions/errtest"},
	}})

	require.NoError(t, err)
	result := string(output)
	requireValidTypeScript(t, result)

	assert.Contains(t, result, "export interface ValidationInput {",
		"an empty struct is how an action declares that it takes no input, so it keeps a named type")
	assert.Contains(t, result, "export interface ValidationResponse {")
	assert.Contains(t, result, "input: ValidationInput")
	assert.Contains(t, result, "ActionBuilder<ValidationResponse>")
}
