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

package driven_code_emitter_go_literal

import (
	"context"
	goast "go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/generator/generator_dto"
	"piko.sh/piko/internal/templater/templater_dto"
)

func TestNewEmitter(t *testing.T) {
	emitter := NewEmitter(context.Background())
	if emitter == nil {
		t.Fatal("NewEmitter returned nil")
	}

	var _ = emitter
}

func TestEmitCode_SimpleComponent(t *testing.T) {
	ctx := context.Background()
	emitter := NewEmitter(context.Background())

	result := createMinimalAnnotationResult("test_hash")
	request := generator_dto.GenerateRequest{
		SourcePath:             "/test/component.pk",
		PackageName:            "testpkg",
		HashedName:             "test_hash",
		CanonicalGoPackagePath: "test.com/pkg",
		BaseDir:                "/test",
		IsPage:                 true,
	}

	code, diagnostics, err := emitter.EmitCode(ctx, result, request)

	if err != nil {
		t.Fatalf("EmitCode failed: %v", err)
	}

	if len(code) == 0 {
		t.Error("EmitCode returned empty code")
	}

	_ = diagnostics

	fset := token.NewFileSet()
	_, parseErr := parser.ParseFile(fset, "generated.go", code, parser.AllErrors)
	if parseErr != nil {
		t.Errorf("Generated code is not valid Go: %v\nCode:\n%s", parseErr, string(code))
	}

	codeString := string(code)
	if !strings.Contains(codeString, "package testpkg") {
		t.Error("Generated code missing correct package declaration")
	}

	if !strings.Contains(codeString, "func BuildAST(") {
		t.Error("Generated code missing BuildAST function")
	}
}

func TestEmitCode_MissingComponent(t *testing.T) {
	ctx := context.Background()
	emitter := NewEmitter(context.Background())

	result := createMinimalAnnotationResult("existing_hash")
	request := generator_dto.GenerateRequest{
		SourcePath:             "/test/component.pk",
		PackageName:            "testpkg",
		HashedName:             "missing_hash",
		CanonicalGoPackagePath: "test.com/pkg",
		BaseDir:                "/test",
		IsPage:                 true,
	}

	_, _, err := emitter.EmitCode(ctx, result, request)

	if err == nil {
		t.Fatal("EmitCode should fail with missing component")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Error message should mention 'not found', got: %v", err)
	}
}

func TestEmitCode_WithAnnotatedAST(t *testing.T) {
	ctx := context.Background()
	emitter := NewEmitter(context.Background())

	result := createAnnotationResultWithAST("test_hash")
	request := generator_dto.GenerateRequest{
		SourcePath:             "/test/component.pk",
		PackageName:            "testpkg",
		HashedName:             "test_hash",
		CanonicalGoPackagePath: "test.com/pkg",
		BaseDir:                "/test",
		IsPage:                 true,
	}

	code, diagnostics, err := emitter.EmitCode(ctx, result, request)

	if err != nil {
		t.Fatalf("EmitCode failed: %v", err)
	}

	if len(code) == 0 {
		t.Error("EmitCode returned empty code")
	}

	_ = diagnostics

	codeString := string(code)
	if !strings.Contains(codeString, "rootAST") {
		t.Error("Generated code should contain rootAST variable")
	}
}

func TestValidateMainComponent(t *testing.T) {
	tests := []struct {
		result      *annotator_dto.AnnotationResult
		name        string
		hashedName  string
		expectError bool
	}{
		{
			name:        "valid component",
			hashedName:  "test_hash",
			result:      createMinimalAnnotationResult("test_hash"),
			expectError: false,
		},
		{
			name:        "missing component",
			hashedName:  "missing",
			result:      createMinimalAnnotationResult("other"),
			expectError: true,
		},
		{
			name:       "nil component in map",
			hashedName: "nil_component",
			result: &annotator_dto.AnnotationResult{
				VirtualModule: &annotator_dto.VirtualModule{
					ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
						"nil_component": nil,
					},
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			em := &emitter{}
			component, err := em.validateMainComponent(tt.hashedName, tt.result)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				}
				if component != nil {
					t.Error("Expected nil component on error")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if component == nil {
					t.Error("Expected non-nil component")
				}
			}
		})
	}
}

func TestDiagnosticError(t *testing.T) {
	diagnostic := ast_domain.NewDiagnostic(
		ast_domain.Error,
		"test error message",
		"context",
		ast_domain.Location{Line: 1, Column: 1},
		"/test/path",
	)

	err := &diagnosticError{diagnostic: diagnostic}

	if err.Error() != "test error message" {
		t.Errorf("Expected 'test error message', got: %s", err.Error())
	}
}

