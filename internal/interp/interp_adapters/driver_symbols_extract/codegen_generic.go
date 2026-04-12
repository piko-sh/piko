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
	"errors"
	"fmt"
	"go/ast"
	"go/types"
	"slices"

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
)

// comparatorFallbackKind describes which reflect-based fallback a generic slice
// function's default type-switch clause should use instead of an unconditional panic.
type comparatorFallbackKind int

const (
	// comparatorFallbackNone indicates the default clause should panic.
	comparatorFallbackNone comparatorFallbackKind = iota

	// comparatorFallbackMin indicates a reflectMinMaxFuncFallback call that selects the
	// minimum element.
	comparatorFallbackMin

	// comparatorFallbackMax indicates a reflectMinMaxFuncFallback call that selects the
	// maximum element.
	comparatorFallbackMax

	// comparatorFallbackSort indicates a reflectSortFuncFallback call that sorts the slice
	// in place.
	comparatorFallbackSort
)

// dispatchKind describes how a generic function's first type parameter should be
// dispatched.
type dispatchKind int

const (
	// dispatchScalar indicates the type parameter is a simple scalar type.
	dispatchScalar dispatchKind = iota

	// dispatchSlice indicates the type parameter is constrained to slice types.
	dispatchSlice

	// dispatchMap indicates the type parameter is constrained to map types.
	dispatchMap
)

// typeParamAnalysis holds the result of analysing a generic function's type parameters.
type typeParamAnalysis struct {
	// elemParam is the element type parameter (E in ~[]E, T in scalar, V in ~map[K]V).
	elemParam *types.TypeParam

	// keyParam is the key type parameter for map dispatch.
	keyParam *types.TypeParam

	// kind is the dispatch strategy determined from the constraint analysis.
	kind dispatchKind
}

// paramKindInWrapper classifies how a parameter relates to the generic type parameters.
type paramKindInWrapper int

const (
	// paramConcrete indicates a non-generic parameter kept as-is.
	paramConcrete paramKindInWrapper = iota

	// paramElem indicates an element type parameter such as E or T.
	paramElem

	// paramSliceOfElem indicates a slice type parameter like S constrained to ~[]E.
	paramSliceOfElem

	// paramMapType indicates a map type parameter like M constrained to ~map[K]V.
	paramMapType

	// paramVariadicElem indicates a variadic parameter with element type like ...E.
	paramVariadicElem

	// paramVariadicSlice indicates a variadic parameter with slice type like ...S where S is
	// ~[]E.
	paramVariadicSlice

	// paramFuncOfElem indicates a function parameter using element type parameters.
	paramFuncOfElem

	// paramFuncOfKeyVal indicates a function parameter using map key and value type
	// parameters.
	paramFuncOfKeyVal

	// paramIterSeq indicates an iter.Seq[E] parameter represented as func(func(E) bool).
	paramIterSeq

	// paramIterSeq2 indicates an iter.Seq2[K,V] parameter represented as func(func(K,V)
	// bool).
	paramIterSeq2
)

// funcParamInfo describes a func-typed parameter's relationship to the outer function's
// type parameters.
type funcParamInfo struct {
	// wrapperFuncType is the AST expression for the wrapper function's type signature.
	wrapperFuncType ast.Expr

	// adapterParams is the list of parameter names for the adapter closure.
	adapterParams []string

	// adapterReturn is the return type name for the adapter closure.
	adapterReturn string

	// elementPositions is the indices of parameters that correspond to the element type
	// parameter.
	elementPositions []int

	// keyPositions is the indices of parameters that correspond to the map key type
	// parameter.
	keyPositions []int

	// valuePositions is the indices of parameters that correspond to the map value type
	// parameter.
	valuePositions []int
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

	// origIndex is the original index of the parameter in the source function signature.
	origIndex int

	// pkind is the classification of how this parameter relates to generic type parameters.
	pkind paramKindInWrapper
}

// analyseTypeParams determines the dispatch pattern from a generic function's type
// parameters.
//
// Takes signature (*types.Signature) which provides the function signature to analyse.
//
// Returns a typeParamAnalysis describing the dispatch strategy for the function.
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

// scanConstraintForDispatch scans an interface constraint's embedded unions for slice
// (~[]E) or map (~map[K]V) structural types.
//
// Takes iface (*types.Interface) which provides the interface constraint to scan.
//
// Returns the analysis result and true if a structural type was found, or a zero value
// and false otherwise.
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

// matchStructuralType checks whether a union term type is a slice or map with
// type-parameter elements, returning the dispatch analysis.
//
// Takes termType (types.Type) which provides the union term type to inspect.
//
// Returns the analysis result and true if a match was found, or a zero value and false
// otherwise.
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

