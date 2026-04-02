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
	"testing"

	goast "go/ast"
	"go/parser"
	"go/token"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/annotator/annotator_dto"
)

func noPropsTypeExpr() goast.Expr {
	return &goast.SelectorExpr{X: cachedIdent(facadePackageName), Sel: cachedIdent(NoPropsTypeName)}
}

func typedPropsTypeExpr() goast.Expr {
	return cachedIdent("Props")
}

func TestBuildCollectionPropsFallbacks_ReturnsNilForEmptyCollectionName(t *testing.T) {
	t.Parallel()

	component := &annotator_dto.VirtualComponent{
		RewrittenScriptAST: mustParseGoSource(t, `
			package main
			type Props struct { Title string `+"`prop:\"Title\"`"+` }
			func Render(r *RequestData, props Props) {}
		`),
	}
	result := buildCollectionPropsFallbacks(component, "", typedPropsTypeExpr())
	assert.Nil(t, result)
}

func TestBuildCollectionPropsFallbacks_ReturnsNilForNilComponent(t *testing.T) {
	t.Parallel()

	result := buildCollectionPropsFallbacks(nil, "blog", typedPropsTypeExpr())
	assert.Nil(t, result)
}

func TestBuildCollectionPropsFallbacks_ReturnsNilForNoProps(t *testing.T) {
	t.Parallel()

	component := &annotator_dto.VirtualComponent{
		RewrittenScriptAST: mustParseGoSource(t, `
			package main
			import piko "piko.sh/piko"
			func Render(r *RequestData, props piko.NoProps) {}
		`),
	}
	result := buildCollectionPropsFallbacks(component, "blog", noPropsTypeExpr())
	assert.Nil(t, result)
}

func TestBuildCollectionPropsFallbacks_GeneratesGuardForTypedProps(t *testing.T) {
	t.Parallel()

	component := &annotator_dto.VirtualComponent{
		RewrittenScriptAST: mustParseGoSource(t, `
			package main
			type Props struct {
				Title string `+"`prop:\"Title\"`"+`
			}
			func Render(r *RequestData, props Props) {}
		`),
	}

	result := buildCollectionPropsFallbacks(component, "blog", typedPropsTypeExpr())
	require.Len(t, result, 1)

	ifStmt, ok := result[0].(*goast.IfStmt)
	require.True(t, ok, "expected if statement")

	binaryExpr, ok := ifStmt.Cond.(*goast.BinaryExpr)
	require.True(t, ok, "expected binary expression in condition")
	assert.Equal(t, token.NEQ, binaryExpr.Op)

	xIdent, ok := binaryExpr.X.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "__metadata", xIdent.Name)

	yIdent, ok := binaryExpr.Y.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "nil", yIdent.Name)

	require.NotNil(t, ifStmt.Body)
	assert.NotEmpty(t, ifStmt.Body.List)
}

func TestBuildCollectionPropsFallbacks_SkipsFieldsWithoutPropTag(t *testing.T) {
	t.Parallel()

	component := &annotator_dto.VirtualComponent{
		RewrittenScriptAST: mustParseGoSource(t, `
			package main
			type Props struct {
				Title string `+"`prop:\"Title\"`"+`
				Internal string
			}
			func Render(r *RequestData, props Props) {}
		`),
	}

	result := buildCollectionPropsFallbacks(component, "blog", typedPropsTypeExpr())
	require.Len(t, result, 1)

	ifStmt, ok := result[0].(*goast.IfStmt)
	if !ok {
		t.Fatal("expected *goast.IfStmt")
	}
	assert.Len(t, ifStmt.Body.List, 1)
}

func TestExtractCollectionPropsFromAST_ExtractsMultipleFields(t *testing.T) {
	t.Parallel()

	file := mustParseGoSource(t, `
		package main
		type Props struct {
			Title string `+"`prop:\"Title\"`"+`
			Slug  string `+"`prop:\"Slug\"`"+`
			Draft bool   `+"`prop:\"Draft\"`"+`
		}
	`)

	props := extractCollectionPropsFromAST(file)
	require.Len(t, props, 3)

	assert.Equal(t, "Title", props[0].GoFieldName)
	assert.Equal(t, "Title", props[0].PropTagName)

	assert.Equal(t, "Slug", props[1].GoFieldName)
	assert.Equal(t, "Slug", props[1].PropTagName)

	assert.Equal(t, "Draft", props[2].GoFieldName)
	assert.Equal(t, "Draft", props[2].PropTagName)
}

