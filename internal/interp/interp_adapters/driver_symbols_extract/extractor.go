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
	"go/constant"
	"go/types"
	"slices"
	"strings"

	"golang.org/x/tools/go/packages"
)

// SymbolKind classifies an exported symbol.
type SymbolKind int

const (
	// SymbolFunc is an exported function.
	SymbolFunc SymbolKind = iota

	// SymbolVar is an exported package-level variable.
	SymbolVar

	// SymbolConst is an exported typed constant.
	SymbolConst

	// SymbolType is an exported named type.
	SymbolType

	// SymbolGenericFunc is an exported generic function requiring a
	// dispatch wrapper.
	SymbolGenericFunc
)

// ExtractedSymbol holds metadata about a single exported symbol.
type ExtractedSymbol struct {
	// Name is the exported identifier (e.g. "Sprintf").
	Name string

	// ConstValue holds the string representation for typed constants,
	// such as "math.Pi", and is empty for non-constants.
	ConstValue string

	// Kind classifies the symbol.
	Kind SymbolKind

	// IsUntypedConst is true for constants without an explicit type.
	// These are resolved at compile time by go/types and do not need
	// runtime reflect.Value entries.
	IsUntypedConst bool
}

// GenericFuncInfo holds metadata about a generic function sufficient
// for code generation of dispatch wrappers.
type GenericFuncInfo struct {
	// Func is the go/types function object.
	Func *types.Func

	// Signature is the function's generic signature.
	Signature *types.Signature

	// Name is the function name (e.g. "Contains").
	Name string
}

// ExtractedPackage holds all exported symbols for a single package.
type ExtractedPackage struct {
	// TypesPackage is the loaded *types.Package for packages with generic
	// functions. Used by the types_loader codegen.
	TypesPackage *types.Package

	// ImportPath is the Go import path (e.g. "encoding/json").
	ImportPath string

	// Name is the package short name (e.g. "json").
	Name string

	// Symbols is the list of exported symbols, sorted by name.
	Symbols []ExtractedSymbol

	// GenericFuncs holds generic functions that need dispatch wrappers.
	GenericFuncs []GenericFuncInfo
}

// Extract loads the given Go packages and extracts their exported
// symbols, returning one ExtractedPackage per import path.
//
// Takes importPaths ([]string) which lists the Go import paths to
// load and extract.
// Takes genericConfigs (map[string]PackageConfig) which maps import
// paths to their generic configuration.
//
// Returns a slice of ExtractedPackage values or an error if loading
// fails.
func Extract(importPaths []string, genericConfigs map[string]PackageConfig) ([]ExtractedPackage, error) {
	config := &packages.Config{
		Mode: packages.NeedTypes | packages.NeedName,
	}

	pkgs, err := packages.Load(config, importPaths...)
	if err != nil {
		return nil, fmt.Errorf("loading packages: %w", err)
	}

	for _, goPackage := range pkgs {
		if len(goPackage.Errors) > 0 {
			return nil, fmt.Errorf("package %s: %s", goPackage.PkgPath, goPackage.Errors[0].Msg)
		}
	}

	result := make([]ExtractedPackage, 0, len(pkgs))
	for _, goPackage := range pkgs {
		_, isGeneric := genericConfigs[goPackage.PkgPath]
		ep := extractPackage(goPackage, isGeneric)
		result = append(result, ep)
	}

	return result, nil
}

// FormatConstantLiteral produces a raw literal for a constant value.
//
// Takes value (constant.Value) which provides the constant value to
// format.
//
// Returns the exact string representation of the constant.
func FormatConstantLiteral(value constant.Value) string {
	return value.ExactString()
}

// extractPackage walks the package scope and collects exported
// symbols.
//
// Takes goPackage (*packages.Package) which provides the loaded package to
// extract symbols from.
// Takes includeGeneric (bool) which controls whether generic
// functions are included.
//
// Returns an ExtractedPackage containing all classified exported
// symbols.
func extractPackage(goPackage *packages.Package, includeGeneric bool) ExtractedPackage {
	ep := ExtractedPackage{
		ImportPath: goPackage.PkgPath,
		Name:       goPackage.Name,
	}

	scope := goPackage.Types.Scope()
	names := scope.Names()

	for _, name := range names {
		typeObject := scope.Lookup(name)
		if !typeObject.Exported() {
			continue
		}

		symbol := classifyObject(typeObject, goPackage.Name, includeGeneric)
		if symbol == nil {
			continue
		}

		if symbol.Kind == SymbolGenericFunc {
			typeFunction, ok := typeObject.(*types.Func)
			if !ok {
				continue
			}
			signature, ok := typeFunction.Type().(*types.Signature)
			if !ok {
				continue
			}
			ep.GenericFuncs = append(ep.GenericFuncs, GenericFuncInfo{
				Name:      typeFunction.Name(),
				Func:      typeFunction,
				Signature: signature,
			})
			continue
		}

		ep.Symbols = append(ep.Symbols, *symbol)
	}

	slices.SortFunc(ep.Symbols, func(a, b ExtractedSymbol) int {
		return strings.Compare(a.Name, b.Name)
	})
	slices.SortFunc(ep.GenericFuncs, func(a, b GenericFuncInfo) int {
		return strings.Compare(a.Name, b.Name)
	})

	if len(ep.GenericFuncs) > 0 {
		ep.TypesPackage = goPackage.Types
	}

	return ep
}

