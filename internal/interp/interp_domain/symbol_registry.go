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

package interp_domain

import (
	"fmt"
	"go/types"
	"maps"
	"path"
	"reflect"
	"sync"
)

// SymbolExports maps package paths to symbol names and their reflected
// values. This mirrors the type definition in templater_domain.
type SymbolExports = map[string]map[string]reflect.Value

// maxSynthesisDepth is the maximum nesting depth for cross-package type
// synthesis via Import. Exceeding this limit indicates a circular or
// pathologically deep dependency chain and triggers a graceful fallback.
const maxSynthesisDepth = 64

// maxTypeConversionDepth is the maximum recursion depth within a single
// reflectTypeConverter.toGoType call chain. Guards against deeply nested
// or self-referential type hierarchies within a single package synthesis.
const maxTypeConversionDepth = 128

// SymbolRegistry provides thread-safe access to pre-registered native
// Go symbols. It maps import paths to package-level exports.
//
// The registry is immutable after initial setup and safe for concurrent
// reads. It is shared across interpreter clones.
type SymbolRegistry struct {
	// symbols maps "path/to/pkg" to {"FuncName": reflect.Value, ...}.
	symbols map[string]map[string]reflect.Value

	// synthesised caches types.Package objects built from reflected
	// symbols for use by the go/types Importer.
	synthesised map[string]*types.Package

	// reflectToTypes caches reflect.Type to types.Type mappings
	// across all synthesised packages.
	//
	// This preserves named type identity when the same Go type
	// appears in multiple packages (e.g. a type alias re-exported
	// from a facade package). Without this, each per-package
	// converter would create independent anonymous types for the
	// same reflect.Type, breaking Go's nominal type system.
	reflectToTypes map[reflect.Type]types.Type

	// protectedPackages contains package paths that cannot be overridden
	// via RegisterPackage. Used for built-in packages like "unsafe".
	protectedPackages map[string]bool

	// typeOwners maps reflect.Type (elem of nil-pointer registrations) to
	// the package path under which it was registered. This handles type
	// aliases where reflect.Type.PkgPath() returns the original type's
	// package rather than the facade package where the alias is exported.
	typeOwners map[reflect.Type]string

	// synthesising tracks packages currently being synthesised to prevent
	// infinite recursion when cross-package named types reference each other.
	synthesising map[string]bool

	// synthesisDepth tracks the current nesting depth of Import calls
	// triggered by cross-package type resolution. Acts as a safety net
	// to prevent stack overflow from circular or deeply nested chains.
	synthesisDepth int

	// mu guards concurrent access during initial setup.
	mu sync.RWMutex
}

// NewSymbolRegistry creates a registry from symbol exports.
//
// Takes exports (SymbolExports) which maps package paths to their
// exported symbols.
//
// Returns *SymbolRegistry which is ready for lookups.
func NewSymbolRegistry(exports SymbolExports) *SymbolRegistry {
	r := &SymbolRegistry{
		symbols:        make(map[string]map[string]reflect.Value, len(exports)),
		synthesised:    make(map[string]*types.Package),
		reflectToTypes: make(map[reflect.Type]types.Type),
		typeOwners:     make(map[reflect.Type]string),
		synthesising:   make(map[string]bool),
	}

	for packagePath, symbols := range exports {
		packageSymbols := make(map[string]reflect.Value, len(symbols))
		maps.Copy(packageSymbols, symbols)
		r.symbols[packagePath] = packageSymbols

		for _, value := range symbols {
			rt := value.Type()
			if rt.Kind() == reflect.Pointer && value.IsNil() {
				r.typeOwners[rt.Elem()] = packagePath
			}
		}
	}

	return r
}

// SynthesiseAll eagerly synthesises types.Package objects for every
// registered package that has not already been provided via
// RegisterTypesPackage. This populates the reflectToTypes cache so that
// concurrent Import calls always hit the fast path, eliminating races
// where two goroutines synthesise the same package simultaneously and
// produce inconsistent named type identities.
//
// Call this after all RegisterTypesPackage calls are complete, so that
// cross-package type resolution uses the real packages rather than
// separately synthesised ones.
func (r *SymbolRegistry) SynthesiseAll() {
	for importPath := range r.symbols {
		if _, ok := r.synthesised[importPath]; ok {
			continue
		}
		_, _ = r.Import(importPath)
	}
}

