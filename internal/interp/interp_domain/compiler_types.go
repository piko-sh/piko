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
	"slices"
	"strings"
	"unsafe"

	"piko.sh/piko/internal/logger/logger_domain"
)

const (
	// embeddedUnexportedPrefix is prepended to the synthesised reflect field name for
	// embedded unexported types because reflect.StructOf rejects anonymous fields with
	// PkgPath set. isAnonymousField and structFieldNameMatches treat such fields as
	// embedded.
	embeddedUnexportedPrefix = "PikoEmbed_"
)

// typeConverterState carries registry lookups, the active cycle-detection set, and an
// optional shared cache through one typeToReflect call chain.
//
// The cache preserves identity for mutually recursive named types across independent
// conversion entry points.
type typeConverterState struct {
	// symbols resolves pre-registered native types via the SymbolRegistry. May be nil when
	// conversion runs outside a registered-symbol context.
	symbols *SymbolRegistry

	// processing tracks the named types currently being synthesised along the active
	// recursion path, enabling cycle detection so self-referential struct fields collapse to
	// reflect.TypeFor[any]().
	processing map[types.Type]bool

	// cache holds the synthesised reflect.Type per go/types.Type so mutually recursive named
	// types share identity across entry points. May be nil.
	cache map[types.Type]reflect.Type

	// globals holds the per-Service shared registry for named interfaces.
	//
	// User-declared named interfaces are published into globals as pikoTypes so the
	// reflect.TypeOf intercept can later wrap *interface{} results with the correct
	// source-level identity. May be nil when conversion runs outside a Service context (e.g.
	// ad-hoc TypeOf in tests).
	globals *globalStore
}

// typeToReflectCached converts a go/types.Type to a reflect.Type using an optional shared
// cache so repeated conversions of the same nominal type yield identity-stable
// reflect.Types.
//
// Takes t (types.Type) which is the go/types.Type to convert.
// Takes symbols (*SymbolRegistry) which resolves pre-registered native reflect.Types; may
// be nil.
// Takes cache (map[types.Type]reflect.Type) which is the shared identity map across
// calls; may be nil.
// Takes globals (*globalStore) which receives user-declared named interface registrations
// as a side effect; may be nil.
//
// Returns the synthesised reflect.Type for t.
func typeToReflectCached(ctx context.Context, t types.Type, symbols *SymbolRegistry, cache map[types.Type]reflect.Type, globals *globalStore) reflect.Type {
	state := &typeConverterState{
		symbols:    symbols,
		processing: make(map[types.Type]bool),
		cache:      cache,
		globals:    globals,
	}
	return convertType(ctx, state, t)
}

// convertNamedStruct converts a named struct type to a reflect.Type with cycle detection.
//
// Linked generics elide the sentinel field so they share structural identity with
// reflect-constructed instances.
//
// Takes state (*typeConverterState) which holds registry, processing set, and cache.
// Takes named (*types.Named) which is the go/types named struct to convert.
//
// Returns the synthesised reflect.Type for named.
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

// isLinkedGenericNamed reports whether named is registered as a linked generic type.
//
// Linked generics rely on reflect's structural type identity, so the converter omits the
// sentinel field for them.
//
// Takes state (*typeConverterState) which holds the symbol registry.
// Takes named (*types.Named) which is the go/types named type to test.
//
// Returns true when named is registered as a linked generic type.
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

// logTypeCycle logs a warning when a recursion cycle is detected during type conversion.
//
// The log entry includes the offending type and the chain of types on the active
// conversion path.
//
// Takes named (*types.Named) which is the named type that triggered the cycle.
// Takes processing (map[types.Type]bool) which is the active set of named types on the
// conversion path.
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
	l.Warn("Type cycle detected in compiler type conversion",
		slog.String("type", typePath),
		slog.Any("activeChain", chain),
	)
}

