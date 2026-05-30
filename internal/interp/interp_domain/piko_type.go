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
	"iter"
	"reflect"
	"strconv"
	"strings"
)

// pikoType is piko's full-identity reflect.Type representation.
//
// Lives in the registerType bank, NEVER crosses into native reflect calls unwrapped.
// Carries source-level identity that Go's reflect package cannot synthesise, particularly
// named interface types (Go has no reflect.InterfaceOf). At the native-call boundary,
// nativeRType is what flows out; on the way back in the metadata is re-attached via the
// per-Service registry.
//
// Equality is by qualifiedName when set, otherwise by nativeRType identity. Two pikoTypes
// with the same qualifiedName are considered the same type even across package or
// bytecode-reload boundaries.
type pikoType struct {
	// nativeRType is the underlying Go reflect.Type handed to native code at the boundary,
	// always non-nil.
	//
	// For named interfaces this is the lossy reflect.TypeFor[any]() collapse; for everything
	// else it is the accurate type built by the existing typeToReflect machinery.
	nativeRType reflect.Type

	// elem is the element type for Pointer/Slice/Array/Chan/Map (map elem is the value
	// type). nil for non-composite types.
	elem *pikoType

	// key is the map key type, nil for non-maps.
	key *pikoType

	// qualifiedName is the rendered source-level type name including any
	// pointer/slice/array/map/chan/func decorations. Empty for anonymous types whose
	// identity is purely structural.
	//
	// Examples: "main.myiface", "*main.myiface", "[]main.myiface",
	// "map[string]main.myiface", "chan<- main.myiface".
	qualifiedName string

	// sourceName is the bare named-type identifier when the type is a named type, otherwise
	// empty.
	sourceName string

	// sourcePackage is the short package name (final segment of pkgPath).
	sourcePackage string

	// pkgPath is the full Go import path of the defining package, or empty for
	// unnamed/predeclared types.
	pkgPath string

	// methodNames lists the sorted method names of an interface type (or the methods of a
	// struct type's method set when populated). Used for Implements / AssignableTo
	// computations.
	methodNames []string

	// inTypes are the parameter types of a function signature.
	inTypes []*pikoType

	// outTypes are the result types of a function signature.
	outTypes []*pikoType

	// arrayLen is the array length, 0 for non-arrays.
	arrayLen int

	// kind is the underlying reflect.Kind (Pointer, Interface, Struct, Slice, Map, Array,
	// Chan, Func, basic kinds, etc.). Always set.
	kind reflect.Kind

	// chanDir is the channel direction, 0 for non-channels.
	chanDir reflect.ChanDir

	// variadic reports whether the final inType is a "..." parameter.
	variadic bool
}

// Name returns the source-level bare type name for named types.
//
// Matches reflect.Type.Name(). Empty string for unnamed types.
//
// Returns string which is the bare type name or "".
func (t *pikoType) Name() string {
	if t == nil {
		return ""
	}
	return t.sourceName
}

// String returns the package-qualified source-level type name.
//
// Includes any pointer, slice, array, map, channel, or func decorations. Matches
// reflect.Type.String().
//
// Returns string which is the rendered type name.
func (t *pikoType) String() string {
	if t == nil {
		return ""
	}
	if t.qualifiedName != "" {
		return t.qualifiedName
	}
	return t.nativeRType.String()
}

// PkgPath returns the full import path of the defining package.
//
// Empty for unnamed types. Matches reflect.Type.PkgPath().
//
// Returns string which is the import path or "".
func (t *pikoType) PkgPath() string {
	if t == nil {
		return ""
	}
	return t.pkgPath
}

// Kind returns the underlying reflect.Kind. Matches reflect.Type.Kind().
//
// Returns reflect.Kind which is the kind of the type.
func (t *pikoType) Kind() reflect.Kind {
	if t == nil {
		return reflect.Invalid
	}
	return t.kind
}

// NumMethod returns the count of accessible methods on the type.
//
// For interfaces this is the size of methodNames. For non-interfaces it delegates to
// nativeRType, which is accurate for piko-synth structs because nativeRType is the
// StructOf-built struct whose method set is empty (piko stores methods in its method
// table); the existing struct-method intercepts handle introspection elsewhere.
//
// Returns int which is the method count.
func (t *pikoType) NumMethod() int {
	if t == nil {
		return 0
	}
	if t.kind == reflect.Interface {
		return len(t.methodNames)
	}
	return t.nativeRType.NumMethod()
}

