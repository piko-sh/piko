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
	"go/types"
	"reflect"
)

// classifyLinkedGenericType inspects an exported generic type and
// returns the metadata required to emit an interp_link.LinkedGenericType
// registration. Only generic struct types are supported in v1; other
// shapes (interface constraints, generic type aliases) fall back to
// the skip path by returning false.
//
// Takes typeName (*types.TypeName) which is the type name being classified.
//
// Returns the LinkedGenericTypeInfo and a bool indicating success.
func classifyLinkedGenericType(typeName *types.TypeName) (LinkedGenericTypeInfo, bool) {
	typeParams, structType, paramIndex, ok := resolveGenericStructShape(typeName)
	if !ok {
		return LinkedGenericTypeInfo{}, false
	}

	fields := make([]LinkedGenericFieldInfo, 0, structType.NumFields())
	for fieldIndex := range structType.NumFields() {
		field := structType.Field(fieldIndex)
		if !field.Exported() {
			continue
		}
		fieldType, ok := describeGenericFieldType(field.Type(), paramIndex)
		if !ok {
			return LinkedGenericTypeInfo{}, false
		}
		fields = append(fields, LinkedGenericFieldInfo{
			Name:      field.Name(),
			Tag:       structType.Tag(fieldIndex),
			FieldType: fieldType,
			Exported:  true,
		})
	}

	return LinkedGenericTypeInfo{
		Name:         typeName.Name(),
		TypeArgCount: typeParams.Len(),
		Fields:       fields,
	}, true
}

// resolveGenericStructShape returns the struct shape of a generic type
// or generic alias, substituting the alias-side type parameters so
// downstream consumers always instantiate against the alias.
//
// Takes typeName (*types.TypeName) which is the exported type name.
//
// Returns the alias-side type parameters, the fully-substituted
// underlying struct, a lookup map for param->index, and a success
// flag. Returns false when the shape cannot be represented (non-
// struct underlying, unresolvable alias target, instantiation failure).
func resolveGenericStructShape(typeName *types.TypeName) (*types.TypeParamList, *types.Struct, map[*types.TypeParam]int, bool) {
	aliasTypeParams, aliasTarget, ok := unaliasGeneric(typeName)
	if !ok {
		return nil, nil, nil, false
	}
	targetNamed, ok := aliasTarget.(*types.Named)
	if !ok {
		return nil, nil, nil, false
	}
	typeParams := aliasTypeParams
	if typeParams == nil {
		typeParams = targetNamed.TypeParams()
	}
	if typeParams == nil || typeParams.Len() == 0 {
		return nil, nil, nil, false
	}

	paramIndex := make(map[*types.TypeParam]int, typeParams.Len())
	typeArgs := make([]types.Type, typeParams.Len())
	for paramPosition := range typeParams.Len() {
		paramIndex[typeParams.At(paramPosition)] = paramPosition
		typeArgs[paramPosition] = typeParams.At(paramPosition)
	}

	targetParams := targetNamed.TypeParams()
	if targetParams == nil || targetParams.Len() == 0 {
		structType, ok := targetNamed.Underlying().(*types.Struct)
		if !ok {
			return nil, nil, nil, false
		}
		return typeParams, structType, paramIndex, true
	}

	instance, err := types.Instantiate(nil, targetNamed, typeArgs, true)
	if err != nil {
		return nil, nil, nil, false
	}
	instantiatedNamed, ok := instance.(*types.Named)
	if !ok {
		return nil, nil, nil, false
	}
	structType, ok := instantiatedNamed.Underlying().(*types.Struct)
	if !ok {
		return nil, nil, nil, false
	}
	return typeParams, structType, paramIndex, true
}

// unaliasGeneric resolves a TypeName to the underlying (possibly
// generic) target type, propagating the declared type parameters so
// classifyLinkedGenericType can use the alias-side names even when
// they differ from the target's.
//
// Takes typeName (*types.TypeName) which is the type name.
//
// Returns the resolved TypeParamList (from the alias or underlying),
// the target type, and a success flag.
func unaliasGeneric(typeName *types.TypeName) (*types.TypeParamList, types.Type, bool) {
	switch typeValue := typeName.Type().(type) {
	case *types.Alias:
		params := typeValue.TypeParams()
		return params, types.Unalias(typeValue), true
	case *types.Named:
		params := typeValue.TypeParams()
		return params, typeValue, true
	}
	return nil, nil, false
}

