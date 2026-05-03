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
	goast "go/ast"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateCollectionDataPopulation(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		collectionName string
	}{
		{
			name:           "blog_collection",
			collectionName: "blog",
		},
		{
			name:           "products_collection",
			collectionName: "products",
		},
		{
			name:           "empty_collection_name",
			collectionName: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			statements := generateCollectionDataPopulation(tc.collectionName, "slug")

			require.Len(t, statements, 3, "expected 3 statements: assign, error check, data assignment")

			assignStmt, ok := statements[0].(*goast.AssignStmt)
			require.True(t, ok, "first statement should be *goast.AssignStmt, got %T", statements[0])
			assert.Len(t, assignStmt.Lhs, 4, "should have 4 leftHandSide variables")
			assert.Equal(t, token.DEFINE, assignStmt.Tok)

			errCheckStmt, ok := statements[1].(*goast.IfStmt)
			require.True(t, ok, "second statement should be *goast.IfStmt, got %T", statements[1])
			cond, ok := errCheckStmt.Cond.(*goast.BinaryExpr)
			require.True(t, ok, "Cond should be *goast.BinaryExpr")
			assert.Equal(t, token.NEQ, cond.Op, "should check __err != nil")
			require.NotNil(t, errCheckStmt.Body)
			require.Len(t, errCheckStmt.Body.List, 1)
			retStmt, ok := errCheckStmt.Body.List[0].(*goast.ReturnStmt)
			require.True(t, ok, "error body should contain a return statement")
			require.Len(t, retStmt.Results, 3, "return should have 3 values (nil, InternalMetadata{...}, nil)")

			metaLit, ok := retStmt.Results[1].(*goast.CompositeLit)
			require.True(t, ok, "second return value should be composite literal")
			require.Len(t, metaLit.Elts, 1, "should have 1 field: RenderError")

			dataAssign, ok := statements[2].(*goast.AssignStmt)
			require.True(t, ok, "third statement should be *goast.AssignStmt, got %T", statements[2])
			assert.Equal(t, token.ASSIGN, dataAssign.Tok)
		})
	}
}

func TestBuildCollectionItemFetchAssign(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		collectionName string
		wantLhsCount   int
	}{
		{
			name:           "standard_collection",
			collectionName: "articles",
			wantLhsCount:   4,
		},
		{
			name:           "empty_collection",
			collectionName: "",
			wantLhsCount:   4,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			statement := buildCollectionItemFetchAssign(tc.collectionName, "slug")

			require.NotNil(t, statement)
			assert.Len(t, statement.Lhs, tc.wantLhsCount)
			assert.Equal(t, token.DEFINE, statement.Tok)

			lhsNames := []string{"__metadata", "__contentAST", "__excerptAST", "__err"}
			for i, expected := range lhsNames {
				identifier, ok := statement.Lhs[i].(*goast.Ident)
				require.True(t, ok)
				assert.Equal(t, expected, identifier.Name)
			}

			require.Len(t, statement.Rhs, 1)
			callExpr, ok := statement.Rhs[0].(*goast.CallExpr)
			require.True(t, ok)

			selExpr, ok := callExpr.Fun.(*goast.SelectorExpr)
			require.True(t, ok)
			xIdent, ok := selExpr.X.(*goast.Ident)
			require.True(t, ok)
			assert.Equal(t, runtimePackageName, xIdent.Name)
			assert.Equal(t, "GetStaticCollectionItem", selExpr.Sel.Name)

			require.Len(t, callExpr.Args, 3)
		})
	}
}

func TestBuildCollectionDataAssignment(t *testing.T) {
	t.Parallel()

	statement := buildCollectionDataAssignment()

	require.NotNil(t, statement)
	assert.Equal(t, token.ASSIGN, statement.Tok)

	require.Len(t, statement.Lhs, 1)
	lhsIdent, ok := statement.Lhs[0].(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, RequestVarName, lhsIdent.Name)

	require.Len(t, statement.Rhs, 1)
	callExpr, ok := statement.Rhs[0].(*goast.CallExpr)
	require.True(t, ok)

	selExpr, ok := callExpr.Fun.(*goast.SelectorExpr)
	require.True(t, ok)
	assert.Equal(t, "WithCollectionData", selExpr.Sel.Name)

	require.Len(t, callExpr.Args, 1)
	compLit, ok := callExpr.Args[0].(*goast.CompositeLit)
	require.True(t, ok)

	mapType, ok := compLit.Type.(*goast.MapType)
	require.True(t, ok)
	keyIdent, ok := mapType.Key.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "string", keyIdent.Name)

	assert.Len(t, compLit.Elts, 3)
}

func TestGenerateDataVariableDeclaration(t *testing.T) {
	t.Parallel()

	statements := generateDataVariableDeclaration()

	require.Len(t, statements, 2)

	declStmt, ok := statements[0].(*goast.DeclStmt)
	require.True(t, ok, "first statement should be *goast.DeclStmt")

	ifStmt, ok := statements[1].(*goast.IfStmt)
	require.True(t, ok, "second statement should be *goast.IfStmt")
	require.NotNil(t, ifStmt.Init)
	require.NotNil(t, ifStmt.Cond)
	require.NotNil(t, ifStmt.Body)

	_ = declStmt
}

