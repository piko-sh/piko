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
	"reflect"
	"strings"

	goast "go/ast"
	"go/token"

	"piko.sh/piko/internal/annotator/annotator_dto"
)

const (
	// goTypeInt is the Go type name for int, used in switch cases
	// and safeconv dispatch.
	goTypeInt = "int"

	// goTypeInt64 is the Go type name for int64, used in switch
	// cases and safeconv dispatch.
	goTypeInt64 = "int64"

	// goTypeUint is the Go type name for uint, used in switch
	// cases and safeconv dispatch.
	goTypeUint = "uint"

	// goTypeFloat32 is the Go type name for float32, used in
	// switch cases and safeconv dispatch.
	goTypeFloat32 = "float32"
)

// queryPropInfo holds details about a struct field that binds to a URL query
// parameter. Fields are ordered for memory alignment.
type queryPropInfo struct {
	// TypeExpr is the Go AST expression for the field type.
	TypeExpr goast.Expr

	// GoFieldName is the name of the struct field in Go code.
	GoFieldName string

	// QueryParamName is the name of the URL query parameter.
	QueryParamName string

	// IsPointer indicates whether the field type is a pointer.
	IsPointer bool

	// ShouldCoerce indicates whether type conversion is enabled for the field.
	ShouldCoerce bool
}

// buildQueryParamFallbacks creates statements that set props from query
// parameters when the parent component has not set them.
//
// Takes mainComponent (*annotator_dto.VirtualComponent) which provides the
// component's AST for reading props metadata.
//
// Returns []goast.Stmt which contains the fallback statements to add after
// props type assertion.
func buildQueryParamFallbacks(mainComponent *annotator_dto.VirtualComponent) []goast.Stmt {
	if mainComponent == nil || mainComponent.RewrittenScriptAST == nil {
		return nil
	}

	queryProps := extractQueryPropsFromAST(mainComponent.RewrittenScriptAST)
	if len(queryProps) == 0 {
		return nil
	}

	var statements []goast.Stmt
	for _, prop := range queryProps {
		statement := buildQueryFallbackStatement(prop)
		if statement != nil {
			statements = append(statements, statement)
		}
	}
	return statements
}

// extractQueryPropsFromAST finds properties with query tags from parsed Go
// source code of a component script.
//
// Takes file (*goast.File) which contains the parsed Go source to search.
//
// Returns []queryPropInfo which contains the query property details found.
func extractQueryPropsFromAST(file *goast.File) []queryPropInfo {
	propsStruct := findPropsStruct(file)
	if propsStruct == nil {
		return nil
	}
	return extractQueryPropsFromStruct(propsStruct)
}

// findPropsStruct finds the Props struct type in a Go file.
//
// Takes file (*goast.File) which is the parsed Go source to search.
//
// Returns *goast.StructType which is the Props struct if found, or nil if not
// present.
func findPropsStruct(file *goast.File) *goast.StructType {
	return findStructByName(file, "Props")
}

// extractQueryPropsFromStruct gets query property details from a struct type.
//
// Takes structType (*goast.StructType) which is the Props struct to analyse.
//
// Returns []queryPropInfo which contains the query properties found.
func extractQueryPropsFromStruct(structType *goast.StructType) []queryPropInfo {
	var result []queryPropInfo
	for _, field := range structType.Fields.List {
		if field.Tag == nil || len(field.Names) == 0 {
			continue
		}
		if queryInfo := parseFieldForQuery(field); queryInfo != nil {
			result = append(result, *queryInfo)
		}
	}
	return result
}

