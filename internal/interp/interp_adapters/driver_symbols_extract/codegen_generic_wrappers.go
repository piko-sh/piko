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

package driver_symbols_extract

import (
	"go/ast"
	"go/token"
	"go/types"
	"slices"
	"strings"

	"piko.sh/piko/internal/goastutil"
)

const (
	// panicKeyword is the Go builtin name emitted when a generated default clause must panic
	// on an unsupported dispatch type.
	panicKeyword = "panic"

	// genericFallbackPanic is the generic_fallback manifest mode that suppresses reflective
	// default clauses, leaving wrappers to panic on unsupported dispatch types.
	genericFallbackPanic = "panic"

	// fallbackParamX names the input slice parameter shared by the reflect fallback helpers.
	fallbackParamX = "x"

	// fallbackVarRv names the local reflect.Value of the input slice.
	fallbackVarRv = "rv"

	// fallbackVarI names the loop induction variable.
	fallbackVarI = "i"

	// fallbackVarBest names the running best element in the min/max helper.
	fallbackVarBest = "best"

	// reflectPkgName is the package selector for the reflect standard library.
	reflectPkgName = "reflect"

	// reflectMethodIndex is the reflect.Value.Index method name.
	reflectMethodIndex = "Index"

	// reflectMethodInterface is the reflect.Value.Interface method name.
	reflectMethodInterface = "Interface"
)

// generateSliceWrapperAST produces a wrapper function AST for a generic function
// dispatched by slice type.
//
// Takes genericFunc (GenericFuncInfo) which provides the generic function info.
// Takes alias (string) which specifies the import alias.
// Takes config (PackageConfig) which provides the type configuration.
// Takes analysis (typeParamAnalysis) which provides the type parameter dispatch analysis.
//
// Returns the generated declaration, its metadata, and any error encountered.
func generateSliceWrapperAST(genericFunc GenericFuncInfo, alias string, config PackageConfig, analysis typeParamAnalysis) (*ast.FuncDecl, *wrapperMeta, error) {
	signature := genericFunc.Signature
	elemTypes, _, _ := config.TypesForFunc(genericFunc.Name)
	if len(elemTypes) == 0 {
		return nil, nil, nil
	}

	functionName := "wrapped" + titleCase(alias) + genericFunc.Name
	parameters := classifyParams(signature, analysis)
	returns := classifyReturns(signature)

	var caseClauses []ast.Stmt
	for _, et := range elemTypes {
		caseClauses = append(caseClauses, buildCaseClauseAST(
			sliceTypeExpr(et), genericFunc, alias, parameters, et, slicePrefix+et, dispatchSlice,
		))
	}

	fallback := detectComparatorFallback(genericFunc.Name, parameters, returns)
	reflectHelper := ""
	if fallback == comparatorFallbackNone && config.GenericFallback != genericFallbackPanic {
		reflectHelper = detectReflectiveFallback(alias, genericFunc.Name)
	}
	declaration := buildWrapperDeclAST(functionName, caseClauses, wrapperBuildSpec{
		alias:             alias,
		origName:          genericFunc.Name,
		parameters:        parameters,
		returns:           returns,
		fallback:          fallback,
		reflectHelperName: reflectHelper,
	})
	return declaration, &wrapperMeta{OriginalName: genericFunc.Name, FuncName: functionName, fallback: fallback, reflectHelperName: reflectHelper}, nil
}

// generateMapWrapperAST produces a wrapper function AST for a generic function dispatched
// by map type.
//
// Takes genericFunc (GenericFuncInfo) which provides the generic function info.
// Takes alias (string) which specifies the import alias.
// Takes config (PackageConfig) which provides the type configuration.
//
// Returns the generated declaration, its metadata, and any error encountered.
func generateMapWrapperAST(genericFunc GenericFuncInfo, alias string, config PackageConfig, _ typeParamAnalysis) (*ast.FuncDecl, *wrapperMeta, error) {
	signature := genericFunc.Signature
	_, keyTypes, valTypes := config.TypesForFunc(genericFunc.Name)
	if len(keyTypes) == 0 || len(valTypes) == 0 {
		return nil, nil, nil
	}

	functionName := "wrapped" + titleCase(alias) + genericFunc.Name
	analysis := analyseTypeParams(signature)
	parameters := classifyParams(signature, analysis)
	returns := classifyReturns(signature)

	var caseClauses []ast.Stmt
	for _, kt := range keyTypes {
		for _, vt := range valTypes {
			mapType := mapPrefix + kt + mapCloser + vt
			caseClauses = append(caseClauses, buildCaseClauseAST(
				mapTypeExpr(kt, vt), genericFunc, alias, parameters, "", mapType, dispatchMap,
			))
		}
	}

	reflectHelper := ""
	if config.GenericFallback != genericFallbackPanic {
		reflectHelper = detectReflectiveFallback(alias, genericFunc.Name)
	}
	declaration := buildWrapperDeclAST(functionName, caseClauses, wrapperBuildSpec{
		alias:             alias,
		origName:          genericFunc.Name,
		parameters:        parameters,
		returns:           returns,
		fallback:          comparatorFallbackNone,
		reflectHelperName: reflectHelper,
	})
	return declaration, &wrapperMeta{OriginalName: genericFunc.Name, FuncName: functionName, reflectHelperName: reflectHelper}, nil
}

// generateScalarWrapperAST produces a wrapper function AST for a generic function
// dispatched by scalar type.
//
// Takes genericFunc (GenericFuncInfo) which provides the generic function info.
// Takes alias (string) which specifies the import alias.
// Takes config (PackageConfig) which provides the type configuration.
//
// Returns the generated declaration, its metadata, and any error encountered.
func generateScalarWrapperAST(genericFunc GenericFuncInfo, alias string, config PackageConfig, _ typeParamAnalysis) (*ast.FuncDecl, *wrapperMeta, error) {
	signature := genericFunc.Signature
	elemTypes, _, _ := config.TypesForFunc(genericFunc.Name)
	if len(elemTypes) == 0 {
		return nil, nil, nil
	}

	functionName := "wrapped" + titleCase(alias) + genericFunc.Name
	analysis := analyseTypeParams(signature)
	parameters := classifyParams(signature, analysis)
	returns := classifyReturns(signature)

	var caseClauses []ast.Stmt
	for _, et := range elemTypes {
		caseClauses = append(caseClauses, buildCaseClauseAST(
			goastutil.CachedIdent(et), genericFunc, alias, parameters, et, et, dispatchScalar,
		))
	}

	reflectHelper := ""
	if config.GenericFallback != genericFallbackPanic {
		reflectHelper = detectReflectiveFallback(alias, genericFunc.Name)
	}
	declaration := buildWrapperDeclAST(functionName, caseClauses, wrapperBuildSpec{
		alias:             alias,
		origName:          genericFunc.Name,
		parameters:        parameters,
		returns:           returns,
		fallback:          comparatorFallbackNone,
		reflectHelperName: reflectHelper,
	})
	return declaration, &wrapperMeta{OriginalName: genericFunc.Name, FuncName: functionName, reflectHelperName: reflectHelper}, nil
}