var (
	// errNotGenericFunc is returned when a generic_functions manifest entry names a symbol
	// that is not an exported generic function in the extracted package.
	errNotGenericFunc = errors.New("not an exported generic function")

	// errInstantiationArgCount is returned when an explicit instantiation supplies a
	// type-argument count that does not match the generic's declared type-parameter count.
	errInstantiationArgCount = errors.New("type-argument count mismatch")
)

// buildGenericInstantiationEntries produces symbolEntry values for every generic
// top-level function with an explicit instantiation configured via the manifest's
// generic_functions mapping. Each entry evaluates to reflect.ValueOf(pkg.Name[T1, T2,
// ...]).
//
// Takes extractedPackage (ExtractedPackage) which holds the package's generic functions
// and import metadata.
// Takes alias (string) which is the import alias used to qualify the generic function
// reference.
// Takes config (PackageConfig) which carries the GenericInstantiations mapping from the
// manifest.
//
// Returns one symbolEntry per configured function, keyed by the bare function name, or an
// error when a configured name does not resolve to an extracted generic function or
// supplies the wrong number of type arguments.
func buildGenericInstantiationEntries(extractedPackage ExtractedPackage, alias string, config PackageConfig) ([]symbolEntry, error) {
	if len(config.GenericInstantiations) == 0 {
		return nil, nil
	}

	genericByName := make(map[string]GenericFuncInfo, len(extractedPackage.GenericFuncs))
	for _, genericFunc := range extractedPackage.GenericFuncs {
		genericByName[genericFunc.Name] = genericFunc
	}

	names := make([]string, 0, len(config.GenericInstantiations))
	for name := range config.GenericInstantiations {
		names = append(names, name)
	}
	slices.Sort(names)

	entries := make([]symbolEntry, 0, len(names))
	for _, name := range names {
		named, err := buildGenericFunctionEntries(name, alias, genericByName, config.GenericInstantiations[name])
		if err != nil {
			return nil, err
		}
		entries = append(entries, named...)
	}
	return entries, nil
}

// buildGenericFunctionEntries produces one symbolEntry per configured instantiation of a
// single generic function, validating that the function resolves and every type-argument
// list matches its declared type-parameter count.
//
// Takes name (string) which is the bare generic function name.
// Takes alias (string) which qualifies the generic function reference.
// Takes genericByName (map[string]GenericFuncInfo) which indexes the package's extracted
// generic functions by name.
// Takes instantiations ([][]string) which lists the configured type-argument lists for
// the named generic.
//
// Returns one symbolEntry per instantiation, or an error when the function does not
// resolve, has no instantiations, or supplies the wrong number of type arguments.
func buildGenericFunctionEntries(name, alias string, genericByName map[string]GenericFuncInfo, instantiations [][]string) ([]symbolEntry, error) {
	genericFunc, ok := genericByName[name]
	if !ok {
		return nil, fmt.Errorf("%s: %w", name, errNotGenericFunc)
	}
	if len(instantiations) == 0 {
		return nil, fmt.Errorf("%s: no instantiations configured", name)
	}
	wantArgCount := genericFunc.Signature.TypeParams().Len()
	entries := make([]symbolEntry, 0, len(instantiations))
	for _, typeArgs := range instantiations {
		if len(typeArgs) == 0 {
			return nil, fmt.Errorf("%s: instantiation has no type arguments", name)
		}
		if len(typeArgs) != wantArgCount {
			return nil, fmt.Errorf("%s: %w: got %d, want %d", name, errInstantiationArgCount, len(typeArgs), wantArgCount)
		}
		entries = append(entries, symbolEntry{
			name:       name,
			expression: genericInstantiationExpr(alias, name, typeArgs),
		})
	}
	return entries, nil
}

// genericInstantiationExpr builds the AST expression reflect.ValueOf(pkg.Name[T1, T2,
// ...]) for a single instantiation.
//
// Takes alias (string) which qualifies the generic function.
// Takes name (string) which is the generic function name.
// Takes typeArgs ([]string) which lists the type-argument names, one per type parameter.
//
// Returns the AST expression wrapping the instantiated function in reflect.ValueOf.
func genericInstantiationExpr(alias, name string, typeArgs []string) ast.Expr {
	qualified := goastutil.SelectorExpr(alias, name)

	var instantiated ast.Expr
	if len(typeArgs) == 1 {
		instantiated = &ast.IndexExpr{X: qualified, Index: parseTypeExpr(typeArgs[0])}
	} else {
		indices := make([]ast.Expr, 0, len(typeArgs))
		for _, typeArg := range typeArgs {
			indices = append(indices, parseTypeExpr(typeArg))
		}
		instantiated = &ast.IndexListExpr{X: qualified, Indices: indices}
	}

	return goastutil.CallExpr(goastutil.SelectorExpr(reflectPackage, "ValueOf"), instantiated)
}

