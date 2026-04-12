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

var (
	// reflectMakeFuncStubPointer caches the shared trampoline code pointer that every
	// reflect.MakeFunc result reports from reflect.Value.Pointer.
	//
	// All MakeFunc closures dispatch through the same makeFuncStub, so this single address
	// identifies any reflect.MakeFunc value regardless of its signature. Native Go functions
	// never report this address.
	reflectMakeFuncStubPointer = reflect.MakeFunc(
		reflect.FuncOf(nil, nil, false),
		func([]reflect.Value) []reflect.Value { return nil },
	).Pointer()
)

// isPikoMakeFuncClosure reports whether reflectedFunction is a reflect.MakeFunc closure
// rather than a genuine native Go function.
//
// piko builds reflect.MakeFunc closures for bound methods, method expressions, and
// adapter callables. Their interface-typed parameters are an internal type-erasure
// device, not user-visible interface{} parameters, so argument coercion must not wrap
// such arguments in stdlib-interface adapters.
//
// Takes reflectedFunction (reflect.Value) which is the resolved callee.
//
// Returns true when the callee is a piko-side reflect.MakeFunc closure.
func isPikoMakeFuncClosure(reflectedFunction reflect.Value) bool {
	return reflectedFunction.Kind() == reflect.Func &&
		reflectedFunction.Pointer() == reflectMakeFuncStubPointer
}

// tryBuildInterfaceAdapter returns a reflect.Value that satisfies the expected interface
// type by wrapping the argument in a piko-side adapter, or an invalid reflect.Value when
// no adapter is applicable. The caller (coerceReflectArgument) falls back to its existing
// Convert path when the returned value is invalid.
//
// Supports the built-in error, fmt.Stringer, and json.Marshaler interfaces, plus the
// empty interface (any) which tries each adapter in turn. Other named interfaces fall
// through to the Convert path.
//
// The detection rule is: the argument's static type appears in vm.rootFunction.typeNames
// (so it is a piko-synthesised type that carries a piko-managed method set) AND the
// expected interface is one of the supported stdlib interfaces. Either condition failing
// means no adapter is built.
//
// Takes vm (*VM) which provides access to the method registry.
// Takes argument (reflect.Value) which is the value being passed through
// coerceReflectArgument.
// Takes expectedType (reflect.Type) which is the parameter's declared interface type.
//
// Returns a wrapped reflect.Value implementing expectedType, or an invalid reflect.Value
// when no adapter applies.
func tryBuildInterfaceAdapter(vm *VM, argument reflect.Value, expectedType reflect.Type, typeCtx argumentTypeContext) reflect.Value {
	if vm == nil || vm.rootFunction == nil || expectedType == nil {
		return reflect.Value{}
	}
	if expectedType.Kind() != reflect.Interface {
		return reflect.Value{}
	}
	allowAny := expectedType.NumMethod() == 0
	if !allowAny && !isSupportedAdapterInterface(expectedType) {
		return reflect.Value{}
	}
	if !argument.IsValid() {
		return reflect.Value{}
	}
	typeName := resolveAdapterTypeName(vm, argument, typeCtx)
	if typeName == "" {
		return reflect.Value{}
	}
	if !allowAny {
		return buildAdapterForInterface(vm, argument, expectedType, typeName)
	}
	return buildAnyInterfaceAdapter(vm, argument, typeName)
}

// isSupportedAdapterInterface reports whether expectedType is one of the stdlib
// interfaces that tryBuildInterfaceAdapter knows how to satisfy with a piko-side adapter.
//
// Takes expectedType (reflect.Type) which is the parameter's declared interface type.
//
// Returns true when an exact-match adapter exists for expectedType.
func isSupportedAdapterInterface(expectedType reflect.Type) bool {
	switch expectedType {
	case errorReflectType,
		stringerReflectType,
		jsonMarshalerReflectType,
		jsonUnmarshalerReflectType,
		ioReaderReflectType,
		ioWriterReflectType,
		fmtFormatterReflectType,
		fmtScannerReflectType,
		sortInterfaceReflectType:
		return true
	}
	return false
}

// resolveAdapterTypeName determines the source-level type name used to look up adapter
// methods for argument. Piko-synthesised types resolve via the typeNames registry,
// falling back to the call site's recorded static name; all other types use the static
// name directly.
//
// Takes vm (*VM) which provides the typeNames registry.
// Takes argument (reflect.Value) which is the value being adapted.
// Takes typeCtx (argumentTypeContext) which carries the static name.
//
// Returns the resolved source-level type name, or "" when unknown.
func resolveAdapterTypeName(vm *VM, argument reflect.Value, typeCtx argumentTypeContext) string {
	if !isPikoSynthesisedReflectType(argument.Type()) {
		return typeCtx.staticTypeName
	}
	if typeName, ok := pikoTypeName(vm, argument); ok {
		return typeName
	}
	return typeCtx.staticTypeName
}

// buildAdapterForInterface dispatches to the exact-match adapter builder for a known
// stdlib interface type.
//
// Takes vm (*VM) which provides the method registry.
// Takes argument (reflect.Value) which is the value being wrapped.
// Takes expectedType (reflect.Type) which is the declared interface.
// Takes typeName (string) which is the source-level type name.
//
// Returns a wrapped reflect.Value implementing expectedType, or an invalid reflect.Value
// when no adapter applies.
func buildAdapterForInterface(vm *VM, argument reflect.Value, expectedType reflect.Type, typeName string) reflect.Value {
	switch expectedType {
	case errorReflectType:
		return buildErrorAdapterIfRegistered(vm, argument, typeName)
	case stringerReflectType:
		return buildStringerAdapterIfRegistered(vm, argument, typeName)
	case jsonMarshalerReflectType:
		return buildMarshalerAdapterIfRegistered(vm, argument, typeName)
	case ioReaderReflectType:
		return buildReaderAdapterIfRegistered(vm, argument, typeName)
	case ioWriterReflectType:
		return buildWriterAdapterIfRegistered(vm, argument, typeName)
	case jsonUnmarshalerReflectType:
		return buildUnmarshalerAdapterIfRegistered(vm, argument, typeName)
	case fmtFormatterReflectType:
		return buildFormatterAdapterIfRegistered(vm, argument, typeName)
	case fmtScannerReflectType:
		return buildScannerAdapterIfRegistered(vm, argument, typeName)
	case sortInterfaceReflectType:
		return buildSortInterfaceAdapterIfRegistered(vm, argument, typeName)
	}
	return reflect.Value{}
}

// buildAnyInterfaceAdapter tries each adapter builder in turn for an empty-interface
// (any) parameter, returning the first applicable adapter or the Stringer adapter as the
// final fallback.
//
// Takes vm (*VM) which provides the method registry.
// Takes argument (reflect.Value) which is the value being wrapped.
// Takes typeName (string) which is the source-level type name.
//
// Returns the first applicable wrapped reflect.Value, or an invalid reflect.Value when no
// adapter applies.
func buildAnyInterfaceAdapter(vm *VM, argument reflect.Value, typeName string) reflect.Value {
	builders := []func(*VM, reflect.Value, string) reflect.Value{
		buildScannerAdapterIfRegistered,
		buildErrorAdapterIfRegistered,
		buildMarshalerAdapterIfRegistered,
		buildUnmarshalerAdapterIfRegistered,
		buildReaderAdapterIfRegistered,
		buildWriterAdapterIfRegistered,
	}
	for _, builder := range builders {
		if adapter := builder(vm, argument, typeName); adapter.IsValid() {
			return adapter
		}
	}
	return buildStringerAdapterIfRegistered(vm, argument, typeName)
}
