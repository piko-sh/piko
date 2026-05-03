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
	"fmt"
	"reflect"
	"strings"

	goast "go/ast"
	"go/token"

	"piko.sh/piko/internal/annotator/annotator_dto"
)

// collectionPropInfo holds details about a Props struct field that should be
// populated from collection frontmatter metadata.
type collectionPropInfo struct {
	// TypeExpr is the Go AST expression for the field type.
	TypeExpr goast.Expr

	// GoFieldName is the name of the struct field in Go code.
	GoFieldName string

	// PropTagName is the metadata key to look up in __metadata.
	PropTagName string

	// NestedProps holds sub-field mappings when the field is a struct type.
	// nil for scalar fields.
	NestedProps []collectionPropInfo

	// IsPointer indicates whether the field type is a pointer.
	IsPointer bool
}

// metadataVarName is the variable name for the collection metadata map in generated
// code. Declared by generateCollectionDataPopulation before buildInitialRenderCall
// runs.
const metadataVarName = "__metadata"

// okVarBase is the base prefix for the ok-flag variable used in type assertions
// and map lookups in generated code.
const okVarBase = "__ok"

// valueVarBase is the base prefix for the temporary value variable used in
// metadata lookups in generated code.
const valueVarBase = "__v"

// depthVar produces a unique variable name for a given nesting depth,
// avoiding shadowed declarations in the generated code. At depth 0 it
// returns "__v0", at depth 1 "__v1", and so on.
//
// Takes base (string) which is the variable name prefix.
// Takes depth (int) which is the nesting depth appended to the prefix.
//
// Returns string which is the depth-qualified variable name.
func depthVar(base string, depth int) string {
	return fmt.Sprintf("%s%d", base, depth)
}

// buildCollectionPropsFallbacks creates statements that populate Props fields
// from collection frontmatter metadata. The generated code reads from
// __metadata (a map[string]any) and assigns matching values to props fields
// using their prop tag names as lookup keys, with case-insensitive fallback
// via pikoruntime.MetadataGet.
//
// Takes mainComponent (*annotator_dto.VirtualComponent) which provides the
// component's AST for reading props metadata.
// Takes collectionName (string) which identifies the collection. When empty,
// no fallbacks are generated.
// Takes propsTypeExpr (goast.Expr) which is the resolved props type expression
// from extractPropsTypeFromComponent. Used to skip NoProps components without
// re-walking the AST.
//
// Returns []goast.Stmt which contains the mapping statements to add after
// props type assertion, wrapped in an `if __metadata != nil { ... }` guard.
func buildCollectionPropsFallbacks(mainComponent *annotator_dto.VirtualComponent, collectionName string, propsTypeExpr goast.Expr) []goast.Stmt {
	if collectionName == "" || mainComponent == nil || mainComponent.RewrittenScriptAST == nil {
		return nil
	}

	if sel, ok := propsTypeExpr.(*goast.SelectorExpr); ok && sel.Sel.Name == NoPropsTypeName {
		return nil
	}

	collectionProps := extractCollectionPropsFromAST(mainComponent.RewrittenScriptAST)
	if len(collectionProps) == 0 {
		return nil
	}

	assignments := buildMetadataAssignments(collectionProps, cachedIdent(metadataVarName), cachedIdent("props"), 0)
	if len(assignments) == 0 {
		return nil
	}

	guard := &goast.IfStmt{
		Cond: &goast.BinaryExpr{
			X:  cachedIdent(metadataVarName),
			Op: token.NEQ,
			Y:  cachedIdent(GoKeywordNil),
		},
		Body: &goast.BlockStmt{List: assignments},
	}

	return []goast.Stmt{guard}
}

// extractCollectionPropsFromAST finds Props struct fields with prop tags from
// parsed Go source code.
//
// Takes file (*goast.File) which contains the parsed Go source to search.
//
// Returns []collectionPropInfo which contains the collection property details
// found.
func extractCollectionPropsFromAST(file *goast.File) []collectionPropInfo {
	propsStruct := findPropsStruct(file)
	if propsStruct == nil {
		return nil
	}
	visited := make(map[string]bool)
	visited["Props"] = true
	return extractCollectionPropsFromStruct(file, propsStruct, visited)
}