// Lookup returns the reflect.Value for a symbol in a package.
//
// Takes packagePath (string) which is the package import path.
// Takes name (string) which is the symbol name.
//
// Returns reflect.Value which is the symbol's value.
// Returns bool which is true if the symbol was found.
//
// Safe for concurrent use by multiple goroutines.
func (r *SymbolRegistry) Lookup(packagePath, name string) (reflect.Value, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	pkg, ok := r.symbols[packagePath]
	if !ok {
		return reflect.Value{}, false
	}

	value, ok := pkg[name]
	return value, ok
}

// LookupPackage returns all exports for a package.
//
// Takes packagePath (string) which is the package import path.
//
// Returns map[string]reflect.Value which contains all exported symbols.
// Returns bool which is true if the package was found.
//
// Safe for concurrent use by multiple goroutines.
func (r *SymbolRegistry) LookupPackage(packagePath string) (map[string]reflect.Value, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	pkg, ok := r.symbols[packagePath]
	return pkg, ok
}

// ZeroValueForType returns an addressable zero value for a named type
// registered via the (*T)(nil) pattern, so pointer receiver methods
// can be called on it.
//
// Takes packagePath (string) which is the package import path.
// Takes name (string) which is the type name to look up.
//
// Returns the addressable zero value and true, or an invalid value
// and false if not found.
func (r *SymbolRegistry) ZeroValueForType(packagePath, name string) (reflect.Value, bool) {
	value, ok := r.Lookup(packagePath, name)
	if !ok {
		return reflect.Value{}, false
	}
	reflectType := value.Type()
	if reflectType.Kind() != reflect.Pointer || !value.IsNil() {
		return reflect.Value{}, false
	}

	return reflect.New(reflectType.Elem()).Elem(), true
}

// HasPackage returns true if the registry contains the given package.
//
// Takes packagePath (string) which is the package import path to check.
//
// Returns true if the package is registered.
//
// Safe for concurrent use by multiple goroutines.
func (r *SymbolRegistry) HasPackage(packagePath string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, ok := r.symbols[packagePath]
	return ok
}

// AllPackages returns all registered package paths.
//
// Returns a slice of all registered package import paths.
//
// Safe for concurrent use by multiple goroutines.
func (r *SymbolRegistry) AllPackages() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	paths := make([]string, 0, len(r.symbols))
	for p := range r.symbols {
		paths = append(paths, p)
	}
	return paths
}

// RegisterPackage adds or replaces a package in the registry at runtime,
// used to register compiled interpreter packages so that other packages
// can import them via the existing Lookup and Import paths.
//
// Protected packages (e.g. "unsafe") are silently ignored.
//
// Takes packagePath (string) which is the package import path.
// Takes symbols (map[string]reflect.Value) which is the exported symbol
// map to register.
//
// Safe for concurrent use by multiple goroutines.
func (r *SymbolRegistry) RegisterPackage(packagePath string, symbols map[string]reflect.Value) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.protectedPackages[packagePath] {
		return
	}

	r.symbols[packagePath] = symbols

	for _, value := range symbols {
		rt := value.Type()
		if rt.Kind() == reflect.Pointer && value.IsNil() {
			r.typeOwners[rt.Elem()] = packagePath
		}
	}
}

// PackageSymbols returns a copy of the exported symbols for a package.
//
// Takes packagePath (string) which is the package import path.
//
// Returns map[string]reflect.Value which maps symbol names to values.
// Returns bool which is true when the package exists.
//
// Safe for concurrent use by multiple goroutines.
func (r *SymbolRegistry) PackageSymbols(packagePath string) (map[string]reflect.Value, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	symbols, ok := r.symbols[packagePath]
	return symbols, ok
}

// ReflectTypeForNamed returns the real Go reflect.Type for a named type
// that was registered via the (*T)(nil) pattern. This allows the
// bytecode compiler to use the actual Go type identity instead of
// synthesising anonymous struct types.
//
// Takes pkgPath (string) which is the package import path.
// Takes typeName (string) which is the type name to look up.
//
// Returns reflect.Type which is the real Go type.
// Returns bool which is true if the type was found.
//
// Safe for concurrent use by multiple goroutines.
func (r *SymbolRegistry) ReflectTypeForNamed(pkgPath, typeName string) (reflect.Type, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	pkg, ok := r.symbols[pkgPath]
	if !ok {
		return nil, false
	}

	value, ok := pkg[typeName]
	if !ok {
		return nil, false
	}

	reflectType := value.Type()
	if reflectType.Kind() == reflect.Pointer && value.IsNil() {
		return reflectType.Elem(), true
	}

	return nil, false
}

