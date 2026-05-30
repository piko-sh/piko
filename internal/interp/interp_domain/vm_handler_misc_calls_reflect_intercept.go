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
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// tryInterceptPikoReflectTypeMethod filters piko-synth sentinels from reflect.Type method
// results.
//
// When piko code calls NumField/Field/Name on a reflect.Type that wraps a piko
// synthesised struct, the underlying reflect.StructOf type carries an extra
// `_pikoID_<Name>` field that should be hidden from user code.
//
// Takes vm (*VM) which owns the dispatch arena.
// Takes registers (*Registers) which receives the call result.
// Takes site (*callSite) which describes the return slot and any argument slots needed
// for Field/FieldByIndex.
// Takes receiverValue (reflect.Value) which is the reflect.Type receiver; intercept exits
// early when not a reflect.Type.
// Takes methodName (string) which is the method being invoked on the receiver.
//
// Returns opResult which is the dispatch outcome when intercepted.
// Returns bool which is true when the call was handled.
func tryInterceptPikoReflectTypeMethod(vm *VM, registers *Registers, site *callSite, receiverValue reflect.Value, methodName string) (opResult, bool) {
	if !receiverValue.IsValid() {
		return opContinue, false
	}
	if !receiverValue.Type().Implements(reflectTypeReflectType) || !receiverValue.CanInterface() {
		return opContinue, false
	}
	rt, ok := reflect.TypeAssert[reflect.Type](receiverValue)
	if !ok {
		return opContinue, false
	}
	switch methodName {
	case "NumField":
		return interceptReflectTypeNumField(registers, site, rt)
	case "Field":
		return interceptReflectTypeField(vm, registers, site, rt)
	case "Name":
		return interceptReflectTypeName(registers, site, rt)
	case "String":
		return interceptReflectTypeString(registers, site, rt)
	}
	return opContinue, false
}

// interceptReflectTypeNumField handles NumField() on a piko struct.
//
// Reports the user-visible field count that excludes piko's internal sentinel fields.
//
// Takes registers (*Registers) which receives the call result.
// Takes site (*callSite) which describes the return slot.
// Takes rt (reflect.Type) which is the receiver reflect.Type.
//
// Returns opResult which is the dispatch outcome.
// Returns bool which is true when the call was handled.
func interceptReflectTypeNumField(registers *Registers, site *callSite, rt reflect.Type) (opResult, bool) {
	if rt.Kind() != reflect.Struct {
		return opContinue, false
	}
	userFieldCount := pikoUserFieldCount(rt)
	if userFieldCount == rt.NumField() {
		return opContinue, false
	}
	storeReflectResults(registers, site.returns, []reflect.Value{reflect.ValueOf(userFieldCount)})
	return opContinue, true
}

// interceptReflectTypeField handles Field(i) on a piko struct.
//
// Bounds-checks the index against the user-visible field count and raises an interpreted
// panic when out of range.
//
// Takes vm (*VM) which owns the dispatch arena.
// Takes registers (*Registers) which receives the call result.
// Takes site (*callSite) which describes the return and argument slots.
// Takes rt (reflect.Type) which is the receiver reflect.Type.
//
// Returns opResult which is the dispatch outcome.
// Returns bool which is true when the call was handled.
func interceptReflectTypeField(vm *VM, registers *Registers, site *callSite, rt reflect.Type) (opResult, bool) {
	if rt.Kind() != reflect.Struct {
		return opContinue, false
	}
	userFieldCount := pikoUserFieldCount(rt)
	if userFieldCount == rt.NumField() {
		return opContinue, false
	}
	if len(site.arguments) < 2 {
		return opContinue, false
	}
	indexValue := registerToReflectValue(vm.arena, registers, site.arguments[1].kind, site.arguments[1].register)
	if !indexValue.IsValid() || !indexValue.CanInt() {
		return opContinue, false
	}
	i := int(indexValue.Int())
	if i < 0 || i >= userFieldCount {
		return raiseNativePanicAsInterpreted(vm, "reflect: Field index out of range"), true
	}
	storeReflectResults(registers, site.returns, []reflect.Value{reflect.ValueOf(rt.Field(i))})
	return opContinue, true
}

