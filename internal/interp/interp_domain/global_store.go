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
	"strings"
	"sync"
	"sync/atomic"

	"piko.sh/piko/wdk/safeconv"
)

const (
	// externalMethodsInitialCapacity is the starting size for the lazy-allocated
	// externalMethods map. Sized for typical small method tables; the map grows naturally
	// past this.
	externalMethodsInitialCapacity = 8
)

// globalStore holds package-level variables. It is shared across all function invocations
// within the same package and across goroutines, so access is synchronised.
//
// Performance note: in the (overwhelmingly common) case where the program never launches
// a goroutine, getter/setter calls take a lock-free fast path guarded by a single
// atomic.Bool load. The mutex is only acquired once a goroutine has actually been
// launched (see (*VM).handleGo, which calls markShared). Once shared, the flag stays set
// for the lifetime of the store; toggling it back would race with concurrent readers.
type globalStore struct {
	// goroutinePanic captures the first unrecovered panic from any spawned goroutine running
	// on this store. Parent VMs check this in their run loop and surface it as an evaluation
	// error so the goroutine panic propagates instead of being silently swallowed.
	goroutinePanic atomic.Pointer[goroutinePanicInfo]

	// goroutinePanicSignal is closed exactly once by the panic-CAS winner.
	//
	// Interpreted blocking channel and select operations select on it (only once the store
	// is shared, that is at least one goroutine has launched) so a goroutine parked on a
	// receive wakes when a sibling panics, instead of deadlocking until the context
	// deadline. Recreated by reset so a reused store starts afresh.
	goroutinePanicSignal chan struct{}

	// externalMethods registers methods from packages compiled in separate
	// Service.CompileProgram calls so cross-package interface-adapter lookups can locate
	// them.
	//
	// Within a single CompileProgram all packages share one rootFunction whose methodTable
	// already covers every package in the call. But when uuid is compiled in one
	// CompileProgram and main in another (the pipit modloader flow), main's
	// rootFunction.methodTable has only main's methods. Without the fallback registered
	// here, the runtime adapter builders (pikoStringerAdapter, pikoSortInterfaceAdapter,
	// ...) would miss uuid.UUID.String and the native callee would see piko's
	// `_pikoID_`-bearing struct without methods.
	//
	// Keyed by `"TypeName.MethodName"` - identical to the per-CompiledFunction methodTable
	// convention. Each entry pins the source rootFunction so adapter dispatch can swap to it
	// (the same way runtimeClosure already does for plain function calls).
	//
	// Populated from Service.bridgePackageExports when a non-main package is bridged.
	// Read-mostly and updated only at CompileProgram completion, far from the dispatch hot
	// path, so the externalMethodsMu RWMutex guards it.
	externalMethods map[string]externalMethodEntry

	// userNamedInterfaces records source-level identity for user-declared named interface
	// types. Keyed by qualifiedName ("pkg.Iface"); the pikoType carries the methodNames set
	// used by Implements / AssignableTo / Method dispatch via pikoNamedInterfaceWrapper.
	//
	// Populated by compiler_types.convertNamedType when it encounters a named interface that
	// is NOT in wellKnownNamedInterfaceRegistry, looked up at runtime by
	// applyPikoReflectTypeOfNaming when wrapping a *interface{} result from
	// reflect.TypeOf((*UserIface)(nil)).
	//
	// Guarded by externalMethodsMu (same lock - both bridges are populated together by
	// bridgePackageExports).
	userNamedInterfaces map[string]*pikoType

	// internedMapKeys holds heap-cloned canonical copies of map-key strings observed across
	// all VMs that share this Execute's globalStore. The dominant case for word-counting
	// workloads with finite vocabularies is a cache hit.
	//
	// Implemented as sync.Map (not RWMutex + map) so the hot read path is a single
	// atomic.LoadPointer on a pointer that rarely changes - each core caches the line in
	// shared state and avoids the cache-line ping-pong that an RWMutex.RLock atomic.AddInt32
	// would incur with many concurrent goroutines.
	//
	// Lives on globalStore (not VM) because spawned goroutines get fresh per-goroutine VMs
	// (runCompiledGoroutine) but share the parent's globalStore pointer. Sharing the intern
	// here collapses the strings.Clone cost across all workers rather than paying it per VM.
	internedMapKeys sync.Map

	// uints holds all package-level uint64 global variables.
	uints []uint64

	// general holds all package-level reflect.Value global variables.
	general []reflect.Value

	// bools holds all package-level bool global variables.
	bools []bool

	// strings holds all package-level string global variables.
	strings []string

	// complexes holds all package-level complex128 global variables.
	complexes []complex128

	// floats holds all package-level float64 global variables.
	floats []float64

	// ints holds all package-level int64 global variables.
	ints []int64

	// mu guards concurrent access to all fields when shared is true.
	mu sync.RWMutex

	// externalMethodsMu guards externalMethods and userNamedInterfaces against concurrent
	// registration and lookup.
	externalMethodsMu sync.RWMutex

	// interpreterLock serialises interpreted bytecode execution across every VM that shares
	// this store, but only under the runtime safe mode.
	//
	// At most one interpreted goroutine runs bytecode at a time, so multi-word writes to
	// shared upvalue cells, maps, and slice headers can never tear or race. It is released
	// around blocking channel ops and at the periodic checkpoint so siblings make progress,
	// and is never locked outside safe mode so fast mode pays nothing.
	interpreterLock sync.Mutex

	// shared toggles the locked vs fast-path mode for this store.
	//
	// False until any VM using this store launches its first goroutine; while false,
	// getter/setter calls skip the mutex (single-threaded fast path). Toggled true once and
	// never reset during normal operation. Read with Load on the hot path; written with
	// Store from handleGo immediately before launching the first goroutine, so the launched
	// goroutine starts in shared mode.
	shared atomic.Bool

	// dispatchDepth counts active piko recover-boundary frames.
	//
	// Bumped at safeReflectCall / guardChannelOp / dispatchNativeFastPath entry, decremented
	// on exit. The MakeFunc closure wrappers (coerceClosureToFunc / closureCallableValue)
	// consult it before re-panicking: when > 0 a piko boundary upstream will catch the
	// panic, so re-panic is safe; when 0 the wrapper was invoked from external host code
	// with no piko recover net upstream, so the wrapper swallows the panic (logging it) to
	// keep the host process contained.
	dispatchDepth atomic.Int32
}

