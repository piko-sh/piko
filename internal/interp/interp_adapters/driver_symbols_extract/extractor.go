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
	"context"
	"errors"
	"fmt"
	"go/constant"
	"go/types"
	"math"
	"reflect"
	"slices"
	"strings"

	"golang.org/x/tools/go/packages"
)

// SymbolKind classifies an exported symbol.
type SymbolKind int

const (
	// errExtractWrap is the format string used to wrap errors surfaced from the extraction
	// pipeline so callers see a consistent prefix.
	errExtractWrap = "extract: %w"
)

const (
	// SymbolFunc is an exported function.
	SymbolFunc SymbolKind = iota

	// SymbolVar is an exported package-level variable.
	SymbolVar

	// SymbolConst is an exported typed constant.
	SymbolConst

	// SymbolType is an exported named type.
	SymbolType

	// SymbolGenericFunc is an exported generic function requiring a dispatch wrapper.
	SymbolGenericFunc

	// SymbolLinkedGenericType is an exported generic type registered via
	// interp_link.LinkedGenericType so the interpreter can instantiate pkg.Name[T] without
	// source access.
	SymbolLinkedGenericType
)

// ExtractedSymbol holds metadata about a single exported symbol.
type ExtractedSymbol struct {
	// Name is the exported identifier (e.g. "Sprintf").
	Name string

	// ConstValue holds the string representation for typed constants, such as "math.Pi", and
	// is empty for non-constants.
	ConstValue string

	// Kind classifies the symbol.
	Kind SymbolKind

	// IsUntypedConst is true for constants without an explicit type. These are resolved at
	// compile time by go/types and do not need runtime reflect.Value entries.
	IsUntypedConst bool
}

// GenericFuncInfo holds metadata about a generic function sufficient for code generation
// of dispatch wrappers.
type GenericFuncInfo struct {
	// Func is the go/types function object.
	Func *types.Func

	// Signature is the function's generic signature.
	Signature *types.Signature

	// Name is the function name (e.g. "Contains").
	Name string
}

// LinkedGenericFuncInfo holds metadata about a generic function whose interpreter
// dispatch is delegated to a non-generic sibling via a //piko:link directive.
type LinkedGenericFuncInfo struct {
	// Name is the exported generic function's name (the user writes pkg.Name[T] in their .pk
	// file).
	Name string

	// LinkTarget is the name of the sibling function the directive points at. Declared in
	// the same package; may be unexported.
	LinkTarget string

	// Params describes the generic's non-type-parameter arguments in declaration order. Nil
	// means the generic has no arguments.
	Params []GenericFieldTypeInfo

	// Results describes the generic's return values in declaration order. Nil means the
	// generic has no returns.
	Results []GenericFieldTypeInfo

	// TypeArgCount records how many type parameters the generic declares; the interpreter
	// prepends this many reflect.Type values before the sibling's regular arguments.
	TypeArgCount int

	// Variadic mirrors the generic's IsVariadic flag so callers can pass `...opts` at the
	// tail.
	Variadic bool
}

// LinkedGenericTypeInfo captures enough structure of an exported generic type for the
// codegen to emit an interp_link.LinkedGenericType sentinel. The interpreter later uses
// this sentinel to synthesise a generic types.Named and, at each user instantiation, a
// concrete reflect.Type built from the type arguments via reflect.StructOf.
type LinkedGenericTypeInfo struct {
	// Name is the exported type identifier.
	Name string

	// Fields describes the struct layout. Only struct-backed generics are supported; other
	// kinds fall back to the skip path.
	Fields []LinkedGenericFieldInfo

	// TypeArgCount is the number of type parameters the generic declares.
	TypeArgCount int
}

// LinkedGenericFieldInfo records one field of a linked generic type along with the
// serialisable type tree needed to rebuild its type at interpreter instantiation time.
type LinkedGenericFieldInfo struct {
	// Name is the exported field identifier.
	Name string

	// Tag is the raw struct tag, without surrounding backticks.
	Tag string

	// FieldType is the serialisable type tree for the field.
	FieldType GenericFieldTypeInfo

	// Exported mirrors the field's Go export visibility.
	Exported bool
}

