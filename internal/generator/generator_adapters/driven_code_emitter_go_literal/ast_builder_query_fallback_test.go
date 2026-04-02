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
	"go/token"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetBaseTypeName(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		typeExpr goast.Expr
		want     string
	}{
		{
			name:     "Ident returns name",
			typeExpr: cachedIdent("string"),
			want:     "string",
		},
		{
			name:     "StarExpr wrapping Ident returns inner name",
			typeExpr: &goast.StarExpr{X: cachedIdent("int")},
			want:     "int",
		},
		{
			name:     "double pointer returns inner name",
			typeExpr: &goast.StarExpr{X: &goast.StarExpr{X: cachedIdent("bool")}},
			want:     "",
		},
		{
			name:     "non-Ident type returns empty",
			typeExpr: &goast.ArrayType{Elt: cachedIdent("int")},
			want:     "",
		},
		{
			name:     "SelectorExpr returns empty",
			typeExpr: &goast.SelectorExpr{X: cachedIdent("pkg"), Sel: cachedIdent("Type")},
			want:     "",
		},
		{
			name:     "MapType returns empty",
			typeExpr: &goast.MapType{Key: cachedIdent("string"), Value: cachedIdent("int")},
			want:     "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := getBaseTypeName(tc.typeExpr)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestParseFieldForQuery(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		field            *goast.Field
		name             string
		wantGoFieldName  string
		wantQueryParam   string
		wantNil          bool
		wantIsPointer    bool
		wantShouldCoerce bool
	}{
		{
			name: "field with query tag extracts name",
			field: &goast.Field{
				Names: []*goast.Ident{cachedIdent("UserID")},
				Type:  cachedIdent("string"),
				Tag:   &goast.BasicLit{Kind: token.STRING, Value: "`query:\"user_id\"`"},
			},
			wantGoFieldName:  "UserID",
			wantQueryParam:   "user_id",
			wantIsPointer:    false,
			wantShouldCoerce: false,
		},
		{
			name: "field without tag returns nil",
			field: &goast.Field{
				Names: []*goast.Ident{cachedIdent("Name")},
				Type:  cachedIdent("string"),
				Tag:   nil,
			},
			wantNil: true,
		},
		{
			name: "field with empty query value returns nil",
			field: &goast.Field{
				Names: []*goast.Ident{cachedIdent("Name")},
				Type:  cachedIdent("string"),
				Tag:   &goast.BasicLit{Kind: token.STRING, Value: "`query:\"\"`"},
			},
			wantNil: true,
		},
		{
			name: "field with coerce tag sets ShouldCoerce",
			field: &goast.Field{
				Names: []*goast.Ident{cachedIdent("Count")},
				Type:  cachedIdent("int"),
				Tag:   &goast.BasicLit{Kind: token.STRING, Value: "`query:\"count\" coerce:\"\"`"},
			},
			wantGoFieldName:  "Count",
			wantQueryParam:   "count",
			wantIsPointer:    false,
			wantShouldCoerce: true,
		},
		{
			name: "pointer field sets IsPointer",
			field: &goast.Field{
				Names: []*goast.Ident{cachedIdent("OptionalID")},
				Type:  &goast.StarExpr{X: cachedIdent("string")},
				Tag:   &goast.BasicLit{Kind: token.STRING, Value: "`query:\"optional_id\"`"},
			},
			wantGoFieldName:  "OptionalID",
			wantQueryParam:   "optional_id",
			wantIsPointer:    true,
			wantShouldCoerce: false,
		},
		{
			name: "field with non-query tag returns nil",
			field: &goast.Field{
				Names: []*goast.Ident{cachedIdent("Name")},
				Type:  cachedIdent("string"),
				Tag:   &goast.BasicLit{Kind: token.STRING, Value: "`json:\"name\"`"},
			},
			wantNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := parseFieldForQuery(tc.field)

			if tc.wantNil {
				assert.Nil(t, got)
				return
			}

			require.NotNil(t, got)
			assert.Equal(t, tc.wantGoFieldName, got.GoFieldName)
			assert.Equal(t, tc.wantQueryParam, got.QueryParamName)
			assert.Equal(t, tc.wantIsPointer, got.IsPointer)
			assert.Equal(t, tc.wantShouldCoerce, got.ShouldCoerce)
		})
	}
}