// interceptReflectTypeName handles Name() on a piko struct.
//
// Substitutes the source-level name recovered from the type's piko sentinel field when
// the reflect-level Name() is empty.
//
// Takes registers (*Registers) which receives the call result.
// Takes site (*callSite) which describes the return slot.
// Takes rt (reflect.Type) which is the receiver reflect.Type.
//
// Returns opResult which is the dispatch outcome.
// Returns bool which is true when the call was handled.
func interceptReflectTypeName(registers *Registers, site *callSite, rt reflect.Type) (opResult, bool) {
	if rt.Name() != "" {
		return opContinue, false
	}
	sentinelName := bareSentinelName(rt)
	if sentinelName == "" {
		return opContinue, false
	}
	storeReflectResults(registers, site.returns, []reflect.Value{reflect.ValueOf(sentinelName)})
	return opContinue, true
}

// interceptReflectTypeString handles String() on a piko struct.
//
// Recovers the package-qualified name from the sentinel field's metadata so the rendered
// string does not leak piko's `_pikoID_<Name>` field. The compiler stores either
// `<pkg-path>.<type-name>` or `<pkg-path>` in the sentinel field's PkgPath (sentinelField
// at compiler_types.go:373); the rest of the type name lives in the sentinel field's Name
// suffix. Package short name is the final `/`-delimited segment of the path, matching
// Go's convention for `reflect.Type.String()` when package directory and package name
// agree.
//
// Yields opContinue and false when rt carries no sentinel so the native rt.String() runs.
//
// Takes registers (*Registers) which receives the call result.
// Takes site (*callSite) which describes the return slot.
// Takes rt (reflect.Type) which is the receiver reflect.Type.
//
// Returns opResult which is the dispatch outcome.
// Returns bool which is true when the call was handled.
func interceptReflectTypeString(registers *Registers, site *callSite, rt reflect.Type) (opResult, bool) {
	qualified := qualifiedNameFromSentinel(rt)
	if qualified == "" {
		return opContinue, false
	}
	storeReflectResults(registers, site.returns, []reflect.Value{reflect.ValueOf(qualified)})
	return opContinue, true
}

// qualifiedNameFromSentinel reconstructs the source-level "<pkg>.<Name>"
// reflect.Type.String() form for a piko-synthesised struct type (or a pointer / slice /
// map / array / channel built on one). Returns "" when no sentinel is present so the
// caller can fall through to native rendering.
//
// Pointer wrappers contribute a leading "*"; arrays a "[N]"; slices a "[]"; maps
// "map[K]V"; channels "chan ". Composite forms recurse so `*[]spew.ConfigState` renders
// correctly without sentinel leak.
//
// Takes rt (reflect.Type) which is the type to render.
//
// Returns the rendered form or the empty string.
func qualifiedNameFromSentinel(rt reflect.Type) string {
	if rt == nil {
		return ""
	}
	if prefix, inner, ok := qualifiedNameRecurseInner(rt); ok {
		if inner == "" {
			return ""
		}
		return prefix + inner
	}
	if rt.Kind() != reflect.Struct {
		return ""
	}
	return qualifiedNameFromStructSentinel(rt)
}

// qualifiedNameRecurseInner peels wrapper kinds off a type.
//
// Produces the prefix string plus the inner-name lookup result. ok is false when rt was
// not a wrapper kind, so the caller should fall through to the struct sentinel handler.
//
// Takes rt (reflect.Type) which is the type to inspect.
//
// Returns prefix (string) which is the wrapper rendering.
// Returns inner (string) which is the resolved inner name.
// Returns ok (bool) which is true when rt was a wrapper kind.
func qualifiedNameRecurseInner(rt reflect.Type) (prefix, inner string, ok bool) {
	switch rt.Kind() {
	case reflect.Pointer:
		return "*", qualifiedNameFromSentinel(rt.Elem()), true
	case reflect.Slice:
		return "[]", qualifiedNameFromSentinel(rt.Elem()), true
	case reflect.Array:
		return fmt.Sprintf("[%d]", rt.Len()), qualifiedNameFromSentinel(rt.Elem()), true
	default:
	}
	return "", "", false
}