func TestBuildImportBlock(t *testing.T) {
	em := &emitter{
		ctx: NewEmitterContext(),
		config: EmitterConfig{
			CanonicalGoPackagePath: "test.com/pkg",
		},
	}

	em.ctx.requiredImports["fmt"] = ""
	em.ctx.requiredImports["strings"] = ""

	result := createMinimalAnnotationResult("test")
	mainComponent := result.VirtualModule.ComponentsByHash["test"]

	genDecl := em.buildImportBlock(result, mainComponent)

	if genDecl == nil {
		t.Fatal("buildImportBlock returned nil")
	}

	if genDecl.Tok != token.IMPORT {
		t.Error("Import declaration has wrong token type")
	}

	if len(genDecl.Specs) == 0 {
		t.Error("Import declaration has no specs")
	}

	hasRuntime := false
	for _, spec := range genDecl.Specs {
		impSpec := requireImportSpec(t, spec, "import spec")
		if strings.Contains(impSpec.Path.Value, "runtime") {
			hasRuntime = true
		}
	}

	if !hasRuntime {
		t.Error("Import block should include runtime package")
	}
}

func TestBuildImportBlock_Empty(t *testing.T) {
	em := &emitter{
		ctx: NewEmitterContext(),
		config: EmitterConfig{
			CanonicalGoPackagePath: "test.com/pkg",
		},
	}

	result := createMinimalAnnotationResult("test")
	mainComponent := result.VirtualModule.ComponentsByHash["test"]

	importDecl := em.buildImportBlock(result, mainComponent)

	if importDecl == nil {
		t.Error("buildImportBlock should return standard imports even with no custom imports")
	}
}

func TestAddImport(t *testing.T) {
	tests := []struct {
		name           string
		canonicalPath  string
		alias          string
		currentPackage string
		shouldAdd      bool
	}{
		{
			name:           "add valid import",
			canonicalPath:  "fmt",
			alias:          "",
			currentPackage: "test.com/pkg",
			shouldAdd:      true,
		},
		{
			name:           "skip current package",
			canonicalPath:  "test.com/pkg",
			alias:          "",
			currentPackage: "test.com/pkg",
			shouldAdd:      false,
		},
		{
			name:           "skip empty path",
			canonicalPath:  "",
			alias:          "",
			currentPackage: "test.com/pkg",
			shouldAdd:      false,
		},
		{
			name:           "add with alias",
			canonicalPath:  "other.com/pkg",
			alias:          "otherpkg",
			currentPackage: "test.com/pkg",
			shouldAdd:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			em := &emitter{
				ctx: NewEmitterContext(),
				config: EmitterConfig{
					CanonicalGoPackagePath: tt.currentPackage,
				},
			}

			em.addImport(tt.canonicalPath, tt.alias)

			_, exists := em.ctx.requiredImports[tt.canonicalPath]
			if tt.shouldAdd && !exists {
				t.Error("Import should have been added")
			}
			if !tt.shouldAdd && exists {
				t.Error("Import should not have been added")
			}
		})
	}
}

func TestAddImport_PreserveExistingAlias(t *testing.T) {
	em := &emitter{
		ctx: NewEmitterContext(),
		config: EmitterConfig{
			CanonicalGoPackagePath: "test.com/pkg",
		},
	}

	em.addImport("other.com/pkg", "original")
	em.addImport("other.com/pkg", "")

	alias := em.ctx.requiredImports["other.com/pkg"]
	if alias != "original" {
		t.Errorf("Expected alias 'original', got: %s", alias)
	}
}

func TestBuildRegistrationInitFunction(t *testing.T) {
	em := &emitter{}

	result := createMinimalAnnotationResult("test")

	initFunc, err := em.buildRegistrationInitFunction(result)

	if err != nil {
		t.Fatalf("buildRegistrationInitFunction failed: %v", err)
	}

	if initFunc == nil {
		t.Fatal("Expected non-nil init function")
	}

	funcDecl, ok := initFunc.(*goast.FuncDecl)
	if !ok {
		t.Fatalf("Expected *goast.FuncDecl, got %T", initFunc)
	}

	if funcDecl.Name.Name != "init" {
		t.Errorf("Expected function name 'init', got: %s", funcDecl.Name.Name)
	}

	if len(funcDecl.Body.List) == 0 {
		t.Error("Init function should have registration calls")
	}
}

