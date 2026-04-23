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
	"reflect"
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

	// SymbolLinkedGenericType is an exported generic type registered
	// via interp_link.LinkedGenericType so the interpreter can
	// instantiate pkg.Name[T] without source access.
	SymbolLinkedGenericType
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

// LinkedGenericFuncInfo holds metadata about a generic function whose
// interpreter dispatch is delegated to a non-generic sibling via a
// //piko:link directive.
type LinkedGenericFuncInfo struct {
	// Name is the exported generic function's name (the user writes
	// pkg.Name[T] in their .pk file).
	Name string

	// LinkTarget is the name of the sibling function the directive
	// points at. Declared in the same package; may be unexported.
	LinkTarget string

	// Params describes the generic's non-type-parameter arguments in
	// declaration order. Nil means the generic has no arguments.
	Params []GenericFieldTypeInfo

	// Results describes the generic's return values in declaration
	// order. Nil means the generic has no returns.
	Results []GenericFieldTypeInfo

	// TypeArgCount records how many type parameters the generic
	// declares; the interpreter prepends this many reflect.Type values
	// before the sibling's regular arguments.
	TypeArgCount int

	// Variadic mirrors the generic's IsVariadic flag so callers can
	// pass `...opts` at the tail.
	Variadic bool
}

// LinkedGenericTypeInfo captures the enough structure of an exported
// generic type for the codegen to emit an interp_link.LinkedGenericType
// sentinel. The interpreter later uses this sentinel to synthesise a
// generic types.Named and, at each user instantiation, a concrete
// reflect.Type built from the type arguments via reflect.StructOf.
type LinkedGenericTypeInfo struct {
	// Name is the exported type identifier.
	Name string

	// Fields describes the struct layout (only struct-backed generics
	// are supported in v1; other kinds fall back to the skip path).
	Fields []LinkedGenericFieldInfo

	// TypeArgCount is the number of type parameters the generic
	// declares.
	TypeArgCount int
}

// LinkedGenericFieldInfo records one field of a linked generic type
// along with the serialisable type tree needed to rebuild its type
// at interpreter instantiation time.
type LinkedGenericFieldInfo struct {
	// Name is the exported field identifier.
	Name string

	// Tag is the raw struct tag, without surrounding backticks.
	Tag string

	// FieldType is the serialisable type tree for this field.
	FieldType GenericFieldTypeInfo

	// Exported mirrors the field's Go export visibility.
	Exported bool
}

// GenericFieldTypeInfo is extract's internal mirror of the runtime
// interp_link.GenericFieldType. Codegen converts between the two so
// the emitted file can use the public runtime type without extract
// depending on reflect-specific constants at compile time.
type GenericFieldTypeInfo struct {
	// Element is the inner type for slice, array, pointer, chan, and
	// map value positions.
	Element *GenericFieldTypeInfo

	// Key is the key type for map positions.
	Key *GenericFieldTypeInfo

	// NamedPackage is the import path for GenericFieldKindNamed and
	// GenericFieldKindNamedGeneric.
	NamedPackage string

	// NamedName is the identifier for GenericFieldKindNamed and
	// GenericFieldKindNamedGeneric.
	NamedName string

	// TypeArgs are the per-position type arguments for
	// GenericFieldKindNamedGeneric. Empty for other kinds.
	TypeArgs []GenericFieldTypeInfo

	// ArrayLength is the fixed size of array types.
	ArrayLength int

	// TypeArgIndex is the 0-based position of the referenced type
	// parameter when Kind == GenericFieldKindTypeArg.
	TypeArgIndex int

	// Kind classifies this node; values mirror
	// interp_link.GenericFieldKind.
	Kind GenericFieldKind

	// BasicKind is the reflect.Kind for primitive types.
	BasicKind reflect.Kind
}

// GenericFieldKind mirrors interp_link.GenericFieldKind but lives
// inside extract so metadata building does not import the public
// package. The generator maps values 1-to-1 when writing the gen file.
type GenericFieldKind uint8

