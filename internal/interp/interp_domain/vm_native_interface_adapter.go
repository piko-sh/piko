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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"runtime"
	"sort"
	"strings"
)

const (
	// pikoReaderMaxNoProgressCalls caps zero-byte no-error Read calls.
	//
	// A well-behaved Read either advances (n > 0) or signals completion (io.EOF / an error);
	// the runaway case is a body that repeatedly returns (0, nil). io.ErrNoProgress already
	// terminates a single stall, so a small cap on no-progress calls bounds the pathological
	// loop without truncating a legitimate large stream whose Read returns small positive
	// counts many times over.
	pikoReaderMaxNoProgressCalls = 100

	// pikoReaderMaxCalls is the per-adapter overall ceiling on Read invocations. Set
	// generously so a legitimate large stream copied through a small buffer is never
	// truncated, while still capping a pathological progress-making infinite loop to bounded
	// CPU rather than a host freeze.
	pikoReaderMaxCalls = 100_000_000

	// pikoUnmarshalerMaxDepth caps recursive UnmarshalJSON entries.
	//
	// Defends against pathological recursion when a piko UnmarshalJSON body calls
	// json.Unmarshal in a way that re-enters the adapter (currently a signal of a bug,
	// typically a register-allocation collision that passes the receiver where a different
	// argument was intended). The limit keeps the host responsive while leaving a clear
	// failure signature.
	pikoUnmarshalerMaxDepth = 32
)

var (
	// errorsAsPointer caches the function pointer of stdlib errors.As so the native-call
	// dispatcher can recognise it by identity and route to pikoErrorsAs without scanning
	// function metadata. Resolved once at package init.
	errorsAsPointer = reflect.ValueOf(errors.As).Pointer()

	// errorsUnwrapPointer caches the function pointer of stdlib errors.Unwrap so
	// handleCallNativeReflect can route piko-synthesised errors through pikoErrorsUnwrap
	// without scanning function metadata. Without this, calling errors.Unwrap on a piko
	// *DBError triggers reflect: Call using *struct{...} as type error because the
	// synthesised type's reflect.Type has no method-set.
	errorsUnwrapPointer = reflect.ValueOf(errors.Unwrap).Pointer()

	// errorsIsPointer caches the function pointer of stdlib errors.Is for the same reason as
	// errorsAsPointer.
	errorsIsPointer = reflect.ValueOf(errors.Is).Pointer()

	// reflectTypeOfPointer caches reflect.TypeOf's function pointer.
	//
	// Lets handleCallNativeReflect recognise it and re-clothe the result in a pikoNamedType
	// when the argument was a piko-defined named primitive (`type Score int`). piko
	// collapses such types to their underlying reflect.Type, so the genuine reflect.TypeOf
	// returns "int" where Go returns "Score"; the wrapper restores the source-level name.
	reflectTypeOfPointer = reflect.ValueOf(reflect.TypeOf).Pointer()

	// reflectTypeConstructorPointers caches the function pointers of the reflect
	// type-constructor family. These functions internally type-assert their reflect.Type
	// argument to the concrete *reflect.rtype and panic when handed a pikoNamedType wrapper,
	// so the wrapper must be unwrapped before the native call.
	reflectTypeConstructorPointers = map[uintptr]struct{}{
		reflect.ValueOf(reflect.PointerTo).Pointer(): {},
		reflect.ValueOf(reflect.SliceOf).Pointer():   {},
		reflect.ValueOf(reflect.ChanOf).Pointer():    {},
		reflect.ValueOf(reflect.MapOf).Pointer():     {},
		reflect.ValueOf(reflect.ArrayOf).Pointer():   {},
		reflect.ValueOf(reflect.FuncOf).Pointer():    {},
		reflect.ValueOf(reflect.New).Pointer():       {},
		reflect.ValueOf(reflect.Zero).Pointer():      {},
		reflect.ValueOf(reflect.MakeSlice).Pointer(): {},
		reflect.ValueOf(reflect.MakeChan).Pointer():  {},
		reflect.ValueOf(reflect.MakeMap).Pointer():   {},
	}

	// reflectMakeFuncPointer caches reflect.MakeFunc's function pointer.
	reflectMakeFuncPointer = reflect.ValueOf(reflect.MakeFunc).Pointer()

	// runtimeGoexitPointer caches the function pointer of runtime.Goexit so the native-call
	// dispatcher can intercept it before it executes on the host goroutine. Calling the real
	// runtime.Goexit would terminate the host goroutine running the VM, violating the
	// isolation contract; instead the interpreter routes to its own goexit unwind machinery.
	runtimeGoexitPointer = reflect.ValueOf(runtime.Goexit).Pointer()

	// errorReflectType is the reflect.Type for the built-in error interface. Cached for
	// cheap comparisons in tryBuildInterfaceAdapter.
	errorReflectType = reflect.TypeFor[error]()

	// stringerReflectType is the reflect.Type for fmt.Stringer. Cached for cheap comparisons
	// in tryBuildInterfaceAdapter.
	stringerReflectType = reflect.TypeFor[fmt.Stringer]()

	// jsonMarshalerReflectType is the reflect.Type for json.Marshaler. Cached for cheap
	// comparisons in tryBuildInterfaceAdapter.
	jsonMarshalerReflectType = reflect.TypeFor[json.Marshaler]()

	// ioReaderReflectType is the reflect.Type for io.Reader. Used so piko-synthesised
	// structs declaring a `Read([]byte) (int, error)` method can satisfy io.Reader at the
	// native call boundary (io.Copy, io.ReadAll, bufio.NewReader, etc.) despite their
	// reflect.Type carrying an empty method set.
	ioReaderReflectType = reflect.TypeFor[io.Reader]()

	// jsonUnmarshalerReflectType is the reflect.Type for json.Unmarshaler. Used so
	// piko-synthesised pointer-receiver types declaring an `UnmarshalJSON([]byte) error`
	// method can satisfy json.Unmarshaler at the native call boundary (json.Unmarshal,
	// json.Decoder), with the adapter forwarding the JSON bytes to the source-level method
	// and surfacing its error.
	jsonUnmarshalerReflectType = reflect.TypeFor[json.Unmarshaler]()

	// ioWriterReflectType is the reflect.Type for io.Writer. Used so piko-synthesised
	// pointer-receiver types declaring a `Write([]byte) (int, error)` method can satisfy
	// io.Writer at the native call boundary (fmt.Fprintf, io.Copy, bufio.NewWriter, etc.).
	ioWriterReflectType = reflect.TypeFor[io.Writer]()

	// wellKnownNamedInterfaceRegistry maps qualified names to types.
	//
	// Holds the (pkgPath.typeName -> reflect.Type) lookup table consulted by
	// wellKnownNamedInterfaceReflectType. Entries are added on first import in compileType
	// to keep convertType O(1) per named-interface query. The set is intentionally narrow:
	// only interfaces whose reflect.Type the runtime already caches above are listed, so
	// preservation does not widen piko's identity surface.
	wellKnownNamedInterfaceRegistry = map[string]reflect.Type{
		"fmt.Stringer":              stringerReflectType,
		"io.Reader":                 ioReaderReflectType,
		"io.Writer":                 ioWriterReflectType,
		"encoding/json.Marshaler":   jsonMarshalerReflectType,
		"encoding/json.Unmarshaler": jsonUnmarshalerReflectType,
	}

	// fmtFormatterReflectType is the reflect.Type for fmt.Formatter, cached for
	// tryBuildInterfaceAdapter's whitelist check.
	fmtFormatterReflectType = reflect.TypeFor[fmt.Formatter]()

	// fmtScannerReflectType is the reflect.Type for fmt.Scanner, cached for
	// tryBuildInterfaceAdapter's whitelist check.
	fmtScannerReflectType = reflect.TypeFor[fmt.Scanner]()

	// sortInterfaceReflectType is the reflect.Type for sort.Interface, cached for
	// tryBuildInterfaceAdapter's whitelist check.
	sortInterfaceReflectType = reflect.TypeFor[sort.Interface]()
)

