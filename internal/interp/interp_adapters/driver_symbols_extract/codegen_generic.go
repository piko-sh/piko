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
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"slices"
	"strings"

	"piko.sh/piko/internal/goastutil"
)

const (
	// slicePrefix is the string prefix for slice type representations.
	slicePrefix = "[]"

	// mapPrefix is the string prefix for map type representations.
	mapPrefix = "map["

	// mapCloser is the closing bracket for map type representations.
	mapCloser = "]"

	// iterPackagePath is the import path for the iter standard library package.
	iterPackagePath = "iter"

	// mapPrefixLen is the precomputed length of the mapPrefix constant.
	mapPrefixLen = len(mapPrefix)

	// identI is the AST identifier name for index variables.
	identI = "i"

	// identK is the AST identifier name for key variables.
	identK = "k"

	// identV is the AST identifier name for value variables.
	identV = "v"

	// identYield is the AST identifier name for yield callback variables.
	identYield = "yield"

	// identAny is the AST identifier name for the any type.
	identAny = "any"

	// identBool is the AST identifier name for the bool type.
	identBool = "bool"

	// qualifiedNameParts is the number of parts in a qualified name
	// split on "." (e.g. "pkg.Name" -> ["pkg", "Name"]).
	qualifiedNameParts = 2
)

// dispatchKind describes how a generic function's first type
// parameter should be dispatched.
type dispatchKind int

const (
	// dispatchScalar indicates the type parameter is a simple scalar type.
	dispatchScalar dispatchKind = iota

	// dispatchSlice indicates the type parameter is constrained to slice types.
	dispatchSlice

	// dispatchMap indicates the type parameter is constrained to map types.
	dispatchMap
)

// typeParamAnalysis holds the result of analysing a generic function's
// type parameters.
type typeParamAnalysis struct {
	// elemParam is the element type parameter (E in ~[]E, T in
	// scalar, V in ~map[K]V).
	elemParam *types.TypeParam

	// keyParam is the key type parameter for map dispatch.
	keyParam *types.TypeParam

	// kind is the dispatch strategy determined from the constraint analysis.
	kind dispatchKind
}

// paramKindInWrapper classifies how a parameter relates to the
// generic type parameters.
type paramKindInWrapper int

const (
	// paramConcrete indicates a non-generic parameter kept as-is.
	paramConcrete paramKindInWrapper = iota

	// paramElem indicates an element type parameter such as E or T.
	paramElem

	// paramSliceOfElem indicates a slice type parameter like S
	// constrained to ~[]E.
	paramSliceOfElem

	// paramMapType indicates a map type parameter like M constrained to ~map[K]V.
	paramMapType

	// paramVariadicElem indicates a variadic parameter with element
	// type like ...E.
	paramVariadicElem

	// paramVariadicSlice indicates a variadic parameter with slice
	// type like ...S where S is ~[]E.
	paramVariadicSlice

	// paramFuncOfElem indicates a function parameter using element
	// type parameters.
	paramFuncOfElem

	// paramFuncOfKeyVal indicates a function parameter using map key
	// and value type parameters.
	paramFuncOfKeyVal

	// paramIterSeq indicates an iter.Seq[E] parameter represented as
	// func(func(E) bool).
	paramIterSeq

	// paramIterSeq2 indicates an iter.Seq2[K,V] parameter
	// represented as func(func(K,V) bool).
	paramIterSeq2
)

// funcParamInfo describes a func-typed parameter's relationship to
// the outer function's type parameters.
type funcParamInfo struct {
	// wrapperFuncType is the AST expression for the wrapper
	// function's type signature.
	wrapperFuncType ast.Expr

	// adapterParams is the list of parameter names for the adapter closure.
	adapterParams []string

	// adapterReturn is the return type name for the adapter closure.
	adapterReturn string

	// elementPositionitions is the indices of parameters that correspond to
	// the element type parameter.
	elementPositionitions []int

	// keyPositionitions is the indices of parameters that correspond to
	// the map key type parameter.
	keyPositionitions []int

	// valuePositionitions is the indices of parameters that correspond to
	// the map value type parameter.
	valuePositionitions []int
}

// paramInfo describes a function parameter in a generated wrapper.
type paramInfo struct {
	// astType is the AST expression for the parameter's type in the wrapper.
	astType ast.Expr

	// funcInfo holds the adapter details when the parameter is a function type.
	funcInfo *funcParamInfo

	// name is the sanitised parameter name used in the generated code.
	name string

	// goType is the Go type name string for the parameter in the wrapper.
	goType string

	// origIndex is the original index of the parameter in the source
	// function signature.
	origIndex int

	// pkind is the classification of how this parameter relates to
	// generic type parameters.
	pkind paramKindInWrapper
}

// analyseTypeParams determines the dispatch pattern from a generic
// function's type parameters.
//
// Takes signature (*types.Signature) which provides the function signature
// to analyse.
//
// Returns a typeParamAnalysis describing the dispatch strategy for
// the function.
func analyseTypeParams(signature *types.Signature) typeParamAnalysis {
	tparams := signature.TypeParams()
	if tparams == nil || tparams.Len() == 0 {
		return typeParamAnalysis{kind: dispatchScalar}
	}

	first := tparams.At(0)
	constraint := first.Constraint()
	iface, ok := constraint.Underlying().(*types.Interface)
	if !ok {
		return typeParamAnalysis{kind: dispatchScalar, elemParam: first}
	}

	if result, found := scanConstraintForDispatch(iface); found {
		return result
	}

	return typeParamAnalysis{kind: dispatchScalar, elemParam: first}
}

// scanConstraintForDispatch scans an interface constraint's embedded
// unions for slice (~[]E) or map (~map[K]V) structural types.
//
// Takes iface (*types.Interface) which provides the interface
// constraint to scan.
//
// Returns the analysis result and true if a structural type was
// found, or a zero value and false otherwise.
func scanConstraintForDispatch(iface *types.Interface) (typeParamAnalysis, bool) {
	for embedded := range iface.EmbeddedTypes() {
		union, ok := embedded.(*types.Union)
		if !ok {
			continue
		}
		for term := range union.Terms() {
			if result, found := matchStructuralType(term.Type()); found {
				return result, true
			}
		}
	}
	return typeParamAnalysis{}, false
}

