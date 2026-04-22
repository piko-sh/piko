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
	"context"
	"go/types"
	"log/slog"
	"reflect"
	"unsafe"

	"piko.sh/piko/internal/logger/logger_domain"
)

// typeConverterState bundles the state carried through a single
// typeToReflect call chain: registry lookups, the active cycle-
// detection set, and an optional shared cache keyed by types.Type.
//
// The cache is critical for mutually recursive named types: without it
// each top-level conversion synthesises a fresh reflect.Type for the
// same named type with its cycle edges baked in differently, so assignments
// between values originating from different conversion entry points
// fail with a type mismatch.
type typeConverterState struct {
	// symbols resolves pre-registered native types via the SymbolRegistry.
	// May be nil when conversion runs outside a registered-symbol context.
	symbols *SymbolRegistry

	// processing tracks the named types currently being synthesised
	// along the active recursion path. Used for cycle detection so
	// self-referential struct fields collapse to reflect.TypeFor[any]().
	processing map[types.Type]bool

	// cache stores the synthesised reflect.Type for previously seen
	// go/types.Type keys. Sharing this across calls keeps mutually
	// recursive named types identity-stable between entry points.
	cache map[types.Type]reflect.Type
}

// namedStructToReflect converts a named struct type to a reflect.Type,
// adding a zero-size sentinel field that encodes the type identity.
// This ensures that different named types with the same underlying
// struct (e.g., two empty structs) produce distinct reflect.Types,
// which is necessary for method dispatch via TypeNames.
//
// Takes ctx (context.Context) which carries the logger.
// Takes named (*types.Named) which is the named struct type to
// convert.
// Takes symbols (*SymbolRegistry) which is used to resolve pre-registered
// native types to their real Go reflect.Type.
//
// Returns a reflect.Type with a sentinel field encoding the type identity.
func namedStructToReflect(ctx context.Context, named *types.Named, symbols *SymbolRegistry) reflect.Type {
	state := &typeConverterState{
		symbols:    symbols,
		processing: make(map[types.Type]bool),
		cache:      make(map[types.Type]reflect.Type),
	}
	return convertNamedStruct(ctx, state, named)
}

// typeToReflectCached converts a go/types.Type to a reflect.Type using
// an optional shared cache for nominal identity across calls.
//
// Takes ctx (context.Context) which carries the logger.
// Takes t (types.Type) which is the go/types.Type to convert.
// Takes symbols (*SymbolRegistry) which resolves pre-registered types.
// Takes cache (map[types.Type]reflect.Type) which is the shared
// cross-call cache. May be nil.
//
// Returns the corresponding reflect.Type.
func typeToReflectCached(ctx context.Context, t types.Type, symbols *SymbolRegistry, cache map[types.Type]reflect.Type) reflect.Type {
	state := &typeConverterState{
		symbols:    symbols,
		processing: make(map[types.Type]bool),
		cache:      cache,
	}
	return convertType(ctx, state, t)
}

// convertNamedStruct converts a named struct type to a reflect.Type
// with cycle detection.
//
// Takes ctx (context.Context) which carries the logger.
// Takes state (*typeConverterState) which carries symbols, cycle
// tracking, and optional cache.
// Takes named (*types.Named) which is the named struct type to convert.
//
// Returns a reflect.Type with a sentinel field encoding the type identity.
func convertNamedStruct(ctx context.Context, state *typeConverterState, named *types.Named) reflect.Type {
	if rt := cachedReflectType(state, named); rt != nil {
		return rt
	}
	st, ok := named.Underlying().(*types.Struct)
	if !ok {
		return convertType(ctx, state, named)
	}

	if state.processing[named] {
		logTypeCycle(ctx, named, state.processing)
		return reflect.TypeFor[any]()
	}
	state.processing[named] = true
	defer delete(state.processing, named)

	fields := buildStructFields(ctx, state, st)
	if !isLinkedGenericNamed(state, named) {
		fields = append(fields, sentinelField(named, st))
	}

	result := reflect.StructOf(fields)
	storeReflectType(state, named, result)
	return result
}