func TestFindPropsStruct(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		file    *goast.File
		name    string
		wantLen int
		wantNil bool
	}{
		{
			name: "file with Props struct returns struct",
			file: &goast.File{
				Name: cachedIdent("main"),
				Decls: []goast.Decl{
					&goast.GenDecl{
						Tok: token.TYPE,
						Specs: []goast.Spec{
							&goast.TypeSpec{
								Name: cachedIdent("Props"),
								Type: &goast.StructType{
									Fields: &goast.FieldList{
										List: []*goast.Field{
											{Names: []*goast.Ident{cachedIdent("Name")}, Type: cachedIdent("string")},
										},
									},
								},
							},
						},
					},
				},
			},
			wantLen: 1,
		},
		{
			name: "file without Props returns nil",
			file: &goast.File{
				Name: cachedIdent("main"),
				Decls: []goast.Decl{
					&goast.GenDecl{
						Tok: token.TYPE,
						Specs: []goast.Spec{
							&goast.TypeSpec{
								Name: cachedIdent("OtherType"),
								Type: &goast.StructType{
									Fields: &goast.FieldList{
										List: []*goast.Field{
											{Names: []*goast.Ident{cachedIdent("Field")}, Type: cachedIdent("int")},
										},
									},
								},
							},
						},
					},
				},
			},
			wantNil: true,
		},
		{
			name: "Props that is not a struct returns nil",
			file: &goast.File{
				Name: cachedIdent("main"),
				Decls: []goast.Decl{
					&goast.GenDecl{
						Tok: token.TYPE,
						Specs: []goast.Spec{
							&goast.TypeSpec{
								Name: cachedIdent("Props"),
								Type: cachedIdent("int"),
							},
						},
					},
				},
			},
			wantNil: true,
		},
		{
			name: "empty file returns nil",
			file: &goast.File{
				Name:  cachedIdent("main"),
				Decls: []goast.Decl{},
			},
			wantNil: true,
		},
		{
			name: "file with func decl only returns nil",
			file: &goast.File{
				Name: cachedIdent("main"),
				Decls: []goast.Decl{
					&goast.FuncDecl{
						Name: cachedIdent("main"),
						Type: &goast.FuncType{},
						Body: &goast.BlockStmt{},
					},
				},
			},
			wantNil: true,
		},
		{
			name: "Props with nil Fields returns nil",
			file: &goast.File{
				Name: cachedIdent("main"),
				Decls: []goast.Decl{
					&goast.GenDecl{
						Tok: token.TYPE,
						Specs: []goast.Spec{
							&goast.TypeSpec{
								Name: cachedIdent("Props"),
								Type: &goast.StructType{
									Fields: nil,
								},
							},
						},
					},
				},
			},
			wantNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := findPropsStruct(tc.file)

			if tc.wantNil {
				assert.Nil(t, got)
				return
			}

			require.NotNil(t, got)
			assert.Len(t, got.Fields.List, tc.wantLen)
		})
	}
}