func TestExtractCollectionPropsFromAST_ReturnsNilWithoutPropsStruct(t *testing.T) {
	t.Parallel()

	file := mustParseGoSource(t, `
		package main
		type Response struct { Title string }
	`)

	props := extractCollectionPropsFromAST(file)
	assert.Nil(t, props)
}

func TestExtractCollectionPropsFromAST_HandlesNestedStruct(t *testing.T) {
	t.Parallel()

	file := mustParseGoSource(t, `
		package main
		type Author struct {
			Name  string `+"`prop:\"name\"`"+`
			Email string `+"`prop:\"email\"`"+`
		}
		type Props struct {
			Title  string `+"`prop:\"Title\"`"+`
			Author Author `+"`prop:\"author\"`"+`
		}
	`)

	props := extractCollectionPropsFromAST(file)
	require.Len(t, props, 2)

	assert.Equal(t, "Title", props[0].GoFieldName)
	assert.Nil(t, props[0].NestedProps)

	assert.Equal(t, "Author", props[1].GoFieldName)
	assert.Equal(t, "author", props[1].PropTagName)
	require.Len(t, props[1].NestedProps, 2)
	assert.Equal(t, "Name", props[1].NestedProps[0].GoFieldName)
	assert.Equal(t, "name", props[1].NestedProps[0].PropTagName)
	assert.Equal(t, "Email", props[1].NestedProps[1].GoFieldName)
	assert.Equal(t, "email", props[1].NestedProps[1].PropTagName)
}

func TestExtractCollectionPropsFromAST_HandlesPointerField(t *testing.T) {
	t.Parallel()

	file := mustParseGoSource(t, `
		package main
		type Props struct {
			Title *string `+"`prop:\"Title\"`"+`
		}
	`)

	props := extractCollectionPropsFromAST(file)
	require.Len(t, props, 1)
	assert.True(t, props[0].IsPointer)
	assert.Equal(t, "Title", props[0].PropTagName)
}

func TestExtractCollectionPropsFromAST_StopsOnCircularTypes(t *testing.T) {
	t.Parallel()

	file := mustParseGoSource(t, `
		package main
		type Node struct {
			Name string `+"`prop:\"name\"`"+`
			Next Node   `+"`prop:\"next\"`"+`
		}
		type Props struct {
			Root Node `+"`prop:\"root\"`"+`
		}
	`)

	props := extractCollectionPropsFromAST(file)
	require.Len(t, props, 1)
	assert.Equal(t, "Root", props[0].GoFieldName)
	require.Len(t, props[0].NestedProps, 2)

	nameField := props[0].NestedProps[0]
	assert.Equal(t, "Name", nameField.GoFieldName)

	nextField := props[0].NestedProps[1]
	assert.Equal(t, "Next", nextField.GoFieldName)

	assert.Empty(t, nextField.NestedProps)
}

func TestParseFieldForCollection_ReturnsNilForNoTag(t *testing.T) {
	t.Parallel()

	field := &goast.Field{
		Names: []*goast.Ident{cachedIdent("Title")},
		Type:  cachedIdent("string"),
	}
	visited := make(map[string]bool)
	result := parseFieldForCollection(nil, field, visited)
	assert.Nil(t, result)
}

func TestParseFieldForCollection_ReturnsNilForEmptyPropTag(t *testing.T) {
	t.Parallel()

	field := &goast.Field{
		Names: []*goast.Ident{cachedIdent("Title")},
		Type:  cachedIdent("string"),
		Tag:   &goast.BasicLit{Kind: token.STRING, Value: "`query:\"title\"`"},
	}
	visited := make(map[string]bool)
	result := parseFieldForCollection(nil, field, visited)
	assert.Nil(t, result)
}

func TestParseFieldForCollection_StripsPropTagOptions(t *testing.T) {
	t.Parallel()

	field := &goast.Field{
		Names: []*goast.Ident{cachedIdent("Title")},
		Type:  cachedIdent("string"),
		Tag:   &goast.BasicLit{Kind: token.STRING, Value: "`prop:\"Title,omitempty\"`"},
	}
	visited := make(map[string]bool)
	result := parseFieldForCollection(nil, field, visited)
	require.NotNil(t, result)
	assert.Equal(t, "Title", result.PropTagName)
}

func TestFindStructByName_FindsNamedStruct(t *testing.T) {
	t.Parallel()

	file := mustParseGoSource(t, `
		package main
		type Author struct {
			Name string
		}
	`)

	result := findStructByName(file, "Author")
	require.NotNil(t, result)
	assert.Len(t, result.Fields.List, 1)
}

