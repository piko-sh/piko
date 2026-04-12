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
	"go/types"
	"reflect"
	"strconv"

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/interp/interp_link"
)

const (
	// maxLinkedDescriptorDepth caps how deeply the synthesiser will recurse into nested
	// Element/Key/TypeArgs positions of a GenericFieldType descriptor. Reached only by
	// pathological or hostile generated code; the cap prevents stack exhaustion without
	// blocking any realistic Go type.
	maxLinkedDescriptorDepth = 64

	// maxLinkedTypeArgCount caps how many type parameters a linked generic may declare. Go's
	// type system supports any count, but real APIs rarely exceed three; clamping at 32
	// means a malformed or hostile descriptor cannot drive unbounded allocation in
	// makeLinkedTypeParams or resolveLinkedNamedGeneric.
	maxLinkedTypeArgCount = 32

	// logFieldPackage is the structured-log field key carrying a symbol's owning package
	// path.
	logFieldPackage = "package"

	// logFieldName is the structured-log field key carrying a symbol's source-level name.
	logFieldName = "name"
)

var (
	// linkedGenericTypeReflectType is cached once so the synthesis path can type-assert
	// against it without rebuilding the descriptor per symbol.
	linkedGenericTypeReflectType = reflect.TypeFor[interp_link.LinkedGenericType]()

	// nativeBackedGenericTypeReflectType is cached for the native-backed generic synthesis
	// path's type assertion.
	nativeBackedGenericTypeReflectType = reflect.TypeFor[interp_link.NativeBackedGenericType]()
)

// registerNativeBackedGenericTypes installs sentinel-backed generic types.
//
// Scans exports for interp_link.NativeBackedGenericType sentinels and installs a generic
// *types.Named for each, carrying the type's REAL native method set (synthesised from the
// sentinel's canonical erased reflect.Type). Unlike registerLinkedGenericTypes (which
// builds a structural reflect.StructOf lookalike with no methods), the path is for
// generic stdlib types whose state is unexported (atomic.Pointer) and which therefore
// cannot be reproduced structurally.
//
// Takes pkg (*types.Package) which is the synthesised package.
// Takes exports (map[string]reflect.Value) which is the export table.
// Takes converter (*reflectTypeConverter) which converts the erased method signatures'
// non-parametric types.
func (*SymbolRegistry) registerNativeBackedGenericTypes(
	pkg *types.Package,
	exports map[string]reflect.Value,
	converter *reflectTypeConverter,
) {
	scope := pkg.Scope()
	for name, value := range exports {
		if value.Type() != nativeBackedGenericTypeReflectType {
			continue
		}
		native, ok := reflect.TypeAssert[interp_link.NativeBackedGenericType](value)
		if !ok || native.TypeArgCount <= 0 || native.TypeArgCount > maxLinkedTypeArgCount ||
			native.ErasedType == nil || len(native.ErasureArgs) != native.TypeArgCount {
			log.Warn("Skipping malformed NativeBackedGenericType sentinel",
				logger_domain.String(logFieldPackage, pkg.Path()),
				logger_domain.String(logFieldName, name),
				logger_domain.Int("type_arg_count", native.TypeArgCount),
				logger_domain.Int("erasure_arg_count", len(native.ErasureArgs)))
			continue
		}
		if native.ErasedType.Kind() != reflect.Struct {
			log.Warn("Skipping NativeBackedGenericType sentinel with non-struct erased type",
				logger_domain.String(logFieldPackage, pkg.Path()),
				logger_domain.String(logFieldName, name))
			continue
		}
		if scope.Lookup(name) != nil {
			log.Warn("Skipping already-registered name",
				logger_domain.String(logFieldPackage, pkg.Path()),
				logger_domain.String(logFieldName, name))
			continue
		}
		typeName := types.NewTypeName(0, pkg, name, nil)
		named := types.NewNamed(typeName, nil, nil)
		typeParams := makeLinkedTypeParams(pkg, native.TypeArgCount)
		named.SetTypeParams(typeParams)
		named.SetUnderlying(types.NewStruct(nil, nil))
		converter.synthesiseNativeBackedMethods(
			reflect.PointerTo(native.ErasedType), named, pkg, len(typeParams), native.ErasureArgs)
		scope.Insert(typeName)
	}
}