// buildCaseClauseAST generates a single case clause in the type switch for a concrete
// type.
//
// Takes caseType (ast.Expr) which provides the AST expression for the case type.
// Takes genericFunc (GenericFuncInfo) which provides the generic function info.
// Takes alias (string) which specifies the import alias.
// Takes parameters ([]paramInfo) which provides the classified parameters.
// Takes elementType (string) which specifies the element type name.
// Takes dispatchType (string) which specifies the full dispatch type string.
// Takes dk (dispatchKind) which specifies the dispatch kind.
//
// Returns the generated case clause AST node.
func buildCaseClauseAST(caseType ast.Expr, genericFunc GenericFuncInfo, alias string, parameters []paramInfo, elementType, dispatchType string, dk dispatchKind) *ast.CaseClause {
	signature := genericFunc.Signature
	firstParam := parameters[0]
	firstIsVariadic := firstParam.pkind == paramVariadicElem || firstParam.pkind == paramVariadicSlice

	if firstIsVariadic {
		return buildVariadicCaseClauseAST(caseType, genericFunc, alias, parameters, elementType, dk, signature)
	}

	var statements []ast.Stmt

	lastParameter := parameters[len(parameters)-1]
	isVariadicElem := signature.Variadic() && lastParameter.pkind == paramVariadicElem
	isVariadicSlice := signature.Variadic() && lastParameter.pkind == paramVariadicSlice

	if isVariadicElem {
		statements = append(statements, buildVariadicConversionAST(lastParameter.name, elementType, false)...)
	}
	if isVariadicSlice {
		statements = append(statements, buildVariadicConversionAST(lastParameter.name, elementType, true)...)
	}

	arguments := buildTypedCallArgs(parameters, elementType, dispatchType)
	callExpression := goastutil.CallExpr(goastutil.SelectorExpr(alias, genericFunc.Name), arguments...)

	if isVariadicElem || isVariadicSlice {
		callExpression.Ellipsis = 1
	}

	if signature.Results().Len() == 0 {
		statements = append(statements, goastutil.ExprStmt(callExpression))
	} else {
		statements = append(statements, goastutil.ReturnStmt(callExpression))
	}

	return &ast.CaseClause{
		List: []ast.Expr{caseType},
		Body: statements,
	}
}

// buildTypedCallArgs builds the argument expressions for a typed call inside a case
// clause, converting each parameter according to its kind.
//
// Takes parameters ([]paramInfo) which provides the classified function parameters.
// Takes elementType (string) which specifies the element type name.
// Takes dispatchType (string) which specifies the full dispatch type string.
//
// Returns the list of argument AST expressions for the typed call.
func buildTypedCallArgs(parameters []paramInfo, elementType, dispatchType string) []ast.Expr {
	keyType, valType := extractKeyValTypes(dispatchType)
	effectiveElemType := elementType
	if effectiveElemType == "" && valType != "" {
		effectiveElemType = valType
	}

	var arguments []ast.Expr
	for i, p := range parameters {
		if i == 0 {
			arguments = append(arguments, goastutil.CachedIdent("typedArg"))
			continue
		}
		arguments = append(arguments, buildParamArgExpr(p, effectiveElemType, keyType, valType, dispatchType))
	}
	return arguments
}

// buildParamArgExpr builds a single argument expression for a parameter based on its
// kind.
//
// Takes p (paramInfo) which provides the parameter info.
// Takes elementType (string) which specifies the element type name.
// Takes keyType (string) which specifies the map key type name.
// Takes valType (string) which specifies the map value type name.
// Takes dispatchType (string) which specifies the full dispatch type string.
//
// Returns the AST expression for the argument in the typed call.
func buildParamArgExpr(p paramInfo, elementType, keyType, valType, dispatchType string) ast.Expr {
	switch p.pkind {
	case paramElem:
		return coerceCallAST(elementType, p.name)
	case paramSliceOfElem:
		return goastutil.TypeAssertExpr(
			goastutil.CachedIdent(p.name), sliceTypeExpr(elementType),
		)
	case paramMapType:
		return goastutil.TypeAssertExpr(
			goastutil.CachedIdent(p.name), parseTypeExpr(dispatchType),
		)
	case paramVariadicElem, paramVariadicSlice:
		return goastutil.CachedIdent("typedVariadic")
	case paramFuncOfElem:
		return buildFuncAdapterAST(p.funcInfo, p.name, elementType, keyType)
	case paramFuncOfKeyVal:
		return buildFuncAdapterAST(p.funcInfo, p.name, valType, keyType)
	case paramIterSeq:
		return buildIteratorAdapterAST(p.funcInfo, p.name, elementType, keyType, false)
	case paramIterSeq2:
		return buildIteratorAdapterAST(p.funcInfo, p.name, valType, keyType, true)
	default:
		return goastutil.CachedIdent(p.name)
	}
}