// externalMethodEntry locates a method body across CompileProgram batches.
//
// Captures which rootFunction owns the body and what index that function sits at inside
// the owner's functions slice. The pair mirrors what `runtimeClosure` already carries for
// plain functions, keyed by method name for adapter lookup.
type externalMethodEntry struct {
	// rootFunction is the source package's compiled root, used by boundMethodVM to pick the
	// right functions slice.
	rootFunction *CompiledFunction

	// methodIndex is the method's position in rootFunction.functions.
	methodIndex uint16
}

// goroutinePanicInfo describes a panic captured from a spawned goroutine that the parent
// VM has not yet observed.
type goroutinePanicInfo struct {
	// value is the panic value as it was thrown.
	value any

	// stack is the captured stack trace from the panicking goroutine.
	stack string
}

// markShared transitions the store into "shared" mode.
//
// Subsequent getter/setter calls take the locked path; calls already running on the fast
// path are unaffected (the spawning code in handleGo guarantees no other goroutine is
// running yet when the transition occurs, so any concurrent unlocked reads/writes are by
// the same goroutine that just called markShared).
func (g *globalStore) markShared() {
	g.shared.Store(true)
}

// lengths returns the current bank lengths as a SlotAllocation.
//
// Used at Service.PackageModule entry and exit to compute how many slots a single compile
// reserved (the delta).
//
// Returns SlotAllocation which holds the per-kind slot counts.
//
// Concurrency: safe to call from any goroutine; takes the shared lock when
// globalStore.shared is set.
func (g *globalStore) lengths() SlotAllocation {
	if g.shared.Load() {
		g.mu.RLock()
		defer g.mu.RUnlock()
	}
	return SlotAllocation{
		safeconv.MustIntToUint16(len(g.ints)),
		safeconv.MustIntToUint16(len(g.floats)),
		safeconv.MustIntToUint16(len(g.strings)),
		safeconv.MustIntToUint16(len(g.general)),
		safeconv.MustIntToUint16(len(g.bools)),
		safeconv.MustIntToUint16(len(g.uints)),
		safeconv.MustIntToUint16(len(g.complexes)),
	}
}