// qualifiedNameFromStructSentinel rebuilds the qualified name.
//
// Pulls the piko-id sentinel out of the final field of a struct type and reconstructs the
// "pkgShortName.TypeName" identifier. Returns "" when the struct lacks the sentinel
// suffix that piko emits via compileStructLiteral.
//
// Takes rt (reflect.Type) which is the struct type to inspect.
//
// Returns string which is the qualified name or empty.
func qualifiedNameFromStructSentinel(rt reflect.Type) string {
	if rt.NumField() == 0 {
		return ""
	}
	last := rt.Field(rt.NumField() - 1)
	if !strings.HasPrefix(last.Name, pikoIDFieldPrefix) {
		return ""
	}
	typeName := last.Name[len(pikoIDFieldPrefix):]
	pkgPath := last.PkgPath
	if suffix := "." + typeName; strings.HasSuffix(pkgPath, suffix) {
		pkgPath = pkgPath[:len(pkgPath)-len(suffix)]
	}
	pkgShortName := pkgPath
	if slash := strings.LastIndex(pkgPath, "/"); slash >= 0 {
		pkgShortName = pkgPath[slash+1:]
	}
	if pkgShortName == "" {
		return typeName
	}
	return pkgShortName + "." + typeName
}

// pikoUserFieldCount returns the number of non-sentinel fields on a piko-synthesised
// struct type. A type is considered piko-synthesised when its trailing field name starts
// with the `_pikoID_` sentinel prefix; in that case the count excludes the sentinel.
//
// Takes rt (reflect.Type) which is the struct type to inspect.
//
// Returns the user-visible field count.
func pikoUserFieldCount(rt reflect.Type) int {
	total := rt.NumField()
	if total == 0 {
		return 0
	}
	if strings.HasPrefix(rt.Field(total-1).Name, pikoIDFieldPrefix) {
		return total - 1
	}
	return total
}

// tryInterceptPikoReflectValueMethod handles piko reflect.Value calls.
//
// A piko-synthesised struct carries an empty Go-level method set (piko keeps methods in
// its own method table), so native reflect.Value.MethodByName / NumMethod return nothing.
// Resolves those against piko's method table and, for MethodByName, hands back a callable
// reflect.Value bound to the piko method (bug 750).
//
// Takes vm (*VM) which owns the method table and dispatch context.
// Takes registers (*Registers) which receives the result.
// Takes site (*callSite) which describes the argument/return slots.
// Takes receiverValue (reflect.Value) which is the reflect.Value receiver wrapping a piko
// value.
// Takes methodName (string) which is the method invoked on it.
//
// Returns opResult which is the dispatch outcome.
// Returns bool which is true when the call was handled.
func tryInterceptPikoReflectValueMethod(vm *VM, registers *Registers, site *callSite, receiverValue reflect.Value, methodName string) (opResult, bool) {
	if !receiverValue.IsValid() || receiverValue.Type() != reflectValueReflectType || !receiverValue.CanInterface() {
		return opContinue, false
	}
	inner, ok := reflect.TypeAssert[reflect.Value](receiverValue)
	if !ok || !inner.IsValid() {
		return opContinue, false
	}
	typeName := pikoReflectValueTypeName(vm, inner)
	if typeName == "" {
		return opContinue, false
	}
	switch methodName {
	case "MethodByName":
		return interceptPikoReflectMethodByName(vm, registers, site, inner, typeName)
	case "NumMethod":
		storeReflectResults(registers, site.returns, []reflect.Value{reflect.ValueOf(pikoMethodCount(vm, typeName))})
		return opContinue, true
	case "NumField":
		return interceptPikoReflectNumField(registers, site, inner)
	case "Field":
		return interceptPikoReflectField(vm, registers, site, inner)
	}
	return opContinue, false
}

// interceptPikoReflectMethodByName resolves a piko method by name.
//
// Resolves a method name argument against the piko method table and stores the callable,
// or the zero Value, into the site's return slot.
//
// Takes vm (*VM) which owns the method table and dispatch context.
// Takes registers (*Registers) which receives the result.
// Takes site (*callSite) which describes the argument/return slots.
// Takes inner (reflect.Value) which is the unwrapped piko receiver.
// Takes typeName (string) which is the source-level type name.
//
// Returns opResult which is the dispatch outcome.
// Returns bool which is true when the call was handled.
func interceptPikoReflectMethodByName(vm *VM, registers *Registers, site *callSite, inner reflect.Value, typeName string) (opResult, bool) {
	if len(site.arguments) < 2 {
		return opContinue, false
	}
	nameArgument := registerToReflectValue(vm.arena, registers, site.arguments[1].kind, site.arguments[1].register)
	if !nameArgument.IsValid() || nameArgument.Kind() != reflect.String {
		return opContinue, false
	}
	callable, found := buildPikoBoundMethodCallable(vm, inner, typeName, nameArgument.String())
	if !found {
		storeReflectResults(registers, site.returns, []reflect.Value{reflect.ValueOf(reflect.Value{})})
		return opContinue, true
	}
	storeReflectResults(registers, site.returns, []reflect.Value{reflect.ValueOf(callable)})
	return opContinue, true
}