// Method returns the i-th method.
//
// For interfaces, synthesises a reflect.Method record from methodNames; for
// non-interfaces delegates to nativeRType.Method.
//
// Takes i (int) which is the method index.
//
// Returns reflect.Method which is the method descriptor.
func (t *pikoType) Method(i int) reflect.Method {
	if t == nil || i < 0 {
		return reflect.Method{}
	}
	if t.kind == reflect.Interface {
		if i >= len(t.methodNames) {
			return reflect.Method{}
		}
		return reflect.Method{
			Name:  t.methodNames[i],
			Index: i,
		}
	}
	return t.nativeRType.Method(i)
}

// MethodByName looks up a method by its source-level name.
//
// For interfaces, scans methodNames; for non-interfaces delegates to
// nativeRType.MethodByName.
//
// Takes name (string) which is the method name.
//
// Returns reflect.Method which is the method descriptor when found.
// Returns bool which reports whether a matching method was found.
func (t *pikoType) MethodByName(name string) (reflect.Method, bool) {
	if t == nil {
		return reflect.Method{}, false
	}
	if t.kind == reflect.Interface {
		for i, n := range t.methodNames {
			if n == name {
				return reflect.Method{Name: n, Index: i}, true
			}
		}
		return reflect.Method{}, false
	}
	return t.nativeRType.MethodByName(name)
}

// Implements reports whether t satisfies the interface u.
//
// When u is a pikoType interface the method-set check is done piko-side; otherwise it
// delegates to nativeRType.Implements.
//
// Takes u (*pikoType) which is the interface type to test against.
//
// Returns bool which reports whether t satisfies u.
func (t *pikoType) Implements(u *pikoType) bool {
	if t == nil || u == nil {
		return false
	}
	if u.kind != reflect.Interface {
		return false
	}
	if len(u.methodNames) == 0 {
		return true
	}
	if t.kind == reflect.Interface {
		return methodSetSatisfies(t.methodNames, u.methodNames)
	}
	return t.nativeRType.Implements(u.nativeRType)
}

// AssignableTo reports whether t is assignable to u.
//
// Computed via piko-side equality when both have qualified names; otherwise falls back to
// native reflect.
//
// Takes u (*pikoType) which is the destination type.
//
// Returns bool which reports whether the assignment is permitted.
func (t *pikoType) AssignableTo(u *pikoType) bool {
	if t == nil || u == nil {
		return false
	}
	if t.qualifiedName != "" && t.qualifiedName == u.qualifiedName {
		return true
	}
	if u.kind == reflect.Interface {
		return t.Implements(u)
	}
	return t.nativeRType.AssignableTo(u.nativeRType)
}

// ConvertibleTo reports whether t is convertible to u.
//
// Falls back to native reflect for the structural rules.
//
// Takes u (*pikoType) which is the destination type.
//
// Returns bool which reports whether the conversion is permitted.
func (t *pikoType) ConvertibleTo(u *pikoType) bool {
	if t == nil || u == nil {
		return false
	}
	if t.AssignableTo(u) {
		return true
	}
	return t.nativeRType.ConvertibleTo(u.nativeRType)
}

// Comparable reports whether values of t are comparable.
//
// Delegates to native reflect, which is correct for every kind.
//
// Returns bool which reports whether t is comparable.
func (t *pikoType) Comparable() bool {
	if t == nil {
		return false
	}
	return t.nativeRType.Comparable()
}

// Elem returns the element type for Pointer/Slice/Array/Chan/Map.
//
// For Map, this is the value type; use Key for the key type.
//
// Returns *pikoType which is the element type, or nil for invalid kinds.
func (t *pikoType) Elem() *pikoType {
	if t == nil {
		return nil
	}
	return t.elem
}

// Key returns the key type of a map.
//
// Returns *pikoType which is the map key type, or nil for non-maps.
func (t *pikoType) Key() *pikoType {
	if t == nil {
		return nil
	}
	return t.key
}

// Len returns the array length for Array kinds. Matches reflect.Type.Len.
//
// Returns int which is the array length, or 0 for non-arrays.
func (t *pikoType) Len() int {
	if t == nil {
		return 0
	}
	return t.arrayLen
}

// ChanDir returns the channel direction. Matches reflect.Type.ChanDir.
//
// Returns reflect.ChanDir which is the channel direction.
func (t *pikoType) ChanDir() reflect.ChanDir {
	if t == nil {
		return reflect.BothDir
	}
	return t.chanDir
}