// reserveSlots grows each bank and returns the pre-grow base offsets.
//
// Grows each global-store bank by the corresponding count in alloc and returns the
// per-kind base offsets the reservation occupied. Used at Service.LoadModule time to
// place a bundle's slots in a non-overlapping region of the target Service's globalStore;
// the returned bases are stored on every CompiledFunction in the loaded bundle so the
// VM's global-access handlers shift the encoded operand by the right amount.
//
// Takes alloc (SlotAllocation) which is the per-kind slot count to add.
//
// Returns SlotAllocation which is the pre-grow lengths (base indices).
//
// Concurrency: takes g.mu in write mode when globalStore.shared is set; otherwise mutates
// banks directly on the single-threaded fast path.
func (g *globalStore) reserveSlots(alloc SlotAllocation) SlotAllocation {
	if g.shared.Load() {
		g.mu.Lock()
		defer g.mu.Unlock()
	}
	bases := SlotAllocation{
		safeconv.MustIntToUint16(len(g.ints)),
		safeconv.MustIntToUint16(len(g.floats)),
		safeconv.MustIntToUint16(len(g.strings)),
		safeconv.MustIntToUint16(len(g.general)),
		safeconv.MustIntToUint16(len(g.bools)),
		safeconv.MustIntToUint16(len(g.uints)),
		safeconv.MustIntToUint16(len(g.complexes)),
	}
	if n := int(alloc[registerInt]); n > 0 {
		g.ints = append(g.ints, make([]int64, n)...)
	}
	if n := int(alloc[registerFloat]); n > 0 {
		g.floats = append(g.floats, make([]float64, n)...)
	}
	if n := int(alloc[registerString]); n > 0 {
		g.strings = append(g.strings, make([]string, n)...)
	}
	if n := int(alloc[registerGeneral]); n > 0 {
		g.general = append(g.general, make([]reflect.Value, n)...)
	}
	if n := int(alloc[registerBool]); n > 0 {
		g.bools = append(g.bools, make([]bool, n)...)
	}
	if n := int(alloc[registerUint]); n > 0 {
		g.uints = append(g.uints, make([]uint64, n)...)
	}
	if n := int(alloc[registerComplex]); n > 0 {
		g.complexes = append(g.complexes, make([]complex128, n)...)
	}
	return bases
}

// allocInt allocates a new int64 global variable and returns its index.
//
// Takes initial (int64) which is the starting value for the variable.
//
// Returns the index of the newly allocated variable.
//
// Safe for concurrent use by multiple goroutines.
func (g *globalStore) allocInt(initial int64) int {
	g.mu.Lock()
	defer g.mu.Unlock()
	index := len(g.ints)
	g.ints = append(g.ints, initial)
	return index
}

// allocFloat allocates a new float64 global variable and returns its index.
//
// Takes initial (float64) which is the starting value for the variable.
//
// Returns the index of the newly allocated variable.
//
// Safe for concurrent use by multiple goroutines.
func (g *globalStore) allocFloat(initial float64) int {
	g.mu.Lock()
	defer g.mu.Unlock()
	index := len(g.floats)
	g.floats = append(g.floats, initial)
	return index
}

