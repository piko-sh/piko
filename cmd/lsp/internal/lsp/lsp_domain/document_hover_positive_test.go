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

package lsp_domain

import (
	"context"
	goast "go/ast"
	"strings"
	"testing"

	"go.lsp.dev/protocol"
	"piko.sh/piko/internal/annotator/annotator_domain"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
)

func TestGetHoverInfo_WithAnnotatedExpression(t *testing.T) {

	node := newTestNodeMultiLine("div", 1, 1, 5, 20)

	stateIdent := &ast_domain.Identifier{Name: "count"}
	stateIdent.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
		BaseCodeGenVarName: new("state"),
		Symbol:             &ast_domain.ResolvedSymbol{Name: "state"},
		ResolvedType: &ast_domain.ResolvedTypeInfo{
			TypeExpression:       goast.NewIdent("int"),
			CanonicalPackagePath: "",
		},
	}

	node.DirText = &ast_domain.Directive{
		RawExpression: "count",
		Expression:    stateIdent,
		Location:      ast_domain.Location{Line: 2, Column: 3},
	}

	tree := newTestAnnotatedAST(node)

	document := newTestDocumentBuilder().
		WithURI("file:///test/component.pk").
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: tree,
		}).
		Build()

	result, err := document.GetHoverInfo(context.Background(), protocol.Position{Line: 2, Character: 5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_ = result
}

func TestGetHoverInfo_NilAnnotationResult(t *testing.T) {
	document := newTestDocumentBuilder().
		WithURI("file:///test/component.pk").
		Build()

	result, err := document.GetHoverInfo(context.Background(), protocol.Position{Line: 0, Character: 0})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Error("expected nil hover for nil AnnotationResult")
	}
}

func TestFormatHoverContentsEnhanced_FieldSymbol(t *testing.T) {
	fieldType := &inspector_dto.Type{
		Name: "User",
		Fields: []*inspector_dto.Field{
			{Name: "ID", TypeString: "int64"},
			{Name: "Name", TypeString: "string"},
		},
	}

	customTI := &resolveNamedTypeMock{
		resolveExprResult: fieldType,
	}

	document := newTestDocumentBuilder().
		WithURI("file:///test/component.pk").
		WithTypeInspector(customTI).
		WithAnalysisMap(make(map[*ast_domain.TemplateNode]*annotator_domain.AnalysisContext)).
		Build()

	expression := &ast_domain.Identifier{Name: "Name"}
	expression.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
		BaseCodeGenVarName: new("state"),
		Symbol: &ast_domain.ResolvedSymbol{
			Name:              "Name",
			ReferenceLocation: ast_domain.Location{Line: 10, Column: 5},
		},
		OriginalSourcePath: new("/project/types.go"),
		ResolvedType: &ast_domain.ResolvedTypeInfo{
			TypeExpression:       goast.NewIdent("string"),
			CanonicalPackagePath: "",
		},
	}

	result := document.formatHoverContentsEnhanced(context.Background(), expression, protocol.Position{}, nil)
	if result == "" {
		t.Fatal("expected non-empty hover content for field symbol")
	}
	if !strings.Contains(result, "field") {
		t.Errorf("expected 'field' in hover, got:\n%s", result)
	}
	if !strings.Contains(result, "Name") {
		t.Errorf("expected 'Name' in hover, got:\n%s", result)
	}
}

func TestFormatHoverContentsEnhanced_StateIdentifier(t *testing.T) {
	document := newTestDocumentBuilder().
		WithURI("file:///test/component.pk").
		WithContent(`<template><div p-text="state.Name"></div></template>
<script type="application/go">
package main

type PageData struct {
	Name string
}

func Render() PageData {
	return PageData{Name: "test"}
}
</script>`).
		Build()

	expression := &ast_domain.Identifier{Name: "state"}
	expression.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
		BaseCodeGenVarName: new("state"),
		ResolvedType: &ast_domain.ResolvedTypeInfo{
			TypeExpression:       goast.NewIdent("Response"),
			CanonicalPackagePath: "",
		},
	}

	result := document.formatHoverContentsEnhanced(context.Background(), expression, protocol.Position{}, nil)
	if result == "" {
		t.Fatal("expected non-empty hover content for state identifier")
	}
	if !strings.Contains(result, "state") {
		t.Errorf("expected 'state' in hover, got:\n%s", result)
	}
}

