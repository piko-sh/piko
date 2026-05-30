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
	"reflect"
)

// pikoNamedInterfaceWrapper wraps reflect.Type so user-declared named interface types
// report the source-level identity Go's reflect can't preserve natively (because there is
// no reflect.InterfaceOf).
//
// The wrapper embeds the underlying reflect.Type so it satisfies the reflect.Type
// interface and inherits every method whose behaviour doesn't depend on source-level
// naming (Size, Align, Comparable, the numeric overflow checks, the
// struct/function/channel introspectors). The identity-bearing methods (Name, String,
// PkgPath, Kind, NumMethod, Method, MethodByName, Implements, AssignableTo,
// ConvertibleTo, Elem, Key) are overridden to consult the pikoType.
//
// Caveat: code that passes a wrapper into native reflect functions expecting the
// unexported *rtype (e.g. reflect.PointerTo, reflect.SliceOf, reflect.MapOf,
// reflect.MakeMap, reflect.New) will panic because those internally do `t.(*rtype)`. Such
// call sites must be intercepted at the piko level before they reach native reflect. For
// introspection-only paths (the common case for tests 918-925 and most user code that
// asks for type identity), the wrapper works.
type pikoNamedInterfaceWrapper struct {
	reflect.Type

	// piko carries the source-level type identity overriding the embedded reflect.Type for
	// identity-bearing methods.
	piko *pikoType
}

// Name returns the source-level bare name of the wrapped type.
//
// For pointer/slice/etc. wrappers the bare name is empty (Go convention); for the inner
// interface it's e.g. "myiface".
//
// Returns string which is the bare name.
func (w pikoNamedInterfaceWrapper) Name() string {
	if w.piko == nil {
		return w.Type.Name()
	}
	return w.piko.Name()
}

// String returns the package-qualified source-level rendering.
//
// Examples: "*main.myiface" or "main.myiface".
//
// Returns string which is the rendered name.
func (w pikoNamedInterfaceWrapper) String() string {
	if w.piko == nil {
		return w.Type.String()
	}
	return w.piko.String()
}

// PkgPath returns the full import path of the defining package.
//
// Returns string which is the import path.
func (w pikoNamedInterfaceWrapper) PkgPath() string {
	if w.piko == nil {
		return w.Type.PkgPath()
	}
	return w.piko.PkgPath()
}

// Elem returns the element type for Pointer/Slice/Array/Chan/Map.
//
// Re-wrapped with the corresponding pikoType so chained introspection like
// `t.Elem().String()` reports the source name.
//
// Returns reflect.Type which is the wrapped element type.
func (w pikoNamedInterfaceWrapper) Elem() reflect.Type {
	if w.piko == nil || w.piko.Elem() == nil {
		return w.Type.Elem()
	}
	return newPikoNamedInterfaceWrapper(w.Type.Elem(), w.piko.Elem())
}

// Key returns the map key type, re-wrapped via pikoType.Key.
//
// Returns reflect.Type which is the wrapped key type.
func (w pikoNamedInterfaceWrapper) Key() reflect.Type {
	if w.piko == nil || w.piko.Key() == nil {
		return w.Type.Key()
	}
	return newPikoNamedInterfaceWrapper(w.Type.Key(), w.piko.Key())
}

// NumMethod returns the source-level method count.
//
// For interfaces the count is the size of pikoType.methodNames; for non-interface
// wrappers it delegates to the embedded reflect.Type.
//
// Returns int which is the method count.
func (w pikoNamedInterfaceWrapper) NumMethod() int {
	if w.piko == nil {
		return w.Type.NumMethod()
	}
	return w.piko.NumMethod()
}

// Method returns the i-th method record.
//
// For interfaces, synthesises from pikoType.methodNames so user code observes the same
// method names native Go would report.
//
// Takes i (int) which is the method index.
//
// Returns reflect.Method which is the method record.
func (w pikoNamedInterfaceWrapper) Method(i int) reflect.Method {
	if w.piko == nil {
		return w.Type.Method(i)
	}
	return w.piko.Method(i)
}