func TestExtractQueryPropsFromStruct(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		structType *goast.StructType
		name       string
		wantFirst  string
		wantLen    int
	}{
		{
			name: "extracts multiple query props",
			structType: &goast.StructType{
				Fields: &goast.FieldList{
					List: []*goast.Field{
						{
							Names: []*goast.Ident{cachedIdent("ID")},
							Type:  cachedIdent("string"),
							Tag:   &goast.BasicLit{Kind: token.STRING, Value: "`query:\"id\"`"},
						},
						{
							Names: []*goast.Ident{cachedIdent("Name")},
							Type:  cachedIdent("string"),
							Tag:   &goast.BasicLit{Kind: token.STRING, Value: "`query:\"name\"`"},
						},
					},
				},
			},
			wantLen:   2,
			wantFirst: "ID",
		},
		{
			name: "skips fields without query tag",
			structType: &goast.StructType{
				Fields: &goast.FieldList{
					List: []*goast.Field{
						{
							Names: []*goast.Ident{cachedIdent("ID")},
							Type:  cachedIdent("string"),
							Tag:   &goast.BasicLit{Kind: token.STRING, Value: "`query:\"id\"`"},
						},
						{
							Names: []*goast.Ident{cachedIdent("Internal")},
							Type:  cachedIdent("string"),
							Tag:   nil,
						},
					},
				},
			},
			wantLen:   1,
			wantFirst: "ID",
		},
		{
			name: "empty struct returns empty slice",
			structType: &goast.StructType{
				Fields: &goast.FieldList{
					List: []*goast.Field{},
				},
			},
			wantLen: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := extractQueryPropsFromStruct(tc.structType)

			assert.Len(t, got, tc.wantLen)
			if tc.wantLen > 0 {
				assert.Equal(t, tc.wantFirst, got[0].GoFieldName)
			}
		})
	}
}

func TestBuildStringQueryFallback(t *testing.T) {
	t.Parallel()

	fieldAccess := &goast.SelectorExpr{
		X:   cachedIdent("props"),
		Sel: cachedIdent("Name"),
	}
	queryCall := &goast.CallExpr{
		Fun: &goast.SelectorExpr{
			X:   cachedIdent(RequestVarName),
			Sel: cachedIdent("QueryParam"),
		},
		Args: []goast.Expr{strLit("name")},
	}

	result := buildStringQueryFallback(fieldAccess, queryCall)

	ifStmt, ok := result.(*goast.IfStmt)
	require.True(t, ok, "Expected IfStmt")

	binaryExpr, ok := ifStmt.Cond.(*goast.BinaryExpr)
	require.True(t, ok)
	assert.Equal(t, token.EQL, binaryExpr.Op)

	require.Len(t, ifStmt.Body.List, 1)
	assignStmt, ok := ifStmt.Body.List[0].(*goast.AssignStmt)
	require.True(t, ok)
	assert.Equal(t, token.ASSIGN, assignStmt.Tok)
}

func TestBuildIntQueryFallback(t *testing.T) {
	t.Parallel()

	fieldAccess := &goast.SelectorExpr{
		X:   cachedIdent("props"),
		Sel: cachedIdent("Count"),
	}
	queryCall := &goast.CallExpr{
		Fun: &goast.SelectorExpr{
			X:   cachedIdent(RequestVarName),
			Sel: cachedIdent("QueryParam"),
		},
		Args: []goast.Expr{strLit("count")},
	}

	result := buildIntQueryFallback(fieldAccess, queryCall, "int")

	ifStmt, ok := result.(*goast.IfStmt)
	require.True(t, ok, "Expected outer IfStmt")

	binaryExpr, ok := ifStmt.Cond.(*goast.BinaryExpr)
	require.True(t, ok)
	assert.Equal(t, token.EQL, binaryExpr.Op)

	require.Len(t, ifStmt.Body.List, 1)
	innerIf, ok := ifStmt.Body.List[0].(*goast.IfStmt)
	require.True(t, ok, "Expected inner IfStmt for query param check")

	require.NotNil(t, innerIf.Init)
}