// parseFieldForQuery extracts query binding details from a struct field.
//
// Takes field (*goast.Field) which is the struct field to check for query tags.
//
// Returns *queryPropInfo which holds the query parameter name, type, and
// options. Returns nil if the field has no query tag.
func parseFieldForQuery(field *goast.Field) *queryPropInfo {
	if field.Tag == nil {
		return nil
	}

	tagValue := strings.Trim(field.Tag.Value, "`")
	tag := reflect.StructTag(tagValue)

	queryParam, hasQuery := tag.Lookup("query")
	if !hasQuery || queryParam == "" {
		return nil
	}

	_, hasCoerce := tag.Lookup("coerce")

	typeExpr := field.Type
	isPointer := false
	if _, ok := typeExpr.(*goast.StarExpr); ok {
		isPointer = true
	}

	return &queryPropInfo{
		GoFieldName:    field.Names[0].Name,
		QueryParamName: queryParam,
		TypeExpr:       typeExpr,
		IsPointer:      isPointer,
		ShouldCoerce:   hasCoerce,
	}
}

// buildQueryFallbackStatement creates a fallback statement for a single query
// parameter property.
//
// For string types:
//
//	if props.Field == "" {
//	    props.Field = r.QueryParam("param_name")
//	}
//
// For *string types:
//
//	if props.Field == nil {
//	    if qp := r.QueryParam("param_name"); qp != "" {
//	        props.Field = &qp
//	    }
//	}
//
// For int types with coerce:
//
//	if props.Field == 0 {
//	    if qp := r.QueryParam("param_name"); qp != "" {
//	        if v, err := strconv.Atoi(qp); err == nil {
//	            props.Field = v
//	        }
//	    }
//	}
//
// Takes prop (queryPropInfo) which describes the query parameter to create a
// fallback for.
//
// Returns goast.Stmt which is the fallback statement, or nil if the property
// type does not support fallback creation.
func buildQueryFallbackStatement(prop queryPropInfo) goast.Stmt {
	fieldAccess := &goast.SelectorExpr{
		X:   cachedIdent("props"),
		Sel: cachedIdent(prop.GoFieldName),
	}

	queryCall := &goast.CallExpr{
		Fun: &goast.SelectorExpr{
			X:   cachedIdent(RequestVarName),
			Sel: cachedIdent("QueryParam"),
		},
		Args: []goast.Expr{strLit(prop.QueryParamName)},
	}

	if prop.IsPointer {
		return buildPointerQueryFallback(prop, fieldAccess, queryCall)
	}

	baseType := getBaseTypeName(prop.TypeExpr)

	switch baseType {
	case "string":
		return buildStringQueryFallback(fieldAccess, queryCall)
	case goTypeInt, "int8", "int16", "int32", goTypeInt64:
		if prop.ShouldCoerce {
			return buildIntQueryFallback(fieldAccess, queryCall, baseType)
		}
	case goTypeUint, "uint8", "uint16", "uint32", "uint64":
		if prop.ShouldCoerce {
			return buildUintQueryFallback(fieldAccess, queryCall, baseType)
		}
	case goTypeFloat32, "float64":
		if prop.ShouldCoerce {
			return buildFloatQueryFallback(fieldAccess, queryCall, baseType)
		}
	case "bool":
		if prop.ShouldCoerce {
			return buildBoolQueryFallback(fieldAccess, queryCall)
		}
	}

	return nil
}

// getBaseTypeName extracts the base type name from a type expression.
//
// Takes typeExpr (goast.Expr) which is the type expression to extract from.
//
// Returns string which is the base type name, or an empty string if the type
// is not a simple identifier.
func getBaseTypeName(typeExpr goast.Expr) string {
	if star, ok := typeExpr.(*goast.StarExpr); ok {
		typeExpr = star.X
	}

	if identifier, ok := typeExpr.(*goast.Ident); ok {
		return identifier.Name
	}
	return ""
}

// buildStringQueryFallback builds an if statement that assigns a query
// parameter value when the field is empty.
//
// Takes fieldAccess (*goast.SelectorExpr) which is the field to check and
// assign to.
// Takes queryCall (*goast.CallExpr) which is the query parameter call to use
// as the fallback value.
//
// Returns goast.Stmt which is an if statement of the form:
// if props.Field == "" { props.Field = r.QueryParam("name") }.
func buildStringQueryFallback(fieldAccess *goast.SelectorExpr, queryCall *goast.CallExpr) goast.Stmt {
	return &goast.IfStmt{
		Cond: &goast.BinaryExpr{
			X:  fieldAccess,
			Op: token.EQL,
			Y:  strLit(""),
		},
		Body: &goast.BlockStmt{
			List: []goast.Stmt{
				&goast.AssignStmt{
					Lhs: []goast.Expr{cloneSelectorExpr(fieldAccess)},
					Tok: token.ASSIGN,
					Rhs: []goast.Expr{queryCall},
				},
			},
		},
	}
}