func TestBuildRegistrationInitFunction_WithOptionalFunctions(t *testing.T) {
	em := &emitter{}

	result := createMinimalAnnotationResult("test")
	mainComp := result.VirtualModule.ComponentsByHash["test"]

	mainComp.Source.Script.HasCachePolicy = true
	mainComp.Source.Script.CachePolicyFuncName = "CachePolicy"
	mainComp.Source.Script.HasMiddleware = true
	mainComp.Source.Script.MiddlewaresFuncName = "Middlewares"
	mainComp.Source.Script.HasSupportedLocales = true
	mainComp.Source.Script.SupportedLocalesFuncName = "SupportedLocales"

	initFunc, err := em.buildRegistrationInitFunction(result)

	if err != nil {
		t.Fatalf("buildRegistrationInitFunction failed: %v", err)
	}

	funcDecl := requireFuncDecl(t, initFunc, "init function declaration")

	if len(funcDecl.Body.List) < 4 {
		t.Errorf("Expected at least 4 registration calls, got: %d", len(funcDecl.Body.List))
	}
}

func TestVerifyGeneratedCode(t *testing.T) {
	tests := []struct {
		name        string
		code        string
		expectError bool
	}{
		{
			name: "valid code",
			code: `package test
func main() {}`,
			expectError: false,
		},
		{
			name: "invalid code",
			code: `package test
func main( {}`,
			expectError: true,
		},
		{
			name:        "empty package",
			code:        ``,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := generator_dto.GenerateRequest{
				SourcePath: "/test/file.pk",
			}

			err := verifyGeneratedCode(request, []byte(tt.code))

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestBuildBoilerplateVarAcks(t *testing.T) {
	acks := buildBoilerplateVarAcks()

	if len(acks) == 0 {
		t.Fatal("buildBoilerplateVarAcks returned empty slice")
	}

	foundFmt := false
	foundRuntime := false

	for _, declaration := range acks {
		genDecl, ok := declaration.(*goast.GenDecl)
		if !ok {
			continue
		}

		for _, spec := range genDecl.Specs {
			valSpec, ok := spec.(*goast.ValueSpec)
			if !ok {
				continue
			}

			if len(valSpec.Values) > 0 {
				if selExpr, ok := valSpec.Values[0].(*goast.SelectorExpr); ok {
					if identifier, ok := selExpr.X.(*goast.Ident); ok {
						if identifier.Name == "fmt" {
							foundFmt = true
						}
						if identifier.Name == runtimePackageName {
							foundRuntime = true
						}
					}
				}
			}
		}
	}

	if !foundFmt {
		t.Error("Boilerplate should acknowledge fmt package")
	}
	if !foundRuntime {
		t.Error("Boilerplate should acknowledge runtime package")
	}
}

func TestNextFetcherName(t *testing.T) {
	em := &emitter{
		ctx: NewEmitterContext(),
	}

	name1 := em.nextFetcherName()
	name2 := em.nextFetcherName()

	if name1 == name2 {
		t.Error("nextFetcherName should return unique names")
	}

	if !strings.Contains(name1, "fetchCollection") {
		t.Errorf("Fetcher name should contain 'fetchCollection', got: %s", name1)
	}
}

func TestNextStaticVarName(t *testing.T) {
	em := &emitter{
		ctx: NewEmitterContext(),
	}

	name1 := em.nextStaticVarName()
	name2 := em.nextStaticVarName()

	if name1 == name2 {
		t.Error("nextStaticVarName should return unique names")
	}

	if !strings.Contains(name1, "staticNode_") {
		t.Errorf("Static var name should contain 'staticNode_', got: %s", name1)
	}
}

func createMinimalAnnotationResult(hashedName string) *annotator_dto.AnnotationResult {
	sourcePath := "/test/component.pk"
	return &annotator_dto.AnnotationResult{
		VirtualModule: &annotator_dto.VirtualModule{
			ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
				hashedName: {
					HashedName:             hashedName,
					PartialName:            "TestComponent",
					CanonicalGoPackagePath: "test.com/pkg",
					Source: &annotator_dto.ParsedComponent{
						SourcePath: sourcePath,
						Script: &annotator_dto.ParsedScript{
							PropsTypeExpression:      nil,
							HasCachePolicy:           false,
							HasMiddleware:            false,
							HasSupportedLocales:      false,
							CachePolicyFuncName:      "",
							MiddlewaresFuncName:      "",
							SupportedLocalesFuncName: "",
						},
					},
					RewrittenScriptAST: &goast.File{
						Name:  cachedIdent("testpkg"),
						Decls: []goast.Decl{},
					},
				},
			},
			Graph: &annotator_dto.ComponentGraph{
				PathToHashedName: map[string]string{
					sourcePath: hashedName,
				},
			},
		},
		AnnotatedAST: &ast_domain.TemplateAST{
			SourcePath: &sourcePath,
			RootNodes:  []*ast_domain.TemplateNode{},
		},
		UniqueInvocations: []*annotator_dto.PartialInvocation{},
		AssetRefs:         []templater_dto.AssetRef{},
		CustomTags:        []string{},
	}
}

func createAnnotationResultWithAST(hashedName string) *annotator_dto.AnnotationResult {
	result := createMinimalAnnotationResult(hashedName)

	sourcePath := "/test/component.pk"
	result.AnnotatedAST = &ast_domain.TemplateAST{
		SourcePath: &sourcePath,
		RootNodes: []*ast_domain.TemplateNode{
			{
				NodeType:    ast_domain.NodeText,
				TextContent: "Hello World",
				Location:    ast_domain.Location{Line: 1, Column: 1},
			},
		},
	}

	return result
}

func TestNextStaticAttrVarName(t *testing.T) {
	t.Parallel()

	em := &emitter{ctx: NewEmitterContext()}

	name1 := em.nextStaticAttrVarName()
	name2 := em.nextStaticAttrVarName()

	assert.Equal(t, "staticAttrs_1", name1)
	assert.Equal(t, "staticAttrs_2", name2)
}

func TestNextLoopIterName(t *testing.T) {
	t.Parallel()

	em := &emitter{ctx: NewEmitterContext()}

	name1 := em.nextLoopIterName()
	name2 := em.nextLoopIterName()

	assert.Equal(t, "loopIter_1", name1)
	assert.Equal(t, "loopIter_2", name2)
}

func TestExtractImportsFromAST(t *testing.T) {
	t.Parallel()

	importSet := make(map[string]goast.Spec)

	file := &goast.File{
		Name: cachedIdent("testpkg"),
		Decls: []goast.Decl{
			&goast.GenDecl{
				Tok: token.IMPORT,
				Specs: []goast.Spec{
					&goast.ImportSpec{
						Path: &goast.BasicLit{Kind: token.STRING, Value: `"fmt"`},
					},
					&goast.ImportSpec{
						Path: &goast.BasicLit{Kind: token.STRING, Value: `"strings"`},
					},
				},
			},
		},
	}

	extractImportsFromAST(importSet, file)

	assert.Len(t, importSet, 2)
}

func TestAddImportSpecsToSet(t *testing.T) {
	t.Parallel()

	importSet := make(map[string]goast.Spec)
	fmtSpec := &goast.ImportSpec{
		Path: &goast.BasicLit{Kind: token.STRING, Value: `"fmt"`},
	}

	addImportSpecsToSet(importSet, []goast.Spec{fmtSpec})
	assert.Len(t, importSet, 1, "first add should insert one entry")

	addImportSpecsToSet(importSet, []goast.Spec{fmtSpec})
	assert.Len(t, importSet, 1, "duplicate add should not increase length")

	nonImportSpec := &goast.ValueSpec{
		Names: []*goast.Ident{cachedIdent("x")},
	}
	addImportSpecsToSet(importSet, []goast.Spec{nonImportSpec})
	assert.Len(t, importSet, 1, "non-ImportSpec should be ignored")
}

func TestComputeRelativePath(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		baseDir string
		absPath string
		want    string
	}{
		{
			name:    "path within base directory",
			baseDir: "/home/user/project",
			absPath: "/home/user/project/src/main.go",
			want:    "src/main.go",
		},
		{
			name:    "path outside base directory does not panic",
			baseDir: "/home/user/project",
			absPath: "/other/path/file.go",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			em := &emitter{config: EmitterConfig{BaseDir: tc.baseDir}}
			got := em.computeRelativePath(tc.absPath)

			if tc.want != "" {
				assert.Equal(t, tc.want, got)
			} else {
				assert.NotEmpty(t, got, "result should not be empty")
			}
		})
	}
}