// pikoNamedType wraps reflect.Type to report a piko source-level name.
//
// The collapsing type converter makes `type Score int` indistinguishable from `int` at
// the reflect.Type level; the wrapper carries the name separately and overrides Name() /
// String() so `reflect.TypeOf(Score(1)).Name()` yields "Score". Kind() is not overridden
// because the embedded type's Kind (reflect.Int) is already correct.
//
// reflect.Type is embedded so the wrapper satisfies the full ~50-method interface for
// free; only the name-bearing methods are overridden.
type pikoNamedType struct {
	reflect.Type

	// sourceName is the piko source-level bare type name ("Score").
	sourceName string

	// sourcePackage is the defining package's short name ("main").
	sourcePackage string
}

// Name returns the piko source-level type name.
//
// Returns string which is the source-level name.
func (t pikoNamedType) Name() string { return t.sourceName }

// String returns the package-qualified source-level type name.
//
// Returns string which is the qualified rendering.
func (t pikoNamedType) String() string {
	if t.sourcePackage == "" {
		return t.sourceName
	}
	return t.sourcePackage + "." + t.sourceName
}

// pikoMarshalerAdapter mirrors pikoStringerAdapter for json.Marshaler: it wraps a
// piko-synthesised value so encoding/json can dispatch to the source-level MarshalJSON
// method even though the synthesised reflect.Type has no MethodSet.
type pikoMarshalerAdapter struct {
	// vm is the parent virtual machine used to dispatch the source MarshalJSON method on the
	// wrapped value.
	vm *VM

	// methodRoot pins the cross-package rootFunction that owns the method body.
	methodRoot *CompiledFunction

	// underlying is the reflect.Value of the piko-synthesised type that source code attaches
	// the MarshalJSON method to.
	underlying reflect.Value

	// methodIndex is the index into methodRoot.functions (or vm.rootFunction.functions when
	// methodRoot is nil) of the compiled MarshalJSON method, cached at adapter construction.
	methodIndex uint16
}

// MarshalJSON invokes the wrapped value's MarshalJSON method.
//
// Interpreted-body panics are recovered and re-raised when the interpreter is still on
// the dispatch stack (dispatchDepth > 0) so an upstream interpreted defer/recover can
// catch them; otherwise they are surfaced as the returned error so the host stays alive.
//
// Returns the JSON bytes produced by the source-level method, or nil when no usable
// result is produced.
// Returns an error when the method body panics outside a piko dispatch frame.
//
// Panics when an interpreted panic occurs while the interpreter is still on the dispatch
// stack, re-raising so an upstream interpreted defer/recover can catch it.
func (a *pikoMarshalerAdapter) MarshalJSON() (raw []byte, returnedError error) {
	callee, root, ok := resolveAdapterCallee(a.vm, a.methodRoot, a.methodIndex)
	if !ok {
		return nil, nil
	}
	defer func() {
		recovered := recover()
		if recovered == nil {
			return
		}
		if a.vm.globals != nil && a.vm.globals.dispatchDepth.Load() > 0 {
			panic(recovered)
		}
		raw = nil
		returnedError = fmt.Errorf("piko marshaler panicked: %v", recovered)
	}()
	bound := newCrossPackageBoundMethod(a.vm, root, callee)
	receiver := receiverValueFor(callee, a.underlying)
	results := bound.invoke(receiver, nil, identityArg)
	if len(results) == 0 {
		return nil, nil
	}
	raw = extractBytesResult(results)
	returnedError = extractErrorResult(results)
	return raw, returnedError
}

// pikoStringerAdapter is the Stringer mirror of pikoErrorAdapter: it wraps a
// piko-synthesised value so it satisfies fmt.Stringer even though the value's
// reflect.Type carries no String method.
type pikoStringerAdapter struct {
	// vm is the parent virtual machine used to dispatch the source String method on the
	// wrapped value.
	vm *VM

	// methodRoot pins the cross-CompileProgram-batch rootFunction owning the method body
	// (nil for in-package targets).
	methodRoot *CompiledFunction

	// underlying is the reflect.Value of the piko-synthesised type that source code attaches
	// the String method to.
	underlying reflect.Value

	// methodIndex is the index into methodRoot.functions (or vm.rootFunction.functions when
	// methodRoot is nil) of the compiled String method, cached at adapter construction.
	methodIndex uint16
}

// String invokes the wrapped piko value's String() method through the interpreter's
// bound-method dispatch. Returns an empty string when the bound method produces no usable
// result, matching pikoErrorAdapter for parity.
//
// Returns the string produced by the source-level method, or an empty string when no
// result is produced.
func (a *pikoStringerAdapter) String() string {
	return invokeStringReturnMethod(a.vm, a.methodRoot, a.methodIndex, a.underlying)
}

// pikoErrorAdapter wraps a piko-synthesised value so it satisfies the stdlib error
// interface.
//
// Synthesised reflect.Types built via reflect.StructOf carry an empty method set, so
// reflect.Type.Implements(errorInterface) (used by errors.As/Is and fmt.Errorf) reports
// false on the raw value. The adapter supplies a real Go Error method that delegates back
// into the interpreter to invoke the source-level Error method registered in the root
// function's method table.
type pikoErrorAdapter struct {
	// vm is the parent virtual machine used to dispatch the source Error() method on the
	// wrapped value.
	vm *VM

	// methodRoot pins the rootFunction owning the method body for the cross-package case, or
	// nil for in-package methods.
	methodRoot *CompiledFunction

	// underlying is the reflect.Value of the piko-synthesised type that source code attaches
	// the Error() method to. Holds either the value or a pointer depending on the receiver
	// kind of the declared method.
	underlying reflect.Value

	// methodIndex is the index into methodRoot.functions (or vm.rootFunction.functions when
	// methodRoot is nil) of the compiled Error() method, cached at adapter construction so
	// each Error() call avoids a method-table lookup.
	methodIndex uint16
}

// Error invokes the wrapped piko value's Error() method via the interpreter's
// bound-method dispatch path. The result is the string returned by the source-level
// method body.
//
// Returns the error message produced by the wrapped value's Error() method, or an empty
// string when no method runs (this should not occur in practice because
// tryBuildInterfaceAdapter only constructs adapters when the method index is known to
// resolve).
func (a *pikoErrorAdapter) Error() string {
	return invokeStringReturnMethod(a.vm, a.methodRoot, a.methodIndex, a.underlying)
}

// pikoWriterAdapter wraps a piko value to satisfy io.Writer.
//
// Mirrors pikoReaderAdapter for `Write([]byte) (int, error)` so piko types passed to
// fmt.Fprintf, io.Copy, bufio.NewWriter, etc. can dispatch back into the interpreter for
// the user-declared Write method.
type pikoWriterAdapter struct {
	// vm is the parent virtual machine used to dispatch the source Write method on the
	// wrapped value.
	vm *VM

	// methodRoot pins the rootFunction owning the method body for the cross-package case, or
	// nil for in-package methods.
	methodRoot *CompiledFunction

	// underlying is the reflect.Value of the piko-synthesised type that source code attaches
	// the Write method to.
	underlying reflect.Value

	// methodIndex is the index into methodRoot.functions (or vm.rootFunction.functions when
	// methodRoot is nil) of the compiled Write method, cached at adapter construction.
	methodIndex uint16
}

// Write invokes the wrapped piko value's Write method through the interpreter's
// bound-method dispatch.
//
// Takes p ([]byte) which is the source buffer Write should consume.
//
// Returns n (int) which is the number of bytes accepted by the piko-side Write
// implementation.
// Returns err (error) which carries any error returned by the piko-side method.
//
// Panics when an interpreted panic occurs while the interpreter is still on the dispatch
// stack, re-raising so an upstream interpreted defer/recover can catch it.
func (a *pikoWriterAdapter) Write(p []byte) (n int, err error) {
	callee, root, ok := resolveAdapterCallee(a.vm, a.methodRoot, a.methodIndex)
	if !ok {
		return 0, errors.New("piko writer: method unavailable")
	}
	defer func() {
		recovered := recover()
		if recovered == nil {
			return
		}
		if a.vm.globals != nil && a.vm.globals.dispatchDepth.Load() > 0 {
			panic(recovered)
		}
		n = 0
		err = fmt.Errorf("piko writer panicked: %v", recovered)
	}()
	bound := newCrossPackageBoundMethod(a.vm, root, callee)
	receiver := receiverValueFor(callee, a.underlying)
	results := bound.invoke(receiver, []reflect.Value{reflect.ValueOf(p)}, identityArg)
	if len(results) >= 1 && results[0].IsValid() {
		n = int(results[0].Int())
	}
	err = extractErrorResult(results)

	if n < 0 {
		n = 0
	}
	if n > len(p) {
		n = len(p)
	}
	if n < len(p) && err == nil {
		err = io.ErrShortWrite
	}
	return n, err
}

