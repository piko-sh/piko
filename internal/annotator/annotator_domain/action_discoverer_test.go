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

package annotator_domain

import (
	"go/ast"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
	"piko.sh/piko/internal/annotator/annotator_dto"
)

func TestKebabToCamel(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple kebab case",
			input:    "delete-user",
			expected: "deleteUser",
		},
		{
			name:     "multiple parts",
			input:    "get-all-users",
			expected: "getAllUsers",
		},
		{
			name:     "no hyphens",
			input:    "deleteuser",
			expected: "deleteuser",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "single character parts",
			input:    "a-b-c",
			expected: "aBC",
		},
		{
			name:     "trailing hyphen",
			input:    "delete-",
			expected: "delete",
		},
		{
			name:     "leading hyphen",
			input:    "-user",
			expected: "User",
		},
		{
			name:     "consecutive hyphens",
			input:    "delete--user",
			expected: "deleteUser",
		},
		{
			name:     "uppercase preserved after hyphen",
			input:    "delete-User",
			expected: "deleteUser",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := kebabToCamel(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestKebabToCamelInSegments(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single segment with kebab",
			input:    "delete-user",
			expected: "deleteUser",
		},
		{
			name:     "multiple segments with kebab",
			input:    "email.delete-user",
			expected: "email.deleteUser",
		},
		{
			name:     "multiple segments all kebab",
			input:    "my-app.delete-user",
			expected: "myApp.deleteUser",
		},
		{
			name:     "no kebab case",
			input:    "email.contact",
			expected: "email.contact",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "single segment no kebab",
			input:    "contact",
			expected: "contact",
		},
		{
			name:     "three segments",
			input:    "api.user-management.delete-account",
			expected: "api.userManagement.deleteAccount",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := kebabToCamelInSegments(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestActionNameToTSFunction(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple action name",
			input:    "email.contact",
			expected: "emailContact",
		},
		{
			name:     "single segment",
			input:    "contact",
			expected: "contact",
		},
		{
			name:     "multiple segments",
			input:    "api.user.delete",
			expected: "apiUserDelete",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "already camelCase segment",
			input:    "actions.DeleteUser",
			expected: "actionsDeleteUser",
		},
		{
			name:     "lowercase segments",
			input:    "actions.deleteuser",
			expected: "actionsDeleteuser",
		},
		{
			name:     "consecutive dots create empty parts",
			input:    "api..delete",
			expected: "apiDelete",
		},
		{
			name:     "trailing dot",
			input:    "api.",
			expected: "api",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := actionNameToTSFunction(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestStructNameToActionName(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		structName  string
		packageName string
		expected    string
	}{
		{
			name:        "struct with Action suffix",
			structName:  "DeleteUserAction",
			packageName: "actions",
			expected:    "actions.DeleteUser",
		},
		{
			name:        "struct without Action suffix",
			structName:  "Contact",
			packageName: "email",
			expected:    "email.Contact",
		},
		{
			name:        "only Action",
			structName:  "Action",
			packageName: "test",
			expected:    "test.",
		},
		{
			name:        "nested package",
			structName:  "FieldAddAction",
			packageName: "blueprint",
			expected:    "blueprint.FieldAdd",
		},
		{
			name:        "empty struct name",
			structName:  "",
			packageName: "pkg",
			expected:    "pkg.",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := structNameToActionName(tc.structName, tc.packageName)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestEmbedsActionMetadata(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		structType *ast.StructType
		name       string
		expected   bool
	}{
		{
			name: "embeds piko.ActionMetadata",
			structType: &ast.StructType{
				Fields: &ast.FieldList{
					List: []*ast.Field{
						{
							Names: nil,
							Type: &ast.SelectorExpr{
								X:   ast.NewIdent("piko"),
								Sel: ast.NewIdent("ActionMetadata"),
							},
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "does not embed ActionMetadata",
			structType: &ast.StructType{
				Fields: &ast.FieldList{
					List: []*ast.Field{
						{
							Names: []*ast.Ident{ast.NewIdent("Name")},
							Type:  ast.NewIdent("string"),
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "named field with piko.ActionMetadata type",
			structType: &ast.StructType{
				Fields: &ast.FieldList{
					List: []*ast.Field{
						{
							Names: []*ast.Ident{ast.NewIdent("meta")},
							Type: &ast.SelectorExpr{
								X:   ast.NewIdent("piko"),
								Sel: ast.NewIdent("ActionMetadata"),
							},
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "wrong package name",
			structType: &ast.StructType{
				Fields: &ast.FieldList{
					List: []*ast.Field{
						{
							Names: nil,
							Type: &ast.SelectorExpr{
								X:   ast.NewIdent("other"),
								Sel: ast.NewIdent("ActionMetadata"),
							},
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "wrong type name",
			structType: &ast.StructType{
				Fields: &ast.FieldList{
					List: []*ast.Field{
						{
							Names: nil,
							Type: &ast.SelectorExpr{
								X:   ast.NewIdent("piko"),
								Sel: ast.NewIdent("SomethingElse"),
							},
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "empty struct",
			structType: &ast.StructType{
				Fields: &ast.FieldList{
					List: []*ast.Field{},
				},
			},
			expected: false,
		},
		{
			name: "multiple fields with embed",
			structType: &ast.StructType{
				Fields: &ast.FieldList{
					List: []*ast.Field{
						{
							Names: []*ast.Ident{ast.NewIdent("Name")},
							Type:  ast.NewIdent("string"),
						},
						{
							Names: nil,
							Type: &ast.SelectorExpr{
								X:   ast.NewIdent("piko"),
								Sel: ast.NewIdent("ActionMetadata"),
							},
						},
						{
							Names: []*ast.Ident{ast.NewIdent("ID")},
							Type:  ast.NewIdent("int"),
						},
					},
				},
			},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := embedsActionMetadata(tc.structType)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestExtractStructDocComment(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		genDecl  *ast.GenDecl
		typeSpec *ast.TypeSpec
		expected string
	}{
		{
			name: "type spec has doc comment",
			genDecl: &ast.GenDecl{
				Tok:   token.TYPE,
				Specs: []ast.Spec{},
				Doc:   nil,
			},
			typeSpec: &ast.TypeSpec{
				Name: ast.NewIdent("MyAction"),
				Doc: &ast.CommentGroup{
					List: []*ast.Comment{
						{Text: "// MyAction handles the action."},
					},
				},
			},
			expected: "MyAction handles the action.",
		},
		{
			name: "gen decl has doc comment single spec",
			genDecl: &ast.GenDecl{
				Tok: token.TYPE,
				Specs: []ast.Spec{
					&ast.TypeSpec{Name: ast.NewIdent("MyAction")},
				},
				Doc: &ast.CommentGroup{
					List: []*ast.Comment{
						{Text: "// MyAction is documented here."},
					},
				},
			},
			typeSpec: &ast.TypeSpec{
				Name: ast.NewIdent("MyAction"),
				Doc:  nil,
			},
			expected: "MyAction is documented here.",
		},
		{
			name: "gen decl doc ignored with multiple specs",
			genDecl: &ast.GenDecl{
				Tok: token.TYPE,
				Specs: []ast.Spec{
					&ast.TypeSpec{Name: ast.NewIdent("Type1")},
					&ast.TypeSpec{Name: ast.NewIdent("Type2")},
				},
				Doc: &ast.CommentGroup{
					List: []*ast.Comment{
						{Text: "// This comment applies to multiple types."},
					},
				},
			},
			typeSpec: &ast.TypeSpec{
				Name: ast.NewIdent("Type1"),
				Doc:  nil,
			},
			expected: "",
		},
		{
			name: "no doc comments",
			genDecl: &ast.GenDecl{
				Tok:   token.TYPE,
				Specs: []ast.Spec{},
				Doc:   nil,
			},
			typeSpec: &ast.TypeSpec{
				Name: ast.NewIdent("MyAction"),
				Doc:  nil,
			},
			expected: "",
		},
		{
			name: "type spec doc takes precedence",
			genDecl: &ast.GenDecl{
				Tok: token.TYPE,
				Specs: []ast.Spec{
					&ast.TypeSpec{Name: ast.NewIdent("MyAction")},
				},
				Doc: &ast.CommentGroup{
					List: []*ast.Comment{
						{Text: "// GenDecl comment"},
					},
				},
			},
			typeSpec: &ast.TypeSpec{
				Name: ast.NewIdent("MyAction"),
				Doc: &ast.CommentGroup{
					List: []*ast.Comment{
						{Text: "// TypeSpec comment"},
					},
				},
			},
			expected: "TypeSpec comment",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := extractStructDocComment(nil, tc.genDecl, tc.typeSpec)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestBuildFilePathInfo(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name               string
		filePath           string
		baseDir            string
		moduleName         string
		expectedRel        string
		expectedPackageDir string
	}{
		{
			name:               "simple case",
			filePath:           "/project/actions/contact.go",
			baseDir:            "/project",
			moduleName:         "github.com/example/myapp",
			expectedRel:        "actions/contact.go",
			expectedPackageDir: "github.com/example/myapp/actions",
		},
		{
			name:               "nested path",
			filePath:           "/project/actions/email/contact.go",
			baseDir:            "/project",
			moduleName:         "github.com/example/myapp",
			expectedRel:        "actions/email/contact.go",
			expectedPackageDir: "github.com/example/myapp/actions/email",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			info := buildFilePathInfo(tc.filePath, tc.baseDir, tc.moduleName)
			assert.Equal(t, tc.expectedRel, info.relPath)
			assert.Equal(t, tc.expectedPackageDir, info.packagePath)
		})
	}
}

func TestCandidateToDefinition(t *testing.T) {
	t.Parallel()

	candidate := &annotator_dto.ActionCandidate{
		FilePath:       "/project/actions/delete-user.go",
		RelativePath:   "actions/delete-user.go",
		PackagePath:    "github.com/example/myapp/actions",
		PackageName:    "actions",
		StructName:     "DeleteUserAction",
		ActionName:     "actions.DeleteUser",
		TSFunctionName: "actionsDeleteUser",
		DocComment:     "DeleteUserAction deletes a user.",
	}

	definition := candidateToDefinition(candidate)

	assert.Equal(t, "actions.DeleteUser", definition.Name)
	assert.Equal(t, "actionsDeleteUser", definition.TSFunctionName)
	assert.Equal(t, "actions/delete-user.go", definition.FilePath)
	assert.Equal(t, "github.com/example/myapp/actions", definition.PackagePath)
	assert.Equal(t, "DeleteUserAction", definition.StructName)
	assert.Equal(t, "actions", definition.PackageName)
	assert.Equal(t, "DeleteUserAction deletes a user.", definition.Description)
	assert.Equal(t, "POST", definition.HTTPMethod)
}

func TestExtractActionCandidates(t *testing.T) {
	t.Parallel()

	t.Run("extracts candidate from file with action struct", func(t *testing.T) {
		t.Parallel()

		file := &ast.File{
			Name: ast.NewIdent("email"),
			Decls: []ast.Decl{
				&ast.GenDecl{
					Tok: token.TYPE,
					Specs: []ast.Spec{
						&ast.TypeSpec{
							Name: ast.NewIdent("ContactAction"),
							Type: &ast.StructType{
								Fields: &ast.FieldList{
									List: []*ast.Field{
										{
											Names: nil,
											Type: &ast.SelectorExpr{
												X:   ast.NewIdent("piko"),
												Sel: ast.NewIdent("ActionMetadata"),
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}
		info := filePathInfo{
			filePath:    "/project/actions/email/contact.go",
			relPath:     "actions/email/contact.go",
			packagePath: "github.com/example/myapp/actions/email",
		}

		candidates := extractActionCandidates(token.NewFileSet(), file, info)

		assert.Len(t, candidates, 1)
		assert.Equal(t, "ContactAction", candidates[0].StructName)
		assert.Equal(t, "email.Contact", candidates[0].ActionName)
		assert.Equal(t, "emailContact", candidates[0].TSFunctionName)
		assert.Equal(t, "email", candidates[0].PackageName)
	})

	t.Run("returns empty for file with no action structs", func(t *testing.T) {
		t.Parallel()

		file := &ast.File{
			Name: ast.NewIdent("models"),
			Decls: []ast.Decl{
				&ast.GenDecl{
					Tok: token.TYPE,
					Specs: []ast.Spec{
						&ast.TypeSpec{
							Name: ast.NewIdent("User"),
							Type: &ast.StructType{
								Fields: &ast.FieldList{
									List: []*ast.Field{
										{
											Names: []*ast.Ident{ast.NewIdent("Name")},
											Type:  ast.NewIdent("string"),
										},
									},
								},
							},
						},
					},
				},
			},
		}
		info := filePathInfo{
			filePath:    "/project/models/user.go",
			relPath:     "models/user.go",
			packagePath: "github.com/example/myapp/models",
		}

		candidates := extractActionCandidates(token.NewFileSet(), file, info)

		assert.Empty(t, candidates)
	})

	t.Run("extracts multiple candidates", func(t *testing.T) {
		t.Parallel()

		file := &ast.File{
			Name: ast.NewIdent("user"),
			Decls: []ast.Decl{
				&ast.GenDecl{
					Tok: token.TYPE,
					Specs: []ast.Spec{
						&ast.TypeSpec{
							Name: ast.NewIdent("CreateAction"),
							Type: &ast.StructType{
								Fields: &ast.FieldList{
									List: []*ast.Field{
										{
											Names: nil,
											Type: &ast.SelectorExpr{
												X:   ast.NewIdent("piko"),
												Sel: ast.NewIdent("ActionMetadata"),
											},
										},
									},
								},
							},
						},
						&ast.TypeSpec{
							Name: ast.NewIdent("DeleteAction"),
							Type: &ast.StructType{
								Fields: &ast.FieldList{
									List: []*ast.Field{
										{
											Names: nil,
											Type: &ast.SelectorExpr{
												X:   ast.NewIdent("piko"),
												Sel: ast.NewIdent("ActionMetadata"),
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}
		info := filePathInfo{
			filePath:    "/project/actions/user.go",
			relPath:     "actions/user.go",
			packagePath: "github.com/example/myapp/actions",
		}

		candidates := extractActionCandidates(token.NewFileSet(), file, info)

		assert.Len(t, candidates, 2)
		assert.Equal(t, "CreateAction", candidates[0].StructName)
		assert.Equal(t, "DeleteAction", candidates[1].StructName)
	})

	t.Run("ignores function declarations", func(t *testing.T) {
		t.Parallel()

		file := &ast.File{
			Name: ast.NewIdent("helpers"),
			Decls: []ast.Decl{
				&ast.FuncDecl{
					Name: ast.NewIdent("Helper"),
					Type: &ast.FuncType{},
				},
			},
		}
		info := filePathInfo{
			filePath:    "/project/helpers.go",
			relPath:     "helpers.go",
			packagePath: "github.com/example/myapp",
		}

		candidates := extractActionCandidates(token.NewFileSet(), file, info)

		assert.Empty(t, candidates)
	})

	t.Run("ignores const declarations", func(t *testing.T) {
		t.Parallel()

		file := &ast.File{
			Name: ast.NewIdent("constants"),
			Decls: []ast.Decl{
				&ast.GenDecl{
					Tok: token.CONST,
					Specs: []ast.Spec{
						&ast.ValueSpec{
							Names:  []*ast.Ident{ast.NewIdent("MaxRetries")},
							Values: []ast.Expr{&ast.BasicLit{Value: "3"}},
						},
					},
				},
			},
		}
		info := filePathInfo{
			filePath:    "/project/constants.go",
			relPath:     "constants.go",
			packagePath: "github.com/example/myapp",
		}

		candidates := extractActionCandidates(token.NewFileSet(), file, info)

		assert.Empty(t, candidates)
	})
}

func TestTryExtractActionCandidate(t *testing.T) {
	t.Parallel()

	t.Run("extracts candidate from valid action struct", func(t *testing.T) {
		t.Parallel()

		file := &ast.File{Name: ast.NewIdent("actions")}
		genDecl := &ast.GenDecl{
			Tok: token.TYPE,
			Specs: []ast.Spec{
				&ast.TypeSpec{Name: ast.NewIdent("SendEmailAction")},
			},
		}
		spec := &ast.TypeSpec{
			Name: ast.NewIdent("SendEmailAction"),
			Type: &ast.StructType{
				Fields: &ast.FieldList{
					List: []*ast.Field{
						{
							Names: nil,
							Type: &ast.SelectorExpr{
								X:   ast.NewIdent("piko"),
								Sel: ast.NewIdent("ActionMetadata"),
							},
						},
					},
				},
			},
		}
		info := filePathInfo{
			filePath:    "/project/actions/email.go",
			relPath:     "actions/email.go",
			packagePath: "github.com/example/myapp/actions",
		}

		candidate := tryExtractActionCandidate(token.NewFileSet(), file, genDecl, spec, info)

		assert.NotNil(t, candidate)
		assert.Equal(t, "SendEmailAction", candidate.StructName)
		assert.Equal(t, "actions.SendEmail", candidate.ActionName)
		assert.Equal(t, "actionsSendEmail", candidate.TSFunctionName)
	})

	t.Run("returns nil for non-TypeSpec", func(t *testing.T) {
		t.Parallel()

		file := &ast.File{Name: ast.NewIdent("pkg")}
		genDecl := &ast.GenDecl{Tok: token.CONST}
		spec := &ast.ValueSpec{
			Names: []*ast.Ident{ast.NewIdent("Constant")},
		}
		info := filePathInfo{}

		candidate := tryExtractActionCandidate(token.NewFileSet(), file, genDecl, spec, info)

		assert.Nil(t, candidate)
	})

	t.Run("returns nil for non-struct type", func(t *testing.T) {
		t.Parallel()

		file := &ast.File{Name: ast.NewIdent("pkg")}
		genDecl := &ast.GenDecl{Tok: token.TYPE}
		spec := &ast.TypeSpec{
			Name: ast.NewIdent("MyAlias"),
			Type: ast.NewIdent("string"),
		}
		info := filePathInfo{}

		candidate := tryExtractActionCandidate(token.NewFileSet(), file, genDecl, spec, info)

		assert.Nil(t, candidate)
	})

	t.Run("returns nil for struct without ActionMetadata", func(t *testing.T) {
		t.Parallel()

		file := &ast.File{Name: ast.NewIdent("models")}
		genDecl := &ast.GenDecl{Tok: token.TYPE}
		spec := &ast.TypeSpec{
			Name: ast.NewIdent("User"),
			Type: &ast.StructType{
				Fields: &ast.FieldList{
					List: []*ast.Field{
						{
							Names: []*ast.Ident{ast.NewIdent("Name")},
							Type:  ast.NewIdent("string"),
						},
					},
				},
			},
		}
		info := filePathInfo{}

		candidate := tryExtractActionCandidate(token.NewFileSet(), file, genDecl, spec, info)

		assert.Nil(t, candidate)
	})

	t.Run("includes doc comment from TypeSpec", func(t *testing.T) {
		t.Parallel()

		file := &ast.File{Name: ast.NewIdent("actions")}
		genDecl := &ast.GenDecl{Tok: token.TYPE, Specs: []ast.Spec{}}
		spec := &ast.TypeSpec{
			Name: ast.NewIdent("DocumentedAction"),
			Doc: &ast.CommentGroup{
				List: []*ast.Comment{
					{Text: "// DocumentedAction does something useful."},
				},
			},
			Type: &ast.StructType{
				Fields: &ast.FieldList{
					List: []*ast.Field{
						{
							Names: nil,
							Type: &ast.SelectorExpr{
								X:   ast.NewIdent("piko"),
								Sel: ast.NewIdent("ActionMetadata"),
							},
						},
					},
				},
			},
		}
		info := filePathInfo{
			filePath:    "/project/actions/doc.go",
			relPath:     "actions/doc.go",
			packagePath: "github.com/example/myapp/actions",
		}

		candidate := tryExtractActionCandidate(token.NewFileSet(), file, genDecl, spec, info)

		assert.NotNil(t, candidate)
		assert.Equal(t, "DocumentedAction does something useful.", candidate.DocComment)
	})
}