// ProtectPackage marks a package as protected, preventing it from being
// overridden via RegisterPackage.
//
// Takes packagePath (string) which is the package import path to protect.
//
// Safe for concurrent use by multiple goroutines.
func (r *SymbolRegistry) ProtectPackage(packagePath string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.protectedPackages == nil {
		r.protectedPackages = make(map[string]bool)
	}
	r.protectedPackages[packagePath] = true
}

// RegisterTypesPackage caches a pre-built *types.Package so that Import
// returns it directly instead of synthesising from reflect values.
//
// Used when the package was type-checked from source and we want to preserve
// the exact type information for downstream importers.
//
// Takes packagePath (string) which is the package import path.
// Takes pkg (*types.Package) which is the pre-built types package to
// cache.
//
// Safe for concurrent use by multiple goroutines.
func (r *SymbolRegistry) RegisterTypesPackage(packagePath string, pkg *types.Package) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.synthesised[packagePath] = pkg

	r.remapReflectTypesToPackage(packagePath, pkg)
}

// Import implements types.Importer by synthesising a types.Package from
// the reflected symbol values registered for the given import path.
//
// Takes importPath (string) which is the package import path to resolve.
//
// Returns the synthesised types package, or an error if not found.
//
// Safe for concurrent use by multiple goroutines.
func (r *SymbolRegistry) Import(importPath string) (*types.Package, error) {
	if importPath == pkgUnsafe {
		return types.Unsafe, nil
	}

	r.mu.RLock()
	if pkg, ok := r.synthesised[importPath]; ok {
		r.mu.RUnlock()
		return pkg, nil
	}
	r.mu.RUnlock()

	exports, ok := r.LookupPackage(importPath)
	if !ok {
		return nil, fmt.Errorf("package %q not found in symbol registry", importPath)
	}

	r.mu.Lock()

	if pkg, ok := r.synthesised[importPath]; ok {
		r.mu.Unlock()
		return pkg, nil
	}
	if r.synthesisDepth >= maxSynthesisDepth {
		r.mu.Unlock()
		return nil, fmt.Errorf(
			"symbol registry: synthesis depth %d exceeded for package %q - "+
				"this indicates a circular or deeply nested type dependency chain "+
				"across registered packages; check piko-symbols.yaml for "+
				"unnecessary package registrations",
			r.synthesisDepth, importPath,
		)
	}
	r.synthesisDepth++
	r.synthesising[importPath] = true
	r.mu.Unlock()

	pkg := r.synthesisePackage(importPath, exports)

	r.mu.Lock()
	r.synthesisDepth--
	delete(r.synthesising, importPath)
	r.synthesised[importPath] = pkg
	r.mu.Unlock()

	return pkg, nil
}

// remapReflectTypesToPackage updates reflectToTypes entries for
// nil-pointer type registrations in the given package.
//
// The entries are repointed to the named types from the provided
// types.Package instead of from a previously synthesised package.
// Must be called with r.mu held.
//
// Takes packagePath (string) which is the package import path.
// Takes pkg (*types.Package) which is the replacement package
// whose named types should be used.
func (r *SymbolRegistry) remapReflectTypesToPackage(packagePath string, pkg *types.Package) {
	exports, ok := r.symbols[packagePath]
	if !ok {
		return
	}

	scope := pkg.Scope()
	for name, value := range exports {
		reflectType := value.Type()
		if reflectType.Kind() != reflect.Pointer || !value.IsNil() {
			continue
		}

		typeObj := scope.Lookup(name)
		if typeObj == nil {
			continue
		}

		tn, ok := typeObj.(*types.TypeName)
		if !ok {
			continue
		}

		elemRT := reflectType.Elem()
		r.reflectToTypes[elemRT] = tn.Type()
		r.reflectToTypes[reflectType] = types.NewPointer(tn.Type())
	}
}

