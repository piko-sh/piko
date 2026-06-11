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

const (
	// maxSynthesisDepth is the maximum nesting depth for cross-package type synthesis via
	// Import. Exceeding this limit indicates a circular or pathologically deep dependency
	// chain and triggers a graceful fallback.
	maxSynthesisDepth = 64

	// maxTypeConversionDepth is the maximum recursion depth within a single
	// reflectTypeConverter.toGoType call chain. Guards against deeply nested or
	// self-referential type hierarchies within a single package synthesis.
	maxTypeConversionDepth = 128
)

var (
	// reflectKindToBasicType maps primitive reflect kinds to basic types.
	//
	// Compound kinds (Slice, Map, etc.) are not included, since they require recursive
	// conversion.
	//nolint:exhaustive // compound kinds intentionally absent; resolved via recursion.
	reflectKindToBasicType = map[reflect.Kind]types.BasicKind{
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
)

// SymbolExports is the per-package export table accepted by NewSymbolRegistry. The outer
// key is an import path; the inner map pairs exported symbol names with their
// reflect.Values (functions, variables, or typed-nil pointers acting as type carriers).
type SymbolExports = map[string]map[string]reflect.Value

// SymbolRegistry holds the host-provided package symbols visible to compiled bytecode. It
// exposes lookup, scoping, and lazy synthesis of go/types representations so the compiler
// can type-check references to native symbols without re-importing their packages.
type SymbolRegistry struct {
	// symbols maps "path/to/pkg" to {"FuncName": reflect.Value, ...}.
	symbols map[string]map[string]reflect.Value

	// synthesised caches types.Package objects built from reflected symbols for use by the
	// go/types Importer.
	synthesised map[string]*types.Package

	// reflectToTypes caches reflect.Type to types.Type mappings across all synthesised
	// packages.
	//
	// This preserves named type identity when the same Go type appears in multiple packages
	// (e.g. a type alias re-exported from a facade package). Without this, each per-package
	// converter would create independent anonymous types for the same reflect.Type, breaking
	// Go's nominal type system.
	reflectToTypes map[reflect.Type]types.Type

	// protectedPackages contains package paths that cannot be overridden via
	// RegisterPackage. Used for built-in packages like "unsafe".
	protectedPackages map[string]bool

	// typeOwners maps reflect.Type (elem of nil-pointer registrations) to the package path
	// under which it was registered. This handles type aliases where reflect.Type.PkgPath()
	// returns the original type's package rather than the facade package where the alias is
	// exported.
	typeOwners map[reflect.Type]string

	// synthesising tracks packages currently being synthesised to prevent infinite recursion
	// when cross-package named types reference each other.
	synthesising map[string]bool

	// synthesisDepth tracks the current nesting depth of Import calls triggered by
	// cross-package type resolution. Acts as a safety net to prevent stack overflow from
	// circular or deeply nested chains.
	synthesisDepth int

	// mu guards concurrent access during initial setup.
	mu sync.RWMutex
}

// NewSymbolRegistry constructs a SymbolRegistry seeded with the supplied per-package
// exports. Typed-nil pointer symbols are recorded as the canonical owner of their element
// reflect.Type so reflect-based dispatch can attribute methods to their declaring
// package.
//
// Takes exports (SymbolExports) which maps import path -> name -> value.
//
// Returns a fully initialised registry ready for Lookup, Scoped, or SynthesiseAll calls.
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

// SynthesiseAll eagerly synthesises go/types.Package shells for every registered import
// path, populating the synthesised cache so later type-checking against host symbols has
// no per-call cost.
func (r *SymbolRegistry) SynthesiseAll() {
	for importPath := range r.symbols {
		if _, ok := r.synthesised[importPath]; ok {
			continue
		}
		_, _ = r.Import(importPath)
	}
}

// Lookup returns the reflect.Value of name within packagePath.
//
// Both the package and the symbol must be registered; returns the zero Value and false
// otherwise.
//
// Takes packagePath (string) which is the import path.
// Takes name (string) which is the symbol name to look up.
//
// Returns reflect.Value which wraps the symbol value on hit.
// Returns bool which is true when the symbol is registered.
//
// Concurrency: acquires the registry read lock for the duration of the call.
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

// LookupPackage returns the full symbol map for packagePath.
//
// Takes packagePath (string) which is the import path.
//
// Returns map[string]reflect.Value which is the symbol map on hit, nil otherwise.
// Returns bool which is true when the package is registered.
//
// Concurrency: acquires the registry read lock for the duration of the call.
func (r *SymbolRegistry) LookupPackage(packagePath string) (map[string]reflect.Value, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	pkg, ok := r.symbols[packagePath]
	return pkg, ok
}

// ZeroValueForType returns the zero Value for a typed-nil pointer symbol's element type.
//
// Yields a zero Value and false when the symbol is not a typed-nil pointer.
//
// Takes packagePath (string) which is the import path.
// Takes name (string) which is the symbol name.
//
// Returns reflect.Value which wraps a fresh zero Value of the element type.
// Returns bool which is true when the symbol is a typed-nil pointer.
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

// HasPackage reports whether the registry has any symbols registered under packagePath.
//
// Takes packagePath (string) which is the import path.
//
// Returns bool which is true when at least one symbol is registered.
//
// Concurrency: acquires the registry read lock for the duration of the call.
func (r *SymbolRegistry) HasPackage(packagePath string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, ok := r.symbols[packagePath]
	return ok
}

// AllPackages returns the import paths of every package currently registered, in
// unspecified order.
//
// Returns []string which is the list of registered import paths.
//
// Concurrency: acquires the registry read lock for the duration of the call.
func (r *SymbolRegistry) AllPackages() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	paths := make([]string, 0, len(r.symbols))
	for p := range r.symbols {
		paths = append(paths, p)
	}
	return paths
}