// allocString allocates a new string global variable and returns its index.
//
// Takes initial (string) which is the starting value for the variable.
//
// Returns the index of the newly allocated variable.
//
// Safe for concurrent use by multiple goroutines.
func (g *globalStore) allocString(initial string) int {
	g.mu.Lock()
	defer g.mu.Unlock()
	index := len(g.strings)
	g.strings = append(g.strings, initial)
	return index
}

// allocGeneral allocates a new reflect.Value global variable and returns its index.
//
// Takes initial (reflect.Value) which is the starting value for the variable.
//
// Returns the index of the newly allocated variable.
//
// Safe for concurrent use by multiple goroutines.
func (g *globalStore) allocGeneral(initial reflect.Value) int {
	g.mu.Lock()
	defer g.mu.Unlock()
	index := len(g.general)
	g.general = append(g.general, initial)
	return index
}

// getInt reads an int64 global variable.
//
// Takes index (int) which is the variable to read.
//
// Returns the current value of the variable.
//
// Safe for concurrent use by multiple goroutines.
func (g *globalStore) getInt(index int) int64 {
	if !g.shared.Load() {
		return g.ints[index]
	}
	g.mu.RLock()
	v := g.ints[index]
	g.mu.RUnlock()
	return v
}

// setInt writes an int64 global variable.
//
// Takes index (int) which is the variable to write.
// Takes v (int64) which is the new value.
//
// Safe for concurrent use by multiple goroutines.
func (g *globalStore) setInt(index int, v int64) {
	if !g.shared.Load() {
		g.ints[index] = v
		return
	}
	g.mu.Lock()
	g.ints[index] = v
	g.mu.Unlock()
}

// getFloat reads a float64 global variable.
//
// Takes index (int) which is the variable to read.
//
// Returns the current value of the variable.
//
// Safe for concurrent use by multiple goroutines.
func (g *globalStore) getFloat(index int) float64 {
	if !g.shared.Load() {
		return g.floats[index]
	}
	g.mu.RLock()
	v := g.floats[index]
	g.mu.RUnlock()
	return v
}

// setFloat writes a float64 global variable.
//
// Takes index (int) which is the variable to write.
// Takes v (float64) which is the new value.
//
// Safe for concurrent use by multiple goroutines.
func (g *globalStore) setFloat(index int, v float64) {
	if !g.shared.Load() {
		g.floats[index] = v
		return
	}
	g.mu.Lock()
	g.floats[index] = v
	g.mu.Unlock()
}

// getString reads a string global variable.
//
// Takes index (int) which is the variable to read.
//
// Returns the current value of the variable.
//
// Safe for concurrent use by multiple goroutines.
func (g *globalStore) getString(index int) string {
	if !g.shared.Load() {
		return g.strings[index]
	}
	g.mu.RLock()
	v := g.strings[index]
	g.mu.RUnlock()
	return v
}

// setString writes a string global variable.
//
// Takes index (int) which is the variable to write.
// Takes v (string) which is the new value.
//
// Safe for concurrent use by multiple goroutines.
func (g *globalStore) setString(index int, v string) {
	if !g.shared.Load() {
		g.strings[index] = v
		return
	}
	g.mu.Lock()
	g.strings[index] = v
	g.mu.Unlock()
}

// getGeneral reads a reflect.Value global variable.
//
// Takes index (int) which is the variable to read.
//
// Returns the current value of the variable.
//
// Safe for concurrent use by multiple goroutines.
func (g *globalStore) getGeneral(index int) reflect.Value {
	if !g.shared.Load() {
		return g.general[index]
	}
	g.mu.RLock()
	v := g.general[index]
	g.mu.RUnlock()
	return v
}