func TestSourceMappingStmt(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		node       *ast_domain.TemplateNode
		wantType   string
		wantIsExpr bool
	}{
		{
			name:     "nil node returns EmptyStmt",
			node:     nil,
			wantType: "empty",
		},
		{
			name: "synthetic location returns EmptyStmt",
			node: &ast_domain.TemplateNode{
				Location:      ast_domain.Location{},
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{OriginalSourcePath: new("/base/test.pk")},
			},
			wantType: "empty",
		},
		{
			name: "nil GoAnnotations returns EmptyStmt",
			node: &ast_domain.TemplateNode{
				Location:      ast_domain.Location{Line: 10},
				GoAnnotations: nil,
			},
			wantType: "empty",
		},
		{
			name: "valid location and source path returns ExprStmt",
			node: &ast_domain.TemplateNode{
				Location: ast_domain.Location{Line: 10},
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalSourcePath: new("/base/test.pk"),
				},
			},
			wantType: "expr",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			em := &emitter{config: EmitterConfig{BaseDir: "/base"}}
			got := em.sourceMappingStmt(tc.node)
			require.NotNil(t, got)

			switch tc.wantType {
			case "empty":
				_, ok := got.(*goast.EmptyStmt)
				assert.True(t, ok, "expected *goast.EmptyStmt, got %T", got)
			case "expr":
				exprStmt, ok := got.(*goast.ExprStmt)
				assert.True(t, ok, "expected *goast.ExprStmt, got %T", got)
				if ok {
					lit, litOk := exprStmt.X.(*goast.BasicLit)
					assert.True(t, litOk, "expected *goast.BasicLit X")
					if litOk {
						assert.True(t, strings.HasPrefix(lit.Value, "// line "), "directive should start with // line (comment form), got %q", lit.Value)
						assert.Contains(t, lit.Value, "test.pk:", "directive should contain source file")
					}
				}
			}
		})
	}
}

