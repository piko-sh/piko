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

package goastutil

import (
	"go/types"
	"strings"

	"golang.org/x/tools/go/packages"
)

// IsGoInternalType checks if a type is a built-in primitive, a pre-declared
// identifier, or part of the standard library (including 'unsafe'). This check
// runs during serialisation when full type information is available.
//
// Takes typ (types.Type) which is the type to check.
// Takes allPackages (map[string]*packages.Package) which provides the set of
// known packages for checking if a type belongs to the standard library.
//
// Returns bool which is true if the type is internal to Go.
func IsGoInternalType(typ types.Type, allPackages map[string]*packages.Package) bool {
	if _, ok := typ.(*types.TypeParam); ok {
		return false
	}

	if IsPrimitive(typ) {
		return true
	}

	if isInternal, handled := checkNamedTypeInternal(typ, allPackages); handled {
		return isInternal
	}

	return checkCompositeTypeInternal(typ, allPackages)
}

// IsPrimitive checks whether a type is a primitive, handling named types and
// aliases correctly.
//
// Takes typ (types.Type) which is the type to check.
//
// Returns bool which is true if the type is a primitive.
func IsPrimitive(typ types.Type) bool {
	return IsPrimitiveRecursive(typ, make(map[types.Type]bool))
}

// IsPrimitiveRecursive checks if a type should be considered
// primitive for PackagePath determination, with cycle detection
// to handle recursive types safely.
//
// Takes typ (types.Type) which is the type to check.
// Takes seen (map[types.Type]bool) which tracks visited types for cycle
// detection.
//
// Returns bool which is true if the type is effectively primitive.
func IsPrimitiveRecursive(typ types.Type, seen map[types.Type]bool) bool {
	if typ == nil {
		return false
	}

	if _, ok := typ.(*types.TypeParam); ok {
		return false
	}

	if seen[typ] {
		return false
	}
	seen[typ] = true

	if isNonPrimitive := checkNamedOrAliasType(typ, seen); isNonPrimitive {
		return false
	}

	return checkUnderlyingPrimitive(typ.Underlying(), seen)
}

// checkNamedTypeInternal checks if a named type is part of Go's internal
// standard library.
//
// Takes typ (types.Type) which is the type to check.
// Takes allPackages (map[string]*packages.Package) which provides package data
// for lookup.
//
// Returns isInternal (bool) which is true when the type belongs to Go's
// internal packages.
// Returns wasHandled (bool) which is false when typ is not a named type.
func checkNamedTypeInternal(typ types.Type, allPackages map[string]*packages.Package) (isInternal bool, wasHandled bool) {
	underlying := types.Unalias(typ)
	named, ok := underlying.(*types.Named)
	if !ok || named.Obj() == nil {
		return false, false
	}

	namedPackage := named.Obj().Pkg()
	if namedPackage == nil {
		return true, true
	}

	if !isStandardLibraryPath(namedPackage.Path()) {
		return false, true
	}

	packagesPackage, ok := allPackages[namedPackage.Path()]
	if !ok {
		return false, true
	}

	if packagesPackage.Module == nil {
		return true, true
	}
	return false, false
}

// isStandardLibraryPath checks if a package path belongs to the standard
// library. Standard library packages do not contain a dot in the first
// segment of their path.
//
// Takes path (string) which is the package path to check.
//
// Returns bool which is true if the path belongs to the standard library.
func isStandardLibraryPath(path string) bool {
	firstSegment, _, found := strings.Cut(path, "/")
	if !found {
		firstSegment = path
	}
	return !strings.Contains(firstSegment, ".") || path == "unsafe"
}

// checkCompositeTypeInternal recursively checks if a composite type's
// elements are internal Go types.
//
// Takes typ (types.Type) which is the composite type to check.
// Takes allPackages (map[string]*packages.Package) which provides the set of
// known packages for type resolution.
//
// Returns bool which is true if all elements of the composite type are
// internal Go types.
func checkCompositeTypeInternal(typ types.Type, allPackages map[string]*packages.Package) bool {
	switch t := typ.Underlying().(type) {
	case *types.Signature:
		return true
	case *types.Array:
		return IsGoInternalType(t.Elem(), allPackages)
	case *types.Slice:
		return IsGoInternalType(t.Elem(), allPackages)
	case *types.Pointer:
		return IsGoInternalType(t.Elem(), allPackages)
	case *types.Chan:
		return IsGoInternalType(t.Elem(), allPackages)
	case *types.Map:
		return IsGoInternalType(t.Key(), allPackages) && IsGoInternalType(t.Elem(), allPackages)
	default:
		return false
	}
}

// checkNamedOrAliasType checks if a type is a named or alias type that would
// make it non-primitive.
//
// Takes typ (types.Type) which is the type to check.
// Takes seen (map[types.Type]bool) which tracks visited types to prevent
// infinite loops.
//
// Returns bool which is true if the type is not primitive.
func checkNamedOrAliasType(typ types.Type, seen map[types.Type]bool) bool {
	if named, isNamed := typ.(*types.Named); isNamed {
		if named.Obj() != nil && named.Obj().Pkg() != nil {
			return true
		}
	}

	if alias, isAlias := typ.(*types.Alias); isAlias {
		underlying := alias.Rhs()
		if underlying != nil {
			return !IsPrimitiveRecursive(underlying, seen)
		}
		if alias.Obj() != nil && alias.Obj().Pkg() != nil {
			return true
		}
	}

	return false
}

// checkUnderlyingPrimitive checks if an underlying type is a primitive type.
//
// Takes underlying (types.Type) which is the type to check.
// Takes seen (map[types.Type]bool) which tracks visited types to prevent loops.
//
// Returns bool which is true if the type is primitive or holds functions.
func checkUnderlyingPrimitive(underlying types.Type, seen map[types.Type]bool) bool {
	switch t := underlying.(type) {
	case *types.Signature:
		return true
	case *types.Basic:
		return true
	case *types.Interface:
		return t.NumMethods() == 0
	case *types.Map:
		return containsFunction(t.Key(), seen) || containsFunction(t.Elem(), seen)
	case *types.Slice:
		return containsFunction(t.Elem(), seen)
	case *types.Array:
		return containsFunction(t.Elem(), seen)
	case *types.Pointer:
		return containsFunction(t.Elem(), seen)
	case *types.Chan:
		return containsFunction(t.Elem(), seen)
	default:
		return false
	}
}

// containsFunction checks if a type is or contains a function type.
//
// Takes typ (types.Type) which is the type to inspect.
// Takes seen (map[types.Type]bool) which tracks visited types to prevent
// infinite recursion.
//
// Returns bool which is true if the type contains a function signature.
func containsFunction(typ types.Type, seen map[types.Type]bool) bool {
	if typ == nil {
		return false
	}

	if seen[typ] {
		return false
	}
	seen[typ] = true

	underlying := typ.Underlying()

	if _, isSignature := underlying.(*types.Signature); isSignature {
		return true
	}

	switch t := underlying.(type) {
	case *types.Map:
		return containsFunction(t.Key(), seen) || containsFunction(t.Elem(), seen)
	case *types.Slice:
		return containsFunction(t.Elem(), seen)
	case *types.Array:
		return containsFunction(t.Elem(), seen)
	case *types.Pointer:
		return containsFunction(t.Elem(), seen)
	case *types.Chan:
		return containsFunction(t.Elem(), seen)
	default:
		return false
	}
}