// interceptPikoReflectNumField reports the user-visible field count.
//
// Matches interceptReflectTypeNumField on the Type side. Without it, code that iterates
// `for i := 0; i < v.NumField(); i++ { vt.Field(i) }` (go-spew's dump.go) walks past the
// type-side last index and panics with "Field index out of range".
//
// Takes registers (*Registers) which receives the result.
// Takes site (*callSite) which describes the return slot.
// Takes inner (reflect.Value) which is the unwrapped piko receiver.
//
// Returns opResult which is the dispatch outcome.
// Returns bool which is true when the call was handled.
func interceptPikoReflectNumField(registers *Registers, site *callSite, inner reflect.Value) (opResult, bool) {
	rt := inner.Type()
	for rt.Kind() == reflect.Pointer {
		rt = rt.Elem()
	}
	if rt.Kind() != reflect.Struct {
		return opContinue, false
	}
	userFieldCount := pikoUserFieldCount(rt)
	if userFieldCount == rt.NumField() {
		return opContinue, false
	}
	storeReflectResults(registers, site.returns, []reflect.Value{reflect.ValueOf(userFieldCount)})
	return opContinue, true
}

// interceptPikoReflectField bounds-checks Field(i) on a piko struct.
//
// Compares the index against the user-visible field count so the sentinel index raises
// the canonical "reflect: Field index out of range" panic instead of returning the
// sentinel zero-struct.
//
// Takes vm (*VM) which owns the dispatch arena.
// Takes registers (*Registers) which receives the result.
// Takes site (*callSite) which describes the argument/return slots.
// Takes inner (reflect.Value) which is the unwrapped piko receiver.
//
// Returns opResult which is the dispatch outcome.
// Returns bool which is true when the call was handled.
func interceptPikoReflectField(vm *VM, registers *Registers, site *callSite, inner reflect.Value) (opResult, bool) {
	if len(site.arguments) < 2 {
		return opContinue, false
	}
	rt := inner.Type()
	for rt.Kind() == reflect.Pointer {
		rt = rt.Elem()
	}
	if rt.Kind() != reflect.Struct {
		return opContinue, false
	}
	userFieldCount := pikoUserFieldCount(rt)
	if userFieldCount == rt.NumField() {
		return opContinue, false
	}
	indexValue := registerToReflectValue(vm.arena, registers, site.arguments[1].kind, site.arguments[1].register)
	if !indexValue.IsValid() || !indexValue.CanInt() {
		return opContinue, false
	}
	i := int(indexValue.Int())
	if i < 0 || i >= userFieldCount {
		return raiseNativePanicAsInterpreted(vm, "reflect: Field index out of range"), true
	}
	structValue := inner
	for structValue.Kind() == reflect.Pointer {
		if structValue.IsNil() {
			return raiseNativePanicAsInterpreted(vm, "reflect: indirection through nil pointer to embedded struct"), true
		}
		structValue = structValue.Elem()
	}
	storeReflectResults(registers, site.returns, []reflect.Value{reflect.ValueOf(structValue.Field(i))})
	return opContinue, true
}

// pikoReflectValueTypeName resolves the piko source-level type name.
//
// Unwraps one pointer layer when resolving the wrapped piko value.
//
// Takes vm (*VM) which provides the typeNames registry.
// Takes inner (reflect.Value) which is the wrapped piko value.
//
// Returns string which is the source-level name or empty when not piko-defined.
func pikoReflectValueTypeName(vm *VM, inner reflect.Value) string {
	rt := inner.Type()
	if name := bareSentinelName(rt); name != "" {
		return name
	}
	if name, ok := pikoTypeName(vm, inner); ok {
		return name
	}
	return ""
}