const (
	// GenericFieldKindBasic is a primitive type.
	GenericFieldKindBasic GenericFieldKind = iota

	// GenericFieldKindTypeArg is a reference to a type parameter.
	GenericFieldKindTypeArg

	// GenericFieldKindSlice is []Element.
	GenericFieldKindSlice

	// GenericFieldKindArray is [Length]Element.
	GenericFieldKindArray

	// GenericFieldKindMap is map[Key]Element.
	GenericFieldKindMap

	// GenericFieldKindPointer is *Element.
	GenericFieldKindPointer

	// GenericFieldKindChan is a channel whose element is Element.
	GenericFieldKindChan

	// GenericFieldKindInterface collapses any interface to the empty
	// interface in reflect terms.
	GenericFieldKindInterface

	// GenericFieldKindNamed references a named non-generic type from
	// another package.
	GenericFieldKindNamed

	// GenericFieldKindNamedGeneric references an instantiation of
	// another generic type, with TypeArgs holding the substitution.
	GenericFieldKindNamedGeneric

	// GenericFieldKindError is the Go built-in `error` interface.
	GenericFieldKindError
)

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

	// LinkedGenericFuncs holds generic functions annotated with
	// //piko:link, routed through non-generic siblings at runtime.
	LinkedGenericFuncs []LinkedGenericFuncInfo

	// LinkedGenericTypes holds generic types registered via the
	// interp_link.LinkedGenericType sentinel so the interpreter can
	// instantiate pkg.Name[T] at compile and runtime.
	LinkedGenericTypes []LinkedGenericTypeInfo
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
		Mode: packages.NeedTypes | packages.NeedName | packages.NeedSyntax | packages.NeedTypesInfo | packages.NeedFiles,
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
		ep, err := extractPackage(goPackage, isGeneric)
		if err != nil {
			return nil, err
		}
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
// symbols and any link-directive parse or validation error.
func extractPackage(goPackage *packages.Package, includeGeneric bool) (ExtractedPackage, error) {
	ep := ExtractedPackage{
		ImportPath: goPackage.PkgPath,
		Name:       goPackage.Name,
	}

	linksByName, err := resolvePackageLinks(goPackage)
	if err != nil {
		return ExtractedPackage{}, err
	}

	scope := goPackage.Types.Scope()
	for _, name := range scope.Names() {
		typeObject := scope.Lookup(name)
		if !typeObject.Exported() {
			continue
		}
		classifyExportedSymbol(&ep, typeObject, goPackage.Name, includeGeneric, linksByName)
	}

	sortExtractedPackage(&ep)

	if len(ep.GenericFuncs) > 0 {
		ep.TypesPackage = goPackage.Types
	}

	return ep, nil
}

// resolvePackageLinks collects and validates every //piko:link
// directive in the package.
//
// Takes goPackage (*packages.Package) which is the loaded package.
//
// Returns the validated directive map keyed by the annotated generic's
// name, and any parse or validation error wrapped with the package
// path for context.
func resolvePackageLinks(goPackage *packages.Package) (map[string]LinkDirective, error) {
	rawDirectives, err := collectLinkDirectives(goPackage)
	if err != nil {
		return nil, fmt.Errorf("package %s: %w", goPackage.PkgPath, err)
	}
	directives, err := validateLinkDirectives(goPackage, rawDirectives)
	if err != nil {
		return nil, fmt.Errorf("package %s: %w", goPackage.PkgPath, err)
	}
	linksByName := make(map[string]LinkDirective, len(directives))
	for _, link := range directives {
		linksByName[link.GenericName] = link
	}
	return linksByName, nil
}

// classifyExportedSymbol routes a scope object into the appropriate
// ExtractedPackage slot.
//
// Takes ep (*ExtractedPackage) which receives the classified entry.
// Takes typeObject (types.Object) which is the scope object under
// consideration.
// Takes packageName (string) which is the Go package name for the
// symbol.
// Takes includeGeneric (bool) which enables manifest-driven generic
// wrapper collection.
// Takes linksByName (map[string]LinkDirective) which maps generic
// function names to their //piko:link directives.
func classifyExportedSymbol(
	ep *ExtractedPackage,
	typeObject types.Object,
	packageName string,
	includeGeneric bool,
	linksByName map[string]LinkDirective,
) {
	if link, linked := linksByName[typeObject.Name()]; linked && appendLinkedGenericIfValid(ep, typeObject, link) {
		return
	}

	symbol := classifyObject(typeObject, packageName, includeGeneric)
	if symbol == nil {
		return
	}

	if symbol.Kind == SymbolGenericFunc {
		appendGenericFuncIfValid(ep, typeObject)
		return
	}

	if symbol.Kind == SymbolLinkedGenericType {
		appendLinkedGenericTypeIfValid(ep, typeObject)
		return
	}

	ep.Symbols = append(ep.Symbols, *symbol)
}