func TestFormatNonFieldHover_FunctionSymbol(t *testing.T) {
	document := newTestDocumentBuilder().
		WithURI("file:///test/component.pk").
		Build()

	ann := &ast_domain.GoGeneratorAnnotation{
		ResolvedType: &ast_domain.ResolvedTypeInfo{
			TypeExpression:       goast.NewIdent("function"),
			CanonicalPackagePath: "",
		},
	}

	expression := &ast_domain.Identifier{Name: "handleClick"}

	result := document.formatNonFieldHover(context.Background(), expression, ann, "function", "handleClick", "func()", nil)
	if result == "" {
		t.Fatal("expected non-empty hover content")
	}
	if !strings.Contains(result, "handleClick") {
		t.Errorf("expected 'handleClick' in hover, got:\n%s", result)
	}
	if !strings.Contains(result, "```go") {
		t.Errorf("expected Go code block, got:\n%s", result)
	}
}

func TestFormatNonFieldHover_MethodSymbol(t *testing.T) {
	document := newTestDocumentBuilder().
		WithURI("file:///test/component.pk").
		Build()

	ann := &ast_domain.GoGeneratorAnnotation{
		ResolvedType: &ast_domain.ResolvedTypeInfo{
			TypeExpression:       &goast.FuncType{},
			CanonicalPackagePath: "",
		},
	}

	expression := &ast_domain.MemberExpression{
		Base:     &ast_domain.Identifier{Name: "obj"},
		Property: &ast_domain.Identifier{Name: "String"},
	}

	result := document.formatNonFieldHover(context.Background(), expression, ann, "method", "obj.String", "func() string", nil)
	if result == "" {
		t.Fatal("expected non-empty hover content")
	}
	if !strings.Contains(result, "method") {
		t.Errorf("expected 'method' in hover, got:\n%s", result)
	}
}

func TestFormatNonFieldHover_WithTypePreview(t *testing.T) {
	namedType := &inspector_dto.Type{
		Name: "Config",
		Fields: []*inspector_dto.Field{
			{Name: "Port", TypeString: "int"},
			{Name: "Host", TypeString: "string"},
		},
	}

	customTI := &resolveNamedTypeMock{
		resolveExprResult: namedType,
	}

	document := newTestDocumentBuilder().
		WithURI("file:///test/component.pk").
		WithTypeInspector(customTI).
		Build()

	ann := &ast_domain.GoGeneratorAnnotation{
		ResolvedType: &ast_domain.ResolvedTypeInfo{
			TypeExpression:       goast.NewIdent("Config"),
			CanonicalPackagePath: "example.com/config",
		},
	}

	expression := &ast_domain.Identifier{Name: "cfg"}

	result := document.formatNonFieldHover(context.Background(), expression, ann, "variable", "cfg", "Config", nil)
	if result == "" {
		t.Fatal("expected non-empty hover content")
	}
	if !strings.Contains(result, "type Config struct") {
		t.Errorf("expected type preview, got:\n%s", result)
	}
}

func TestFormatNonFieldHover_WithPackageLink(t *testing.T) {
	document := newTestDocumentBuilder().
		WithURI("file:///test/component.pk").
		Build()

	ann := &ast_domain.GoGeneratorAnnotation{
		ResolvedType: &ast_domain.ResolvedTypeInfo{
			TypeExpression:       goast.NewIdent("UUID"),
			PackageAlias:         "uuid",
			CanonicalPackagePath: "github.com/google/uuid",
		},
	}

	expression := &ast_domain.Identifier{Name: "id"}

	result := document.formatNonFieldHover(context.Background(), expression, ann, "variable", "id", "uuid.UUID", nil)
	if !strings.Contains(result, "pkg.go.dev") {
		t.Errorf("expected package link, got:\n%s", result)
	}
}