// GenericFieldTypeInfo is extract's internal mirror of the runtime
// interp_link.GenericFieldType. Codegen converts between the two so the emitted file can
// use the public runtime type without extract depending on reflect-specific constants at
// compile time.
type GenericFieldTypeInfo struct {
	// Element is the inner type for slice, array, pointer, chan, and map value positions.
	Element *GenericFieldTypeInfo

	// Key is the key type for map positions.
	Key *GenericFieldTypeInfo

	// NamedPackage is the import path for GenericFieldKindNamed and
	// GenericFieldKindNamedGeneric.
	NamedPackage string

	// NamedName is the identifier for GenericFieldKindNamed and
	// GenericFieldKindNamedGeneric.
	NamedName string

	// TypeArgs are the per-position type arguments for GenericFieldKindNamedGeneric. Empty
	// for other kinds.
	TypeArgs []GenericFieldTypeInfo

	// ArrayLength is the fixed size of array types.
	ArrayLength int

	// TypeArgIndex is the 0-based position of the referenced type parameter when Kind ==
	// GenericFieldKindTypeArg.
	TypeArgIndex int

	// Kind classifies this node; values mirror interp_link.GenericFieldKind.
	Kind GenericFieldKind

	// BasicKind is the reflect.Kind for primitive types.
	BasicKind reflect.Kind
}

// GenericFieldKind mirrors interp_link.GenericFieldKind but lives inside extract so
// metadata building does not import the public package. The generator maps values 1-to-1
// when writing the gen file.
type GenericFieldKind uint8

const (
	// GenericFieldKindBasic is a primitive type.
	GenericFieldKindBasic GenericFieldKind = iota

	// GenericFieldKindTypeArg is a reference to a type parameter.
	GenericFieldKindTypeArg

	// GenericFieldKindSlice is []Element.
	GenericFieldKindSlice

	// GenericFieldKindArray is LengthElement.
	GenericFieldKindArray

	// GenericFieldKindMap is map[Key]Element.
	GenericFieldKindMap

	// GenericFieldKindPointer is *Element.
	GenericFieldKindPointer

	// GenericFieldKindChan is a channel whose element is Element.
	GenericFieldKindChan

	// GenericFieldKindInterface collapses any interface to the empty interface in reflect
	// terms.
	GenericFieldKindInterface

	// GenericFieldKindNamed references a named non-generic type from another package.
	GenericFieldKindNamed

	// GenericFieldKindNamedGeneric references an instantiation of another generic type, with
	// TypeArgs holding the substitution.
	GenericFieldKindNamedGeneric

	// GenericFieldKindError is the Go built-in `error` interface.
	GenericFieldKindError
)

var (
	// errNotGenericType is returned when a generic_types manifest entry names a type that is
	// not an exported, native-backed-eligible generic type in the extracted package.
	errNotGenericType = errors.New("not an exported generic type")

	// errErasureArgCount is returned when a generic_types entry supplies a canonical erasure
	// type-argument list whose length does not match the generic type's declared
	// type-parameter count.
	errErasureArgCount = errors.New("erasure type-argument count mismatch")
)