// pikoMethodCount counts piko methods registered for a type.
//
// Takes vm (*VM) which provides the method table.
// Takes typeName (string) which is the source-level type name.
//
// Returns int which is the method count.
func pikoMethodCount(vm *VM, typeName string) int {
	prefix := typeName + "."
	count := 0
	for key := range vm.rootFunction.methodTable {
		if strings.HasPrefix(key, prefix) {
			count++
		}
	}
	return count
}

// tryPikoReflectValueMethodGet handles the method-VALUE selector.
//
// Handles opGetMethod / subOpGetMethod for `reflect.Value` receivers whose wrapped value
// is a piko type. Native MethodByName / NumMethod cannot see piko's method set (it lives
// in the VM method table), so produces a native Go func value that handleCallNative
// invokes.
//
// For `v.MethodByName("Add")` stores a `func(string) reflect.Value` that, given the
// method name, returns a callable reflect.Value bound to the piko method via
// boundMethodVM. The follow-up `.Call(...)` dispatches through native reflect because the
// callable is a genuine reflect.MakeFunc value.
//
// Takes vm (*VM) which owns the method table.
// Takes registers (*Registers) which receives the func value.
// Takes destinationRegister (uint8) which is the general-bank slot.
// Takes receiverValue (reflect.Value) which is the reflect.Value receiver wrapping a piko
// value.
// Takes methodName (string) which is the reflect.Value method being selected
// (MethodByName / Method / NumMethod).
//
// Returns bool which is true when the selector was handled; false to fall through to
// native reflect.Value method dispatch.
func tryPikoReflectValueMethodGet(vm *VM, registers *Registers, destinationRegister uint8, receiverValue reflect.Value, methodName string) bool {
	if !receiverValue.CanInterface() {
		return false
	}
	inner, ok := reflect.TypeAssert[reflect.Value](receiverValue)
	if !ok || !inner.IsValid() {
		return false
	}
	typeName := pikoReflectValueTypeName(vm, inner)
	if typeName == "" {
		return false
	}
	switch methodName {
	case "MethodByName":
		registers.general[destinationRegister] = reflect.ValueOf(func(name string) reflect.Value {
			callable, found := buildPikoBoundMethodCallable(vm, inner, typeName, name)
			if !found {
				return reflect.Value{}
			}
			return callable
		})
		return true
	case "NumMethod":
		count := pikoMethodCount(vm, typeName)
		registers.general[destinationRegister] = reflect.ValueOf(func() int { return count })
		return true
	case "NumField":
		return installPikoReflectNumFieldOverride(registers, destinationRegister, inner)
	case "Field":
		return installPikoReflectFieldOverride(registers, destinationRegister, inner)
	}
	return false
}

// installPikoReflectNumFieldOverride hides the piko sentinel field.
//
// Hides the synthetic `_pikoID_<Name>` field from the reflect.Value side so library code
// that uses the `for i := 0; i < v.NumField(); i++` idiom (go-spew's struct dump path is
// the canonical example) sees the same count piko's reflect.Type intercept reports.
// Without it the Value side over-counts by one and the matching vt.Field(i) intercept
// raises "reflect: Field index out of range".
//
// Takes registers (*Registers) which receives the func value.
// Takes destinationRegister (uint8) which is the general-bank slot.
// Takes inner (reflect.Value) which is the unwrapped piko receiver.
//
// Returns bool which is true when the override was installed.
func installPikoReflectNumFieldOverride(registers *Registers, destinationRegister uint8, inner reflect.Value) bool {
	structType, ok := pikoStructTypeForValue(inner)
	if !ok {
		return false
	}
	userFieldCount := pikoUserFieldCount(structType)
	if userFieldCount == structType.NumField() {
		return false
	}
	registers.general[destinationRegister] = reflect.ValueOf(func() int { return userFieldCount })
	return true
}