// setGeneral writes a reflect.Value global variable.
//
// Takes index (int) which is the variable to write.
// Takes v (reflect.Value) which is the new value.
//
// Safe for concurrent use by multiple goroutines.
func (g *globalStore) setGeneral(index int, v reflect.Value) {
	if !g.shared.Load() {
		g.general[index] = v
		return
	}
	g.mu.Lock()
	g.general[index] = v
	g.mu.Unlock()
}

// allocBool allocates a new bool global variable and returns its index.
//
// Takes initial (bool) which is the starting value for the variable.
//
// Returns the index of the newly allocated variable.
//
// Safe for concurrent use by multiple goroutines.
func (g *globalStore) allocBool(initial bool) int {
	g.mu.Lock()
	defer g.mu.Unlock()
	index := len(g.bools)
	g.bools = append(g.bools, initial)
	return index
}

// getBool reads a bool global variable.
//
// Takes index (int) which is the variable to read.
//
// Returns the current value of the variable.
//
// Safe for concurrent use by multiple goroutines.
func (g *globalStore) getBool(index int) bool {
	if !g.shared.Load() {
		return g.bools[index]
	}
	g.mu.RLock()
	v := g.bools[index]
	g.mu.RUnlock()
	return v
}

// setBool writes a bool global variable.
//
// Takes index (int) which is the variable to write.
// Takes v (bool) which is the new value.
//
// Safe for concurrent use by multiple goroutines.
func (g *globalStore) setBool(index int, v bool) {
	if !g.shared.Load() {
		g.bools[index] = v
		return
	}
	g.mu.Lock()
	g.bools[index] = v
	g.mu.Unlock()
}

// allocUint allocates a new uint64 global variable and returns its index.
//
// Takes initial (uint64) which is the starting value for the variable.
//
// Returns the index of the newly allocated variable.
//
// Safe for concurrent use by multiple goroutines.
func (g *globalStore) allocUint(initial uint64) int {
	g.mu.Lock()
	defer g.mu.Unlock()
	index := len(g.uints)
	g.uints = append(g.uints, initial)
	return index
}

// getUint reads a uint64 global variable.
//
// Takes index (int) which is the variable to read.
//
// Returns the current value of the variable.
//
// Safe for concurrent use by multiple goroutines.
func (g *globalStore) getUint(index int) uint64 {
	if !g.shared.Load() {
		return g.uints[index]
	}
	g.mu.RLock()
	v := g.uints[index]
	g.mu.RUnlock()
	return v
}

// setUint writes a uint64 global variable.
//
// Takes index (int) which is the variable to write.
// Takes v (uint64) which is the new value.
//
// Safe for concurrent use by multiple goroutines.
func (g *globalStore) setUint(index int, v uint64) {
	if !g.shared.Load() {
		g.uints[index] = v
		return
	}
	g.mu.Lock()
	g.uints[index] = v
	g.mu.Unlock()
}

// allocComplex allocates a new complex128 global variable and returns its index.
//
// Takes initial (complex128) which is the starting value for the variable.
//
// Returns the index of the newly allocated variable.
//
// Safe for concurrent use by multiple goroutines.
func (g *globalStore) allocComplex(initial complex128) int {
	g.mu.Lock()
	defer g.mu.Unlock()
	index := len(g.complexes)
	g.complexes = append(g.complexes, initial)
	return index
}

// getComplex reads a complex128 global variable.
//
// Takes index (int) which is the variable to read.
//
// Returns the current value of the variable.
//
// Safe for concurrent use by multiple goroutines.
func (g *globalStore) getComplex(index int) complex128 {
	if !g.shared.Load() {
		return g.complexes[index]
	}
	g.mu.RLock()
	v := g.complexes[index]
	g.mu.RUnlock()
	return v
}

// setComplex writes a complex128 global variable.
//
// Takes index (int) which is the variable to write.
// Takes v (complex128) which is the new value.
//
// Safe for concurrent use by multiple goroutines.
func (g *globalStore) setComplex(index int, v complex128) {
	if !g.shared.Load() {
		g.complexes[index] = v
		return
	}
	g.mu.Lock()
	g.complexes[index] = v
	g.mu.Unlock()
}