func TestDirectiveMappingStmt(t *testing.T) {
	t.Parallel()

	srcPath := "/base/pages/main.pk"

	t.Run("nil node returns EmptyStmt", func(t *testing.T) {
		t.Parallel()
		em := &emitter{config: EmitterConfig{BaseDir: "/base"}}
		got := em.directiveMappingStmt(nil, &ast_domain.Directive{})
		_, ok := got.(*goast.EmptyStmt)
		assert.True(t, ok, "expected EmptyStmt, got %T", got)
	})

	t.Run("nil directive returns EmptyStmt", func(t *testing.T) {
		t.Parallel()
		em := &emitter{config: EmitterConfig{BaseDir: "/base"}}
		node := &ast_domain.TemplateNode{
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{OriginalSourcePath: &srcPath},
		}
		got := em.directiveMappingStmt(node, nil)
		_, ok := got.(*goast.EmptyStmt)
		assert.True(t, ok, "expected EmptyStmt, got %T", got)
	})

	t.Run("synthetic NameLocation returns EmptyStmt", func(t *testing.T) {
		t.Parallel()
		em := &emitter{config: EmitterConfig{BaseDir: "/base"}}
		node := &ast_domain.TemplateNode{
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{OriginalSourcePath: &srcPath},
		}
		dir := &ast_domain.Directive{NameLocation: ast_domain.Location{}}
		got := em.directiveMappingStmt(node, dir)
		_, ok := got.(*goast.EmptyStmt)
		assert.True(t, ok, "expected EmptyStmt, got %T", got)
	})

	t.Run("valid directive with column emits comment form by default", func(t *testing.T) {
		t.Parallel()
		em := &emitter{config: EmitterConfig{BaseDir: "/base"}}
		node := &ast_domain.TemplateNode{
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{OriginalSourcePath: &srcPath},
		}
		dir := &ast_domain.Directive{NameLocation: ast_domain.Location{Line: 25, Column: 9}}
		got := em.directiveMappingStmt(node, dir)
		exprStmt, ok := got.(*goast.ExprStmt)
		assert.True(t, ok, "expected ExprStmt, got %T", got)
		if ok {
			lit, litOK := exprStmt.X.(*goast.BasicLit)
			assert.True(t, litOK, "expected BasicLit, got %T", exprStmt.X)
			if litOK {
				assert.Equal(t, "// line pages/main.pk:25:9", lit.Value)
			}
		}
	})

	t.Run("valid directive without column emits comment form by default", func(t *testing.T) {
		t.Parallel()
		em := &emitter{config: EmitterConfig{BaseDir: "/base"}}
		node := &ast_domain.TemplateNode{
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{OriginalSourcePath: &srcPath},
		}
		dir := &ast_domain.Directive{NameLocation: ast_domain.Location{Line: 25, Column: 0}}
		got := em.directiveMappingStmt(node, dir)
		exprStmt, ok := got.(*goast.ExprStmt)
		assert.True(t, ok, "expected ExprStmt, got %T", got)
		if ok {
			lit, litOK := exprStmt.X.(*goast.BasicLit)
			assert.True(t, litOK, "expected BasicLit, got %T", exprStmt.X)
			if litOK {
				assert.Equal(t, "// line pages/main.pk:25", lit.Value)
			}
		}
	})

	t.Run("dwarf enabled emits valid //line directive with column", func(t *testing.T) {
		t.Parallel()
		em := &emitter{config: EmitterConfig{BaseDir: "/base", EnableDwarfLineDirectives: true}}
		node := &ast_domain.TemplateNode{
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{OriginalSourcePath: &srcPath},
		}
		dir := &ast_domain.Directive{NameLocation: ast_domain.Location{Line: 25, Column: 9}}
		got := em.directiveMappingStmt(node, dir)
		exprStmt, ok := got.(*goast.ExprStmt)
		assert.True(t, ok, "expected ExprStmt, got %T", got)
		if ok {
			lit, litOK := exprStmt.X.(*goast.BasicLit)
			assert.True(t, litOK, "expected BasicLit, got %T", exprStmt.X)
			if litOK {
				assert.Equal(t, "//line pages/main.pk:25:9", lit.Value)
			}
		}
	})

	t.Run("dwarf enabled emits valid //line directive without column", func(t *testing.T) {
		t.Parallel()
		em := &emitter{config: EmitterConfig{BaseDir: "/base", EnableDwarfLineDirectives: true}}
		node := &ast_domain.TemplateNode{
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{OriginalSourcePath: &srcPath},
		}
		dir := &ast_domain.Directive{NameLocation: ast_domain.Location{Line: 25, Column: 0}}
		got := em.directiveMappingStmt(node, dir)
		exprStmt, ok := got.(*goast.ExprStmt)
		assert.True(t, ok, "expected ExprStmt, got %T", got)
		if ok {
			lit, litOK := exprStmt.X.(*goast.BasicLit)
			assert.True(t, litOK, "expected BasicLit, got %T", exprStmt.X)
			if litOK {
				assert.Equal(t, "//line pages/main.pk:25", lit.Value)
			}
		}
	})
}