// registerLinkedGenericTypes scans the exports map for interp_link.LinkedGenericType
// sentinels and installs a generic *types.Named for each one in pkg's scope. Field types
// reference *types.TypeParam placeholders so go/types can instantiate the generic when
// user code writes pkg.Name[Concrete]; Underlying() on the instantiated named then
// returns a struct with the concrete type arguments substituted.
//
// Takes pkg (*types.Package) which is the synthesised package.
// Takes exports (map[string]reflect.Value) which is the export table being synthesised.
func (r *SymbolRegistry) registerLinkedGenericTypes(
	pkg *types.Package,
	exports map[string]reflect.Value,
) {
	scope := pkg.Scope()
	for name, value := range exports {
		if value.Type() != linkedGenericTypeReflectType {
			continue
		}
		linked, ok := reflect.TypeAssert[interp_link.LinkedGenericType](value)
		if !ok || linked.TypeArgCount <= 0 {
			log.Warn("Skipping malformed LinkedGenericType sentinel",
				logger_domain.String(logFieldPackage, pkg.Path()),
				logger_domain.String(logFieldName, name),
				logger_domain.Int("type_arg_count", linked.TypeArgCount))
			continue
		}
		if linked.TypeArgCount > maxLinkedTypeArgCount {
			log.Warn("Skipping generic with too many type parameters",
				logger_domain.String(logFieldPackage, pkg.Path()),
				logger_domain.String(logFieldName, name),
				logger_domain.Int("type_arg_count", linked.TypeArgCount),
				logger_domain.Int("limit", maxLinkedTypeArgCount))
			continue
		}
		if scope.Lookup(name) != nil {
			log.Warn("Skipping already-registered name",
				logger_domain.String(logFieldPackage, pkg.Path()),
				logger_domain.String(logFieldName, name))
			continue
		}
		typeName := types.NewTypeName(0, pkg, name, nil)
		named := types.NewNamed(typeName, nil, nil)
		typeParams := makeLinkedTypeParams(pkg, linked.TypeArgCount)
		named.SetTypeParams(typeParams)

		structType := buildLinkedStruct(r, linked, typeParams)
		named.SetUnderlying(structType)

		scope.Insert(typeName)
	}
}

// resolveNamedForLinkedField looks up a non-generic named type that a linked generic's
// field references.
//
// The registry's own reflectToTypes cache is authoritative; the lookup falls back to nil
// when the type is not yet synthesised.
//
// Takes packagePath (string) which is the referenced import path.
// Takes typeName (string) which is the referenced identifier.
//
// Returns the resolved types.Type or nil on failure.
//
// Concurrency: takes the registry's read lock; safe for concurrent readers, and does not
// hold the lock across any blocking call.
func (r *SymbolRegistry) resolveNamedForLinkedField(packagePath, typeName string) types.Type {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if pkg, ok := r.synthesised[packagePath]; ok {
		if obj := pkg.Scope().Lookup(typeName); obj != nil {
			return obj.Type()
		}
	}
	return nil
}

