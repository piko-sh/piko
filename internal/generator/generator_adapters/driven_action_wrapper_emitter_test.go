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
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/goastutil"
)

const (
	testActionPackagePath = "mymod/actions/test"
	testTypePackagePath   = "mymod/pkg/contracts"
)

var (
	testActionPackageAlias = goastutil.GoPackageAliasWithStem("test", testActionPackagePath)
	testTypePackageAlias   = goastutil.GoPackageAliasWithStem("contracts", testTypePackagePath)
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
			PackagePath: testActionPackagePath,
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
				"a := action.(*" + testActionPackageAlias + ".TestAction)",
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
						PackagePath: testActionPackagePath,
					},
				}),
			},
			wantContains: []string{
				"var input " + testActionPackageAlias + ".CreateInput",
				"pikobinder.BindMap",
				"pikobinder.IgnoreUnknownKeys(true)",
			},
		},
		{
			name: "action with struct param containing a file upload field",
			specs: []annotator_dto.ActionSpec{
				baseSpec("test.action", annotator_dto.ParamSpec{
					Name:     "input",
					GoType:   "CreateInput",
					JSONName: "input",
					Struct: &annotator_dto.TypeSpec{
						Name:        "CreateInput",
						PackagePath: testActionPackagePath,
						Fields: []annotator_dto.FieldSpec{
							{Name: "Title", GoType: "string", JSONName: "title"},
							{Name: "Avatar", GoType: "piko.FileUpload", JSONName: "avatar", TSType: "File", IsFileUpload: true},
						},
					},
				}),
			},
			wantContains: []string{
				"var input " + testActionPackageAlias + ".CreateInput",
				`argsMap["avatar"].(*multipart.FileHeader)`,
				"input.Avatar = piko.NewFileUpload(fh)",
				`delete(argsMap, "avatar")`,
				"pikobinder.BindMap",
			},
		},
		{
			name: "struct file upload field is extracted by its resolved json key",
			specs: []annotator_dto.ActionSpec{
				baseSpec("test.action", annotator_dto.ParamSpec{
					Name:     "input",
					GoType:   "CreateInput",
					JSONName: "input",
					Struct: &annotator_dto.TypeSpec{
						Name:        "CreateInput",
						PackagePath: testActionPackagePath,
						Fields: []annotator_dto.FieldSpec{
							{Name: "Title", GoType: "string", JSONName: "title"},
							{Name: "Avatar", GoType: "piko.FileUpload", JSONName: "avatar", TSType: "File", IsFileUpload: true},
						},
					},
				}),
			},
			wantContains: []string{
				`argsMap["avatar"].(*multipart.FileHeader)`,
			},
			wantNotContain: []string{
				`argsMap["Avatar"]`,
			},
		},
		{
			name: "struct pointer file upload field is assigned by address",
			specs: []annotator_dto.ActionSpec{
				baseSpec("test.action", annotator_dto.ParamSpec{
					Name:     "input",
					GoType:   "CreateInput",
					JSONName: "input",
					Struct: &annotator_dto.TypeSpec{
						Name:        "CreateInput",
						PackagePath: testActionPackagePath,
						Fields: []annotator_dto.FieldSpec{
							{Name: "Avatar", GoType: "*piko.FileUpload", JSONName: "avatar", TSType: "File", IsFileUpload: true, IsPointer: true},
						},
					},
				}),
			},
			wantContains: []string{
				"fu := piko.NewFileUpload(fh)",
				"input.Avatar = &fu",
			},
			wantNotContain: []string{
				"input.Avatar = piko.NewFileUpload(fh)",
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
					PackagePath: testActionPackagePath,
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
						PackagePath: testActionPackagePath,
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

func TestEmitWrappers_ReservedPackageNames(t *testing.T) {
	t.Parallel()

	specs := []annotator_dto.ActionSpec{
		makeTestActionSpec("log.Write", "mymod/actions/log", "log", "WriteAction", "POST"),
		makeTestActionSpec("context.Read", "mymod/actions/context", "context", "ReadAction", "POST"),
		makeTestActionSpec("logger.Tail", "mymod/actions/logger", "logger", "TailAction", "POST"),
		{
			Name:        "multipart.Send",
			PackagePath: "mymod/actions/multipart",
			PackageName: "multipart",
			StructName:  "SendAction",
			HTTPMethod:  "POST",
			HasError:    true,
			CallParams: []annotator_dto.ParamSpec{
				{Name: "attachment", JSONName: "attachment", IsFileUpload: true},
			},
		},
	}

	result, err := NewActionWrapperEmitter().EmitWrappers(context.Background(), specs)
	require.NoError(t, err)

	output := string(result)
	assert.Contains(t, output, "var "+actionLogVarName+" = logger.GetLogger")
	assert.Contains(t, output, "argsMap[\"attachment\"].(*multipart.FileHeader)",
		"the standard library multipart package must keep its own identifier")

	for _, spec := range specs {
		alias := goastutil.GoPackageAliasWithStem(spec.PackageName, spec.PackagePath)
		assert.Contains(t, output, alias+" \""+spec.PackagePath+"\"",
			"action package %s must be imported under its alias", spec.PackagePath)
		assert.Contains(t, output, "action.(*"+alias+"."+spec.StructName+")")
	}

	fset := token.NewFileSet()
	_, parseErr := parser.ParseFile(fset, "wrappers.go", result, parser.AllErrors)
	require.NoError(t, parseErr, "generated code should be valid Go:\n%s", output)
}

func TestEmitRegistryAndWrappers_AgreeOnGeneratedNames(t *testing.T) {
	t.Parallel()

	specs := []annotator_dto.ActionSpec{
		makeTestActionSpec("zza.BC", "mymod/actions/zza", "zza", "BCAction", "POST"),
		makeTestActionSpec("zzaB.C", "mymod/actions/zzaB", "zzaB", "CAction", "POST"),
		makeTestActionSpec("repo.GithubGet", "mymod/actions/github/repo", "repo", "GithubGetAction", "POST"),
		makeTestActionSpec("repo.GitlabGet", "mymod/actions/gitlab/repo", "repo", "GitlabGetAction", "POST"),
		makeTestActionSpec("log.Write", "mymod/actions/log", "log", "WriteAction", "POST"),
	}

	registryCode, err := NewActionRegistryEmitter().EmitRegistry(context.Background(), specs)
	require.NoError(t, err)
	wrapperCode, err := NewActionWrapperEmitter().EmitWrappers(context.Background(), specs)
	require.NoError(t, err)

	fset := token.NewFileSet()
	registryFile, err := parser.ParseFile(fset, "registry.go", registryCode, parser.AllErrors)
	require.NoError(t, err)
	wrapperFile, err := parser.ParseFile(fset, "wrappers.go", wrapperCode, parser.AllErrors)
	require.NoError(t, err)

	declared := actionDeclaredFunctionNames(wrapperFile)
	assert.Len(t, declared, len(specs), "every action needs its own wrapper function")

	referenced := actionInvokedWrapperNames(registryFile)
	require.NotEmpty(t, referenced)
	for _, name := range referenced {
		assert.Contains(t, declared, name, "registry invokes %s, which no wrapper declares", name)
	}

	assert.Equal(t, actionImportAliases(registryFile), actionImportAliases(wrapperFile),
		"both files must import the action packages under the same aliases")
}

func actionDeclaredFunctionNames(file *ast.File) map[string]struct{} {
	names := make(map[string]struct{})
	for _, decl := range file.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok || funcDecl.Recv != nil {
			continue
		}
		names[funcDecl.Name.Name] = struct{}{}
	}
	return names
}

func actionInvokedWrapperNames(file *ast.File) []string {
	var names []string
	ast.Inspect(file, func(node ast.Node) bool {
		keyValue, ok := node.(*ast.KeyValueExpr)
		if !ok {
			return true
		}
		key, ok := keyValue.Key.(*ast.Ident)
		if !ok || key.Name != "Invoke" {
			return true
		}
		if value, ok := keyValue.Value.(*ast.Ident); ok {
			names = append(names, value.Name)
		}
		return true
	})
	return names
}

func actionImportAliases(file *ast.File) map[string]string {
	aliases := make(map[string]string)
	for _, importSpec := range file.Imports {
		if importSpec.Name == nil {
			continue
		}
		path, err := strconv.Unquote(importSpec.Path.Value)
		if err != nil {
			continue
		}
		aliases[path] = importSpec.Name.Name
	}
	return aliases
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
			name: "action with struct param containing a file upload field",
			specs: []annotator_dto.ActionSpec{
				{
					Name:        "profile.action",
					PackagePath: "mymod/actions/profile",
					PackageName: "profile",
					StructName:  "ProfileAction",
					HTTPMethod:  "POST",
					HasError:    true,
					CallParams: []annotator_dto.ParamSpec{
						{
							Name:     "input",
							GoType:   "ProfileInput",
							JSONName: "input",
							Struct: &annotator_dto.TypeSpec{
								Name:        "ProfileInput",
								PackagePath: "mymod/actions/profile",
								Fields: []annotator_dto.FieldSpec{
									{Name: "Title", GoType: "string", JSONName: "title"},
									{Name: "Avatar", GoType: "piko.FileUpload", JSONName: "avatar", TSType: "File", IsFileUpload: true},
								},
							},
						},
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
		typeSpec *annotator_dto.TypeSpec
		name     string
		want     string
	}{
		{
			name:     "nil type spec",
			typeSpec: nil,
			want:     "",
		},
		{
			name: "type in an aliased action package",
			typeSpec: &annotator_dto.TypeSpec{
				Name:        "CreateInput",
				PackagePath: "mymod/actions/user",
			},
			want: goastutil.GoPackageAliasWithStem("user", "mymod/actions/user") + ".CreateInput",
		},
		{
			name: "type in a package that holds no action still gets a claimed alias",
			typeSpec: &annotator_dto.TypeSpec{
				Name:        "Output",
				PackagePath: "mymod/internal/actions/deep/nested/pkg",
			},
			want: goastutil.GoPackageAliasWithStem("pkg", "mymod/internal/actions/deep/nested/pkg") + ".Output",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			naming := newActionNaming([]annotator_dto.ActionSpec{
				makeTestActionSpec("user.create", "mymod/actions/user", "user", "CreateAction", "POST"),
			})

			assert.Equal(t, tt.want, wrapperQualifiedTypeName(&naming, tt.typeSpec))
		})
	}
}

func TestEmitWrappers_SSEActionEmitsBindFunction(t *testing.T) {
	t.Parallel()

	specs := []annotator_dto.ActionSpec{
		{
			Name:        "stream.Export",
			PackagePath: "mymod/actions/stream",
			PackageName: "stream",
			StructName:  "ExportAction",
			HTTPMethod:  "POST",
			HasError:    true,
			HasSSE:      true,
			CallParams: []annotator_dto.ParamSpec{
				{
					Name:     "input",
					JSONName: "input",
					GoType:   "stream.ExportInput",
					Struct: &annotator_dto.TypeSpec{
						Name:        "ExportInput",
						PackagePath: "mymod/actions/stream",
						PackageName: "stream",
					},
				},
			},
		},
	}

	output, err := NewActionWrapperEmitter().EmitWrappers(context.Background(), specs)
	require.NoError(t, err)

	source := string(output)
	assert.Contains(t, source, "func bindStreamExport(ctx context.Context, action any, argsMap map[string]any) error")
	assert.Contains(t, source, "piko.SetActionInput(action, input)")
	assert.Contains(t, source, "\t\treturn err\n", "the bind function must return its error alone")
	assert.Contains(t, source, "return nil, err", "the invoke wrapper keeps its two-result return")
	assert.Contains(t, source, `"piko.sh/piko"`, "SetActionInput requires the piko import")

	_, parseErr := parser.ParseFile(token.NewFileSet(), "wrappers.go", source, parser.AllErrors)
	require.NoError(t, parseErr)
}

func TestEmitWrappers_NonSSEActionEmitsNoBindFunction(t *testing.T) {
	t.Parallel()

	specs := []annotator_dto.ActionSpec{
		{
			Name:        "user.Create",
			PackagePath: "mymod/actions/user",
			PackageName: "user",
			StructName:  "CreateAction",
			HTTPMethod:  "POST",
			HasError:    true,
		},
	}

	output, err := NewActionWrapperEmitter().EmitWrappers(context.Background(), specs)
	require.NoError(t, err)

	assert.NotContains(t, string(output), "func bindUserCreate")
}

func TestEmitWrappers_SSEActionWithoutParamsStillBinds(t *testing.T) {
	t.Parallel()

	specs := []annotator_dto.ActionSpec{
		{
			Name:        "ping.Simple",
			PackagePath: "mymod/actions/ping",
			PackageName: "ping",
			StructName:  "SimpleAction",
			HTTPMethod:  "GET",
			HasError:    true,
			HasSSE:      true,
		},
	}

	output, err := NewActionWrapperEmitter().EmitWrappers(context.Background(), specs)
	require.NoError(t, err)

	source := string(output)
	assert.Contains(t, source, "func bindPingSimple")
	assert.Contains(t, source, "piko.SetActionInput(action)")

	_, parseErr := parser.ParseFile(token.NewFileSet(), "wrappers.go", source, parser.AllErrors)
	require.NoError(t, parseErr)
}

func makeTypePackageSpec(paramName string, optional bool) annotator_dto.ActionSpec {
	spec := makeTestActionSpec("order.Create", "mymod/actions/order", "order", "CreateAction", "POST")
	spec.CallParams = []annotator_dto.ParamSpec{
		{
			Name:     paramName,
			GoType:   "CreateInput",
			JSONName: "input",
			Optional: optional,
			Struct: &annotator_dto.TypeSpec{
				Name:        "CreateInput",
				PackagePath: testTypePackagePath,
				PackageName: "contracts",
			},
		},
	}
	return spec
}

func TestEmitWrappers_TypePackageWithoutActionsIsImported(t *testing.T) {
	t.Parallel()

	result, err := NewActionWrapperEmitter().EmitWrappers(context.Background(), []annotator_dto.ActionSpec{makeTypePackageSpec("input", false)})
	require.NoError(t, err)

	output := string(result)
	assert.Contains(t, output, testTypePackageAlias+` "`+testTypePackagePath+`"`,
		"a parameter type's package must be imported even though it holds no action")
	assert.Contains(t, output, "var input "+testTypePackageAlias+".CreateInput")
	assert.NotContains(t, output, "var input contracts.CreateInput",
		"the bare package name is not bound by any import in the generated file")

	_, parseErr := parser.ParseFile(token.NewFileSet(), "wrappers.go", output, parser.AllErrors)
	require.NoError(t, parseErr, "generated code should be valid Go:\n%s", output)
}

func TestEmitWrappers_UnreferencedTypePackageIsNotImported(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		spec annotator_dto.ActionSpec
	}{
		{
			name: "return type is never named by a wrapper",
			spec: func() annotator_dto.ActionSpec {
				spec := makeTestActionSpec("order.Create", "mymod/actions/order", "order", "CreateAction", "POST")
				spec.ReturnType = &annotator_dto.TypeSpec{
					Name:        "CreateOutput",
					PackagePath: testTypePackagePath,
					PackageName: "contracts",
				}
				return spec
			}(),
		},
		{
			name: "blank optional parameter is passed as nil",
			spec: makeTypePackageSpec(blankParamName, true),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := NewActionWrapperEmitter().EmitWrappers(context.Background(), []annotator_dto.ActionSpec{tt.spec})
			require.NoError(t, err)

			output := string(result)
			assert.NotContains(t, output, testTypePackagePath,
				"a package the wrapper file never references must not be imported")

			_, parseErr := parser.ParseFile(token.NewFileSet(), "wrappers.go", output, parser.AllErrors)
			require.NoError(t, parseErr, "generated code should be valid Go:\n%s", output)
		})
	}
}