// matchStructuralType checks whether a union term type is a slice or
// map with type-parameter elements, returning the dispatch analysis.
//
// Takes termType (types.Type) which provides the union term type to
// inspect.
//
// Returns the analysis result and true if a match was found, or a
// zero value and false otherwise.
func matchStructuralType(termType types.Type) (typeParamAnalysis, bool) {
	if sl, ok := termType.(*types.Slice); ok {
		if elemTP, ok := sl.Elem().(*types.TypeParam); ok {
			return typeParamAnalysis{kind: dispatchSlice, elemParam: elemTP}, true
		}
	}
	if m, ok := termType.(*types.Map); ok {
		keyTP, keyOk := m.Key().(*types.TypeParam)
		valTP, valOk := m.Elem().(*types.TypeParam)
		if keyOk && valOk {
			return typeParamAnalysis{kind: dispatchMap, keyParam: keyTP, elemParam: valTP}, true
		}
	}
	return typeParamAnalysis{}, false
}

// generateWrappers produces AST function declarations for all generic
// functions in a package.
//
// Takes extractedPackage (ExtractedPackage) which provides the extracted package
// containing generic functions.
// Takes alias (string) which specifies the import alias for the
// package.
// Takes config (PackageConfig) which provides the type configuration
// for wrapper generation.
//
// Returns the generated function declarations, their metadata, and
// any error encountered.
func generateWrappers(extractedPackage ExtractedPackage, alias string, config PackageConfig) ([]*ast.FuncDecl, []wrapperMeta, error) {
	var decls []*ast.FuncDecl
	var metas []wrapperMeta
	for _, gf := range extractedPackage.GenericFuncs {
		declaration, meta, err := generateWrapper(gf, alias, config)
		if err != nil {
			return nil, nil, fmt.Errorf("%s: %w", gf.Name, err)
		}
		if declaration != nil {
			decls = append(decls, declaration)
			metas = append(metas, *meta)
		}
	}
	return decls, metas, nil
}

// generateWrapper produces a single AST wrapper function declaration
// for a generic function.
//
// Takes gf (GenericFuncInfo) which provides the generic function
// info to wrap.
// Takes alias (string) which specifies the import alias for the
// package.
// Takes config (PackageConfig) which provides the type configuration
// for wrapper generation.
//
// Returns the generated declaration, its metadata, and any error
// encountered.
func generateWrapper(gf GenericFuncInfo, alias string, config PackageConfig) (*ast.FuncDecl, *wrapperMeta, error) {
	signature := gf.Signature
	analysis := analyseTypeParams(signature)

	if hasUndispatchableIterParam(signature, analysis) {
		return nil, nil, nil
	}

	switch analysis.kind {
	case dispatchSlice:
		return generateSliceWrapperAST(gf, alias, config, analysis)
	case dispatchMap:
		return generateMapWrapperAST(gf, alias, config, analysis)
	case dispatchScalar:
		return generateScalarWrapperAST(gf, alias, config, analysis)
	default:
		return nil, nil, nil
	}
}

// hasUndispatchableIterParam returns true for a signature that has an iterator
// param but no collection param (slice/map) to dispatch on, making it
// unsuitable for runtime type-switch dispatch.
//
// Takes signature (*types.Signature) which provides the function signature
// to inspect.
// Takes analysis (typeParamAnalysis) which provides the type
// parameter analysis for the function.
//
// Returns true if the function has iterator parameters but no
// dispatchable collection parameters.
func hasUndispatchableIterParam(signature *types.Signature, analysis typeParamAnalysis) bool {
	hasIter := false
	hasCollection := false
	for p := range signature.Params().Variables() {
		pt := p.Type()
		if isIteratorType(pt) {
			hasIter = true
			continue
		}
		if tp, ok := pt.(*types.TypeParam); ok {
			if isSliceConstraint(tp) || isMapConstraint(tp) || analysis.kind == dispatchScalar {
				hasCollection = true
			}
		}
		if _, ok := pt.(*types.Slice); ok {
			hasCollection = true
		}
		if _, ok := pt.(*types.Map); ok {
			hasCollection = true
		}
	}
	return hasIter && !hasCollection
}

// isIteratorType returns true if t is iter.Seq or iter.Seq2.
//
// Takes t (types.Type) which provides the type to check.
//
// Returns true if the type is an iterator type from the iter
// package.
func isIteratorType(t types.Type) bool {
	if named, ok := t.(*types.Named); ok {
		return named.Obj().Pkg() != nil && named.Obj().Pkg().Path() == iterPackagePath
	}
	if tp, ok := t.(*types.TypeParam); ok {
		return hasYieldMethod(tp)
	}
	return false
}

// hasYieldMethod returns true if the type parameter's constraint
// interface has a method named "Yield".
//
// Takes tp (*types.TypeParam) which provides the type parameter
// whose constraint is inspected.
//
// Returns true if the constraint interface contains a Yield method.
func hasYieldMethod(tp *types.TypeParam) bool {
	iface, ok := tp.Constraint().Underlying().(*types.Interface)
	if !ok || iface.NumMethods() == 0 {
		return false
	}
	for method := range iface.Methods() {
		if method.Name() == "Yield" {
			return true
		}
	}
	return false
}