// synthesisePackage builds a types.Package from reflected symbol values
// using a two-pass approach for correct named type resolution.
//
// The first pass registers (*T)(nil) type patterns so that subsequent
// function signatures (e.g. func() *T) resolve to the named type rather
// than an anonymous struct.
//
// Takes importPath (string) which is the package import path.
// Takes exports (map[string]reflect.Value) which maps symbol names to
// their reflected values.
//
// Returns the synthesised types package.
func (r *SymbolRegistry) synthesisePackage(importPath string, exports map[string]reflect.Value) *types.Package {
	packageName := path.Base(importPath)
	pkg := types.NewPackage(importPath, packageName)

	converter := &reflectTypeConverter{
		seen:       make(map[reflect.Type]types.Type),
		pkg:        pkg,
		registry:   r,
		localTypes: make(map[reflect.Type]bool),
	}

	r.registerNamedTypes(pkg, exports, converter)
	r.registerFunctionsAndVariables(pkg, exports, converter)

	pkg.MarkComplete()
	return pkg
}

// pendingNamedType holds a named type whose underlying type has not
// yet been resolved because it may reference other types in the same
// package.
type pendingNamedType struct {
	// elemRT is the element reflect.Type (the T in *T).
	elemRT reflect.Type

	// ptrRT is the pointer reflect.Type (*T).
	ptrRT reflect.Type

	// named is the forward-declared go/types named type.
	named *types.Named
}

// registerNamedTypes scans exports for nil-pointer type registrations,
// creates forward-declared named types, then resolves their underlying
// types and methods.
//
// Takes pkg (*types.Package) which is the target package.
// Takes exports (map[string]reflect.Value) which maps symbol names
// to their reflected values.
// Takes converter (*reflectTypeConverter) which performs the type
// conversion.
//
// Safe for concurrent use; acquires r.mu internally as needed.
func (r *SymbolRegistry) registerNamedTypes(
	pkg *types.Package,
	exports map[string]reflect.Value,
	converter *reflectTypeConverter,
) {
	scope := pkg.Scope()
	var pending []pendingNamedType

	for name, value := range exports {
		reflectType := value.Type()
		if reflectType.Kind() != reflect.Pointer || !value.IsNil() {
			continue
		}

		elemRT := reflectType.Elem()
		typeName := types.NewTypeName(0, pkg, name, nil)
		named := types.NewNamed(typeName, nil, nil)

		converter.seen[elemRT] = named
		ptrNamed := types.NewPointer(named)
		converter.seen[reflectType] = ptrNamed

		converter.localTypes[elemRT] = true
		converter.localTypes[reflectType] = true

		r.mu.Lock()
		r.reflectToTypes[elemRT] = named
		r.reflectToTypes[reflectType] = ptrNamed
		r.mu.Unlock()

		scope.Insert(typeName)
		pending = append(pending, pendingNamedType{elemRT: elemRT, ptrRT: reflectType, named: named})
	}

	for _, p := range pending {
		underlying := converter.synthesiseNamedUnderlying(p.elemRT)
		p.named.SetUnderlying(underlying)
		converter.synthesiseMethods(p.ptrRT, p.named, pkg)
	}
}

// synthesiseNamedUnderlying computes the underlying types.Type for a
// registered named type. It dispatches by reflect.Kind and skips the
// seen-cache short-circuit so the placeholder *types.Named stays
// available for recursive field and method references.
//
// Takes reflectType (reflect.Type) which is the element reflect.Type
// (the T in *T) of the registered named type.
//
// Returns the synthesised underlying types.Type (struct, interface,
// signature, slice, etc.).
func (c *reflectTypeConverter) synthesiseNamedUnderlying(reflectType reflect.Type) types.Type {
	if basicKind, ok := reflectKindToBasicType[reflectType.Kind()]; ok {
		return types.Typ[basicKind]
	}
	return c.convertCompositeType(reflectType)
}

// registerFunctionsAndVariables inserts exported functions and
// variables into the package scope, skipping nil-pointer type
// registrations that were already handled by registerNamedTypes.
//
// Takes pkg (*types.Package) which is the target package.
// Takes exports (map[string]reflect.Value) which maps symbol names
// to their reflected values.
// Takes converter (*reflectTypeConverter) which performs the type
// conversion.
func (*SymbolRegistry) registerFunctionsAndVariables(
	pkg *types.Package,
	exports map[string]reflect.Value,
	converter *reflectTypeConverter,
) {
	scope := pkg.Scope()

	for name, value := range exports {
		reflectType := value.Type()

		switch {
		case reflectType.Kind() == reflect.Pointer && value.IsNil():
			continue

		case reflectType.Kind() == reflect.Func:
			signature := converter.funcSignature(reflectType)
			typeObject := types.NewFunc(0, pkg, name, signature)
			scope.Insert(typeObject)

		default:
			goType := converter.toGoType(reflectType)
			typeObject := types.NewVar(0, pkg, name, goType)
			scope.Insert(typeObject)
		}
	}
}

