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
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/annotator/annotator_dto"
)

func TestNewActionWrapperEmitter(t *testing.T) {
	t.Parallel()

	emitter := NewActionWrapperEmitter()
	require.NotNil(t, emitter)
}

func TestEmitWrappers(t *testing.T) {
	t.Parallel()

	baseSpec := func(name string, params ...annotator_dto.ParamSpec) annotator_dto.ActionSpec {
		return annotator_dto.ActionSpec{
			Name:        name,
			PackagePath: "mymod/actions/test",
			PackageName: "test",
			StructName:  "TestAction",
			HTTPMethod:  "POST",
			HasError:    true,
			CallParams:  params,
		}
	}

	tests := []struct {
		name           string
		specs          []annotator_dto.ActionSpec
		wantContains   []string
		wantNotContain []string
	}{
		{
			name:  "single action no params",
			specs: []annotator_dto.ActionSpec{baseSpec("test.action")},
			wantContains: []string{
				"package actions",
				"invokeTestAction",
				"a := action.(*test.TestAction)",
			},
		},
		{
			name: "action with string param",
			specs: []annotator_dto.ActionSpec{
				baseSpec("test.action", annotator_dto.ParamSpec{
					Name: "name", GoType: "string", JSONName: "name",
				}),
			},
			wantContains: []string{
				`name, _ := argsMap["name"].(string)`,
			},
		},
		{
			name: "action with int param",
			specs: []annotator_dto.ActionSpec{
				baseSpec("test.action", annotator_dto.ParamSpec{
					Name: "count", GoType: "int", JSONName: "count",
				}),
			},
			wantContains: []string{
				`countRaw, _ := argsMap["count"].(float64)`,
				"count := int(countRaw)",
			},
		},
		{
			name: "action with int64 param",
			specs: []annotator_dto.ActionSpec{
				baseSpec("test.action", annotator_dto.ParamSpec{
					Name: "id", GoType: "int64", JSONName: "id",
				}),
			},
			wantContains: []string{
				`idRaw, _ := argsMap["id"].(float64)`,
				"id := int64(idRaw)",
			},
		},
		{
			name: "action with float64 param",
			specs: []annotator_dto.ActionSpec{
				baseSpec("test.action", annotator_dto.ParamSpec{
					Name: "price", GoType: "float64", JSONName: "price",
				}),
			},
			wantContains: []string{
				`price, _ := argsMap["price"].(float64)`,
			},
		},
		{
			name: "action with bool param",
			specs: []annotator_dto.ActionSpec{
				baseSpec("test.action", annotator_dto.ParamSpec{
					Name: "active", GoType: "bool", JSONName: "active",
				}),
			},
			wantContains: []string{
				`active, _ := argsMap["active"].(bool)`,
			},
		},
		{
			name: "action with struct param",
			specs: []annotator_dto.ActionSpec{
				baseSpec("test.action", annotator_dto.ParamSpec{
					Name:     "input",
					GoType:   "CreateInput",
					JSONName: "input",
					Struct: &annotator_dto.TypeSpec{
						Name:        "CreateInput",
						PackagePath: "mymod/actions/test",
					},
				}),
			},
			wantContains: []string{
				"var input test.CreateInput",
				"pikobinder.BindMap",
				"pikobinder.IgnoreUnknownKeys(true)",
			},
		},
		{
			name: "action with file upload",
			specs: []annotator_dto.ActionSpec{
				baseSpec("test.action", annotator_dto.ParamSpec{
					Name:         "avatar",
					JSONName:     "avatar",
					IsFileUpload: true,
				}),
			},
			wantContains: []string{
				"var avatar piko.FileUpload",
				"multipart.FileHeader",
				"piko.NewFileUpload",
			},
		},
		{
			name: "action with file upload slice",
			specs: []annotator_dto.ActionSpec{
				baseSpec("test.action", annotator_dto.ParamSpec{
					Name:              "files",
					JSONName:          "files",
					IsFileUploadSlice: true,
				}),
			},
			wantContains: []string{
				"var files []piko.FileUpload",
				"[]*multipart.FileHeader",
			},
		},
		{
			name: "action with raw body",
			specs: []annotator_dto.ActionSpec{
				baseSpec("test.action", annotator_dto.ParamSpec{
					Name:      "body",
					JSONName:  "body",
					IsRawBody: true,
				}),
			},
			wantContains: []string{
				"var body piko.RawBody",
				`"_rawBody"`,
			},
		},
		{
			name: "action without error returns nil error",
			specs: []annotator_dto.ActionSpec{
				{
					Name:        "test.action",
					PackagePath: "mymod/actions/test",
					PackageName: "test",
					StructName:  "TestAction",
					HTTPMethod:  "POST",
					HasError:    false,
				},
			},
			wantContains: []string{
				"result :=",
				"return result, nil",
			},
		},
		{
			name: "action with error returns directly",
			specs: []annotator_dto.ActionSpec{
				baseSpec("test.action"),
			},
			wantContains: []string{
				"return a.Call(",
			},
		},
		{
			name: "action with optional param uses address-of",
			specs: []annotator_dto.ActionSpec{
				baseSpec("test.action", annotator_dto.ParamSpec{
					Name:     "input",
					GoType:   "CreateInput",
					JSONName: "input",
					Optional: true,
					Struct: &annotator_dto.TypeSpec{
						Name:        "CreateInput",
						PackagePath: "mymod/actions/test",
					},
				}),
			},
			wantContains: []string{
				"&input",
			},
		},
		{
			name: "multiple actions sorted",
			specs: []annotator_dto.ActionSpec{
				baseSpec("zeta.action"),
				baseSpec("alpha.action"),
			},
			wantContains: []string{
				"invokeAlphaAction",
				"invokeZetaAction",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			emitter := NewActionWrapperEmitter()
			result, err := emitter.EmitWrappers(context.Background(), tt.specs)

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

func TestEmitWrappers_ValidGoSyntax(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		specs []annotator_dto.ActionSpec
	}{
		{
			name: "basic action",
			specs: []annotator_dto.ActionSpec{
				makeTestActionSpec("basic.action", "mymod/actions/basic", "basic", "BasicAction", "POST"),
			},
		},
		{
			name: "action with all param types",
			specs: []annotator_dto.ActionSpec{
				{
					Name:        "complex.action",
					PackagePath: "mymod/actions/complex",
					PackageName: "complex",
					StructName:  "ComplexAction",
					HTTPMethod:  "POST",
					HasError:    true,
					CallParams: []annotator_dto.ParamSpec{
						{Name: "name", GoType: "string", JSONName: "name"},
						{Name: "count", GoType: "int", JSONName: "count"},
						{Name: "id", GoType: "int64", JSONName: "id"},
						{Name: "price", GoType: "float64", JSONName: "price"},
						{Name: "active", GoType: "bool", JSONName: "active"},
					},
				},
			},
		},
		{
			name: "action with struct and file upload",
			specs: []annotator_dto.ActionSpec{
				{
					Name:        "upload.action",
					PackagePath: "mymod/actions/upload",
					PackageName: "upload",
					StructName:  "UploadAction",
					HTTPMethod:  "POST",
					HasError:    true,
					CallParams: []annotator_dto.ParamSpec{
						{
							Name:     "input",
							GoType:   "UploadInput",
							JSONName: "input",
							Struct: &annotator_dto.TypeSpec{
								Name:        "UploadInput",
								PackagePath: "mymod/actions/upload",
							},
						},
						{Name: "avatar", JSONName: "avatar", IsFileUpload: true},
						{Name: "files", JSONName: "files", IsFileUploadSlice: true},
						{Name: "body", JSONName: "body", IsRawBody: true},
					},
				},
			},
		},
		{
			name: "action without error",
			specs: []annotator_dto.ActionSpec{
				{
					Name:        "noerr.action",
					PackagePath: "mymod/actions/noerr",
					PackageName: "noerr",
					StructName:  "NoErrAction",
					HTTPMethod:  "GET",
					HasError:    false,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			emitter := NewActionWrapperEmitter()
			result, err := emitter.EmitWrappers(context.Background(), tt.specs)

			require.NoError(t, err)

			fset := token.NewFileSet()
			_, parseErr := parser.ParseFile(fset, "wrappers.go", result, parser.AllErrors)
			require.NoError(t, parseErr, "generated code should be valid Go:\n%s", string(result))
		})
	}
}

func TestEmitWrappers_DeterministicOutput(t *testing.T) {
	t.Parallel()

	specs := []annotator_dto.ActionSpec{
		makeTestActionSpec("user.create", "mymod/actions/user", "user", "CreateAction", "POST"),
		makeTestActionSpec("admin.delete", "mymod/actions/admin", "admin", "DeleteAction", "DELETE"),
	}

	emitter := NewActionWrapperEmitter()
	results := make([][]byte, 5)
	for i := range 5 {
		result, err := emitter.EmitWrappers(context.Background(), specs)
		require.NoError(t, err)
		results[i] = result
	}

	for i := 1; i < len(results); i++ {
		assert.Equal(t, string(results[0]), string(results[i]),
			"output should be deterministic across multiple builds")
	}
}

func TestCheckSpecialTypeImports(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		specs         []annotator_dto.ActionSpec
		wantPiko      bool
		wantMultipart bool
	}{
		{
			name: "no special types",
			specs: []annotator_dto.ActionSpec{
				{CallParams: []annotator_dto.ParamSpec{{GoType: "string"}}},
			},
			wantPiko:      false,
			wantMultipart: false,
		},
		{
			name: "file upload",
			specs: []annotator_dto.ActionSpec{
				{CallParams: []annotator_dto.ParamSpec{{IsFileUpload: true}}},
			},
			wantPiko:      true,
			wantMultipart: true,
		},
		{
			name: "file upload slice",
			specs: []annotator_dto.ActionSpec{
				{CallParams: []annotator_dto.ParamSpec{{IsFileUploadSlice: true}}},
			},
			wantPiko:      true,
			wantMultipart: true,
		},
		{
			name: "raw body",
			specs: []annotator_dto.ActionSpec{
				{CallParams: []annotator_dto.ParamSpec{{IsRawBody: true}}},
			},
			wantPiko:      true,
			wantMultipart: false,
		},
		{
			name: "mixed types",
			specs: []annotator_dto.ActionSpec{
				{CallParams: []annotator_dto.ParamSpec{{IsFileUpload: true}}},
				{CallParams: []annotator_dto.ParamSpec{{IsRawBody: true}}},
			},
			wantPiko:      true,
			wantMultipart: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			emitter := NewActionWrapperEmitter()
			gotPiko, gotMultipart := emitter.checkSpecialTypeImports(tt.specs)

			assert.Equal(t, tt.wantPiko, gotPiko, "needsPiko")
			assert.Equal(t, tt.wantMultipart, gotMultipart, "needsMultipart")
		})
	}
}

func TestParseTypeExpr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		typeName string
		wantType string
	}{
		{
			name:     "simple identifier",
			typeName: "string",
			wantType: "*ast.Ident",
		},
		{
			name:     "qualified name",
			typeName: "pkg.Type",
			wantType: "*ast.SelectorExpr",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := parseTypeExpr(tt.typeName)
			require.NotNil(t, result)

			switch tt.wantType {
			case "*ast.Ident":
				_, ok := result.(*ast.Ident)
				assert.True(t, ok, "expected *ast.Ident")
			case "*ast.SelectorExpr":
				_, ok := result.(*ast.SelectorExpr)
				assert.True(t, ok, "expected *ast.SelectorExpr")
			}
		})
	}
}

func TestWrapperQualifiedTypeName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		typeSpec *annotator_dto.TypeSpec
		want     string
	}{
		{
			name:     "nil type spec",
			typeSpec: nil,
			want:     "",
		},
		{
			name: "normal type",
			typeSpec: &annotator_dto.TypeSpec{
				Name:        "CreateInput",
				PackagePath: "mymod/actions/user",
			},
			want: "user.CreateInput",
		},
		{
			name: "deeply nested package",
			typeSpec: &annotator_dto.TypeSpec{
				Name:        "Output",
				PackagePath: "mymod/internal/actions/deep/nested/pkg",
			},
			want: "pkg.Output",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, wrapperQualifiedTypeName(tt.typeSpec))
		})
	}
}
