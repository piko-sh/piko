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

package generator_adapters

import (
	"context"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/annotator/annotator_dto"
)

func makeTestActionSpec(name, packagePath, packageName, structName, method string) annotator_dto.ActionSpec {
	return annotator_dto.ActionSpec{
		Name:        name,
		PackagePath: packagePath,
		PackageName: packageName,
		StructName:  structName,
		HTTPMethod:  method,
		HasError:    true,
	}
}

func TestNewActionRegistryEmitter(t *testing.T) {
	t.Parallel()

	emitter := NewActionRegistryEmitter()
	require.NotNil(t, emitter)
}

func TestEmitRegistry(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		specs          []annotator_dto.ActionSpec
		wantContains   []string
		wantNotContain []string
	}{
		{
			name:  "single action",
			specs: []annotator_dto.ActionSpec{makeTestActionSpec("user.create", "mymod/actions/user", "user", "CreateAction", "POST")},
			wantContains: []string{
				"package actions",
				`"user.create"`,
				"CreateAction",
				"func init()",
			},
		},
		{
			name: "multiple actions sorted alphabetically",
			specs: []annotator_dto.ActionSpec{
				makeTestActionSpec("zeta.action", "mymod/actions/zeta", "zeta", "ZetaAction", "POST"),
				makeTestActionSpec("alpha.action", "mymod/actions/alpha", "alpha", "AlphaAction", "GET"),
			},
			wantContains: []string{
				`"alpha.action"`,
				`"zeta.action"`,
			},
		},
		{
			name: "action with SSE",
			specs: []annotator_dto.ActionSpec{
				{
					Name:        "stream.action",
					PackagePath: "mymod/actions/stream",
					PackageName: "stream",
					StructName:  "StreamAction",
					HTTPMethod:  "POST",
					HasSSE:      true,
				},
			},
			wantContains: []string{
				"HasSSE:",
			},
		},
		{
			name: "action needing import alias",
			specs: []annotator_dto.ActionSpec{
				{
					Name:        "test.action",
					PackagePath: "mymod/actions/some_v2",
					PackageName: "some",
					StructName:  "TestAction",
					HTTPMethod:  "POST",
				},
			},
			wantContains: []string{
				`some "mymod/actions/some_v2"`,
			},
		},
		{
			name: "action with pretouch types from params",
			specs: []annotator_dto.ActionSpec{
				{
					Name:        "typed.action",
					PackagePath: "mymod/actions/typed",
					PackageName: "typed",
					StructName:  "TypedAction",
					HTTPMethod:  "POST",
					CallParams: []annotator_dto.ParamSpec{
						{
							Name:   "input",
							GoType: "CreateInput",
							Struct: &annotator_dto.TypeSpec{
								Name:        "CreateInput",
								PackagePath: "mymod/actions/typed",
							},
						},
					},
				},
			},
			wantContains: []string{
				"reflect.TypeFor",
				"pikojson.Pretouch",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			emitter := NewActionRegistryEmitter()
			result, err := emitter.EmitRegistry(context.Background(), tt.specs)

			require.NoError(t, err)
			require.NotEmpty(t, result)

			output := string(result)

			for _, want := range tt.wantContains {
				assert.Contains(t, output, want, "output should contain: %s", want)
			}

			for _, notWant := range tt.wantNotContain {
				assert.NotContains(t, output, notWant, "output should not contain: %s", notWant)
			}
		})
	}
}