// analyseFuncParam inspects a func-typed parameter's signature and
// determines how it relates to the outer function's type parameters.
//
// Takes paramType (types.Type) which provides the type of the
// function parameter to analyse.
// Takes analysis (typeParamAnalysis) which provides the outer
// function's type parameter analysis.
//
// Returns the func parameter info, or nil if the parameter does not
// use type parameters.
func analyseFuncParam(paramType types.Type, analysis typeParamAnalysis) *funcParamInfo {
	signature, ok := paramType.Underlying().(*types.Signature)
	if !ok {
		return nil
	}

	var parameters []string
	var elementPosition, keyPosition, valuePosition []int
	hasTypeParam := false

	i := 0
	for p := range signature.Params().Variables() {
		pName := fmt.Sprintf("p%d", i)
		parameters = append(parameters, pName)

		if tp, ok := p.Type().(*types.TypeParam); ok {
			hasTypeParam = true
			if analysis.keyParam != nil && tp.Index() == analysis.keyParam.Index() {
				keyPosition = append(keyPosition, i)
			} else {
				elementPosition = append(elementPosition, i)
			}
		}
		i++
	}

	if !hasTypeParam {
		return nil
	}

	var retType string
	if signature.Results().Len() > 0 {
		retType = concreteTypeName(signature.Results().At(0).Type())
	}

	var wrapperParams []*ast.Field
	for _, pName := range parameters {
		wrapperParams = append(wrapperParams, goastutil.Field(pName, goastutil.CachedIdent(identAny)))
	}
	var wrapperResults *ast.FieldList
	if retType != "" {
		wrapperResults = goastutil.FieldList(goastutil.Field("", parseTypeExpr(retType)))
	}
	wrapperFuncType := goastutil.FuncType(goastutil.FieldList(wrapperParams...), wrapperResults)

	return &funcParamInfo{
		wrapperFuncType:       wrapperFuncType,
		adapterParams:         parameters,
		adapterReturn:         retType,
		elementPositionitions: elementPosition,
		keyPositionitions:     keyPosition,
		valuePositionitions:   valuePosition,
	}
}

// yieldWrapperFuncType builds a func(yield func(parameters...) bool)
// AST expression from the given yield parameter fields.
//
// Takes yieldParams (...*ast.Field) which provides the AST field
// definitions for the yield callback parameters.
//
// Returns the AST expression representing the iterator function
// type.
func yieldWrapperFuncType(yieldParams ...*ast.Field) ast.Expr {
	yieldType := goastutil.FuncType(
		goastutil.FieldList(yieldParams...),
		goastutil.FieldList(goastutil.Field("", goastutil.CachedIdent(identBool))),
	)
	return goastutil.FuncType(
		goastutil.FieldList(goastutil.Field(identYield, yieldType)),
		nil,
	)
}

// analyseIterParam inspects an iterator-typed parameter and returns
// a funcParamInfo for the wrapper's any-ified iterator signature.
//
// Takes paramType (types.Type) which provides the type to inspect
// for iterator patterns.
//
// Returns the parameter kind and func param info, or paramConcrete
// and nil if not an iterator.
func analyseIterParam(paramType types.Type, _ typeParamAnalysis) (paramKindInWrapper, *funcParamInfo) {
	named, ok := paramType.(*types.Named)
	if !ok {
		return paramConcrete, nil
	}
	if named.Obj().Pkg() == nil || named.Obj().Pkg().Path() != iterPackagePath {
		return paramConcrete, nil
	}

	targs := named.TypeArgs()
	if targs == nil || targs.Len() == 0 {
		return paramConcrete, nil
	}

	switch named.Obj().Name() {
	case "Seq":
		wft := yieldWrapperFuncType(goastutil.Field(identV, goastutil.CachedIdent(identAny)))
		var elementPosition []int
		if targs.Len() > 0 {
			elementPosition = []int{0}
		}
		return paramIterSeq, &funcParamInfo{
			wrapperFuncType:       wft,
			adapterParams:         []string{identV},
			adapterReturn:         identBool,
			elementPositionitions: elementPosition,
		}

	case "Seq2":
		wft := yieldWrapperFuncType(
			goastutil.Field(identK, goastutil.CachedIdent(identAny)),
			goastutil.Field(identV, goastutil.CachedIdent(identAny)),
		)
		return paramIterSeq2, &funcParamInfo{
			wrapperFuncType:     wft,
			adapterParams:       []string{identK, identV},
			adapterReturn:       identBool,
			keyPositionitions:   []int{0},
			valuePositionitions: []int{1},
		}
	}

	return paramConcrete, nil
}

// generateSliceWrapperAST produces a wrapper function AST for a
// generic function dispatched by slice type.
//
// Takes gf (GenericFuncInfo) which provides the generic function
// info.
// Takes alias (string) which specifies the import alias.
// Takes config (PackageConfig) which provides the type configuration.
// Takes analysis (typeParamAnalysis) which provides the type
// parameter dispatch analysis.
//
// Returns the generated declaration, its metadata, and any error
// encountered.
func generateSliceWrapperAST(gf GenericFuncInfo, alias string, config PackageConfig, analysis typeParamAnalysis) (*ast.FuncDecl, *wrapperMeta, error) {
	signature := gf.Signature
	elemTypes, _, _ := config.TypesForFunc(gf.Name)
	if len(elemTypes) == 0 {
		return nil, nil, nil
	}

	functionName := "wrapped" + titleCase(alias) + gf.Name
	parameters := classifyParams(signature, analysis)
	returns := classifyReturns(signature)

	var caseClauses []ast.Stmt
	for _, et := range elemTypes {
		caseClauses = append(caseClauses, buildCaseClauseAST(
			sliceTypeExpr(et), gf, alias, parameters, et, slicePrefix+et, dispatchSlice,
		))
	}

	declaration := buildWrapperDeclAST(functionName, parameters, returns, caseClauses, alias, gf.Name)
	return declaration, &wrapperMeta{OriginalName: gf.Name, FuncName: functionName}, nil
}