// extractLinkedFuncTuple walks a go/types.Tuple (the params or
// results of a generic function signature) and produces the
// serialisable descriptor list used by LinkedGenericFuncInfo. It
// reuses describeGenericFuncType so the top-level walk matches the
// field walk used by LinkedGenericType, with an extra fallback that
// maps error-returning and generic-typed positions to their
// appropriate kinds.
//
// Takes tuple (*types.Tuple) which is the parameter or result list.
// Takes paramIndex (map[*types.TypeParam]int) which resolves the
// generic's type parameters.
//
// Returns the per-position descriptor list (nil when empty).
func extractLinkedFuncTuple(tuple *types.Tuple, paramIndex map[*types.TypeParam]int) []GenericFieldTypeInfo {
	if tuple == nil || tuple.Len() == 0 {
		return nil
	}
	result := make([]GenericFieldTypeInfo, tuple.Len())
	for index := range tuple.Len() {
		descriptor, ok := describeGenericFuncType(tuple.At(index).Type(), paramIndex)
		if !ok {
			result[index] = GenericFieldTypeInfo{Kind: GenericFieldKindInterface}
			continue
		}
		result[index] = descriptor
	}
	return result
}

// describeGenericFuncType is the broader sibling of
// describeGenericFieldType: it additionally understands generic named
// types (SearchResult[T], Foo[K,V]) and the built-in error interface,
// both of which appear routinely in function signatures but are
// forbidden inside generic struct fields.
//
// Takes t (types.Type) which is the type being described.
// Takes paramIndex (map[*types.TypeParam]int) which resolves the
// owning generic's type parameters.
//
// Returns the descriptor and a bool flag; a false flag signals the
// shape cannot be captured and the caller should substitute
// `interface{}` for the position.
func describeGenericFuncType(t types.Type, paramIndex map[*types.TypeParam]int) (GenericFieldTypeInfo, bool) {
	if typeParam, ok := t.(*types.TypeParam); ok {
		index, ok := paramIndex[typeParam]
		if !ok {
			return GenericFieldTypeInfo{}, false
		}
		return GenericFieldTypeInfo{
			Kind:         GenericFieldKindTypeArg,
			TypeArgIndex: index,
		}, true
	}
	if named, ok := t.(*types.Named); ok {
		if descriptor, ok := describeNamedOrAlias(named, paramIndex); ok {
			return descriptor, true
		}
	}
	if alias, ok := t.(*types.Alias); ok {
		if descriptor, ok := describeAlias(alias, paramIndex); ok {
			return descriptor, true
		}
	}
	if _, ok := t.(*types.Interface); ok {
		if iface, isIface := t.Underlying().(*types.Interface); isIface && iface.Empty() {
			return GenericFieldTypeInfo{Kind: GenericFieldKindInterface}, true
		}
	}
	return describeGenericFieldType(t, paramIndex)
}

// describeAlias produces a descriptor for a *types.Alias.
//
// The alias's own Obj() is preserved so the emitted reference matches
// the source-level name the user writes (piko.RequestData rather than
// templater_dto.RequestData). When the alias resolves to a generic
// instantiation its TypeArgs are copied from the underlying Named.
//
// Takes alias (*types.Alias) which is the alias type.
// Takes paramIndex (map[*types.TypeParam]int) which resolves type
// parameter references in TypeArgs.
//
// Returns the descriptor and a bool flag; false signals the alias
// cannot be represented and the caller should fall back to other
// strategies.
func describeAlias(alias *types.Alias, paramIndex map[*types.TypeParam]int) (GenericFieldTypeInfo, bool) {
	obj := alias.Obj()
	if obj == nil || obj.Pkg() == nil {
		return GenericFieldTypeInfo{}, false
	}

	underlying := types.Unalias(alias)
	named, ok := underlying.(*types.Named)
	if !ok {
		return GenericFieldTypeInfo{
			Kind:         GenericFieldKindNamed,
			NamedPackage: obj.Pkg().Path(),
			NamedName:    obj.Name(),
		}, true
	}

	aliasTypeParams := alias.TypeParams()
	if aliasTypeParams != nil && aliasTypeParams.Len() > 0 {
		return describeGenericAlias(alias, named, paramIndex)
	}
	if named.TypeParams() != nil && named.TypeParams().Len() > 0 {
		return describeGenericAlias(alias, named, paramIndex)
	}
	if named.TypeArgs() != nil && named.TypeArgs().Len() > 0 {
		return describeGenericAlias(alias, named, paramIndex)
	}
	return GenericFieldTypeInfo{
		Kind:         GenericFieldKindNamed,
		NamedPackage: obj.Pkg().Path(),
		NamedName:    obj.Name(),
	}, true
}