// NumIn returns the function parameter count.
//
// Returns int which is the number of input parameters.
func (t *pikoType) NumIn() int {
	if t == nil {
		return 0
	}
	return len(t.inTypes)
}

// NumOut returns the function result count.
//
// Returns int which is the number of output results.
func (t *pikoType) NumOut() int {
	if t == nil {
		return 0
	}
	return len(t.outTypes)
}

// In returns the i-th function parameter type.
//
// Takes i (int) which is the parameter index.
//
// Returns *pikoType which is the parameter type, or nil when out of range.
func (t *pikoType) In(i int) *pikoType {
	if t == nil || i < 0 || i >= len(t.inTypes) {
		return nil
	}
	return t.inTypes[i]
}

// Out returns the i-th function result type.
//
// Takes i (int) which is the result index.
//
// Returns *pikoType which is the result type, or nil when out of range.
func (t *pikoType) Out(i int) *pikoType {
	if t == nil || i < 0 || i >= len(t.outTypes) {
		return nil
	}
	return t.outTypes[i]
}

// IsVariadic reports whether the final input is a "..." parameter.
//
// Returns bool which reports whether t is a variadic function type.
func (t *pikoType) IsVariadic() bool {
	if t == nil {
		return false
	}
	return t.variadic
}

// Ins returns an iterator over function parameter types.
//
// Returns iter.Seq[*pikoType] which yields each parameter in order.
func (t *pikoType) Ins() iter.Seq[*pikoType] {
	return func(yield func(*pikoType) bool) {
		if t == nil {
			return
		}
		for _, p := range t.inTypes {
			if !yield(p) {
				return
			}
		}
	}
}

// Outs returns an iterator over function result types.
//
// Returns iter.Seq[*pikoType] which yields each result in order.
func (t *pikoType) Outs() iter.Seq[*pikoType] {
	return func(yield func(*pikoType) bool) {
		if t == nil {
			return
		}
		for _, p := range t.outTypes {
			if !yield(p) {
				return
			}
		}
	}
}

// NumField returns the field count for Struct kinds.
//
// Delegates to nativeRType because pikoType does not carry field metadata; the existing
// piko-side sentinel-hiding intercept lives at a higher layer.
//
// Returns int which is the number of struct fields.
func (t *pikoType) NumField() int {
	if t == nil || t.kind != reflect.Struct {
		return 0
	}
	return t.nativeRType.NumField()
}

// Field returns the i-th struct field.
//
// Takes i (int) which is the field index.
//
// Returns reflect.StructField which is the field descriptor.
func (t *pikoType) Field(i int) reflect.StructField {
	if t == nil || t.kind != reflect.Struct {
		return reflect.StructField{}
	}
	return t.nativeRType.Field(i)
}

// Fields returns an iterator over each struct field.
//
// Returns iter.Seq[reflect.StructField] which yields each field in order.
func (t *pikoType) Fields() iter.Seq[reflect.StructField] {
	return func(yield func(reflect.StructField) bool) {
		if t == nil || t.kind != reflect.Struct {
			return
		}
		n := t.nativeRType.NumField()
		for i := range n {
			if !yield(t.nativeRType.Field(i)) {
				return
			}
		}
	}
}

// FieldByIndex returns the nested struct field for the index sequence.
//
// Takes index ([]int) which is the index path through embedded fields.
//
// Returns reflect.StructField which is the resolved field descriptor.
func (t *pikoType) FieldByIndex(index []int) reflect.StructField {
	if t == nil || t.kind != reflect.Struct {
		return reflect.StructField{}
	}
	return t.nativeRType.FieldByIndex(index)
}

// FieldByName looks up a struct field by name.
//
// Takes name (string) which is the field name.
//
// Returns reflect.StructField which is the field descriptor when found.
// Returns bool which reports whether a matching field was found.
func (t *pikoType) FieldByName(name string) (reflect.StructField, bool) {
	if t == nil || t.kind != reflect.Struct {
		return reflect.StructField{}, false
	}
	return t.nativeRType.FieldByName(name)
}

// FieldByNameFunc looks up a struct field whose name satisfies match.
//
// Takes match (func(string) bool) which tests each field name.
//
// Returns reflect.StructField which is the field descriptor when found.
// Returns bool which reports whether a matching field was found.
func (t *pikoType) FieldByNameFunc(match func(string) bool) (reflect.StructField, bool) {
	if t == nil || t.kind != reflect.Struct {
		return reflect.StructField{}, false
	}
	return t.nativeRType.FieldByNameFunc(match)
}