// extractCollectionPropsFromStruct gets collection property details from a
// struct type.
//
// Takes file (*goast.File) which is the full source file for resolving nested
// struct types.
// Takes structType (*goast.StructType) which is the struct to analyse.
// Takes visited (map[string]bool) which tracks already-visited type names to
// prevent infinite recursion on circular struct definitions.
//
// Returns []collectionPropInfo which contains the properties found.
func extractCollectionPropsFromStruct(file *goast.File, structType *goast.StructType, visited map[string]bool) []collectionPropInfo {
	var result []collectionPropInfo
	for _, field := range structType.Fields.List {
		if field.Tag == nil || len(field.Names) == 0 {
			continue
		}
		if propInfo := parseFieldForCollection(file, field, visited); propInfo != nil {
			result = append(result, *propInfo)
		}
	}
	return result
}

// parseFieldForCollection extracts collection binding details from a struct
// field.
//
// Takes file (*goast.File) which is the source file for resolving nested types.
// Takes field (*goast.Field) which is the struct field to check for prop tags.
// Takes visited (map[string]bool) which tracks already-visited type names to
// prevent infinite recursion on circular struct definitions.
//
// Returns *collectionPropInfo which holds the prop tag name, type, and nested
// props. Returns nil if the field has no prop tag.
func parseFieldForCollection(file *goast.File, field *goast.Field, visited map[string]bool) *collectionPropInfo {
	if field.Tag == nil {
		return nil
	}

	tagValue := strings.Trim(field.Tag.Value, "`")
	tag := reflect.StructTag(tagValue)

	propName, hasProp := tag.Lookup("prop")
	if !hasProp || propName == "" {
		return nil
	}

	if commaIndex := strings.IndexByte(propName, ','); commaIndex >= 0 {
		propName = propName[:commaIndex]
	}

	typeExpr := field.Type
	isPointer := false
	if star, ok := typeExpr.(*goast.StarExpr); ok {
		isPointer = true
		typeExpr = star.X
	}

	info := &collectionPropInfo{
		GoFieldName: field.Names[0].Name,
		PropTagName: propName,
		TypeExpr:    field.Type,
		IsPointer:   isPointer,
	}

	if ident, ok := typeExpr.(*goast.Ident); ok && visited[ident.Name] {
		return info
	}

	nestedStruct := resolveStructType(file, typeExpr)
	if nestedStruct != nil {
		if ident, ok := typeExpr.(*goast.Ident); ok {
			visited[ident.Name] = true
		}
		info.NestedProps = extractCollectionPropsFromStruct(file, nestedStruct, visited)
	}

	return info
}

// resolveStructType attempts to find a struct type definition for a type
// expression. It handles both inline struct types and named types defined
// in the same file.
//
// Takes file (*goast.File) which is the source file containing type
// definitions.
// Takes typeExpr (goast.Expr) which is the type expression to resolve.
//
// Returns *goast.StructType which is the resolved struct, or nil if the
// type is not a struct.
func resolveStructType(file *goast.File, typeExpr goast.Expr) *goast.StructType {
	if structType, ok := typeExpr.(*goast.StructType); ok {
		return structType
	}

	if ident, ok := typeExpr.(*goast.Ident); ok && file != nil {
		return findStructByName(file, ident.Name)
	}

	return nil
}

// findStructByName finds a named struct type definition in a Go file.
//
// Takes file (*goast.File) which is the parsed Go source to search.
// Takes name (string) which is the type name to find.
//
// Returns *goast.StructType which is the struct type if found, or nil if not
// present.
func findStructByName(file *goast.File, name string) *goast.StructType {
	for _, declaration := range file.Decls {
		genDecl, ok := declaration.(*goast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*goast.TypeSpec)
			if !ok || typeSpec.Name.Name != name {
				continue
			}

			structType, ok := typeSpec.Type.(*goast.StructType)
			if ok && structType.Fields != nil {
				return structType
			}
		}
	}
	return nil
}

// buildMetadataAssignments generates AST statements that extract values from a
// metadata map and assign them to props fields. Handles scalar types, nested
// structs, and pointer types.
//
// Takes props ([]collectionPropInfo) which describes the fields to map.
// Takes sourceMap (goast.Expr) which is the metadata map expression to read
// from.
// Takes targetStruct (goast.Expr) which is the struct expression to assign to.
// Takes depth (int) which is the current nesting depth, used for generating
// unique variable names.
//
// Returns []goast.Stmt which contains the generated assignment statements.
func buildMetadataAssignments(props []collectionPropInfo, sourceMap goast.Expr, targetStruct goast.Expr, depth int) []goast.Stmt {
	var statements []goast.Stmt

	for _, prop := range props {
		fieldAccess := &goast.SelectorExpr{
			X:   targetStruct,
			Sel: cachedIdent(prop.GoFieldName),
		}

		var statement goast.Stmt
		if len(prop.NestedProps) > 0 {
			statement = buildNestedStructAssignment(prop, sourceMap, fieldAccess, depth)
		} else {
			statement = buildScalarMetadataAssignment(prop, sourceMap, fieldAccess, depth)
		}

		if statement != nil {
			statements = append(statements, statement)
		}
	}

	return statements
}