// pikoFormatterAdapter wraps a piko value to satisfy fmt.Formatter.
//
// A piko type with a `Format(fmt.State, rune)` method gets its custom formatting honoured
// by fmt.Sprintf et al. The fmt.State argument is a genuine native object passed through
// to the interpreter-side method, which calls State.Write / fmt.Fprintf on it via the
// normal native-dispatch path; no reverse adapter is needed.
type pikoFormatterAdapter struct {
	// vm is the parent virtual machine used to dispatch the source Format method on the
	// wrapped value.
	vm *VM

	// methodRoot pins the rootFunction owning the method body for the cross-package case, or
	// nil for in-package methods.
	methodRoot *CompiledFunction

	// underlying is the reflect.Value of the piko-synthesised type the Format method is
	// attached to.
	underlying reflect.Value

	// methodIndex is the index into methodRoot.functions (or vm.rootFunction.functions when
	// methodRoot is nil) of the compiled Format method.
	methodIndex uint16
}

// Format invokes the wrapped piko value's Format method, forwarding fmt's State and verb.
// Panics from the interpreted method are contained unless raised inside an active native
// dispatch.
//
// Takes state (fmt.State) which fmt supplies for output + flags.
// Takes verb (rune) which is the format verb.
func (a *pikoFormatterAdapter) Format(state fmt.State, verb rune) {
	callee, root, ok := resolveAdapterCallee(a.vm, a.methodRoot, a.methodIndex)
	if !ok {
		return
	}
	defer func() {
		recovered := recover()
		if recovered == nil {
			return
		}
		if a.vm.globals != nil && a.vm.globals.dispatchDepth.Load() > 0 {
			panic(recovered)
		}
		_, _ = fmt.Fprintf(state, "%%!%c(piko formatter panicked: %v)", verb, recovered)
	}()
	bound := newCrossPackageBoundMethod(a.vm, root, callee)
	receiver := receiverValueFor(callee, a.underlying)
	bound.invoke(receiver, []reflect.Value{reflect.ValueOf(state), reflect.ValueOf(verb)}, identityArg)
}

// pikoScannerAdapter wraps a piko pointer to satisfy fmt.Scanner.
//
// A piko type with a `Scan(fmt.ScanState, rune) error` method gets its custom scanning
// honoured by fmt.Fscan et al. The fmt.ScanState argument is a genuine native object
// passed through to the interpreter-side method; the method calls ScanState.Token /
// ReadRune on it via normal native dispatch.
type pikoScannerAdapter struct {
	// vm is the parent virtual machine used to dispatch the source Scan method on the
	// wrapped value.
	vm *VM

	// methodRoot pins the rootFunction owning the method body for the cross-package case, or
	// nil for in-package methods.
	methodRoot *CompiledFunction

	// underlying is the reflect.Value of the piko-synthesised type the Scan method is
	// attached to. Held as a pointer because Scan mutates the receiver.
	underlying reflect.Value

	// methodIndex is the index into methodRoot.functions (or vm.rootFunction.functions when
	// methodRoot is nil) of the compiled Scan method.
	methodIndex uint16
}

// Scan invokes the wrapped piko value's Scan method.
//
// Forwards fmt's ScanState and verb and returns the interpreted method's error.
//
// Takes state (fmt.ScanState) which fmt supplies for token reading.
// Takes verb (rune) which is the scan verb.
//
// Returns error which is produced by the source-level Scan method.
//
// Panics when an interpreted panic occurs while the interpreter is still on the dispatch
// stack, re-raising so an upstream interpreted defer/recover can catch it.
func (a *pikoScannerAdapter) Scan(state fmt.ScanState, verb rune) (returnedError error) {
	callee, root, ok := resolveAdapterCallee(a.vm, a.methodRoot, a.methodIndex)
	if !ok {
		return errors.New("piko scanner: method unavailable")
	}
	defer func() {
		recovered := recover()
		if recovered == nil {
			return
		}
		if a.vm.globals != nil && a.vm.globals.dispatchDepth.Load() > 0 {
			panic(recovered)
		}
		returnedError = fmt.Errorf("piko scanner panicked: %v", recovered)
	}()
	bound := newCrossPackageBoundMethod(a.vm, root, callee)
	receiver := receiverValueFor(callee, a.underlying)
	results := bound.invoke(receiver, []reflect.Value{reflect.ValueOf(state), reflect.ValueOf(verb)}, identityArg)
	if len(results) >= 1 && results[0].IsValid() {
		if e, ok := reflect.TypeAssert[error](results[0]); ok && e != nil {
			return e
		}
	}
	return nil
}

// pikoSortInterfaceAdapter wraps a piko-synthesised value (typically a named slice such
// as `type ByLen []string`) so it satisfies sort.Interface -
// `sort.Sort`/`sort.Stable`/`sort.IsSorted` then dispatch Len/Less/Swap back into the
// interpreter.
type pikoSortInterfaceAdapter struct {
	// vm is the parent virtual machine used to dispatch the source Len/Less/Swap methods.
	vm *VM

	// methodRoot pins the rootFunction owning Len/Less/Swap when they live in a different
	// CompileProgram batch than vm.rootFunction.
	//
	// Nil for in-package types. All three methods must come from the same rootFunction - the
	// typeName lookup guarantees this because methodTable entries for one type are always
	// co-located.
	methodRoot *CompiledFunction

	// underlying is the reflect.Value of the piko value the sort methods are attached to.
	underlying reflect.Value

	// lenIndex is the index of the compiled Len method.
	lenIndex uint16

	// lessIndex is the index of the compiled Less method.
	lessIndex uint16

	// swapIndex is the index of the compiled Swap method.
	swapIndex uint16
}

// Len dispatches the source-level Len method.
//
// Returns int which is the slice length reported by the bound Len.
func (a *pikoSortInterfaceAdapter) Len() int {
	results := a.dispatch(a.lenIndex, nil)
	if len(results) >= 1 && results[0].IsValid() && results[0].CanInt() {
		return int(results[0].Int())
	}
	return 0
}

// Less dispatches the source-level Less method.
//
// Takes i (int) which is the first index to compare.
// Takes j (int) which is the second index to compare.
//
// Returns bool which is the comparison result from the bound Less.
func (a *pikoSortInterfaceAdapter) Less(i, j int) bool {
	results := a.dispatch(a.lessIndex, []reflect.Value{reflect.ValueOf(i), reflect.ValueOf(j)})
	if len(results) >= 1 && results[0].IsValid() && results[0].Kind() == reflect.Bool {
		return results[0].Bool()
	}
	return false
}

// Swap dispatches the source-level Swap method.
//
// Takes i (int) which is the first index to exchange.
// Takes j (int) which is the second index to exchange.
func (a *pikoSortInterfaceAdapter) Swap(i, j int) {
	a.dispatch(a.swapIndex, []reflect.Value{reflect.ValueOf(i), reflect.ValueOf(j)})
}

// dispatch invokes one of the wrapped value's sort methods through boundMethodVM, with
// the panic-containment boilerplate shared by the other piko adapters.
//
// Takes methodIndex (uint16) which selects Len/Less/Swap.
// Takes arguments ([]reflect.Value) which are the method arguments.
//
// Returns the method's reflect results.
func (a *pikoSortInterfaceAdapter) dispatch(methodIndex uint16, arguments []reflect.Value) (results []reflect.Value) {
	callee, root, ok := resolveAdapterCallee(a.vm, a.methodRoot, methodIndex)
	if !ok {
		return nil
	}
	defer func() {
		recovered := recover()
		if recovered == nil {
			return
		}
		if a.vm.globals != nil && a.vm.globals.dispatchDepth.Load() > 0 {
			panic(recovered)
		}
		results = nil
	}()
	bound := newCrossPackageBoundMethod(a.vm, root, callee)
	receiver := receiverValueFor(callee, a.underlying)
	return bound.invoke(receiver, arguments, identityArg)
}