// Size returns the in-memory size in bytes. Delegates to nativeRType.
//
// Returns uintptr which is the size in bytes.
func (t *pikoType) Size() uintptr {
	if t == nil {
		return 0
	}
	return t.nativeRType.Size()
}

// Align returns the in-memory alignment in bytes.
//
// Returns int which is the alignment in bytes.
func (t *pikoType) Align() int {
	if t == nil {
		return 0
	}
	return t.nativeRType.Align()
}

// FieldAlign returns the alignment when used as a struct field.
//
// Returns int which is the field alignment in bytes.
func (t *pikoType) FieldAlign() int {
	if t == nil {
		return 0
	}
	return t.nativeRType.FieldAlign()
}

// Bits returns the bit width for numeric kinds.
//
// Returns int which is the bit width.
func (t *pikoType) Bits() int {
	if t == nil {
		return 0
	}
	return t.nativeRType.Bits()
}

// OverflowInt reports whether x cannot be represented in t.
//
// Takes x (int64) which is the candidate value.
//
// Returns bool which reports whether x overflows t.
func (t *pikoType) OverflowInt(x int64) bool {
	if t == nil {
		return false
	}
	return t.nativeRType.OverflowInt(x)
}

// OverflowUint reports whether x cannot be represented in t.
//
// Takes x (uint64) which is the candidate value.
//
// Returns bool which reports whether x overflows t.
func (t *pikoType) OverflowUint(x uint64) bool {
	if t == nil {
		return false
	}
	return t.nativeRType.OverflowUint(x)
}

// OverflowFloat reports whether x cannot be represented in t.
//
// Takes x (float64) which is the candidate value.
//
// Returns bool which reports whether x overflows t.
func (t *pikoType) OverflowFloat(x float64) bool {
	if t == nil {
		return false
	}
	return t.nativeRType.OverflowFloat(x)
}

// OverflowComplex reports whether x cannot be represented in t.
//
// Takes x (complex128) which is the candidate value.
//
// Returns bool which reports whether x overflows t.
func (t *pikoType) OverflowComplex(x complex128) bool {
	if t == nil {
		return false
	}
	return t.nativeRType.OverflowComplex(x)
}

// newPikoTypeFromReflect builds a pikoType from a native reflect.Type.
//
// Used at the native -> piko boundary when no registry entry exists for the incoming
// type. Structural fields (kind, elem, key) are populated from the reflect.Type;
// named-type metadata is left empty.
//
// Takes rt (reflect.Type) which is the native reflect.Type to wrap.
//
// Returns *pikoType with structural identity only; qualifiedName is empty for non-named
// types or set to rt.String() for named ones.
func newPikoTypeFromReflect(rt reflect.Type) *pikoType {
	if rt == nil {
		return nil
	}
	t := &pikoType{
		kind:        rt.Kind(),
		nativeRType: rt,
	}
	populateNamedTypeIdentity(t, rt)
	populateCompositeShape(t, rt)
	return t
}

// populateNamedTypeIdentity fills named-type metadata on t from rt.
//
// Sets sourceName, pkgPath, sourcePackage, and qualifiedName when rt is a named type,
// otherwise sets qualifiedName to the reflect.Type's natural rendering.
//
// Takes t (*pikoType) which receives the populated metadata.
// Takes rt (reflect.Type) which is the source reflect.Type.
func populateNamedTypeIdentity(t *pikoType, rt reflect.Type) {
	if rt.Name() != "" {
		t.sourceName = rt.Name()
		t.pkgPath = rt.PkgPath()
		if t.pkgPath != "" {
			t.sourcePackage = shortPackageName(t.pkgPath)
			t.qualifiedName = t.sourcePackage + "." + t.sourceName
		} else {
			t.qualifiedName = t.sourceName
		}
		return
	}
	t.qualifiedName = rt.String()
}