func TestBuildUintQueryFallback(t *testing.T) {
	t.Parallel()

	fieldAccess := &goast.SelectorExpr{
		X:   cachedIdent("props"),
		Sel: cachedIdent("ID"),
	}
	queryCall := &goast.CallExpr{
		Fun: &goast.SelectorExpr{
			X:   cachedIdent(RequestVarName),
			Sel: cachedIdent("QueryParam"),
		},
		Args: []goast.Expr{strLit("id")},
	}

	result := buildUintQueryFallback(fieldAccess, queryCall, "uint")

	ifStmt, ok := result.(*goast.IfStmt)
	require.True(t, ok, "Expected IfStmt")

	binaryExpr, ok := ifStmt.Cond.(*goast.BinaryExpr)
	require.True(t, ok)
	assert.Equal(t, token.EQL, binaryExpr.Op)
}

func TestBuildFloatQueryFallback(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		baseType string
	}{
		{name: "float64", baseType: "float64"},
		{name: "float32", baseType: "float32"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fieldAccess := &goast.SelectorExpr{
				X:   cachedIdent("props"),
				Sel: cachedIdent("Price"),
			}
			queryCall := &goast.CallExpr{
				Fun: &goast.SelectorExpr{
					X:   cachedIdent(RequestVarName),
					Sel: cachedIdent("QueryParam"),
				},
				Args: []goast.Expr{strLit("price")},
			}

			result := buildFloatQueryFallback(fieldAccess, queryCall, tc.baseType)

			ifStmt, ok := result.(*goast.IfStmt)
			require.True(t, ok, "Expected IfStmt")

			binaryExpr, ok := ifStmt.Cond.(*goast.BinaryExpr)
			require.True(t, ok)
			assert.Equal(t, token.EQL, binaryExpr.Op)

			zeroLit, ok := binaryExpr.Y.(*goast.BasicLit)
			require.True(t, ok)
			assert.Equal(t, token.FLOAT, zeroLit.Kind)
		})
	}
}

func TestBuildBoolQueryFallback(t *testing.T) {
	t.Parallel()

	fieldAccess := &goast.SelectorExpr{
		X:   cachedIdent("props"),
		Sel: cachedIdent("Active"),
	}
	queryCall := &goast.CallExpr{
		Fun: &goast.SelectorExpr{
			X:   cachedIdent(RequestVarName),
			Sel: cachedIdent("QueryParam"),
		},
		Args: []goast.Expr{strLit("active")},
	}

	result := buildBoolQueryFallback(fieldAccess, queryCall)

	ifStmt, ok := result.(*goast.IfStmt)
	require.True(t, ok, "Expected IfStmt")

	require.NotNil(t, ifStmt.Init)

	binaryExpr, ok := ifStmt.Cond.(*goast.BinaryExpr)
	require.True(t, ok)
	assert.Equal(t, token.NEQ, binaryExpr.Op)
}

func TestBuildPointerQueryFallback(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		prop    queryPropInfo
		wantNil bool
	}{
		{
			name: "pointer to string generates fallback",
			prop: queryPropInfo{
				GoFieldName:    "Name",
				QueryParamName: "name",
				TypeExpr:       &goast.StarExpr{X: cachedIdent("string")},
				IsPointer:      true,
				ShouldCoerce:   false,
			},
			wantNil: false,
		},
		{
			name: "pointer to int with coerce generates fallback",
			prop: queryPropInfo{
				GoFieldName:    "Count",
				QueryParamName: "count",
				TypeExpr:       &goast.StarExpr{X: cachedIdent("int")},
				IsPointer:      true,
				ShouldCoerce:   true,
			},
			wantNil: false,
		},
		{
			name: "pointer to int without coerce returns nil",
			prop: queryPropInfo{
				GoFieldName:    "Count",
				QueryParamName: "count",
				TypeExpr:       &goast.StarExpr{X: cachedIdent("int")},
				IsPointer:      true,
				ShouldCoerce:   false,
			},
			wantNil: true,
		},
		{
			name: "pointer to bool with coerce generates fallback",
			prop: queryPropInfo{
				GoFieldName:    "Active",
				QueryParamName: "active",
				TypeExpr:       &goast.StarExpr{X: cachedIdent("bool")},
				IsPointer:      true,
				ShouldCoerce:   true,
			},
			wantNil: false,
		},
		{
			name: "pointer to unsupported type returns nil",
			prop: queryPropInfo{
				GoFieldName:    "Data",
				QueryParamName: "data",
				TypeExpr:       &goast.StarExpr{X: cachedIdent("MyStruct")},
				IsPointer:      true,
				ShouldCoerce:   false,
			},
			wantNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fieldAccess := &goast.SelectorExpr{
				X:   cachedIdent("props"),
				Sel: cachedIdent(tc.prop.GoFieldName),
			}
			queryCall := &goast.CallExpr{
				Fun: &goast.SelectorExpr{
					X:   cachedIdent(RequestVarName),
					Sel: cachedIdent("QueryParam"),
				},
				Args: []goast.Expr{strLit(tc.prop.QueryParamName)},
			}

			result := buildPointerQueryFallback(tc.prop, fieldAccess, queryCall)

			if tc.wantNil {
				assert.Nil(t, result)
				return
			}

			require.NotNil(t, result)
			ifStmt, ok := result.(*goast.IfStmt)
			require.True(t, ok, "Expected IfStmt")

			binaryExpr, ok := ifStmt.Cond.(*goast.BinaryExpr)
			require.True(t, ok)
			assert.Equal(t, token.EQL, binaryExpr.Op)
			assert.Equal(t, GoKeywordNil, binaryExpr.Y.(*goast.Ident).Name)
		})
	}
}