// pikoReaderAdapter wraps a piko value to satisfy io.Reader.
//
// Mirrors pikoErrorAdapter for the `Read([]byte) (int, error)` shape so piko types passed
// to io.ReadAll, io.Copy, bufio.NewReader, etc. can dispatch back into the interpreter
// for the user-declared Read method.
type pikoReaderAdapter struct {
	// vm is the parent virtual machine used to dispatch the source Read method on the
	// wrapped value.
	vm *VM

	// methodRoot pins the rootFunction owning the method body for the cross-package case, or
	// nil for in-package methods.
	methodRoot *CompiledFunction

	// underlying is the reflect.Value of the piko-synthesised type that source code attaches
	// the Read method to.
	underlying reflect.Value

	// methodIndex is the index into methodRoot.functions (or vm.rootFunction.functions when
	// methodRoot is nil) of the compiled Read method, cached at adapter construction.
	methodIndex uint16

	// callCount tracks the total number of times Read has been invoked. Defensive overall
	// cap against a progress-making infinite loop; the adapter forces io.EOF after
	// pikoReaderMaxCalls invocations to prevent host hangs.
	callCount int

	// noProgressCount tracks how many times Read has returned with no bytes copied and no
	// error. The adapter forces io.EOF once this crosses pikoReaderMaxNoProgressCalls, the
	// true runaway signature, without penalising a legitimate large stream that makes steady
	// small-count progress.
	noProgressCount int
}

// Read invokes the wrapped piko value's Read method through the interpreter's
// bound-method dispatch.
//
// Takes p ([]byte) which is the destination buffer Read should fill.
//
// Returns n (int) which is the number of bytes copied into p.
// Returns err (error) which carries any error returned by the piko-side method (io.EOF or
// otherwise).
//
// Panics when an interpreted panic occurs while the interpreter is still on the dispatch
// stack, re-raising so an upstream interpreted defer/recover can catch it.
func (a *pikoReaderAdapter) Read(p []byte) (n int, err error) {
	callee, root, ok := resolveAdapterCallee(a.vm, a.methodRoot, a.methodIndex)
	if !ok {
		return 0, io.EOF
	}
	a.callCount++
	if a.callCount > pikoReaderMaxCalls {
		return 0, io.EOF
	}
	defer func() {
		recovered := recover()
		if recovered == nil {
			return
		}
		if a.vm.globals != nil && a.vm.globals.dispatchDepth.Load() > 0 {
			panic(recovered)
		}
		n = 0
		err = fmt.Errorf("piko reader panicked: %v", recovered)
	}()
	bound := newCrossPackageBoundMethod(a.vm, root, callee)
	receiver := receiverValueFor(callee, a.underlying)
	results := bound.invoke(receiver, []reflect.Value{reflect.ValueOf(p)}, identityArg)
	if len(results) >= 1 && results[0].IsValid() {
		n = int(results[0].Int())
	}
	err = extractErrorResult(results)

	if n < 0 {
		n = 0
	}
	if n > len(p) {
		n = len(p)
	}
	if n == 0 && err == nil && len(p) > 0 {
		a.noProgressCount++
		if a.noProgressCount > pikoReaderMaxNoProgressCalls {
			return 0, io.EOF
		}
		err = io.ErrNoProgress
	}
	return n, err
}

// pikoUnmarshalerAdapter wraps a piko-synthesised pointer value so it satisfies the
// encoding/json.Unmarshaler interface. Mirrors pikoMarshalerAdapter but in the reverse
// direction: json.Unmarshal dispatches the raw JSON bytes here, and we forward them to
// the source-level UnmarshalJSON method bound to the wrapped pointer.
//
// Always wraps a *T (pointer receiver) because UnmarshalJSON mutates the receiver;
// buildUnmarshalerAdapterIfRegistered rejects value receivers via
// methodReceiverSatisfiesValue.
type pikoUnmarshalerAdapter struct {
	// vm is the parent virtual machine used to dispatch the source UnmarshalJSON method on
	// the wrapped value.
	vm *VM

	// methodRoot pins the rootFunction owning the method body for the cross-package case, or
	// nil for in-package methods.
	methodRoot *CompiledFunction

	// underlying is the reflect.Value of the piko-synthesised type that source code attaches
	// the UnmarshalJSON method to. Held as a *T pointer so mutations performed by the method
	// body are visible to the caller.
	underlying reflect.Value

	// methodIndex is the index into methodRoot.functions (or vm.rootFunction.functions when
	// methodRoot is nil) of the compiled UnmarshalJSON method, cached at adapter
	// construction.
	methodIndex uint16

	// callDepth tracks recursive entry into this adapter; capped at pikoUnmarshalerMaxDepth
	// to bail before a host stack overflow.
	callDepth int
}

// UnmarshalJSON invokes the wrapped piko value's UnmarshalJSON method.
// Returns the error produced by the source-level method, or any panic captured in the
// dispatch boundary.
//
// Takes data ([]byte) which is the JSON bytes to decode.
//
// Returns the error from the source-level UnmarshalJSON method, or nil on success.
func (a *pikoUnmarshalerAdapter) UnmarshalJSON(data []byte) (returnedError error) {
	callee, root, ok := resolveAdapterCallee(a.vm, a.methodRoot, a.methodIndex)
	if !ok {
		return nil
	}
	a.callDepth++
	defer func() { a.callDepth-- }()
	if a.callDepth > pikoUnmarshalerMaxDepth {
		return errors.New("piko unmarshaler recursion depth exceeded")
	}
	defer func() {
		recovered := recover()
		if recovered == nil {
			return
		}
		if a.vm.globals != nil && a.vm.globals.dispatchDepth.Load() > 0 {
			panic(recovered)
		}
		returnedError = fmt.Errorf("piko unmarshaler panicked: %v", recovered)
	}()
	bound := newCrossPackageBoundMethod(a.vm, root, callee)
	receiver := receiverValueFor(callee, a.underlying)
	results := bound.invoke(receiver, []reflect.Value{reflect.ValueOf(data)}, identityArg)
	if len(results) >= 1 && results[0].IsValid() {
		if e, ok := reflect.TypeAssert[error](results[0]); ok && e != nil {
			return e
		}
	}
	return nil
}

// unwrapPikoNamedType returns the embedded concrete reflect.Type when v holds a
// pikoNamedType wrapper, else v unchanged.
//
// Takes v (reflect.Value).
//
// Returns the unwrapped reflect.Value.
func unwrapPikoNamedType(v reflect.Value) reflect.Value {
	if !v.IsValid() {
		return v
	}
	if named, ok := reflect.TypeAssert[pikoNamedType](v); ok {
		return reflect.ValueOf(named.Type)
	}
	if wrapped, ok := reflect.TypeAssert[pikoNamedInterfaceWrapper](v); ok {
		return reflect.ValueOf(wrapped.Type)
	}
	if v.Kind() == reflect.Slice && v.Type().Elem() == reflectTypeReflectType {
		out := reflect.MakeSlice(v.Type(), v.Len(), v.Len())
		for i := range v.Len() {
			out.Index(i).Set(unwrapReflectTypeSliceElement(v.Index(i)))
		}
		return out
	}
	return v
}

// unwrapReflectTypeSliceElement strips a pikoNamedType or pikoNamedInterfaceWrapper from
// a single []reflect.Type element so reflect type-constructor families don't see piko
// wrappers when they internally type-assert to *rtype.
//
// Takes element (reflect.Value) which is one slice element of type reflect.Type.
//
// Returns the unwrapped reflect.Value, or element unchanged when it is not a known piko
// wrapper.
func unwrapReflectTypeSliceElement(element reflect.Value) reflect.Value {
	if named, ok := reflect.TypeAssert[pikoNamedType](element); ok {
		return reflect.ValueOf(named.Type)
	}
	if wrapped, ok := reflect.TypeAssert[pikoNamedInterfaceWrapper](element); ok {
		return reflect.ValueOf(wrapped.Type)
	}
	return element
}