// installPikoReflectFieldOverride installs a bounds-checked Field(i).
//
// Installs a Field(i) closure that bounds-checks against the user-visible field count so
// the sentinel index raises the canonical "reflect: Field index out of range" panic
// instead of exposing the synthetic identity field. The captured structValue preserves
// addressability semantics on the original receiver - reflect.Value. Field on a
// non-pointer Struct returns a Value whose addressability matches the receiver's.
//
// Takes registers (*Registers) which receives the func value.
// Takes destinationRegister (uint8) which is the general-bank slot.
// Takes inner (reflect.Value) which is the unwrapped piko receiver.
//
// Returns bool which is true when the override was installed.
//
// Panics when the installed Field(i) closure is invoked with an index outside the
// user-visible field range, matching the canonical reflect.Value.Field bounds-check
// panic.
func installPikoReflectFieldOverride(registers *Registers, destinationRegister uint8, inner reflect.Value) bool {
	structType, ok := pikoStructTypeForValue(inner)
	if !ok {
		return false
	}
	userFieldCount := pikoUserFieldCount(structType)
	if userFieldCount == structType.NumField() {
		return false
	}
	structValue := inner
	for structValue.Kind() == reflect.Pointer {
		if structValue.IsNil() {
			return false
		}
		structValue = structValue.Elem()
	}
	if structValue.Kind() != reflect.Struct {
		return false
	}
	registers.general[destinationRegister] = reflect.ValueOf(func(i int) reflect.Value {
		if i < 0 || i >= userFieldCount {
			panic("reflect: Field index out of range")
		}
		return structValue.Field(i)
	})
	return true
}

// safeMethodByName invokes MethodByName under a recover guard.
//
// Callers that would otherwise crash with "Method on nil interface value" (when
// receiverValue is a nil interface) or other reflect panics get a regular error return
// that piko can surface as a recoverable "undefined method" panic. spf13/cast's
// indirectToStringerOrError pattern lands here whenever the input flows through piko's
// interpreted interface boxing.
//
// Takes receiverValue (reflect.Value) which is the method receiver.
// Takes methodName (string) which is the method name being resolved.
//
// Returns method (reflect.Value) which is the bound method, or the zero Value when
// MethodByName found no match.
// Returns lookupErr (error) when the reflect call itself panicked.
func safeMethodByName(receiverValue reflect.Value, methodName string) (method reflect.Value, lookupErr error) {
	defer func() {
		if r := recover(); r != nil {
			method = reflect.Value{}
			lookupErr = fmt.Errorf("reflect MethodByName panic: %v", r)
		}
	}()
	if !receiverValue.IsValid() {
		return reflect.Value{}, errors.New("invalid receiver")
	}
	return receiverValue.MethodByName(methodName), nil
}

// pikoStructTypeForValue returns the underlying struct reflect.Type.
//
// Transparently dereferences one pointer layer. Used by the reflect.Value NumField/Field
// interceptors so the sentinel-stripping logic that runs on the Type side can be mirrored
// on the Value side without duplicating receiver-shape handling.
//
// Takes v (reflect.Value) which is the receiver wrapped by reflect.Value.
//
// Returns reflect.Type which is the struct type on success.
// Returns bool which is true on success; false when v is not a struct or
// pointer-to-struct value.
func pikoStructTypeForValue(v reflect.Value) (reflect.Type, bool) {
	rt := v.Type()
	for rt.Kind() == reflect.Pointer {
		rt = rt.Elem()
	}
	if rt.Kind() != reflect.Struct {
		return nil, false
	}
	return rt, true
}

// buildPikoBoundMethodCallable synthesises a piko-bound callable.
//
// Produces a callable reflect.Value for a piko method bound to receiver, so
// reflect.Value.MethodByName followed by .Call works on piko types. The returned func has
// interface-typed parameters (so exact-typed reflect args are accepted without an
// assignability panic) and dispatches into the interpreter through boundMethodVM.
//
// Takes vm (*VM) which provides the function table and dispatch.
// Takes receiver (reflect.Value) which is the bound piko value.
// Takes typeName (string) which is the source-level type name.
// Takes methodName (string) which is the method to resolve.
//
// Returns reflect.Value which is the callable on success.
// Returns bool which is true when the method exists; false and the zero Value otherwise.
func buildPikoBoundMethodCallable(vm *VM, receiver reflect.Value, typeName, methodName string) (reflect.Value, bool) {
	callee, methodRoot, ok := resolvePikoMethodCallee(vm, typeName, methodName)
	if !ok {
		return reflect.Value{}, false
	}
	inTypes := make([]reflect.Type, len(callee.parameterKinds)-1)
	for i := range inTypes {
		inTypes[i] = reflect.TypeFor[any]()
	}
	if callee.isVariadic && len(inTypes) > 0 {
		inTypes[len(inTypes)-1] = reflect.TypeFor[[]any]()
	}
	outTypes := make([]reflect.Type, len(callee.resultKinds))
	for i, k := range callee.resultKinds {
		outTypes[i] = kindDefaultReflectType(k)
	}
	funcType := reflect.FuncOf(inTypes, outTypes, callee.isVariadic)
	callable := reflect.MakeFunc(funcType, func(arguments []reflect.Value) []reflect.Value {
		bound := newCrossPackageBoundMethod(vm, methodRoot, callee)
		boundReceiver := receiverValueFor(callee, receiver)
		results := bound.invoke(boundReceiver, unwrapInterfaceArguments(arguments), identityArg)
		return shapeBoundMethodResults(results, outTypes)
	})
	return callable, true
}