func TestEmitRegistryAndWrappers_ImportEveryAliasTheyUse(t *testing.T) {
	t.Parallel()

	inputOnly := makeTypePackageSpec("input", false)
	outputOnly := makeTestActionSpec("report.Fetch", "mymod/actions/report", "report", "FetchAction", "POST")
	outputOnly.ReturnType = &annotator_dto.TypeSpec{
		Name:        "FetchOutput",
		PackagePath: "mymod/pkg/results",
		PackageName: "results",
	}
	specs := []annotator_dto.ActionSpec{inputOnly, outputOnly}

	registryCode, err := NewActionRegistryEmitter().EmitRegistry(context.Background(), specs)
	require.NoError(t, err)
	wrapperCode, err := NewActionWrapperEmitter().EmitWrappers(context.Background(), specs)
	require.NoError(t, err)

	fset := token.NewFileSet()
	registryFile, err := parser.ParseFile(fset, "registry.go", registryCode, parser.AllErrors)
	require.NoError(t, err)
	wrapperFile, err := parser.ParseFile(fset, "wrappers.go", wrapperCode, parser.AllErrors)
	require.NoError(t, err)

	for path, alias := range actionImportAliases(registryFile) {
		assert.Contains(t, actionUsedQualifiers(registryFile), alias,
			"registry.go imports %s but never references it", path)
	}
	for path, alias := range actionImportAliases(wrapperFile) {
		assert.Contains(t, actionUsedQualifiers(wrapperFile), alias,
			"wrappers.go imports %s but never references it", path)
	}

	assert.Contains(t, actionImportAliases(registryFile), "mymod/pkg/results",
		"the registry pretouches the return type, so it must import its package")
	assert.NotContains(t, actionImportAliases(wrapperFile), "mymod/pkg/results",
		"no wrapper names the return type, so importing its package would leave it unused")
}