// ExtractedPackage holds all exported symbols for a single package.
type ExtractedPackage struct {
	// TypesPackage is the loaded *types.Package for packages with generic functions. Used by
	// the types_loader codegen.
	TypesPackage *types.Package

	// ImportPath is the Go import path (e.g. "encoding/json").
	ImportPath string

	// Name is the package short name (e.g. "json").
	Name string

	// Symbols is the list of exported symbols, sorted by name.
	Symbols []ExtractedSymbol

	// GenericFuncs holds generic functions that need dispatch wrappers.
	GenericFuncs []GenericFuncInfo

	// LinkedGenericFuncs holds generic functions annotated with //piko:link, routed through
	// non-generic siblings at runtime.
	LinkedGenericFuncs []LinkedGenericFuncInfo

	// LinkedGenericTypes holds generic types registered via the
	// interp_link.LinkedGenericType sentinel so the interpreter can instantiate pkg.Name[T]
	// at compile and runtime.
	LinkedGenericTypes []LinkedGenericTypeInfo

	// NativeBackedGenericTypes holds generic types opted into the native-backed pipeline via
	// the manifest's generic_types: config. They are registered with their real native
	// method set rather than a structural reflect.StructOf lookalike.
	NativeBackedGenericTypes []NativeBackedGenericTypeInfo
}

// NativeBackedGenericTypeInfo captures a generic stdlib type that the manifest opted into
// the native-backed pipeline. ErasureTypeArgs is the canonical erased type-argument list
// (one per type parameter) the codegen splices into the generated reflect.TypeOf
// expression.
type NativeBackedGenericTypeInfo struct {
	// Name is the exported generic type identifier (e.g. "Pointer").
	Name string

	// ErasureTypeArgs is the canonical erased type-argument list.
	ErasureTypeArgs []string

	// TypeArgCount is the number of type parameters the generic declares.
	TypeArgCount int
}

// Extract loads the given Go packages and extracts their exported symbols, returning one
// ExtractedPackage per import path.
//
// This is the context-free entry point for callers that have no cancellation to thread;
// it delegates to ExtractContext with a background context.
//
// Takes importPaths ([]string) which lists the Go import paths to load and extract.
// Takes genericConfigs (map[string]PackageConfig) which maps import paths to their
// generic configuration.
//
// Returns a slice of ExtractedPackage values or an error if loading fails.
func Extract(importPaths []string, genericConfigs map[string]PackageConfig) ([]ExtractedPackage, error) {
	return ExtractContext(context.Background(), importPaths, genericConfigs)
}

// ExtractContext loads the given Go packages and extracts their exported symbols,
// returning one ExtractedPackage per import path. The context is forwarded to
// packages.Load and checked before and after loading and inside the per-package
// extraction loop so a cancelled run aborts promptly.
//
// Takes ctx (context.Context) which carries cancellation.
// Takes importPaths ([]string) which lists the Go import paths to load and extract.
// Takes genericConfigs (map[string]PackageConfig) which maps import paths to their
// generic configuration.
//
// Returns a slice of ExtractedPackage values or an error if loading fails or the context
// is cancelled.
func ExtractContext(ctx context.Context, importPaths []string, genericConfigs map[string]PackageConfig) ([]ExtractedPackage, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf(errExtractWrap, err)
	}

	config := &packages.Config{
		Context: ctx,
		Mode:    packages.NeedTypes | packages.NeedName | packages.NeedSyntax | packages.NeedTypesInfo | packages.NeedFiles,
	}

	pkgs, err := packages.Load(config, importPaths...)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, fmt.Errorf(errExtractWrap, ctxErr)
		}
		return nil, fmt.Errorf("loading packages: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf(errExtractWrap, err)
	}

	for _, goPackage := range pkgs {
		if len(goPackage.Errors) > 0 {
			return nil, fmt.Errorf("package %s: %s", goPackage.PkgPath, goPackage.Errors[0].Msg)
		}
	}

	result := make([]ExtractedPackage, 0, len(pkgs))
	for _, goPackage := range pkgs {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf(errExtractWrap, err)
		}
		packageConfig, isGeneric := genericConfigs[goPackage.PkgPath]
		extractedPackage, err := extractPackage(goPackage, isGeneric, packageConfig.GenericTypes)
		if err != nil {
			return nil, err
		}
		result = append(result, extractedPackage)
	}

	return result, nil
}

// FormatConstantLiteral produces a raw literal for a constant value.
//
// Takes value (constant.Value) which provides the constant value to format.
//
// Returns the exact string representation of the constant.
func FormatConstantLiteral(value constant.Value) string {
	return value.ExactString()
}