// buildPointerQueryFallback builds fallback AST for pointer types.
//
// The generated code follows this pattern:
//
//	if props.Field == nil {
//	    if qp := r.QueryParam("name"); qp != "" {
//	        props.Field = &qp
//	    }
//	}
//
// Takes prop (queryPropInfo) which describes the property being handled.
// Takes fieldAccess (*goast.SelectorExpr) which is the field selector.
// Takes queryCall (*goast.CallExpr) which gets the query parameter value.
//
// Returns goast.Stmt which is the fallback statement, or nil if the type is
// not supported or coercion is not enabled.
func buildPointerQueryFallback(prop queryPropInfo, fieldAccess *goast.SelectorExpr, queryCall *goast.CallExpr) goast.Stmt {
	innerBody := buildPointerInnerBody(prop, fieldAccess)
	if innerBody == nil {
		return nil
	}
	return wrapInNilCheckWithQueryInit(fieldAccess, queryCall, innerBody)
}

// buildPointerInnerBody creates inner body statements for pointer type fallback.
//
// Takes prop (queryPropInfo) which describes the property being processed.
// Takes fieldAccess (*goast.SelectorExpr) which is the field selector expression.
//
// Returns []goast.Stmt for the inner body, or nil if type is unsupported.
func buildPointerInnerBody(prop queryPropInfo, fieldAccess *goast.SelectorExpr) []goast.Stmt {
	baseType := getBaseTypeName(prop.TypeExpr)

	switch baseType {
	case "string":
		return buildPointerStringAssignment(fieldAccess)
	case goTypeInt, "int8", "int16", "int32", goTypeInt64:
		if !prop.ShouldCoerce {
			return nil
		}
		return buildPointerIntParseAssignment(fieldAccess, baseType)
	case "bool":
		if !prop.ShouldCoerce {
			return nil
		}
		return buildPointerBoolParseAssignment(fieldAccess)
	default:
		return nil
	}
}

// buildPointerStringAssignment creates an assignment statement for pointer
// string types.
//
// Takes fieldAccess (*goast.SelectorExpr) which specifies the struct field to
// assign to.
//
// Returns []goast.Stmt which contains the assignment statement that takes the
// address of the cached query parameter variable.
func buildPointerStringAssignment(fieldAccess *goast.SelectorExpr) []goast.Stmt {
	return []goast.Stmt{
		&goast.AssignStmt{
			Lhs: []goast.Expr{cloneSelectorExpr(fieldAccess)},
			Tok: token.ASSIGN,
			Rhs: []goast.Expr{&goast.UnaryExpr{Op: token.AND, X: cachedIdent(varNameQP)}},
		},
	}
}

// buildPointerIntParseAssignment creates AST statements that parse a string to
// an integer and assign the result to a pointer field. For narrowing types
// (int8, int16, int32), the value is converted via safeconv before taking its
// address.
//
// Takes fieldAccess (*goast.SelectorExpr) which specifies the struct field to
// receive the parsed integer value.
// Takes baseType (string) which is the target integer type (e.g. "int32").
//
// Returns []goast.Stmt which contains the AST statements for parsing and
// assignment.
func buildPointerIntParseAssignment(fieldAccess *goast.SelectorExpr, baseType string) []goast.Stmt {
	parseCall := &goast.CallExpr{
		Fun:  &goast.SelectorExpr{X: cachedIdent(pkgStrconv), Sel: cachedIdent("Atoi")},
		Args: []goast.Expr{cachedIdent(varNameQP)},
	}

	if baseType == goTypeInt {
		return []goast.Stmt{buildParseAndAssignPointer(parseCall, fieldAccess)}
	}

	convertExpr := safeconvIntExpr(baseType)
	return []goast.Stmt{
		buildParseConvertAndAssignPointer(parseCall, convertExpr, fieldAccess),
	}
}