func TestFindStructByName_ReturnsNilForMissing(t *testing.T) {
	t.Parallel()

	file := mustParseGoSource(t, `
		package main
		type Response struct { Title string }
	`)

	result := findStructByName(file, "NotHere")
	assert.Nil(t, result)
}

func TestIsTimeType(t *testing.T) {
	t.Parallel()

	assert.True(t, isTimeType(&goast.SelectorExpr{
		X:   cachedIdent("time"),
		Sel: cachedIdent("Time"),
	}))

	assert.True(t, isTimeType(&goast.StarExpr{
		X: &goast.SelectorExpr{
			X:   cachedIdent("time"),
			Sel: cachedIdent("Time"),
		},
	}))

	assert.False(t, isTimeType(cachedIdent("string")))
	assert.False(t, isTimeType(&goast.SelectorExpr{
		X:   cachedIdent("time"),
		Sel: cachedIdent("Duration"),
	}))
}

func TestIsStringSliceType(t *testing.T) {
	t.Parallel()

	assert.True(t, isStringSliceType(&goast.ArrayType{
		Elt: cachedIdent("string"),
	}))

	assert.False(t, isStringSliceType(&goast.ArrayType{
		Elt: cachedIdent("int"),
	}))

	assert.False(t, isStringSliceType(cachedIdent("string")))
}

func TestDepthVar(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "__v0", depthVar("__v", 0))
	assert.Equal(t, "__v1", depthVar("__v", 1))
	assert.Equal(t, "__m2", depthVar("__m", 2))
	assert.Equal(t, "__ok0", depthVar("__ok", 0))
}

func TestBuildScalarMetadataAssignment_GeneratesStringAssert(t *testing.T) {
	t.Parallel()

	prop := collectionPropInfo{
		TypeExpr:    cachedIdent("string"),
		GoFieldName: "Title",
		PropTagName: "Title",
	}
	fieldAccess := &goast.SelectorExpr{
		X:   cachedIdent("props"),
		Sel: cachedIdent("Title"),
	}

	result := buildScalarMetadataAssignment(prop, cachedIdent(metadataVarName), fieldAccess, 0)
	require.NotNil(t, result)

	outerIf, ok := result.(*goast.IfStmt)
	require.True(t, ok)
	require.NotNil(t, outerIf.Init)
	require.NotNil(t, outerIf.Body)

	condIdent, ok := outerIf.Cond.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "__ok0", condIdent.Name)

	innerIf, ok := outerIf.Body.List[0].(*goast.IfStmt)
	require.True(t, ok)

	innerCondIdent, ok := innerIf.Cond.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "__ok0", innerCondIdent.Name)
}

func TestBuildScalarMetadataAssignment_GeneratesIntCoercion(t *testing.T) {
	t.Parallel()

	prop := collectionPropInfo{
		TypeExpr:    cachedIdent("int"),
		GoFieldName: "ReadingTime",
		PropTagName: "ReadingTime",
	}
	fieldAccess := &goast.SelectorExpr{
		X:   cachedIdent("props"),
		Sel: cachedIdent("ReadingTime"),
	}

	result := buildScalarMetadataAssignment(prop, cachedIdent(metadataVarName), fieldAccess, 0)
	require.NotNil(t, result)

	outerIf, ok := result.(*goast.IfStmt)
	if !ok {
		t.Fatal("expected *goast.IfStmt")
	}
	innerIf, ok := outerIf.Body.List[0].(*goast.IfStmt)
	require.True(t, ok)

	initAssign, ok := innerIf.Init.(*goast.AssignStmt)
	if !ok {
		t.Fatal("expected *goast.AssignStmt")
	}
	callExpr, ok := initAssign.Rhs[0].(*goast.CallExpr)
	require.True(t, ok)

	selector, ok := callExpr.Fun.(*goast.SelectorExpr)
	require.True(t, ok)
	selectorXIdent, ok := selector.X.(*goast.Ident)
	if !ok {
		t.Fatal("expected *goast.Ident")
	}
	assert.Equal(t, runtimePackageName, selectorXIdent.Name)
	assert.Equal(t, "CoerceInt", selector.Sel.Name)
}