// RegisterPackage installs or replaces the symbol map for packagePath.
//
// No-op when the package has been protected via ProtectPackage; the owner side-table for
// typed-nil pointers is refreshed otherwise.
//
// Takes packagePath (string) which is the import path.
// Takes symbols (map[string]reflect.Value) which provides the new symbol map.
//
// Concurrency: acquires the registry write lock for the duration of the call.
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

// Scoped returns a derived registry filtered by allowlist.
//
// Exposes only the import paths and symbol names matching allowlist. Entries are either
// "pkg" for a whole-package wildcard or "pkg.Name" for a single symbol. A nil allowlist
// returns the receiver unchanged.
//
// Takes allowlist ([]string) which carries the allowlist entries.
//
// Returns *SymbolRegistry which is either a fresh derived registry or the receiver itself
// when allowlist is nil.
//
// Concurrency: acquires the receiver's read lock for the duration of the call.
func (r *SymbolRegistry) Scoped(allowlist []string) *SymbolRegistry {
	if allowlist == nil {
		return r
	}
	r.mu.RLock()
	defer r.mu.RUnlock()

	packageWildcards, symbolEntries := classifyAllowlistEntries(allowlist)

	scoped := &SymbolRegistry{
		symbols:           make(map[string]map[string]reflect.Value),
		synthesised:       r.synthesised,
		reflectToTypes:    r.reflectToTypes,
		typeOwners:        r.typeOwners,
		synthesising:      make(map[string]bool),
		protectedPackages: r.protectedPackages,
	}
	for packagePath, packageSymbols := range r.symbols {
		_, wildcardOK := packageWildcards[packagePath]
		nameSet, nameSetOK := symbolEntries[packagePath]
		if !wildcardOK && !nameSetOK {
			continue
		}
		if wildcardOK {
			copyPkg := make(map[string]reflect.Value, len(packageSymbols))
			maps.Copy(copyPkg, packageSymbols)
			scoped.symbols[packagePath] = copyPkg
			continue
		}
		scoped.symbols[packagePath] = filterPackageSymbols(packageSymbols, nameSet)
	}
	return scoped
}