func TestEmitRegistry_ValidGoSyntax(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		specs []annotator_dto.ActionSpec
	}{
		{
			name:  "single action",
			specs: []annotator_dto.ActionSpec{makeTestActionSpec("user.create", "mymod/actions/user", "user", "CreateAction", "POST")},
		},
		{
			name: "multiple actions",
			specs: []annotator_dto.ActionSpec{
				makeTestActionSpec("a.action", "mymod/actions/a", "a", "AAction", "GET"),
				makeTestActionSpec("b.action", "mymod/actions/b", "b", "BAction", "POST"),
				makeTestActionSpec("c.action", "mymod/actions/c", "c", "CAction", "PUT"),
			},
		},
		{
			name: "action with pretouch types",
			specs: []annotator_dto.ActionSpec{
				{
					Name:        "typed.action",
					PackagePath: "mymod/actions/typed",
					PackageName: "typed",
					StructName:  "TypedAction",
					HTTPMethod:  "POST",
					CallParams: []annotator_dto.ParamSpec{
						{
							Name:   "input",
							GoType: "Input",
							Struct: &annotator_dto.TypeSpec{
								Name:        "Input",
								PackagePath: "mymod/actions/typed",
							},
						},
					},
					ReturnType: &annotator_dto.TypeSpec{
						Name:        "Output",
						PackagePath: "mymod/actions/typed",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			emitter := NewActionRegistryEmitter()
			result, err := emitter.EmitRegistry(context.Background(), tt.specs)

			require.NoError(t, err)

			fset := token.NewFileSet()
			_, parseErr := parser.ParseFile(fset, "registry.go", result, parser.AllErrors)
			require.NoError(t, parseErr, "generated code should be valid Go:\n%s", string(result))
		})
	}
}

func TestEmitRegistry_DeterministicOutput(t *testing.T) {
	t.Parallel()

	specs := []annotator_dto.ActionSpec{
		makeTestActionSpec("user.create", "mymod/actions/user", "user", "CreateAction", "POST"),
		makeTestActionSpec("admin.delete", "mymod/actions/admin", "admin", "DeleteAction", "DELETE"),
	}

	emitter := NewActionRegistryEmitter()
	results := make([][]byte, 5)
	for i := range 5 {
		result, err := emitter.EmitRegistry(context.Background(), specs)
		require.NoError(t, err)
		results[i] = result
	}

	for i := 1; i < len(results); i++ {
		assert.Equal(t, string(results[0]), string(results[i]),
			"output should be deterministic across multiple builds")
	}
}

func TestActionSortSpecs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     []annotator_dto.ActionSpec
		wantNames []string
	}{
		{
			name:      "empty",
			input:     []annotator_dto.ActionSpec{},
			wantNames: []string{},
		},
		{
			name:      "single",
			input:     []annotator_dto.ActionSpec{{Name: "alpha"}},
			wantNames: []string{"alpha"},
		},
		{
			name: "already sorted",
			input: []annotator_dto.ActionSpec{
				{Name: "alpha"},
				{Name: "beta"},
				{Name: "gamma"},
			},
			wantNames: []string{"alpha", "beta", "gamma"},
		},
		{
			name: "reversed",
			input: []annotator_dto.ActionSpec{
				{Name: "gamma"},
				{Name: "beta"},
				{Name: "alpha"},
			},
			wantNames: []string{"alpha", "beta", "gamma"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := actionSortSpecs(tt.input)

			require.Len(t, result, len(tt.wantNames))
			for i, name := range tt.wantNames {
				assert.Equal(t, name, result[i].Name)
			}
		})
	}
}

func TestActionSortSpecs_DoesNotModifyOriginal(t *testing.T) {
	t.Parallel()

	original := []annotator_dto.ActionSpec{
		{Name: "gamma"},
		{Name: "alpha"},
		{Name: "beta"},
	}

	_ = actionSortSpecs(original)

	assert.Equal(t, "gamma", original[0].Name)
	assert.Equal(t, "alpha", original[1].Name)
	assert.Equal(t, "beta", original[2].Name)
}

func TestActionNeedsAlias(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		packagePath string
		packageName string
		want        bool
	}{
		{
			name:        "matching - no alias needed",
			packagePath: "mymod/actions/email",
			packageName: "email",
			want:        false,
		},
		{
			name:        "not matching - alias needed",
			packagePath: "mymod/actions/email_v2",
			packageName: "email",
			want:        true,
		},
		{
			name:        "single segment matching",
			packagePath: "email",
			packageName: "email",
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, actionNeedsAlias(tt.packagePath, tt.packageName))
		})
	}
}

func TestActionToPascalCase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "single word", input: "hello", want: "Hello"},
		{name: "dotted name", input: "email.contact", want: "EmailContact"},
		{name: "triple dotted", input: "user.auth.login", want: "UserAuthLogin"},
		{name: "empty string", input: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, actionToPascalCase(tt.input))
		})
	}
}