// registerExternalMethod publishes a method entry for cross-package use.
//
// Publishes a single (typeName, methodName) -> (rootFunction, methodIndex) entry so
// adapters in OTHER CompileProgram batches can dispatch to the method body. Called from
// Service.bridgePackageExports for each entry in the just-compiled package's methodTable.
// Re-registration overrides the previous owner, matching Go's last-write-wins semantics
// if two packages declare a same-named type with the same method (rare outside test
// fixtures).
//
// Takes key (string) which is "TypeName.MethodName".
// Takes rootFunction (*CompiledFunction) which owns the method body.
// Takes methodIndex (uint16) which is the function's slot in rootFunction.functions.
//
// Concurrency: takes g.externalMethodsMu in write mode.
func (g *globalStore) registerExternalMethod(key string, rootFunction *CompiledFunction, methodIndex uint16) {
	if g == nil || rootFunction == nil || key == "" {
		return
	}
	g.externalMethodsMu.Lock()
	defer g.externalMethodsMu.Unlock()
	if g.externalMethods == nil {
		g.externalMethods = make(map[string]externalMethodEntry, externalMethodsInitialCapacity)
	}
	g.externalMethods[key] = externalMethodEntry{
		rootFunction: rootFunction,
		methodIndex:  methodIndex,
	}
}

// lookupExternalMethod returns the cross-package entry matching key.
//
// Called by adapter builders after a local vm.rootFunction.methodTable miss.
//
// Takes key (string) which is "TypeName.MethodName".
//
// Returns externalMethodEntry which is the registered entry on hit.
// Returns bool which reports whether a matching entry was found.
//
// Concurrency: takes g.externalMethodsMu in read mode.
func (g *globalStore) lookupExternalMethod(key string) (externalMethodEntry, bool) {
	if g == nil {
		return externalMethodEntry{}, false
	}
	g.externalMethodsMu.RLock()
	defer g.externalMethodsMu.RUnlock()
	entry, ok := g.externalMethods[key]
	return entry, ok
}

// registerUserNamedInterface records the pikoType identity for a named interface declared
// in user code.
//
// Keyed by qualifiedName ("pkg.Iface"). Called from compiler_types.convertNamedType for
// every named type whose underlying is *types.Interface and which is NOT in
// wellKnownNamedInterfaceRegistry (those have canonical native reflect.Types and do not
// need the side-channel). Re-registration overrides; last-write-wins matches the
// externalMethods bridge.
//
// Takes qualifiedName (string) which is "pkg.Iface".
// Takes piko (*pikoType) which is the populated pikoType identity.
//
// Concurrency: takes g.externalMethodsMu in write mode.
func (g *globalStore) registerUserNamedInterface(qualifiedName string, piko *pikoType) {
	if g == nil || qualifiedName == "" || piko == nil {
		return
	}
	g.externalMethodsMu.Lock()
	defer g.externalMethodsMu.Unlock()
	if g.userNamedInterfaces == nil {
		g.userNamedInterfaces = make(map[string]*pikoType, externalMethodsInitialCapacity)
	}
	g.userNamedInterfaces[qualifiedName] = piko
}

// lookupUserNamedInterface returns the pikoType identity for a name.
//
// Called by the reflect.TypeOf intercept to decide whether to wrap a *interface{} result.
//
// Takes qualifiedName (string) which is "pkg.Iface" or just "Iface".
//
// Returns *pikoType which is the registered identity on hit.
// Returns bool which reports whether an entry was found.
//
// Concurrency: takes g.externalMethodsMu in read mode.
func (g *globalStore) lookupUserNamedInterface(qualifiedName string) (*pikoType, bool) {
	if g == nil || qualifiedName == "" {
		return nil, false
	}
	g.externalMethodsMu.RLock()
	defer g.externalMethodsMu.RUnlock()
	piko, ok := g.userNamedInterfaces[qualifiedName]
	return piko, ok
}