// OverlayPackage merges overlay into the symbols of packagePath.
//
// Replaces existing entries on collision. No-op when the package is protected via
// ProtectPackage; the package is created when absent.
//
// Takes packagePath (string) which is the import path.
// Takes overlay (map[string]reflect.Value) which contains the symbols to merge in.
//
// Concurrency: acquires the registry write lock for the duration of the call.
func (r *SymbolRegistry) OverlayPackage(packagePath string, overlay map[string]reflect.Value) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.protectedPackages[packagePath] {
		return
	}
	existing, ok := r.symbols[packagePath]
	if !ok {
		merged := make(map[string]reflect.Value, len(overlay))
		maps.Copy(merged, overlay)
		r.symbols[packagePath] = merged
		return
	}
	maps.Copy(existing, overlay)
}

// PackageSymbols returns the live symbol map for packagePath.
//
// The returned map must not be mutated by callers.
//
// Takes packagePath (string) which is the import path.
//
// Returns map[string]reflect.Value which is the live symbol map.
// Returns bool which is true when the package is registered.
//
// Concurrency: acquires the registry read lock for the duration of the call.
func (r *SymbolRegistry) PackageSymbols(packagePath string) (map[string]reflect.Value, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	symbols, ok := r.symbols[packagePath]
	return symbols, ok
}

// ReflectTypeForNamed returns the typed-nil pointer's element type.
//
// Yields nil and false when the symbol is missing or is not a typed-nil pointer.
//
// Takes pkgPath (string) which is the import path.
// Takes typeName (string) which is the type symbol name.
//
// Returns reflect.Type which is the named element type on hit.
// Returns bool which is true when the symbol is a typed-nil pointer.
//
// Concurrency: acquires the registry read lock for the duration of the call.
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

// ReflectTypeForNamedByPackageName resolves a named type whose qualifier is a package
// short name rather than a full import path.
//
// Takes packageName (string) which is the package short name from the static type string.
// Takes typeName (string) which is the bare type symbol name.
//
// Returns reflect.Type which is the named element type on a hit, or nil on a miss.
// Returns bool which is true when a matching named type was found.
//
// Concurrency: acquires the registry read lock for the duration of the call.
func (r *SymbolRegistry) ReflectTypeForNamedByPackageName(packageName, typeName string) (reflect.Type, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for packagePath, pkg := range r.symbols {
		if path.Base(packagePath) != packageName {
			continue
		}
		value, ok := pkg[typeName]
		if !ok {
			continue
		}
		reflectType := value.Type()
		if reflectType.Kind() == reflect.Pointer && value.IsNil() {
			return reflectType.Elem(), true
		}
	}

	return nil, false
}

// ProtectPackage marks packagePath as immutable.
//
// Subsequent RegisterPackage and OverlayPackage calls become no-ops. Used to lock down
// host-provided packages (e.g. unsafe) after initial registration.
//
// Takes packagePath (string) which is the import path to protect.
//
// Concurrency: acquires the registry write lock for the duration of the call.
func (r *SymbolRegistry) ProtectPackage(packagePath string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.protectedPackages == nil {
		r.protectedPackages = make(map[string]bool)
	}
	r.protectedPackages[packagePath] = true
}

// RegisterTypesPackage installs a pre-synthesised go/types.Package.
//
// Refreshes the reflectToTypes side-table so existing reflect.Types resolve to the new
// go/types objects.
//
// Takes packagePath (string) which is the import path.
// Takes pkg (*types.Package) which is the pre-synthesised package.
//
// Concurrency: acquires the registry write lock for the duration of the call.
func (r *SymbolRegistry) RegisterTypesPackage(packagePath string, pkg *types.Package) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.synthesised[packagePath] = pkg

	r.remapReflectTypesToPackage(packagePath, pkg)
}