func TestBuildQueryFallbackStatement_Dispatch(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		prop    queryPropInfo
		wantNil bool
	}{
		{
			name: "string type generates fallback",
			prop: queryPropInfo{
				GoFieldName:    "Name",
				QueryParamName: "name",
				TypeExpr:       cachedIdent("string"),
				IsPointer:      false,
				ShouldCoerce:   false,
			},
			wantNil: false,
		},
		{
			name: "int type with coerce generates fallback",
			prop: queryPropInfo{
				GoFieldName:    "Count",
				QueryParamName: "count",
				TypeExpr:       cachedIdent("int"),
				IsPointer:      false,
				ShouldCoerce:   true,
			},
			wantNil: false,
		},
		{
			name: "int type without coerce returns nil",
			prop: queryPropInfo{
				GoFieldName:    "Count",
				QueryParamName: "count",
				TypeExpr:       cachedIdent("int"),
				IsPointer:      false,
				ShouldCoerce:   false,
			},
			wantNil: true,
		},
		{
			name: "uint type with coerce generates fallback",
			prop: queryPropInfo{
				GoFieldName:    "ID",
				QueryParamName: "id",
				TypeExpr:       cachedIdent("uint"),
				IsPointer:      false,
				ShouldCoerce:   true,
			},
			wantNil: false,
		},
		{
			name: "float64 type with coerce generates fallback",
			prop: queryPropInfo{
				GoFieldName:    "Price",
				QueryParamName: "price",
				TypeExpr:       cachedIdent("float64"),
				IsPointer:      false,
				ShouldCoerce:   true,
			},
			wantNil: false,
		},
		{
			name: "bool type with coerce generates fallback",
			prop: queryPropInfo{
				GoFieldName:    "Active",
				QueryParamName: "active",
				TypeExpr:       cachedIdent("bool"),
				IsPointer:      false,
				ShouldCoerce:   true,
			},
			wantNil: false,
		},
		{
			name: "unsupported type returns nil",
			prop: queryPropInfo{
				GoFieldName:    "Data",
				QueryParamName: "data",
				TypeExpr:       cachedIdent("MyStruct"),
				IsPointer:      false,
				ShouldCoerce:   false,
			},
			wantNil: true,
		},
		{
			name: "pointer type delegates to buildPointerQueryFallback",
			prop: queryPropInfo{
				GoFieldName:    "Name",
				QueryParamName: "name",
				TypeExpr:       &goast.StarExpr{X: cachedIdent("string")},
				IsPointer:      true,
				ShouldCoerce:   false,
			},
			wantNil: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := buildQueryFallbackStatement(tc.prop)

			if tc.wantNil {
				assert.Nil(t, result)
				return
			}

			require.NotNil(t, result)
			_, ok := result.(*goast.IfStmt)
			assert.True(t, ok, "Expected IfStmt")
		})
	}
}