// unwrapPikoNamedTypeArguments strips wrappers for reflect constructors.
//
// Unwraps any pikoNamedType wrappers in the argument slice when reflectedFunction is one
// of the reflect type-constructor family. reflect.PointerTo / SliceOf / ... panic on a
// pikoNamedType because they down-assert to *reflect.rtype; the wrapper exists only so
// reflect.TypeOf(...).Name() reports the piko source name, and that identity is not
// needed by the constructors.
//
// Takes reflectedFunction (reflect.Value) which is the native callee.
// Takes arguments ([]reflect.Value) which is mutated in place.
func unwrapPikoNamedTypeArguments(reflectedFunction reflect.Value, arguments []reflect.Value) {
	if reflectedFunction.Kind() != reflect.Func {
		return
	}
	if _, ok := reflectTypeConstructorPointers[reflectedFunction.Pointer()]; !ok {
		return
	}
	for i := range arguments {
		arguments[i] = unwrapPikoNamedType(arguments[i])
	}
}

// shimReflectMakeFuncImpl rewrites the impl argument of a reflect.MakeFunc call so the
// values it returns are converted to the exact element types the target funcType
// declares.
//
// piko boxes scalars from its register banks at canonical widths (int64/float64/...).
// When an interpreted reflect.MakeFunc impl returns, say, an int64 but the funcType
// declares an int result, reflect.MakeFunc's trampoline rejects the assignment and
// panics. The shim intercepts each returned []reflect.Value and Converts any element
// whose type differs from funcType.Out(i) (bug 756).
//
// Takes reflectedFunction (reflect.Value) which is the native callee.
// Takes arguments ([]reflect.Value) which is mutated in place when the callee is
// reflect.MakeFunc.
func shimReflectMakeFuncImpl(reflectedFunction reflect.Value, arguments []reflect.Value) {
	if reflectedFunction.Kind() != reflect.Func || reflectedFunction.Pointer() != reflectMakeFuncPointer {
		return
	}
	if len(arguments) != 2 || !arguments[0].IsValid() || !arguments[1].IsValid() {
		return
	}
	funcType, ok := reflect.TypeAssert[reflect.Type](arguments[0])
	if !ok || funcType == nil || funcType.Kind() != reflect.Func {
		return
	}
	inner := arguments[1]
	if inner.Kind() != reflect.Func {
		return
	}
	arguments[1] = makeFuncImplConvertingShim(funcType, inner)
}

// makeFuncImplConvertingShim builds a replacement reflect.MakeFunc impl that forwards to
// inner and converts each value the impl returns to the exact element type funcType
// declares.
//
// Takes funcType (reflect.Type) which is the target function type whose Out(i) element
// types the returned values must match.
// Takes inner (reflect.Value) which is the original interpreted impl closure.
//
// Returns a reflect.Value holding the wrapped impl closure.
func makeFuncImplConvertingShim(funcType reflect.Type, inner reflect.Value) reflect.Value {
	return reflect.MakeFunc(inner.Type(), func(in []reflect.Value) []reflect.Value {
		innerOut := inner.Call(in)
		if len(innerOut) != 1 || !innerOut[0].IsValid() {
			return innerOut
		}
		results, ok := reflect.TypeAssert[[]reflect.Value](innerOut[0])
		if !ok {
			return innerOut
		}
		convertMakeFuncResults(funcType, results)
		return []reflect.Value{reflect.ValueOf(results)}
	})
}

// convertMakeFuncResults converts each element of results in place to the corresponding
// declared Out(i) type of funcType when the runtime type differs and a conversion is
// possible.
//
// Takes funcType (reflect.Type) which declares the target Out types.
// Takes results ([]reflect.Value) which is mutated in place.
func convertMakeFuncResults(funcType reflect.Type, results []reflect.Value) {
	for i := range results {
		if i >= funcType.NumOut() || !results[i].IsValid() {
			continue
		}
		want := funcType.Out(i)
		if results[i].Type() != want && results[i].Type().ConvertibleTo(want) {
			results[i] = results[i].Convert(want)
		}
	}
}

// applyPikoReflectTypeOfNaming re-clothes the result of a reflect.TypeOf call so the
// rendered name preserves source-level identity that piko's type converter would
// otherwise lose.
//
// When the argument was a piko-defined named primitive, the result is wrapped in
// pikoNamedType so .Name() / .String() report the source-level name (e.g. `type Score
// int` reports "Score" not "int"). When the argument was a typed-nil pointer to a
// user-declared named interface (e.g. `(*myiface)(nil)`), the result is wrapped in
// pikoNamedInterfaceWrapper so .Elem().String() reports the source-level interface name
// (e.g. "main.myiface") instead of the lossy "interface {}" that Go's reflect produces
// for piko's synthesised `*interface{}` type.
//
// Takes vm (*VM) which provides the globalStore for user-interface lookups. May be nil
// for tests; the user-interface wrap is then skipped.
// Takes reflectedFunction (reflect.Value) which is the native callee; the wrap is applied
// only when it is reflect.TypeOf.
// Takes site (*callSite) which carries the per-argument static type metadata recorded by
// the compiler.
// Takes results ([]reflect.Value) which is the native call's return slice.
//
// Returns results unchanged unless a wrap applies.
func applyPikoReflectTypeOfNaming(vm *VM, reflectedFunction reflect.Value, site *callSite, results []reflect.Value) []reflect.Value {
	if reflectedFunction.Kind() != reflect.Func || reflectedFunction.Pointer() != reflectTypeOfPointer {
		return results
	}
	if len(results) != 1 || !results[0].IsValid() {
		return results
	}
	if len(site.argumentStaticTypeNames) == 0 {
		return results
	}
	sourceName := site.argumentStaticTypeNames[0]
	if sourceName == "" {
		return results
	}
	resolvedType, ok := reflect.TypeAssert[reflect.Type](results[0])
	if !ok || resolvedType == nil {
		return results
	}
	if resolvedType.Name() == sourceName {
		return results
	}
	sourcePackage := pikoTypeOfSourcePackage(site)
	if isScalarReflectKind(resolvedType.Kind()) && !isBuiltinScalarTypeName(sourceName) {
		results[0] = reflect.ValueOf(pikoNamedType{
			Type:          resolvedType,
			sourceName:    sourceName,
			sourcePackage: sourcePackage,
		})
		return results
	}
	if wrapped, ok := tryWrapNamedInterfacePointer(vm, site, sourceName, sourcePackage, resolvedType); ok {
		results[0] = reflect.ValueOf(wrapped)
		return results
	}
	return results
}

// pikoTypeOfSourcePackage extracts the package short name.
//
// Reads the call site's recorded static type string for the first argument; e.g.
// "*main.myiface" -> "main".
//
// Takes site (*callSite) which carries the recorded static type strings.
//
// Returns string which is the package short name, or empty.
func pikoTypeOfSourcePackage(site *callSite) string {
	if len(site.argumentStaticTypeStrings) == 0 {
		return ""
	}
	s := site.argumentStaticTypeStrings[0]

	for len(s) > 0 && (s[0] == '*' || s[0] == '[' || s[0] == ']') {
		if s[0] == '[' {
			if closeIdx := indexByteString(s, ']'); closeIdx > 0 {
				s = s[closeIdx+1:]
				continue
			}
			return ""
		}
		s = s[1:]
	}
	if dot := indexByteString(s, '.'); dot > 0 {
		return s[:dot]
	}
	return ""
}