func TestFormatFieldHover_WithTypePreview(t *testing.T) {
	namedType := &inspector_dto.Type{
		Name: "Address",
		Fields: []*inspector_dto.Field{
			{Name: "Street", TypeString: "string"},
			{Name: "City", TypeString: "string"},
		},
	}

	customTI := &resolveNamedTypeMock{
		resolveExprResult: namedType,
	}

	document := newTestDocumentBuilder().
		WithURI("file:///test/component.pk").
		WithTypeInspector(customTI).
		WithAnalysisMap(make(map[*ast_domain.TemplateNode]*annotator_domain.AnalysisContext)).
		Build()

	ann := &ast_domain.GoGeneratorAnnotation{
		OriginalSourcePath: new("/project/types.go"),
		ResolvedType: &ast_domain.ResolvedTypeInfo{
			TypeExpression:       goast.NewIdent("Address"),
			CanonicalPackagePath: "example.com/models",
		},
	}

	result := document.formatFieldHover(context.Background(), ann, "HomeAddress", "Address")
	if result == "" {
		t.Fatal("expected non-empty result")
	}
	if !strings.Contains(result, "field HomeAddress Address") {
		t.Errorf("expected 'field HomeAddress Address', got:\n%s", result)
	}
	if !strings.Contains(result, "type Address struct") {
		t.Errorf("expected type preview, got:\n%s", result)
	}
}

func TestFormatFieldHover_WithPackageLinkAndTag(t *testing.T) {
	document := newTestDocumentBuilder().
		WithURI("file:///test/component.pk").
		Build()

	ann := &ast_domain.GoGeneratorAnnotation{
		FieldTag: new(`json:"created_at"`),
		ResolvedType: &ast_domain.ResolvedTypeInfo{
			TypeExpression:       goast.NewIdent("Time"),
			CanonicalPackagePath: "github.com/some/timepkg",
		},
	}

	result := document.formatFieldHover(context.Background(), ann, "CreatedAt", "time.Time")
	if !strings.Contains(result, `json:"created_at"`) {
		t.Errorf("expected tag in hover, got:\n%s", result)
	}
	if !strings.Contains(result, "pkg.go.dev") {
		t.Errorf("expected package link, got:\n%s", result)
	}
}

func TestGetTypePreview_WithTypeInspector(t *testing.T) {
	namedType := &inspector_dto.Type{
		Name: "Config",
		Fields: []*inspector_dto.Field{
			{Name: "Port", TypeString: "int"},
			{Name: "Host", TypeString: "string"},
		},
	}

	customTI := &resolveNamedTypeMock{
		resolveExprResult: namedType,
	}

	document := newTestDocumentBuilder().
		WithURI("file:///test/component.pk").
		WithTypeInspector(customTI).
		WithAnalysisMap(make(map[*ast_domain.TemplateNode]*annotator_domain.AnalysisContext)).
		Build()

	ann := &ast_domain.GoGeneratorAnnotation{
		OriginalSourcePath: new("/project/types.go"),
		ResolvedType: &ast_domain.ResolvedTypeInfo{
			TypeExpression:       goast.NewIdent("Config"),
			CanonicalPackagePath: "example.com/config",
		},
	}

	result := document.getTypePreview(context.Background(), ann, 10)
	if result == "" {
		t.Fatal("expected non-empty type preview")
	}
	if !strings.Contains(result, "type Config struct") {
		t.Errorf("expected struct declaration, got:\n%s", result)
	}
	if !strings.Contains(result, "Port") {
		t.Errorf("expected 'Port' field, got:\n%s", result)
	}
}

func TestGetTypePreview_WithInitialContext(t *testing.T) {
	namedType := &inspector_dto.Type{
		Name: "Settings",
		Fields: []*inspector_dto.Field{
			{Name: "Debug", TypeString: "bool"},
		},
	}

	customTI := &resolveNamedTypeMock{
		resolveExprResult: namedType,
	}

	document := newTestDocumentBuilder().
		WithURI("file:///test/component.pk").
		WithTypeInspector(customTI).
		WithAnalysisMap(make(map[*ast_domain.TemplateNode]*annotator_domain.AnalysisContext)).
		Build()

	ann := &ast_domain.GoGeneratorAnnotation{
		OriginalSourcePath: new("/project/types.go"),
		ResolvedType: &ast_domain.ResolvedTypeInfo{
			TypeExpression:       goast.NewIdent("Settings"),
			CanonicalPackagePath: "example.com/config",
			InitialPackagePath:   "example.com/init",
			InitialFilePath:      "/project/init.go",
		},
	}

	result := document.getTypePreview(context.Background(), ann, 10)
	if result == "" {
		t.Fatal("expected non-empty type preview with initial context")
	}
}