// buildStructFields converts each field of st into a reflect.StructField.
//
// Cycle-bearing field types collapse to any, and embedded unexported fields are renamed
// to satisfy reflect.StructOf's anonymous-plus-PkgPath restriction.
//
// Takes state (*typeConverterState) which is the per-call converter state.
// Takes st (*types.Struct) which is the go/types struct whose fields are being converted.
//
// Returns the reflect.StructField slice ready for reflect.StructOf.
func buildStructFields(ctx context.Context, state *typeConverterState, st *types.Struct) []reflect.StructField {
	fields := make([]reflect.StructField, st.NumFields())
	for i := range st.NumFields() {
		f := st.Field(i)
		var fieldType reflect.Type
		cycleBroken := fieldTypeInvolvesCycle(f.Type(), state.processing)
		if cycleBroken {
			fieldType = convertFieldBreakingCycles(ctx, state, f.Type())
		} else {
			fieldType = convertType(ctx, state, f.Type())
		}
		tag := st.Tag(i)
		if cycleBroken {
			tag = appendCycleBrokenTag(tag)
		}
		fields[i] = reflect.StructField{
			Name:      f.Name(),
			Type:      fieldType,
			Tag:       reflect.StructTag(tag),
			Anonymous: f.Embedded(),
		}
		if f.Embedded() && embeddedFieldNeedsRename(f, fieldType) {
			fields[i].Name = embeddedUnexportedPrefix + f.Name()
			fields[i].Anonymous = false
			continue
		}
		if !f.Exported() && f.Pkg() != nil {
			fields[i].PkgPath = f.Pkg().Path()
		}
	}
	return fields
}

// embeddedFieldNeedsRename reports whether an embedded field must be renamed.
//
// The rename emits a non-anonymous reflect.StructField to keep reflect.StructOf happy. It
// applies for unexported embedded types (anonymous plus PkgPath set is illegal) and for
// any embedded type that carries methods, because every piko-synthesised struct gains the
// `_pikoID_` sentinel field and so always has more than one field; without the rename a
// piko struct embedding *bytes.Buffer would panic the host at reflect.StructOf time.
// Renamed embedded fields carry the embeddedUnexportedPrefix marker, which
// isAnonymousField and the field-access / method-promotion paths already recognise, so
// promotion (logger.String()) and named access (logger.Buffer) keep working.
//
// Takes f (*types.Var) which is the source-level field.
// Takes fieldType (reflect.Type) which is the field's converted type.
//
// Returns true when the field must be renamed and de-anonymised.
func embeddedFieldNeedsRename(f *types.Var, fieldType reflect.Type) bool {
	if !f.Exported() && f.Pkg() != nil {
		return true
	}
	if fieldType == nil {
		return false
	}
	return fieldType.NumMethod() > 0
}

// appendCycleBrokenTag returns the struct tag with the cycleBrokenTagKey marker appended,
// or returns the marker alone when tag is empty.
//
// buildStructFieldLayout consults the marker to decide whether the field qualifies for
// the opGetStructFieldRawPointerT0 specialisation.
//
// Takes tag (string) which is the existing struct tag string.
//
// Returns the tag with the cycle-broken marker appended.
func appendCycleBrokenTag(tag string) string {
	marker := cycleBrokenTagKey + `:"` + cycleBrokenTagValue + `"`
	if tag == "" {
		return marker
	}
	return tag + " " + marker
}

// convertFieldBreakingCycles lowers a cycle-bearing field type to a reflect.Type,
// collapsing only the cycle-causing leaf to any while preserving the surrounding
// container.
//
// Cycle-free sub-branches fall through to convertType for their exact reflect.Type.
//
// Takes state (*typeConverterState) which is the per-call converter state.
// Takes fieldType (types.Type) which is the field type that involves the cycle.
//
// Returns the lowered reflect.Type for fieldType.
func convertFieldBreakingCycles(ctx context.Context, state *typeConverterState, fieldType types.Type) reflect.Type {
	if !fieldTypeInvolvesCycle(fieldType, state.processing) {
		return convertType(ctx, state, fieldType)
	}
	switch typ := fieldType.(type) {
	case *types.Pointer:
		return reflect.TypeFor[any]()
	case *types.Array:
		return reflect.ArrayOf(int(typ.Len()),
			convertFieldBreakingCycles(ctx, state, typ.Elem()))
	case *types.Slice:
		return reflect.SliceOf(convertFieldBreakingCycles(ctx, state, typ.Elem()))
	case *types.Map:
		return reflect.MapOf(
			convertFieldBreakingCycles(ctx, state, typ.Key()),
			convertFieldBreakingCycles(ctx, state, typ.Elem()),
		)
	case *types.Chan:
		elem := convertFieldBreakingCycles(ctx, state, typ.Elem())
		var direction reflect.ChanDir
		switch typ.Dir() {
		case types.SendRecv:
			direction = reflect.BothDir
		case types.SendOnly:
			direction = reflect.SendDir
		case types.RecvOnly:
			direction = reflect.RecvDir
		}
		return reflect.ChanOf(direction, elem)
	case *types.Alias:
		return convertFieldBreakingCycles(ctx, state, types.Unalias(fieldType))
	case *types.Named:

		if _, isSig := typ.Underlying().(*types.Signature); isSig {
			return reflect.TypeFor[any]()
		}
		return reflect.TypeFor[any]()
	case *types.Signature:

		_ = typ
		return reflect.TypeFor[any]()
	default:
		return reflect.TypeFor[any]()
	}
}

