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
	"reflect"
	"unsafe"

	"piko.sh/piko/wdk/safeconv"
)

// generalConstantKind identifies the source of a general constant,
// enabling reconstruction from a descriptor without the original
// reflect.Value.
type generalConstantKind uint8

const (
	// generalConstantPackageSymbol is a package-qualified symbol such as
	// fmt.Println, resolved via SymbolRegistry.Lookup.
	generalConstantPackageSymbol generalConstantKind = iota

	// generalConstantNamedTypeZero is a zero value for a named type
	// from a registered package, resolved via
	// SymbolRegistry.ZeroValueForType.
	generalConstantNamedTypeZero

	// generalConstantCompositeZero is a zero value for a composite
	// type (array or struct), reconstructed via
	// reflect.New(reconstructedType).Elem().
	generalConstantCompositeZero
)

// generalConstantDescriptor records how to reconstruct a general
// constant from its serialised form.
type generalConstantDescriptor struct {
	// packagePath is the import path of the package containing
	// the symbol or type.
	packagePath string

	// symbolName is the name of the symbol within its package.
	symbolName string

	// typeDesc is the type descriptor used for composite zero
	// value reconstruction.
	typeDesc typeDescriptor

	// kind identifies which reconstruction strategy to use.
	kind generalConstantKind
}

// typeDescKind identifies the structural category of a type
// descriptor.
type typeDescKind uint8

const (
	// typeDescBasic is a primitive type such as int, string,
	// or bool.
	typeDescBasic typeDescKind = iota

	// typeDescNamed is a named type from a registered package.
	typeDescNamed //nolint:revive // naming is intentional

	// typeDescPtr is a pointer type.
	typeDescPtr //nolint:revive // naming is intentional

	// typeDescSlice is a slice type.
	typeDescSlice //nolint:revive // naming is intentional

	// typeDescArray is a fixed-length array type.
	typeDescArray //nolint:revive // naming is intentional

	// typeDescMap is a map type with key and value descriptors.
	typeDescMap //nolint:revive // naming is intentional

	// typeDescChan is a channel type with a direction.
	typeDescChan //nolint:revive // naming is intentional

	// typeDescFunc is a function type with parameter and result
	// descriptors.
	typeDescFunc //nolint:revive // naming is intentional

	// typeDescStruct is a struct type with field descriptors.
	typeDescStruct //nolint:revive // naming is intentional

	// typeDescInterface is an interface type used as a
	// placeholder for unresolvable or nil types.
	typeDescInterface //nolint:revive // naming is intentional
)

// typeDescriptor is a serialisable description of a reflect.Type that
// can be stored in FlatBuffers and later reconstructed via
// descriptorToReflectType.
type typeDescriptor struct {
	// packagePath is the import path of the package that owns
	// this named type, empty for unnamed types.
	packagePath string

	// name is the symbol name of the type within its package.
	name string

	// element is the descriptor for the element type of
	// pointers, slices, arrays, and channels.
	element *typeDescriptor

	// key is the descriptor for the key type of a map.
	key *typeDescriptor

	// value is the descriptor for the value type of a map.
	value *typeDescriptor

	// fields holds the struct field descriptors when the kind
	// is typeDescStruct.
	fields []typeDescField

	// params holds the parameter type descriptors when the
	// kind is typeDescFunc.
	params []typeDescriptor

	// results holds the return type descriptors when the kind
	// is typeDescFunc.
	results []typeDescriptor

	// length is the fixed size of an array type.
	length int

	// dir is the channel direction, stored as an int cast from
	// reflect.ChanDir.
	dir int

	// basicKind is the reflect.Kind value stored as uint8 for
	// primitive types.
	basicKind uint8

	// kind identifies the structural category of the type
	// descriptor.
	kind typeDescKind

	// isVariadic indicates whether a function type accepts
	// variadic arguments in its last parameter.
	isVariadic bool
}