func TestGetTypePreviewForAnySymbol_WithResolvedType(t *testing.T) {
	namedType := &inspector_dto.Type{
		Name: "User",
		Fields: []*inspector_dto.Field{
			{Name: "ID", TypeString: "int"},
			{Name: "Name", TypeString: "string"},
		},
	}

	customTI := &resolveNamedTypeMock{
		resolveExprResult: namedType,
	}

	document := newTestDocumentBuilder().
		WithURI("file:///test/component.pk").
		WithTypeInspector(customTI).
		Build()

	ann := &ast_domain.GoGeneratorAnnotation{
		ResolvedType: &ast_domain.ResolvedTypeInfo{
			TypeExpression:       goast.NewIdent("User"),
			CanonicalPackagePath: "example.com/models",
		},
	}

	result := document.getTypePreviewForAnySymbol(context.Background(), ann, 10)
	if result == "" {
		t.Fatal("expected non-empty type preview")
	}
	if !strings.Contains(result, "type User struct") {
		t.Errorf("expected struct preview, got:\n%s", result)
	}
}

func TestGetTypePreviewForAnySymbol_SliceType(t *testing.T) {
	namedType := &inspector_dto.Type{
		Name: "Item",
		Fields: []*inspector_dto.Field{
			{Name: "ID", TypeString: "int"},
		},
	}

	customTI := &resolveNamedTypeMock{
		resolveExprResult: namedType,
	}

	document := newTestDocumentBuilder().
		WithURI("file:///test/component.pk").
		WithTypeInspector(customTI).
		Build()

	ann := &ast_domain.GoGeneratorAnnotation{
		ResolvedType: &ast_domain.ResolvedTypeInfo{
			TypeExpression: &goast.ArrayType{
				Elt: goast.NewIdent("Item"),
			},
			CanonicalPackagePath: "example.com/models",
		},
	}

	result := document.getTypePreviewForAnySymbol(context.Background(), ann, 10)
	if result == "" {
		t.Fatal("expected non-empty type preview for slice")
	}
	if !strings.Contains(result, "element type of []Item") {
		t.Errorf("expected slice element comment, got:\n%s", result)
	}
}

func TestGetTypePreviewForAnySymbol_WithInitialPaths(t *testing.T) {
	namedType := &inspector_dto.Type{
		Name: "Config",
		Fields: []*inspector_dto.Field{
			{Name: "Port", TypeString: "int"},
		},
	}

	customTI := &resolveNamedTypeMock{
		resolveExprResult: namedType,
	}

	document := newTestDocumentBuilder().
		WithURI("file:///test/component.pk").
		WithTypeInspector(customTI).
		Build()

	ann := &ast_domain.GoGeneratorAnnotation{
		OriginalSourcePath: new("/project/types.go"),
		ResolvedType: &ast_domain.ResolvedTypeInfo{
			TypeExpression:       goast.NewIdent("Config"),
			CanonicalPackagePath: "example.com/config",
			InitialPackagePath:   "example.com/init",
			InitialFilePath:      "/project/init.go",
		},
	}

	result := document.getTypePreviewForAnySymbol(context.Background(), ann, 10)
	if result == "" {
		t.Fatal("expected non-empty type preview with initial paths")
	}
}

func TestGetTypePreviewForAnySymbol_ResolverReturnsNil(t *testing.T) {
	customTI := &resolveNamedTypeMock{
		resolveExprResult: nil,
	}

	document := newTestDocumentBuilder().
		WithURI("file:///test/component.pk").
		WithTypeInspector(customTI).
		Build()

	ann := &ast_domain.GoGeneratorAnnotation{
		ResolvedType: &ast_domain.ResolvedTypeInfo{
			TypeExpression:       goast.NewIdent("Unknown"),
			CanonicalPackagePath: "example.com/models",
		},
	}

	result := document.getTypePreviewForAnySymbol(context.Background(), ann, 10)
	if result != "" {
		t.Errorf("expected empty for unresolvable type, got %q", result)
	}
}