// isLinkedGenericNamed reports whether the named type is backed by a
// linked generic registration.
//
// The converter skips the sentinel field that otherwise distinguishes
// structurally identical user types. Linked generics rely on reflect's
// structural type identity so sibling functions building instances
// through reflect.StructOf converge on the same reflect.Type as the
// interpreter's own conversion.
//
// Takes state (*typeConverterState) which carries the symbol registry.
// Takes named (*types.Named) which is the type under conversion.
//
// Returns true when the named type has a linked generic registration.
func isLinkedGenericNamed(state *typeConverterState, named *types.Named) bool {
	if state.symbols == nil {
		return false
	}
	obj := named.Obj()
	if obj == nil || obj.Pkg() == nil {
		return false
	}
	return state.symbols.IsLinkedGenericType(obj.Pkg().Path(), obj.Name())
}

// logTypeCycle logs a warning when a cycle is detected during type
// conversion.
//
// Takes ctx (context.Context) which carries the logger.
// Takes named (*types.Named) which is the named type that
// triggered the cycle.
func logTypeCycle(ctx context.Context, named *types.Named, processing map[types.Type]bool) {
	typePath := named.Obj().Name()
	if named.Obj().Pkg() != nil {
		typePath = named.Obj().Pkg().Path() + "." + named.Obj().Name()
	}

	var chain []string
	for processingType := range processing {
		if processingNamed, ok := processingType.(*types.Named); ok {
			entryPath := processingNamed.Obj().Name()
			if processingNamed.Obj().Pkg() != nil {
				entryPath = processingNamed.Obj().Pkg().Path() + "." + processingNamed.Obj().Name()
			}
			chain = append(chain, entryPath)
		}
	}

	_, l := logger_domain.From(ctx, log)
	l.Warn("[INTERP] Type cycle detected in compiler type conversion",
		slog.String("type", typePath),
		slog.Any("activeChain", chain),
	)
}

// buildStructFields converts each field of a types.Struct into a
// reflect.StructField slice.
//
// Takes ctx (context.Context) which carries the logger.
// Takes state (*typeConverterState) which carries symbols, cycle
// tracking, and cache.
// Takes st (*types.Struct) which is the struct type whose fields
// are converted.
//
// Returns a slice of reflect.StructField for the struct.
func buildStructFields(ctx context.Context, state *typeConverterState, st *types.Struct) []reflect.StructField {
	fields := make([]reflect.StructField, st.NumFields())
	for i := range st.NumFields() {
		f := st.Field(i)
		var fieldType reflect.Type
		if fieldTypeInvolvesCycle(f.Type(), state.processing) {
			fieldType = reflect.TypeFor[any]()
		} else {
			fieldType = convertType(ctx, state, f.Type())
		}
		fields[i] = reflect.StructField{
			Name:      f.Name(),
			Type:      fieldType,
			Tag:       reflect.StructTag(st.Tag(i)),
			Anonymous: f.Embedded(),
		}
		if !f.Exported() && f.Pkg() != nil {
			fields[i].PkgPath = f.Pkg().Path()
		}
	}
	return fields
}

// sentinelField creates the zero-size sentinel field that encodes
// type identity for the named struct.
//
// Takes named (*types.Named) which is the named type whose
// identity is encoded.
// Takes st (*types.Struct) which is the underlying struct used
// to determine the package path.
//
// Returns a reflect.StructField encoding the type identity.
func sentinelField(named *types.Named, st *types.Struct) reflect.StructField {
	sentinelPackagePath := ""
	if named.Obj().Pkg() != nil {
		sentinelPackagePath = named.Obj().Pkg().Path() + "." + named.Obj().Name()
	}
	for f := range st.Fields() {
		if !f.Exported() && f.Pkg() != nil {
			sentinelPackagePath = f.Pkg().Path()
			break
		}
	}
	return reflect.StructField{
		Name:    "_pikoID_" + named.Obj().Name(),
		Type:    reflect.TypeFor[struct{}](),
		PkgPath: sentinelPackagePath,
	}
}

// convertType converts a go/types.Type to a reflect.Type, resolving
// named types via the symbol registry and handling cycle detection.
//
// Takes ctx (context.Context) which carries the logger.
// Takes state (*typeConverterState) which carries symbols, cycle
// tracking, and cache.
// Takes t (types.Type) which is the type to convert.
//
// Returns the corresponding reflect.Type.
func convertType(ctx context.Context, state *typeConverterState, t types.Type) reflect.Type {
	if alias, ok := t.(*types.Alias); ok {
		if rt := resolveRegisteredType(alias.Obj(), state.symbols); rt != nil {
			return rt
		}
		return convertType(ctx, state, types.Unalias(t))
	}

	if named, ok := t.(*types.Named); ok {
		if rt := resolveRegisteredType(named.Obj(), state.symbols); rt != nil {
			return rt
		}
		if _, isStruct := named.Underlying().(*types.Struct); isStruct {
			return convertNamedStruct(ctx, state, named)
		}
	}

	return convertUnderlying(ctx, state, t.Underlying())
}