// snapshotVarAs reads a global slot and wraps it in a reflect.Value.
//
// Used by the cross-package var-bridge finaliser to expose an interpreted package's
// exported vars to other compiled packages.
//
// The numeric banks (int, uint, float, complex) store the raw primitive promoted to the
// bank's widest representation (for example all signed ints live in g.ints as int64). The
// caller-supplied reflectType is used to convert that primitive back to the originally
// declared Go type so a downstream `var X MyEnum = uuid.X` keeps the right concrete type
// instead of leaking int64. For the general bank the stored value already carries its
// full type and is returned untouched.
//
// When the slot is out of range for its bank, the bridge skips the var rather than
// panicking, on the principle that a Service in an inconsistent state should not take the
// host process down.
//
// Takes slot (globalVariableInfo) which addresses the value.
// Takes targetType (reflect.Type) which is the var's declared Go type.
//
// Returns reflect.Value which is the wrapped value on success.
// Returns bool which reports whether the snapshot succeeded.
func (g *globalStore) snapshotVarAs(slot globalVariableInfo, targetType reflect.Type) (reflect.Value, bool) {
	if targetType == nil {
		return reflect.Value{}, false
	}
	if !g.indexInBank(slot) {
		return reflect.Value{}, false
	}
	switch slot.kind {
	case registerInt:
		return reflect.ValueOf(g.getInt(slot.index)).Convert(targetType), true
	case registerFloat:
		return reflect.ValueOf(g.getFloat(slot.index)).Convert(targetType), true
	case registerString:
		return reflect.ValueOf(g.getString(slot.index)).Convert(targetType), true
	case registerGeneral:
		return snapshotGeneralSlotAs(g.getGeneral(slot.index), targetType), true
	case registerBool:
		return reflect.ValueOf(g.getBool(slot.index)), true
	case registerUint:
		return reflect.ValueOf(g.getUint(slot.index)).Convert(targetType), true
	case registerComplex:
		return reflect.ValueOf(g.getComplex(slot.index)).Convert(targetType), true
	default:
	}
	return reflect.Value{}, false
}

// indexInBank reports whether slot.index is in range for slot.kind.
//
// Read-only; callers that need the value follow up with the bank-specific getter so the
// existing goroutine-safety fast path is reused.
//
// Takes slot (globalVariableInfo) which addresses the bank and index.
//
// Returns bool which reports whether the index is in range.
//
// Concurrency: takes g.mu in read mode while inspecting bank lengths.
func (g *globalStore) indexInBank(slot globalVariableInfo) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	switch slot.kind {
	case registerInt:
		return slot.index >= 0 && slot.index < len(g.ints)
	case registerFloat:
		return slot.index >= 0 && slot.index < len(g.floats)
	case registerString:
		return slot.index >= 0 && slot.index < len(g.strings)
	case registerGeneral:
		return slot.index >= 0 && slot.index < len(g.general)
	case registerBool:
		return slot.index >= 0 && slot.index < len(g.bools)
	case registerUint:
		return slot.index >= 0 && slot.index < len(g.uints)
	case registerComplex:
		return slot.index >= 0 && slot.index < len(g.complexes)
	default:
	}
	return false
}