// appendLinkedGenericTypeIfValid re-runs classifyLinkedGenericType to
// capture the field-by-field descriptor and stores it on the
// ExtractedPackage.
//
// Takes ep (*ExtractedPackage) which receives the descriptor.
// Takes typeObject (types.Object) which is the candidate type name.
func appendLinkedGenericTypeIfValid(ep *ExtractedPackage, typeObject types.Object) {
	typeName, ok := typeObject.(*types.TypeName)
	if !ok {
		return
	}
	info, ok := classifyLinkedGenericType(typeName)
	if !ok {
		return
	}
	ep.LinkedGenericTypes = append(ep.LinkedGenericTypes, info)
}

// appendLinkedGenericIfValid records a directive-matched generic
// function on ep.
//
// Takes ep (*ExtractedPackage) which receives the entry.
// Takes typeObject (types.Object) which is the candidate function.
// Takes link (LinkDirective) which is the matching //piko:link
// directive.
//
// Returns true when the symbol was consumed via the linked path;
// false when the object is not a generic function and the normal
// classification should continue.
func appendLinkedGenericIfValid(ep *ExtractedPackage, typeObject types.Object, link LinkDirective) bool {
	fn, ok := typeObject.(*types.Func)
	if !ok {
		return false
	}
	signature, ok := fn.Type().(*types.Signature)
	if !ok || signature.TypeParams() == nil {
		return false
	}
	paramIndex := make(map[*types.TypeParam]int, signature.TypeParams().Len())
	for index := range signature.TypeParams().Len() {
		paramIndex[signature.TypeParams().At(index)] = index
	}
	paramInfos := extractLinkedFuncTuple(signature.Params(), paramIndex)
	resultInfos := extractLinkedFuncTuple(signature.Results(), paramIndex)

	ep.LinkedGenericFuncs = append(ep.LinkedGenericFuncs, LinkedGenericFuncInfo{
		Name:         fn.Name(),
		LinkTarget:   link.LinkTarget,
		Params:       paramInfos,
		Results:      resultInfos,
		Variadic:     signature.Variadic(),
		TypeArgCount: signature.TypeParams().Len(),
	})
	return true
}

// appendGenericFuncIfValid records a generic function exposed to the
// manifest-driven dispatch-wrapper path.
//
// Takes ep (*ExtractedPackage) which receives the entry.
// Takes typeObject (types.Object) which is the candidate function.
func appendGenericFuncIfValid(ep *ExtractedPackage, typeObject types.Object) {
	typeFunction, ok := typeObject.(*types.Func)
	if !ok {
		return
	}
	signature, ok := typeFunction.Type().(*types.Signature)
	if !ok {
		return
	}
	ep.GenericFuncs = append(ep.GenericFuncs, GenericFuncInfo{
		Name:      typeFunction.Name(),
		Func:      typeFunction,
		Signature: signature,
	})
}

// sortExtractedPackage sorts every slice of the ExtractedPackage by
// symbol name for deterministic codegen output.
//
// Takes ep (*ExtractedPackage) whose slices are sorted in place.
func sortExtractedPackage(ep *ExtractedPackage) {
	slices.SortFunc(ep.Symbols, func(a, b ExtractedSymbol) int {
		return strings.Compare(a.Name, b.Name)
	})
	slices.SortFunc(ep.GenericFuncs, func(a, b GenericFuncInfo) int {
		return strings.Compare(a.Name, b.Name)
	})
	slices.SortFunc(ep.LinkedGenericFuncs, func(a, b LinkedGenericFuncInfo) int {
		return strings.Compare(a.Name, b.Name)
	})
	slices.SortFunc(ep.LinkedGenericTypes, func(a, b LinkedGenericTypeInfo) int {
		return strings.Compare(a.Name, b.Name)
	})
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
// generic types that cannot be represented (non-struct underlying,
// constraint interfaces), and tagging registrable generics with
// SymbolLinkedGenericType so the caller can populate
// LinkedGenericTypes.
//
// Takes o (*types.TypeName) which provides the type name object to
// classify.
//
// Returns a pointer to the classified symbol or nil if skipped.
func classifyTypeName(o *types.TypeName) *ExtractedSymbol {
	if isGenericType(o) {
		if _, ok := classifyLinkedGenericType(o); ok {
			return &ExtractedSymbol{Name: o.Name(), Kind: SymbolLinkedGenericType}
		}
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