// buildScalarMetadataAssignment generates an if statement that extracts a
// scalar value from a metadata map and assigns it to a props field.
//
// The generated pattern varies by type:
//
// For string:
//
//	if __v0, __ok0 := pikoruntime.MetadataGet(source, "key"); __ok0 {
//	    if __typed0, __ok0 := __v0.(string); __ok0 {
//	        target.Field = __typed0
//	    }
//	}
//
// For int (with coercion):
//
//	if __v0, __ok0 := pikoruntime.MetadataGet(source, "key"); __ok0 {
//	    if __typed0, __ok0 := pikoruntime.CoerceInt(__v0); __ok0 {
//	        target.Field = __typed0
//	    }
//	}
//
// Takes prop (collectionPropInfo) which describes the field to map.
// Takes sourceMap (goast.Expr) which is the map to read from.
// Takes fieldAccess (*goast.SelectorExpr) which is the target field.
// Takes depth (int) which is the nesting depth for variable name generation.
//
// Returns goast.Stmt which is the generated if statement, or nil for
// unsupported types.
func buildScalarMetadataAssignment(prop collectionPropInfo, sourceMap goast.Expr, fieldAccess *goast.SelectorExpr, depth int) goast.Stmt {
	baseType := getBaseTypeName(prop.TypeExpr)

	var innerAssignment goast.Stmt

	switch baseType {
	case "string":
		innerAssignment = buildDirectTypeAssertAssignment(fieldAccess, "string", prop.IsPointer, depth)

	case "bool":
		innerAssignment = buildDirectTypeAssertAssignment(fieldAccess, "bool", prop.IsPointer, depth)

	case "int":
		innerAssignment = buildCoercionAssignment(fieldAccess, "CoerceInt", prop.IsPointer, depth)
	case "int8", "int16", "int32", "int64":
		innerAssignment = buildGenericCoercionAssignment(fieldAccess, "CoerceSignedInt", baseType, prop.IsPointer, depth)
	case "uint", "uint8", "uint16", "uint32", "uint64":
		innerAssignment = buildGenericCoercionAssignment(fieldAccess, "CoerceUnsignedInt", baseType, prop.IsPointer, depth)
	case "float64":
		innerAssignment = buildCoercionAssignment(fieldAccess, "CoerceFloat64", prop.IsPointer, depth)
	case "float32":
		innerAssignment = buildCoercionAssignment(fieldAccess, "CoerceFloat32", prop.IsPointer, depth)

	default:

		if isTimeType(prop.TypeExpr) {
			innerAssignment = buildCoercionAssignment(fieldAccess, "CoerceTime", prop.IsPointer, depth)
		} else if isStringSliceType(prop.TypeExpr) {
			innerAssignment = buildCoercionAssignment(fieldAccess, "CoerceStringSlice", prop.IsPointer, depth)
		} else {
			return nil
		}
	}

	if innerAssignment == nil {
		return nil
	}

	return wrapInMetadataLookup(sourceMap, prop.PropTagName, innerAssignment, depth)
}

// buildNestedStructAssignment generates code that extracts a nested map from
// metadata and recursively maps its fields to a nested struct.
//
// Generated pattern:
//
//	if __v0, __ok0 := pikoruntime.MetadataGet(source, "key"); __ok0 {
//	    if __m0, __ok0 := __v0.(map[string]any); __ok0 {
//	        // recursive field assignments at depth+1
//	    }
//	}
//
// Takes prop (collectionPropInfo) which describes the nested struct field.
// Takes sourceMap (goast.Expr) which is the outer metadata map.
// Takes fieldAccess (*goast.SelectorExpr) which is the target struct field.
// Takes depth (int) which is the nesting depth for variable name generation.
//
// Returns goast.Stmt which is the generated nested mapping statement.
func buildNestedStructAssignment(prop collectionPropInfo, sourceMap goast.Expr, fieldAccess *goast.SelectorExpr, depth int) goast.Stmt {
	mapVarName := depthVar("__m", depth)
	nestedMapVar := cachedIdent(mapVarName)

	innerAssignments := buildMetadataAssignments(prop.NestedProps, nestedMapVar, fieldAccess, depth+1)
	if len(innerAssignments) == 0 {
		return nil
	}

	vName := depthVar(valueVarBase, depth)
	okName := depthVar(okVarBase, depth)

	mapAssertStmt := &goast.IfStmt{
		Init: &goast.AssignStmt{
			Lhs: []goast.Expr{nestedMapVar, cachedIdent(okName)},
			Tok: token.DEFINE,
			Rhs: []goast.Expr{
				&goast.TypeAssertExpr{
					X: cachedIdent(vName),
					Type: &goast.MapType{
						Key:   cachedIdent(StringTypeName),
						Value: cachedIdent("any"),
					},
				},
			},
		},
		Cond: cachedIdent(okName),
		Body: &goast.BlockStmt{List: innerAssignments},
	}

	return wrapInMetadataLookup(sourceMap, prop.PropTagName, mapAssertStmt, depth)
}