// buildVariadicCaseClauseAST builds a case clause for functions where the dispatch param
// is the variadic param (e.g. cmp.Or, slices.Concat).
//
// Takes caseType (ast.Expr) which provides the AST expression for the case type.
// Takes genericFunc (GenericFuncInfo) which provides the generic function info.
// Takes alias (string) which specifies the import alias.
// Takes parameters ([]paramInfo) which provides the classified parameters.
// Takes elementType (string) which specifies the element type name.
// Takes dk (dispatchKind) which specifies the dispatch kind.
// Takes signature (*types.Signature) which provides the function signature.
//
// Returns the generated case clause AST node.
func buildVariadicCaseClauseAST(caseType ast.Expr, genericFunc GenericFuncInfo, alias string, parameters []paramInfo, elementType string, dk dispatchKind, signature *types.Signature) *ast.CaseClause {
	p := parameters[0]

	var targetSliceType ast.Expr
	var coerceType string
	switch dk {
	case dispatchSlice:
		targetSliceType = &ast.ArrayType{Elt: sliceTypeExpr(elementType)}
		coerceType = slicePrefix + elementType
	default:
		targetSliceType = sliceTypeExpr(elementType)
		coerceType = elementType
	}

	makeCall := goastutil.CallExpr(
		goastutil.CachedIdent("make"),
		targetSliceType,
		goastutil.CallExpr(goastutil.CachedIdent("len"), goastutil.CachedIdent(p.name)),
	)
	defineTyped := goastutil.DefineStmt("typed", makeCall)

	rangeBody := goastutil.AssignStmt(
		goastutil.IndexExpr(goastutil.CachedIdent("typed"), goastutil.CachedIdent(identI)),
		coerceCallAST(coerceType, identV),
	)
	rangeStmt := &ast.RangeStmt{
		Key:   goastutil.CachedIdent(identI),
		Value: goastutil.CachedIdent(identV),
		Tok:   token.DEFINE,
		X:     goastutil.CachedIdent(p.name),
		Body:  goastutil.BlockStmt(rangeBody),
	}

	callExpression := goastutil.CallExpr(
		goastutil.SelectorExpr(alias, genericFunc.Name),
		goastutil.CachedIdent("typed"),
	)
	callExpression.Ellipsis = 1

	var callStmt ast.Stmt
	if signature.Results().Len() == 0 {
		callStmt = goastutil.ExprStmt(callExpression)
	} else {
		callStmt = goastutil.ReturnStmt(callExpression)
	}

	return &ast.CaseClause{
		List: []ast.Expr{caseType},
		Body: []ast.Stmt{defineTyped, rangeStmt, callStmt},
	}
}

// buildVariadicConversionAST generates the typed conversion loop for a variadic parameter
// that is not the dispatch parameter.
//
// Takes parameterName (string) which specifies the name of the variadic parameter.
// Takes elementType (string) which specifies the element type name.
// Takes isSlice (bool) which indicates whether the variadic elements are themselves
// slices.
//
// Returns the AST statements for the typed conversion loop.
func buildVariadicConversionAST(parameterName, elementType string, isSlice bool) []ast.Stmt {
	var targetType ast.Expr
	var coerceType string
	if isSlice {
		targetType = &ast.ArrayType{Elt: sliceTypeExpr(elementType)}
		coerceType = slicePrefix + elementType
	} else {
		targetType = sliceTypeExpr(elementType)
		coerceType = elementType
	}

	makeCall := goastutil.CallExpr(
		goastutil.CachedIdent("make"),
		targetType,
		goastutil.CallExpr(goastutil.CachedIdent("len"), goastutil.CachedIdent(parameterName)),
	)
	defineTyped := goastutil.DefineStmt("typedVariadic", makeCall)

	rangeBody := goastutil.AssignStmt(
		goastutil.IndexExpr(goastutil.CachedIdent("typedVariadic"), goastutil.CachedIdent(identI)),
		coerceCallAST(coerceType, identV),
	)
	rangeStmt := &ast.RangeStmt{
		Key:   goastutil.CachedIdent(identI),
		Value: goastutil.CachedIdent(identV),
		Tok:   token.DEFINE,
		X:     goastutil.CachedIdent(parameterName),
		Body:  goastutil.BlockStmt(rangeBody),
	}

	return []ast.Stmt{defineTyped, rangeStmt}
}

// wrapperBuildSpec bundles the dispatch-target and fallback-strategy fields shared by
// buildWrapperDeclAST and buildDefaultClauseAST, keeping both functions within the
// per-function argument limit.
type wrapperBuildSpec struct {
	// alias is the import alias for the generic function's package.
	alias string

	// origName is the original generic function name.
	origName string

	// reflectHelperName, when non-empty, names a hand-written reflective fallback helper to
	// call from the default clause; it takes precedence over fallback. Empty selects the
	// panic-or-comparator behaviour.
	reflectHelperName string

	// parameters holds the classified wrapper parameters.
	parameters []paramInfo

	// returns holds the classified wrapper return types.
	returns []paramInfo

	// fallback selects the reflect-based comparator default clause, or
	// comparatorFallbackNone for no comparator-shaped fallback.
	fallback comparatorFallbackKind
}

// buildWrapperDeclAST constructs the complete wrapper function declaration with type
// switch and default panic clause.
//
// Takes functionName (string) which specifies the wrapper function name.
// Takes caseClauses ([]ast.Stmt) which provides the type switch case clauses.
// Takes spec (wrapperBuildSpec) which bundles the dispatch target and fallback strategy
// for the wrapper.
//
// Returns the complete wrapper function declaration AST node.
func buildWrapperDeclAST(functionName string, caseClauses []ast.Stmt, spec wrapperBuildSpec) *ast.FuncDecl {
	paramFields := buildParamFieldList(spec.parameters)
	resultFields := buildResultFieldList(spec.returns)

	firstParam := spec.parameters[0]
	isVariadicDispatch := firstParam.pkind == paramVariadicElem || firstParam.pkind == paramVariadicSlice

	var comparatorName string
	if spec.fallback != comparatorFallbackNone && len(spec.parameters) > 1 {
		comparatorName = spec.parameters[1].name
	}

	caseClauses = append(caseClauses, buildDefaultClauseAST(spec, firstParam.name, isVariadicDispatch, comparatorName))

	var bodyStmts []ast.Stmt

	if isVariadicDispatch {
		bodyStmts = append(bodyStmts, buildLengthGuardAST(firstParam.name, spec.returns))

		switchStmt := &ast.TypeSwitchStmt{
			Assign: goastutil.ExprStmt(&ast.TypeAssertExpr{
				X: goastutil.IndexExpr(
					goastutil.CachedIdent(firstParam.name),
					goastutil.IntLit(0),
				),
			}),
			Body: goastutil.BlockStmt(caseClauses...),
		}
		bodyStmts = append(bodyStmts, switchStmt)
	} else {
		switchStmt := &ast.TypeSwitchStmt{
			Assign: goastutil.DefineStmt("typedArg", &ast.TypeAssertExpr{
				X: goastutil.CachedIdent(firstParam.name),
			}),
			Body: goastutil.BlockStmt(caseClauses...),
		}
		bodyStmts = append(bodyStmts, switchStmt)
	}

	return goastutil.FuncDecl(functionName, paramFields, resultFields, goastutil.BlockStmt(bodyStmts...))
}