func TestBuildScalarMetadataAssignment_GeneratesGenericCoercionForInt32(t *testing.T) {
	t.Parallel()

	prop := collectionPropInfo{
		TypeExpr:    cachedIdent("int32"),
		GoFieldName: "Count",
		PropTagName: "Count",
	}
	fieldAccess := &goast.SelectorExpr{
		X:   cachedIdent("props"),
		Sel: cachedIdent("Count"),
	}

	result := buildScalarMetadataAssignment(prop, cachedIdent(metadataVarName), fieldAccess, 0)
	require.NotNil(t, result)

	outerIf, ok := result.(*goast.IfStmt)
	if !ok {
		t.Fatal("expected *goast.IfStmt")
	}
	innerIf, ok := outerIf.Body.List[0].(*goast.IfStmt)
	require.True(t, ok)

	initAssign, ok := innerIf.Init.(*goast.AssignStmt)
	if !ok {
		t.Fatal("expected *goast.AssignStmt")
	}
	callExpr, ok := initAssign.Rhs[0].(*goast.CallExpr)
	require.True(t, ok)

	indexExpr, ok := callExpr.Fun.(*goast.IndexExpr)
	require.True(t, ok)

	selector, ok := indexExpr.X.(*goast.SelectorExpr)
	require.True(t, ok)
	selectorXIdent, ok := selector.X.(*goast.Ident)
	if !ok {
		t.Fatal("expected *goast.Ident")
	}
	assert.Equal(t, runtimePackageName, selectorXIdent.Name)
	assert.Equal(t, "CoerceSignedInt", selector.Sel.Name)

	typeParam, ok := indexExpr.Index.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "int32", typeParam.Name)
}

func TestBuildScalarMetadataAssignment_ReturnsNilForUnsupportedType(t *testing.T) {
	t.Parallel()

	prop := collectionPropInfo{
		TypeExpr:    &goast.MapType{Key: cachedIdent("string"), Value: cachedIdent("any")},
		GoFieldName: "Data",
		PropTagName: "data",
	}
	fieldAccess := &goast.SelectorExpr{
		X:   cachedIdent("props"),
		Sel: cachedIdent("Data"),
	}

	result := buildScalarMetadataAssignment(prop, cachedIdent(metadataVarName), fieldAccess, 0)
	assert.Nil(t, result)
}

func TestBuildNestedStructAssignment_GeneratesNestedMapping(t *testing.T) {
	t.Parallel()

	prop := collectionPropInfo{
		TypeExpr:    cachedIdent("Author"),
		GoFieldName: "Author",
		PropTagName: "author",
		NestedProps: []collectionPropInfo{
			{
				TypeExpr:    cachedIdent("string"),
				GoFieldName: "Name",
				PropTagName: "name",
			},
		},
	}
	fieldAccess := &goast.SelectorExpr{
		X:   cachedIdent("props"),
		Sel: cachedIdent("Author"),
	}

	result := buildNestedStructAssignment(prop, cachedIdent(metadataVarName), fieldAccess, 0)
	require.NotNil(t, result)

	outerIf, ok := result.(*goast.IfStmt)
	require.True(t, ok)
	require.Len(t, outerIf.Body.List, 1)

	mapAssertIf, ok := outerIf.Body.List[0].(*goast.IfStmt)
	require.True(t, ok)

	initAssign, ok := mapAssertIf.Init.(*goast.AssignStmt)
	require.True(t, ok)
	lhsIdent, ok := initAssign.Lhs[0].(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "__m0", lhsIdent.Name)

	require.NotEmpty(t, mapAssertIf.Body.List)
}

func TestWrapInMetadataLookup_UsesMetadataGet(t *testing.T) {
	t.Parallel()

	inner := &goast.ExprStmt{X: cachedIdent("placeholder")}
	result := wrapInMetadataLookup(cachedIdent(metadataVarName), "Title", inner, 0)

	require.NotNil(t, result.Init)
	initAssign, ok := result.Init.(*goast.AssignStmt)
	require.True(t, ok)

	callExpr, ok := initAssign.Rhs[0].(*goast.CallExpr)
	require.True(t, ok)

	selector, ok := callExpr.Fun.(*goast.SelectorExpr)
	require.True(t, ok)
	selectorXIdent, ok := selector.X.(*goast.Ident)
	if !ok {
		t.Fatal("expected *goast.Ident")
	}
	assert.Equal(t, runtimePackageName, selectorXIdent.Name)
	assert.Equal(t, "MetadataGet", selector.Sel.Name)

	vIdent, ok := initAssign.Lhs[0].(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "__v0", vIdent.Name)

	condIdent, ok := result.Cond.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "__ok0", condIdent.Name)
}

func mustParseGoSource(t *testing.T, source string) *goast.File {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", source, parser.ParseComments)
	require.NoError(t, err, "failed to parse Go source")
	return file
}
