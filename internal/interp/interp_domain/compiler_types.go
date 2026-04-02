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
	processing := make(map[types.Type]bool)
	return convertNamedStruct(ctx, symbols, processing, named)
}

// typeToReflect converts a go/types.Type to a reflect.Type.
//
// Takes ctx (context.Context) which carries the logger.
// Takes t (types.Type) which is the go/types.Type to convert.
// Takes symbols (*SymbolRegistry) which is used to resolve pre-registered
// native types to their real Go reflect.Type. May be nil.
//
// Returns the corresponding reflect.Type for the given type.
func typeToReflect(ctx context.Context, t types.Type, symbols *SymbolRegistry) reflect.Type {
	processing := make(map[types.Type]bool)
	return convertType(ctx, symbols, processing, t)
}

// convertNamedStruct converts a named struct type to a reflect.Type
// with cycle detection.
//
// Takes ctx (context.Context) which carries the logger.
// Takes symbols (*SymbolRegistry) which resolves pre-registered types.
// Takes processing (map[types.Type]bool) which tracks types being converted.
// Takes named (*types.Named) which is the named struct type to convert.
//
// Returns a reflect.Type with a sentinel field encoding the type identity.
func convertNamedStruct(ctx context.Context, symbols *SymbolRegistry, processing map[types.Type]bool, named *types.Named) reflect.Type {
	st, ok := named.Underlying().(*types.Struct)
	if !ok {
		return convertType(ctx, symbols, processing, named)
	}

	if processing[named] {
		logTypeCycle(ctx, named, processing)
		return reflect.TypeFor[any]()
	}
	processing[named] = true
	defer delete(processing, named)

	fields := buildStructFields(ctx, symbols, processing, st)
	fields = append(fields, sentinelField(named, st))

	return reflect.StructOf(fields)
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
// Takes symbols (*SymbolRegistry) which resolves pre-registered
// types.
// Takes processing (map[types.Type]bool) which tracks types
// being converted.
// Takes st (*types.Struct) which is the struct type whose fields
// are converted.
//
// Returns a slice of reflect.StructField for the struct.
func buildStructFields(ctx context.Context, symbols *SymbolRegistry, processing map[types.Type]bool, st *types.Struct) []reflect.StructField {
	fields := make([]reflect.StructField, st.NumFields())
	for i := range st.NumFields() {
		f := st.Field(i)
		var fieldType reflect.Type
		if fieldTypeInvolvesCycle(f.Type(), processing) {
			fieldType = reflect.TypeFor[any]()
		} else {
			fieldType = convertType(ctx, symbols, processing, f.Type())
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
// Takes symbols (*SymbolRegistry) which resolves pre-registered types.
// Takes processing (map[types.Type]bool) which tracks types being converted.
// Takes t (types.Type) which is the type to convert.
//
// Returns the corresponding reflect.Type.
func convertType(ctx context.Context, symbols *SymbolRegistry, processing map[types.Type]bool, t types.Type) reflect.Type {
	if alias, ok := t.(*types.Alias); ok {
		if rt := resolveRegisteredType(alias.Obj(), symbols); rt != nil {
			return rt
		}
		return convertType(ctx, symbols, processing, types.Unalias(t))
	}

	if named, ok := t.(*types.Named); ok {
		if rt := resolveRegisteredType(named.Obj(), symbols); rt != nil {
			return rt
		}
		if _, isStruct := named.Underlying().(*types.Struct); isStruct {
			return convertNamedStruct(ctx, symbols, processing, named)
		}
	}

	return convertUnderlying(ctx, symbols, processing, t.Underlying())
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
// Takes symbols (*SymbolRegistry) which resolves pre-registered types.
// Takes processing (map[types.Type]bool) which tracks types being converted.
// Takes underlying (types.Type) which is the underlying type to convert.
//
// Returns the corresponding reflect.Type.
func convertUnderlying(ctx context.Context, symbols *SymbolRegistry, processing map[types.Type]bool, underlying types.Type) reflect.Type {
	switch typ := underlying.(type) {
	case *types.Basic:
		return basicToReflect(typ.Kind())
	case *types.Slice:
		return reflect.SliceOf(convertType(ctx, symbols, processing, typ.Elem()))
	case *types.Map:
		return reflect.MapOf(convertType(ctx, symbols, processing, typ.Key()), convertType(ctx, symbols, processing, typ.Elem()))
	case *types.Pointer:
		return reflect.PointerTo(convertType(ctx, symbols, processing, typ.Elem()))
	case *types.Array:
		return reflect.ArrayOf(int(typ.Len()), convertType(ctx, symbols, processing, typ.Elem()))
	case *types.Struct:
		return convertStruct(ctx, symbols, processing, typ)
	case *types.Signature:
		return convertSignature(ctx, symbols, processing, typ)
	case *types.Chan:
		return convertChan(ctx, symbols, processing, typ)
	case *types.Interface:
		return reflect.TypeFor[any]()
	default:
		return reflect.TypeFor[any]()
	}
}

// convertStruct converts a go/types anonymous struct to reflect.Type.
//
// Takes ctx (context.Context) which carries the logger.
// Takes symbols (*SymbolRegistry) which resolves pre-registered types.
// Takes processing (map[types.Type]bool) which tracks types being converted.
// Takes typ (*types.Struct) which is the struct type to convert.
//
// Returns the corresponding reflect.Type.
func convertStruct(ctx context.Context, symbols *SymbolRegistry, processing map[types.Type]bool, typ *types.Struct) reflect.Type {
	fields := make([]reflect.StructField, typ.NumFields())
	for i := range typ.NumFields() {
		f := typ.Field(i)
		fields[i] = reflect.StructField{
			Name: f.Name(),
			Type: convertType(ctx, symbols, processing, f.Type()),
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
// Takes symbols (*SymbolRegistry) which resolves pre-registered types.
// Takes processing (map[types.Type]bool) which tracks types being converted.
// Takes typ (*types.Signature) which is the function signature to convert.
//
// Returns the corresponding reflect.FuncOf type.
func convertSignature(ctx context.Context, symbols *SymbolRegistry, processing map[types.Type]bool, typ *types.Signature) reflect.Type {
	var paramTypes []reflect.Type
	for v := range typ.Params().Variables() {
		paramTypes = append(paramTypes, convertType(ctx, symbols, processing, v.Type()))
	}
	var resultTypes []reflect.Type
	for v := range typ.Results().Variables() {
		resultTypes = append(resultTypes, convertType(ctx, symbols, processing, v.Type()))
	}
	return reflect.FuncOf(paramTypes, resultTypes, typ.Variadic())
}

// convertChan converts a go/types channel type to reflect.Type.
//
// Takes ctx (context.Context) which carries the logger.
// Takes symbols (*SymbolRegistry) which resolves pre-registered types.
// Takes processing (map[types.Type]bool) which tracks types being converted.
// Takes typ (*types.Chan) which is the channel type to convert.
//
// Returns the corresponding reflect.ChanOf type.
func convertChan(ctx context.Context, symbols *SymbolRegistry, processing map[types.Type]bool, typ *types.Chan) reflect.Type {
	elemType := convertType(ctx, symbols, processing, typ.Elem())
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