func TestDeclNameAndSignature(t *testing.T) {
	t.Parallel()

	t.Run("func declaration", func(t *testing.T) {
		t.Parallel()
		decl := &goast.FuncDecl{Name: goast.NewIdent("Render")}
		name, sig := declNameAndSignature(decl)
		assert.Equal(t, "Render", name)
		assert.Equal(t, "func Render(", sig)
	})

	t.Run("type declaration", func(t *testing.T) {
		t.Parallel()
		decl := &goast.GenDecl{
			Tok:   token.TYPE,
			Specs: []goast.Spec{&goast.TypeSpec{Name: goast.NewIdent("Response")}},
		}
		name, sig := declNameAndSignature(decl)
		assert.Equal(t, "Response", name)
		assert.Equal(t, "type Response ", sig)
	})

	t.Run("var declaration", func(t *testing.T) {
		t.Parallel()
		decl := &goast.GenDecl{
			Tok: token.VAR,
			Specs: []goast.Spec{&goast.ValueSpec{
				Names: []*goast.Ident{goast.NewIdent("myVar")},
			}},
		}
		name, sig := declNameAndSignature(decl)
		assert.Equal(t, "myVar", name)
		assert.Equal(t, "var myVar ", sig)
	})

	t.Run("import declaration returns empty", func(t *testing.T) {
		t.Parallel()
		decl := &goast.GenDecl{
			Tok:   token.IMPORT,
			Specs: []goast.Spec{&goast.ImportSpec{Path: &goast.BasicLit{Value: `"fmt"`}}},
		}
		name, sig := declNameAndSignature(decl)
		assert.Empty(t, name)
		assert.Empty(t, sig)
	})

	t.Run("empty GenDecl returns empty", func(t *testing.T) {
		t.Parallel()
		decl := &goast.GenDecl{Tok: token.TYPE}
		name, sig := declNameAndSignature(decl)
		assert.Empty(t, name)
		assert.Empty(t, sig)
	})
}

func TestInjectUserCodeLineDirectives(t *testing.T) {
	t.Parallel()

	t.Run("injects directive before matching line", func(t *testing.T) {
		t.Parallel()

		src := []byte("package main\n\nfunc Render(r int) int {\n\treturn r\n}\n")
		directives := []userCodeLineDirective{
			{declSignature: "func Render(", directive: "//line test.pk:10"},
		}

		result := injectUserCodeLineDirectives(src, directives)
		lines := strings.Split(string(result), "\n")

		found := false
		for i, line := range lines {
			if strings.TrimSpace(line) == "//line test.pk:10" {
				found = true
				assert.True(t, i+1 < len(lines), "directive should have a following line")
				assert.True(t, strings.HasPrefix(strings.TrimSpace(lines[i+1]), "func Render("), "next line should be the function")
				break
			}
		}
		assert.True(t, found, "//line directive should be present in output")
	})

	t.Run("empty directives returns unchanged", func(t *testing.T) {
		t.Parallel()

		src := []byte("package main\n")
		result := injectUserCodeLineDirectives(src, nil)
		assert.Equal(t, src, result)
	})

	t.Run("multiple directives", func(t *testing.T) {
		t.Parallel()

		src := []byte("type Response struct{}\n\nfunc Render() {}\n")
		directives := []userCodeLineDirective{
			{declSignature: "type Response ", directive: "//line test.pk:5"},
			{declSignature: "func Render(", directive: "//line test.pk:10"},
		}

		result := injectUserCodeLineDirectives(src, directives)
		resultStr := string(result)

		assert.Contains(t, resultStr, "//line test.pk:5\ntype Response")
		assert.Contains(t, resultStr, "//line test.pk:10\nfunc Render")
	})

	t.Run("unmatched directive is silently dropped", func(t *testing.T) {
		t.Parallel()

		src := []byte("func Render() {}\n")
		directives := []userCodeLineDirective{
			{declSignature: "func NonExistent(", directive: "//line test.pk:99"},
		}

		result := injectUserCodeLineDirectives(src, directives)
		assert.Equal(t, string(src), string(result))
	})
}