// resolveRegisteredType checks whether a named or aliased type
// object has a pre-registered reflect.Type in the symbol registry.
//
// Takes obj (*types.TypeName) which is the type name to look up.
// Takes symbols (*SymbolRegistry) which is the registry to search.
//
// Returns the registered reflect.Type, or nil if not found.
func resolveRegisteredType(obj *types.TypeName, symbols *SymbolRegistry) reflect.Type {
	if symbols == nil || obj.Pkg() == nil {
		return nil
	}
	rt, found := symbols.ReflectTypeForNamed(obj.Pkg().Path(), obj.Name())
	if !found {
		return nil
	}
	return rt
}

// convertUnderlying converts the underlying form of a go/types.Type
// to a reflect.Type by dispatching on the concrete type.
//
// Takes ctx (context.Context) which carries the logger.
// Takes state (*typeConverterState) which carries symbols, cycle
// tracking, and cache.
// Takes underlying (types.Type) which is the underlying type to convert.
//
// Returns the corresponding reflect.Type.
func convertUnderlying(ctx context.Context, state *typeConverterState, underlying types.Type) reflect.Type {
	switch typ := underlying.(type) {
	case *types.Basic:
		return basicToReflect(typ.Kind())
	case *types.Slice:
		return reflect.SliceOf(convertType(ctx, state, typ.Elem()))
	case *types.Map:
		return reflect.MapOf(convertType(ctx, state, typ.Key()), convertType(ctx, state, typ.Elem()))
	case *types.Pointer:
		return reflect.PointerTo(convertType(ctx, state, typ.Elem()))
	case *types.Array:
		return reflect.ArrayOf(int(typ.Len()), convertType(ctx, state, typ.Elem()))
	case *types.Struct:
		return convertStruct(ctx, state, typ)
	case *types.Signature:
		return convertSignature(ctx, state, typ)
	case *types.Chan:
		return convertChan(ctx, state, typ)
	case *types.Interface:
		return reflect.TypeFor[any]()
	default:
		return reflect.TypeFor[any]()
	}
}

// convertStruct converts a go/types anonymous struct to reflect.Type.
//
// Takes ctx (context.Context) which carries the logger.
// Takes state (*typeConverterState) which carries symbols, cycle
// tracking, and cache.
// Takes typ (*types.Struct) which is the struct type to convert.
//
// Returns the corresponding reflect.Type.
func convertStruct(ctx context.Context, state *typeConverterState, typ *types.Struct) reflect.Type {
	fields := make([]reflect.StructField, typ.NumFields())
	for i := range typ.NumFields() {
		f := typ.Field(i)
		fields[i] = reflect.StructField{
			Name: f.Name(),
			Type: convertType(ctx, state, f.Type()),
			Tag:  reflect.StructTag(typ.Tag(i)),
		}
		if !f.Exported() && f.Pkg() != nil {
			fields[i].PkgPath = f.Pkg().Path()
		}
	}
	return reflect.StructOf(fields)
}

// convertSignature converts a go/types function signature to
// reflect.Type.
//
// Takes ctx (context.Context) which carries the logger.
// Takes state (*typeConverterState) which carries symbols, cycle
// tracking, and cache.
// Takes typ (*types.Signature) which is the function signature to convert.
//
// Returns the corresponding reflect.FuncOf type.
func convertSignature(ctx context.Context, state *typeConverterState, typ *types.Signature) reflect.Type {
	var paramTypes []reflect.Type
	for v := range typ.Params().Variables() {
		paramTypes = append(paramTypes, convertType(ctx, state, v.Type()))
	}
	var resultTypes []reflect.Type
	for v := range typ.Results().Variables() {
		resultTypes = append(resultTypes, convertType(ctx, state, v.Type()))
	}
	return reflect.FuncOf(paramTypes, resultTypes, typ.Variadic())
}