// buildParamFieldList converts classified parameter info into an AST field list for the
// wrapper declaration.
//
// Takes parameters ([]paramInfo) which provides the classified function parameters.
//
// Returns the AST field list representing the wrapper function's parameters.
func buildParamFieldList(parameters []paramInfo) *ast.FieldList {
	fields := make([]*ast.Field, 0, len(parameters))
	for _, p := range parameters {
		var typ ast.Expr
		if p.pkind == paramVariadicElem || p.pkind == paramVariadicSlice {
			typ = &ast.Ellipsis{Elt: parseTypeExpr(p.goType)}
		} else if p.astType != nil {
			typ = p.astType
		} else {
			typ = parseTypeExpr(p.goType)
		}
		fields = append(fields, goastutil.Field(p.name, typ))
	}
	return goastutil.FieldList(fields...)
}

// buildResultFieldList converts classified return type info into an AST field list for
// the wrapper declaration.
//
// Takes returns ([]paramInfo) which provides the classified return types.
//
// Returns the AST field list representing the wrapper function's return types, or nil if
// there are none.
func buildResultFieldList(returns []paramInfo) *ast.FieldList {
	if len(returns) == 0 {
		return nil
	}
	var fields []*ast.Field
	for _, r := range returns {
		fields = append(fields, goastutil.Field("", parseTypeExpr(r.goType)))
	}
	return goastutil.FieldList(fields...)
}

// buildLengthGuardAST generates the early return for variadic dispatch when the input is
// empty.
//
// Takes parameterName (string) which specifies the variadic parameter name to check.
// Takes returns ([]paramInfo) which provides the classified return types for the
// zero-value return.
//
// Returns the if-statement AST node that guards against empty variadic input.
func buildLengthGuardAST(parameterName string, returns []paramInfo) *ast.IfStmt {
	condition := &ast.BinaryExpr{
		X:  goastutil.CallExpr(goastutil.CachedIdent("len"), goastutil.CachedIdent(parameterName)),
		Op: token.EQL,
		Y:  goastutil.IntLit(0),
	}

	var bodyStmts []ast.Stmt
	if len(returns) > 0 {
		bodyStmts = append(bodyStmts,
			goastutil.VarDecl("zero", parseTypeExpr(returns[0].goType)),
			goastutil.ReturnStmt(goastutil.CachedIdent("zero")),
		)
	} else {
		bodyStmts = append(bodyStmts, &ast.ReturnStmt{})
	}

	return goastutil.IfStmt(nil, condition, goastutil.BlockStmt(bodyStmts...))
}

// isComparatorParam returns true if a parameter is a comparator of the shape func(E, E)
// int, where both parameters are the generic element type and the result is int.
//
// Takes p (paramInfo) which provides the classified parameter info.
//
// Returns true if the parameter is a two-argument int-returning comparator over the
// element type.
func isComparatorParam(p paramInfo) bool {
	if p.pkind != paramFuncOfElem || p.funcInfo == nil {
		return false
	}
	fi := p.funcInfo
	return fi.adapterReturn == "int" &&
		len(fi.adapterParams) == 2 &&
		len(fi.elementPositions) == 2 &&
		len(fi.keyPositions) == 0 &&
		len(fi.valuePositions) == 0
}

var (
	// reflectiveFallbackTable maps "<alias>.<origName>" to a fallback name.
	//
	// Each value names a hand-written reflective fallback helper in the
	// driven_system_symbols package (see wrapped_generic_fallbacks.go). Functions in this
	// table get a default type-switch clause that dispatches reflectively for any element
	// type meeting the generic's constraint, instead of panicking with "unsupported type
	// %T".
	//
	// Adding a new entry requires a helper of the same name in wrapped_generic_fallbacks.go
	// whose signature matches the wrapper's "any"-typed parameter list, plus a regenerated
	// gen_*.go file to pick up the new default clause.
	//
	// Functions with comparator-shaped signatures (SortFunc / MinFunc / MaxFunc /
	// SortStableFunc) route through the comparator fallback machinery instead; do not add
	// them here.
	//
	//nolint:gochecknoglobals // generator-side lookup table.
	reflectiveFallbackTable = map[string]string{
		"slices.Sort":             "reflectSlicesSort",
		"slices.Min":              "reflectSlicesMin",
		"slices.Max":              "reflectSlicesMax",
		"slices.BinarySearch":     "reflectSlicesBinarySearch",
		"slices.IsSorted":         "reflectSlicesIsSorted",
		"slices.Compare":          "reflectSlicesCompare",
		"slices.Contains":         "reflectSlicesContains",
		"slices.Equal":            "reflectSlicesEqual",
		"slices.Index":            "reflectSlicesIndex",
		"slices.Clip":             "reflectSlicesClip",
		"slices.Clone":            "reflectSlicesClone",
		"slices.Reverse":          "reflectSlicesReverse",
		"slices.Concat":           "reflectSlicesConcat",
		"slices.All":              "reflectSlicesAll",
		"slices.Backward":         "reflectSlicesBackward",
		"slices.Values":           "reflectSlicesValues",
		"slices.Chunk":            "reflectSlicesChunk",
		"slices.AppendSeq":        "reflectSlicesAppendSeq",
		"slices.Compact":          "reflectSlicesCompact",
		"slices.CompactFunc":      "reflectSlicesCompactFunc",
		"slices.Delete":           "reflectSlicesDelete",
		"slices.DeleteFunc":       "reflectSlicesDeleteFunc",
		"slices.Grow":             "reflectSlicesGrow",
		"slices.Insert":           "reflectSlicesInsert",
		"slices.Repeat":           "reflectSlicesRepeat",
		"slices.Replace":          "reflectSlicesReplace",
		"slices.ContainsFunc":     "reflectSlicesContainsFunc",
		"slices.IndexFunc":        "reflectSlicesIndexFunc",
		"slices.IsSortedFunc":     "reflectSlicesIsSortedFunc",
		"slices.BinarySearchFunc": "reflectSlicesBinarySearchFunc",
		"slices.CompareFunc":      "reflectSlicesCompareFunc",
		"slices.EqualFunc":        "reflectSlicesEqualFunc",
		"maps.Clone":              "reflectMapsClone",
		"maps.Copy":               "reflectMapsCopy",
		"maps.Equal":              "reflectMapsEqual",
		"maps.All":                "reflectMapsAll",
		"maps.Keys":               "reflectMapsKeys",
		"maps.Values":             "reflectMapsValues",
		"maps.Insert":             "reflectMapsInsert",
		"maps.DeleteFunc":         "reflectMapsDeleteFunc",
		"maps.EqualFunc":          "reflectMapsEqualFunc",
		"cmp.Compare":             "reflectCmpCompare",
		"cmp.Less":                "reflectCmpLess",
		"cmp.Or":                  "reflectCmpOr",
	}
)