// typeDescField describes a single struct field within a type
// descriptor.
type typeDescField struct {
	// name is the Go identifier of the struct field.
	name string

	// tag is the raw struct tag string for the field.
	tag string

	// packagePath is the import path of the package that
	// defines the field, empty for exported fields.
	packagePath string

	// typ is the type descriptor for the field's type.
	typ typeDescriptor
}

// reflectTypeToDescriptor converts a reflect.Type into a serialisable
// typeDescriptor.
//
// Takes reflectType (reflect.Type) which is the runtime type to
// describe.
//
// Returns typeDescriptor which contains enough information to
// reconstruct the type via descriptorToReflectType.
func reflectTypeToDescriptor(reflectType reflect.Type) typeDescriptor {
	b := &descriptorBuilder{visiting: make(map[reflect.Type]bool)}
	return b.typeToDescriptor(reflectType)
}

// descriptorBuilder recursively converts reflect.Type to typeDescriptor
// with cycle detection to prevent stack overflow on recursive struct types.
type descriptorBuilder struct {
	// visiting tracks struct types currently being converted
	// to detect and break recursive cycles.
	visiting map[reflect.Type]bool
}

// typeToDescriptor converts a single reflect.Type, tracking
// visited struct types to break cycles.
//
// Takes reflectType (reflect.Type) which is the runtime type
// to convert.
//
// Returns typeDescriptor which is the serialisable description
// of the type.
func (b *descriptorBuilder) typeToDescriptor(reflectType reflect.Type) typeDescriptor {
	if reflectType == nil {
		return typeDescriptor{kind: typeDescInterface}
	}

	switch reflectType.Kind() {
	case reflect.Pointer:
		return typeDescriptor{kind: typeDescPtr, element: new(b.typeToDescriptor(reflectType.Elem()))}

	case reflect.Slice:
		return typeDescriptor{kind: typeDescSlice, element: new(b.typeToDescriptor(reflectType.Elem()))}

	case reflect.Array:
		return typeDescriptor{kind: typeDescArray, element: new(b.typeToDescriptor(reflectType.Elem())), length: reflectType.Len()}

	case reflect.Map:
		return typeDescriptor{kind: typeDescMap, key: new(b.typeToDescriptor(reflectType.Key())), value: new(b.typeToDescriptor(reflectType.Elem()))}

	case reflect.Chan:
		return typeDescriptor{kind: typeDescChan, element: new(b.typeToDescriptor(reflectType.Elem())), dir: int(reflectType.ChanDir())}

	case reflect.Func:
		return b.funcToDescriptor(reflectType)

	case reflect.Struct:
		return b.structToDescriptor(reflectType)

	case reflect.Interface:
		return typeDescriptor{kind: typeDescInterface}

	default:
		return typeDescriptor{kind: typeDescBasic, basicKind: safeconv.MustUintToUint8(uint(reflectType.Kind()))}
	}
}

// funcToDescriptor converts a function reflect.Type to a
// typeDescriptor.
//
// Takes reflectType (reflect.Type) which is the function type
// to convert.
//
// Returns typeDescriptor which contains the parameter and
// result type descriptors.
func (b *descriptorBuilder) funcToDescriptor(reflectType reflect.Type) typeDescriptor {
	paramDescriptors := make([]typeDescriptor, reflectType.NumIn())
	for i := range reflectType.NumIn() {
		paramDescriptors[i] = b.typeToDescriptor(reflectType.In(i))
	}
	resultDescriptors := make([]typeDescriptor, reflectType.NumOut())
	for i := range reflectType.NumOut() {
		resultDescriptors[i] = b.typeToDescriptor(reflectType.Out(i))
	}
	return typeDescriptor{
		kind:       typeDescFunc,
		params:     paramDescriptors,
		results:    resultDescriptors,
		isVariadic: reflectType.IsVariadic(),
	}
}