// generateWrappers produces AST function declarations for all generic functions in a
// package.
//
// Takes extractedPackage (ExtractedPackage) which provides the extracted package
// containing generic functions.
// Takes alias (string) which specifies the import alias for the package.
// Takes config (PackageConfig) which provides the type configuration for wrapper
// generation.
//
// Returns the generated function declarations, their metadata, and any error encountered.
func generateWrappers(extractedPackage ExtractedPackage, alias string, config PackageConfig) ([]*ast.FuncDecl, []wrapperMeta, error) {
	var decls []*ast.FuncDecl
	var metas []wrapperMeta
	for _, genericFunc := range extractedPackage.GenericFuncs {
		declaration, meta, err := generateWrapper(genericFunc, alias, config)
		if err != nil {
			return nil, nil, fmt.Errorf("%s: %w", genericFunc.Name, err)
		}
		if declaration != nil {
			decls = append(decls, declaration)
			metas = append(metas, *meta)
		}
	}
	return decls, metas, nil
}

// generateWrapper produces a single AST wrapper function declaration for a generic
// function.
//
// Takes genericFunc (GenericFuncInfo) which provides the generic function info to wrap.
// Takes alias (string) which specifies the import alias for the package.
// Takes config (PackageConfig) which provides the type configuration for wrapper
// generation.
//
// Returns the generated declaration, its metadata, and any error encountered.
func generateWrapper(genericFunc GenericFuncInfo, alias string, config PackageConfig) (*ast.FuncDecl, *wrapperMeta, error) {
	signature := genericFunc.Signature
	analysis := analyseTypeParams(signature)

	if hasUndispatchableIterParam(signature, analysis) {
		return nil, nil, nil
	}

	switch analysis.kind {
	case dispatchSlice:
		return generateSliceWrapperAST(genericFunc, alias, config, analysis)
	case dispatchMap:
		return generateMapWrapperAST(genericFunc, alias, config, analysis)
	case dispatchScalar:
		return generateScalarWrapperAST(genericFunc, alias, config, analysis)
	default:
		return nil, nil, nil
	}
}

// hasUndispatchableIterParam returns true for a signature that has an iterator param but
// no collection param (slice/map) to dispatch on, making it unsuitable for runtime
// type-switch dispatch.
//
// Takes signature (*types.Signature) which provides the function signature to inspect.
// Takes analysis (typeParamAnalysis) which provides the type parameter analysis for the
// function.
//
// Returns true if the function has iterator parameters but no dispatchable collection
// parameters.
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
// Returns true if the type is an iterator type from the iter package.
func isIteratorType(t types.Type) bool {
	if named, ok := t.(*types.Named); ok {
		return named.Obj().Pkg() != nil && named.Obj().Pkg().Path() == iterPackagePath
	}
	if tp, ok := t.(*types.TypeParam); ok {
		return hasYieldMethod(tp)
	}
	return false
}

// hasYieldMethod returns true if the type parameter's constraint interface has a method
// named "Yield".
//
// Takes tp (*types.TypeParam) which provides the type parameter whose constraint is
// inspected.
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

// analyseFuncParam inspects a func-typed parameter's signature and determines how it
// relates to the outer function's type parameters.
//
// Takes paramType (types.Type) which provides the type of the function parameter to
// analyse.
// Takes analysis (typeParamAnalysis) which provides the outer function's type parameter
// analysis.
//
// Returns the func parameter info, or nil if the parameter does not use type parameters.
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
		wrapperFuncType:  wrapperFuncType,
		adapterParams:    parameters,
		adapterReturn:    retType,
		elementPositions: elementPosition,
		keyPositions:     keyPosition,
		valuePositions:   valuePosition,
	}
}

// yieldWrapperFuncType builds a func(yield func(parameters...) bool) AST expression from
// the given yield parameter fields.
//
// Takes yieldParams (...*ast.Field) which provides the AST field definitions for the
// yield callback parameters.
//
// Returns the AST expression representing the iterator function type.
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

// analyseIterParam inspects an iterator-typed parameter and returns a funcParamInfo for
// the wrapper's any-ified iterator signature.
//
// Takes paramType (types.Type) which provides the type to inspect for iterator patterns.
//
// Returns the parameter kind and func param info, or paramConcrete and nil if not an
// iterator.
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
			wrapperFuncType:  wft,
			adapterParams:    []string{identV},
			adapterReturn:    identBool,
			elementPositions: elementPosition,
		}

	case "Seq2":
		wft := yieldWrapperFuncType(
			goastutil.Field(identK, goastutil.CachedIdent(identAny)),
			goastutil.Field(identV, goastutil.CachedIdent(identAny)),
		)
		return paramIterSeq2, &funcParamInfo{
			wrapperFuncType: wft,
			adapterParams:   []string{identK, identV},
			adapterReturn:   identBool,
			keyPositions:    []int{0},
			valuePositions:  []int{1},
		}
	}

	return paramConcrete, nil
}