func TestGetFunctionSignatureForHover_MemberExprBranch(t *testing.T) {
	methodSig := &inspector_dto.FunctionSignature{
		Params:  []string{"string"},
		Results: []string{"error"},
	}

	customTI := &methodSignatureMock{
		methodInfo: &inspector_dto.Method{
			Name:      "Save",
			Signature: *methodSig,
		},
	}

	document := newTestDocumentBuilder().
		WithURI("file:///test/component.pk").
		WithTypeInspector(customTI).
		Build()

	baseIdent := &ast_domain.Identifier{Name: "user"}
	baseIdent.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
		ResolvedType: &ast_domain.ResolvedTypeInfo{
			TypeExpression:       goast.NewIdent("User"),
			CanonicalPackagePath: "example.com/models",
		},
	}

	memberExpr := &ast_domain.MemberExpression{
		Base:     baseIdent,
		Property: &ast_domain.Identifier{Name: "Save"},
	}

	ann := &ast_domain.GoGeneratorAnnotation{
		ResolvedType: &ast_domain.ResolvedTypeInfo{
			TypeExpression:       &goast.FuncType{},
			CanonicalPackagePath: "example.com/models",
		},
	}

	result := document.getFunctionSignatureForHover(memberExpr, "Save", ann, nil)
	if result == "" {
		t.Fatal("expected non-empty signature for method")
	}
	if !strings.Contains(result, "func(") {
		t.Errorf("expected func signature, got %q", result)
	}
}

func TestGetFunctionSignatureForHover_MemberContextBranch(t *testing.T) {
	methodSig := &inspector_dto.FunctionSignature{
		Results: []string{"string"},
	}

	customTI := &methodSignatureMock{
		methodInfo: &inspector_dto.Method{
			Name:      "String",
			Signature: *methodSig,
		},
	}

	document := newTestDocumentBuilder().
		WithURI("file:///test/component.pk").
		WithTypeInspector(customTI).
		Build()

	baseIdent := &ast_domain.Identifier{Name: "obj"}
	baseIdent.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
		ResolvedType: &ast_domain.ResolvedTypeInfo{
			TypeExpression:       goast.NewIdent("MyType"),
			CanonicalPackagePath: "example.com/types",
		},
	}

	memberContext := &ast_domain.MemberExpression{
		Base:     baseIdent,
		Property: &ast_domain.Identifier{Name: "String"},
	}

	ann := &ast_domain.GoGeneratorAnnotation{
		ResolvedType: &ast_domain.ResolvedTypeInfo{
			TypeExpression:       &goast.FuncType{},
			CanonicalPackagePath: "example.com/types",
		},
	}

	expression := &ast_domain.Identifier{Name: "String"}

	result := document.getFunctionSignatureForHover(expression, "String", ann, memberContext)
	if result == "" {
		t.Fatal("expected non-empty signature from memberContext branch")
	}
}

func TestGetFunctionSignatureFromInspector_FoundSignature(t *testing.T) {
	customTI := &mockTypeInspector{
		FindFuncSignatureFunc: func(pkgAlias, functionName, importerPackagePath, importerFilePath string) *inspector_dto.FunctionSignature {
			if functionName == "HandleSubmit" {
				return &inspector_dto.FunctionSignature{
					Params:  []string{"*http.Request"},
					Results: []string{"error"},
				}
			}
			return nil
		},
	}

	document := newTestDocumentBuilder().
		WithURI("file:///test/component.pk").
		WithTypeInspector(customTI).
		Build()

	ann := &ast_domain.GoGeneratorAnnotation{
		ResolvedType: &ast_domain.ResolvedTypeInfo{
			PackageAlias:         "handlers",
			CanonicalPackagePath: "example.com/handlers",
		},
	}

	result := document.getFunctionSignatureFromInspector("HandleSubmit", ann)
	if result == "" {
		t.Fatal("expected non-empty function signature")
	}
	if !strings.Contains(result, "func(") {
		t.Errorf("expected func signature, got %q", result)
	}
}

func TestGetFunctionSignatureFromInspector_NotFound(t *testing.T) {
	customTI := &mockTypeInspector{
		FindFuncSignatureFunc: func(_, _, _, _ string) *inspector_dto.FunctionSignature {
			return nil
		},
	}

	document := newTestDocumentBuilder().
		WithURI("file:///test/component.pk").
		WithTypeInspector(customTI).
		Build()

	ann := &ast_domain.GoGeneratorAnnotation{
		ResolvedType: &ast_domain.ResolvedTypeInfo{
			PackageAlias:         "handlers",
			CanonicalPackagePath: "example.com/handlers",
		},
	}

	result := document.getFunctionSignatureFromInspector("NonExistent", ann)
	if result != "" {
		t.Errorf("expected empty string for non-existent function, got %q", result)
	}
}