// resolvePikoMethodCallee finds a piko method's CompiledFunction.
//
// Checks the local methodTable first and falls back to globalStore.externalMethods so
// cross-package method values emitted by compileSelectorMethodValue resolve to the
// foreign package's body. The returned methodRoot is nil for local methods (callee lives
// in vm.functions) and non-nil for cross-package methods (callee lives in the foreign
// rootFunction's functions slice). Callers feed methodRoot to newCrossPackageBoundMethod
// so the boundMethodVM dispatches against the correct functions slice.
//
// Takes vm (*VM) which provides both lookup spaces.
// Takes typeName (string) which is the source-level receiver type name.
// Takes methodName (string) which is the method identifier without qualifier.
//
// Returns callee (*CompiledFunction) which is the resolved function.
// Returns methodRoot (*CompiledFunction) which is nil for in-package methods.
// Returns ok (bool) which is true on success.
func resolvePikoMethodCallee(vm *VM, typeName, methodName string) (callee, methodRoot *CompiledFunction, ok bool) {
	if vm == nil {
		return nil, nil, false
	}
	key := typeName + "." + methodName
	if vm.rootFunction != nil {
		if funcIndex, ok := vm.rootFunction.methodTable[key]; ok && int(funcIndex) < len(vm.functions) {
			callee := vm.functions[funcIndex]
			if callee != nil && len(callee.parameterKinds) > 0 {
				return callee, nil, true
			}
		}
	}
	if vm.globals == nil {
		return nil, nil, false
	}
	entry, ok := vm.globals.lookupExternalMethod(key)
	if !ok || entry.rootFunction == nil {
		return nil, nil, false
	}
	if int(entry.methodIndex) >= len(entry.rootFunction.functions) {
		return nil, nil, false
	}
	callee = entry.rootFunction.functions[entry.methodIndex]
	if callee == nil || len(callee.parameterKinds) == 0 {
		return nil, nil, false
	}
	return callee, entry.rootFunction, true
}

// unwrapInterfaceArguments unwraps interface-typed arguments.
//
// Unwraps each non-nil interface-typed argument down to its concrete value so the
// interpreter receives exact-typed values rather than interface wrappers.
//
// Takes arguments ([]reflect.Value) which holds the reflect.MakeFunc arguments.
//
// Returns []reflect.Value which is a fresh slice with interface arguments unwrapped.
func unwrapInterfaceArguments(arguments []reflect.Value) []reflect.Value {
	unwrapped := make([]reflect.Value, len(arguments))
	for i := range arguments {
		unwrapped[i] = arguments[i]
		if unwrapped[i].Kind() == reflect.Interface && !unwrapped[i].IsNil() {
			unwrapped[i] = unwrapped[i].Elem()
		}
	}
	return unwrapped
}

// shapeBoundMethodResults coerces results to the declared output types.
//
// Converts where assignable and substitutes zero values for missing or invalid results.
//
// Takes results ([]reflect.Value) which holds the interpreter results.
// Takes outTypes ([]reflect.Type) which are the declared output types.
//
// Returns []reflect.Value which is shaped to the output types.
func shapeBoundMethodResults(results []reflect.Value, outTypes []reflect.Type) []reflect.Value {
	shaped := make([]reflect.Value, len(outTypes))
	for i := range outTypes {
		if i < len(results) && results[i].IsValid() {
			if results[i].Type() != outTypes[i] && results[i].Type().ConvertibleTo(outTypes[i]) {
				shaped[i] = results[i].Convert(outTypes[i])
			} else {
				shaped[i] = results[i]
			}
			continue
		}
		shaped[i] = reflect.Zero(outTypes[i])
	}
	return shaped
}