// internMapKey returns a stable heap-cloned copy of s.
//
// Reuses the cached copy when the same content has been observed before. Hot path (cache
// hit) reads the sync.Map's internal read-only map via atomic.LoadPointer with no mutex
// and no contention. The miss path allocates strings.Clone once and LoadOrStores; if
// another goroutine inserted concurrently with the same content, the already-cached copy
// wins.
//
// Takes s (string) which is the candidate map-key content. Caller should only invoke this
// for strings whose backing storage is in the arena (transient) and therefore needs
// cloning before being retained as a map key.
//
// Returns string which is the canonical cloned copy. First observation of any content
// value allocates strings.Clone plus a sync.Map insert; subsequent observations return
// the cached copy with no allocation beyond the map lookup itself.
func (g *globalStore) internMapKey(s string) string {
	if cached, ok := g.internedMapKeys.Load(s); ok {
		if cachedString, isString := cached.(string); isString {
			return cachedString
		}
	}
	cloned := strings.Clone(s)
	actual, _ := g.internedMapKeys.LoadOrStore(cloned, cloned)
	if actualString, isString := actual.(string); isString {
		return actualString
	}
	return cloned
}

// materialiseStrings replaces any arena-backed string globals with heap-backed copies so
// that globals do not hold dangling pointers.
//
// Takes arena (*RegisterArena) which is the arena whose byte slabs are checked for
// ownership.
//
// Safe for concurrent use by multiple goroutines.
func (g *globalStore) materialiseStrings(arena *RegisterArena) {
	g.mu.Lock()
	defer g.mu.Unlock()
	for i, s := range g.strings {
		if arena.ownsString(s) {
			g.strings[i] = strings.Clone(s)
		}
	}
}

// reset clears all global variables and returns the store to single-threaded fast-path
// mode. Callers must guarantee no concurrent users remain when reset is invoked.
//
// Safe for concurrent use by multiple goroutines.
func (g *globalStore) reset() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.ints = g.ints[:0]
	g.floats = g.floats[:0]
	g.strings = g.strings[:0]
	g.general = g.general[:0]
	g.bools = g.bools[:0]
	g.uints = g.uints[:0]
	g.complexes = g.complexes[:0]
	g.internedMapKeys.Clear()
	g.shared.Store(false)
	g.goroutinePanic.Store(nil)
	g.goroutinePanicSignal = make(chan struct{})
}

// recordGoroutinePanic stores the first panic captured from a spawned goroutine and wakes
// any sibling parked on a blocking channel or select operation. The compare-and-swap
// guarantees a single winner, so goroutinePanicSignal is closed exactly once.
//
// Takes value (any) which is the panic value as it was thrown.
// Takes stack (string) which is the captured stack trace, or empty when unavailable.
func (g *globalStore) recordGoroutinePanic(value any, stack string) {
	if g.goroutinePanic.CompareAndSwap(nil, &goroutinePanicInfo{value: value, stack: stack}) {
		close(g.goroutinePanicSignal)
	}
}

// goroutinePanicWakeChan returns the channel a blocking operation should select on to
// observe a sibling goroutine's panic, or nil when the store has never launched a
// goroutine (so no panic can occur and the caller keeps its lock-free fast path).
//
// Returns the panic-signal channel when the store is shared, nil otherwise.
func (g *globalStore) goroutinePanicWakeChan() <-chan struct{} {
	if !g.shared.Load() {
		return nil
	}
	return g.goroutinePanicSignal
}

// snapshotGeneralSlotAs coerces a general-bank value into targetType.
//
// Allocated-but-never-written slots return a typed zero so downstream reflect operations
// have a valid Value to work with.
//
// Takes value (reflect.Value) which is the stored general-bank value.
// Takes targetType (reflect.Type) which is the destination Go type.
//
// Returns reflect.Value which is the coerced value.
func snapshotGeneralSlotAs(value reflect.Value, targetType reflect.Type) reflect.Value {
	if !value.IsValid() {
		return reflect.Zero(targetType)
	}
	if value.Type() != targetType && value.Type().ConvertibleTo(targetType) {
		return value.Convert(targetType)
	}
	return value
}

// newGlobalStore creates an empty global variable store.
//
// Returns a newly allocated globalStore with no variables.
func newGlobalStore() *globalStore {
	return &globalStore{goroutinePanicSignal: make(chan struct{})}
}