// buildParseConvertAndAssignPointer creates an if statement that parses a
// value, converts it, and assigns its address to a pointer field.
//
// Generated code:
//
//	if v, err := strconv.Atoi(qp); err == nil {
//	    _c := safeconv.IntToInt32(v)
//	    props.Field = &_c
//	}
//
// Takes parseCall (*goast.CallExpr) which is the parse function call.
// Takes convertExpr (goast.Expr) which is the conversion expression applied
// to v.
// Takes fieldAccess (*goast.SelectorExpr) which is the pointer field to set.
//
// Returns *goast.IfStmt which wraps the parse, convert, and assign.
func buildParseConvertAndAssignPointer(parseCall *goast.CallExpr, convertExpr goast.Expr, fieldAccess *goast.SelectorExpr) *goast.IfStmt {
	const convertedVar = "_c"
	return &goast.IfStmt{
		Init: &goast.AssignStmt{
			Lhs: []goast.Expr{cachedIdent(varNameV), cachedIdent(varNameErr)},
			Tok: token.DEFINE,
			Rhs: []goast.Expr{parseCall},
		},
		Cond: &goast.BinaryExpr{
			X:  cachedIdent(varNameErr),
			Op: token.EQL,
			Y:  cachedIdent(GoKeywordNil),
		},
		Body: &goast.BlockStmt{
			List: []goast.Stmt{
				&goast.AssignStmt{
					Lhs: []goast.Expr{cachedIdent(convertedVar)},
					Tok: token.DEFINE,
					Rhs: []goast.Expr{convertExpr},
				},
				&goast.AssignStmt{
					Lhs: []goast.Expr{cloneSelectorExpr(fieldAccess)},
					Tok: token.ASSIGN,
					Rhs: []goast.Expr{&goast.UnaryExpr{Op: token.AND, X: cachedIdent(convertedVar)}},
				},
			},
		},
	}
}

// buildPointerBoolParseAssignment creates statements that parse a string to a
// bool and assign the result to a pointer field.
//
// Takes fieldAccess (*goast.SelectorExpr) which specifies the target field for
// the parsed bool value.
//
// Returns []goast.Stmt which contains the AST statements for parsing and
// assignment.
func buildPointerBoolParseAssignment(fieldAccess *goast.SelectorExpr) []goast.Stmt {
	return []goast.Stmt{
		buildParseAndAssignPointer(
			&goast.CallExpr{
				Fun:  &goast.SelectorExpr{X: cachedIdent(pkgStrconv), Sel: cachedIdent("ParseBool")},
				Args: []goast.Expr{cachedIdent(varNameQP)},
			},
			fieldAccess,
		),
	}
}

// buildParseAndAssignPointer creates an if statement that parses a value and
// assigns its address to a pointer field.
//
// Takes parseCall (*goast.CallExpr) which is the parse function call to run.
// Takes fieldAccess (*goast.SelectorExpr) which is the pointer field to set.
//
// Returns *goast.IfStmt which assigns the address of the parsed value to the
// field when parsing succeeds.
func buildParseAndAssignPointer(parseCall *goast.CallExpr, fieldAccess *goast.SelectorExpr) *goast.IfStmt {
	return &goast.IfStmt{
		Init: &goast.AssignStmt{
			Lhs: []goast.Expr{cachedIdent(varNameV), cachedIdent(varNameErr)},
			Tok: token.DEFINE,
			Rhs: []goast.Expr{parseCall},
		},
		Cond: &goast.BinaryExpr{
			X:  cachedIdent(varNameErr),
			Op: token.EQL,
			Y:  cachedIdent(GoKeywordNil),
		},
		Body: &goast.BlockStmt{
			List: []goast.Stmt{
				&goast.AssignStmt{
					Lhs: []goast.Expr{cloneSelectorExpr(fieldAccess)},
					Tok: token.ASSIGN,
					Rhs: []goast.Expr{&goast.UnaryExpr{Op: token.AND, X: cachedIdent(varNameV)}},
				},
			},
		},
	}
}