// generateMapWrapperAST produces a wrapper function AST for a
// generic function dispatched by map type.
//
// Takes gf (GenericFuncInfo) which provides the generic function
// info.
// Takes alias (string) which specifies the import alias.
// Takes config (PackageConfig) which provides the type configuration.
//
// Returns the generated declaration, its metadata, and any error
// encountered.
func generateMapWrapperAST(gf GenericFuncInfo, alias string, config PackageConfig, _ typeParamAnalysis) (*ast.FuncDecl, *wrapperMeta, error) {
	signature := gf.Signature
	_, keyTypes, valTypes := config.TypesForFunc(gf.Name)
	if len(keyTypes) == 0 || len(valTypes) == 0 {
		return nil, nil, nil
	}

	functionName := "wrapped" + titleCase(alias) + gf.Name
	analysis := analyseTypeParams(signature)
	parameters := classifyParams(signature, analysis)
	returns := classifyReturns(signature)

	var caseClauses []ast.Stmt
	for _, kt := range keyTypes {
		for _, vt := range valTypes {
			mapType := mapPrefix + kt + mapCloser + vt
			caseClauses = append(caseClauses, buildCaseClauseAST(
				mapTypeExpr(kt, vt), gf, alias, parameters, "", mapType, dispatchMap,
			))
		}
	}

	declaration := buildWrapperDeclAST(functionName, parameters, returns, caseClauses, alias, gf.Name)
	return declaration, &wrapperMeta{OriginalName: gf.Name, FuncName: functionName}, nil
}

// generateScalarWrapperAST produces a wrapper function AST for a
// generic function dispatched by scalar type.
//
// Takes gf (GenericFuncInfo) which provides the generic function
// info.
// Takes alias (string) which specifies the import alias.
// Takes config (PackageConfig) which provides the type configuration.
//
// Returns the generated declaration, its metadata, and any error
// encountered.
func generateScalarWrapperAST(gf GenericFuncInfo, alias string, config PackageConfig, _ typeParamAnalysis) (*ast.FuncDecl, *wrapperMeta, error) {
	signature := gf.Signature
	elemTypes, _, _ := config.TypesForFunc(gf.Name)
	if len(elemTypes) == 0 {
		return nil, nil, nil
	}

	functionName := "wrapped" + titleCase(alias) + gf.Name
	analysis := analyseTypeParams(signature)
	parameters := classifyParams(signature, analysis)
	returns := classifyReturns(signature)

	var caseClauses []ast.Stmt
	for _, et := range elemTypes {
		caseClauses = append(caseClauses, buildCaseClauseAST(
			goastutil.CachedIdent(et), gf, alias, parameters, et, et, dispatchScalar,
		))
	}

	declaration := buildWrapperDeclAST(functionName, parameters, returns, caseClauses, alias, gf.Name)
	return declaration, &wrapperMeta{OriginalName: gf.Name, FuncName: functionName}, nil
}

// buildCaseClauseAST generates a single case clause in the type
// switch for a concrete type.
//
// Takes caseType (ast.Expr) which provides the AST expression for
// the case type.
// Takes gf (GenericFuncInfo) which provides the generic function
// info.
// Takes alias (string) which specifies the import alias.
// Takes parameters ([]paramInfo) which provides the classified
// parameters.
// Takes elemType (string) which specifies the element type name.
// Takes dispatchType (string) which specifies the full dispatch type
// string.
// Takes dk (dispatchKind) which specifies the dispatch kind.
//
// Returns the generated case clause AST node.
func buildCaseClauseAST(caseType ast.Expr, gf GenericFuncInfo, alias string, parameters []paramInfo, elemType, dispatchType string, dk dispatchKind) *ast.CaseClause {
	signature := gf.Signature
	firstParam := parameters[0]
	firstIsVariadic := firstParam.pkind == paramVariadicElem || firstParam.pkind == paramVariadicSlice

	if firstIsVariadic {
		return buildVariadicCaseClauseAST(caseType, gf, alias, parameters, elemType, dk, signature)
	}

	var statements []ast.Stmt

	lastParam := parameters[len(parameters)-1]
	isVariadicElem := signature.Variadic() && lastParam.pkind == paramVariadicElem
	isVariadicSlice := signature.Variadic() && lastParam.pkind == paramVariadicSlice

	if isVariadicElem {
		statements = append(statements, buildVariadicConversionAST(lastParam.name, elemType, false)...)
	}
	if isVariadicSlice {
		statements = append(statements, buildVariadicConversionAST(lastParam.name, elemType, true)...)
	}

	arguments := buildTypedCallArgs(parameters, elemType, dispatchType)
	callExpr := goastutil.CallExpr(goastutil.SelectorExpr(alias, gf.Name), arguments...)

	if isVariadicElem || isVariadicSlice {
		callExpr.Ellipsis = 1
	}

	if signature.Results().Len() == 0 {
		statements = append(statements, goastutil.ExprStmt(callExpr))
	} else {
		statements = append(statements, goastutil.ReturnStmt(callExpr))
	}

	return &ast.CaseClause{
		List: []ast.Expr{caseType},
		Body: statements,
	}
}