// IsLinkedGenericType reports whether the registry has a linked generic type sentinel for
// the given package and type name.
//
// The compiler uses this hint to skip the per-type sentinel reflect field when
// synthesising struct types, because linked generics rely on structural identity of the
// reflect.Type rather than a baked-in package/type-name tag. Without the skip, a sibling
// that builds a SearchResult[Post] via its own reflect.StructOf call would produce a
// structurally identical but nominally distinct reflect.Type from the one the interpreter
// synthesises at the call site.
//
// Takes packagePath (string) which is the owning import path.
// Takes typeName (string) which is the generic's exported name.
//
// Returns true when a LinkedGenericType sentinel is registered.
//
// Safe for concurrent use by multiple goroutines.
func (r *SymbolRegistry) IsLinkedGenericType(packagePath, typeName string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	exports, ok := r.symbols[packagePath]
	if !ok {
		return false
	}
	value, ok := exports[typeName]
	if !ok || !value.IsValid() {
		return false
	}
	return value.Type() == linkedGenericTypeReflectType
}

// NativeBackedReflectType returns the canonical erased reflect.Type registered for a
// native-backed generic type: the real, method-bearing type the compiler resolves every
// instantiation to.
//
// Takes packagePath (string) which is the owning import path.
// Takes typeName (string) which is the generic's exported name.
//
// Returns the erased reflect.Type and true, or nil and false when no native-backed
// sentinel is registered.
//
// Safe for concurrent use by multiple goroutines.
func (r *SymbolRegistry) NativeBackedReflectType(packagePath, typeName string) (reflect.Type, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	exports, ok := r.symbols[packagePath]
	if !ok {
		return nil, false
	}
	value, ok := exports[typeName]
	if !ok || !value.IsValid() || value.Type() != nativeBackedGenericTypeReflectType {
		return nil, false
	}
	native, ok := reflect.TypeAssert[interp_link.NativeBackedGenericType](value)
	if !ok {
		return nil, false
	}
	return native.ErasedType, true
}

// makeLinkedTypeParams produces a []*types.TypeParam for a linked generic's TypeArgCount
// declaration. The constraint is the empty interface (the `any` constraint) because the
// interpreter never enforces custom type constraints and accepting `any` matches every
// call site a user is likely to write.
//
// Takes pkg (*types.Package) which owns the synthesised type names.
// Takes count (int) which is the declared TypeArgCount.
//
// Returns the ordered []*types.TypeParam.
func makeLinkedTypeParams(pkg *types.Package, count int) []*types.TypeParam {
	if count <= 0 {
		return nil
	}
	count = min(count, maxLinkedTypeArgCount)
	parameters := make([]*types.TypeParam, count)
	anyInterface := types.NewInterfaceType(nil, nil)
	for index := range count {
		parameterName := "T"
		if index > 0 {
			parameterName = "T" + strconv.Itoa(index+1)
		}
		typeName := types.NewTypeName(0, pkg, parameterName, nil)
		parameters[index] = types.NewTypeParam(typeName, anyInterface)
	}
	return parameters
}

// buildLinkedStruct converts a LinkedGenericType sentinel into a go/types.Struct whose
// field types mix concrete types with the supplied TypeParam placeholders. Only exported
// fields are emitted; unexported fields would collide with the package-local visibility
// rules of the synthesised package.
//
// Takes registry (*SymbolRegistry) which resolves named non-generic cross-package
// references.
// Takes linked (interp_link.LinkedGenericType) which is the sentinel.
// Takes typeParams ([]*types.TypeParam) which are the declaration- order type parameters
// already attached to the owning *types.Named.
//
// Returns the *types.Struct ready to be set as the Named's underlying.
func buildLinkedStruct(
	registry *SymbolRegistry,
	linked interp_link.LinkedGenericType,
	typeParams []*types.TypeParam,
) *types.Struct {
	var (
		fields []*types.Var
		tags   []string
	)
	for i := range linked.Fields {
		field := &linked.Fields[i]
		if !field.Exported {
			continue
		}
		fieldType := linkedFieldToType(registry, field.FieldType, typeParams, 0)
		if fieldType == nil {
			fieldType = types.NewInterfaceType(nil, nil)
		}
		fields = append(fields, types.NewField(0, nil, field.Name, fieldType, false))
		tags = append(tags, field.Tag)
	}
	return types.NewStruct(fields, tags)
}