// structToDescriptor converts a struct reflect.Type to a
// typeDescriptor. Tracks the struct in the visiting set to
// detect cycles; if a struct is encountered while already
// being processed, an interface placeholder is returned to
// break the cycle.
//
// Takes reflectType (reflect.Type) which is the struct type
// to convert.
//
// Returns typeDescriptor which contains the field descriptors
// for the struct.
func (b *descriptorBuilder) structToDescriptor(reflectType reflect.Type) typeDescriptor {
	if b.visiting[reflectType] {
		return typeDescriptor{kind: typeDescInterface}
	}
	b.visiting[reflectType] = true
	defer delete(b.visiting, reflectType)

	fieldDescriptors := make([]typeDescField, reflectType.NumField())
	for i := range reflectType.NumField() {
		field := reflectType.Field(i)
		fieldDescriptors[i] = typeDescField{
			name:        field.Name,
			tag:         string(field.Tag),
			packagePath: field.PkgPath,
			typ:         b.typeToDescriptor(field.Type),
		}
	}
	return typeDescriptor{kind: typeDescStruct, fields: fieldDescriptors}
}

// descriptorToReflectType reconstructs a reflect.Type from a
// typeDescriptor using the SymbolRegistry to resolve named types.
//
// Takes descriptor (typeDescriptor) which describes the type to
// reconstruct.
// Takes registry (*SymbolRegistry) which provides access to
// registered package symbols for named type lookups.
//
// Returns reflect.Type which is the reconstructed runtime type.
// Returns error when a named type cannot be found in the registry.
func descriptorToReflectType(descriptor typeDescriptor, registry *SymbolRegistry) (reflect.Type, error) {
	switch descriptor.kind {
	case typeDescBasic:
		return basicKindToReflect(reflect.Kind(descriptor.basicKind)), nil
	case typeDescNamed:
		return resolveNamedType(descriptor, registry)
	case typeDescPtr:
		return resolveElementContainer(descriptor, registry, reflect.PointerTo)
	case typeDescSlice:
		return resolveElementContainer(descriptor, registry, reflect.SliceOf)
	case typeDescArray:
		return resolveArrayType(descriptor, registry)
	case typeDescMap:
		return resolveMapType(descriptor, registry)
	case typeDescChan:
		return resolveChanType(descriptor, registry)
	case typeDescFunc:
		return reconstructFuncType(descriptor, registry)
	case typeDescStruct:
		return reconstructStructType(descriptor, registry)
	default:
		return reflect.TypeFor[any](), nil
	}
}

// resolveElementContainer resolves the element type of a
// descriptor and wraps it with the given constructor.
//
// Takes descriptor (typeDescriptor) which describes the
// container type to resolve.
// Takes registry (*SymbolRegistry) which provides named type
// lookups.
// Takes wrap (func(reflect.Type) reflect.Type) which
// constructs the outer type from the resolved element.
//
// Returns reflect.Type which is the wrapped container type.
// Returns error when the element type cannot be resolved.
func resolveElementContainer(
	descriptor typeDescriptor,
	registry *SymbolRegistry,
	wrap func(reflect.Type) reflect.Type,
) (reflect.Type, error) {
	elemType, err := descriptorToReflectType(*descriptor.element, registry)
	if err != nil {
		return nil, err
	}
	return wrap(elemType), nil
}

// resolveArrayType rebuilds an array reflect.Type from a
// descriptor.
//
// Takes descriptor (typeDescriptor) which describes the array
// type including its length and element type.
// Takes registry (*SymbolRegistry) which provides named type
// lookups.
//
// Returns reflect.Type which is the reconstructed array type.
// Returns error when the element type cannot be resolved.
func resolveArrayType(descriptor typeDescriptor, registry *SymbolRegistry) (reflect.Type, error) {
	elemType, err := descriptorToReflectType(*descriptor.element, registry)
	if err != nil {
		return nil, err
	}
	return reflect.ArrayOf(descriptor.length, elemType), nil
}