// detectReflectiveFallback returns the name of a hand-written reflective fallback helper
// for the given function, or "" if the function is not in the table.
//
// Takes alias (string) which is the package alias (e.g. "slices").
// Takes origName (string) which is the original function name (e.g. "Sort").
//
// Returns the helper function name or empty string when no reflective fallback applies.
func detectReflectiveFallback(alias, origName string) string {
	return reflectiveFallbackTable[alias+"."+origName]
}

// detectComparatorFallback classifies a generic slice function's shape.
//
// Determines whether the default clause should emit a reflect-based fallback instead of a
// panic. Matches the shape of slices.MinFunc, slices.MaxFunc, slices.SortFunc and
// slices.SortStableFunc: a two-parameter function whose first parameter is a slice of the
// element type and whose second parameter is a func(E, E) int comparator.
//
// Takes origName (string) which specifies the original function name, used only to
// distinguish minimum from maximum selection.
// Takes parameters ([]paramInfo) which provides the classified wrapper parameters.
// Takes returns ([]paramInfo) which provides the classified wrapper return values.
//
// Returns the comparator fallback kind for the function.
func detectComparatorFallback(origName string, parameters []paramInfo, returns []paramInfo) comparatorFallbackKind {
	if len(parameters) != 2 {
		return comparatorFallbackNone
	}
	if parameters[0].pkind != paramSliceOfElem {
		return comparatorFallbackNone
	}
	if !isComparatorParam(parameters[1]) {
		return comparatorFallbackNone
	}

	switch {
	case len(returns) == 0:
		return comparatorFallbackSort
	case len(returns) == 1 && returns[0].pkind == paramElem:
		if strings.Contains(origName, "Min") {
			return comparatorFallbackMin
		}
		return comparatorFallbackMax
	default:
		return comparatorFallbackNone
	}
}

// buildDefaultClauseAST generates the default clause for a generic dispatch type switch.
// Precedence: reflective fallback helper (when spec.reflectHelperName is set) >
// comparator fallback (when spec.fallback is non-None) > panic.
//
// Takes spec (wrapperBuildSpec) which bundles the dispatch target and fallback strategy.
// Takes paramName (string) which specifies the dispatch parameter name.
// Takes isVariadicDispatch (bool) which indicates whether the dispatch is on a variadic
// parameter.
// Takes comparatorName (string) which specifies the comparator parameter name passed to
// the comparator fallback helper.
//
// Returns the default case clause AST node.
func buildDefaultClauseAST(spec wrapperBuildSpec, paramName string, isVariadicDispatch bool, comparatorName string) *ast.CaseClause {
	if spec.reflectHelperName != "" {
		return buildReflectiveFallbackClauseAST(spec.reflectHelperName, spec.parameters, spec.returns)
	}
	if spec.fallback != comparatorFallbackNone {
		return buildComparatorFallbackClauseAST(spec.fallback, paramName, comparatorName)
	}

	var fmtArg ast.Expr
	if isVariadicDispatch {
		fmtArg = goastutil.IndexExpr(goastutil.CachedIdent(paramName), goastutil.IntLit(0))
	} else {
		fmtArg = goastutil.CachedIdent(paramName)
	}

	panicCall := goastutil.CallExpr(
		goastutil.CachedIdent(panicKeyword),
		goastutil.CallExpr(
			goastutil.SelectorExpr("fmt", "Sprintf"),
			goastutil.StrLit(spec.alias+"."+spec.origName+": unsupported type %T"),
			fmtArg,
		),
	)

	return &ast.CaseClause{
		Body: []ast.Stmt{goastutil.ExprStmt(panicCall)},
	}
}

// buildReflectiveFallbackClauseAST builds a default-clause body that delegates to a
// hand-written reflective fallback helper in the driven_system_symbols package. The
// clause forwards every wrapper parameter as-is; the helper is responsible for argument
// coercion via reflect.
//
// For void-returning wrappers it emits the call as a statement. Otherwise it emits a
// return statement passing the helper's results straight through.
//
// Takes helperName (string) which is the function to call (e.g. "reflectSlicesSort").
// Takes parameters ([]paramInfo) which provides argument names.
// Takes returns ([]paramInfo) which provides return arity, used to decide between
// bare-call and return.
//
// Returns the default case clause AST node.
func buildReflectiveFallbackClauseAST(helperName string, parameters []paramInfo, returns []paramInfo) *ast.CaseClause {
	args := make([]ast.Expr, 0, len(parameters))
	for _, p := range parameters {
		args = append(args, goastutil.CachedIdent(p.name))
	}
	call := &ast.CallExpr{
		Fun:  goastutil.CachedIdent(helperName),
		Args: args,
	}
	if len(parameters) > 0 {
		lastKind := parameters[len(parameters)-1].pkind
		if lastKind == paramVariadicElem || lastKind == paramVariadicSlice {
			call.Ellipsis = token.Pos(1)
		}
	}
	var body []ast.Stmt
	if len(returns) == 0 {
		body = []ast.Stmt{goastutil.ExprStmt(call), &ast.ReturnStmt{}}
	} else {
		body = []ast.Stmt{goastutil.ReturnStmt(call)}
	}
	return &ast.CaseClause{Body: body}
}