func TestBuildPointerInnerBody(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		prop    queryPropInfo
		wantNil bool
		wantLen int
	}{
		{
			name: "string returns assignment",
			prop: queryPropInfo{
				GoFieldName:  "Name",
				TypeExpr:     &goast.StarExpr{X: cachedIdent("string")},
				IsPointer:    true,
				ShouldCoerce: false,
			},
			wantLen: 1,
		},
		{
			name: "int with coerce returns parse assignment",
			prop: queryPropInfo{
				GoFieldName:  "Count",
				TypeExpr:     &goast.StarExpr{X: cachedIdent("int")},
				IsPointer:    true,
				ShouldCoerce: true,
			},
			wantLen: 1,
		},
		{
			name: "int without coerce returns nil",
			prop: queryPropInfo{
				GoFieldName:  "Count",
				TypeExpr:     &goast.StarExpr{X: cachedIdent("int")},
				IsPointer:    true,
				ShouldCoerce: false,
			},
			wantNil: true,
		},
		{
			name: "bool with coerce returns parse assignment",
			prop: queryPropInfo{
				GoFieldName:  "Active",
				TypeExpr:     &goast.StarExpr{X: cachedIdent("bool")},
				IsPointer:    true,
				ShouldCoerce: true,
			},
			wantLen: 1,
		},
		{
			name: "unsupported type returns nil",
			prop: queryPropInfo{
				GoFieldName:  "Data",
				TypeExpr:     &goast.StarExpr{X: cachedIdent("float64")},
				IsPointer:    true,
				ShouldCoerce: false,
			},
			wantNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fieldAccess := &goast.SelectorExpr{
				X:   cachedIdent("props"),
				Sel: cachedIdent(tc.prop.GoFieldName),
			}

			result := buildPointerInnerBody(tc.prop, fieldAccess)

			if tc.wantNil {
				assert.Nil(t, result)
				return
			}

			require.NotNil(t, result)
			assert.Len(t, result, tc.wantLen)
		})
	}
}

func TestBuildPointerStringAssignment(t *testing.T) {
	t.Parallel()

	fieldAccess := &goast.SelectorExpr{
		X:   cachedIdent("props"),
		Sel: cachedIdent("Name"),
	}

	result := buildPointerStringAssignment(fieldAccess)

	require.Len(t, result, 1)

	assignStmt, ok := result[0].(*goast.AssignStmt)
	require.True(t, ok)

	unaryExpr, ok := assignStmt.Rhs[0].(*goast.UnaryExpr)
	require.True(t, ok)
	assert.Equal(t, token.AND, unaryExpr.Op)
	assert.Equal(t, varNameQP, unaryExpr.X.(*goast.Ident).Name)
}

func TestBuildPointerIntParseAssignment(t *testing.T) {
	t.Parallel()

	fieldAccess := &goast.SelectorExpr{
		X:   cachedIdent("props"),
		Sel: cachedIdent("Count"),
	}

	result := buildPointerIntParseAssignment(fieldAccess, "int")

	require.Len(t, result, 1)

	ifStmt, ok := result[0].(*goast.IfStmt)
	require.True(t, ok)

	require.NotNil(t, ifStmt.Init)
	assignInit, ok := ifStmt.Init.(*goast.AssignStmt)
	require.True(t, ok)

	parseCall, ok := assignInit.Rhs[0].(*goast.CallExpr)
	require.True(t, ok)

	selector, ok := parseCall.Fun.(*goast.SelectorExpr)
	require.True(t, ok)
	assert.Equal(t, pkgStrconv, selector.X.(*goast.Ident).Name)
	assert.Equal(t, "Atoi", selector.Sel.Name)
}