// extractPackage walks the package scope and collects exported symbols.
//
// Takes goPackage (*packages.Package) which provides the loaded package to extract
// symbols from.
// Takes includeGeneric (bool) which controls whether generic functions are included.
// Takes nativeBackedTypes (map[string][]string) which maps generic type names opted into
// the native-backed pipeline to their canonical erased type-argument lists.
//
// Returns an ExtractedPackage containing all classified exported symbols and any
// link-directive parse or validation error.
func extractPackage(goPackage *packages.Package, includeGeneric bool, nativeBackedTypes map[string][]string) (ExtractedPackage, error) {
	extractedPackage := ExtractedPackage{
		ImportPath: goPackage.PkgPath,
		Name:       goPackage.Name,
	}

	linksByName, err := resolvePackageLinks(goPackage)
	if err != nil {
		return ExtractedPackage{}, err
	}

	matchedNativeTypes := make(map[string]struct{}, len(nativeBackedTypes))
	scope := goPackage.Types.Scope()
	for _, name := range scope.Names() {
		typeObject := scope.Lookup(name)
		if !typeObject.Exported() {
			continue
		}
		if err := classifyExportedSymbol(&extractedPackage, typeObject, goPackage.Name, includeGeneric, linksByName, nativeBackedTypes, matchedNativeTypes); err != nil {
			return ExtractedPackage{}, fmt.Errorf("package %s: %w", goPackage.PkgPath, err)
		}
	}

	if err := reportUnmatchedNativeTypes(goPackage.PkgPath, nativeBackedTypes, matchedNativeTypes); err != nil {
		return ExtractedPackage{}, err
	}

	sortExtractedPackage(&extractedPackage)

	if len(extractedPackage.GenericFuncs) > 0 {
		extractedPackage.TypesPackage = goPackage.Types
	}

	return extractedPackage, nil
}

// reportUnmatchedNativeTypes returns an error naming every generic_types manifest entry
// that did not match an exported, native-backed-eligible generic type in the package. A
// configured name that never matches would otherwise be silently ignored.
//
// Takes importPath (string) which identifies the package for error messages.
// Takes nativeBackedTypes (map[string][]string) which is the configured generic_types
// mapping.
// Takes matched (map[string]struct{}) which records the names that matched an extracted
// generic type.
//
// Returns an error wrapping errNotGenericType for the first unmatched name, or nil when
// every configured name matched.
func reportUnmatchedNativeTypes(importPath string, nativeBackedTypes map[string][]string, matched map[string]struct{}) error {
	leftovers := make([]string, 0, len(nativeBackedTypes))
	for name := range nativeBackedTypes {
		if _, ok := matched[name]; !ok {
			leftovers = append(leftovers, name)
		}
	}
	if len(leftovers) == 0 {
		return nil
	}
	slices.Sort(leftovers)
	return fmt.Errorf("package %s: %s: %w", importPath, leftovers[0], errNotGenericType)
}

// resolvePackageLinks collects and validates every //piko:link directive in the package.
//
// Takes goPackage (*packages.Package) which is the loaded package.
//
// Returns the validated directive map keyed by the annotated generic's name, and any
// parse or validation error wrapped with the package path for context.
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

