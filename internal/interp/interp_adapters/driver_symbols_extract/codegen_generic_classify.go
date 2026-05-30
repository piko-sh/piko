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
	"go/types"
	"strings"

	"piko.sh/piko/internal/goastutil"
)

// sliceTypeExpr returns []T as an AST expression.
//
// Takes elementType (string) which specifies the element type name for the slice.
//
// Returns the AST array type expression representing the slice type.
func sliceTypeExpr(elementType string) ast.Expr {
	return &ast.ArrayType{Elt: parseTypeExpr(elementType)}
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

// splitMapKeyVal splits a map type string into key and value parts.
//
// Balances brackets so that the leading "map[" opens a bracket, the key runs up to the
// "]" that returns the nesting depth to zero, and the value is whatever follows.
// Balancing the brackets correctly handles array-keyed maps such as "map[[4]byte]string",
// where the first "]" closes the inner array rather than the map key.
//
// Takes typeString (string) which provides the map type string to split. It must begin
// with the "map[" prefix.
//
// Returns keyType (string) which is the parsed key type.
// Returns valType (string) which is the parsed value type.
// Returns ok (bool) which is true when a balanced matching close bracket is found and
// false for a malformed prefix.
func splitMapKeyVal(typeString string) (keyType, valType string, ok bool) {
	depth := 1
	for position := mapPrefixLen; position < len(typeString); position++ {
		switch typeString[position] {
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				return typeString[mapPrefixLen:position], typeString[position+1:], true
			}
		}
	}
	return "", "", false
}

// parseTypeExpr converts a type name string to an AST expression.
//
// Takes typeString (string) which provides the type name string to parse, supporting
// simple types, slices, maps, pointers, and qualified names.
//
// Returns the AST expression representing the parsed type.
func parseTypeExpr(typeString string) ast.Expr {
	if strings.HasPrefix(typeString, slicePrefix) {
		return &ast.ArrayType{Elt: parseTypeExpr(typeString[2:])}
	}
	if strings.HasPrefix(typeString, mapPrefix) {
		if keyType, valType, ok := splitMapKeyVal(typeString); ok {
			return goastutil.MapType(parseTypeExpr(keyType), parseTypeExpr(valType))
		}
	}
	if strings.HasPrefix(typeString, "*") {
		return goastutil.StarExpr(parseTypeExpr(typeString[1:]))
	}
	if pkg, name, ok := strings.Cut(typeString, "."); ok {
		return goastutil.SelectorExpr(pkg, name)
	}
	return goastutil.CachedIdent(typeString)
}

// classifyParams determines each parameter's relationship to the generic type parameters.
//
// Takes signature (*types.Signature) which provides the function signature.
// Takes analysis (typeParamAnalysis) which provides the type parameter dispatch analysis.
//
// Returns a slice of paramInfo describing each parameter's classification.
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
		default:
		}

		parameters[i] = pi
		i++
	}
	return parameters
}

// classifyVariadicParam checks whether a variadic param's element type is a type
// parameter, returning the appropriate kind.
//
// Takes t (types.Type) which provides the variadic parameter's type to classify.
//
// Returns the appropriate paramKindInWrapper for the variadic parameter.
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

// classifyParamType determines the wrapper kind for a parameter based on its type and
// position.
//
// Takes t (types.Type) which provides the parameter type.
// Takes signature (*types.Signature) which provides the function signature.
// Takes index (int) which specifies the parameter index.
// Takes analysis (typeParamAnalysis) which provides the type parameter dispatch analysis.
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

// classifyTypeParamKind determines the wrapper kind for a bare type parameter based on
// its constraint.
//
// Takes tp (*types.TypeParam) which provides the type parameter to classify.
//
// Returns the appropriate paramKindInWrapper based on the constraint analysis.
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

// classifyCompositeParamType classifies function, slice, and map parameters that contain
// type parameters.
//
// Takes t (types.Type) which provides the composite type to classify.
// Takes analysis (typeParamAnalysis) which provides the type parameter dispatch analysis.
//
// Returns the appropriate paramKindInWrapper for the composite parameter.
func classifyCompositeParamType(t types.Type, analysis typeParamAnalysis) paramKindInWrapper {
	if _, ok := t.Underlying().(*types.Signature); ok {
		fi := analyseFuncParam(t, analysis)
		if fi != nil {
			if len(fi.keyPositions) > 0 || len(fi.valuePositions) > 0 {
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

// isSliceConstraint returns true if the type param's constraint has a structural type
// ~[]E.
//
// Takes tp (*types.TypeParam) which provides the type parameter whose constraint is
// inspected.
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

// isMapConstraint returns true if the type param's constraint has a structural type
// ~map[K]V.
//
// Takes tp (*types.TypeParam) which provides the type parameter whose constraint is
// inspected.
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

// classifyReturns determines the paramInfo for each return value of the function
// signature.
//
// Takes signature (*types.Signature) which provides the function signature to analyse.
//
// Returns a slice of paramInfo describing each return value's classification.
func classifyReturns(signature *types.Signature) []paramInfo {
	results := make([]paramInfo, signature.Results().Len())
	i := 0
	for r := range signature.Results().Variables() {
		results[i] = classifyReturnType(r.Type())
		i++
	}
	return results
}

// classifyReturnType determines the paramInfo for a single return type, mapping generic
// return types to "any".
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

// isIterNamed returns true if the given type is a named type from the iter package.
//
// Takes t (types.Type) which provides the type to check.
//
// Returns true if t is a named type with package path equal to iterPackagePath.
func isIterNamed(t types.Type) bool {
	named, ok := t.(*types.Named)
	return ok && named.Obj().Pkg() != nil && named.Obj().Pkg().Path() == iterPackagePath
}

// isSignatureUnderlying returns true if the given type has a function signature as its
// underlying type.
//
// Takes t (types.Type) which provides the type to check.
//
// Returns true if the underlying type of t is a types.Signature.
func isSignatureUnderlying(t types.Type) bool {
	_, ok := t.Underlying().(*types.Signature)
	return ok
}

// isGenericSlice returns true if the given type is a slice whose element is a type
// parameter.
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

// isGenericMap returns true if the given type is a map with type parameter keys or
// values.
//
// Takes t (types.Type) which provides the type to check.
//
// Returns true if t is a map type where the key or value is a type parameter.
func isGenericMap(t types.Type) bool {
	m, ok := t.(*types.Map)
	if !ok {
		return false
	}
	_, keyTP := m.Key().(*types.TypeParam)
	_, valTP := m.Elem().(*types.TypeParam)
	return keyTP || valTP
}

// sanitiseParamName ensures the parameter name is valid and doesn't shadow common
// imports.
//
// Takes name (string) which provides the original parameter name.
// Takes index (int) which specifies the parameter index used as a fallback name.
//
// Returns the sanitised parameter name safe for use in generated code.
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

// concreteTypeName returns the string representation of a concrete Go type for code
// generation.
//
// Takes t (types.Type) which provides the type to convert to its string name.
//
// Returns the type name string suitable for use in generated source code.
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

// titleCase returns the string with its first character converted to uppercase.
//
// Takes s (string) which provides the string to convert.
//
// Returns the title-cased string, or the original string if it is empty.
func titleCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