// buildTypedCallArgs builds the argument expressions for a typed
// call inside a case clause, converting each parameter according to
// its kind.
//
// Takes parameters ([]paramInfo) which provides the classified function
// parameters.
// Takes elemType (string) which specifies the element type name.
// Takes dispatchType (string) which specifies the full dispatch type
// string.
//
// Returns the list of argument AST expressions for the typed call.
func buildTypedCallArgs(parameters []paramInfo, elemType, dispatchType string) []ast.Expr {
	keyType, valType := extractKeyValTypes(dispatchType)
	effectiveElemType := elemType
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

// buildParamArgExpr builds a single argument expression for a
// parameter based on its kind.
//
// Takes p (paramInfo) which provides the parameter info.
// Takes elemType (string) which specifies the element type name.
// Takes keyType (string) which specifies the map key type name.
// Takes valType (string) which specifies the map value type name.
// Takes dispatchType (string) which specifies the full dispatch type
// string.
//
// Returns the AST expression for the argument in the typed call.
func buildParamArgExpr(p paramInfo, elemType, keyType, valType, dispatchType string) ast.Expr {
	switch p.pkind {
	case paramElem:
		return coerceCallAST(elemType, p.name)
	case paramSliceOfElem:
		return goastutil.TypeAssertExpr(
			goastutil.CachedIdent(p.name), sliceTypeExpr(elemType),
		)
	case paramMapType:
		return goastutil.TypeAssertExpr(
			goastutil.CachedIdent(p.name), parseTypeExpr(dispatchType),
		)
	case paramVariadicElem, paramVariadicSlice:
		return goastutil.CachedIdent("typedVariadic")
	case paramFuncOfElem:
		return buildFuncAdapterAST(p.funcInfo, p.name, elemType, keyType)
	case paramFuncOfKeyVal:
		return buildFuncAdapterAST(p.funcInfo, p.name, valType, keyType)
	case paramIterSeq:
		return buildIteratorAdapterAST(p.funcInfo, p.name, elemType, keyType, false)
	case paramIterSeq2:
		return buildIteratorAdapterAST(p.funcInfo, p.name, valType, keyType, true)
	default:
		return goastutil.CachedIdent(p.name)
	}
}

// buildVariadicCaseClauseAST builds a case clause for functions
// where the dispatch param is the variadic param (e.g. cmp.Or,
// slices.Concat).
//
// Takes caseType (ast.Expr) which provides the AST expression for
// the case type.
// Takes gf (GenericFuncInfo) which provides the generic function
// info.
// Takes alias (string) which specifies the import alias.
// Takes parameters ([]paramInfo) which provides the classified
// parameters.
// Takes elemType (string) which specifies the element type name.
// Takes dk (dispatchKind) which specifies the dispatch kind.
// Takes signature (*types.Signature) which provides the function
// signature.
//
// Returns the generated case clause AST node.
func buildVariadicCaseClauseAST(caseType ast.Expr, gf GenericFuncInfo, alias string, parameters []paramInfo, elemType string, dk dispatchKind, signature *types.Signature) *ast.CaseClause {
	p := parameters[0]

	var targetSliceType ast.Expr
	var coerceType string
	switch dk {
	case dispatchSlice:
		targetSliceType = &ast.ArrayType{Elt: sliceTypeExpr(elemType)}
		coerceType = slicePrefix + elemType
	default:
		targetSliceType = sliceTypeExpr(elemType)
		coerceType = elemType
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

	callExpr := goastutil.CallExpr(
		goastutil.SelectorExpr(alias, gf.Name),
		goastutil.CachedIdent("typed"),
	)
	callExpr.Ellipsis = 1

	var callStmt ast.Stmt
	if signature.Results().Len() == 0 {
		callStmt = goastutil.ExprStmt(callExpr)
	} else {
		callStmt = goastutil.ReturnStmt(callExpr)
	}

	return &ast.CaseClause{
		List: []ast.Expr{caseType},
		Body: []ast.Stmt{defineTyped, rangeStmt, callStmt},
	}
}

// buildVariadicConversionAST generates the typed conversion loop for
// a variadic parameter that is not the dispatch parameter.
//
// Takes paramName (string) which specifies the name of the variadic
// parameter.
// Takes elemType (string) which specifies the element type name.
// Takes isSlice (bool) which indicates whether the variadic elements
// are themselves slices.
//
// Returns the AST statements for the typed conversion loop.
func buildVariadicConversionAST(paramName, elemType string, isSlice bool) []ast.Stmt {
	var targetType ast.Expr
	var coerceType string
	if isSlice {
		targetType = &ast.ArrayType{Elt: sliceTypeExpr(elemType)}
		coerceType = slicePrefix + elemType
	} else {
		targetType = sliceTypeExpr(elemType)
		coerceType = elemType
	}

	makeCall := goastutil.CallExpr(
		goastutil.CachedIdent("make"),
		targetType,
		goastutil.CallExpr(goastutil.CachedIdent("len"), goastutil.CachedIdent(paramName)),
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
		X:     goastutil.CachedIdent(paramName),
		Body:  goastutil.BlockStmt(rangeBody),
	}

	return []ast.Stmt{defineTyped, rangeStmt}
}

// buildWrapperDeclAST constructs the complete wrapper function
// declaration with type switch and default panic clause.
//
// Takes functionName (string) which specifies the wrapper function name.
// Takes parameters ([]paramInfo) which provides the classified
// parameters.
// Takes returns ([]paramInfo) which provides the classified return
// types.
// Takes caseClauses ([]ast.Stmt) which provides the type switch
// case clauses.
// Takes alias (string) which specifies the import alias.
// Takes origName (string) which specifies the original generic
// function name.
//
// Returns the complete wrapper function declaration AST node.
func buildWrapperDeclAST(functionName string, parameters []paramInfo, returns []paramInfo, caseClauses []ast.Stmt, alias, origName string) *ast.FuncDecl {
	paramFields := buildParamFieldList(parameters)
	resultFields := buildResultFieldList(returns)

	firstParam := parameters[0]
	isVariadicDispatch := firstParam.pkind == paramVariadicElem || firstParam.pkind == paramVariadicSlice

	caseClauses = append(caseClauses, buildDefaultClauseAST(alias, origName, firstParam.name, isVariadicDispatch))

	var bodyStmts []ast.Stmt

	if isVariadicDispatch {
		bodyStmts = append(bodyStmts, buildLengthGuardAST(firstParam.name, returns))

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

// buildParamFieldList converts classified parameter info into an AST
// field list for the wrapper declaration.
//
// Takes parameters ([]paramInfo) which provides the classified function
// parameters.
//
// Returns the AST field list representing the wrapper function's
// parameters.
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

// buildResultFieldList converts classified return type info into an
// AST field list for the wrapper declaration.
//
// Takes returns ([]paramInfo) which provides the classified return
// types.
//
// Returns the AST field list representing the wrapper function's
// return types, or nil if there are none.
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

// buildLengthGuardAST generates the early return for variadic
// dispatch when the input is empty.
//
// Takes paramName (string) which specifies the variadic parameter
// name to check.
// Takes returns ([]paramInfo) which provides the classified return
// types for the zero-value return.
//
// Returns the if-statement AST node that guards against empty
// variadic input.
func buildLengthGuardAST(paramName string, returns []paramInfo) *ast.IfStmt {
	condition := &ast.BinaryExpr{
		X:  goastutil.CallExpr(goastutil.CachedIdent("len"), goastutil.CachedIdent(paramName)),
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

// buildDefaultClauseAST generates the default clause that panics
// with a formatted error message.
//
// Takes alias (string) which specifies the import alias.
// Takes origName (string) which specifies the original function
// name.
// Takes paramName (string) which specifies the dispatch parameter
// name.
// Takes isVariadicDispatch (bool) which indicates whether the
// dispatch is on a variadic parameter.
//
// Returns the default case clause AST node containing the panic
// call.
func buildDefaultClauseAST(alias, origName, paramName string, isVariadicDispatch bool) *ast.CaseClause {
	var fmtArg ast.Expr
	if isVariadicDispatch {
		fmtArg = goastutil.IndexExpr(goastutil.CachedIdent(paramName), goastutil.IntLit(0))
	} else {
		fmtArg = goastutil.CachedIdent(paramName)
	}

	panicCall := goastutil.CallExpr(
		goastutil.CachedIdent("panic"),
		goastutil.CallExpr(
			goastutil.SelectorExpr("fmt", "Sprintf"),
			goastutil.StrLit(alias+"."+origName+": unsupported type %T"),
			fmtArg,
		),
	)

	return &ast.CaseClause{
		Body: []ast.Stmt{goastutil.ExprStmt(panicCall)},
	}
}

// coerceCallAST generates a coerce[T](v) expression.
//
// Takes targetType (string) which specifies the target type name for
// the coercion.
// Takes varName (string) which specifies the variable name to
// coerce.
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

// extractKeyValTypes splits a dispatch type like "map[string]int"
// into key and value parts. For non-map types, returns elemType as
// valType with empty keyType.
//
// Takes dispatchType (string) which provides the type string to
// split.
//
// Returns the key type and value type extracted from the dispatch
// type string.
func extractKeyValTypes(dispatchType string) (keyType, valType string) {
	if strings.HasPrefix(dispatchType, mapPrefix) {
		closeBracket := strings.Index(dispatchType, "]")
		return dispatchType[mapPrefixLen:closeBracket], dispatchType[closeBracket+1:]
	}
	return "", dispatchType
}

// buildFuncAdapterAST generates a closure literal that adapts an
// any-typed func param to a concrete-typed func.
//
// Takes fi (*funcParamInfo) which provides the func parameter info.
// Takes paramName (string) which specifies the original parameter
// name.
// Takes elemType (string) which specifies the element type name.
// Takes keyType (string) which specifies the map key type name.
//
// Returns the AST expression for the adapter closure literal.
func buildFuncAdapterAST(fi *funcParamInfo, paramName, elemType, keyType string) ast.Expr {
	adapterFields := make([]*ast.Field, 0, len(fi.adapterParams))
	for i, pName := range fi.adapterParams {
		var concreteType string
		if slices.Contains(fi.keyPositionitions, i) {
			concreteType = keyType
		} else if slices.Contains(fi.valuePositionitions, i) || slices.Contains(fi.elementPositionitions, i) {
			concreteType = elemType
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
	callExpr := goastutil.CallExpr(goastutil.CachedIdent(paramName), callArgs...)

	var body *ast.BlockStmt
	if fi.adapterReturn != "" {
		body = goastutil.BlockStmt(goastutil.ReturnStmt(callExpr))
	} else {
		body = goastutil.BlockStmt(goastutil.ExprStmt(callExpr))
	}

	return goastutil.FuncLit(adapterType, body)
}

// buildIteratorAdapterAST generates a closure that adapts an
// any-typed iterator to a concrete-typed one.
//
// Takes paramName (string) which specifies the iterator parameter
// name.
// Takes elemType (string) which specifies the element type name.
// Takes keyType (string) which specifies the map key type name.
// Takes isSeq2 (bool) which indicates whether this is a two-element
// iterator.
//
// Returns the AST expression for the iterator adapter closure
// literal.
func buildIteratorAdapterAST(_ *funcParamInfo, paramName, elemType, keyType string, isSeq2 bool) ast.Expr {
	var yieldParams []*ast.Field
	var innerCallArgs []ast.Expr

	if isSeq2 {
		yieldParams = append(yieldParams,
			goastutil.Field(identK, parseTypeExpr(keyType)),
			goastutil.Field(identV, parseTypeExpr(elemType)),
		)
		innerCallArgs = append(innerCallArgs,
			coerceCallAST(keyType, identK),
			coerceCallAST(elemType, identV),
		)
	} else {
		yieldParams = append(yieldParams,
			goastutil.Field(identV, parseTypeExpr(elemType)),
		)
		innerCallArgs = append(innerCallArgs,
			coerceCallAST(elemType, identV),
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

	sequenceCall := goastutil.CallExpr(goastutil.CachedIdent(paramName), innerFunc)
	outerBody := goastutil.BlockStmt(goastutil.ExprStmt(sequenceCall))

	return goastutil.FuncLit(outerType, outerBody)
}

// sliceTypeExpr returns []T as an AST expression.
//
// Takes elemType (string) which specifies the element type name for
// the slice.
//
// Returns the AST array type expression representing the slice
// type.
func sliceTypeExpr(elemType string) ast.Expr {
	return &ast.ArrayType{Elt: parseTypeExpr(elemType)}
}

// mapTypeExpr returns map[K]V as an AST expression.
//
// Takes keyType (string) which specifies the map key type name.
// Takes valType (string) which specifies the map value type name.
//
// Returns the AST map type expression.
func mapTypeExpr(keyType, valType string) ast.Expr {
	return goastutil.MapType(parseTypeExpr(keyType), parseTypeExpr(valType))
}

// parseTypeExpr converts a type name string to an AST expression.
//
// Takes typeString (string) which provides the type name string to
// parse, supporting simple types, slices, maps, pointers, and
// qualified names.
//
// Returns the AST expression representing the parsed type.
func parseTypeExpr(typeString string) ast.Expr {
	if strings.HasPrefix(typeString, slicePrefix) {
		return &ast.ArrayType{Elt: parseTypeExpr(typeString[2:])}
	}
	if strings.HasPrefix(typeString, mapPrefix) {
		closeBracket := strings.Index(typeString, "]")
		keyType := typeString[mapPrefixLen:closeBracket]
		valType := typeString[closeBracket+1:]
		return goastutil.MapType(parseTypeExpr(keyType), parseTypeExpr(valType))
	}
	if strings.HasPrefix(typeString, "*") {
		return goastutil.StarExpr(parseTypeExpr(typeString[1:]))
	}
	if strings.Contains(typeString, ".") {
		parts := strings.SplitN(typeString, ".", qualifiedNameParts)
		return goastutil.SelectorExpr(parts[0], parts[1])
	}
	return goastutil.CachedIdent(typeString)
}

// classifyParams determines each parameter's relationship to the
// generic type parameters.
//
// Takes signature (*types.Signature) which provides the function
// signature.
// Takes analysis (typeParamAnalysis) which provides the type
// parameter dispatch analysis.
//
// Returns a slice of paramInfo describing each parameter's
// classification.
func classifyParams(signature *types.Signature, analysis typeParamAnalysis) []paramInfo {
	parameters := make([]paramInfo, signature.Params().Len())
	i := 0
	for p := range signature.Params().Variables() {
		name := sanitiseParamName(p.Name(), i)
		pkind := classifyParamType(p.Type(), signature, i, analysis)

		goType := identAny
		if pkind == paramConcrete {
			goType = concreteTypeName(p.Type())
		}

		pi := paramInfo{name: name, pkind: pkind, goType: goType, origIndex: i}

		switch pkind {
		case paramFuncOfElem, paramFuncOfKeyVal:
			pi.funcInfo = analyseFuncParam(p.Type(), analysis)
			if pi.funcInfo != nil {
				pi.astType = pi.funcInfo.wrapperFuncType
			}
		case paramIterSeq, paramIterSeq2:
			_, fi := analyseIterParam(p.Type(), analysis)
			pi.funcInfo = fi
			if fi != nil {
				pi.astType = fi.wrapperFuncType
			}
		}

		parameters[i] = pi
		i++
	}
	return parameters
}

// classifyVariadicParam checks whether a variadic param's element
// type is a type parameter, returning the appropriate kind.
//
// Takes t (types.Type) which provides the variadic parameter's type
// to classify.
//
// Returns the appropriate paramKindInWrapper for the variadic
// parameter.
func classifyVariadicParam(t types.Type) paramKindInWrapper {
	sl, ok := t.(*types.Slice)
	if !ok {
		return paramConcrete
	}
	tp, ok := sl.Elem().(*types.TypeParam)
	if !ok {
		return paramConcrete
	}
	if isSliceConstraint(tp) {
		return paramVariadicSlice
	}
	return paramVariadicElem
}

// classifyParamType determines the wrapper kind for a parameter
// based on its type and position.
//
// Takes t (types.Type) which provides the parameter type.
// Takes signature (*types.Signature) which provides the function
// signature.
// Takes index (int) which specifies the parameter index.
// Takes analysis (typeParamAnalysis) which provides the type
// parameter dispatch analysis.
//
// Returns the appropriate paramKindInWrapper for the parameter.
func classifyParamType(t types.Type, signature *types.Signature, index int, analysis typeParamAnalysis) paramKindInWrapper {
	if signature.Variadic() && index == signature.Params().Len()-1 {
		if vk := classifyVariadicParam(t); vk != paramConcrete {
			return vk
		}
	}

	if pkind, _ := analyseIterParam(t, analysis); pkind != paramConcrete {
		return pkind
	}

	if tp, ok := t.(*types.TypeParam); ok {
		return classifyTypeParamKind(tp)
	}

	return classifyCompositeParamType(t, analysis)
}

// classifyTypeParamKind determines the wrapper kind for a bare
// type parameter based on its constraint.
//
// Takes tp (*types.TypeParam) which provides the type parameter to
// classify.
//
// Returns the appropriate paramKindInWrapper based on the constraint
// analysis.
func classifyTypeParamKind(tp *types.TypeParam) paramKindInWrapper {
	if isMapConstraint(tp) {
		return paramMapType
	}
	if isSliceConstraint(tp) {
		return paramSliceOfElem
	}
	if iface, ok := tp.Constraint().Underlying().(*types.Interface); ok && iface.NumMethods() > 0 {
		return paramFuncOfElem
	}
	return paramElem
}

// classifyCompositeParamType classifies function, slice, and map
// parameters that contain type parameters.
//
// Takes t (types.Type) which provides the composite type to
// classify.
// Takes analysis (typeParamAnalysis) which provides the type
// parameter dispatch analysis.
//
// Returns the appropriate paramKindInWrapper for the composite
// parameter.
func classifyCompositeParamType(t types.Type, analysis typeParamAnalysis) paramKindInWrapper {
	if _, ok := t.Underlying().(*types.Signature); ok {
		fi := analyseFuncParam(t, analysis)
		if fi != nil {
			if len(fi.keyPositionitions) > 0 || len(fi.valuePositionitions) > 0 {
				return paramFuncOfKeyVal
			}
			return paramFuncOfElem
		}
	}

	if sl, ok := t.(*types.Slice); ok {
		if _, ok := sl.Elem().(*types.TypeParam); ok {
			return paramSliceOfElem
		}
	}

	if m, ok := t.(*types.Map); ok {
		_, keyTP := m.Key().(*types.TypeParam)
		_, valTP := m.Elem().(*types.TypeParam)
		if keyTP || valTP {
			return paramMapType
		}
	}

	return paramConcrete
}

// isSliceConstraint returns true if the type param's constraint has
// a structural type ~[]E.
//
// Takes tp (*types.TypeParam) which provides the type parameter
// whose constraint is inspected.
//
// Returns true if the constraint includes a slice structural type.
func isSliceConstraint(tp *types.TypeParam) bool {
	iface, ok := tp.Constraint().Underlying().(*types.Interface)
	if !ok {
		return false
	}
	for embedded := range iface.EmbeddedTypes() {
		if union, ok := embedded.(*types.Union); ok {
			for term := range union.Terms() {
				if _, ok := term.Type().(*types.Slice); ok {
					return true
				}
			}
		}
	}
	return false
}

// isMapConstraint returns true if the type param's constraint has a
// structural type ~map[K]V.
//
// Takes tp (*types.TypeParam) which provides the type parameter
// whose constraint is inspected.
//
// Returns true if the constraint includes a map structural type.
func isMapConstraint(tp *types.TypeParam) bool {
	iface, ok := tp.Constraint().Underlying().(*types.Interface)
	if !ok {
		return false
	}
	for embedded := range iface.EmbeddedTypes() {
		if union, ok := embedded.(*types.Union); ok {
			for term := range union.Terms() {
				if _, ok := term.Type().(*types.Map); ok {
					return true
				}
			}
		}
	}
	return false
}

// classifyReturns determines the paramInfo for each return value of
// the function signature.
//
// Takes signature (*types.Signature) which provides the function signature
// to analyse.
//
// Returns a slice of paramInfo describing each return value's
// classification.
func classifyReturns(signature *types.Signature) []paramInfo {
	results := make([]paramInfo, signature.Results().Len())
	i := 0
	for r := range signature.Results().Variables() {
		results[i] = classifyReturnType(r.Type())
		i++
	}
	return results
}

// classifyReturnType determines the paramInfo for a single return
// type, mapping generic return types to "any".
//
// Takes t (types.Type) which provides the return type to classify.
//
// Returns a paramInfo describing the return type's classification.
func classifyReturnType(t types.Type) paramInfo {
	switch {
	case isTypeParam(t):
		return paramInfo{pkind: paramElem, goType: identAny}
	case isIterNamed(t):
		return paramInfo{pkind: paramConcrete, goType: identAny}
	case isSignatureUnderlying(t):
		return paramInfo{pkind: paramConcrete, goType: identAny}
	case isGenericSlice(t):
		return paramInfo{pkind: paramSliceOfElem, goType: identAny}
	case isGenericMap(t):
		return paramInfo{pkind: paramMapType, goType: identAny}
	default:
		return paramInfo{pkind: paramConcrete, goType: concreteTypeName(t)}
	}
}

// isTypeParam returns true if the given type is a type parameter.
//
// Takes t (types.Type) which provides the type to check.
//
// Returns true if t is a types.TypeParam.
func isTypeParam(t types.Type) bool {
	_, ok := t.(*types.TypeParam)
	return ok
}

// isIterNamed returns true if the given type is a named type from
// the iter package.
//
// Takes t (types.Type) which provides the type to check.
//
// Returns true if t is a named type with package path equal to
// iterPackagePath.
func isIterNamed(t types.Type) bool {
	named, ok := t.(*types.Named)
	return ok && named.Obj().Pkg() != nil && named.Obj().Pkg().Path() == iterPackagePath
}

// isSignatureUnderlying returns true if the given type has a
// function signature as its underlying type.
//
// Takes t (types.Type) which provides the type to check.
//
// Returns true if the underlying type of t is a types.Signature.
func isSignatureUnderlying(t types.Type) bool {
	_, ok := t.Underlying().(*types.Signature)
	return ok
}

// isGenericSlice returns true if the given type is a slice whose
// element is a type parameter.
//
// Takes t (types.Type) which provides the type to check.
//
// Returns true if t is a slice type with a type parameter element.
func isGenericSlice(t types.Type) bool {
	sl, ok := t.(*types.Slice)
	if !ok {
		return false
	}
	_, ok = sl.Elem().(*types.TypeParam)
	return ok
}

// isGenericMap returns true if the given type is a map with type
// parameter keys or values.
//
// Takes t (types.Type) which provides the type to check.
//
// Returns true if t is a map type where the key or value is a type
// parameter.
func isGenericMap(t types.Type) bool {
	m, ok := t.(*types.Map)
	if !ok {
		return false
	}
	_, keyTP := m.Key().(*types.TypeParam)
	_, valTP := m.Elem().(*types.TypeParam)
	return keyTP || valTP
}

// sanitiseParamName ensures the parameter name is valid and doesn't
// shadow common imports.
//
// Takes name (string) which provides the original parameter name.
// Takes index (int) which specifies the parameter index used as a
// fallback name.
//
// Returns the sanitised parameter name safe for use in generated
// code.
func sanitiseParamName(name string, index int) string {
	if name == "" || name == "_" {
		return fmt.Sprintf("p%d", index)
	}
	reserved := map[string]bool{
		"slices": true, "maps": true, "cmp": true,
		"fmt": true, "reflect": true, "strings": true,
	}
	if reserved[name] {
		return name + "Arg"
	}
	return name
}

// concreteTypeName returns the string representation of a concrete
// Go type for code generation.
//
// Takes t (types.Type) which provides the type to convert to its
// string name.
//
// Returns the type name string suitable for use in generated source
// code.
func concreteTypeName(t types.Type) string {
	switch v := t.(type) {
	case *types.Basic:
		return v.Name()
	case *types.Slice:
		return slicePrefix + concreteTypeName(v.Elem())
	case *types.Map:
		return mapPrefix + concreteTypeName(v.Key()) + mapCloser + concreteTypeName(v.Elem())
	case *types.Pointer:
		return "*" + concreteTypeName(v.Elem())
	case *types.Named:
		if v.Obj().Pkg() != nil {
			return v.Obj().Pkg().Name() + "." + v.Obj().Name()
		}
		return v.Obj().Name()
	default:
		return t.String()
	}
}

// titleCase returns the string with its first character converted to
// uppercase.
//
// Takes s (string) which provides the string to convert.
//
// Returns the title-cased string, or the original string if it is
// empty.
func titleCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