func TestAddImport_AliasConflict(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		setup           func(em *emitter)
		wantImports     map[string]string
		wantAliases     map[string]string
		name            string
		canonicalPath   string
		alias           string
		wantImportCount int
		wantAliasCount  int
	}{
		{
			name:            "empty canonical path adds no import",
			canonicalPath:   "",
			alias:           "whatever",
			wantImportCount: 0,
			wantAliasCount:  0,
		},
		{
			name:            "self-import is skipped",
			canonicalPath:   "my/package",
			alias:           "pkg",
			wantImportCount: 0,
			wantAliasCount:  0,
		},
		{
			name: "already imported path is not duplicated",
			setup: func(em *emitter) {
				em.ctx.requiredImports["other/pkg"] = "opkg"
				em.ctx.usedAliases["opkg"] = "other/pkg"
			},
			canonicalPath:   "other/pkg",
			alias:           "opkg",
			wantImportCount: 1,
			wantImports:     map[string]string{"other/pkg": "opkg"},
			wantAliasCount:  1,
		},
		{
			name:            "normal import is added with alias",
			canonicalPath:   "github.com/foo/bar",
			alias:           "bar",
			wantImportCount: 1,
			wantImports:     map[string]string{"github.com/foo/bar": "bar"},
			wantAliases:     map[string]string{"bar": "github.com/foo/bar"},
			wantAliasCount:  1,
		},
		{
			name: "alias conflict produces renamed alias with _1 suffix",
			setup: func(em *emitter) {

				em.ctx.requiredImports["existing/dto"] = "dto"
				em.ctx.usedAliases["dto"] = "existing/dto"
			},
			canonicalPath:   "other/dto",
			alias:           "dto",
			wantImportCount: 2,
			wantImports:     map[string]string{"existing/dto": "dto", "other/dto": "dto_1"},
			wantAliases:     map[string]string{"dto": "existing/dto", "dto_1": "other/dto"},
			wantAliasCount:  2,
		},
		{
			name: "multiple alias conflicts produce incrementing suffixes",
			setup: func(em *emitter) {

				em.ctx.requiredImports["first/dto"] = "dto"
				em.ctx.usedAliases["dto"] = "first/dto"

				em.ctx.requiredImports["second/dto"] = "dto_1"
				em.ctx.usedAliases["dto_1"] = "second/dto"
			},
			canonicalPath:   "third/dto",
			alias:           "dto",
			wantImportCount: 3,
			wantImports: map[string]string{
				"first/dto":  "dto",
				"second/dto": "dto_1",
				"third/dto":  "dto_2",
			},
			wantAliases: map[string]string{
				"dto":   "first/dto",
				"dto_1": "second/dto",
				"dto_2": "third/dto",
			},
			wantAliasCount: 3,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			em := requireEmitter(t)
			em.config.CanonicalGoPackagePath = "my/package"
			em.ctx = NewEmitterContext()

			if tc.setup != nil {
				tc.setup(em)
			}

			em.addImport(tc.canonicalPath, tc.alias)

			assert.Len(t, em.ctx.requiredImports, tc.wantImportCount,
				"requiredImports should have %d entries", tc.wantImportCount)

			if tc.wantImports != nil {
				for path, expectedAlias := range tc.wantImports {
					actualAlias, exists := em.ctx.requiredImports[path]
					require.True(t, exists, "requiredImports should contain %q", path)
					assert.Equal(t, expectedAlias, actualAlias,
						"alias for %q should be %q", path, expectedAlias)
				}
			}

			if tc.wantAliases != nil {
				for alias, expectedPath := range tc.wantAliases {
					actualPath, exists := em.ctx.usedAliases[alias]
					require.True(t, exists, "usedAliases should contain %q", alias)
					assert.Equal(t, expectedPath, actualPath,
						"usedAliases[%q] should be %q", alias, expectedPath)
				}
			}

			if tc.wantAliasCount > 0 {
				assert.Len(t, em.ctx.usedAliases, tc.wantAliasCount,
					"usedAliases should have %d entries", tc.wantAliasCount)
			}
		})
	}
}