// classifyObject determines the symbol kind and metadata for a
// types.Object, returning nil for objects that should be skipped.
//
// Takes typeObject (types.Object) which provides the type object to
// classify.
// Takes packageName (string) which specifies the package name for
// constant formatting.
// Takes includeGeneric (bool) which controls whether generic
// functions are classified.
//
// Returns a pointer to the classified symbol or nil if skipped.
func classifyObject(typeObject types.Object, packageName string, includeGeneric bool) *ExtractedSymbol {
	switch o := typeObject.(type) {
	case *types.Func:
		return classifyFunc(o, includeGeneric)
	case *types.Var:
		return &ExtractedSymbol{Name: o.Name(), Kind: SymbolVar}
	case *types.Const:
		return classifyConst(o, packageName)
	case *types.TypeName:
		return classifyTypeName(o)
	default:
		return nil
	}
}

// classifyFunc classifies an exported function, distinguishing
// generic functions when includeGeneric is set.
//
// Takes o (*types.Func) which provides the function object to
// classify.
// Takes includeGeneric (bool) which controls whether generic
// functions are classified.
//
// Returns a pointer to the classified symbol or nil if the function
// is skipped.
func classifyFunc(o *types.Func, includeGeneric bool) *ExtractedSymbol {
	signature, ok := o.Type().(*types.Signature)
	if !ok {
		return nil
	}
	if signature.TypeParams() != nil {
		if includeGeneric {
			return &ExtractedSymbol{Name: o.Name(), Kind: SymbolGenericFunc}
		}
		return nil
	}
	return &ExtractedSymbol{Name: o.Name(), Kind: SymbolFunc}
}

// classifyConst classifies an exported constant, skipping untyped
// constants and producing literal expressions for typed ones.
//
// Takes o (*types.Const) which provides the constant object to
// classify.
// Takes packageName (string) which specifies the package name for value
// formatting.
//
// Returns a pointer to the classified symbol or nil if skipped.
func classifyConst(o *types.Const, packageName string) *ExtractedSymbol {
	constValue := o.Val()
	if constValue == nil {
		return nil
	}

	basic, ok := o.Type().(*types.Basic)
	if ok && basic.Info()&types.IsUntyped != 0 {
		return &ExtractedSymbol{
			Name:           o.Name(),
			Kind:           SymbolConst,
			IsUntypedConst: true,
		}
	}

	constExpr := formatConstant(o, packageName)
	return &ExtractedSymbol{
		Name:       o.Name(),
		Kind:       SymbolConst,
		ConstValue: constExpr,
	}
}

// classifyTypeName classifies an exported named type, skipping
// generic types and constraint interfaces.
//
// Takes o (*types.TypeName) which provides the type name object to
// classify.
//
// Returns a pointer to the classified symbol or nil if skipped.
func classifyTypeName(o *types.TypeName) *ExtractedSymbol {
	if isGenericType(o) {
		return nil
	}

	if iface, ok := o.Type().Underlying().(*types.Interface); ok {
		if !iface.IsMethodSet() {
			return nil
		}
	}

	return &ExtractedSymbol{Name: o.Name(), Kind: SymbolType}
}

// isGenericType returns true if the type name refers to a generic
// type that cannot be represented as reflect.Value without
// instantiation.
//
// Takes o (*types.TypeName) which provides the type name object to
// inspect.
//
// Returns true if the type has type parameters.
func isGenericType(o *types.TypeName) bool {
	if o.IsAlias() {
		if alias, ok := o.Type().(*types.Alias); ok && alias.TypeParams() != nil {
			return true
		}
		if named, ok := o.Type().(*types.Named); ok && named.TypeParams() != nil {
			return true
		}
		return false
	}
	if named, ok := o.Type().(*types.Named); ok && named.TypeParams() != nil {
		return true
	}
	return false
}

// formatConstant produces a Go expression string for a typed
// constant's value, suitable for embedding in reflect.ValueOf(...).
//
// Takes c (*types.Const) which provides the constant to format.
// Takes packageName (string) which specifies the package name for
// qualified references.
//
// Returns the package-qualified constant reference string.
func formatConstant(c *types.Const, packageName string) string {
	return packageName + "." + c.Name()
}