func TestCollectPretouchTypes(t *testing.T) {
	t.Parallel()

	emitter := NewActionRegistryEmitter()

	tests := []struct {
		name    string
		specs   []annotator_dto.ActionSpec
		wantLen int
	}{
		{
			name: "no types",
			specs: []annotator_dto.ActionSpec{
				{Name: "basic", CallParams: nil, ReturnType: nil},
			},
			wantLen: 0,
		},
		{
			name: "params with struct types",
			specs: []annotator_dto.ActionSpec{
				{
					Name: "typed",
					CallParams: []annotator_dto.ParamSpec{
						{Struct: &annotator_dto.TypeSpec{Name: "Input", PackagePath: "mymod/types"}},
					},
				},
			},
			wantLen: 1,
		},
		{
			name: "return type",
			specs: []annotator_dto.ActionSpec{
				{
					Name:       "typed",
					ReturnType: &annotator_dto.TypeSpec{Name: "Output", PackagePath: "mymod/types"},
				},
			},
			wantLen: 1,
		},
		{
			name: "deduplication",
			specs: []annotator_dto.ActionSpec{
				{
					Name: "typed",
					CallParams: []annotator_dto.ParamSpec{
						{Struct: &annotator_dto.TypeSpec{Name: "Shared", PackagePath: "mymod/types"}},
					},
					ReturnType: &annotator_dto.TypeSpec{Name: "Shared", PackagePath: "mymod/types"},
				},
			},
			wantLen: 1,
		},
		{
			name: "skips empty package path",
			specs: []annotator_dto.ActionSpec{
				{
					Name: "builtin",
					CallParams: []annotator_dto.ParamSpec{
						{Struct: &annotator_dto.TypeSpec{Name: "string", PackagePath: ""}},
					},
				},
			},
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := emitter.collectPretouchTypes(tt.specs)
			assert.Len(t, result, tt.wantLen)
		})
	}
}

func TestCollectParamTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		params  []annotator_dto.ParamSpec
		wantLen int
	}{
		{
			name:    "nil struct",
			params:  []annotator_dto.ParamSpec{{Name: "basic", Struct: nil}},
			wantLen: 0,
		},
		{
			name:    "empty package path",
			params:  []annotator_dto.ParamSpec{{Struct: &annotator_dto.TypeSpec{Name: "string", PackagePath: ""}}},
			wantLen: 0,
		},
		{
			name:    "valid struct",
			params:  []annotator_dto.ParamSpec{{Struct: &annotator_dto.TypeSpec{Name: "Input", PackagePath: "mod/types"}}},
			wantLen: 1,
		},
		{
			name: "duplicate types",
			params: []annotator_dto.ParamSpec{
				{Struct: &annotator_dto.TypeSpec{Name: "Input", PackagePath: "mod/types"}},
				{Struct: &annotator_dto.TypeSpec{Name: "Input", PackagePath: "mod/types"}},
			},
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			seen := make(map[string]bool)
			result := collectParamTypes(tt.params, seen, nil)
			assert.Len(t, result, tt.wantLen)
		})
	}
}

func TestCollectReturnType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		returnType *annotator_dto.TypeSpec
		name       string
		wantLen    int
	}{
		{name: "nil", returnType: nil, wantLen: 0},
		{name: "empty package path", returnType: &annotator_dto.TypeSpec{Name: "string", PackagePath: ""}, wantLen: 0},
		{name: "valid type", returnType: &annotator_dto.TypeSpec{Name: "Output", PackagePath: "mod/types"}, wantLen: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			seen := make(map[string]bool)
			result := collectReturnType(tt.returnType, seen, nil)
			assert.Len(t, result, tt.wantLen)
		})
	}
}

func TestEmitRegistry_AlphabeticalOrder(t *testing.T) {
	t.Parallel()

	specs := []annotator_dto.ActionSpec{
		makeTestActionSpec("zeta.action", "mymod/actions/zeta", "zeta", "ZetaAction", "POST"),
		makeTestActionSpec("alpha.action", "mymod/actions/alpha", "alpha", "AlphaAction", "GET"),
		makeTestActionSpec("mid.action", "mymod/actions/mid", "mid", "MidAction", "PUT"),
	}

	emitter := NewActionRegistryEmitter()
	result, err := emitter.EmitRegistry(context.Background(), specs)
	require.NoError(t, err)

	output := string(result)
	alphaIndex := strings.Index(output, `"alpha.action"`)
	midIndex := strings.Index(output, `"mid.action"`)
	zetaIndex := strings.Index(output, `"zeta.action"`)

	assert.Less(t, alphaIndex, midIndex, "alpha should come before mid")
	assert.Less(t, midIndex, zetaIndex, "mid should come before zeta")
}