func TestBuildPointerBoolParseAssignment(t *testing.T) {
	t.Parallel()

	fieldAccess := &goast.SelectorExpr{
		X:   cachedIdent("props"),
		Sel: cachedIdent("Active"),
	}

	result := buildPointerBoolParseAssignment(fieldAccess)

	require.Len(t, result, 1)

	ifStmt, ok := result[0].(*goast.IfStmt)
	require.True(t, ok)

	require.NotNil(t, ifStmt.Init)
	assignInit, ok := ifStmt.Init.(*goast.AssignStmt)
	require.True(t, ok)

	parseCall, ok := assignInit.Rhs[0].(*goast.CallExpr)
	require.True(t, ok)

	selector, ok := parseCall.Fun.(*goast.SelectorExpr)
	require.True(t, ok)
	assert.Equal(t, pkgStrconv, selector.X.(*goast.Ident).Name)
	assert.Equal(t, "ParseBool", selector.Sel.Name)
}

func TestBuildParseAndAssignValue(t *testing.T) {
	t.Parallel()

	parseCall := &goast.CallExpr{
		Fun:  &goast.SelectorExpr{X: cachedIdent(pkgStrconv), Sel: cachedIdent("Atoi")},
		Args: []goast.Expr{cachedIdent(varNameQP)},
	}
	fieldAccess := &goast.SelectorExpr{
		X:   cachedIdent("props"),
		Sel: cachedIdent("Count"),
	}
	assignExpr := cachedIdent(varNameV)

	result := buildParseAndAssignValue(parseCall, fieldAccess, assignExpr)

	require.NotNil(t, result)

	assignInit, ok := result.Init.(*goast.AssignStmt)
	require.True(t, ok)
	assert.Len(t, assignInit.Lhs, 2)
	assert.Equal(t, varNameV, assignInit.Lhs[0].(*goast.Ident).Name)
	assert.Equal(t, varNameErr, assignInit.Lhs[1].(*goast.Ident).Name)

	binaryExpr, ok := result.Cond.(*goast.BinaryExpr)
	require.True(t, ok)
	assert.Equal(t, token.EQL, binaryExpr.Op)
	assert.Equal(t, varNameErr, binaryExpr.X.(*goast.Ident).Name)
	assert.Equal(t, GoKeywordNil, binaryExpr.Y.(*goast.Ident).Name)
}

func TestBuildQueryParamCheck(t *testing.T) {
	t.Parallel()

	queryCall := &goast.CallExpr{
		Fun: &goast.SelectorExpr{
			X:   cachedIdent(RequestVarName),
			Sel: cachedIdent("QueryParam"),
		},
		Args: []goast.Expr{strLit("name")},
	}
	innerBody := []goast.Stmt{
		&goast.ExprStmt{X: cachedIdent("doSomething")},
	}

	result := buildQueryParamCheck(queryCall, innerBody)

	require.NotNil(t, result)

	assignInit, ok := result.Init.(*goast.AssignStmt)
	require.True(t, ok)
	assert.Equal(t, varNameQP, assignInit.Lhs[0].(*goast.Ident).Name)

	binaryExpr, ok := result.Cond.(*goast.BinaryExpr)
	require.True(t, ok)
	assert.Equal(t, token.NEQ, binaryExpr.Op)

	assert.Len(t, result.Body.List, 1)
}