// reflectTypeConverter converts reflect.Type to types.Type, handling
// recursive types via a cache to break cycles.
type reflectTypeConverter struct {
	// seen caches previously converted types to break recursive cycles.
	seen map[reflect.Type]types.Type

	// pkg is the target package for named type declarations.
	pkg *types.Package

	// registry is a back-reference to the owning SymbolRegistry, used to
	// resolve named types from foreign packages during synthesis so that
	// cross-package type identity is preserved.
	registry *SymbolRegistry

	// localTypes contains reflect.Types being defined as named types in the
	// current package (from (*T)(nil) exports). These must not be resolved
	// as foreign types even when reflect.PkgPath() differs from the
	// synthesised package path (which happens for re-exported types).
	localTypes map[reflect.Type]bool

	// depth tracks the current recursion depth of toGoType calls within
	// this converter, guarding against pathologically deep type hierarchies.
	depth int
}

// synthesiseMethods adds exported methods from ptrType's method set to
// the named type.
//
// The pointer method set includes all value receiver methods, so this single
// pass covers everything. Methods that also appear on the value type's
// method set are registered with a value receiver; pointer-only methods
// use a pointer receiver. This distinction matters for the type checker
// when calling methods on non-addressable values.
//
// Takes ptrType (reflect.Type) which is the pointer type whose methods
// to scan.
// Takes named (*types.Named) which is the target named type.
// Takes pkg (*types.Package) which is the package for method
// declarations.
func (c *reflectTypeConverter) synthesiseMethods(
	ptrType reflect.Type,
	named *types.Named,
	pkg *types.Package,
) {
	elemType := ptrType.Elem()
	valueMethodCount := elemType.NumMethod()
	valueMethodSet := make(map[string]bool, valueMethodCount)
	for valueMethod := range elemType.Methods() {
		valueMethodSet[valueMethod.Name] = true
	}

	for m := range ptrType.Methods() {
		if !m.IsExported() {
			continue
		}

		mt := m.Type
		numIn := mt.NumIn() - 1
		parameters := make([]*types.Var, numIn)
		for j := range numIn {
			parameters[j] = types.NewParam(0, nil, "", c.toGoType(mt.In(j+1)))
		}
		numOut := mt.NumOut()
		results := make([]*types.Var, numOut)
		for j := range numOut {
			results[j] = types.NewParam(0, nil, "", c.toGoType(mt.Out(j)))
		}

		var receiver types.Type = named
		if !valueMethodSet[m.Name] {
			receiver = types.NewPointer(named)
		}

		signature := types.NewSignatureType(
			types.NewParam(0, pkg, "", receiver),
			nil, nil,
			types.NewTuple(parameters...),
			types.NewTuple(results...),
			mt.IsVariadic(),
		)
		named.AddMethod(types.NewFunc(0, pkg, m.Name, signature))
	}
}

// reflectKindToBasicType maps reflect.Kind values for primitive types to
// their corresponding go/types basic type. Compound kinds (Slice, Map,
// etc.) are not included, since they require recursive conversion.
var reflectKindToBasicType = map[reflect.Kind]types.BasicKind{
	reflect.Bool:          types.Bool,
	reflect.Int:           types.Int,
	reflect.Int8:          types.Int8,
	reflect.Int16:         types.Int16,
	reflect.Int32:         types.Int32,
	reflect.Int64:         types.Int64,
	reflect.Uint:          types.Uint,
	reflect.Uint8:         types.Uint8,
	reflect.Uint16:        types.Uint16,
	reflect.Uint32:        types.Uint32,
	reflect.Uint64:        types.Uint64,
	reflect.Uintptr:       types.Uintptr,
	reflect.Float32:       types.Float32,
	reflect.Float64:       types.Float64,
	reflect.Complex64:     types.Complex64,
	reflect.Complex128:    types.Complex128,
	reflect.String:        types.String,
	reflect.UnsafePointer: types.UnsafePointer,
}