// isAnonymousField reports whether field represents an embedded field at the source
// level.
//
// Includes embedded unexported types renamed with the embeddedUnexportedPrefix marker.
//
// Takes field (reflect.StructField) which is the reflect.StructField to test.
//
// Returns true when field is embedded (anonymous or marker-prefixed).
func isAnonymousField(field reflect.StructField) bool {
	if field.Anonymous {
		return true
	}
	return strings.HasPrefix(field.Name, embeddedUnexportedPrefix)
}

// sentinelField builds the zero-size struct field that encodes named's type identity.
//
// The field is unexported with a PkgPath derived from named (falling back to the first
// unexported field's package) so reflect treats it as private to the original package.
//
// Takes named (*types.Named) which is the go/types named type whose identity is being
// encoded.
// Takes st (*types.Struct) which is the underlying struct type used to source a fallback
// PkgPath.
//
// Returns the unexported sentinel StructField.
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

// convertType converts a go/types.Type to a reflect.Type, preferring a pre-registered
// native reflect.Type from the symbol registry where available.
//
// Aliases and named types resolve via the registry first; otherwise the call dispatches
// to the structural converter for the underlying form.
//
// Takes state (*typeConverterState) which is the per-call converter state.
// Takes t (types.Type) which is the go/types.Type to convert.
//
// Returns the synthesised or registered reflect.Type for t.
func convertType(ctx context.Context, state *typeConverterState, t types.Type) reflect.Type {
	if alias, ok := t.(*types.Alias); ok {
		if rt := resolveRegisteredType(alias.Obj(), state.symbols); rt != nil {
			return rt
		}
		return convertType(ctx, state, types.Unalias(t))
	}
	if named, ok := t.(*types.Named); ok {
		if rt := convertNamedType(ctx, state, named); rt != nil {
			return rt
		}
	}
	return convertUnderlying(ctx, state, t.Underlying())
}

// convertNamedType handles the named-type branch of convertType.
//
// Resolves native backing, pre-registered reflect.Types, well-known interface
// short-circuits, and struct-typed named declarations.
//
// Takes ctx (context.Context) which drives recursive reflect-type synthesis.
// Takes state (*typeConverterState) which carries the active cache and registries.
// Takes named (*types.Named) which is the named type to convert.
//
// Returns reflect.Type which is the resolved type, or nil to signal "fall through to
// convertUnderlying on t.Underlying()".
func convertNamedType(ctx context.Context, state *typeConverterState, named *types.Named) reflect.Type {
	if rt := resolveNativeBackedType(named.Obj(), state.symbols); rt != nil {
		return rt
	}
	if rt := resolveRegisteredType(named.Obj(), state.symbols); rt != nil {
		return rt
	}
	if intf, isInterface := named.Underlying().(*types.Interface); isInterface {
		pkgPath := ""
		if named.Obj().Pkg() != nil {
			pkgPath = named.Obj().Pkg().Path()
		}
		if rt, ok := wellKnownNamedInterfaceReflectType(pkgPath, named.Obj().Name()); ok {
			return rt
		}
		registerUserNamedInterfacePikoType(state, pkgPath, named.Obj().Name(), intf)
	}
	if _, isStruct := named.Underlying().(*types.Struct); isStruct {
		return convertNamedStruct(ctx, state, named)
	}
	return nil
}