// resolveMapType rebuilds a map reflect.Type from a
// descriptor.
//
// Takes descriptor (typeDescriptor) which describes the map
// type including its key and value types.
// Takes registry (*SymbolRegistry) which provides named type
// lookups.
//
// Returns reflect.Type which is the reconstructed map type.
// Returns error when the key or value type cannot be resolved.
func resolveMapType(descriptor typeDescriptor, registry *SymbolRegistry) (reflect.Type, error) {
	keyType, err := descriptorToReflectType(*descriptor.key, registry)
	if err != nil {
		return nil, err
	}
	valueType, err := descriptorToReflectType(*descriptor.value, registry)
	if err != nil {
		return nil, err
	}
	return reflect.MapOf(keyType, valueType), nil
}

// resolveChanType rebuilds a channel reflect.Type from a
// descriptor.
//
// Takes descriptor (typeDescriptor) which describes the
// channel type including its direction and element type.
// Takes registry (*SymbolRegistry) which provides named type
// lookups.
//
// Returns reflect.Type which is the reconstructed channel
// type.
// Returns error when the element type cannot be resolved.
func resolveChanType(descriptor typeDescriptor, registry *SymbolRegistry) (reflect.Type, error) {
	elemType, err := descriptorToReflectType(*descriptor.element, registry)
	if err != nil {
		return nil, err
	}
	return reflect.ChanOf(reflect.ChanDir(descriptor.dir), elemType), nil
}

// reconstructFuncType rebuilds a function reflect.Type from a
// descriptor.
//
// Takes descriptor (typeDescriptor) which describes the
// function type including its parameter and result types.
// Takes registry (*SymbolRegistry) which provides named type
// lookups.
//
// Returns reflect.Type which is the reconstructed function
// type.
// Returns error when any parameter or result type cannot be
// resolved.
func reconstructFuncType(descriptor typeDescriptor, registry *SymbolRegistry) (reflect.Type, error) {
	paramTypes := make([]reflect.Type, len(descriptor.params))
	for i := range descriptor.params {
		paramType, err := descriptorToReflectType(descriptor.params[i], registry)
		if err != nil {
			return nil, err
		}
		paramTypes[i] = paramType
	}
	resultTypes := make([]reflect.Type, len(descriptor.results))
	for i := range descriptor.results {
		resultType, err := descriptorToReflectType(descriptor.results[i], registry)
		if err != nil {
			return nil, err
		}
		resultTypes[i] = resultType
	}
	return reflect.FuncOf(paramTypes, resultTypes, descriptor.isVariadic), nil
}

// reconstructStructType rebuilds a struct reflect.Type from a
// descriptor.
//
// Takes descriptor (typeDescriptor) which describes the
// struct type including its field names, types, and tags.
// Takes registry (*SymbolRegistry) which provides named type
// lookups.
//
// Returns reflect.Type which is the reconstructed struct
// type.
// Returns error when any field type cannot be resolved.
func reconstructStructType(descriptor typeDescriptor, registry *SymbolRegistry) (reflect.Type, error) {
	structFields := make([]reflect.StructField, len(descriptor.fields))
	for i := range descriptor.fields {
		fieldType, err := descriptorToReflectType(descriptor.fields[i].typ, registry)
		if err != nil {
			return nil, err
		}
		structFields[i] = reflect.StructField{
			Name:    descriptor.fields[i].name,
			Type:    fieldType,
			Tag:     reflect.StructTag(descriptor.fields[i].tag),
			PkgPath: descriptor.fields[i].packagePath,
		}
	}
	return reflect.StructOf(structFields), nil
}