// reflectChanDirToTypes maps a reflect.ChanDir to the corresponding
// types.ChanDir constant.
//
// Takes direction (reflect.ChanDir) which is the channel direction
// to convert.
//
// Returns the equivalent go/types channel direction.
func reflectChanDirToTypes(direction reflect.ChanDir) types.ChanDir {
	switch direction {
	case reflect.SendDir:
		return types.SendOnly
	case reflect.RecvDir:
		return types.RecvOnly
	default:
		return types.SendRecv
	}
}

// toGoType converts a reflect.Type to the corresponding types.Type.
//
// Takes reflectType (reflect.Type) which is the reflect type to convert.
//
// Returns the equivalent go/types representation.
func (c *reflectTypeConverter) toGoType(reflectType reflect.Type) types.Type {
	if cached, ok := c.seen[reflectType]; ok {
		return cached
	}

	c.depth++
	defer func() { c.depth-- }()
	if c.depth > maxTypeConversionDepth {
		return types.NewInterfaceType(nil, nil)
	}

	if resolved := c.resolveFromRegistry(reflectType); resolved != nil {
		return resolved
	}

	if basicKind, ok := reflectKindToBasicType[reflectType.Kind()]; ok {
		return types.Typ[basicKind]
	}

	return c.convertCompositeType(reflectType)
}

// resolveFromRegistry checks the registry cache and foreign type
// resolution for a previously seen or registered type.
//
// Takes reflectType (reflect.Type) which is the type to look up.
//
// Returns the cached types.Type, or nil if not found.
//
// Safe for concurrent use; acquires registry.mu internally.
func (c *reflectTypeConverter) resolveFromRegistry(reflectType reflect.Type) types.Type {
	if c.registry == nil || c.localTypes[reflectType] {
		return nil
	}
	c.registry.mu.RLock()
	cached, ok := c.registry.reflectToTypes[reflectType]
	c.registry.mu.RUnlock()
	if ok {
		c.seen[reflectType] = cached
		return cached
	}
	if resolved := c.resolveForeignNamedType(reflectType); resolved != nil {
		c.seen[reflectType] = resolved
		return resolved
	}
	return nil
}

// convertCompositeType handles conversion of composite reflect
// types (slices, maps, structs, interfaces, etc.) to go/types.
//
// Takes reflectType (reflect.Type) which is the composite type
// to convert.
//
// Returns the equivalent go/types representation.
func (c *reflectTypeConverter) convertCompositeType(reflectType reflect.Type) types.Type {
	switch reflectType.Kind() {
	case reflect.Slice:
		if reflectType.Elem().Kind() == reflect.Uint8 {
			return types.NewSlice(types.Typ[types.Byte])
		}
		return types.NewSlice(c.toGoType(reflectType.Elem()))
	case reflect.Array:
		return types.NewArray(c.toGoType(reflectType.Elem()), int64(reflectType.Len()))
	case reflect.Map:
		return types.NewMap(c.toGoType(reflectType.Key()), c.toGoType(reflectType.Elem()))
	case reflect.Pointer:
		return types.NewPointer(c.toGoType(reflectType.Elem()))
	case reflect.Chan:
		return types.NewChan(reflectChanDirToTypes(reflectType.ChanDir()), c.toGoType(reflectType.Elem()))
	case reflect.Func:
		return c.funcSignature(reflectType)
	case reflect.Struct:
		return c.structType(reflectType)
	case reflect.Interface:
		return c.interfaceType(reflectType)
	default:
		return types.NewInterfaceType(nil, nil)
	}
}

// interfaceType converts a reflect interface type to a go/types
// interface, using a placeholder to handle recursive types.
//
// Takes reflectType (reflect.Type) which is the interface type
// to convert.
//
// Returns the equivalent go/types interface.
func (c *reflectTypeConverter) interfaceType(reflectType reflect.Type) types.Type {
	_, hasNamedPlaceholder := c.seen[reflectType].(*types.Named)
	var placeholder types.Type
	if !hasNamedPlaceholder {
		placeholder = types.NewInterfaceType(nil, nil)
		c.seen[reflectType] = placeholder
	}

	var methods []*types.Func
	for m := range reflectType.Methods() {
		signature := c.funcSignature(m.Type)
		methods = append(methods, types.NewFunc(0, nil, m.Name, signature))
	}
	if len(methods) > 0 {
		iface := types.NewInterfaceType(methods, nil)
		iface.Complete()
		if !hasNamedPlaceholder {
			c.seen[reflectType] = iface
		}
		return iface
	}

	if hasNamedPlaceholder {
		return types.NewInterfaceType(nil, nil)
	}
	return placeholder
}