// registerUserNamedInterfacePikoType publishes a pikoType for a user-declared named
// interface into the per-Service registry so the runtime reflect.TypeOf intercept can
// wrap *interface{} results with the source-level identity (pkg.IfaceName, method set,
// etc.) that Go's reflect cannot otherwise preserve.
//
// Called from convertNamedType when the named type's underlying is *types.Interface and
// the type is NOT in the wellKnownNamedInterfaceRegistry. No-op when state.globals is nil
// (ad-hoc type conversion outside a Service context).
//
// Takes state (*typeConverterState) which must carry a non-nil globals to publish.
// Takes pkgPath (string) which is the defining package's full import path.
// Takes name (string) which is the bare interface name (e.g. "myiface").
// Takes intf (*types.Interface) whose methods are read to populate the pikoType
// method-name set.
func registerUserNamedInterfacePikoType(state *typeConverterState, pkgPath, name string, intf *types.Interface) {
	if state == nil || state.globals == nil || name == "" {
		return
	}
	completed := intf.Complete()
	count := completed.NumMethods()
	methods := make([]string, count)
	for i := range count {
		methods[i] = completed.Method(i).Name()
	}
	slices.Sort(methods)
	piko := newPikoTypeNamedInterface(pkgPath, name, methods)
	state.globals.registerUserNamedInterface(piko.qualifiedName, piko)
}

// resolveNativeBackedType returns the canonical erased reflect.Type for obj.
//
// Applies when obj names a native-backed generic type (e.g. atomic.Pointer). Every
// instantiation atomic.Pointer[Config] resolves to the same method-bearing erased
// atomic.Pointer[struct{}]: methods are layout-erased over a single machine pointer, so
// one reflect.Type serves all element types.
//
// Takes obj (*types.TypeName) which is the go/types type name.
// Takes symbols (*SymbolRegistry) which is the symbol registry; may be nil.
//
// Returns the erased reflect.Type or nil when obj is not native-backed.
func resolveNativeBackedType(obj *types.TypeName, symbols *SymbolRegistry) reflect.Type {
	if symbols == nil || obj.Pkg() == nil {
		return nil
	}
	rt, found := symbols.NativeBackedReflectType(obj.Pkg().Path(), obj.Name())
	if !found {
		return nil
	}
	return rt
}

// resolveRegisteredType returns the pre-registered reflect.Type for obj when the symbol
// registry holds one, otherwise nil.
//
// Takes obj (*types.TypeName) which is the go/types type name to look up.
// Takes symbols (*SymbolRegistry) which is the symbol registry; may be nil.
//
// Returns the registered reflect.Type or nil when no entry exists.
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

// convertUnderlying converts the underlying form of a go/types.Type to a reflect.Type by
// dispatching on the concrete type.
//
// Interfaces and unknown types collapse to reflect.TypeFor[any]().
//
// Takes state (*typeConverterState) which is the per-call converter state.
// Takes underlying (types.Type) which is the underlying go/types form to convert.
//
// Returns the reflect.Type for the underlying form.
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
		return convertChannel(ctx, state, typ)
	case *types.Interface:
		return reflect.TypeFor[any]()
	default:
		return reflect.TypeFor[any]()
	}
}

// convertStruct converts an anonymous struct type to a reflect.Type without adding a
// sentinel identity field.
//
// Unexported fields keep their PkgPath so reflect honours their visibility.
//
// Takes state (*typeConverterState) which is the per-call converter state.
// Takes typ (*types.Struct) which is the anonymous struct type to convert.
//
// Returns the synthesised reflect.Type for typ.
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

// convertSignature converts a go/types function signature to a reflect.FuncOf type,
// preserving the variadic flag.
//
// Takes state (*typeConverterState) which is the per-call converter state.
// Takes typ (*types.Signature) which is the function signature to convert.
//
// Returns the synthesised reflect.Type for typ.
func convertSignature(ctx context.Context, state *typeConverterState, typ *types.Signature) reflect.Type {
	var parameterTypes []reflect.Type
	for v := range typ.Params().Variables() {
		parameterTypes = append(parameterTypes, convertType(ctx, state, v.Type()))
	}
	var resultTypes []reflect.Type
	for v := range typ.Results().Variables() {
		resultTypes = append(resultTypes, convertType(ctx, state, v.Type()))
	}
	return reflect.FuncOf(parameterTypes, resultTypes, typ.Variadic())
}