func TestAddPartialScriptImports(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		componentsByHash     map[string]*annotator_dto.VirtualComponent
		name                 string
		currentComponentHash string
		invocations          []*annotator_dto.PartialInvocation
		wantImportPaths      []string
		wantImportCount      int
	}{
		{
			name:                 "empty invocations add no imports",
			currentComponentHash: "current_hash",
			invocations:          []*annotator_dto.PartialInvocation{},
			componentsByHash:     map[string]*annotator_dto.VirtualComponent{},
			wantImportCount:      0,
		},
		{
			name:                 "self-import is skipped",
			currentComponentHash: "current_hash",
			invocations: []*annotator_dto.PartialInvocation{
				{PartialHashedName: "current_hash"},
			},
			componentsByHash: map[string]*annotator_dto.VirtualComponent{
				"current_hash": {HashedName: "current_hash"},
			},
			wantImportCount: 0,
		},
		{
			name:                 "component without RewrittenScriptAST is skipped",
			currentComponentHash: "current_hash",
			invocations: []*annotator_dto.PartialInvocation{
				{PartialHashedName: "other_hash"},
			},
			componentsByHash: map[string]*annotator_dto.VirtualComponent{
				"other_hash": {
					HashedName:         "other_hash",
					RewrittenScriptAST: nil,
				},
			},
			wantImportCount: 0,
		},
		{
			name:                 "nil component entry is skipped",
			currentComponentHash: "current_hash",
			invocations: []*annotator_dto.PartialInvocation{
				{PartialHashedName: "missing_hash"},
			},
			componentsByHash: map[string]*annotator_dto.VirtualComponent{},
			wantImportCount:  0,
		},
		{
			name:                 "component with RewrittenScriptAST containing imports extracts them",
			currentComponentHash: "current_hash",
			invocations: []*annotator_dto.PartialInvocation{
				{PartialHashedName: "partial_hash"},
			},
			componentsByHash: map[string]*annotator_dto.VirtualComponent{
				"partial_hash": {
					HashedName: "partial_hash",
					RewrittenScriptAST: &goast.File{
						Name: cachedIdent("partialpkg"),
						Decls: []goast.Decl{
							&goast.GenDecl{
								Tok: token.IMPORT,
								Specs: []goast.Spec{
									&goast.ImportSpec{
										Path: &goast.BasicLit{Kind: token.STRING, Value: `"fmt"`},
									},
									&goast.ImportSpec{
										Path: &goast.BasicLit{Kind: token.STRING, Value: `"strings"`},
									},
								},
							},
						},
					},
				},
			},
			wantImportCount: 2,
			wantImportPaths: []string{"fmt", "strings"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			importSet := make(map[string]goast.Spec)
			result := &annotator_dto.AnnotationResult{
				UniqueInvocations: tc.invocations,
				VirtualModule: &annotator_dto.VirtualModule{
					ComponentsByHash: tc.componentsByHash,
				},
			}

			addPartialScriptImports(importSet, result, tc.currentComponentHash)

			assert.Len(t, importSet, tc.wantImportCount,
				"importSet should contain %d entries", tc.wantImportCount)

			for _, path := range tc.wantImportPaths {
				_, exists := importSet[path]
				assert.True(t, exists, "importSet should contain path %q", path)
			}
		})
	}
}

func TestSourceMappingCommentGroup(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		node    *ast_domain.TemplateNode
		name    string
		wantNil bool
	}{
		{
			name:    "nil node returns nil",
			node:    nil,
			wantNil: true,
		},
		{
			name: "synthetic location returns nil",
			node: &ast_domain.TemplateNode{
				Location:      ast_domain.Location{},
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{OriginalSourcePath: new("/base/test.pk")},
			},
			wantNil: true,
		},
		{
			name: "valid node returns non-nil CommentGroup",
			node: &ast_domain.TemplateNode{
				Location: ast_domain.Location{Line: 10},
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalSourcePath: new("/base/test.pk"),
				},
			},
			wantNil: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			em := &emitter{config: EmitterConfig{BaseDir: "/base"}}
			got := em.sourceMappingCommentGroup(tc.node)

			if tc.wantNil {
				assert.Nil(t, got)
			} else {
				require.NotNil(t, got)
				assert.NotEmpty(t, got.List, "CommentGroup should contain at least one comment")
			}
		})
	}
}