// linkedFieldToType recursively converts a GenericFieldType descriptor into a
// go/types.Type, substituting type-arg references with the supplied TypeParam
// placeholders. Recursion is bounded by maxLinkedDescriptorDepth.
//
// Takes registry (*SymbolRegistry) which resolves cross-package named type references.
// Takes descriptor (interp_link.GenericFieldType) which is the node.
// Takes typeParams ([]*types.TypeParam) which are the owning generic's type parameters in
// declaration order.
// Takes depth (int) which tracks the current recursion depth.
//
// Returns the corresponding go/types.Type or nil when the descriptor cannot be resolved
// (caller substitutes any in that case).
func linkedFieldToType(
	registry *SymbolRegistry,
	descriptor interp_link.GenericFieldType,
	typeParams []*types.TypeParam,
	depth int,
) types.Type {
	if depth >= maxLinkedDescriptorDepth {
		return nil
	}

	switch descriptor.Kind {
	case interp_link.FieldKindBasic:
		return basicKindToType(descriptor.BasicKind)
	case interp_link.FieldKindTypeArg:
		return linkedTypeArgToType(descriptor, typeParams)
	case interp_link.FieldKindSlice, interp_link.FieldKindPointer, interp_link.FieldKindChan:
		return linkedSingleElementToType(registry, descriptor, typeParams, depth+1)
	case interp_link.FieldKindArray:
		return linkedArrayToType(registry, descriptor, typeParams, depth+1)
	case interp_link.FieldKindMap:
		return linkedMapToType(registry, descriptor, typeParams, depth+1)
	case interp_link.FieldKindInterface, interp_link.FieldKindNamedGeneric:
		return types.NewInterfaceType(nil, nil)
	case interp_link.FieldKindNamed:
		return linkedNamedToType(registry, descriptor)
	case interp_link.FieldKindError:
		return types.Universe.Lookup("error").Type()
	}
	return nil
}

// linkedTypeArgToType resolves a FieldKindTypeArg descriptor to the owning generic's
// declaration-order TypeParam.
//
// Takes descriptor (interp_link.GenericFieldType) which is the node.
// Takes typeParams ([]*types.TypeParam) which are the owning generic's declared
// parameters.
//
// Returns the matched TypeParam, or nil when the index is out of range.
func linkedTypeArgToType(
	descriptor interp_link.GenericFieldType,
	typeParams []*types.TypeParam,
) types.Type {
	if descriptor.TypeArgIndex < 0 || descriptor.TypeArgIndex >= len(typeParams) {
		return nil
	}
	return typeParams[descriptor.TypeArgIndex]
}

// linkedSingleElementToType builds a slice, pointer, or chan type from the descriptor's
// Element.
//
// Takes registry (*SymbolRegistry) which resolves Named references.
// Takes descriptor (interp_link.GenericFieldType) which is the node.
// Takes typeParams ([]*types.TypeParam) which resolve type parameter references.
// Takes depth (int) which is the already-incremented recursion depth.
//
// Returns the composite type, or nil for an unsupported kind or missing element.
func linkedSingleElementToType(
	registry *SymbolRegistry,
	descriptor interp_link.GenericFieldType,
	typeParams []*types.TypeParam,
	depth int,
) types.Type {
	if descriptor.Element == nil {
		return nil
	}
	elementType := linkedFieldToType(registry, *descriptor.Element, typeParams, depth)
	switch descriptor.Kind {
	case interp_link.FieldKindSlice:
		return types.NewSlice(elementType)
	case interp_link.FieldKindPointer:
		return types.NewPointer(elementType)
	case interp_link.FieldKindChan:
		return types.NewChan(types.SendRecv, elementType)
	default:
	}
	return nil
}