// buildDirectTypeAssertAssignment generates an if statement that type-asserts
// a value and assigns the result.
//
// Generated pattern:
//
//	if __typed0, __ok0 := __v0.(typeName); __ok0 {
//	    target.Field = __typed0
//	}
//
// Takes fieldAccess (*goast.SelectorExpr) which is the target field.
// Takes typeName (string) which is the Go type to assert.
// Takes isPointer (bool) which indicates whether to take the address of the
// result.
// Takes depth (int) which is the nesting depth for variable name generation.
//
// Returns goast.Stmt which is the generated if statement.
func buildDirectTypeAssertAssignment(fieldAccess *goast.SelectorExpr, typeName string, isPointer bool, depth int) goast.Stmt {
	typedName := depthVar("__typed", depth)
	okName := depthVar(okVarBase, depth)
	vName := depthVar(valueVarBase, depth)

	var assignRHS goast.Expr = cachedIdent(typedName)
	if isPointer {
		assignRHS = &goast.UnaryExpr{Op: token.AND, X: cachedIdent(typedName)}
	}

	return &goast.IfStmt{
		Init: &goast.AssignStmt{
			Lhs: []goast.Expr{cachedIdent(typedName), cachedIdent(okName)},
			Tok: token.DEFINE,
			Rhs: []goast.Expr{
				&goast.TypeAssertExpr{
					X:    cachedIdent(vName),
					Type: cachedIdent(typeName),
				},
			},
		},
		Cond: cachedIdent(okName),
		Body: &goast.BlockStmt{
			List: []goast.Stmt{
				&goast.AssignStmt{
					Lhs: []goast.Expr{cloneSelectorExpr(fieldAccess)},
					Tok: token.ASSIGN,
					Rhs: []goast.Expr{assignRHS},
				},
			},
		},
	}
}

// buildCoercionAssignment generates an if statement that calls a runtime
// coercion function and assigns the result.
//
// Generated pattern:
//
//	if __typed0, __ok0 := pikoruntime.CoerceFunc(__v0); __ok0 {
//	    target.Field = __typed0
//	}
//
// Takes fieldAccess (*goast.SelectorExpr) which is the target field.
// Takes coerceFunc (string) which is the runtime function name to call.
// Takes isPointer (bool) which indicates whether to take the address of the
// result.
// Takes depth (int) which is the nesting depth for variable name generation.
//
// Returns goast.Stmt which is the generated if statement.
func buildCoercionAssignment(fieldAccess *goast.SelectorExpr, coerceFunc string, isPointer bool, depth int) goast.Stmt {
	typedName := depthVar("__typed", depth)
	okName := depthVar(okVarBase, depth)
	vName := depthVar(valueVarBase, depth)

	var assignRHS goast.Expr = cachedIdent(typedName)
	if isPointer {
		assignRHS = &goast.UnaryExpr{Op: token.AND, X: cachedIdent(typedName)}
	}

	return &goast.IfStmt{
		Init: &goast.AssignStmt{
			Lhs: []goast.Expr{cachedIdent(typedName), cachedIdent(okName)},
			Tok: token.DEFINE,
			Rhs: []goast.Expr{
				&goast.CallExpr{
					Fun: &goast.SelectorExpr{
						X:   cachedIdent(runtimePackageName),
						Sel: cachedIdent(coerceFunc),
					},
					Args: []goast.Expr{cachedIdent(vName)},
				},
			},
		},
		Cond: cachedIdent(okName),
		Body: &goast.BlockStmt{
			List: []goast.Stmt{
				&goast.AssignStmt{
					Lhs: []goast.Expr{cloneSelectorExpr(fieldAccess)},
					Tok: token.ASSIGN,
					Rhs: []goast.Expr{assignRHS},
				},
			},
		},
	}
}