// tryWrapNamedInterfacePointer wraps named-interface pointer types.
//
// Detects a *interface{} reflect.Type (the piko-synth shape for a typed-nil pointer to a
// user-declared named interface) and wraps it in a pikoNamedInterfaceWrapper carrying the
// source-level identity from the per-Service registry. The static type string is required
// to be "*"-prefixed because that is how piko's compiler renders `(*myiface)(nil)` for
// the call site.
//
// Takes vm (*VM) which provides globalStore; nil means no wrap.
// Takes site (*callSite) which carries argumentStaticTypeStrings.
// Takes sourceName (string) which is the bare interface name.
// Takes sourcePackage (string) which is the short package name.
// Takes resolvedType (reflect.Type) which the call produced.
//
// Returns pikoNamedInterfaceWrapper which carries the wrapped type.
// Returns bool which is true on success.
func tryWrapNamedInterfacePointer(vm *VM, site *callSite, sourceName, sourcePackage string, resolvedType reflect.Type) (pikoNamedInterfaceWrapper, bool) {
	if vm == nil || vm.globals == nil {
		return pikoNamedInterfaceWrapper{}, false
	}
	if resolvedType.Kind() != reflect.Pointer || resolvedType.Elem().Kind() != reflect.Interface {
		return pikoNamedInterfaceWrapper{}, false
	}
	if sourcePackage == "" {
		return pikoNamedInterfaceWrapper{}, false
	}
	qualifiedInner := sourcePackage + "." + sourceName
	innerPiko, ok := vm.globals.lookupUserNamedInterface(qualifiedInner)
	if !ok {
		return pikoNamedInterfaceWrapper{}, false
	}

	if len(site.argumentStaticTypeStrings) > 0 {
		if !startsWithStarOfQualified(site.argumentStaticTypeStrings[0], qualifiedInner) {
			return pikoNamedInterfaceWrapper{}, false
		}
	}
	ptrPiko := pikoTypeOfPointer(innerPiko)
	return newPikoNamedInterfaceWrapper(resolvedType, ptrPiko), true
}

// startsWithStarOfQualified checks for a "*<qualified>" prefix.
//
// Verifies the call site's static type is a pointer to the named interface that was
// resolved.
//
// Takes s (string) which is the static type string to inspect.
// Takes qualified (string) which is the expected qualified name.
//
// Returns bool which is true when s starts with "*<qualified>".
func startsWithStarOfQualified(s, qualified string) bool {
	if len(s) < len(qualified)+1 || s[0] != '*' {
		return false
	}
	return s[1:len(qualified)+1] == qualified
}

// isScalarReflectKind reports whether k is a numeric, boolean, or string kind - the only
// kinds for which pikoNamedType wrapping is applied. Struct / slice / map / pointer types
// are left untouched so reflect.Type identity comparisons and the existing struct-field
// sentinel filtering keep working.
//
// Takes k (reflect.Kind).
//
// Returns true for scalar kinds.
func isScalarReflectKind(k reflect.Kind) bool {
	switch k {
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64,
		reflect.Complex64, reflect.Complex128,
		reflect.String:
		return true
	default:
	}
	return false
}

// indexByteString returns the index of the first occurrence of b in s, or -1. Thin
// wrapper over strings.IndexByte, which the compiler intrinsifies to an
// architecture-tuned scan; the wrapper keeps the call sites' intent legible.
//
// Takes s (string) and b (byte).
//
// Returns the index, or -1 when absent.
func indexByteString(s string, b byte) int {
	return strings.IndexByte(s, b)
}

// wellKnownNamedInterfaceReflectType returns canonical stdlib types.
//
// Yields the cached reflect.Type for a well-known stdlib named interface so compileType
// does not collapse e.g. `error` or `fmt.Stringer` to `interface{}` at compile time. The
// boundary adapter at tryBuildInterfaceAdapter already caches these exact reflect.Types,
// so re-using them keeps the typeTableInterfaceMethods sidecar well-behaved for type
// assertions while restoring the natural type name everywhere reflect surfaces.
//
// Lookup is on the (package path, type name) pair. The builtin `error` type has obj.Pkg()
// == nil and is matched by name only.
//
// Takes pkgPath (string) which is the importer-visible package path.
// Takes typeName (string) which is the bare type name.
//
// Returns reflect.Type which is the cached reflect.Type on hit.
// Returns bool which is true on hit; false otherwise so callers fall through to
// convertUnderlying.
func wellKnownNamedInterfaceReflectType(pkgPath, typeName string) (reflect.Type, bool) {
	if pkgPath == "" {
		if typeName == "error" {
			return errorReflectType, true
		}
		return nil, false
	}
	key := pkgPath + "." + typeName
	rt, ok := wellKnownNamedInterfaceRegistry[key]
	return rt, ok
}

// extractBytesResult returns the first reflect.Value in results as []byte when valid and
// type-compatible, or nil otherwise.
//
// Takes results ([]reflect.Value) which are the method invocation results.
//
// Returns []byte which is the converted first result, or nil.
func extractBytesResult(results []reflect.Value) []byte {
	if len(results) < 1 || !results[0].IsValid() {
		return nil
	}
	bytes, ok := reflect.TypeAssert[[]byte](results[0])
	if !ok {
		return nil
	}
	return bytes
}

// extractErrorResult returns the second reflect.Value in results as error when valid and
// type-compatible, or nil otherwise.
//
// Takes results ([]reflect.Value) which are the method invocation results.
//
// Returns error which is the converted second result, or nil.
func extractErrorResult(results []reflect.Value) error {
	if len(results) < 2 || !results[1].IsValid() {
		return nil
	}
	errInterface, ok := reflect.TypeAssert[error](results[1])
	if !ok {
		return nil
	}
	return errInterface
}

// buildMarshalerAdapterIfRegistered returns a pikoMarshalerAdapter wrapping argument when
// the type-name has a registered MarshalJSON method, or an invalid value otherwise.
//
// Takes vm (*VM) which provides the method registry.
// Takes argument (reflect.Value) which is the value being wrapped.
// Takes typeName (string) which is the source-level type name used for method-table
// lookup.
//
// Returns a wrapped reflect.Value implementing json.Marshaler, or an invalid
// reflect.Value when no MarshalJSON method is registered.
func buildMarshalerAdapterIfRegistered(vm *VM, argument reflect.Value, typeName string) reflect.Value {
	methodRoot, methodIndex, ok := lookupAdapterMethod(vm, typeName+".MarshalJSON")
	if !ok {
		return reflect.Value{}
	}
	if !methodReceiverSatisfiesValueIn(methodRoot, vm, methodIndex, argument) {
		return reflect.Value{}
	}
	return reflect.ValueOf(&pikoMarshalerAdapter{
		vm:          vm,
		methodRoot:  methodRoot,
		underlying:  argument,
		methodIndex: methodIndex,
	})
}

// invokeStringReturnMethod dispatches a bound method whose first result is a string,
// returning that string or "" when dispatch is not possible (nil VM, unresolved method
// index, or empty/invalid first result).
//
// When methodRoot is non-nil it overrides vm.rootFunction for the dispatch - the
// cross-package case where the method body lives in another CompileProgram's
// rootFunction.
//
// Takes vm (*VM) which provides the dispatch root.
// Takes methodRoot (*CompiledFunction) which holds the method body when it lives outside
// vm.rootFunction; nil for the in-package case.
// Takes methodIndex (uint16) which selects the compiled method inside methodRoot (or
// vm.rootFunction when methodRoot is nil).
// Takes underlying (reflect.Value) which is the receiver value.
//
// Returns string which is the first result of the bound method or "".
func invokeStringReturnMethod(vm *VM, methodRoot *CompiledFunction, methodIndex uint16, underlying reflect.Value) string {
	callee, root, ok := resolveAdapterCallee(vm, methodRoot, methodIndex)
	if !ok {
		return ""
	}
	bound := newCrossPackageBoundMethod(vm, root, callee)
	receiver := receiverValueFor(callee, underlying)
	results := bound.invoke(receiver, nil, identityArg)
	if len(results) == 0 || !results[0].IsValid() {
		return ""
	}
	return results[0].String()
}