// SnapshotTypesPackages returns a shallow copy of all currently registered
// go/types.Packages, keyed by import path. Used by the module-load path to seed
// gcexportdata's "imports" cache so a freshly-decoded package's references to
// previously-registered packages resolve to the same *types.Package instances.
//
// The returned map is owned by the caller and safe to mutate; the *types.Package values
// must be treated as read-only - they are shared with the registry.
//
// Returns map[string]*types.Package which is the snapshot map.
//
// Concurrency: acquires the registry read lock for the duration of the call.
func (r *SymbolRegistry) SnapshotTypesPackages() map[string]*types.Package {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make(map[string]*types.Package, len(r.synthesised))
	maps.Copy(out, r.synthesised)
	return out
}

// Import returns the go/types.Package for importPath.
//
// Synthesises one from the registered reflect.Values on first call and caches the result.
// The "unsafe" package is special-cased to types.Unsafe.
//
// Takes importPath (string) which is the import path to resolve.
//
// Returns *types.Package which is the synthesised or cached package.
// Returns error when importPath is not registered or synthesis fails.
//
// Concurrency: acquires the registry write lock briefly to publish the synthesised
// package; reads use the read lock.
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
		return nil, fmt.Errorf("%w: %q", errPackageNotInRegistry, importPath)
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

	defer func() {
		r.mu.Lock()
		r.synthesisDepth--
		delete(r.synthesising, importPath)
		r.mu.Unlock()
	}()

	pkg := r.synthesisePackage(importPath, exports)

	r.mu.Lock()
	r.synthesised[importPath] = pkg
	r.mu.Unlock()

	return pkg, nil
}

// remapReflectTypesToPackage refreshes reflectToTypes entries so the package's named
// types resolve through the supplied package.
//
// Takes packagePath (string) which is the import path.
// Takes pkg (*types.Package) which provides the new scope.
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

		elementReflectType := reflectType.Elem()
		r.reflectToTypes[elementReflectType] = tn.Type()
		r.reflectToTypes[reflectType] = types.NewPointer(tn.Type())
	}
}

// synthesisePackage builds a go/types.Package from reflect exports.
//
// Takes importPath (string) which is the synthesised package path.
// Takes exports (map[string]reflect.Value) which contains the registered reflect symbols.
//
// Returns *types.Package which is the completed synthesised package.
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
	r.registerLinkedGenericTypes(pkg, exports)
	r.registerNativeBackedGenericTypes(pkg, exports, converter)
	r.registerFunctionsAndVariables(pkg, exports, converter)

	pkg.MarkComplete()
	return pkg
}

// pendingNamedType holds a named type whose underlying type has not yet been resolved
// because it may reference other types in the same package.
type pendingNamedType struct {
	// elementReflectType is the element reflect.Type (the T in *T).
	elementReflectType reflect.Type

	// ptrRT is the pointer reflect.Type (*T).
	ptrRT reflect.Type

	// named is the forward-declared go/types named type.
	named *types.Named
}

// registerNamedTypes installs named type declarations for exports.
//
// Takes pkg (*types.Package) which receives the named-type declarations.
// Takes exports (map[string]reflect.Value) which provides the typed-nil pointer
// registrations.
// Takes converter (*reflectTypeConverter) which lowers reflect underlying types to
// go/types.
//
// Concurrency: takes r.mu in write mode while populating the reflect-to-types index for
// each new named type.
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

		elementReflectType := reflectType.Elem()
		typeName := types.NewTypeName(0, pkg, name, nil)
		named := types.NewNamed(typeName, nil, nil)

		converter.seen[elementReflectType] = named
		ptrNamed := types.NewPointer(named)
		converter.seen[reflectType] = ptrNamed

		converter.localTypes[elementReflectType] = true
		converter.localTypes[reflectType] = true

		r.mu.Lock()
		r.reflectToTypes[elementReflectType] = named
		r.reflectToTypes[reflectType] = ptrNamed
		r.mu.Unlock()

		scope.Insert(typeName)
		pending = append(pending, pendingNamedType{elementReflectType: elementReflectType, ptrRT: reflectType, named: named})
	}

	for _, p := range pending {
		underlying := converter.synthesiseNamedUnderlying(p.elementReflectType)
		p.named.SetUnderlying(underlying)
		converter.synthesiseMethods(p.ptrRT, p.named, pkg)
	}
}