// resolveNamedType looks up a named type in the SymbolRegistry
// and extracts its element type from the (*T)(nil) registration
// pattern.
//
// Takes descriptor (typeDescriptor) which identifies the named
// type by its package path and name.
// Takes registry (*SymbolRegistry) which provides the type
// lookup.
//
// Returns reflect.Type which is the resolved named type.
// Returns error when the named type is not found in the
// registry.
func resolveNamedType(descriptor typeDescriptor, registry *SymbolRegistry) (reflect.Type, error) {
	value, ok := registry.Lookup(descriptor.packagePath, descriptor.name)
	if !ok {
		return nil, fmt.Errorf("named type %s.%s not found in symbol registry", descriptor.packagePath, descriptor.name)
	}
	reflectType := value.Type()
	if reflectType.Kind() == reflect.Pointer {
		return reflectType.Elem(), nil
	}
	return reflectType, nil
}

// basicKindToReflect converts a reflect.Kind for basic types
// to a reflect.Type.
//
// Takes kind (reflect.Kind) which is the basic type kind to
// convert.
//
// Returns reflect.Type which is the corresponding runtime
// type, or any for unsupported kinds.
func basicKindToReflect(kind reflect.Kind) reflect.Type {
	switch kind {
	case reflect.Bool:
		return reflect.TypeFor[bool]()
	case reflect.Int:
		return reflect.TypeFor[int]()
	case reflect.Int8:
		return reflect.TypeFor[int8]()
	case reflect.Int16:
		return reflect.TypeFor[int16]()
	case reflect.Int32:
		return reflect.TypeFor[int32]()
	case reflect.Int64:
		return reflect.TypeFor[int64]()
	case reflect.Uint:
		return reflect.TypeFor[uint]()
	case reflect.Uint8:
		return reflect.TypeFor[uint8]()
	case reflect.Uint16:
		return reflect.TypeFor[uint16]()
	case reflect.Uint32:
		return reflect.TypeFor[uint32]()
	case reflect.Uint64:
		return reflect.TypeFor[uint64]()
	case reflect.Uintptr:
		return reflect.TypeFor[uintptr]()
	case reflect.Float32:
		return reflect.TypeFor[float32]()
	case reflect.Float64:
		return reflect.TypeFor[float64]()
	case reflect.Complex64:
		return reflect.TypeFor[complex64]()
	case reflect.Complex128:
		return reflect.TypeFor[complex128]()
	case reflect.String:
		return reflect.TypeFor[string]()
	case reflect.UnsafePointer:
		return reflect.TypeFor[unsafe.Pointer]()
	default:
		return reflect.TypeFor[any]()
	}
}

// reconstructGeneralConstant rebuilds a reflect.Value from its
// serialised descriptor using the SymbolRegistry.
//
// Takes descriptor (generalConstantDescriptor) which describes the
// constant to reconstruct.
// Takes registry (*SymbolRegistry) which provides access to
// registered package symbols.
//
// Returns reflect.Value which is the reconstructed runtime value.
// Returns error when the referenced symbol or type cannot be found.
func reconstructGeneralConstant(descriptor generalConstantDescriptor, registry *SymbolRegistry) (reflect.Value, error) {
	switch descriptor.kind {
	case generalConstantPackageSymbol:
		value, ok := registry.Lookup(descriptor.packagePath, descriptor.symbolName)
		if !ok {
			return reflect.Value{}, fmt.Errorf("symbol %s.%s not found in registry", descriptor.packagePath, descriptor.symbolName)
		}
		return value, nil

	case generalConstantNamedTypeZero:
		zeroValue, ok := registry.ZeroValueForType(descriptor.packagePath, descriptor.symbolName)
		if !ok {
			return reflect.Value{}, fmt.Errorf("named type %s.%s not found in registry", descriptor.packagePath, descriptor.symbolName)
		}
		return zeroValue, nil

	case generalConstantCompositeZero:
		reflectType, err := descriptorToReflectType(descriptor.typeDesc, registry)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("reconstructing composite type: %w", err)
		}
		return reflect.New(reflectType).Elem(), nil

	default:
		return reflect.Value{}, fmt.Errorf("unknown general constant kind: %d", descriptor.kind)
	}
}