// describeGenericAlias emits a NamedGeneric descriptor for a generic
// alias, taking the package + name from the alias itself so the
// interpreter synthesiser finds the alias's own registration.
//
// Takes alias (*types.Alias) which is the alias type.
// Takes named (*types.Named) which is the alias target.
// Takes paramIndex (map[*types.TypeParam]int) which resolves type
// parameters referenced in TypeArgs.
//
// Returns the NamedGeneric descriptor and a bool indicating whether
// the shape could be captured.
func describeGenericAlias(
	alias *types.Alias,
	named *types.Named,
	paramIndex map[*types.TypeParam]int,
) (GenericFieldTypeInfo, bool) {
	obj := alias.Obj()
	info := GenericFieldTypeInfo{
		Kind:         GenericFieldKindNamedGeneric,
		NamedPackage: obj.Pkg().Path(),
		NamedName:    obj.Name(),
	}
	typeArgs := named.TypeArgs()
	if typeArgs != nil && typeArgs.Len() > 0 {
		info.TypeArgs = make([]GenericFieldTypeInfo, typeArgs.Len())
		for i := range typeArgs.Len() {
			descriptor, ok := describeGenericFuncType(typeArgs.At(i), paramIndex)
			if !ok {
				return GenericFieldTypeInfo{}, false
			}
			info.TypeArgs[i] = descriptor
		}
		return info, true
	}
	aliasParams := alias.TypeParams()
	if aliasParams != nil && aliasParams.Len() > 0 {
		info.TypeArgs = make([]GenericFieldTypeInfo, aliasParams.Len())
		for i := range aliasParams.Len() {
			info.TypeArgs[i] = GenericFieldTypeInfo{
				Kind:         GenericFieldKindTypeArg,
				TypeArgIndex: i,
			}
		}
		return info, true
	}
	if typeParams := named.TypeParams(); typeParams != nil {
		info.TypeArgs = make([]GenericFieldTypeInfo, typeParams.Len())
		for i := range typeParams.Len() {
			info.TypeArgs[i] = GenericFieldTypeInfo{
				Kind:         GenericFieldKindTypeArg,
				TypeArgIndex: i,
			}
		}
	}
	return info, true
}

// describeNamedOrAlias resolves a (possibly generic, possibly alias)
// Named type to the correct descriptor kind, handling every shape
// that can appear in a linked function's parameter or result
// position: built-in error, instantiated generic, uninstantiated
// generic (used as a type reference), and non-generic named.
//
// Takes named (*types.Named) which is the resolved named type.
// Takes paramIndex (map[*types.TypeParam]int) which resolves the
// owning generic's type parameters.
//
// Returns the descriptor and a bool flag; false means the caller
// should fall back to structural decomposition (slice/map/etc).
func describeNamedOrAlias(named *types.Named, paramIndex map[*types.TypeParam]int) (GenericFieldTypeInfo, bool) {
	obj := named.Obj()
	if obj != nil && obj.Pkg() == nil && obj.Name() == "error" {
		return GenericFieldTypeInfo{Kind: GenericFieldKindError}, true
	}
	if named.TypeArgs() != nil && named.TypeArgs().Len() > 0 {
		return describeNamedGeneric(named, paramIndex)
	}
	if named.TypeParams() != nil && named.TypeParams().Len() > 0 {
		return describeNamedGeneric(named, paramIndex)
	}
	if obj == nil || obj.Pkg() == nil {
		return GenericFieldTypeInfo{}, false
	}
	return GenericFieldTypeInfo{
		Kind:         GenericFieldKindNamed,
		NamedPackage: obj.Pkg().Path(),
		NamedName:    obj.Name(),
	}, true
}