// registerFunctionsAndVariables installs functions and var declarations for non-typed-nil
// exports.
//
// Takes pkg (*types.Package) which receives the declarations.
// Takes exports (map[string]reflect.Value) which provides the reflect-side function and
// variable values.
// Takes converter (*reflectTypeConverter) which lowers reflect signatures to go/types.
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

		case reflectType == linkedGenericTypeReflectType:

			continue

		case reflectType == linkedFunctionReflectType:
			typeObject := converter.linkedGenericFunc(pkg, name, value)
			scope.Insert(typeObject)

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

// classifyAllowlistEntries splits an allowlist into the set of whole-package wildcard
// imports and the per-package symbol-name sets. Empty package paths are dropped.
//
// Takes allowlist ([]string) which carries the entries to classify.
//
// Returns map[string]struct{} which is the set of wildcard packages.
// Returns map[string]map[string]struct{} which is the per-package symbol-name set.
func classifyAllowlistEntries(allowlist []string) (map[string]struct{}, map[string]map[string]struct{}) {
	packageWildcards := make(map[string]struct{})
	symbolEntries := make(map[string]map[string]struct{})
	for _, entry := range allowlist {
		packagePath, name := parseAllowlistEntry(entry)
		if packagePath == "" {
			continue
		}
		if name == "" || name == "*" {
			packageWildcards[packagePath] = struct{}{}
			continue
		}
		set, ok := symbolEntries[packagePath]
		if !ok {
			set = make(map[string]struct{})
			symbolEntries[packagePath] = set
		}
		set[name] = struct{}{}
	}
	return packageWildcards, symbolEntries
}

// filterPackageSymbols returns a new map containing only the entries of packageSymbols
// whose name appears in nameSet.
//
// Takes packageSymbols (map[string]reflect.Value) which is the source symbol map.
// Takes nameSet (map[string]struct{}) which is the allowed-name set.
//
// Returns map[string]reflect.Value which is the filtered map.
func filterPackageSymbols(packageSymbols map[string]reflect.Value, nameSet map[string]struct{}) map[string]reflect.Value {
	filteredPkg := make(map[string]reflect.Value, len(nameSet))
	for name := range nameSet {
		if value, ok := packageSymbols[name]; ok {
			filteredPkg[name] = value
		}
	}
	return filteredPkg
}

// parseAllowlistEntry splits a Scoped allowlist entry into the import path and
// symbol-name components. A trailing "/*" denotes a whole- package wildcard; otherwise
// the last dot after the final slash delimits the import path from the symbol name.
//
// Takes entry (string) which is the allowlist entry to parse.
//
// Returns packagePath (string) which is the import path component.
// Returns symbolName (string) which is the symbol name or "*" for wildcards.
func parseAllowlistEntry(entry string) (packagePath, symbolName string) {
	if entry == "" {
		return "", ""
	}
	if len(entry) >= 2 && entry[len(entry)-2:] == "/*" {
		return entry[:len(entry)-2], "*"
	}
	lastSlash := -1
	for i := len(entry) - 1; i >= 0; i-- {
		if entry[i] == '/' {
			lastSlash = i
			break
		}
	}
	dotIndex := -1
	for i := len(entry) - 1; i > lastSlash; i-- {
		if entry[i] == '.' {
			dotIndex = i
			break
		}
	}
	if dotIndex < 0 {
		return entry, ""
	}
	return entry[:dotIndex], entry[dotIndex+1:]
}
