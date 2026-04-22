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

// LinkedFunction marks a registered symbol as a generic function whose
// interpreter dispatch is delegated to a non-generic sibling. The
// interpreter recognises this type when it retrieves a symbol and
// prepends one reflect.Type per declared type parameter before the
// regular call arguments.
type LinkedFunction struct {
	// Target is a reflect.Value of the sibling function. The sibling
	// must accept TypeArgCount reflect.Type values before the generic
	// function's non-type-parameter arguments.
	Target reflect.Value

	// Params captures the generic's non-type-parameter argument shape
	// in declaration order.
	//
	// The synthesiser uses these descriptors when rebuilding the
	// generic's go/types signature for user code to type-check
	// against. Empty or nil indicates legacy Wrap callers; the
	// synthesiser falls back to reading the sibling's signature
	// directly in that case.
	Params []GenericFieldType

	// Results captures the generic's return shape in declaration
	// order. This is what allows returns like ([]SearchResult[T],
	// error) to round-trip: a sibling whose reflect return is
	// (reflect.Value, error) cannot by itself tell the synthesiser
	// that the first return is a slice-of-generic-type.
	Results []GenericFieldType

	// Variadic mirrors reflect.Type.IsVariadic for the generic's
	// declared parameter list. The synthesiser uses it to emit
	// signatures that accept `opts ...Option` style tail arguments.
	Variadic bool

	// TypeArgCount is the number of type parameters on the generic
	// source. The interpreter synthesises this many reflect.Type
	// arguments from its compile-time types.Info.Instances view.
	TypeArgCount int
}

// Wrap constructs a LinkedFunction for the sibling function captured
// at codegen time. This short form is retained for backwards
// compatibility; new registrations should prefer WrapFunc which
// also records the generic's original parameter and return shape.
//
// Takes typeArgCount (int) which is the number of type parameters on
// the generic source the sibling links to.
// Takes target (any) which is the sibling function value.
//
// Returns a LinkedFunction ready to be stored as a reflect.Value in
// the interpreter's symbol map.
func Wrap(typeArgCount int, target any) LinkedFunction {
	return LinkedFunction{
		Target:       reflect.ValueOf(target),
		TypeArgCount: typeArgCount,
	}
}

// WrapFunc constructs a LinkedFunction with explicit parameter and
// return descriptors.
//
// Extract's codegen emits WrapFunc whenever the generic's signature
// references type parameters outside of the direct T position, because
// inferring the shape from the sibling's reflect signature alone is
// ambiguous once slices, maps, pointers, or generic-typed returns are
// involved.
//
// Takes typeArgCount (int) which is the number of type parameters on
// the generic source.
// Takes target (any) which is the sibling function value.
// Takes params ([]GenericFieldType) which describe the generic's
// non-type-parameter arguments in declaration order.
// Takes results ([]GenericFieldType) which describe the generic's
// return values in declaration order.
// Takes variadic (bool) which mirrors the generic's IsVariadic flag.
//
// Returns a LinkedFunction ready to be stored as a reflect.Value in
// the interpreter's symbol map.
func WrapFunc(typeArgCount int, target any, params, results []GenericFieldType, variadic bool) LinkedFunction {
	return LinkedFunction{
		Target:       reflect.ValueOf(target),
		Params:       params,
		Results:      results,
		Variadic:     variadic,
		TypeArgCount: typeArgCount,
	}
}