func actionUsedQualifiers(file *ast.File) map[string]struct{} {
	qualifiers := make(map[string]struct{})
	ast.Inspect(file, func(node ast.Node) bool {
		selector, ok := node.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if ident, ok := selector.X.(*ast.Ident); ok {
			qualifiers[ident.Name] = struct{}{}
		}
		return true
	})
	return qualifiers
}

func TestActionParamLocalsClaimNamesTheWrapperBodyUses(t *testing.T) {
	t.Parallel()

	spec := &annotator_dto.ActionSpec{
		Name:        "report.Export",
		PackageName: "report",
		PackagePath: "app/actions/report",
		StructName:  "ExportAction",
		CallParams: []annotator_dto.ParamSpec{
			{Name: "ctx", JSONName: "ctx", GoType: goTypeString},
			{Name: "err", JSONName: "err", GoType: goTypeString},
			{Name: "result", JSONName: "result", GoType: goTypeString},
			{Name: "_", JSONName: "", GoType: goTypeString},
			{Name: "title", JSONName: "title", GoType: goTypeString},
		},
	}

	locals := actionParamLocals(spec)

	require.Len(t, locals, len(spec.CallParams))
	assert.NotEqual(t, "ctx", locals[0], "a parameter named ctx would redeclare the wrapper's own ctx")
	assert.NotEqual(t, "err", locals[1])
	assert.NotEqual(t, "result", locals[2])
	assert.Empty(t, locals[3], "a blank parameter is never extracted")
	assert.Equal(t, "title", locals[4], "an ordinary parameter keeps its own name")

	claimed := map[string]int{}
	for _, name := range locals {
		if name != "" {
			claimed[name]++
		}
	}
	for name, count := range claimed {
		assert.Equal(t, 1, count, "local %q was claimed more than once", name)
	}
}

func TestEmitWrappersCompilesWhenParametersShadowWrapperLocals(t *testing.T) {
	t.Parallel()

	emitter := NewActionWrapperEmitter()
	source, err := emitter.EmitWrappers(t.Context(), []annotator_dto.ActionSpec{{
		Name:        "report.Export",
		PackageName: "report",
		PackagePath: "app/actions/report",
		StructName:  "ExportAction",
		HasError:    true,
		CallParams: []annotator_dto.ParamSpec{
			{Name: "ctx", JSONName: "ctx", GoType: goTypeString},
			{Name: "result", JSONName: "result", GoType: goTypeString},
		},
	}})

	require.NoError(t, err)
	fset := token.NewFileSet()
	_, parseErr := parser.ParseFile(fset, "wrappers.go", source, parser.AllErrors)
	require.NoError(t, parseErr, "generated wrappers must parse:\n%s", source)
}