// resolveAdapterCallee picks the callee and its rootFunction.
//
// When methodRoot is non-nil the entry is in that root; otherwise it is in
// vm.rootFunction. Returns ok=false when either input is malformed, matching the adapter
// behaviour of returning a zero result rather than panicking.
//
// Takes vm (*VM) which is the parent VM.
// Takes methodRoot (*CompiledFunction) which is the cross-package override; nil for
// in-package methods.
// Takes methodIndex (uint16) which is the slot inside the chosen rootFunction's
// functions.
//
// Returns (callee, rootFunction, true) on success; (nil, nil, false) on lookup failure.
func resolveAdapterCallee(vm *VM, methodRoot *CompiledFunction, methodIndex uint16) (callee, rootFunction *CompiledFunction, ok bool) {
	if vm == nil {
		return nil, nil, false
	}
	root := methodRoot
	if root == nil {
		root = vm.rootFunction
	}
	if root == nil {
		return nil, nil, false
	}
	if int(methodIndex) >= len(root.functions) {
		return nil, nil, false
	}
	callee = root.functions[methodIndex]
	if callee == nil {
		return nil, nil, false
	}
	return callee, root, true
}

// newCrossPackageBoundMethod constructs a boundMethodVM whose rootFunctionOverride is set
// when the method body lives in a different CompileProgram batch than the parent VM's
// root. When the method is local (root == vm.rootFunction) the override stays nil and
// boundMethodVM.invoke takes its existing in-package path.
//
// Takes vm (*VM) which is the parent VM.
// Takes root (*CompiledFunction) which owns callee.
// Takes callee (*CompiledFunction) which is the method body to run.
//
// Returns a boundMethodVM ready for invoke().
func newCrossPackageBoundMethod(vm *VM, root *CompiledFunction, callee *CompiledFunction) *boundMethodVM {
	bound := &boundMethodVM{vm: vm, callee: callee, limits: vm.limits}
	if root != nil && root != vm.rootFunction {
		bound.rootFunctionOverride = root
	}
	return bound
}

// lookupAdapterMethod resolves a "TypeName.MethodName" entry to (rootFunction,
// methodIndex, ok). The in-package methodTable wins over the cross-package external
// registry - matching the spirit of Go's import lookup where a same-named
// locally-declared method shadows an imported one.
//
// Takes vm (*VM) which provides both lookup spaces.
// Takes key (string) which is "TypeName.MethodName".
//
// Returns the resolved entry. methodRoot is nil when the entry is local to
// vm.rootFunction.
func lookupAdapterMethod(vm *VM, key string) (methodRoot *CompiledFunction, methodIndex uint16, ok bool) {
	if vm == nil {
		return nil, 0, false
	}
	if vm.rootFunction != nil {
		if idx, found := vm.rootFunction.methodTable[key]; found {
			return nil, idx, true
		}
	}
	if vm.globals == nil {
		return nil, 0, false
	}
	entry, found := vm.globals.lookupExternalMethod(key)
	if !found {
		return nil, 0, false
	}
	return entry.rootFunction, entry.methodIndex, true
}

// buildFormatterAdapterIfRegistered returns a pikoFormatterAdapter wrapping argument when
// the type-name has a registered Format method, or an invalid value otherwise.
//
// Takes vm (*VM) which provides the method registry.
// Takes argument (reflect.Value) which is the value being wrapped.
// Takes typeName (string) which is the source-level type name.
//
// Returns a wrapped reflect.Value implementing fmt.Formatter, or an invalid reflect.Value
// when no Format method is registered.
func buildFormatterAdapterIfRegistered(vm *VM, argument reflect.Value, typeName string) reflect.Value {
	methodRoot, methodIndex, ok := lookupAdapterMethod(vm, typeName+".Format")
	if !ok {
		return reflect.Value{}
	}
	if !methodReceiverSatisfiesValueIn(methodRoot, vm, methodIndex, argument) {
		return reflect.Value{}
	}
	return reflect.ValueOf(&pikoFormatterAdapter{
		vm:          vm,
		methodRoot:  methodRoot,
		underlying:  argument,
		methodIndex: methodIndex,
	})
}

// buildSortInterfaceAdapterIfRegistered returns a pikoSortInterfaceAdapter wrapping
// argument when the type-name has registered Len, Less and Swap methods, or an invalid
// value otherwise.
//
// Takes vm (*VM) which provides the method registry.
// Takes argument (reflect.Value) which is the value being wrapped.
// Takes typeName (string) which is the source-level type name.
//
// Returns a wrapped reflect.Value implementing sort.Interface, or an invalid
// reflect.Value.
func buildSortInterfaceAdapterIfRegistered(vm *VM, argument reflect.Value, typeName string) reflect.Value {
	lenRoot, lenIndex, hasLen := lookupAdapterMethod(vm, typeName+".Len")
	lessRoot, lessIndex, hasLess := lookupAdapterMethod(vm, typeName+".Less")
	swapRoot, swapIndex, hasSwap := lookupAdapterMethod(vm, typeName+".Swap")
	if !hasLen || !hasLess || !hasSwap {
		if recovered, ok := pikoTypeName(vm, argument); ok && recovered != typeName {
			lenRoot, lenIndex, hasLen = lookupAdapterMethod(vm, recovered+".Len")
			lessRoot, lessIndex, hasLess = lookupAdapterMethod(vm, recovered+".Less")
			swapRoot, swapIndex, hasSwap = lookupAdapterMethod(vm, recovered+".Swap")
		}
	}
	if !hasLen || !hasLess || !hasSwap {
		return reflect.Value{}
	}

	if lenRoot != lessRoot || lessRoot != swapRoot {
		return reflect.Value{}
	}
	if !methodReceiverSatisfiesValueIn(lenRoot, vm, lenIndex, argument) ||
		!methodReceiverSatisfiesValueIn(lessRoot, vm, lessIndex, argument) ||
		!methodReceiverSatisfiesValueIn(swapRoot, vm, swapIndex, argument) {
		return reflect.Value{}
	}
	return reflect.ValueOf(&pikoSortInterfaceAdapter{
		vm:         vm,
		methodRoot: lenRoot,
		underlying: argument,
		lenIndex:   lenIndex,
		lessIndex:  lessIndex,
		swapIndex:  swapIndex,
	})
}

// buildScannerAdapterIfRegistered returns a pikoScannerAdapter wrapping argument when the
// type-name has a registered Scan method, or an invalid value otherwise.
//
// Takes vm (*VM) which provides the method registry.
// Takes argument (reflect.Value) which is the value being wrapped.
// Takes typeName (string) which is the source-level type name.
//
// Returns a wrapped reflect.Value implementing fmt.Scanner, or an invalid reflect.Value
// when no Scan method is registered.
func buildScannerAdapterIfRegistered(vm *VM, argument reflect.Value, typeName string) reflect.Value {
	methodRoot, methodIndex, ok := lookupAdapterMethod(vm, typeName+".Scan")
	if !ok {
		return reflect.Value{}
	}
	if !methodReceiverSatisfiesValueIn(methodRoot, vm, methodIndex, argument) {
		return reflect.Value{}
	}
	return reflect.ValueOf(&pikoScannerAdapter{
		vm:          vm,
		methodRoot:  methodRoot,
		underlying:  argument,
		methodIndex: methodIndex,
	})
}

// buildWriterAdapterIfRegistered returns a pikoWriterAdapter wrapping argument when the
// type-name has a registered Write method, or an invalid value otherwise.
//
// Takes vm (*VM) which provides the method registry.
// Takes argument (reflect.Value) which is the value being wrapped.
// Takes typeName (string) which is the source-level type name used for method-table
// lookup.
//
// Returns a wrapped reflect.Value implementing io.Writer, or an invalid reflect.Value
// when no Write method is registered.
func buildWriterAdapterIfRegistered(vm *VM, argument reflect.Value, typeName string) reflect.Value {
	methodRoot, methodIndex, ok := lookupAdapterMethod(vm, typeName+".Write")
	if !ok {
		return reflect.Value{}
	}
	if !methodReceiverSatisfiesValueIn(methodRoot, vm, methodIndex, argument) {
		return reflect.Value{}
	}
	return reflect.ValueOf(&pikoWriterAdapter{
		vm:          vm,
		methodRoot:  methodRoot,
		underlying:  argument,
		methodIndex: methodIndex,
	})
}