// populateCompositeShape fills composite-shape fields on t from rt.
//
// Populates elem, key, arrayLen, chanDir, inTypes, outTypes, variadic, and methodNames
// based on rt.Kind(). Scalar and struct kinds rely on nativeRType for their introspection
// so nothing is populated for them.
//
// Takes t (*pikoType) which receives the populated metadata.
// Takes rt (reflect.Type) which is the source reflect.Type.
func populateCompositeShape(t *pikoType, rt reflect.Type) {
	switch t.kind {
	case reflect.Pointer, reflect.Slice:
		t.elem = newPikoTypeFromReflect(rt.Elem())
	case reflect.Array:
		t.elem = newPikoTypeFromReflect(rt.Elem())
		t.arrayLen = rt.Len()
	case reflect.Chan:
		t.elem = newPikoTypeFromReflect(rt.Elem())
		t.chanDir = rt.ChanDir()
	case reflect.Map:
		t.elem = newPikoTypeFromReflect(rt.Elem())
		t.key = newPikoTypeFromReflect(rt.Key())
	case reflect.Func:
		t.variadic = rt.IsVariadic()
		t.inTypes = make([]*pikoType, rt.NumIn())
		for i := range t.inTypes {
			t.inTypes[i] = newPikoTypeFromReflect(rt.In(i))
		}
		t.outTypes = make([]*pikoType, rt.NumOut())
		for i := range t.outTypes {
			t.outTypes[i] = newPikoTypeFromReflect(rt.Out(i))
		}
	case reflect.Interface:
		n := rt.NumMethod()
		t.methodNames = make([]string, n)
		for i := range t.methodNames {
			t.methodNames[i] = rt.Method(i).Name
		}
	default:
	}
}

// newPikoTypeNamedInterface builds a pikoType for a user-declared named interface where
// Go's reflect cannot preserve identity. The nativeRType is the lossy
// reflect.TypeFor[any]() since reflect.InterfaceOf does not exist.
//
// Takes pkgPath, sourceName, methodNames. methodNames must be sorted by the caller to
// preserve a canonical iteration order.
//
// Returns the pikoType. Caller is responsible for registering it in the per-Service
// pikoTypeRegistry if cross-package identity is wanted.
func newPikoTypeNamedInterface(pkgPath, sourceName string, methodNames []string) *pikoType {
	short := shortPackageName(pkgPath)
	qualified := sourceName
	if short != "" {
		qualified = short + "." + sourceName
	}
	return &pikoType{
		kind:          reflect.Interface,
		qualifiedName: qualified,
		sourceName:    sourceName,
		sourcePackage: short,
		pkgPath:       pkgPath,
		methodNames:   methodNames,
		nativeRType:   reflect.TypeFor[any](),
	}
}

// shortPackageName returns the final '/' segment of pkgPath, matching Go's
// reflect.Type.String() convention for package-qualified names when the package directory
// and package name agree.
//
// Takes pkgPath (string) which is the Go import path.
//
// Returns the package short name, or "" when pkgPath is empty.
func shortPackageName(pkgPath string) string {
	if pkgPath == "" {
		return ""
	}
	if slash := strings.LastIndexByte(pkgPath, '/'); slash >= 0 {
		return pkgPath[slash+1:]
	}
	return pkgPath
}

// pikoTypeOfPointer constructs a pointer-to-elem pikoType.
//
// The native reflect.Type is built via reflect.PointerTo(elem.nativeRType) so the result
// is a real Go pointer type usable at the native boundary.
//
// Takes elem (*pikoType) which is the pointee type.
//
// Returns *pikoType which is the pointer type wrapping elem.
func pikoTypeOfPointer(elem *pikoType) *pikoType {
	return &pikoType{
		kind:          reflect.Pointer,
		qualifiedName: "*" + elem.qualifiedName,
		elem:          elem,
		nativeRType:   reflect.PointerTo(elem.nativeRType),
	}
}

// pikoTypeOfSlice constructs a slice-of-elem pikoType.
//
// Takes elem (*pikoType) which is the element type.
//
// Returns *pikoType which is the slice type wrapping elem.
func pikoTypeOfSlice(elem *pikoType) *pikoType {
	return &pikoType{
		kind:          reflect.Slice,
		qualifiedName: "[]" + elem.qualifiedName,
		elem:          elem,
		nativeRType:   reflect.SliceOf(elem.nativeRType),
	}
}

// pikoTypeOfArray constructs an array-of-elem pikoType.
//
// Takes length (int) which is the array length.
// Takes elem (*pikoType) which is the element type.
//
// Returns *pikoType which is the array type with the given length.
func pikoTypeOfArray(length int, elem *pikoType) *pikoType {
	return &pikoType{
		kind:          reflect.Array,
		qualifiedName: "[" + strconv.Itoa(length) + "]" + elem.qualifiedName,
		elem:          elem,
		arrayLen:      length,
		nativeRType:   reflect.ArrayOf(length, elem.nativeRType),
	}
}