// buildGenericCoercionAssignment generates an if statement that calls a
// generic runtime coercion function with a type parameter and assigns the
// result.
//
// Generated pattern:
//
//	if __typed0, __ok0 := pikoruntime.CoerceSignedInt[int32](__v0); __ok0 {
//	    target.Field = __typed0
//	}
//
// Takes fieldAccess (*goast.SelectorExpr) which is the target field.
// Takes coerceFunc (string) which is the runtime function name to call.
// Takes typeParam (string) which is the Go type name for the generic parameter.
// Takes isPointer (bool) which indicates whether to take the address of the
// result.
// Takes depth (int) which is the nesting depth for variable name generation.
//
// Returns goast.Stmt which is the generated if statement.
func buildGenericCoercionAssignment(fieldAccess *goast.SelectorExpr, coerceFunc string, typeParam string, isPointer bool, depth int) goast.Stmt {
	typedName := depthVar("__typed", depth)
	okName := depthVar(okVarBase, depth)
	vName := depthVar(valueVarBase, depth)

	var assignRHS goast.Expr = cachedIdent(typedName)
	if isPointer {
		assignRHS = &goast.UnaryExpr{Op: token.AND, X: cachedIdent(typedName)}
	}

	return &goast.IfStmt{
		Init: &goast.AssignStmt{
			Lhs: []goast.Expr{cachedIdent(typedName), cachedIdent(okName)},
			Tok: token.DEFINE,
			Rhs: []goast.Expr{
				&goast.CallExpr{
					Fun: &goast.IndexExpr{
						X: &goast.SelectorExpr{
							X:   cachedIdent(runtimePackageName),
							Sel: cachedIdent(coerceFunc),
						},
						Index: cachedIdent(typeParam),
					},
					Args: []goast.Expr{cachedIdent(vName)},
				},
			},
		},
		Cond: cachedIdent(okName),
		Body: &goast.BlockStmt{
			List: []goast.Stmt{
				&goast.AssignStmt{
					Lhs: []goast.Expr{cloneSelectorExpr(fieldAccess)},
					Tok: token.ASSIGN,
					Rhs: []goast.Expr{assignRHS},
				},
			},
		},
	}
}

// wrapInMetadataLookup wraps an inner statement in a case-insensitive metadata
// map lookup via pikoruntime.MetadataGet.
//
// Generated pattern:
//
//	if __v0, __ok0 := pikoruntime.MetadataGet(source, "key"); __ok0 {
//	    <innerStatement>
//	}
//
// Takes sourceMap (goast.Expr) which is the map to search.
// Takes key (string) which is the map key to look up.
// Takes innerStatement (goast.Stmt) which is the statement to run when the
// key exists.
// Takes depth (int) which is the nesting depth for variable name generation.
//
// Returns *goast.IfStmt which is the wrapping if statement.
func wrapInMetadataLookup(sourceMap goast.Expr, key string, innerStatement goast.Stmt, depth int) *goast.IfStmt {
	vName := depthVar(valueVarBase, depth)
	okName := depthVar(okVarBase, depth)

	return &goast.IfStmt{
		Init: &goast.AssignStmt{
			Lhs: []goast.Expr{cachedIdent(vName), cachedIdent(okName)},
			Tok: token.DEFINE,
			Rhs: []goast.Expr{
				&goast.CallExpr{
					Fun: &goast.SelectorExpr{
						X:   cachedIdent(runtimePackageName),
						Sel: cachedIdent("MetadataGet"),
					},
					Args: []goast.Expr{sourceMap, strLit(key)},
				},
			},
		},
		Cond: cachedIdent(okName),
		Body: &goast.BlockStmt{List: []goast.Stmt{innerStatement}},
	}
}

// isTimeType checks whether a type expression refers to time.Time.
//
// Takes typeExpr (goast.Expr) which is the type expression to check.
//
// Returns bool which is true when the type is time.Time.
func isTimeType(typeExpr goast.Expr) bool {
	if star, ok := typeExpr.(*goast.StarExpr); ok {
		typeExpr = star.X
	}
	sel, ok := typeExpr.(*goast.SelectorExpr)
	if !ok {
		return false
	}
	xIdent, ok := sel.X.(*goast.Ident)
	if !ok {
		return false
	}
	return xIdent.Name == "time" && sel.Sel.Name == "Time"
}

// isStringSliceType checks whether a type expression refers to []string.
//
// Takes typeExpr (goast.Expr) which is the type expression to check.
//
// Returns bool which is true when the type is []string.
func isStringSliceType(typeExpr goast.Expr) bool {
	if star, ok := typeExpr.(*goast.StarExpr); ok {
		typeExpr = star.X
	}
	arrayType, ok := typeExpr.(*goast.ArrayType)
	if !ok {
		return false
	}
	ident, ok := arrayType.Elt.(*goast.Ident)
	if !ok {
		return false
	}
	return ident.Name == "string"
}