// describeNamedGeneric builds a GenericFieldKindNamedGeneric
// descriptor for a generic named type, whether fully instantiated
// (TypeArgs populated) or referenced by its own type parameters.
//
// Takes named (*types.Named) which names the generic type.
// Takes paramIndex (map[*types.TypeParam]int) which resolves type
// parameter references in the TypeArgs.
//
// Returns the descriptor plus a flag indicating whether the shape is
// fully representable.
func describeNamedGeneric(named *types.Named, paramIndex map[*types.TypeParam]int) (GenericFieldTypeInfo, bool) {
	obj := named.Obj()
	if obj == nil || obj.Pkg() == nil {
		return GenericFieldTypeInfo{}, false
	}
	info := GenericFieldTypeInfo{
		Kind:         GenericFieldKindNamedGeneric,
		NamedPackage: obj.Pkg().Path(),
		NamedName:    obj.Name(),
	}
	typeArgs := named.TypeArgs()
	if typeArgs != nil && typeArgs.Len() > 0 {
		info.TypeArgs = make([]GenericFieldTypeInfo, typeArgs.Len())
		for i := range typeArgs.Len() {
			descriptor, ok := describeGenericFuncType(typeArgs.At(i), paramIndex)
			if !ok {
				return GenericFieldTypeInfo{}, false
			}
			info.TypeArgs[i] = descriptor
		}
	} else if typeParams := named.TypeParams(); typeParams != nil {
		info.TypeArgs = make([]GenericFieldTypeInfo, typeParams.Len())
		for i := range typeParams.Len() {
			info.TypeArgs[i] = GenericFieldTypeInfo{
				Kind:         GenericFieldKindTypeArg,
				TypeArgIndex: i,
			}
		}
	}
	return info, true
}

// describeGenericFieldType walks a field's go/types.Type and produces
// a serialisable GenericFieldTypeInfo tree.
//
// Type parameters are recorded by their positional index; nested
// composites (slice, map, pointer, array, chan) recurse. Returns false
// for shapes that cannot be faithfully represented so the caller can
// skip the whole generic type rather than emit a malformed
// registration.
//
// Takes t (types.Type) which is the field type to describe.
// Takes paramIndex (map[*types.TypeParam]int) which resolves type
// parameters to their declaration index.
//
// Returns the descriptor and a bool indicating success.
func describeGenericFieldType(t types.Type, paramIndex map[*types.TypeParam]int) (GenericFieldTypeInfo, bool) {
	switch typeValue := t.(type) {
	case *types.TypeParam:
		return describeTypeParamField(typeValue, paramIndex)
	case *types.Basic:
		return describeBasicField(typeValue)
	case *types.Slice:
		return describeSingleElementField(typeValue.Elem(), paramIndex, GenericFieldKindSlice)
	case *types.Array:
		return describeArrayField(typeValue, paramIndex)
	case *types.Map:
		return describeMapField(typeValue, paramIndex)
	case *types.Pointer:
		return describeSingleElementField(typeValue.Elem(), paramIndex, GenericFieldKindPointer)
	case *types.Chan:
		return describeSingleElementField(typeValue.Elem(), paramIndex, GenericFieldKindChan)
	case *types.Interface:
		return GenericFieldTypeInfo{Kind: GenericFieldKindInterface}, true
	case *types.Named:
		if descriptor, ok := describeNamedOrAlias(typeValue, paramIndex); ok {
			return descriptor, true
		}
		return GenericFieldTypeInfo{}, false
	case *types.Alias:
		return describeAlias(typeValue, paramIndex)
	}
	return GenericFieldTypeInfo{}, false
}

// describeTypeParamField resolves a generic's type parameter to its
// positional descriptor.
//
// Takes typeParam (*types.TypeParam) which is the parameter reference.
// Takes paramIndex (map[*types.TypeParam]int) which maps parameters to
// declaration positions.
//
// Returns the descriptor and true when typeParam belongs to the owning
// generic's parameter list; otherwise zero and false.
func describeTypeParamField(typeParam *types.TypeParam, paramIndex map[*types.TypeParam]int) (GenericFieldTypeInfo, bool) {
	index, ok := paramIndex[typeParam]
	if !ok {
		return GenericFieldTypeInfo{}, false
	}
	return GenericFieldTypeInfo{
		Kind:         GenericFieldKindTypeArg,
		TypeArgIndex: index,
	}, true
}