// convertChannel converts a go/types channel type to a reflect.ChanOf type, preserving
// the channel direction.
//
// Takes state (*typeConverterState) which is the per-call converter state.
// Takes typ (*types.Chan) which is the channel type to convert.
//
// Returns the synthesised reflect.Type for typ.
func convertChannel(ctx context.Context, state *typeConverterState, typ *types.Chan) reflect.Type {
	elementType := convertType(ctx, state, typ.Elem())
	var directory reflect.ChanDir
	switch typ.Dir() {
	case types.SendRecv:
		directory = reflect.BothDir
	case types.SendOnly:
		directory = reflect.SendDir
	case types.RecvOnly:
		directory = reflect.RecvDir
	}
	return reflect.ChanOf(directory, elementType)
}

// fieldTypeInvolvesCycle reports whether fieldType references any named type on the
// active conversion path, transitively through pointers, slices, arrays, maps, channels,
// and aliases.
//
// Takes fieldType (types.Type) which is the field type to inspect.
// Takes processing (map[types.Type]bool) which is the active set of named types on the
// conversion path.
//
// Returns true when fieldType reaches a named type already on the path.
func fieldTypeInvolvesCycle(fieldType types.Type, processing map[types.Type]bool) bool {
	switch t := fieldType.(type) {
	case *types.Named:
		return namedTypeInvolvesCycle(t, processing)
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
	case *types.Signature:
		return signatureInvolvesCycle(t, processing)
	default:
		return false
	}
}

// namedTypeInvolvesCycle reports whether a *types.Named field reaches the active
// named-cycle set. Named func types descend into their signature so cyclic function-typed
// fields take the cycle-breaking conversion path rather than the noisy logTypeCycle bail.
//
// Takes t (*types.Named) which is the named type to inspect.
// Takes processing (map[types.Type]bool) which holds the active conversion path.
//
// Returns bool which is true when t (or its signature underlying) is on the processing
// path.
func namedTypeInvolvesCycle(t *types.Named, processing map[types.Type]bool) bool {
	if processing[t] {
		return true
	}
	if _, isSig := t.Underlying().(*types.Signature); isSig {
		return fieldTypeInvolvesCycle(t.Underlying(), processing)
	}
	return false
}

// signatureInvolvesCycle reports whether either the parameter or the result list of a
// function signature reaches a named type currently on the conversion path (e.g.
// yaml_parser_t.read_handler takes *yaml_parser_t).
//
// Takes t (*types.Signature) which is the signature to inspect.
// Takes processing (map[types.Type]bool) which holds the active conversion path.
//
// Returns bool which is true when any parameter or result type reaches the processing
// path.
func signatureInvolvesCycle(t *types.Signature, processing map[types.Type]bool) bool {
	if params := t.Params(); params != nil {
		for v := range params.Variables() {
			if fieldTypeInvolvesCycle(v.Type(), processing) {
				return true
			}
		}
	}
	if results := t.Results(); results != nil {
		for v := range results.Variables() {
			if fieldTypeInvolvesCycle(v.Type(), processing) {
				return true
			}
		}
	}
	return false
}

// cachedReflectType returns the cached reflect.Type for t, or nil when no cache is
// configured or no entry exists.
//
// Takes state (*typeConverterState) which holds the shared cache.
// Takes t (types.Type) which is the go/types.Type whose cached entry is requested.
//
// Returns the cached reflect.Type or nil when absent.
func cachedReflectType(state *typeConverterState, t types.Type) reflect.Type {
	if state.cache == nil {
		return nil
	}
	return state.cache[t]
}

// storeReflectType records rt for t in the shared cache after full synthesis so
// subsequent conversions yield an identical reflect.Type.
//
// No-op when no cache is configured.
//
// Takes state (*typeConverterState) which holds the shared cache.
// Takes t (types.Type) which is the go/types.Type used as the cache key.
// Takes rt (reflect.Type) which is the synthesised reflect.Type to store.
func storeReflectType(state *typeConverterState, t types.Type, rt reflect.Type) {
	if state.cache == nil {
		return
	}
	state.cache[t] = rt
}

// basicToReflect converts a types.BasicKind to the matching reflect.Type, returning
// reflect.TypeFor[any]() for unrecognised kinds.
//
// Takes k (types.BasicKind) which is the basic kind to convert.
//
// Returns the matching reflect.Type or reflect.TypeFor[any]() when unknown.
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