// buildComparatorFallbackClauseAST builds the default clause body that delegates to a
// reflect-based fallback helper.
//
// Takes fallback (comparatorFallbackKind) which selects the helper.
// Takes paramName (string) which specifies the dispatch (slice) parameter name.
// Takes comparatorName (string) which specifies the comparator parameter name.
//
// Returns the default case clause AST node containing the fallback call.
func buildComparatorFallbackClauseAST(fallback comparatorFallbackKind, paramName, comparatorName string) *ast.CaseClause {
	switch fallback {
	case comparatorFallbackSort:
		call := goastutil.CallExpr(
			goastutil.CachedIdent("reflectSortFuncFallback"),
			goastutil.CachedIdent(paramName),
			goastutil.CachedIdent(comparatorName),
		)
		return &ast.CaseClause{Body: []ast.Stmt{goastutil.ExprStmt(call)}}
	default:
		takeMin := "false"
		if fallback == comparatorFallbackMin {
			takeMin = "true"
		}
		call := goastutil.CallExpr(
			goastutil.CachedIdent("reflectMinMaxFuncFallback"),
			goastutil.CachedIdent(paramName),
			goastutil.CachedIdent(comparatorName),
			goastutil.CachedIdent(takeMin),
		)
		return &ast.CaseClause{Body: []ast.Stmt{goastutil.ReturnStmt(call)}}
	}
}

// fallbackHelpersNeeded inspects the generated wrapper metadata and reports which
// reflect-based fallback helpers must be emitted.
//
// Takes metas ([]wrapperMeta) which provides the metadata for every generated wrapper in
// the file.
//
// Returns needMinMax true if reflectMinMaxFuncFallback is used and needSort true if
// reflectSortFuncFallback is used.
func fallbackHelpersNeeded(metas []wrapperMeta) (needMinMax, needSort bool) {
	for _, meta := range metas {
		switch meta.fallback {
		case comparatorFallbackMin, comparatorFallbackMax:
			needMinMax = true
		case comparatorFallbackSort:
			needSort = true
		case comparatorFallbackNone:
		}
	}
	return needMinMax, needSort
}

// buildFallbackHelperDecls parses and returns the reflect-based fallback helper function
// declarations required by the generated wrappers in a file.
//
// Takes metas ([]wrapperMeta) which provides the metadata for every generated wrapper in
// the file.
//
// Returns the helper function declarations, true if the sort import is required, and any
// parse error encountered.
func buildFallbackHelperDecls(metas []wrapperMeta) ([]*ast.FuncDecl, bool, error) {
	needMinMax, needSort := fallbackHelpersNeeded(metas)
	var decls []*ast.FuncDecl
	if needMinMax {
		decls = append(decls, buildReflectMinMaxFuncFallbackDecl())
	}
	if needSort {
		decls = append(decls, buildReflectSortFuncFallbackDecl())
	}
	return decls, needSort, nil
}

// buildReflectMinMaxFuncFallbackDecl builds the reflect-based fallback helper that
// selects the minimum or maximum slice element using a comparator. Used by wrappers for
// slices.MinFunc and slices.MaxFunc when the concrete slice type is not in the type
// switch.
//
// Returns the assembled function declaration.
func buildReflectMinMaxFuncFallbackDecl() *ast.FuncDecl {
	parameters := goastutil.FieldList(
		goastutil.Field(fallbackParamX, anyTypeIdent()),
		goastutil.Field("cmpArg", anyComparatorType()),
		goastutil.Field("takeMin", goastutil.CachedIdent("bool")),
	)
	results := goastutil.FieldList(goastutil.Field("", anyTypeIdent()))
	body := goastutil.BlockStmt(
		reflectValueOfXDefine(),
		buildMinMaxGuard(),
		bestDefineFromIndex(0),
		buildMinMaxLoop(),
		goastutil.ReturnStmt(goastutil.CachedIdent(fallbackVarBest)),
	)
	return goastutil.FuncDecl("reflectMinMaxFuncFallback", parameters, results, body)
}

// buildReflectSortFuncFallbackDecl builds the reflect-based fallback helper that sorts a
// slice in place using a comparator. Used by wrappers for slices.SortFunc and
// slices.SortStableFunc when the concrete slice type is not in the type switch.
//
// Returns the assembled function declaration.
func buildReflectSortFuncFallbackDecl() *ast.FuncDecl {
	parameters := goastutil.FieldList(
		goastutil.Field(fallbackParamX, anyTypeIdent()),
		goastutil.Field("cmpArg", anyComparatorType()),
	)
	sortCall := goastutil.ExprStmt(goastutil.CallExpr(
		goastutil.SelectorExpr("sort", "SliceStable"),
		goastutil.CallExpr(goastutil.SelectorExpr(fallbackVarRv, reflectMethodInterface)),
		buildSortLessLit(),
	))
	body := goastutil.BlockStmt(
		reflectValueOfXDefine(),
		buildSortGuard(),
		sortCall,
	)
	return goastutil.FuncDecl("reflectSortFuncFallback", parameters, nil, body)
}

// anyComparatorType returns the AST type for func(p0, p1 any) int, the comparator
// parameter signature shared by the min/max and sort reflect fallbacks.
//
// Returns the assembled function type.
func anyComparatorType() *ast.FuncType {
	return goastutil.FuncType(
		&ast.FieldList{List: []*ast.Field{groupedAnyField("p0", "p1")}},
		goastutil.FieldList(goastutil.Field("", goastutil.CachedIdent("int"))),
	)
}

// reflectValueOfXDefine returns the AST statement `rv := reflect.ValueOf(x)`.
//
// Returns the assembled define statement.
func reflectValueOfXDefine() ast.Stmt {
	return goastutil.DefineStmt(
		fallbackVarRv,
		goastutil.CallExpr(
			goastutil.SelectorExpr(reflectPkgName, "ValueOf"),
			goastutil.CachedIdent(fallbackParamX),
		),
	)
}

// reflectIndexInterface returns the AST expression `rv.Index(idx).Interface()`.
//
// Takes idx (ast.Expr) which is the index expression passed to reflect.Value.Index.
//
// Returns the assembled call expression.
func reflectIndexInterface(idx ast.Expr) ast.Expr {
	return goastutil.CallExpr(goastutil.SelectorExprFrom(
		goastutil.CallExpr(goastutil.SelectorExpr(fallbackVarRv, reflectMethodIndex), idx),
		reflectMethodInterface,
	))
}