// linkedArrayToType builds a fixed-length array from the descriptor.
//
// Takes registry (*SymbolRegistry) which resolves Named references.
// Takes descriptor (interp_link.GenericFieldType) which is the node.
// Takes typeParams ([]*types.TypeParam) which resolve type parameter references.
// Takes depth (int) which is the already-incremented recursion depth.
//
// Returns the array type, or nil when Element is missing.
func linkedArrayToType(
	registry *SymbolRegistry,
	descriptor interp_link.GenericFieldType,
	typeParams []*types.TypeParam,
	depth int,
) types.Type {
	if descriptor.Element == nil {
		return nil
	}
	return types.NewArray(
		linkedFieldToType(registry, *descriptor.Element, typeParams, depth),
		int64(descriptor.ArrayLength),
	)
}

// linkedMapToType builds a map type from the descriptor.
//
// Takes registry (*SymbolRegistry) which resolves Named references.
// Takes descriptor (interp_link.GenericFieldType) which is the node.
// Takes typeParams ([]*types.TypeParam) which resolve type parameter references.
// Takes depth (int) which is the already-incremented recursion depth.
//
// Returns the map type, or nil when Key or Element is missing.
func linkedMapToType(
	registry *SymbolRegistry,
	descriptor interp_link.GenericFieldType,
	typeParams []*types.TypeParam,
	depth int,
) types.Type {
	if descriptor.Key == nil || descriptor.Element == nil {
		return nil
	}
	return types.NewMap(
		linkedFieldToType(registry, *descriptor.Key, typeParams, depth),
		linkedFieldToType(registry, *descriptor.Element, typeParams, depth),
	)
}

// linkedNamedToType resolves a non-generic named reference through the registry.
//
// Takes registry (*SymbolRegistry) which holds synthesised packages.
// Takes descriptor (interp_link.GenericFieldType) which carries NamedPackage and
// NamedName.
//
// Returns the resolved type; the empty interface when the target is unknown; nil when the
// descriptor is empty.
func linkedNamedToType(
	registry *SymbolRegistry,
	descriptor interp_link.GenericFieldType,
) types.Type {
	if descriptor.NamedPackage == "" || descriptor.NamedName == "" {
		return nil
	}
	if resolved := registry.resolveNamedForLinkedField(descriptor.NamedPackage, descriptor.NamedName); resolved != nil {
		return resolved
	}
	return types.NewInterfaceType(nil, nil)
}

// basicKindToType maps a reflect.Kind to its go/types.Basic equivalent so linked field
// descriptors can reference primitive types without serialising the go/types enum.
//
// Takes kind (reflect.Kind) which is the primitive kind to translate.
//
// Returns the matching *types.Basic, or the empty-interface fallback when the kind has no
// direct go/types counterpart.
func basicKindToType(kind reflect.Kind) types.Type {
	switch kind {
	case reflect.Bool:
		return types.Typ[types.Bool]
	case reflect.Int:
		return types.Typ[types.Int]
	case reflect.Int8:
		return types.Typ[types.Int8]
	case reflect.Int16:
		return types.Typ[types.Int16]
	case reflect.Int32:
		return types.Typ[types.Int32]
	case reflect.Int64:
		return types.Typ[types.Int64]
	case reflect.Uint:
		return types.Typ[types.Uint]
	case reflect.Uint8:
		return types.Typ[types.Uint8]
	case reflect.Uint16:
		return types.Typ[types.Uint16]
	case reflect.Uint32:
		return types.Typ[types.Uint32]
	case reflect.Uint64:
		return types.Typ[types.Uint64]
	case reflect.Uintptr:
		return types.Typ[types.Uintptr]
	case reflect.Float32:
		return types.Typ[types.Float32]
	case reflect.Float64:
		return types.Typ[types.Float64]
	case reflect.Complex64:
		return types.Typ[types.Complex64]
	case reflect.Complex128:
		return types.Typ[types.Complex128]
	case reflect.String:
		return types.Typ[types.String]
	default:
	}
	return types.NewInterfaceType(nil, nil)
}