// describeBasicField emits a FieldKindBasic descriptor for primitive
// types.
//
// Takes basicType (*types.Basic) which is the primitive type.
//
// Returns the descriptor and true for a representable kind; otherwise
// zero and false for untyped or unrepresentable kinds.
func describeBasicField(basicType *types.Basic) (GenericFieldTypeInfo, bool) {
	kind, ok := basicReflectKind(basicType)
	if !ok {
		return GenericFieldTypeInfo{}, false
	}
	return GenericFieldTypeInfo{
		Kind:      GenericFieldKindBasic,
		BasicKind: kind,
	}, true
}

// describeSingleElementField emits a descriptor with only an Element
// (slice, pointer, chan) using the caller-supplied kind so the same
// helper can serve each case.
//
// Takes element (types.Type) which is the inner type.
// Takes paramIndex (map[*types.TypeParam]int) which resolves type
// parameter references.
// Takes kind (GenericFieldKind) which selects the composite kind.
//
// Returns the descriptor and true on success, or zero and false when
// the inner type cannot be described.
func describeSingleElementField(
	element types.Type,
	paramIndex map[*types.TypeParam]int,
	kind GenericFieldKind,
) (GenericFieldTypeInfo, bool) {
	descriptor, ok := describeGenericFuncType(element, paramIndex)
	if !ok {
		return GenericFieldTypeInfo{}, false
	}
	return GenericFieldTypeInfo{
		Kind:    kind,
		Element: &descriptor,
	}, true
}

// describeArrayField emits a FieldKindArray descriptor, preserving the
// fixed length alongside the element descriptor.
//
// Takes arrayType (*types.Array) which is the array type.
// Takes paramIndex (map[*types.TypeParam]int) which resolves type
// parameter references.
//
// Returns the descriptor and true on success, or zero and false when
// the element type cannot be described.
func describeArrayField(arrayType *types.Array, paramIndex map[*types.TypeParam]int) (GenericFieldTypeInfo, bool) {
	element, ok := describeGenericFuncType(arrayType.Elem(), paramIndex)
	if !ok {
		return GenericFieldTypeInfo{}, false
	}
	return GenericFieldTypeInfo{
		Kind:        GenericFieldKindArray,
		Element:     &element,
		ArrayLength: int(arrayType.Len()),
	}, true
}

// describeMapField emits a FieldKindMap descriptor carrying Key and
// Element.
//
// Takes mapType (*types.Map) which is the map type.
// Takes paramIndex (map[*types.TypeParam]int) which resolves type
// parameter references.
//
// Returns the descriptor and true on success, or zero and false when
// either side cannot be represented.
func describeMapField(mapType *types.Map, paramIndex map[*types.TypeParam]int) (GenericFieldTypeInfo, bool) {
	key, keyOK := describeGenericFuncType(mapType.Key(), paramIndex)
	value, valOK := describeGenericFuncType(mapType.Elem(), paramIndex)
	if !keyOK || !valOK {
		return GenericFieldTypeInfo{}, false
	}
	return GenericFieldTypeInfo{
		Kind:    GenericFieldKindMap,
		Key:     &key,
		Element: &value,
	}, true
}

// basicReflectKind maps a go/types basic kind to its reflect.Kind
// counterpart, returning false for kinds the interpreter cannot
// currently represent (untyped basics are resolved before reaching
// this code path, so an unsupported result here indicates an
// unhandled edge case rather than a user-facing constraint).
//
// Takes basic (*types.Basic) which is the go/types basic type.
//
// Returns the reflect.Kind and a bool signalling match.
func basicReflectKind(basic *types.Basic) (reflect.Kind, bool) {
	switch basic.Kind() {
	case types.Bool:
		return reflect.Bool, true
	case types.Int:
		return reflect.Int, true
	case types.Int8:
		return reflect.Int8, true
	case types.Int16:
		return reflect.Int16, true
	case types.Int32:
		return reflect.Int32, true
	case types.Int64:
		return reflect.Int64, true
	case types.Uint:
		return reflect.Uint, true
	case types.Uint8:
		return reflect.Uint8, true
	case types.Uint16:
		return reflect.Uint16, true
	case types.Uint32:
		return reflect.Uint32, true
	case types.Uint64:
		return reflect.Uint64, true
	case types.Uintptr:
		return reflect.Uintptr, true
	case types.Float32:
		return reflect.Float32, true
	case types.Float64:
		return reflect.Float64, true
	case types.Complex64:
		return reflect.Complex64, true
	case types.Complex128:
		return reflect.Complex128, true
	case types.String:
		return reflect.String, true
	}
	return 0, false
}