// reflectKindIsNotSlice returns the AST expression `rv.Kind() != reflect.Slice`.
//
// Returns the assembled binary comparison expression.
func reflectKindIsNotSlice() ast.Expr {
	return &ast.BinaryExpr{
		X:  goastutil.CallExpr(goastutil.SelectorExpr(fallbackVarRv, "Kind")),
		Op: token.NEQ,
		Y:  goastutil.SelectorExpr(reflectPkgName, "Slice"),
	}
}

// panicWithFmtSprintf returns a panic(fmt.Sprintf(format, x)) statement using the
// fallback's input parameter as the formatted argument.
//
// Takes format (string) which is the fmt verb string passed to Sprintf.
//
// Returns the assembled expression statement.
func panicWithFmtSprintf(format string) ast.Stmt {
	return goastutil.ExprStmt(goastutil.CallExpr(
		goastutil.CachedIdent(panicKeyword),
		goastutil.CallExpr(
			goastutil.SelectorExpr("fmt", "Sprintf"),
			goastutil.StrLit(format),
			goastutil.CachedIdent(fallbackParamX),
		),
	))
}

// buildMinMaxGuard returns the precondition check for the min/max fallback.
//
// Returns the assembled if statement that panics on a non-slice input or an empty slice.
func buildMinMaxGuard() ast.Stmt {
	emptySlice := &ast.BinaryExpr{
		X:  goastutil.CallExpr(goastutil.SelectorExpr(fallbackVarRv, "Len")),
		Op: token.EQL,
		Y:  goastutil.IntLit(0),
	}
	return &ast.IfStmt{
		Cond: &ast.BinaryExpr{X: reflectKindIsNotSlice(), Op: token.LOR, Y: emptySlice},
		Body: goastutil.BlockStmt(panicWithFmtSprintf("slices.Min/MaxFunc: unsupported type %T or empty slice")),
	}
}

// buildSortGuard returns the precondition check for the sort fallback.
//
// Returns the assembled if statement that panics on a non-slice input.
func buildSortGuard() ast.Stmt {
	return &ast.IfStmt{
		Cond: reflectKindIsNotSlice(),
		Body: goastutil.BlockStmt(panicWithFmtSprintf("slices.SortFunc: unsupported type %T")),
	}
}

// bestDefineFromIndex returns the AST statement `best := rv.Index(idx).Interface()` used
// to seed the min/max running candidate from a fixed slice index.
//
// Takes idx (int) which is the literal slice index used to seed best.
//
// Returns the assembled define statement.
func bestDefineFromIndex(idx int) ast.Stmt {
	return goastutil.DefineStmt(fallbackVarBest, reflectIndexInterface(goastutil.IntLit(idx)))
}

// buildMinMaxLoop returns the for-loop that scans the slice from index 1 and updates best
// when the comparator says the candidate is better than the current best.
//
// Returns the assembled for statement.
func buildMinMaxLoop() ast.Stmt {
	candidateDefine := goastutil.DefineStmt("candidate", reflectIndexInterface(goastutil.CachedIdent(fallbackVarI)))
	cmpDefine := goastutil.DefineStmt(
		"cmp",
		goastutil.CallExpr(
			goastutil.CachedIdent("cmpArg"),
			goastutil.CachedIdent("candidate"),
			goastutil.CachedIdent(fallbackVarBest),
		),
	)
	takeMinBranch := &ast.ParenExpr{X: &ast.BinaryExpr{
		X:  goastutil.CachedIdent("takeMin"),
		Op: token.LAND,
		Y:  &ast.BinaryExpr{X: goastutil.CachedIdent("cmp"), Op: token.LSS, Y: goastutil.IntLit(0)},
	}}
	takeMaxBranch := &ast.ParenExpr{X: &ast.BinaryExpr{
		X:  &ast.UnaryExpr{Op: token.NOT, X: goastutil.CachedIdent("takeMin")},
		Op: token.LAND,
		Y:  &ast.BinaryExpr{X: goastutil.CachedIdent("cmp"), Op: token.GTR, Y: goastutil.IntLit(0)},
	}}
	updateIf := &ast.IfStmt{
		Cond: &ast.BinaryExpr{X: takeMinBranch, Op: token.LOR, Y: takeMaxBranch},
		Body: goastutil.BlockStmt(
			goastutil.AssignStmt(goastutil.CachedIdent(fallbackVarBest), goastutil.CachedIdent("candidate")),
		),
	}
	return &ast.ForStmt{
		Init: goastutil.DefineStmt(fallbackVarI, goastutil.IntLit(1)),
		Cond: &ast.BinaryExpr{
			X:  goastutil.CachedIdent(fallbackVarI),
			Op: token.LSS,
			Y:  goastutil.CallExpr(goastutil.SelectorExpr(fallbackVarRv, "Len")),
		},
		Post: &ast.IncDecStmt{X: goastutil.CachedIdent(fallbackVarI), Tok: token.INC},
		Body: goastutil.BlockStmt(candidateDefine, cmpDefine, updateIf),
	}
}

// buildSortLessLit returns the closure literal passed to sort.SliceStable that returns
// true when cmpArg(a, b) < 0.
//
// Returns the assembled function literal.
func buildSortLessLit() *ast.FuncLit {
	return goastutil.FuncLit(
		goastutil.FuncType(
			&ast.FieldList{List: []*ast.Field{{
				Names: []*ast.Ident{goastutil.CachedIdent(fallbackVarI), goastutil.CachedIdent("j")},
				Type:  goastutil.CachedIdent("int"),
			}}},
			goastutil.FieldList(goastutil.Field("", goastutil.CachedIdent("bool"))),
		),
		goastutil.BlockStmt(
			goastutil.DefineStmt("a", reflectIndexInterface(goastutil.CachedIdent(fallbackVarI))),
			goastutil.DefineStmt("b", reflectIndexInterface(goastutil.CachedIdent("j"))),
			goastutil.ReturnStmt(&ast.BinaryExpr{
				X: goastutil.CallExpr(
					goastutil.CachedIdent("cmpArg"),
					goastutil.CachedIdent("a"),
					goastutil.CachedIdent("b"),
				),
				Op: token.LSS,
				Y:  goastutil.IntLit(0),
			}),
		),
	)
}

// anyTypeIdent returns the AST identifier for the predeclared `any` type used as the
// boxed parameter and return type in the reflect fallbacks.
//
// Returns the cached ident for `any`.
func anyTypeIdent() *ast.Ident {
	return goastutil.CachedIdent("any")
}