func TestWrapInNilCheckWithQueryInit(t *testing.T) {
	t.Parallel()

	fieldAccess := &goast.SelectorExpr{
		X:   cachedIdent("props"),
		Sel: cachedIdent("Name"),
	}
	queryCall := &goast.CallExpr{
		Fun: &goast.SelectorExpr{
			X:   cachedIdent(RequestVarName),
			Sel: cachedIdent("QueryParam"),
		},
		Args: []goast.Expr{strLit("name")},
	}
	innerBody := []goast.Stmt{
		&goast.ExprStmt{X: cachedIdent("assign")},
	}

	result := wrapInNilCheckWithQueryInit(fieldAccess, queryCall, innerBody)

	require.NotNil(t, result)

	binaryExpr, ok := result.Cond.(*goast.BinaryExpr)
	require.True(t, ok)
	assert.Equal(t, token.EQL, binaryExpr.Op)
	assert.Equal(t, GoKeywordNil, binaryExpr.Y.(*goast.Ident).Name)

	require.Len(t, result.Body.List, 1)
	innerIf, ok := result.Body.List[0].(*goast.IfStmt)
	require.True(t, ok)

	require.NotNil(t, innerIf.Init)
}

func TestCloneSelectorExpr(t *testing.T) {
	t.Parallel()

	original := &goast.SelectorExpr{
		X:   cachedIdent("props"),
		Sel: cachedIdent("Name"),
	}

	cloned := cloneSelectorExpr(original)

	require.NotNil(t, cloned)
	assert.NotSame(t, original, cloned)
	assert.Equal(t, original.X, cloned.X)
	assert.Equal(t, original.Sel, cloned.Sel)
}

func TestBuildZeroCheckWithParseFallback(t *testing.T) {
	t.Parallel()

	fieldAccess := &goast.SelectorExpr{
		X:   cachedIdent("props"),
		Sel: cachedIdent("Count"),
	}
	queryCall := &goast.CallExpr{
		Fun: &goast.SelectorExpr{
			X:   cachedIdent(RequestVarName),
			Sel: cachedIdent("QueryParam"),
		},
		Args: []goast.Expr{strLit("count")},
	}
	parseCall := &goast.CallExpr{
		Fun:  &goast.SelectorExpr{X: cachedIdent(pkgStrconv), Sel: cachedIdent("Atoi")},
		Args: []goast.Expr{cachedIdent(varNameQP)},
	}
	assignExpr := cachedIdent(varNameV)
	zeroLit := intLit(0)

	result := buildZeroCheckWithParseFallback(fieldAccess, queryCall, parseCall, assignExpr, zeroLit)

	require.NotNil(t, result)

	binaryExpr, ok := result.Cond.(*goast.BinaryExpr)
	require.True(t, ok)
	assert.Equal(t, token.EQL, binaryExpr.Op)

	require.Len(t, result.Body.List, 1)
	queryCheckIf, ok := result.Body.List[0].(*goast.IfStmt)
	require.True(t, ok)
	require.NotNil(t, queryCheckIf.Init)
}

func TestBuildParseAndAssignPointer(t *testing.T) {
	t.Parallel()

	parseCall := &goast.CallExpr{
		Fun:  &goast.SelectorExpr{X: cachedIdent(pkgStrconv), Sel: cachedIdent("Atoi")},
		Args: []goast.Expr{cachedIdent(varNameQP)},
	}
	fieldAccess := &goast.SelectorExpr{
		X:   cachedIdent("props"),
		Sel: cachedIdent("Count"),
	}

	result := buildParseAndAssignPointer(parseCall, fieldAccess)

	require.NotNil(t, result)

	assignInit, ok := result.Init.(*goast.AssignStmt)
	require.True(t, ok)
	assert.Len(t, assignInit.Lhs, 2)

	binaryExpr, ok := result.Cond.(*goast.BinaryExpr)
	require.True(t, ok)
	assert.Equal(t, token.EQL, binaryExpr.Op)
	assert.Equal(t, GoKeywordNil, binaryExpr.Y.(*goast.Ident).Name)

	require.Len(t, result.Body.List, 1)
	assignStmt, ok := result.Body.List[0].(*goast.AssignStmt)
	require.True(t, ok)

	unaryExpr, ok := assignStmt.Rhs[0].(*goast.UnaryExpr)
	require.True(t, ok)
	assert.Equal(t, token.AND, unaryExpr.Op)
}