// MethodByName looks up a method by name.
//
// Interface lookup walks pikoType.methodNames; non-interface lookup delegates to the
// embedded reflect.Type.
//
// Takes name (string) which is the method name.
//
// Returns reflect.Method which is the resolved method record.
// Returns bool which is true when the method exists.
func (w pikoNamedInterfaceWrapper) MethodByName(name string) (reflect.Method, bool) {
	if w.piko == nil {
		return w.Type.MethodByName(name)
	}
	return w.piko.MethodByName(name)
}

// Implements reports whether the wrapped type satisfies u.
//
// Computed piko-side when both are wrappers (or u is wrappable); falls back to native
// reflect.Implements otherwise.
//
// Takes u (reflect.Type) which is the target interface type.
//
// Returns bool which is true when the wrapped type satisfies u.
func (w pikoNamedInterfaceWrapper) Implements(u reflect.Type) bool {
	if w.piko == nil {
		return w.Type.Implements(u)
	}
	if other, ok := unwrapPikoNamedInterfaceWrapper(u); ok && other.piko != nil {
		return w.piko.Implements(other.piko)
	}
	if u.Kind() == reflect.Interface && u.NumMethod() == 0 {
		return true
	}
	return w.Type.Implements(u)
}

// AssignableTo reports whether a value of the wrapped type is assignable.
//
// Same delegation rules as Implements apply.
//
// Takes u (reflect.Type) which is the target type.
//
// Returns bool which is true when the wrapped type is assignable to u.
func (w pikoNamedInterfaceWrapper) AssignableTo(u reflect.Type) bool {
	if w.piko == nil {
		return w.Type.AssignableTo(u)
	}
	if other, ok := unwrapPikoNamedInterfaceWrapper(u); ok && other.piko != nil {
		return w.piko.AssignableTo(other.piko)
	}
	return w.Type.AssignableTo(u)
}

// ConvertibleTo reports whether a value of the wrapped type is convertible.
//
// Takes u (reflect.Type) which is the target type.
//
// Returns bool which is true when the wrapped type is convertible to u.
func (w pikoNamedInterfaceWrapper) ConvertibleTo(u reflect.Type) bool {
	if w.piko == nil {
		return w.Type.ConvertibleTo(u)
	}
	if other, ok := unwrapPikoNamedInterfaceWrapper(u); ok && other.piko != nil {
		return w.piko.ConvertibleTo(other.piko)
	}
	return w.Type.ConvertibleTo(u)
}

// newPikoNamedInterfaceWrapper builds a wrapper around nativeRType.
//
// The source-level identity is described by piko. nativeRType is usually
// reflect.TypeFor[any]() for the bare interface case, or a reflect.PointerTo /
// reflect.SliceOf / etc. for composite shapes.
//
// Takes nativeRType (reflect.Type) which is the underlying reflect type.
// Takes piko (*pikoType) which carries source-level identity.
//
// Returns pikoNamedInterfaceWrapper which combines both.
func newPikoNamedInterfaceWrapper(nativeRType reflect.Type, piko *pikoType) pikoNamedInterfaceWrapper {
	return pikoNamedInterfaceWrapper{Type: nativeRType, piko: piko}
}

// unwrapPikoNamedInterfaceWrapper returns the wrapper when t holds one.
//
// Used at the native-call boundary to strip the wrapper before passing into reflect
// functions that internally type-assert to *rtype.
//
// Takes t (reflect.Type) which is the type to unwrap.
//
// Returns pikoNamedInterfaceWrapper which is the unwrapped wrapper, or the zero value
// when t is not a wrapper.
// Returns bool which is true when t is a wrapper.
func unwrapPikoNamedInterfaceWrapper(t reflect.Type) (pikoNamedInterfaceWrapper, bool) {
	w, ok := t.(pikoNamedInterfaceWrapper)
	return w, ok
}

// nativeReflectTypeFromWrapper returns the underlying reflect.Type.
//
// Passes t through unchanged when t is not a wrapper. Use at any boundary that feeds a
// reflect.Type back into native reflect functions (PointerTo, SliceOf, MakeMap, New,
// Zero, etc.).
//
// Takes t (reflect.Type) which may or may not be a wrapper.
//
// Returns reflect.Type which is the underlying native type, or t itself when t is not a
// wrapper.
func nativeReflectTypeFromWrapper(t reflect.Type) reflect.Type {
	if w, ok := unwrapPikoNamedInterfaceWrapper(t); ok {
		return w.Type
	}
	return t
}