func TestBuildDataVarDecl(t *testing.T) {
	t.Parallel()

	statement := buildDataVarDecl()

	require.NotNil(t, statement)

	genDecl, ok := statement.Decl.(*goast.GenDecl)
	require.True(t, ok)
	assert.Equal(t, token.VAR, genDecl.Tok)

	require.Len(t, genDecl.Specs, 1)
	valueSpec, ok := genDecl.Specs[0].(*goast.ValueSpec)
	require.True(t, ok)

	require.Len(t, valueSpec.Names, 1)
	assert.Equal(t, "data", valueSpec.Names[0].Name)

	typeIdent, ok := valueSpec.Type.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, EmptyInterfaceTypeName, typeIdent.Name)
}

func TestBuildCollectionDataCallExpr(t *testing.T) {
	t.Parallel()

	callExpr := buildCollectionDataCallExpr()

	require.NotNil(t, callExpr)

	selExpr, ok := callExpr.Fun.(*goast.SelectorExpr)
	require.True(t, ok)

	xIdent, ok := selExpr.X.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, RequestVarName, xIdent.Name)
	assert.Equal(t, "CollectionData", selExpr.Sel.Name)

	assert.Empty(t, callExpr.Args)
}

func TestBuildDataExtractionIfStmt(t *testing.T) {
	t.Parallel()

	ifStmt := buildDataExtractionIfStmt()

	require.NotNil(t, ifStmt)

	require.NotNil(t, ifStmt.Init)
	initAssign, ok := ifStmt.Init.(*goast.AssignStmt)
	require.True(t, ok)

	require.Len(t, initAssign.Lhs, 2)
	rootMapIdent, ok := initAssign.Lhs[0].(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "rootMap", rootMapIdent.Name)

	okIdent, ok := initAssign.Lhs[1].(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, OkVarName, okIdent.Name)

	require.Len(t, initAssign.Rhs, 1)
	typeAssert, ok := initAssign.Rhs[0].(*goast.TypeAssertExpr)
	require.True(t, ok)

	mapType, ok := typeAssert.Type.(*goast.MapType)
	require.True(t, ok)
	_ = mapType

	condIdent, ok := ifStmt.Cond.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, OkVarName, condIdent.Name)

	require.NotNil(t, ifStmt.Body)
	require.Len(t, ifStmt.Body.List, 1)
	_, ok = ifStmt.Body.List[0].(*goast.IfStmt)
	require.True(t, ok, "body should contain nested if statement")
}

func TestBuildPageKeyExtractionIfStmt(t *testing.T) {
	t.Parallel()

	ifStmt := buildPageKeyExtractionIfStmt()

	require.NotNil(t, ifStmt)

	require.NotNil(t, ifStmt.Init)
	initAssign, ok := ifStmt.Init.(*goast.AssignStmt)
	require.True(t, ok)

	require.Len(t, initAssign.Lhs, 2)
	pageValIdent, ok := initAssign.Lhs[0].(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "pageVal", pageValIdent.Name)

	require.Len(t, initAssign.Rhs, 1)
	indexExpr, ok := initAssign.Rhs[0].(*goast.IndexExpr)
	require.True(t, ok)

	xIdent, ok := indexExpr.X.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "rootMap", xIdent.Name)

	indexLit, ok := indexExpr.Index.(*goast.BasicLit)
	require.True(t, ok)
	assert.Equal(t, `"page"`, indexLit.Value)

	require.NotNil(t, ifStmt.Body)
	require.Len(t, ifStmt.Body.List, 1)
	bodyAssign, ok := ifStmt.Body.List[0].(*goast.AssignStmt)
	require.True(t, ok)

	require.Len(t, bodyAssign.Lhs, 1)
	dataIdent, ok := bodyAssign.Lhs[0].(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "data", dataIdent.Name)

	require.NotNil(t, ifStmt.Else)
	elseBlock, ok := ifStmt.Else.(*goast.BlockStmt)
	require.True(t, ok)
	require.Len(t, elseBlock.List, 1)
	elseAssign, ok := elseBlock.List[0].(*goast.AssignStmt)
	require.True(t, ok)

	require.Len(t, elseAssign.Rhs, 1)
	callExpr, ok := elseAssign.Rhs[0].(*goast.CallExpr)
	require.True(t, ok)
	selExpr, ok := callExpr.Fun.(*goast.SelectorExpr)
	require.True(t, ok)
	assert.Equal(t, "CollectionData", selExpr.Sel.Name)
}

func TestPathParamExpr_NamedSlugWrapsWithCmpOr(t *testing.T) {
	t.Parallel()

	got := pathParamExpr("slug")
	call, ok := got.(*goast.CallExpr)
	require.True(t, ok, "expected CallExpr, got %T", got)
	selector, ok := call.Fun.(*goast.SelectorExpr)
	require.True(t, ok)
	pkg, ok := selector.X.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "cmp", pkg.Name)
	assert.Equal(t, "Or", selector.Sel.Name)
	require.Len(t, call.Args, 2, "cmp.Or should receive both named and wildcard lookups")
}

func TestPathParamExpr_StarParamSkipsCmpOr(t *testing.T) {
	t.Parallel()

	got := pathParamExpr("*")
	call, ok := got.(*goast.CallExpr)
	require.True(t, ok, "expected direct PathParam call, got %T", got)
	selector, ok := call.Fun.(*goast.SelectorExpr)
	require.True(t, ok)
	assert.Equal(t, "PathParam", selector.Sel.Name)
}