// wrapInNilCheckWithQueryInit wraps the inner body in a nil check and query
// parameter setup.
//
// Takes fieldAccess (*goast.SelectorExpr) which is the field to check for nil.
// Takes queryCall (*goast.CallExpr) which sets up the query parameter.
// Takes innerBody ([]goast.Stmt) which contains the statements to run when the
// query parameter is not empty.
//
// Returns *goast.IfStmt which is an if statement that first checks if the field
// is nil, then sets up the query parameter and runs the inner body if the
// parameter is not empty.
func wrapInNilCheckWithQueryInit(fieldAccess *goast.SelectorExpr, queryCall *goast.CallExpr, innerBody []goast.Stmt) *goast.IfStmt {
	innerIf := &goast.IfStmt{
		Init: &goast.AssignStmt{
			Lhs: []goast.Expr{cachedIdent(varNameQP)},
			Tok: token.DEFINE,
			Rhs: []goast.Expr{queryCall},
		},
		Cond: &goast.BinaryExpr{
			X:  cachedIdent(varNameQP),
			Op: token.NEQ,
			Y:  strLit(""),
		},
		Body: &goast.BlockStmt{List: innerBody},
	}

	return &goast.IfStmt{
		Cond: &goast.BinaryExpr{
			X:  fieldAccess,
			Op: token.EQL,
			Y:  cachedIdent(GoKeywordNil),
		},
		Body: &goast.BlockStmt{List: []goast.Stmt{innerIf}},
	}
}

// buildIntQueryFallback builds a fallback AST for int types that converts
// query parameters. For narrowing conversions (int8, int16, int32), the
// generated code uses safeconv to clamp the value to the target range.
//
// For int and int64 (no narrowing):
//
//	if props.Field == 0 {
//	    if qp := r.QueryParam("name"); qp != "" {
//	        if v, err := strconv.Atoi(qp); err == nil {
//	            props.Field = v
//	        }
//	    }
//	}
//
// For int32 (narrowing via safeconv):
//
//	if props.Field == 0 {
//	    if qp := r.QueryParam("name"); qp != "" {
//	        if v, err := strconv.Atoi(qp); err == nil {
//	            props.Field = safeconv.IntToInt32(v)
//	        }
//	    }
//	}
//
// Takes fieldAccess (*goast.SelectorExpr) which is the struct field to check
// and assign.
// Takes queryCall (*goast.CallExpr) which gets the query parameter value.
// Takes baseType (string) which is the target integer type name.
//
// Returns goast.Stmt which is the complete if-statement AST node.
func buildIntQueryFallback(fieldAccess *goast.SelectorExpr, queryCall *goast.CallExpr, baseType string) goast.Stmt {
	parseCall := &goast.CallExpr{
		Fun:  &goast.SelectorExpr{X: cachedIdent(pkgStrconv), Sel: cachedIdent("Atoi")},
		Args: []goast.Expr{cachedIdent(varNameQP)},
	}
	assignExpr := safeconvIntExpr(baseType)
	return buildZeroCheckWithParseFallback(fieldAccess, queryCall, parseCall, assignExpr, intLit(0))
}

// safeconvIntExpr returns the assignment expression for converting
// the parsed int value (v) to the target integer type.
//
// Takes baseType (string) which is the target type name.
//
// Returns goast.Expr which is either the bare identifier v or a
// safeconv call.
func safeconvIntExpr(baseType string) goast.Expr {
	switch baseType {
	case goTypeInt:
		return cachedIdent(varNameV)
	case goTypeInt64:
		return &goast.CallExpr{
			Fun:  cachedIdent(goTypeInt64),
			Args: []goast.Expr{cachedIdent(varNameV)},
		}
	default:

		funcName := safeconvFuncName("Int", baseType)
		return &goast.CallExpr{
			Fun:  &goast.SelectorExpr{X: cachedIdent(pkgSafeconv), Sel: cachedIdent(funcName)},
			Args: []goast.Expr{cachedIdent(varNameV)},
		}
	}
}