// funcSignature converts a reflect function type to types.Signature.
//
// Takes reflectType (reflect.Type) which is the function type to convert.
//
// Returns the equivalent go/types function signature.
func (c *reflectTypeConverter) funcSignature(reflectType reflect.Type) *types.Signature {
	var parameters []*types.Var
	for in := range reflectType.Ins() {
		parameters = append(parameters, types.NewParam(0, nil, "", c.toGoType(in)))
	}

	var results []*types.Var
	for out := range reflectType.Outs() {
		results = append(results, types.NewParam(0, nil, "", c.toGoType(out)))
	}

	return types.NewSignatureType(
		nil,
		nil, nil,
		types.NewTuple(parameters...),
		types.NewTuple(results...),
		reflectType.IsVariadic(),
	)
}

// structType converts a reflect struct type to types.Struct.
//
// Takes reflectType (reflect.Type) which is the struct type to convert.
//
// Returns the equivalent go/types struct type.
func (c *reflectTypeConverter) structType(reflectType reflect.Type) types.Type {
	_, hasNamedPlaceholder := c.seen[reflectType].(*types.Named)
	if !hasNamedPlaceholder {
		placeholder := types.NewStruct(nil, nil)
		c.seen[reflectType] = placeholder
	}

	var fields []*types.Var
	var tags []string
	for f := range reflectType.Fields() {
		var fieldPkg *types.Package
		if !f.IsExported() && f.PkgPath != "" {
			fieldPkg = types.NewPackage(f.PkgPath, path.Base(f.PkgPath))
		}
		fields = append(fields, types.NewField(0, fieldPkg, f.Name, c.toGoType(f.Type), f.Anonymous))
		tags = append(tags, string(f.Tag))
	}

	result := types.NewStruct(fields, tags)
	if !hasNamedPlaceholder {
		c.seen[reflectType] = result
	}
	return result
}

// resolveForeignNamedType resolves a named type from a foreign
// registered package that has not yet been synthesised.
//
// This handles cross-package named type resolution: when package
// A's struct has a field of type B.SomeType, and package B is
// registered but not yet synthesised, the B.SomeType entry will
// be missing from reflectToTypes. Triggering B's synthesis first
// ensures the named type (with its methods) is used instead of
// the anonymous underlying type.
//
// Takes reflectType (reflect.Type) which is the type to resolve.
//
// Returns the resolved types.Type, or nil if the type cannot be
// resolved this way.
//
// Safe for concurrent use; acquires registry.mu internally.
func (c *reflectTypeConverter) resolveForeignNamedType(reflectType reflect.Type) types.Type {
	foreignPackagePath := reflectType.PkgPath()
	if foreignPackagePath == "" || reflectType.Name() == "" {
		return nil
	}

	if foreignPackagePath == c.pkg.Path() {
		return nil
	}

	if !c.registry.HasPackage(foreignPackagePath) {
		c.registry.mu.RLock()
		ownerPath, ok := c.registry.typeOwners[reflectType]
		c.registry.mu.RUnlock()
		if !ok || ownerPath == c.pkg.Path() {
			return nil
		}
		foreignPackagePath = ownerPath
	}

	c.registry.mu.RLock()
	inProgress := c.registry.synthesising[foreignPackagePath]
	c.registry.mu.RUnlock()
	if inProgress {
		return nil
	}

	foreignPackage, _ := c.registry.Import(foreignPackagePath)

	c.registry.mu.RLock()
	resolved, ok := c.registry.reflectToTypes[reflectType]
	c.registry.mu.RUnlock()
	if ok {
		return resolved
	}

	if foreignPackage != nil {
		typeObject := foreignPackage.Scope().Lookup(reflectType.Name())
		if typeObject != nil {
			if typeName, isTypeName := typeObject.(*types.TypeName); isTypeName {
				return typeName.Type()
			}
		}
	}

	return nil
}