// classifyExportedSymbol routes a scope object into the appropriate ExtractedPackage
// slot.
//
// Takes extractedPackage (*ExtractedPackage) which receives the classified entry.
// Takes typeObject (types.Object) which is the scope object under consideration.
// Takes packageName (string) which is the Go package name for the symbol.
// Takes includeGeneric (bool) which enables manifest-driven generic wrapper collection.
// Takes linksByName (map[string]LinkDirective) which maps generic function names to their
// //piko:link directives.
// Takes nativeBackedTypes (map[string][]string) which maps generic type names opted into
// the native-backed pipeline to their erased type-argument lists.
// Takes matchedNativeTypes (map[string]struct{}) which records the native-backed names
// that matched an extracted generic type.
//
// Returns an error when a native-backed generic type's erasure type-argument count does
// not match its declared type-parameter count.
func classifyExportedSymbol(
	extractedPackage *ExtractedPackage,
	typeObject types.Object,
	packageName string,
	includeGeneric bool,
	linksByName map[string]LinkDirective,
	nativeBackedTypes map[string][]string,
	matchedNativeTypes map[string]struct{},
) error {
	if link, linked := linksByName[typeObject.Name()]; linked && appendLinkedGenericIfValid(extractedPackage, typeObject, link) {
		return nil
	}

	symbol := classifyObject(typeObject, packageName, includeGeneric)
	if symbol == nil {
		return nil
	}

	if symbol.Kind == SymbolGenericFunc {
		appendGenericFuncIfValid(extractedPackage, typeObject)
		return nil
	}

	if symbol.Kind == SymbolLinkedGenericType {
		return appendLinkedGenericTypeIfValid(extractedPackage, typeObject, nativeBackedTypes, matchedNativeTypes)
	}

	extractedPackage.Symbols = append(extractedPackage.Symbols, *symbol)
	return nil
}

// appendLinkedGenericTypeIfValid re-runs classifyLinkedGenericType to capture the
// field-by-field descriptor and stores it on the ExtractedPackage. Types named in
// nativeBackedTypes are diverted to the native-backed pipeline instead: they keep only
// their type-arg count and the manifest's erased type-argument list.
//
// Takes extractedPackage (*ExtractedPackage) which receives the descriptor.
// Takes typeObject (types.Object) which is the candidate type name.
// Takes nativeBackedTypes (map[string][]string) which maps native-backed generic type
// names to their erased type-argument lists.
// Takes matchedNativeTypes (map[string]struct{}) which records the native-backed names
// matched against an extracted generic type.
//
// Returns an error when a native-backed entry's erasure type-argument count does not
// match the generic type's declared type-parameter count.
func appendLinkedGenericTypeIfValid(
	extractedPackage *ExtractedPackage,
	typeObject types.Object,
	nativeBackedTypes map[string][]string,
	matchedNativeTypes map[string]struct{},
) error {
	typeName, ok := typeObject.(*types.TypeName)
	if !ok {
		return nil
	}
	info, ok := classifyLinkedGenericType(typeName)
	if !ok {
		return nil
	}
	if erasure, isNative := nativeBackedTypes[typeName.Name()]; isNative {
		matchedNativeTypes[typeName.Name()] = struct{}{}
		if len(erasure) != info.TypeArgCount {
			return fmt.Errorf("%s: %w: got %d, want %d", info.Name, errErasureArgCount, len(erasure), info.TypeArgCount)
		}
		extractedPackage.NativeBackedGenericTypes = append(extractedPackage.NativeBackedGenericTypes, NativeBackedGenericTypeInfo{
			Name:            info.Name,
			ErasureTypeArgs: erasure,
			TypeArgCount:    info.TypeArgCount,
		})
		return nil
	}
	extractedPackage.LinkedGenericTypes = append(extractedPackage.LinkedGenericTypes, info)
	return nil
}

// appendLinkedGenericIfValid records a directive-matched generic function on
// extractedPackage.
//
// Takes extractedPackage (*ExtractedPackage) which receives the entry.
// Takes typeObject (types.Object) which is the candidate function.
// Takes link (LinkDirective) which is the matching //piko:link directive.
//
// Returns true when the symbol was consumed via the linked path; false when the object is
// not a generic function and the normal classification should continue.
func appendLinkedGenericIfValid(extractedPackage *ExtractedPackage, typeObject types.Object, link LinkDirective) bool {
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

	extractedPackage.LinkedGenericFuncs = append(extractedPackage.LinkedGenericFuncs, LinkedGenericFuncInfo{
		Name:         fn.Name(),
		LinkTarget:   link.LinkTarget,
		Params:       paramInfos,
		Results:      resultInfos,
		Variadic:     signature.Variadic(),
		TypeArgCount: signature.TypeParams().Len(),
	})
	return true
}