// safeconvFuncName returns the safeconv function name for converting from a
// source type prefix to the target type. For example, ("Int", "int32")
// returns "IntToInt32".
//
// Takes sourcePrefix (string) which is the source type prefix (e.g. "Int",
// "Uint64").
// Takes targetType (string) which is the target Go type name (e.g. "int32",
// "uint16").
//
// Returns string which is the safeconv function name.
func safeconvFuncName(sourcePrefix, targetType string) string {
	suffix := strings.ToUpper(targetType[:1]) + targetType[1:]
	return sourcePrefix + "To" + suffix
}

// buildUintQueryFallback creates fallback logic for uint types. For narrowing
// conversions (uint8, uint16, uint32), the generated code uses safeconv to
// clamp the value.
//
// Takes fieldAccess (*goast.SelectorExpr) which identifies the struct field to
// set.
// Takes queryCall (*goast.CallExpr) which gets the query parameter value.
// Takes baseType (string) which is the target unsigned integer type name.
//
// Returns goast.Stmt which is an if statement that parses and sets the uint
// value when the field is zero.
func buildUintQueryFallback(fieldAccess *goast.SelectorExpr, queryCall *goast.CallExpr, baseType string) goast.Stmt {
	parseCall := &goast.CallExpr{
		Fun: &goast.SelectorExpr{X: cachedIdent(pkgStrconv), Sel: cachedIdent("ParseUint")},
		Args: []goast.Expr{
			cachedIdent(varNameQP),
			intLit(numericBaseDecimal),
			intLit(bitSize64),
		},
	}
	assignExpr := safeconvUintExpr(baseType)
	return buildZeroCheckWithParseFallback(fieldAccess, queryCall, parseCall, assignExpr, intLit(0))
}

// safeconvUintExpr returns the assignment expression for converting
// the parsed uint64 value (v) to the target unsigned integer type.
//
// Takes baseType (string) which is the target type name.
//
// Returns goast.Expr which is either a simple cast or a safeconv
// call.
func safeconvUintExpr(baseType string) goast.Expr {
	switch baseType {
	case "uint64":
		return cachedIdent(varNameV)
	case goTypeUint:
		return &goast.CallExpr{
			Fun:  cachedIdent(goTypeUint),
			Args: []goast.Expr{cachedIdent(varNameV)},
		}
	default:

		funcName := safeconvFuncName("Uint64", baseType)
		return &goast.CallExpr{
			Fun:  &goast.SelectorExpr{X: cachedIdent(pkgSafeconv), Sel: cachedIdent(funcName)},
			Args: []goast.Expr{cachedIdent(varNameV)},
		}
	}
}

// buildFloatQueryFallback builds a fallback AST statement for float types
// parsed from query parameters.
//
// Takes fieldAccess (*goast.SelectorExpr) which identifies the struct field
// to assign.
// Takes queryCall (*goast.CallExpr) which retrieves the query parameter value.
// Takes baseType (string) which specifies the float type (float32 or float64).
//
// Returns goast.Stmt which is an if statement that parses and assigns the
// query parameter when the field is zero.
func buildFloatQueryFallback(fieldAccess *goast.SelectorExpr, queryCall *goast.CallExpr, baseType string) goast.Stmt {
	bitSize := bitSize64
	if baseType == goTypeFloat32 {
		bitSize = bitSize32
	}

	parseCall := &goast.CallExpr{
		Fun: &goast.SelectorExpr{X: cachedIdent(pkgStrconv), Sel: cachedIdent("ParseFloat")},
		Args: []goast.Expr{
			cachedIdent(varNameQP),
			intLit(bitSize),
		},
	}

	var assignExpr goast.Expr = cachedIdent(varNameV)
	if baseType == goTypeFloat32 {
		assignExpr = &goast.CallExpr{
			Fun:  cachedIdent(goTypeFloat32),
			Args: []goast.Expr{cachedIdent(varNameV)},
		}
	}

	zeroLit := &goast.BasicLit{Kind: token.FLOAT, Value: "0"}
	return buildZeroCheckWithParseFallback(fieldAccess, queryCall, parseCall, assignExpr, zeroLit)
}