// convertChan converts a go/types channel type to reflect.Type.
//
// Takes ctx (context.Context) which carries the logger.
// Takes state (*typeConverterState) which carries symbols, cycle
// tracking, and cache.
// Takes typ (*types.Chan) which is the channel type to convert.
//
// Returns the corresponding reflect.ChanOf type.
func convertChan(ctx context.Context, state *typeConverterState, typ *types.Chan) reflect.Type {
	elemType := convertType(ctx, state, typ.Elem())
	var directory reflect.ChanDir
	switch typ.Dir() {
	case types.SendRecv:
		directory = reflect.BothDir
	case types.SendOnly:
		directory = reflect.SendDir
	case types.RecvOnly:
		directory = reflect.RecvDir
	}
	return reflect.ChanOf(directory, elemType)
}

// fieldTypeInvolvesCycle checks whether a struct field's type
// references any type currently being constructed.
//
// Takes fieldType (types.Type) which is the field's declared type.
// Takes processing (map[types.Type]bool) which tracks types being
// constructed.
//
// Returns true if the field type references a type in the processing
// set.
func fieldTypeInvolvesCycle(fieldType types.Type, processing map[types.Type]bool) bool {
	switch t := fieldType.(type) {
	case *types.Named:
		return processing[t]
	case *types.Pointer:
		return fieldTypeInvolvesCycle(t.Elem(), processing)
	case *types.Slice:
		return fieldTypeInvolvesCycle(t.Elem(), processing)
	case *types.Array:
		return fieldTypeInvolvesCycle(t.Elem(), processing)
	case *types.Map:
		return fieldTypeInvolvesCycle(t.Key(), processing) || fieldTypeInvolvesCycle(t.Elem(), processing)
	case *types.Chan:
		return fieldTypeInvolvesCycle(t.Elem(), processing)
	case *types.Alias:
		return fieldTypeInvolvesCycle(types.Unalias(t), processing)
	default:
		return false
	}
}

// cachedReflectType returns a cached reflect.Type for the given
// go/types.Type, or nil if no cache is configured or no entry exists.
//
// Takes state (*typeConverterState) which carries the optional cache.
// Takes t (types.Type) which is the key to look up.
//
// Returns the cached reflect.Type, or nil when absent.
func cachedReflectType(state *typeConverterState, t types.Type) reflect.Type {
	if state.cache == nil {
		return nil
	}
	return state.cache[t]
}

// storeReflectType records a reflect.Type in the shared cache when
// one is configured. This runs after full synthesis so subsequent
// conversions of the same go/types.Type yield an identical
// reflect.Type, preserving nominal identity for recursive and mutually
// recursive named structs.
//
// Takes state (*typeConverterState) which carries the optional cache.
// Takes t (types.Type) which is the key to record.
// Takes rt (reflect.Type) which is the synthesised reflect.Type.
func storeReflectType(state *typeConverterState, t types.Type, rt reflect.Type) {
	if state.cache == nil {
		return
	}
	state.cache[t] = rt
}

// basicToReflect converts a types.BasicKind to reflect.Type.
//
// Takes k (types.BasicKind) which is the basic type kind to convert.
//
// Returns the corresponding reflect.Type, or reflect.TypeFor[any] for
// unrecognised kinds.
func basicToReflect(k types.BasicKind) reflect.Type {
	switch k {
	case types.Bool, types.UntypedBool:
		return reflect.TypeFor[bool]()
	case types.Int, types.UntypedInt:
		return reflect.TypeFor[int]()
	case types.Int8:
		return reflect.TypeFor[int8]()
	case types.Int16:
		return reflect.TypeFor[int16]()
	case types.Int32, types.UntypedRune:
		return reflect.TypeFor[int32]()
	case types.Int64:
		return reflect.TypeFor[int64]()
	case types.Uint:
		return reflect.TypeFor[uint]()
	case types.Uint8:
		return reflect.TypeFor[uint8]()
	case types.Uint16:
		return reflect.TypeFor[uint16]()
	case types.Uint32:
		return reflect.TypeFor[uint32]()
	case types.Uint64:
		return reflect.TypeFor[uint64]()
	case types.Uintptr:
		return reflect.TypeFor[uintptr]()
	case types.Float32:
		return reflect.TypeFor[float32]()
	case types.Float64, types.UntypedFloat:
		return reflect.TypeFor[float64]()
	case types.String, types.UntypedString:
		return reflect.TypeFor[string]()
	case types.UnsafePointer:
		return reflect.TypeFor[unsafe.Pointer]()
	default:
		return reflect.TypeFor[any]()
	}
}