// buildUnmarshalerAdapterIfRegistered returns a pikoUnmarshalerAdapter wrapping argument
// when the type-name has a registered UnmarshalJSON method, or an invalid value
// otherwise.
//
// Takes vm (*VM) which provides the method registry.
// Takes argument (reflect.Value) which is the value being wrapped.
// Takes typeName (string) which is the source-level type name used for method-table
// lookup.
//
// Returns a wrapped reflect.Value implementing json.Unmarshaler, or an invalid
// reflect.Value when no UnmarshalJSON method is registered (or when the method has a
// pointer receiver but the argument is not addressable).
func buildUnmarshalerAdapterIfRegistered(vm *VM, argument reflect.Value, typeName string) reflect.Value {
	methodRoot, methodIndex, ok := lookupAdapterMethod(vm, typeName+".UnmarshalJSON")
	if !ok {
		return reflect.Value{}
	}
	if !methodReceiverSatisfiesValueIn(methodRoot, vm, methodIndex, argument) {
		return reflect.Value{}
	}
	return reflect.ValueOf(&pikoUnmarshalerAdapter{
		vm:          vm,
		methodRoot:  methodRoot,
		underlying:  argument,
		methodIndex: methodIndex,
	})
}

// buildReaderAdapterIfRegistered returns a pikoReaderAdapter wrapping argument when the
// type-name has a registered Read method, or an invalid value otherwise.
//
// Takes vm (*VM) which provides the method registry.
// Takes argument (reflect.Value) which is the value being wrapped.
// Takes typeName (string) which is the source-level type name used for method-table
// lookup.
//
// Returns a wrapped reflect.Value implementing io.Reader, or an invalid reflect.Value
// when no Read method is registered.
func buildReaderAdapterIfRegistered(vm *VM, argument reflect.Value, typeName string) reflect.Value {
	methodRoot, methodIndex, ok := lookupAdapterMethod(vm, typeName+".Read")
	if !ok {
		return reflect.Value{}
	}
	if !methodReceiverSatisfiesValueIn(methodRoot, vm, methodIndex, argument) {
		return reflect.Value{}
	}
	return reflect.ValueOf(&pikoReaderAdapter{
		vm:          vm,
		methodRoot:  methodRoot,
		underlying:  argument,
		methodIndex: methodIndex,
	})
}

// receiverValueFor coerces an adapter's underlying value to match the callee's receiver
// kind.
//
// Pointer-receiver methods get the address (so bound-method dispatch can take Addr in the
// standard way); value-receiver methods get the raw value (so reading a typed primitive
// such as time.Duration or Colour returns the actual numeric value rather than reading
// through a *int64 wrapper the body never unboxes).
//
// Takes callee (*CompiledFunction) which records isPointerReceiver for the source-level
// method.
// Takes underlying (reflect.Value) which is the adapter's wrapped value.
//
// Returns the receiver shape suitable for boundMethodVM.invoke.
func receiverValueFor(callee *CompiledFunction, underlying reflect.Value) reflect.Value {
	if callee != nil && !callee.isPointerReceiver {
		return underlying
	}
	return pointerReceiverFor(underlying)
}

// pointerReceiverFor coerces a value to the address shape needed by pointer-receiver
// dispatch.
//
// Already-pointer values pass through; addressable values return their Addr;
// non-addressable struct values are copied into a fresh reflect.New so the callee can
// take Addr in the standard way bound method dispatch expects.
//
// Takes value (reflect.Value) which is the piko-synthesised value.
//
// Returns a reflect.Value usable as the receiver argument to bound method dispatch.
func pointerReceiverFor(value reflect.Value) reflect.Value {
	if !value.IsValid() {
		return value
	}
	if value.Kind() == reflect.Pointer {
		return value
	}
	if value.CanAddr() {
		return value.Addr()
	}
	addressable := reflect.New(value.Type())
	addressable.Elem().Set(value)
	return addressable
}

// buildErrorAdapterIfRegistered returns a pikoErrorAdapter wrapping argument when the
// type-name has a registered Error method, or an invalid value otherwise.
//
// Takes vm (*VM) which provides the method registry.
// Takes argument (reflect.Value) which is the value being wrapped.
// Takes typeName (string) which is the source-level type name used for method-table
// lookup.
//
// Returns a wrapped reflect.Value implementing error, or an invalid reflect.Value when no
// Error method is registered.
func buildErrorAdapterIfRegistered(vm *VM, argument reflect.Value, typeName string) reflect.Value {
	methodRoot, methodIndex, ok := lookupAdapterMethod(vm, typeName+".Error")
	if !ok {
		return reflect.Value{}
	}
	if !methodReceiverSatisfiesValueIn(methodRoot, vm, methodIndex, argument) {
		return reflect.Value{}
	}
	return reflect.ValueOf(&pikoErrorAdapter{
		vm:          vm,
		methodRoot:  methodRoot,
		underlying:  argument,
		methodIndex: methodIndex,
	})
}

// methodReceiverSatisfiesValueIn enforces Go's method-set rule.
//
// A method declared on `*T` is in T's method set only when the T value is addressable; a
// method declared on `T` (value receiver) is in both T's and *T's method set. Variant of
// methodReceiverSatisfiesValue that is cross-package aware: when methodRoot is non-nil
// the callee is looked up in methodRoot.functions instead of vm.rootFunction.functions,
// the same indirection invokeStringReturnMethod uses.
//
// Takes methodRoot (*CompiledFunction) which owns the callee; nil for the in-package
// case.
// Takes vm (*VM) which is the parent virtual machine.
// Takes methodIndex (uint16) which selects the callee.
// Takes argument (reflect.Value) which is the value about to be adapted.
//
// Returns true when adapter construction may proceed.
func methodReceiverSatisfiesValueIn(methodRoot *CompiledFunction, vm *VM, methodIndex uint16, argument reflect.Value) bool {
	callee, _, ok := resolveAdapterCallee(vm, methodRoot, methodIndex)
	if !ok || callee == nil {
		return true
	}
	if !callee.isPointerReceiver {
		return true
	}
	if !argument.IsValid() {
		return false
	}
	probe := argument
	if probe.Kind() == reflect.Interface {
		if probe.IsNil() {
			return false
		}
		probe = probe.Elem()
	}
	return probe.Kind() == reflect.Pointer
}

// buildStringerAdapterIfRegistered returns a pikoStringerAdapter wrapping argument when
// the type-name has a registered String method, or an invalid value otherwise.
//
// Takes vm (*VM) which provides the method registry.
// Takes argument (reflect.Value) which is the value being wrapped.
// Takes typeName (string) which is the source-level type name used for method-table
// lookup.
//
// Returns a wrapped reflect.Value implementing fmt.Stringer, or an invalid reflect.Value
// when no String method is registered.
func buildStringerAdapterIfRegistered(vm *VM, argument reflect.Value, typeName string) reflect.Value {
	methodRoot, methodIndex, ok := lookupAdapterMethod(vm, typeName+".String")
	if !ok {
		return reflect.Value{}
	}
	if !methodReceiverSatisfiesValueIn(methodRoot, vm, methodIndex, argument) {
		return reflect.Value{}
	}
	return reflect.ValueOf(&pikoStringerAdapter{
		vm:          vm,
		methodRoot:  methodRoot,
		underlying:  argument,
		methodIndex: methodIndex,
	})
}

// pikoTypeName resolves the source-level type name for a value whose reflect.Type was
// synthesised by piko (via reflect.StructOf with the _pikoID_ sentinel field). Looks the
// type up in the root function's typeNames registry, transparently unwrapping pointer
// kinds so that both *MyErr and MyErr resolve to the same name.
//
// Takes vm (*VM) which provides typeNames access.
// Takes value (reflect.Value) which is the runtime value to identify.
//
// Returns the source-level type name and true on success, or "" and false when the
// value's type is not a piko-synthesised type.
func pikoTypeName(vm *VM, value reflect.Value) (string, bool) {
	if !value.IsValid() {
		return "", false
	}
	if vm == nil || vm.rootFunction == nil {
		return "", false
	}
	t := value.Type()
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	name, ok := vm.rootFunction.typeNames[t]
	return name, ok
}