func TestGetMethodSignatureFromInspector_FoundMethod(t *testing.T) {
	customTI := &methodSignatureMock{
		methodInfo: &inspector_dto.Method{
			Name: "Validate",
			Signature: inspector_dto.FunctionSignature{
				Results: []string{"error"},
			},
		},
	}

	document := newTestDocumentBuilder().
		WithURI("file:///test/component.pk").
		WithTypeInspector(customTI).
		Build()

	baseIdent := &ast_domain.Identifier{Name: "form"}
	baseIdent.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
		ResolvedType: &ast_domain.ResolvedTypeInfo{
			TypeExpression:       goast.NewIdent("Form"),
			CanonicalPackagePath: "example.com/forms",
		},
	}

	memberExpr := &ast_domain.MemberExpression{
		Base:     baseIdent,
		Property: &ast_domain.Identifier{Name: "Validate"},
	}

	ann := &ast_domain.GoGeneratorAnnotation{
		ResolvedType: &ast_domain.ResolvedTypeInfo{
			CanonicalPackagePath: "example.com/forms",
		},
	}

	result := document.getMethodSignatureFromInspector(memberExpr, "Validate", ann)
	if result == "" {
		t.Fatal("expected non-empty method signature")
	}
	if !strings.Contains(result, "func(") {
		t.Errorf("expected func signature, got %q", result)
	}
}

func TestGetMethodSignatureFromInspector_NilMethodInfo(t *testing.T) {
	customTI := &methodSignatureMock{
		methodInfo: nil,
	}

	document := newTestDocumentBuilder().
		WithURI("file:///test/component.pk").
		WithTypeInspector(customTI).
		Build()

	baseIdent := &ast_domain.Identifier{Name: "obj"}
	baseIdent.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
		ResolvedType: &ast_domain.ResolvedTypeInfo{
			TypeExpression:       goast.NewIdent("MyType"),
			CanonicalPackagePath: "example.com/types",
		},
	}

	memberExpr := &ast_domain.MemberExpression{
		Base:     baseIdent,
		Property: &ast_domain.Identifier{Name: "Unknown"},
	}

	ann := &ast_domain.GoGeneratorAnnotation{
		ResolvedType: &ast_domain.ResolvedTypeInfo{
			CanonicalPackagePath: "example.com/types",
		},
	}

	result := document.getMethodSignatureFromInspector(memberExpr, "Unknown", ann)
	if result != "" {
		t.Errorf("expected empty for nil method info, got %q", result)
	}
}

func TestGetLocalFunctionSignature_WithScriptBlock(t *testing.T) {
	content := `<template><div>Hello</div></template>
<script type="application/go">
package main

func HandleClick(name string) error {
	return nil
}
</script>`

	document := newTestDocumentBuilder().
		WithURI("file:///test/component.pk").
		WithContent(content).
		Build()

	result := document.getLocalFunctionSignature("HandleClick")
	if result == "" {
		t.Fatal("expected non-empty function signature")
	}
	if !strings.Contains(result, "func(") {
		t.Errorf("expected func signature, got %q", result)
	}
}

func TestGetLocalFunctionSignature_NoMatchingFunction(t *testing.T) {
	content := `<template><div>Hello</div></template>
<script type="application/go">
package main

func Render() PageData {
	return PageData{}
}
</script>`

	document := newTestDocumentBuilder().
		WithURI("file:///test/component.pk").
		WithContent(content).
		Build()

	result := document.getLocalFunctionSignature("NonExistentFunc")
	if result != "" {
		t.Errorf("expected empty string for non-existent function, got %q", result)
	}
}

func TestGetLocalFunctionSignature_NoScript(t *testing.T) {
	document := newTestDocumentBuilder().
		WithURI("file:///test/component.pk").
		WithContent(`<template><div>Hello</div></template>`).
		Build()

	result := document.getLocalFunctionSignature("AnyFunc")
	if result != "" {
		t.Errorf("expected empty for no script block, got %q", result)
	}
}

type methodSignatureMock struct {
	mockTypeInspector
	methodInfo *inspector_dto.Method
}

func (m *methodSignatureMock) FindMethodInfo(_ goast.Expr, _, _, _ string) *inspector_dto.Method {
	return m.methodInfo
}