// buildZeroCheckWithParseFallback builds a fallback statement that checks if a
// field is zero, gets a query parameter, parses it, and assigns the result.
//
// Takes fieldAccess (*goast.SelectorExpr) which is the field to check.
// Takes queryCall (*goast.CallExpr) which gets the query parameter value.
// Takes parseCall (*goast.CallExpr) which parses the query parameter string.
// Takes assignExpr (goast.Expr) which is the value to assign to the field.
// Takes zeroLit (goast.Expr) which is the zero value for comparison.
//
// Returns *goast.IfStmt which wraps the check, query, parse, and assignment.
func buildZeroCheckWithParseFallback(
	fieldAccess *goast.SelectorExpr,
	queryCall *goast.CallExpr,
	parseCall *goast.CallExpr,
	assignExpr goast.Expr,
	zeroLit goast.Expr,
) *goast.IfStmt {
	innerParse := buildParseAndAssignValue(parseCall, fieldAccess, assignExpr)
	innerQuery := buildQueryParamCheck(queryCall, []goast.Stmt{innerParse})

	return &goast.IfStmt{
		Cond: &goast.BinaryExpr{X: fieldAccess, Op: token.EQL, Y: zeroLit},
		Body: &goast.BlockStmt{List: []goast.Stmt{innerQuery}},
	}
}

// buildParseAndAssignValue creates an if statement that parses a value and
// assigns it only when there is no error.
//
// Takes parseCall (*goast.CallExpr) which is the call to the parsing function.
// Takes fieldAccess (*goast.SelectorExpr) which is the field to assign to.
// Takes assignExpr (goast.Expr) which is the value to assign on success.
//
// Returns *goast.IfStmt which checks for a nil error before assigning.
func buildParseAndAssignValue(parseCall *goast.CallExpr, fieldAccess *goast.SelectorExpr, assignExpr goast.Expr) *goast.IfStmt {
	return &goast.IfStmt{
		Init: &goast.AssignStmt{
			Lhs: []goast.Expr{cachedIdent(varNameV), cachedIdent(varNameErr)},
			Tok: token.DEFINE,
			Rhs: []goast.Expr{parseCall},
		},
		Cond: &goast.BinaryExpr{
			X:  cachedIdent(varNameErr),
			Op: token.EQL,
			Y:  cachedIdent(GoKeywordNil),
		},
		Body: &goast.BlockStmt{
			List: []goast.Stmt{
				&goast.AssignStmt{
					Lhs: []goast.Expr{cloneSelectorExpr(fieldAccess)},
					Tok: token.ASSIGN,
					Rhs: []goast.Expr{assignExpr},
				},
			},
		},
	}
}

// buildQueryParamCheck creates an if statement that fetches and checks a query
// parameter.
//
// Takes queryCall (*goast.CallExpr) which is the call expression that gets the
// query parameter value.
// Takes innerBody ([]goast.Stmt) which contains the statements to run when the
// parameter is present.
//
// Returns *goast.IfStmt which assigns the query result to a variable and runs
// innerBody when the value is not empty.
func buildQueryParamCheck(queryCall *goast.CallExpr, innerBody []goast.Stmt) *goast.IfStmt {
	return &goast.IfStmt{
		Init: &goast.AssignStmt{
			Lhs: []goast.Expr{cachedIdent(varNameQP)},
			Tok: token.DEFINE,
			Rhs: []goast.Expr{queryCall},
		},
		Cond: &goast.BinaryExpr{
			X:  cachedIdent(varNameQP),
			Op: token.NEQ,
			Y:  strLit(""),
		},
		Body: &goast.BlockStmt{List: innerBody},
	}
}