// appendGenericFuncIfValid records a generic function exposed to the manifest-driven
// dispatch-wrapper path.
//
// Takes extractedPackage (*ExtractedPackage) which receives the entry.
// Takes typeObject (types.Object) which is the candidate function.
func appendGenericFuncIfValid(extractedPackage *ExtractedPackage, typeObject types.Object) {
	typeFunction, ok := typeObject.(*types.Func)
	if !ok {
		return
	}
	signature, ok := typeFunction.Type().(*types.Signature)
	if !ok {
		return
	}
	extractedPackage.GenericFuncs = append(extractedPackage.GenericFuncs, GenericFuncInfo{
		Name:      typeFunction.Name(),
		Func:      typeFunction,
		Signature: signature,
	})
}

// sortExtractedPackage sorts every slice of the ExtractedPackage by symbol name for
// deterministic codegen output.
//
// Takes extractedPackage (*ExtractedPackage) whose slices are sorted in place.
func sortExtractedPackage(extractedPackage *ExtractedPackage) {
	slices.SortFunc(extractedPackage.Symbols, func(a, b ExtractedSymbol) int {
		return strings.Compare(a.Name, b.Name)
	})
	slices.SortFunc(extractedPackage.GenericFuncs, func(a, b GenericFuncInfo) int {
		return strings.Compare(a.Name, b.Name)
	})
	slices.SortFunc(extractedPackage.LinkedGenericFuncs, func(a, b LinkedGenericFuncInfo) int {
		return strings.Compare(a.Name, b.Name)
	})
	slices.SortFunc(extractedPackage.LinkedGenericTypes, func(a, b LinkedGenericTypeInfo) int {
		return strings.Compare(a.Name, b.Name)
	})
	slices.SortFunc(extractedPackage.NativeBackedGenericTypes, func(a, b NativeBackedGenericTypeInfo) int {
		return strings.Compare(a.Name, b.Name)
	})
}

// classifyObject determines the symbol kind and metadata for a types.Object, returning
// nil for objects that should be skipped.
//
// Takes typeObject (types.Object) which provides the type object to classify.
// Takes packageName (string) which specifies the package name for constant formatting.
// Takes includeGeneric (bool) which controls whether generic functions are classified.
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

// classifyFunc classifies an exported function, distinguishing generic functions when
// includeGeneric is set.
//
// Takes o (*types.Func) which provides the function object to classify.
// Takes includeGeneric (bool) which controls whether generic functions are classified.
//
// Returns a pointer to the classified symbol or nil if the function is skipped.
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

// classifyConst classifies an exported constant.
//
// Typed constants always produce a literal expression. Untyped constants are emitted only
// when their value is representable in their default type; otherwise they are skipped so
// the generated code still compiles.
//
// Takes o (*types.Const) which provides the constant object to classify.
// Takes packageName (string) which specifies the package name for value formatting.
//
// Returns a pointer to the classified symbol or nil if skipped.
func classifyConst(o *types.Const, packageName string) *ExtractedSymbol {
	constValue := o.Val()
	if constValue == nil {
		return nil
	}

	basic, ok := o.Type().(*types.Basic)
	if ok && basic.Info()&types.IsUntyped != 0 {
		if !untypedConstRepresentableInDefault(constValue, o.Type()) {
			return &ExtractedSymbol{
				Name:           o.Name(),
				Kind:           SymbolConst,
				IsUntypedConst: true,
			}
		}
	}

	constExpr := formatConstant(o, packageName)
	return &ExtractedSymbol{
		Name:       o.Name(),
		Kind:       SymbolConst,
		ConstValue: constExpr,
	}
}

