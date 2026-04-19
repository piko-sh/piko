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

package interp_link

import "reflect"

// GenericFieldKind classifies a single FieldType node. The registry
// synthesiser uses the kind to build both the go/types representation
// (for type-checking user code that instantiates the generic) and the
// reflect.Type (at runtime when the interpreter encounters an
// instantiation with user-supplied type arguments).
type GenericFieldKind uint8

const (
	// FieldKindBasic is a primitive type such as int, string, or bool.
	FieldKindBasic GenericFieldKind = iota

	// FieldKindTypeArg references the Nth type parameter of the
	// enclosing LinkedGenericType (e.g. the T in Item T).
	FieldKindTypeArg

	// FieldKindSlice is []Element.
	FieldKindSlice

	// FieldKindArray is [Length]Element.
	FieldKindArray

	// FieldKindMap is map[Key]Element.
	FieldKindMap

	// FieldKindPointer is *Element.
	FieldKindPointer

	// FieldKindChan is a channel whose element is Element; channel
	// direction information is not preserved because it rarely matters
	// for the interpreter's type-checking purposes.
	FieldKindChan

	// FieldKindInterface collapses any interface type down to the
	// empty interface for reflect purposes, matching how the compiler
	// treats interface-typed fields today.
	FieldKindInterface

	// FieldKindNamed references a concrete non-generic named type from
	// another package. The registry resolves it via its own name-based
	// lookup during synthesis.
	FieldKindNamed

	// FieldKindNamedGeneric references an instantiation of another
	// LinkedGenericType (or an aliased generic), with TypeArgs giving
	// the substitution for each of its declared type parameters.
	FieldKindNamedGeneric

	// FieldKindError is the Go built-in `error` interface. Keeping it
	// as its own kind avoids serialising the interface method set.
	FieldKindError
)

// GenericFieldType is a type-tree node describing one field's type in
// a LinkedGenericType. It is emitted verbatim by `piko extract generate`
// into the generated symbol file, so its representation must be
// JSON-style plain data: no closures, no reflect.Type values captured
// at extract time.
type GenericFieldType struct {
	// Element is the inner type for Slice, Array, Pointer, Chan, and
	// the value side of Map. Nil for leaf kinds.
	Element *GenericFieldType

	// Key is the key type for FieldKindMap; nil otherwise.
	Key *GenericFieldType

	// NamedPackage is the import path for FieldKindNamed and
	// FieldKindNamedGeneric.
	NamedPackage string

	// NamedName is the exported type name for FieldKindNamed and
	// FieldKindNamedGeneric.
	NamedName string

	// TypeArgs are the per-position type arguments for
	// FieldKindNamedGeneric (empty for other kinds). Each element
	// recursively describes one type argument in declaration order.
	TypeArgs []GenericFieldType

	// ArrayLength is the fixed size of FieldKindArray entries.
	ArrayLength int

	// TypeArgIndex is the 0-based position of the referenced type
	// parameter for FieldKindTypeArg.
	TypeArgIndex int

	// Kind classifies this node.
	Kind GenericFieldKind

	// BasicKind is the reflect.Kind for FieldKindBasic.
	BasicKind reflect.Kind
}

// GenericField describes a single exported field of a linked generic
// type, including its tag and resolved type-tree.
type GenericField struct {
	// Name is the exported Go identifier.
	Name string

	// Tag is the raw struct tag, without surrounding backticks.
	Tag string

	// FieldType is the serialisable type tree for the field. The
	// registry builds both a go/types.Type and a reflect.Type from it.
	FieldType GenericFieldType

	// Exported carries the Go export visibility so the registry can
	// populate reflect.StructField.PkgPath correctly for hidden
	// fields. Today every registered symbol is exported, but keeping
	// this explicit avoids a future mismatch when the extract tool
	// learns to surface sealed types.
	Exported bool
}

// LinkedGenericType is the sentinel value registered in place of a
// generic Go type. The interpreter recognises it during package
// synthesis and builds a generic types.Named whose field types
// reference the declared type parameters, so user code writing
// pkg.Generic[Concrete] type-checks and instantiates into a proper
// reflect.Type at runtime.
type LinkedGenericType struct {
	// Name mirrors the exported type's identifier (e.g. "SearchResult").
	//
	// It is duplicated here because the registry walks the sentinel
	// outside the map entry that recorded the key.
	Name string

	// Fields describes each exported field of the generic type in
	// declaration order.
	Fields []GenericField

	// TypeArgCount is the number of type parameters the generic
	// declares. The interpreter accepts pkg.Name[T1, ..., Tn]
	// instantiations where n == TypeArgCount.
	TypeArgCount int
}

// WrapType constructs a LinkedGenericType for extract's codegen path.
// Keeping the constructor explicit keeps the emitted Go source
// readable and lets us add validation later without changing the
// generated files' shape.
//
// Takes name (string) which is the exported type identifier.
// Takes typeArgCount (int) which is the number of type parameters.
// Takes fields ([]GenericField) which describe the struct layout.
//
// Returns a LinkedGenericType suitable for reflect.ValueOf wrapping.
func WrapType(name string, typeArgCount int, fields []GenericField) LinkedGenericType {
	return LinkedGenericType{
		Name:         name,
		Fields:       fields,
		TypeArgCount: typeArgCount,
	}
}