// buildBoolQueryFallback builds a fallback statement for boolean type fields.
// It converts query parameters into boolean values.
//
// Takes fieldAccess (*goast.SelectorExpr) which specifies the target field to
// assign the parsed boolean value.
// Takes queryCall (*goast.CallExpr) which gets the query parameter value.
//
// Returns goast.Stmt which is an if statement that parses the query parameter
// as a boolean and assigns it to the field when valid.
func buildBoolQueryFallback(fieldAccess *goast.SelectorExpr, queryCall *goast.CallExpr) goast.Stmt {
	return &goast.IfStmt{
		Init: &goast.AssignStmt{
			Lhs: []goast.Expr{cachedIdent(varNameQP)},
			Tok: token.DEFINE,
			Rhs: []goast.Expr{queryCall},
		},
		Cond: &goast.BinaryExpr{
			X:  cachedIdent(varNameQP),
			Op: token.NEQ,
			Y:  strLit(""),
		},
		Body: &goast.BlockStmt{
			List: []goast.Stmt{
				&goast.IfStmt{
					Init: &goast.AssignStmt{
						Lhs: []goast.Expr{cachedIdent(varNameV), cachedIdent(varNameErr)},
						Tok: token.DEFINE,
						Rhs: []goast.Expr{
							&goast.CallExpr{
								Fun:  &goast.SelectorExpr{X: cachedIdent(pkgStrconv), Sel: cachedIdent("ParseBool")},
								Args: []goast.Expr{cachedIdent(varNameQP)},
							},
						},
					},
					Cond: &goast.BinaryExpr{
						X:  cachedIdent(varNameErr),
						Op: token.EQL,
						Y:  cachedIdent(GoKeywordNil),
					},
					Body: &goast.BlockStmt{
						List: []goast.Stmt{
							&goast.AssignStmt{
								Lhs: []goast.Expr{cloneSelectorExpr(fieldAccess)},
								Tok: token.ASSIGN,
								Rhs: []goast.Expr{cachedIdent(varNameV)},
							},
						},
					},
				},
			},
		},
	}
}

// cloneSelectorExpr creates a shallow copy of a selector expression.
//
// Takes expression (*goast.SelectorExpr) which is the selector expression to copy.
//
// Returns *goast.SelectorExpr which is a new node with the same X and Sel
// values as the original.
func cloneSelectorExpr(expression *goast.SelectorExpr) *goast.SelectorExpr {
	return &goast.SelectorExpr{
		X:   expression.X,
		Sel: expression.Sel,
	}
}

// isPrimitiveQueryType checks whether a base type name is a primitive that can
// be serialised to a query parameter string.
//
// Takes baseType (string) which is the Go type name to check.
//
// Returns bool which is true when the type is a serialisable primitive.
func isPrimitiveQueryType(baseType string) bool {
	switch baseType {
	case "string",
		goTypeInt, "int8", "int16", "int32", goTypeInt64,
		goTypeUint, "uint8", "uint16", "uint32", "uint64",
		goTypeFloat32, "float64",
		"bool":
		return true
	}
	return false
}

// extractPrimitiveQueryPropsFromComponent returns query-tagged props from a
// VirtualComponent's script AST that have primitive, non-pointer types. Used to
// determine whether a public partial needs a partial_props attribute and which
// fields to include in the query string.
//
// Takes component (*annotator_dto.VirtualComponent) which provides the
// rewritten script AST containing the Props struct.
//
// Returns []queryPropInfo which contains the primitive query-bound props, or
// nil when none exist.
func extractPrimitiveQueryPropsFromComponent(component *annotator_dto.VirtualComponent) []queryPropInfo {
	if component == nil || component.RewrittenScriptAST == nil {
		return nil
	}
	allQueryProps := extractQueryPropsFromAST(component.RewrittenScriptAST)
	if len(allQueryProps) == 0 {
		return nil
	}
	var primitiveProps []queryPropInfo
	for _, prop := range allQueryProps {
		if prop.IsPointer {
			continue
		}
		baseType := getBaseTypeName(prop.TypeExpr)
		if baseType != "" && isPrimitiveQueryType(baseType) {
			primitiveProps = append(primitiveProps, prop)
		}
	}
	return primitiveProps
}