// groupedAnyField returns a single ast.Field grouping multiple names under the shared
// `any` type, producing the canonical `name1, name2 any` parameter syntax instead of the
// verbose per-name form.
//
// Takes names (...string) which are the grouped parameter names.
//
// Returns the assembled field with the shared type.
func groupedAnyField(names ...string) *ast.Field {
	idents := make([]*ast.Ident, len(names))
	for i, name := range names {
		idents[i] = goastutil.CachedIdent(name)
	}
	return &ast.Field{Names: idents, Type: anyTypeIdent()}
}

// coerceCallAST generates a coerce[T](v) expression.
//
// Takes targetType (string) which specifies the target type name for the coercion.
// Takes varName (string) which specifies the variable name to coerce.
//
// Returns the AST expression for the coerce call.
func coerceCallAST(targetType, varName string) ast.Expr {
	return &ast.CallExpr{
		Fun: &ast.IndexExpr{
			X:     goastutil.CachedIdent("coerce"),
			Index: parseTypeExpr(targetType),
		},
		Args: []ast.Expr{goastutil.CachedIdent(varName)},
	}
}

// extractKeyValTypes splits a dispatch type like "map[string]int" into key and value
// parts. For non-map types, or for a malformed map prefix without a closing bracket,
// returns the dispatch type as valType with an empty keyType.
//
// Takes dispatchType (string) which provides the type string to split.
//
// Returns the key type and value type extracted from the dispatch type string.
func extractKeyValTypes(dispatchType string) (keyType, valType string) {
	if strings.HasPrefix(dispatchType, mapPrefix) {
		if key, val, ok := splitMapKeyVal(dispatchType); ok {
			return key, val
		}
	}
	return "", dispatchType
}

// buildFuncAdapterAST generates a closure literal that adapts an any-typed func param to
// a concrete-typed func.
//
// Takes fi (*funcParamInfo) which provides the func parameter info.
// Takes parameterName (string) which specifies the original parameter name.
// Takes elementType (string) which specifies the element type name.
// Takes keyType (string) which specifies the map key type name.
//
// Returns the AST expression for the adapter closure literal.
func buildFuncAdapterAST(fi *funcParamInfo, parameterName, elementType, keyType string) ast.Expr {
	adapterFields := make([]*ast.Field, 0, len(fi.adapterParams))
	for i, pName := range fi.adapterParams {
		var concreteType string
		if slices.Contains(fi.keyPositions, i) {
			concreteType = keyType
		} else if slices.Contains(fi.valuePositions, i) || slices.Contains(fi.elementPositions, i) {
			concreteType = elementType
		} else {
			concreteType = identAny
		}
		adapterFields = append(adapterFields, goastutil.Field(pName, parseTypeExpr(concreteType)))
	}

	var resultFields *ast.FieldList
	if fi.adapterReturn != "" {
		resultFields = goastutil.FieldList(goastutil.Field("", parseTypeExpr(fi.adapterReturn)))
	}

	adapterType := goastutil.FuncType(goastutil.FieldList(adapterFields...), resultFields)

	callArgs := make([]ast.Expr, 0, len(fi.adapterParams))
	for _, pName := range fi.adapterParams {
		callArgs = append(callArgs, goastutil.CachedIdent(pName))
	}
	callExpression := goastutil.CallExpr(goastutil.CachedIdent(parameterName), callArgs...)

	var body *ast.BlockStmt
	if fi.adapterReturn != "" {
		body = goastutil.BlockStmt(goastutil.ReturnStmt(callExpression))
	} else {
		body = goastutil.BlockStmt(goastutil.ExprStmt(callExpression))
	}

	return goastutil.FuncLit(adapterType, body)
}

// buildIteratorAdapterAST generates a closure that adapts an any-typed iterator to a
// concrete-typed one.
//
// Takes parameterName (string) which specifies the iterator parameter name.
// Takes elementType (string) which specifies the element type name.
// Takes keyType (string) which specifies the map key type name.
// Takes isSeq2 (bool) which indicates whether this is a two-element iterator.
//
// Returns the AST expression for the iterator adapter closure literal.
func buildIteratorAdapterAST(_ *funcParamInfo, parameterName, elementType, keyType string, isSeq2 bool) ast.Expr {
	var yieldParams []*ast.Field
	var innerCallArgs []ast.Expr

	if isSeq2 {
		yieldParams = append(yieldParams,
			goastutil.Field(identK, parseTypeExpr(keyType)),
			goastutil.Field(identV, parseTypeExpr(elementType)),
		)
		innerCallArgs = append(innerCallArgs,
			coerceCallAST(keyType, identK),
			coerceCallAST(elementType, identV),
		)
	} else {
		yieldParams = append(yieldParams,
			goastutil.Field(identV, parseTypeExpr(elementType)),
		)
		innerCallArgs = append(innerCallArgs,
			coerceCallAST(elementType, identV),
		)
	}

	yieldType := goastutil.FuncType(
		goastutil.FieldList(yieldParams...),
		goastutil.FieldList(goastutil.Field("", goastutil.CachedIdent(identBool))),
	)
	outerType := goastutil.FuncType(
		goastutil.FieldList(goastutil.Field(identYield, yieldType)),
		nil,
	)

	var innerParams []*ast.Field
	if isSeq2 {
		innerParams = append(innerParams,
			goastutil.Field(identK, goastutil.CachedIdent(identAny)),
			goastutil.Field(identV, goastutil.CachedIdent(identAny)),
		)
	} else {
		innerParams = append(innerParams,
			goastutil.Field(identV, goastutil.CachedIdent(identAny)),
		)
	}

	innerType := goastutil.FuncType(
		goastutil.FieldList(innerParams...),
		goastutil.FieldList(goastutil.Field("", goastutil.CachedIdent(identBool))),
	)

	yieldCall := goastutil.CallExpr(goastutil.CachedIdent(identYield), innerCallArgs...)
	innerBody := goastutil.BlockStmt(goastutil.ReturnStmt(yieldCall))
	innerFunc := goastutil.FuncLit(innerType, innerBody)

	sequenceCall := goastutil.CallExpr(goastutil.CachedIdent(parameterName), innerFunc)
	outerBody := goastutil.BlockStmt(goastutil.ExprStmt(sequenceCall))

	return goastutil.FuncLit(outerType, outerBody)
}