// untypedConstRepresentableInDefault reports whether an untyped constant value is
// representable in its default type. An untyped constant given its default type can
// overflow that type (for example math.MaxUint64 overflows the default type int), in
// which case reflect.ValueOf of the bare reference would not compile.
//
// Takes value (constant.Value) which provides the untyped constant value to test.
// Takes constType (types.Type) which provides the constant's declared (untyped) type used
// to derive the default type.
//
// Returns true when the value fits its default type and can be safely emitted as a
// package-qualified reference.
func untypedConstRepresentableInDefault(value constant.Value, constType types.Type) bool {
	defaultType, ok := types.Default(constType).(*types.Basic)
	if !ok {
		return false
	}

	switch defaultType.Kind() {
	case types.Bool:
		return value.Kind() == constant.Bool
	case types.String:
		return value.Kind() == constant.String
	case types.Int:
		return intConstRepresentable(value)
	case types.Int32:
		return int32ConstRepresentable(value)
	case types.Float64:
		return float64ConstRepresentable(value)
	case types.Complex128:
		return complex128ConstRepresentable(value)
	default:
		return false
	}
}

// intConstRepresentable reports whether an untyped constant fits the default int type.
//
// Takes value (constant.Value) which provides the untyped constant value to test.
//
// Returns true when the value is an exact 64-bit integer.
func intConstRepresentable(value constant.Value) bool {
	intValue := constant.ToInt(value)
	if intValue.Kind() != constant.Int {
		return false
	}
	_, exact := constant.Int64Val(intValue)
	return exact
}

// int32ConstRepresentable reports whether an untyped constant fits the int32 type, used
// for the default type of untyped rune constants.
//
// Takes value (constant.Value) which provides the untyped constant value to test.
//
// Returns true when the value is an exact integer within the int32 range.
func int32ConstRepresentable(value constant.Value) bool {
	intValue := constant.ToInt(value)
	if intValue.Kind() != constant.Int {
		return false
	}
	representation, exact := constant.Int64Val(intValue)
	if !exact {
		return false
	}
	return representation >= math.MinInt32 && representation <= math.MaxInt32
}

// float64ConstRepresentable reports whether an untyped constant fits the default float64
// type.
//
// Takes value (constant.Value) which provides the untyped constant value to test.
//
// Returns true when the value is a finite float64.
func float64ConstRepresentable(value constant.Value) bool {
	floatValue := constant.ToFloat(value)
	if floatValue.Kind() != constant.Float {
		return false
	}
	representation, _ := constant.Float64Val(floatValue)
	return !math.IsInf(representation, 0)
}

// complex128ConstRepresentable reports whether an untyped constant fits the default
// complex128 type.
//
// Takes value (constant.Value) which provides the untyped constant value to test.
//
// Returns true when both the real and imaginary parts are finite.
func complex128ConstRepresentable(value constant.Value) bool {
	complexValue := constant.ToComplex(value)
	if complexValue.Kind() != constant.Complex {
		return false
	}
	realPart, _ := constant.Float64Val(constant.Real(complexValue))
	imagPart, _ := constant.Float64Val(constant.Imag(complexValue))
	return !math.IsInf(realPart, 0) && !math.IsInf(imagPart, 0)
}

// classifyTypeName classifies an exported named type, skipping generic types that cannot
// be represented (non-struct underlying, constraint interfaces), and tagging registrable
// generics with SymbolLinkedGenericType so the caller can populate LinkedGenericTypes.
//
// Takes o (*types.TypeName) which provides the type name object to classify.
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

// isGenericType returns true if the type name refers to a generic type that cannot be
// represented as reflect.Value without instantiation.
//
// Takes o (*types.TypeName) which provides the type name object to inspect.
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

// formatConstant produces a Go expression string for a typed constant's value, suitable
// for embedding in reflect.ValueOf(...).
//
// Takes c (*types.Const) which provides the constant to format.
// Takes packageName (string) which specifies the package name for qualified references.
//
// Returns the package-qualified constant reference string.
func formatConstant(c *types.Const, packageName string) string {
	return packageName + "." + c.Name()
}