// pikoTypeOfMap constructs a map[key]elem pikoType.
//
// Takes key (*pikoType) which is the map key type.
// Takes elem (*pikoType) which is the map value type.
//
// Returns *pikoType which is the map type.
func pikoTypeOfMap(key, elem *pikoType) *pikoType {
	return &pikoType{
		kind:          reflect.Map,
		qualifiedName: "map[" + key.qualifiedName + "]" + elem.qualifiedName,
		elem:          elem,
		key:           key,
		nativeRType:   reflect.MapOf(key.nativeRType, elem.nativeRType),
	}
}

// pikoTypeOfChan constructs a chan-of-elem pikoType with direction dir.
//
// Takes dir (reflect.ChanDir) which is the channel direction.
// Takes elem (*pikoType) which is the channel element type.
//
// Returns *pikoType which is the channel type.
func pikoTypeOfChan(dir reflect.ChanDir, elem *pikoType) *pikoType {
	prefix := "chan "
	switch dir {
	case reflect.RecvDir:
		prefix = "<-chan "
	case reflect.SendDir:
		prefix = "chan<- "
	case reflect.BothDir:
	}
	return &pikoType{
		kind:          reflect.Chan,
		qualifiedName: prefix + elem.qualifiedName,
		elem:          elem,
		chanDir:       dir,
		nativeRType:   reflect.ChanOf(dir, elem.nativeRType),
	}
}

// pikoTypeOfFunc constructs a function-signature pikoType.
//
// Takes in ([]*pikoType) which is the parameter type list.
// Takes out ([]*pikoType) which is the result type list.
// Takes variadic (bool) which reports whether the final parameter is a "..." variadic
// parameter.
//
// Returns *pikoType which is the function-signature type.
func pikoTypeOfFunc(in, out []*pikoType, variadic bool) *pikoType {
	inRT := make([]reflect.Type, len(in))
	for i, p := range in {
		inRT[i] = p.nativeRType
	}
	outRT := make([]reflect.Type, len(out))
	for i, p := range out {
		outRT[i] = p.nativeRType
	}
	return &pikoType{
		kind:          reflect.Func,
		qualifiedName: renderFuncQualifiedName(in, out, variadic),
		inTypes:       in,
		outTypes:      out,
		variadic:      variadic,
		nativeRType:   reflect.FuncOf(inRT, outRT, variadic),
	}
}

// renderFuncQualifiedName produces a Go-syntax rendering of a function signature suitable
// for pikoType.qualifiedName.
//
// Takes in ([]*pikoType) which is the parameter type list.
// Takes out ([]*pikoType) which is the result type list.
// Takes variadic (bool) which reports whether the final parameter is variadic.
//
// Returns string which is the Go-syntax rendering.
func renderFuncQualifiedName(in, out []*pikoType, variadic bool) string {
	var sb strings.Builder
	sb.WriteString("func(")
	for i, p := range in {
		if i > 0 {
			sb.WriteString(", ")
		}
		if variadic && i == len(in)-1 && p.kind == reflect.Slice {
			sb.WriteString("...")
			sb.WriteString(p.elem.qualifiedName)
		} else {
			sb.WriteString(p.qualifiedName)
		}
	}
	sb.WriteByte(')')
	switch len(out) {
	case 0:
	case 1:
		sb.WriteByte(' ')
		sb.WriteString(out[0].qualifiedName)
	default:
		sb.WriteString(" (")
		for i, p := range out {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(p.qualifiedName)
		}
		sb.WriteByte(')')
	}
	return sb.String()
}

// methodSetSatisfies reports whether available contains every required name.
//
// Both slices must be sorted; the check is O(len(available) + len(required)).
//
// Takes available ([]string) which is the sorted method-set of the subject type.
// Takes required ([]string) which is the sorted method-set the subject must satisfy.
//
// Returns bool which reports whether every required name is present.
func methodSetSatisfies(available, required []string) bool {
	if len(required) == 0 {
		return true
	}
	if len(available) < len(required) {
		return false
	}
	i, j := 0, 0
	for i < len(available) && j < len(required) {
		switch {
		case available[i] < required[j]:
			i++
		case available[i] == required[j]:
			i++
			j++
		default:
			return false
		}
	}
	return j == len(required)
}
