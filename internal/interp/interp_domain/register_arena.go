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
	"runtime"
	"strings"
	"unsafe"

	"piko.sh/piko/wdk/stablepool"
)

const (
	// initialIntSlabs is the starting capacity for the int64 slab. Typical: 16 registers x
	// 32 call depth = 512.
	initialIntSlabs = 512

	// initialFloatSlabs is the starting capacity for the float64 slab.
	initialFloatSlabs = 128

	// initialStringSlabs is the starting capacity for the string slab. Kept small because
	// strings are GC-visible; sizeArenaFromFunctions grows the slab to match the compiled
	// function's actual needs.
	initialStringSlabs = 32

	// initialGeneralSlabs is the starting capacity for the reflect.Value slab. Kept small
	// because reflect.Value is pointer-rich and scanned on every GC cycle;
	// sizeArenaFromFunctions grows as needed.
	initialGeneralSlabs = 32

	// initialBoolSlabs is the starting capacity for the bool slab.
	initialBoolSlabs = 128

	// initialUintSlabs is the starting capacity for the uint64 slab.
	initialUintSlabs = 128

	// initialComplexSlabs is the starting capacity for the complex128 slab.
	initialComplexSlabs = 64

	// initialSlicesIntSlabs is the starting capacity for the typed []int64 slice-header
	// slab. Each slot is a 24-byte slice header; kept conservative because typed slice usage
	// scales with code that explicitly chooses int-kind slice element types.
	initialSlicesIntSlabs = 32

	// initialSlicesFloatSlabs is the starting capacity for the typed []float64 slice-header
	// slab.
	initialSlicesFloatSlabs = 32

	// initialSlicesStringSlabs is the starting capacity for the typed []string slice-header
	// slab.
	initialSlicesStringSlabs = 32

	// initialSlicesBoolSlabs is the starting capacity for the typed []bool slice-header
	// slab.
	initialSlicesBoolSlabs = 32

	// initialSlicesUintSlabs is the starting capacity for the typed []uint64 slice-header
	// slab.
	initialSlicesUintSlabs = 32

	// initialSlicesByteSlabs is the starting capacity for the typed []byte slice-header
	// slab. Byte iteration is the hot path for parsers / word-count / brainfuck workloads,
	// so the same modest default as the other typed-slice slabs.
	initialSlicesByteSlabs = 32

	// initialUpvalueCellSlabs is the starting capacity for the upvalueCell slab. Typical:
	// 1-3 upvalues per closure x ~32 call depth.
	initialUpvalueCellSlabs = 64

	// initialUpvalueRefSlabs is the starting capacity for the upvalue ref slab.
	initialUpvalueRefSlabs = 64

	// initialFrameSlabs is the cold-pool starting frame capacity.
	//
	// The compile-time call-graph analysis in sizeArenaFromFunctions raises the slab to the
	// per-program maximum static call depth, so the value here only governs the pre-warmed
	// pool slot footprint for tiny programs and is kept small to avoid wasting GC scan
	// cycles on unused frames.
	initialFrameSlabs = 64

	// initialByteSlabSize is the starting capacity for the byte slab used to intern string
	// character data. Strings created during VM execution have their bytes bump-allocated
	// here instead of on the Go heap, eliminating per-concat GC pressure.
	//
	// 16 KiB is small enough that one-shot programs (e.g. piko evaluating a single
	// expression) don't waste memory, large enough that typical short-running programs
	// (formatting a few dozen numbers, building a config string) never trigger a grow.
	// Programs that exceed this get exact pre-sizing via sizeArenaFromFunctions' bytecode
	// walk + EnsureCapacity, so the warmup-time grow happens at most once.
	initialByteSlabSize = 16384

	// initialIntBackingSize is the starting capacity for the int64 backing slab used by
	// arena-routed make([]int64, ...) and append-grow on []int64. Smaller than
	// initialIntSlabs because slice backings tend to be one-per-make rather than one-per-
	// register-bank; EnsureCapacity grows it post-compilation if the program's static
	// analysis demands more.
	initialIntBackingSize = 256

	// initialFloatBackingSize is the starting capacity for the float64 backing slab used by
	// arena-routed make([]float64, ...) and append-grow on []float64.
	initialFloatBackingSize = 256

	// initialStringBackingSize is the starting capacity for the string backing slab used by
	// arena-routed make([]string, ...) and append-grow on []string. Kept smaller than the
	// int variant because []string backings hold GC-visible pointers and large
	// over-allocations waste GC scan work.
	initialStringBackingSize = 64

	// initialBoolBackingSize is the starting capacity for the bool backing slab used by
	// arena-routed make([]bool, ...) and append-grow on []bool.
	initialBoolBackingSize = 256

	// initialUintBackingSize is the starting capacity for the uint64 backing slab used by
	// arena-routed make([]uint64, ...) and append-grow on []uint64.
	initialUintBackingSize = 256

	// maxArenaMultiplier is the DoS protection threshold that caps slab growth. If a slab
	// grows beyond initialSize * maxArenaMultiplier, it is shrunk on reset, preventing
	// memory bloat from pathological inputs while allowing legitimate growth to be retained
	// across requests.
	maxArenaMultiplier = 8

	// backingSlabIdleResetsBeforeShrink caps idle-reset hysteresis.
	//
	// The number of consecutive Reset() calls with low slab utilisation that must elapse
	// before an overgrown backing slab is shrunk back to initial size. Hysteresis here
	// avoids the doom-loop where a workload consistently needs a large slab but pays a fresh
	// grow chain every Reset; the slab is only reclaimed once the workload has demonstrably
	// stopped needing it across several iterations.
	backingSlabIdleResetsBeforeShrink = 4

	// backingSlabLowUseRatioDenominator gates the idle-reset counter.
	//
	// If the previous Reset's peak index was below
	// len(slab)/backingSlabLowUseRatioDenominator, the iteration counts as "idle" for the
	// slab. Setting the denominator to 4 means a slab counts as idle only when less than a
	// quarter of its capacity was touched, so a workload genuinely using the slab keeps it
	// warm.
	backingSlabLowUseRatioDenominator = 4

	// stringBoxBytes is the byte size of one allocated string box slot (a Go string header:
	// pointer plus length).
	stringBoxBytes int64 = 16

	// int64BoxBytes is the byte size of one allocated int64 box slot.
	int64BoxBytes int64 = 8

	// float64BoxBytes is the byte size of one allocated float64 box slot.
	float64BoxBytes int64 = 8

	// uint64BoxBytes is the byte size of one allocated uint64 box slot.
	uint64BoxBytes int64 = 8

	// complex128BoxBytes is the byte size of one allocated complex128 box slot (two
	// float64s).
	complex128BoxBytes int64 = 16

	// arenaSliceHeaderBoxBytes is the byte size of one arenaSliceHeader slot in the
	// slice-header slab.
	arenaSliceHeaderBoxBytes int64 = 24

	// initialBoxSlabCapacity is the starting capacity for the per-type box slabs (string,
	// int64, float64, uint64). Sized so a typical closure-heavy program fills the slab once
	// and never grows.
	initialBoxSlabCapacity = 64

	// initialGenericBytesCapacity is the starting capacity in bytes for the generic byte
	// slab used by arena-backed reflect.Value struct and array storage.
	initialGenericBytesCapacity = 4096

	// initialSliceHeaderCapacity is the slice-header slab capacity.
	//
	// Sized so a cold pool arena skips the smallest doublings of the grow chain when first
	// used by a slice-header-heavy workload (byte-builder appends produce one header per
	// call). 2048 slots cost ~48 KiB per arena which is small enough for the NumCPU*2
	// pre-warmed pool but large enough to absorb typical short- program needs without any
	// grow.
	initialSliceHeaderCapacity = 2048

	// allocMaskInt flags that the int64 register bank is non-empty.
	allocMaskInt uint16 = 1 << 0

	// allocMaskFloat flags that the float64 register bank is non-empty.
	allocMaskFloat uint16 = 1 << 1

	// allocMaskString flags that the string register bank is non-empty.
	allocMaskString uint16 = 1 << 2

	// allocMaskGeneral flags that the reflect.Value register bank is non-empty.
	allocMaskGeneral uint16 = 1 << 3

	// allocMaskBool flags that the bool register bank is non-empty.
	allocMaskBool uint16 = 1 << 4

	// allocMaskUint flags that the uint64 register bank is non-empty.
	allocMaskUint uint16 = 1 << 5

	// allocMaskComplex flags that the complex128 register bank is non-empty.
	allocMaskComplex uint16 = 1 << 6

	// allocMaskSliceInt flags that the []int64 slice-header bank is non-empty.
	allocMaskSliceInt uint16 = 1 << 7

	// allocMaskSliceFloat flags that the []float64 slice-header bank is non-empty.
	allocMaskSliceFloat uint16 = 1 << 8

	// allocMaskSliceString flags that the []string slice-header bank is non-empty.
	allocMaskSliceString uint16 = 1 << 9

	// allocMaskSliceBool flags that the []bool slice-header bank is non-empty.
	allocMaskSliceBool uint16 = 1 << 10

	// allocMaskSliceUint flags that the []uint64 slice-header bank is non-empty.
	allocMaskSliceUint uint16 = 1 << 11

	// allocMaskSliceByte flags that the []byte slice-header bank is non-empty.
	allocMaskSliceByte uint16 = 1 << 12

	// typedSliceBankMask flags banks needing base-pointer repair.
	//
	// The set of register banks whose ctx base pointer repairRegisterBasesFromCallers may
	// need to restore on a cross-mask return. It is exactly the banks the inline-return ASM
	// does not unconditionally refresh (the typed-slice headers plus complex). When no
	// compiled function in the program uses any of these banks, the repair walk is a
	// provable no-op and is skipped.
	typedSliceBankMask uint16 = allocMaskSliceInt | allocMaskSliceFloat | allocMaskSliceString |
		allocMaskSliceBool | allocMaskSliceUint | allocMaskSliceByte | allocMaskComplex

	// constMaskInt flags that the int64 constant pool is non-empty.
	constMaskInt uint16 = 1 << 0

	// constMaskFloat flags that the float64 constant pool is non-empty.
	constMaskFloat uint16 = 1 << 1

	// constMaskString flags that the string constant pool is non-empty.
	constMaskString uint16 = 1 << 2

	// constMaskBool flags that the bool constant pool is non-empty.
	constMaskBool uint16 = 1 << 3

	// constMaskStructLayoutTable flags that the struct-layout table is non-empty.
	constMaskStructLayoutTable uint16 = 1 << 4

	// constMaskTypeTable flags that the type table is non-empty.
	constMaskTypeTable uint16 = 1 << 5

	// constMaskComplex flags that the complex128 constant pool is non-empty.
	constMaskComplex uint16 = 1 << 6

	// constMaskUint flags that the uint64 constant pool is non-empty.
	constMaskUint uint16 = 1 << 7

	// defaultMaxArenaBytesSmallHostThreshold gates the small-host path.
	//
	// The candidate value below which the host-aware resolver switches from "quarter of RAM"
	// to the more generous "three quarters of RAM" policy. Hosts whose 1/4 share already
	// exceeds this figure are treated as comfortable and only get the conservative slice;
	// small hosts (where 1/4 would leave the script with too little headroom) get a much
	// larger share so a Piko Execute is not artificially starved on, say, a 4 GiB box.
	defaultMaxArenaBytesSmallHostThreshold uint64 = 2 << 30

	// defaultMaxArenaBytesCeiling is the highest budget the host-aware resolver may select.
	// Acts as a hard upper bound so a 1 TiB server still keeps a single Execute well clear
	// of host OOM territory.
	defaultMaxArenaBytesCeiling uint64 = 16 << 30

	// defaultMaxArenaBytesNormalShift is the conservative-path shift.
	//
	// The right-shift applied to detected host memory on the conservative path (i.e.
	// dividing by four). A Piko Execute may use at most a quarter of host RAM by default,
	// leaving headroom for the OS, the Go runtime, and other workloads sharing the box.
	defaultMaxArenaBytesNormalShift = 2

	// defaultMaxArenaBytesGenerousNumerator and ...Denominator together express the
	// small-host fallback fraction: 3/4 of detected host memory. A 4 GiB box returns 3 GiB
	// rather than the 1 GiB the normal 1/4 share would yield.
	defaultMaxArenaBytesGenerousNumerator uint64 = 3

	// defaultMaxArenaBytesGenerousDenominator denominator for the 3/4 small-host fallback
	// fraction.
	defaultMaxArenaBytesGenerousDenominator uint64 = 4

	// defaultMaxArenaBytesUndetectedFallback is the budget chosen when the host's total RAM
	// cannot be probed (non-Linux platforms today). A middle-of-the-road figure: large
	// enough for ordinary scripts, small enough not to wipe out a typical workstation.
	defaultMaxArenaBytesUndetectedFallback uint64 = 2 << 30

	// maxRetainedOldSlabsPerType caps retained old-slab references.
	//
	// The number of retired old slab references the arena keeps for any single typed-backing
	// list (oldByteSlabs, oldIntBackings, oldStringBackings, oldGenericByteSlabs,
	// oldSliceHeaderSlabs, ...). When a grow helper would push a list past this threshold,
	// the oldest retained entry is dropped so the Go GC can reclaim its backing memory once
	// no outstanding user-visible value still references it.
	maxRetainedOldSlabsPerType = 16
)

var (
	// defaultMaxArenaBytes is the resolved per-Execute budget for arena slab growth (the sum
	// of every retained slab's byte capacity at any moment). Computed once at init from the
	// host's detected RAM via the small-host-aware policy implemented by
	// resolveDefaultMaxArenaBytes.
	defaultMaxArenaBytes = resolveDefaultMaxArenaBytes()

	// registerArenaPool is the single pool for arena instances. Backed by stablepool so the
	// arena's internal slabs (byte, slice-header, generic bytes, backing arrays, etc.)
	// survive every Go GC cycle, preserving slab warmth across collection boundaries.
	//
	// Pre-warm sized as NumCPU()*2: stablepool uses nextPow2(NumCPU) shards, so capacity =
	// NumCPU*2 puts at least one warm arena per shard with a little headroom. Real
	// concurrent arena demand is bounded by the host process's parallel piko Eval() / opGo
	// goroutine count, which is itself capped by GOMAXPROCS in practice. Larger pre-warms
	// commit memory the typical workload never uses; smaller pre-warms leave most shards
	// empty and force a steal walk on every Get.
	//
	// Growth ceiling NumCPU()*32 absorbs deep-recursion or pathological fanout (a host
	// running hundreds of concurrent piko scripts) without unbounded memory. MustGet falls
	// through to a fresh allocation if growth is exhausted, so the cap is safety, not a hard
	// limit.
	//
	// Cleaner is nil. PutRegisterArena calls a.Reset() explicitly so the arena is fully
	// drained before returning to the pool.
	registerArenaPool, _ = stablepool.New[RegisterArena](
		initRegisterArenaInPlace,
		nil,
		runtime.NumCPU()*2,
		stablepool.WithGrowth[RegisterArena](runtime.NumCPU()*32),
	)

	// zeroSizeAllocBacking is a single shared byte used as the storage for zero-size types
	// (empty structs / arrays). Mirrors Go's runtime.zerobase pattern: all zero-size
	// allocations alias the same pointer because the type has no observable bytes.
	//
	//nolint:gochecknoglobals // singleton, written once at init
	zeroSizeAllocBacking byte

	// zeroSizeAllocPtr is the pointer returned by AllocBytes for zero-size types. Stable for
	// the process lifetime.
	//
	//nolint:gochecknoglobals // singleton, written once at init
	zeroSizeAllocPtr = unsafe.Pointer(&zeroSizeAllocBacking)
)

// arenaIndices is the contiguous block of arena allocation indices saved and restored per
// frame.
//
// Pulling these into one struct lets RegisterArena and ArenaSavePoint both embed the
// block and use a single struct assignment (a memmove of the whole 120-byte block)
// instead of fifteen sequential dependent stores when snapshotting or restoring the arena
// state.
//
// Important: field order is load-bearing. The byte offsets must match the ASM hard-coded
// CF_ARENA_SAVE+{0,8,16,24,32,40,48,96} references in asm_dispatch_offsets.h. Tested by
// TestCallFrameOffsets in vm_dispatch_offsets_test.go; any reorder will be caught at
// build time.
type arenaIndices struct {
	// intIndex stores the allocation index for the int64 slab.
	intIndex int

	// floatIndex stores the allocation index for the float64 slab.
	floatIndex int

	// stringIndex stores the allocation index for the string slab.
	stringIndex int

	// generalIndex stores the allocation index for the reflect.Value slab.
	generalIndex int

	// boolIndex stores the allocation index for the bool slab.
	boolIndex int

	// uintIndex stores the allocation index for the uint64 slab.
	uintIndex int

	// complexIndex stores the allocation index for the complex128 slab.
	complexIndex int

	// slicesIntIndex stores the allocation index for the typed []int64 slice-header slab.
	slicesIntIndex int

	// slicesFloatIndex stores the allocation index for the typed []float64 slice-header
	// slab.
	slicesFloatIndex int

	// slicesStringIndex stores the allocation index for the typed []string slice-header
	// slab.
	slicesStringIndex int

	// slicesBoolIndex stores the allocation index for the typed []bool slice-header slab.
	slicesBoolIndex int

	// slicesUintIndex stores the allocation index for the typed []uint64 slice-header slab.
	slicesUintIndex int

	// slicesByteIndex stores the allocation index for the typed []byte slice-header slab.
	slicesByteIndex int

	// upvalueCellIndex stores the allocation index for the upvalueCell slab.
	upvalueCellIndex int

	// upvalueReferenceIndex stores the allocation index for the upvalue reference slab.
	upvalueReferenceIndex int
}

// ArenaSavePoint records the arena allocation indices at a point in time, enabling call
// frames to restore the arena when popped. This turns the bump allocator into a stack
// allocator: a recursive function only needs max-depth x registers-per-frame arena slots,
// not total-calls x registers-per-frame.
//
// The embedded arenaIndices block keeps every per-frame restorable index contiguous so
// SaveInto and Restore can use a single struct assignment (memmove) instead of fifteen
// sequential stores. Selectors like sp.intIndex continue to work via Go's field
// promotion.
type ArenaSavePoint struct {
	arenaIndices
}

// RegisterArena provides arena-based allocation for register banks. Instead of calling
// make() per call frame (one sync.Pool.Get per function call), the arena pre-allocates
// contiguous slabs and hands out sub-slices via simple index bumping - zero sync overhead
// in the hot path.
//
// Frames record a save point before allocating. When a frame is popped, the save point is
// restored, reclaiming the arena space. This makes the arena a stack allocator - ideal
// for a call stack where allocation and deallocation follow LIFO order.
//
// The arena also holds VM-parallel arrays (callInfoBasesSlab, dispatchSavesSlab) that
// shadow the call stack. These are pooled alongside the frame slab to avoid per-Execute()
// allocations.
//
// One arena is obtained per Eval() call (or per goroutine for opGo). The arena itself is
// pooled via wdk/stablepool, which keeps the arena's internal slabs warm across Go GC
// cycles. The Link field (first) is the intrusive header stablepool requires.
type RegisterArena struct {
	// Link is stablepool's intrusive free-list header. MUST be the first field. The pool
	// writes to it when the arena is on the free stack; user code must not touch it.
	stablepool.Link

	// ownerVM is the VM that currently owns this arena.
	//
	// Populated by the execute boundary so chargeArenaAllocation can drive a MinorGC
	// fallback before reporting a budget exhaustion. Nil when the arena is sitting in the
	// pool or has been released by its owner.
	ownerVM *VM

	// stringBoxSlab backs boxed string-to-general reflect.Values.
	//
	// Replaces the per-call heap allocation that reflect.ValueOf(s) does internally
	// (convTstring -> mallocgc, ~8% of markov CPU). Each AllocStringBox call bump-allocates
	// one entry; the returned reflect.Value's internal ptr field points into this slab. The
	// slab itself is a real []string so writes go through Go's normal write barriers and the
	// slab stays GC-reachable while reflect.Values hold pointers into it.
	stringBoxSlab []string

	// complexBoxSlab backs boxed complex128-to-general reflect.Values.
	//
	// Sibling of intBoxSlab / floatBoxSlab. complex128 has no runtime small-value cache, so
	// every reflect.ValueOf(complex128) without arena routing allocates 16 B on the heap.
	// Bump-allocating from the slab replaces that per-call mallocgc with a single index
	// bump, and the resulting reflect.Value points at the slab slot.
	complexBoxSlab []complex128

	// stringSlab holds the contiguous string register memory.
	stringSlab []string

	// generalSlab holds the contiguous reflect.Value register memory.
	generalSlab []reflect.Value

	// boolSlab holds the contiguous bool register memory.
	boolSlab []bool

	// uintSlab holds the contiguous uint64 register memory.
	uintSlab []uint64

	// complexSlab holds the contiguous complex128 register memory.
	complexSlab []complex128

	// slicesIntSlab holds the contiguous []int64 slice-header storage for the typed
	// slicesInt register bank.
	slicesIntSlab [][]int64

	// slicesFloatSlab holds the contiguous []float64 slice-header storage for the typed
	// slicesFloat register bank.
	slicesFloatSlab [][]float64

	// slicesStringSlab holds the contiguous []string slice-header storage for the typed
	// slicesString register bank.
	slicesStringSlab [][]string

	// slicesBoolSlab holds the contiguous []bool slice-header storage for the typed
	// slicesBool register bank.
	slicesBoolSlab [][]bool

	// slicesUintSlab holds the contiguous []uint64 slice-header storage for the typed
	// slicesUint register bank.
	slicesUintSlab [][]uint64

	// slicesByteSlab holds the contiguous []byte slice-header storage for the typed
	// slicesByte register bank.
	slicesByteSlab [][]byte

	// frameSlab holds the pre-allocated call frame storage.
	frameSlab []callFrame

	// callInfoBasesSlab holds ASM call-info base pointers parallel to frameSlab.
	callInfoBasesSlab []uintptr

	// floatBoxSlab backs boxed float64-to-general reflect.Values.
	//
	// Sibling of intBoxSlab; one 8-byte slot per AllocFloatBox call replaces
	// reflect.ValueOf's convT32 heap allocation.
	floatBoxSlab []float64

	// byteSlab holds bump-allocated byte storage for interned strings.
	byteSlab []byte

	// oldByteSlabs retains previous byte slabs so existing strings remain valid.
	oldByteSlabs [][]byte

	// intBackingSlab holds bump-allocated int64 element storage used by arena-routed
	// make([]int64, ...) and append-grow handlers. Slices carved out of this slab use the
	// three-index form slab[idx:idx+n:idx+n] so user append past cap triggers our
	// arenaAppendInt helper rather than silently spilling into the next allocation's region.
	intBackingSlab []int64

	// oldIntBackings retains previous intBackingSlab arrays so user slices still pointing
	// into them remain valid after a grow.
	oldIntBackings [][]int64

	// floatBackingSlab holds bump-allocated float64 element storage for arena-routed
	// make([]float64, ...) / append-grow.
	floatBackingSlab []float64

	// oldFloatBackings retains previous floatBackingSlab arrays.
	oldFloatBackings [][]float64

	// stringBackingSlab holds bump-allocated string element storage for arena-routed
	// make([]string, ...) / append-grow. Writes into carved-out sub-slices go through Go's
	// normal write barrier because the slab is a real []string, so the typed slice can
	// safely hold pointers into the arena's byte slab or the Go heap.
	stringBackingSlab []string

	// oldStringBackings retains previous stringBackingSlab arrays.
	oldStringBackings [][]string

	// boolBackingSlab holds bump-allocated bool element storage for arena-routed
	// make([]bool, ...) / append-grow.
	boolBackingSlab []bool

	// oldBoolBackings retains previous boolBackingSlab arrays.
	oldBoolBackings [][]bool

	// uintBackingSlab holds bump-allocated uint64 element storage for arena-routed
	// make([]uint64, ...) / append-grow.
	uintBackingSlab []uint64

	// oldUintBackings retains previous uintBackingSlab arrays.
	oldUintBackings [][]uint64

	// upvalueCellSlab holds the contiguous upvalueCell storage.
	upvalueCellSlab []upvalueCell

	// upvalueReferenceSlab holds the contiguous upvalue reference storage.
	upvalueReferenceSlab []upvalue

	// floatSlab holds the contiguous float64 register memory.
	floatSlab []float64

	// intSlab holds the contiguous int64 register memory.
	intSlab []int64

	// dispatchSavesSlab holds ASM dispatch register saves parallel to frameSlab.
	dispatchSavesSlab []asmDispatchSave

	// uintBoxSlab backs boxed uint64-to-general reflect.Values.
	//
	// Sibling of intBoxSlab; uints in [0,255] hit the runtime cache, values outside that
	// range bump-allocate here.
	uintBoxSlab []uint64

	// intBoxSlab backs boxed int64-to-general reflect.Values.
	//
	// Sibling of stringBoxSlab used by boxInt64ToGeneral to bump- allocate one 8-byte slot
	// per call, replacing reflect.ValueOf's convT64 heap allocation. Most ints fall in the
	// [-128,256] cache so values outside that range (expr_eval arithmetic, dijkstra
	// distances, large counts) hit this slab.
	intBoxSlab []int64

	// genericBytesSlab backs arena copies of pointer-free reflect.Values.
	//
	// 8-byte-aligned byte arena used by copyReflectValue for struct and array kinds with no
	// pointer fields. Routing pointer-free struct copies here removes the per-call
	// reflect.New heap allocation: the slab grows once and is reused. Pointer-containing
	// types fall back to reflect.New, because writing pointers into a []byte hides them from
	// the GC and lets their pointees be collected.
	genericBytesSlab []byte

	// oldGenericByteSlabs retains retired generic-byte slabs.
	//
	// Keeps reflect.Values whose ptr field points into a grown-away slab valid until they
	// themselves go out of scope. growGenericBytesSlab appends here before installing the
	// new backing. Mirrors oldByteSlabs and is required so ownsBytePointer can answer
	// correctly for values whose storage lives in a retired slab.
	oldGenericByteSlabs [][]byte

	// sliceHeaderSlab pools slice-header storage for reflect.Values.
	//
	// Backs the reflect.Value returned by appendGenericFastPath's sliceValue.Slice(0, len+1)
	// call. The slab is GC-aware because each slot's Data field is typed as unsafe.Pointer
	// rather than uintptr, so the GC keeps the backing array of any live slice header
	// reachable. Cuts ~10.7M reflect.Value.Slice heap allocs per expr_eval iteration.
	sliceHeaderSlab []arenaSliceHeader

	// oldSliceHeaderSlabs retains previous slice-header slabs after growth so reflect.Values
	// whose ptr is an arenaSliceHeader in a retired slab remain valid for the lifetime of
	// the arena. Same shape as oldGenericByteSlabs / oldByteSlabs.
	oldSliceHeaderSlabs [][]arenaSliceHeader

	// arenaIndices is the contiguous block of fifteen per-frame save/restore indices
	// (intIndex through upvalueReferenceIndex). Embedding it lets SaveInto and Restore copy
	// the entire block in a single struct assignment instead of fifteen sequential stores,
	// and Go's field promotion keeps existing a.intIndex selectors working unchanged.
	arenaIndices

	// maxArenaBytes is the per-Execute byte budget for arena growth.
	//
	// Zero means use defaultMaxArenaBytes. Configured via WithMaxArenaSizeBytes on the
	// Service and propagated through vmLimits to the arena before first allocation.
	maxArenaBytes uint64

	// sliceHeaderIndex tracks the bump position in sliceHeaderSlab. Not saved per-frame;
	// cleared only on arena Reset.
	sliceHeaderIndex int

	// stringBoxIndex tracks the bump position in stringBoxSlab. Not saved per-frame; cleared
	// only on arena Reset.
	stringBoxIndex int

	// intBoxIndex tracks the bump position in intBoxSlab. Not saved per-frame; cleared only
	// on arena Reset.
	intBoxIndex int

	// floatBoxIndex tracks the bump position in floatBoxSlab. Not saved per-frame; cleared
	// only on arena Reset.
	floatBoxIndex int

	// complexBoxIndex tracks the bump position in complexBoxSlab.
	complexBoxIndex int

	// uintBoxIndex tracks the bump position in uintBoxSlab. Not saved per-frame; cleared
	// only on arena Reset.
	uintBoxIndex int

	// floatBackingIndex tracks the current bump position in floatBackingSlab. Not saved
	// per-frame; cleared only on arena Reset.
	floatBackingIndex int

	// boolBackingIndex tracks the current bump position in boolBackingSlab. Not saved
	// per-frame; cleared only on arena Reset.
	boolBackingIndex int

	// intBackingIndex tracks the current bump position in intBackingSlab. Not saved
	// per-frame; cleared only on arena Reset.
	intBackingIndex int

	// byteIndex tracks the current allocation position in byteSlab. Not saved per-frame;
	// cleared only on arena Reset.
	byteIndex int

	// byteSlabPeakIndexLastReset captures the previous-iteration peak.
	//
	// The byteIndex value captured at the start of the most recent Reset() (i.e. the peak
	// usage of the previous iteration). shrinkOvergrownBackingSlabs consults this to drive
	// the hysteretic shrink policy: a slab whose previous iteration touched most of it is
	// kept; one repeatedly idle across several Resets is dropped back to initial size.
	byteSlabPeakIndexLastReset int

	// genericBytesIndex tracks the bump position in genericBytesSlab. Not saved per-frame;
	// cleared only on arena Reset.
	genericBytesIndex int

	// uintBackingIndex tracks the current bump position in uintBackingSlab. Not saved
	// per-frame; cleared only on arena Reset.
	uintBackingIndex int

	// bytesAllocated tracks the running total of bytes bump-allocated into the arena's data
	// slabs since the last MinorGC (or arena reset). Used by the adaptive GC trigger: when
	// this crosses nextGCAt, the arena signals vm.checkpointFlags |= checkpointFlagGCPending
	// and the dispatch loop runs MinorGC at the next safe point.
	bytesAllocated int64

	// nextGCAt is the byte-allocation threshold that triggers the next MinorGC.
	//
	// Initialised lazily on first allocation that would cross it; updated adaptively after
	// each GC based on observed live ratio. A value of 0 disables GC (used for short-lived
	// arenas that are guaranteed to Reset before reaching threshold).
	nextGCAt int64

	// bytesAtLastGC records the bytesAllocated value immediately after the most recent
	// MinorGC compaction. Used to compute growth-since- last-GC and tune nextGCAt.
	bytesAtLastGC int64

	// framesUsed is the high-water mark of frames touched (pushFrame'd) in the current
	// execution.
	framesUsed int

	// totalAllocatedBytes tracks the arena's current working-set bytes.
	//
	// The current working-set capacity of the arena's typed slabs plus any
	// retained-for-safety old slabs that have not yet been dropped by a MinorGC compaction.
	// Each grow helper updates it via chargeArenaAllocation by the delta between the new and
	// prior slab capacities; MinorGC then deducts the bytes reclaimed by compactPhase. The
	// counter is reset to zero by Reset(). When totalAllocatedBytes would exceed
	// maxArenaBytes, the grow path first attempts a MinorGC to drop dead retained slabs;
	// only if the figure remains above budget does it panic with errArenaBudgetExceeded.
	totalAllocatedBytes uint64

	// stringBackingIndex tracks the current bump position in stringBackingSlab. Not saved
	// per-frame; cleared only on arena Reset.
	stringBackingIndex int

	// byteSlabIdleResetCount is the number of consecutive Reset() calls during which the
	// byteSlab was below the low-use ratio. Reaches backingSlabIdleResetsBeforeShrink to
	// trigger the shrink and is then cleared back to zero.
	byteSlabIdleResetCount uint32

	// gcCount is a diagnostic counter of how many MinorGC cycles this arena has performed in
	// its current Execute. Reset on PutRegisterArena.
	gcCount uint32

	// isLeaf marks arenas that will Reset before observation.
	//
	// Goroutine child arenas are leaf; the main goroutine arena is not, because it lives for
	// the entire Execute call and the boundary materialiseAnyForArena already detaches the
	// return value. Materialisation routines short-circuit when this is false, leaving
	// values pointing into the arena's slabs and avoiding the heap allocations + GC pressure
	// that defeat the arena's whole purpose.
	isLeaf bool
}

// CallInfoBases returns the arena's pre-allocated slab for ASM call-info base pointers,
// parallel to the frame stack.
//
// Returns the callInfoBasesSlab slice.
func (a *RegisterArena) CallInfoBases() []uintptr {
	return a.callInfoBasesSlab
}

// arenaSliceHeader mirrors Go's runtime/internal slice header so the arena slab can back
// reflect.Value slice headers. Data is typed as unsafe.Pointer (not uintptr) so the GC
// keeps the backing array of any in-use header reachable as long as the slab itself is
// reachable.
//
// The AllocXxx helpers that consume this header (and every other allocator on
// RegisterArena) live in register_arena_alloc.go.
type arenaSliceHeader struct {
	// Data is the slice's backing-array pointer; typed as unsafe.Pointer so the GC traces
	// through it.
	Data unsafe.Pointer

	// Len is the slice's logical length.
	Len int

	// Cap is the slice's capacity in elements.
	Cap int
}

// EnsureCapacity pre-sizes the arena slabs so that AllocRegisters never triggers a grow
// during execution.
//
// Called once after compilation with hints derived from the compiled function table.
//
// Takes counts (typedSlabCounts) which carries the minimum required capacity for every
// slab kind. Bundling avoids a long argument list that grows every time a new typed
// register bank is added.
func (a *RegisterArena) EnsureCapacity(counts typedSlabCounts) {
	a.ensureScalarBankCapacity(counts)
	a.ensureSliceBankCapacity(counts)
	a.ensureBackingCapacity(counts)
	a.ensureFrameStackCapacity(counts.frameStack)
	a.ensureBoxSlabCapacity(counts)
	a.ensureUpvalueCapacity(counts)
}

// typedSlabCounts bundles the minimum capacity required for every arena slab kind. Used
// by EnsureCapacity so adding a new typed register bank only changes the struct layout,
// not the function signature.
type typedSlabCounts struct {
	// ints is the int64 register count for one frame.
	ints int

	// floats is the float64 register count for one frame.
	floats int

	// strings is the string register count for one frame.
	strings int

	// generals is the reflect.Value register count for one frame.
	generals int

	// bools is the bool register count for one frame.
	bools int

	// uints is the uint64 register count for one frame.
	uints int

	// complexes is the complex128 register count for one frame.
	complexes int

	// slicesInts is the []int64 slice-header register count.
	slicesInts int

	// slicesFloats is the []float64 slice-header register count.
	slicesFloats int

	// slicesStrings is the []string slice-header register count.
	slicesStrings int

	// slicesBools is the []bool slice-header register count.
	slicesBools int

	// slicesUints is the []uint64 slice-header register count.
	slicesUints int

	// slicesBytes is the []byte slice-header register count.
	slicesBytes int

	// bytes is the byte-slab budget needed for arena-routed string production:
	// BytesToString, string concatenation, Itoa/FormatInt output. Populated by
	// sizeArenaFromFunctions from a bytecode walk over byteSlab-writing opcodes;
	// EnsureCapacity grows the byte slab to at least this many bytes so the dispatch loop
	// never triggers a mallocgc inside growByteSlab in steady state.
	bytes int

	// intBacking is the budget for arena-routed []int64 storage (make([]int64, ...) +
	// append-grow). Populated from opMakeSliceInt counts.
	intBacking int

	// floatBacking is the budget for arena-routed []float64 storage.
	floatBacking int

	// stringBacking is the budget for arena-routed []string storage.
	stringBacking int

	// boolBacking is the budget for arena-routed []bool storage.
	boolBacking int

	// uintBacking is the budget for arena-routed []uint64 storage.
	uintBacking int

	// genericBytes is the byte budget for the snapshot slab.
	//
	// Fed from a bytecode walk of composite-literal / array-snapshot / make-slice opcodes.
	// Without this hint the slab starts at initialGenericBytesCapacity and doubles
	// repeatedly inside the dispatch loop, paying a mallocgc + GC scan window per doubling.
	genericBytes int

	// sliceHeaders is the slot budget for the slice-header slab. Each arena-routed
	// reflect.Value of kind Slice consumes one slot; growing the slab inside dispatch is
	// identical-cost to growing the byte slab.
	sliceHeaders int

	// frameStack is the minimum frame-stack depth the arena must support without growing.
	// Computed from the static call graph via estimateMaxCallDepth; clamped at
	// maxFrameStackPreSize.
	frameStack int

	// intBoxes is the int64 box-slab slot budget. Charged by opPackInterface (registerInt
	// source) and opPackTyped (Int* destinations).
	intBoxes int

	// uintBoxes is the uint64 box-slab slot budget.
	uintBoxes int

	// floatBoxes is the float64 box-slab slot budget.
	floatBoxes int

	// stringBoxes is the string box-slab slot budget.
	stringBoxes int

	// complexBoxes is the complex128 box-slab slot budget.
	complexBoxes int

	// upvalueCells is the upvalueCell slab slot budget. Charged by opMakeClosure occurrences
	// using the callee's upvalueDescriptor count.
	upvalueCells int

	// upvalueReferences is the upvalue reference slab slot budget. Tracks alongside
	// upvalueCells; current ABI has them coincide but the field exists separately so future
	// shape changes do not require a hint-struct churn.
	upvalueReferences int
}

// AllocRegisters returns a registers struct backed by sub-slices of the arena's
// contiguous slabs via O(1) index bumping.
//
// The compiler emits explicit opLoadZero instructions for any variables that require
// zero-initialisation (named returns, uninitialised var declarations), so the arena does
// not zero int/float registers here.
//
// Takes numRegs ([NumRegisterKinds]uint32) which is the number of registers to allocate
// for each register kind.
//
// Returns a registers struct with each bank pointing into the arena slabs.
func (a *RegisterArena) AllocRegisters(numRegs [NumRegisterKinds]uint32) Registers {
	counts := arenaCountsFromNumRegs(numRegs)
	if a.needsGrow(counts) {
		a.growSlabs(counts)
	}

	r := Registers{
		ints:         a.intSlab[a.intIndex : a.intIndex+counts.ints],
		floats:       a.floatSlab[a.floatIndex : a.floatIndex+counts.floats],
		strings:      a.stringSlab[a.stringIndex : a.stringIndex+counts.strings],
		general:      a.generalSlab[a.generalIndex : a.generalIndex+counts.generals],
		bools:        a.boolSlab[a.boolIndex : a.boolIndex+counts.bools],
		uints:        a.uintSlab[a.uintIndex : a.uintIndex+counts.uints],
		complex:      a.complexSlab[a.complexIndex : a.complexIndex+counts.complexes],
		slicesInt:    a.slicesIntSlab[a.slicesIntIndex : a.slicesIntIndex+counts.slicesInts],
		slicesFloat:  a.slicesFloatSlab[a.slicesFloatIndex : a.slicesFloatIndex+counts.slicesFloats],
		slicesString: a.slicesStringSlab[a.slicesStringIndex : a.slicesStringIndex+counts.slicesStrings],
		slicesBool:   a.slicesBoolSlab[a.slicesBoolIndex : a.slicesBoolIndex+counts.slicesBools],
		slicesUint:   a.slicesUintSlab[a.slicesUintIndex : a.slicesUintIndex+counts.slicesUints],
		slicesByte:   a.slicesByteSlab[a.slicesByteIndex : a.slicesByteIndex+counts.slicesBytes],
	}

	a.bumpIndices(counts)
	return r
}

// AllocRegistersInto writes arena-backed sub-slices directly into the target registers
// pointer, avoiding the 168-byte by-value copy that AllocRegisters incurs.
//
// Used in the hot call path where the target is a callFrame.registers field already at
// its final address.
//
// Takes r (*Registers) which is the target registers to populate in place.
// Takes numRegs ([NumRegisterKinds]uint32) which is the number of registers to allocate
// for each register kind.
func (a *RegisterArena) AllocRegistersInto(r *Registers, numRegs [NumRegisterKinds]uint32) {
	counts := arenaCountsFromNumRegs(numRegs)
	a.allocRegistersIntoFromCounts(r, counts)
}

// AllocRegistersIntoCached is the hot-path register-bank allocator.
//
// Used by pushCompiledFrame / handleCall. Consumes the function's pre-computed
// typedSlabCounts (populated once per function via ensurePrecomputedAllocCounts) so the
// 13-field conversion from [NumRegisterKinds]uint32 is paid once per function instead of
// once per call. Combined with the bank-mask short-circuit it reduces the per-call
// AllocRegistersInto cost. The fast paths covered by the bitmask handle the common case
// where most closures/methods touch at most four of the thirteen banks.
//
// Takes r (*Registers) which is the destination register file to populate.
// Takes counts (typedSlabCounts) which is the precomputed slab request derived from
// callee.numRegisters.
// Takes nonZeroMask (uint16) which is the bitmask of banks the callee actually uses; bits
// map to registerKind values and empty banks short-circuit their slice-header write.
func (a *RegisterArena) AllocRegistersIntoCached(r *Registers, counts typedSlabCounts, nonZeroMask uint16) {
	if a.needsGrow(counts) {
		a.growSlabs(counts)
	}
	a.assignCachedScalarBanks(r, counts, nonZeroMask)
	a.assignCachedSliceBanks(r, counts, nonZeroMask)
	a.bumpIndices(counts)
}

// ensurePrecomputedAllocCounts lazily populates the per-function alloc cache. Idempotent
// and safe to call concurrently - the cache fields are only ever transitioned from zero
// to populated, and any torn read just falls through to the (correct) lazy path on the
// next call.
func (cf *CompiledFunction) ensurePrecomputedAllocCounts() {
	if cf.precomputedAllocCountsValid {
		return
	}
	cf.precomputedAllocCounts = arenaCountsFromNumRegs(cf.numRegisters)
	cf.nonZeroBankMask = computeNonZeroBankMask(cf.numRegisters)
	cf.nonEmptyConstantMask = computeNonEmptyConstantMask(cf)
	cf.precomputedAllocCountsValid = true
}

// Save returns a save point capturing the current allocation indices.
//
// Called before AllocRegisters to record where the arena was before allocating a frame's
// registers. Prefer SaveInto on the hot path - returning the 15-field struct by value
// forces Go to build it on the caller's stack and then copy it to its final home (~30
// moves per call instead of 15).
//
// Returns an ArenaSavePoint recording all current slab positions.
func (a *RegisterArena) Save() ArenaSavePoint {
	var sp ArenaSavePoint
	a.SaveInto(&sp)
	return sp
}

// SaveInto writes the current allocation indices directly into the destination
// ArenaSavePoint, avoiding the by-value return that Save() incurs. Used by
// pushCompiledFrame on the hot frame-push path.
//
// Takes sp (*ArenaSavePoint) which is the destination save point to populate; typically
// &callFrame.arenaSave.
//
//go:nosplit
func (a *RegisterArena) SaveInto(sp *ArenaSavePoint) {
	sp.arenaIndices = a.arenaIndices
}

// Restore rolls the arena back to a previous save point, reclaiming all register slots
// allocated since the save point was taken.
//
// Only the GC-visible slabs (string, general, and every typed slice-header bank) are
// zeroed because they hold pointers the GC must not retain. Primitive slabs (int, float,
// bool, uint, complex, upvalue indices) are left dirty because the compiler guarantees a
// register is written before it is read, so stale numeric data is harmless and skipping
// the clear saves significant time in call-heavy workloads.
//
// Takes sp (ArenaSavePoint) which is the save point to restore to.
func (a *RegisterArena) Restore(sp ArenaSavePoint) {
	if sp.stringIndex < a.stringIndex {
		clear(a.stringSlab[sp.stringIndex:a.stringIndex])
	}
	if sp.generalIndex < a.generalIndex {
		clear(a.generalSlab[sp.generalIndex:a.generalIndex])
	}
	if sp.slicesIntIndex < a.slicesIntIndex {
		clear(a.slicesIntSlab[sp.slicesIntIndex:a.slicesIntIndex])
	}
	if sp.slicesFloatIndex < a.slicesFloatIndex {
		clear(a.slicesFloatSlab[sp.slicesFloatIndex:a.slicesFloatIndex])
	}
	if sp.slicesStringIndex < a.slicesStringIndex {
		clear(a.slicesStringSlab[sp.slicesStringIndex:a.slicesStringIndex])
	}
	if sp.slicesBoolIndex < a.slicesBoolIndex {
		clear(a.slicesBoolSlab[sp.slicesBoolIndex:a.slicesBoolIndex])
	}
	if sp.slicesUintIndex < a.slicesUintIndex {
		clear(a.slicesUintSlab[sp.slicesUintIndex:a.slicesUintIndex])
	}
	if sp.slicesByteIndex < a.slicesByteIndex {
		clear(a.slicesByteSlab[sp.slicesByteIndex:a.slicesByteIndex])
	}

	a.arenaIndices = sp.arenaIndices
}

// Reset zeroes the used portions of each slab, resets indices, and shrinks any slabs
// exceeding the DoS protection threshold.
//
// Slabs are not shrunk unless they exceed the threshold, allowing naturally-grown slabs
// to be reused without reallocation.
func (a *RegisterArena) Reset() {
	a.byteSlabPeakIndexLastReset = a.byteIndex
	a.clearAllocatedRegisterRanges()
	a.clearBoxSlabsAndReset()
	a.genericBytesIndex = 0
	clear(a.sliceHeaderSlab[:a.sliceHeaderIndex])
	a.sliceHeaderIndex = 0
	a.resetFrameSlots()
	a.resetAllocationIndices()
	a.dropRetainedOldSlabs()
	a.bytesAllocated = 0
	a.bytesAtLastGC = 0
	a.nextGCAt = 0
	a.gcCount = 0
	a.framesUsed = 0
	a.totalAllocatedBytes = 0
	a.ownerVM = nil
	a.isLeaf = false
	a.shrinkOvergrownSlabs()
}

// ensureFrameStackCapacity grows the parallel frame slabs (frameSlab + callInfoBasesSlab
// + dispatchSavesSlab) when the compile-time depth estimate exceeds the current capacity.
// Idempotent and grow-only; calling with zero or a value already covered is a no-op so
// multiple Execute calls amortise correctly.
//
// Takes minDepth (int) which is the minimum frame-stack depth the arena must support
// without paying a growCallStack allocation.
func (a *RegisterArena) ensureFrameStackCapacity(minDepth int) {
	if minDepth <= len(a.frameSlab) {
		return
	}
	a.growFrameStack(minDepth)
}

// ensureBoxSlabCapacity grows each of the five per-kind box slabs (int / uint / float /
// string / complex) when the supplied counts exceed the current capacity. Each box slab
// is independent so a workload that packs only one kind does not over-provision the other
// four.
//
// Takes counts (typedSlabCounts) which carries the per-kind box slot budget computed by
// sizeArenaFromFunctions.
func (a *RegisterArena) ensureBoxSlabCapacity(counts typedSlabCounts) {
	if counts.intBoxes > len(a.intBoxSlab) {
		a.intBoxSlab = make([]int64, counts.intBoxes)
		a.intBoxIndex = 0
	}
	if counts.uintBoxes > len(a.uintBoxSlab) {
		a.uintBoxSlab = make([]uint64, counts.uintBoxes)
		a.uintBoxIndex = 0
	}
	if counts.floatBoxes > len(a.floatBoxSlab) {
		a.floatBoxSlab = make([]float64, counts.floatBoxes)
		a.floatBoxIndex = 0
	}
	if counts.stringBoxes > len(a.stringBoxSlab) {
		a.stringBoxSlab = make([]string, counts.stringBoxes)
		a.stringBoxIndex = 0
	}
	if counts.complexBoxes > len(a.complexBoxSlab) {
		a.complexBoxSlab = make([]complex128, counts.complexBoxes)
		a.complexBoxIndex = 0
	}
}

// ensureUpvalueCapacity grows the upvalue cell and reference slabs when the supplied
// counts exceed the current capacity. Sized from the per-program opMakeClosure descriptor
// count.
//
// Takes counts (typedSlabCounts) which carries the budget for both upvalue slabs.
func (a *RegisterArena) ensureUpvalueCapacity(counts typedSlabCounts) {
	if counts.upvalueCells > len(a.upvalueCellSlab) {
		a.upvalueCellSlab = make([]upvalueCell, counts.upvalueCells)
	}
	if counts.upvalueReferences > len(a.upvalueReferenceSlab) {
		a.upvalueReferenceSlab = make([]upvalue, counts.upvalueReferences)
	}
}

// clearAllocatedRegisterRanges zeroes the in-use prefix of every register and upvalue
// slab so the next allocation hands out freshly zeroed sub-slices.
func (a *RegisterArena) clearAllocatedRegisterRanges() {
	clear(a.intSlab[:a.intIndex])
	clear(a.floatSlab[:a.floatIndex])
	clear(a.stringSlab[:a.stringIndex])
	clear(a.generalSlab[:a.generalIndex])
	clear(a.boolSlab[:a.boolIndex])
	clear(a.uintSlab[:a.uintIndex])
	clear(a.complexSlab[:a.complexIndex])
	clear(a.slicesIntSlab[:a.slicesIntIndex])
	clear(a.slicesFloatSlab[:a.slicesFloatIndex])
	clear(a.slicesStringSlab[:a.slicesStringIndex])
	clear(a.slicesBoolSlab[:a.slicesBoolIndex])
	clear(a.slicesUintSlab[:a.slicesUintIndex])
	clear(a.slicesByteSlab[:a.slicesByteIndex])
	clear(a.upvalueCellSlab[:a.upvalueCellIndex])
	clear(a.upvalueReferenceSlab[:a.upvalueReferenceIndex])
}

// clearBoxSlabsAndReset zeroes the four per-type box slabs (string, int, float, uint) and
// resets their bump indices.
func (a *RegisterArena) clearBoxSlabsAndReset() {
	clear(a.stringBoxSlab[:a.stringBoxIndex])
	a.stringBoxIndex = 0
	clear(a.intBoxSlab[:a.intBoxIndex])
	a.intBoxIndex = 0
	clear(a.floatBoxSlab[:a.floatBoxIndex])
	a.floatBoxIndex = 0
	clear(a.uintBoxSlab[:a.uintBoxIndex])
	a.uintBoxIndex = 0
	clear(a.complexBoxSlab[:a.complexBoxIndex])
	a.complexBoxIndex = 0
}

// resetFrameSlots zeroes per-frame register slices, function pointers, and shared-cell
// references.
//
// Iterates only [0:framesUsed], the high-water mark of frames touched during the current
// execution and maintained by VM.pushFrame. Frames past framesUsed are guaranteed nil
// (either pristine from make() or cleared by the previous Reset whose own framesUsed
// bound covered them); skipping them avoids dereferencing stale frame.sharedCells
// pointers that would SEGFAULT under pool implementations (stablepool) that persist
// arenas across Go GC cycles.
func (a *RegisterArena) resetFrameSlots() {
	for i := range a.framesUsed {
		f := &a.frameSlab[i]
		f.registers.strings = nil
		f.registers.general = nil
		f.registers.slicesInt = nil
		f.registers.slicesFloat = nil
		f.registers.slicesString = nil
		f.registers.slicesBool = nil
		f.registers.slicesUint = nil
		f.registers.slicesByte = nil
		f.function = nil
		releaseSharedCellMap(f.sharedCells)
		f.sharedCells = nil
		f.upvalues = nil
		f.returnDestination = nil
	}
}

// resetAllocationIndices zeroes every per-slab bump index so the next AllocRegisters call
// hands out sub-slices starting at position 0 in each slab.
func (a *RegisterArena) resetAllocationIndices() {
	a.intIndex = 0
	a.floatIndex = 0
	a.stringIndex = 0
	a.generalIndex = 0
	a.boolIndex = 0
	a.uintIndex = 0
	a.complexIndex = 0
	a.slicesIntIndex = 0
	a.slicesFloatIndex = 0
	a.slicesStringIndex = 0
	a.slicesBoolIndex = 0
	a.slicesUintIndex = 0
	a.slicesByteIndex = 0
	a.byteIndex = 0
	a.intBackingIndex = 0
	a.floatBackingIndex = 0
	a.stringBackingIndex = 0
	a.boolBackingIndex = 0
	a.uintBackingIndex = 0
	a.upvalueCellIndex = 0
	a.upvalueReferenceIndex = 0
}

// dropRetainedOldSlabs truncates the retention lists for retired slabs. Each list is
// cleared before truncating so GC can reclaim the dropped slabs once any remaining user
// references are themselves collected.
func (a *RegisterArena) dropRetainedOldSlabs() {
	a.oldByteSlabs = a.oldByteSlabs[:0]
	clear(a.oldGenericByteSlabs)
	a.oldGenericByteSlabs = a.oldGenericByteSlabs[:0]
	clear(a.oldSliceHeaderSlabs)
	a.oldSliceHeaderSlabs = a.oldSliceHeaderSlabs[:0]
	clear(a.oldIntBackings)
	a.oldIntBackings = a.oldIntBackings[:0]
	clear(a.oldFloatBackings)
	a.oldFloatBackings = a.oldFloatBackings[:0]
	clear(a.oldStringBackings)
	a.oldStringBackings = a.oldStringBackings[:0]
	clear(a.oldBoolBackings)
	a.oldBoolBackings = a.oldBoolBackings[:0]
	clear(a.oldUintBackings)
	a.oldUintBackings = a.oldUintBackings[:0]
}

// allocRegistersIntoFromCounts is the fallback used by AllocRegistersInto when the caller
// has not precomputed counts. Same shape as AllocRegistersIntoCached but derives the
// non-zero mask from the counts struct directly, paying one extra load per bank.
//
// Takes r (*Registers) which is the destination register file to populate.
// Takes counts (typedSlabCounts) which is the per-bank request size.
func (a *RegisterArena) allocRegistersIntoFromCounts(r *Registers, counts typedSlabCounts) {
	if a.needsGrow(counts) {
		a.growSlabs(counts)
	}
	r.ints = a.intSlab[a.intIndex : a.intIndex+counts.ints]
	r.general = a.generalSlab[a.generalIndex : a.generalIndex+counts.generals]
	r.uints = a.uintSlab[a.uintIndex : a.uintIndex+counts.uints]
	a.assignScalarBanksFromCounts(r, counts)
	a.assignSliceBanksFromCounts(r, counts)
	a.bumpIndices(counts)
}

// ensureScalarBankCapacity grows the scalar register banks to fit counts.
//
// Takes counts (typedSlabCounts) which are the required per-kind counts for
// int/float/string/general/bool/uint/complex banks.
func (a *RegisterArena) ensureScalarBankCapacity(counts typedSlabCounts) {
	if counts.ints > len(a.intSlab) {
		a.intSlab = make([]int64, counts.ints)
	}
	if counts.floats > len(a.floatSlab) {
		a.floatSlab = make([]float64, counts.floats)
	}
	if counts.strings > len(a.stringSlab) {
		a.stringSlab = make([]string, counts.strings)
	}
	if counts.generals > len(a.generalSlab) {
		a.generalSlab = make([]reflect.Value, counts.generals)
	}
	if counts.bools > len(a.boolSlab) {
		a.boolSlab = make([]bool, counts.bools)
	}
	if counts.uints > len(a.uintSlab) {
		a.uintSlab = make([]uint64, counts.uints)
	}
	if counts.complexes > len(a.complexSlab) {
		a.complexSlab = make([]complex128, counts.complexes)
	}
}

// ensureSliceBankCapacity grows the typed slice-header banks to fit counts.
//
// Takes counts (typedSlabCounts) which are the required per-kind counts for
// int/float/string/bool/uint/byte slice-header banks.
func (a *RegisterArena) ensureSliceBankCapacity(counts typedSlabCounts) {
	if counts.slicesInts > len(a.slicesIntSlab) {
		a.slicesIntSlab = make([][]int64, counts.slicesInts)
	}
	if counts.slicesFloats > len(a.slicesFloatSlab) {
		a.slicesFloatSlab = make([][]float64, counts.slicesFloats)
	}
	if counts.slicesStrings > len(a.slicesStringSlab) {
		a.slicesStringSlab = make([][]string, counts.slicesStrings)
	}
	if counts.slicesBools > len(a.slicesBoolSlab) {
		a.slicesBoolSlab = make([][]bool, counts.slicesBools)
	}
	if counts.slicesUints > len(a.slicesUintSlab) {
		a.slicesUintSlab = make([][]uint64, counts.slicesUints)
	}
	if counts.slicesBytes > len(a.slicesByteSlab) {
		a.slicesByteSlab = make([][]byte, counts.slicesBytes)
	}
}

// ensureBackingCapacity grows the byte slab and per-type backing slabs used by
// arena-routed make and append-grow.
//
// Takes counts (typedSlabCounts) which are the required per-kind backing capacities.
func (a *RegisterArena) ensureBackingCapacity(counts typedSlabCounts) {
	if counts.bytes > len(a.byteSlab) {
		a.byteSlab = make([]byte, counts.bytes)
	}
	if counts.intBacking > len(a.intBackingSlab) {
		a.intBackingSlab = make([]int64, counts.intBacking)
	}
	if counts.floatBacking > len(a.floatBackingSlab) {
		a.floatBackingSlab = make([]float64, counts.floatBacking)
	}
	if counts.stringBacking > len(a.stringBackingSlab) {
		a.stringBackingSlab = make([]string, counts.stringBacking)
	}
	if counts.boolBacking > len(a.boolBackingSlab) {
		a.boolBackingSlab = make([]bool, counts.boolBacking)
	}
	if counts.uintBacking > len(a.uintBackingSlab) {
		a.uintBackingSlab = make([]uint64, counts.uintBacking)
	}
	if counts.genericBytes > len(a.genericBytesSlab) {
		a.genericBytesSlab = make([]byte, counts.genericBytes)
	}
	if counts.sliceHeaders > len(a.sliceHeaderSlab) {
		a.sliceHeaderSlab = make([]arenaSliceHeader, counts.sliceHeaders)
	}
}

// assignCachedScalarBanks writes the scalar bank slice headers onto r using the
// precomputed mask.
//
// Banks whose bit in nonZeroMask is clear are reset to nil.
//
// Takes r (*Registers) which is the register set receiving the assignments.
// Takes counts (typedSlabCounts) which are the per-kind slot counts to expose on r.
// Takes nonZeroMask (uint16) which is the bitmask selecting which scalar banks receive
// non-nil slices.
func (a *RegisterArena) assignCachedScalarBanks(r *Registers, counts typedSlabCounts, nonZeroMask uint16) {
	if nonZeroMask&allocMaskInt != 0 {
		r.ints = a.intSlab[a.intIndex : a.intIndex+counts.ints]
	} else {
		r.ints = nil
	}
	if nonZeroMask&allocMaskGeneral != 0 {
		r.general = a.generalSlab[a.generalIndex : a.generalIndex+counts.generals]
	} else {
		r.general = nil
	}
	if nonZeroMask&allocMaskUint != 0 {
		r.uints = a.uintSlab[a.uintIndex : a.uintIndex+counts.uints]
	} else {
		r.uints = nil
	}
	if nonZeroMask&allocMaskFloat != 0 {
		r.floats = a.floatSlab[a.floatIndex : a.floatIndex+counts.floats]
	} else {
		r.floats = nil
	}
	if nonZeroMask&allocMaskString != 0 {
		r.strings = a.stringSlab[a.stringIndex : a.stringIndex+counts.strings]
	} else {
		r.strings = nil
	}
	if nonZeroMask&allocMaskBool != 0 {
		r.bools = a.boolSlab[a.boolIndex : a.boolIndex+counts.bools]
	} else {
		r.bools = nil
	}
	if nonZeroMask&allocMaskComplex != 0 {
		r.complex = a.complexSlab[a.complexIndex : a.complexIndex+counts.complexes]
	} else {
		r.complex = nil
	}
}

// assignCachedSliceBanks writes the typed-slice bank slice headers onto r using the
// precomputed mask.
//
// Banks whose bit in nonZeroMask is clear are reset to nil.
//
// Takes r (*Registers) which is the register set receiving the assignments.
// Takes counts (typedSlabCounts) which are the per-kind slice-header counts to expose on
// r.
// Takes nonZeroMask (uint16) which is the bitmask selecting which typed-slice banks
// receive non-nil slices.
func (a *RegisterArena) assignCachedSliceBanks(r *Registers, counts typedSlabCounts, nonZeroMask uint16) {
	if nonZeroMask&allocMaskSliceInt != 0 {
		r.slicesInt = a.slicesIntSlab[a.slicesIntIndex : a.slicesIntIndex+counts.slicesInts]
	} else {
		r.slicesInt = nil
	}
	if nonZeroMask&allocMaskSliceFloat != 0 {
		r.slicesFloat = a.slicesFloatSlab[a.slicesFloatIndex : a.slicesFloatIndex+counts.slicesFloats]
	} else {
		r.slicesFloat = nil
	}
	if nonZeroMask&allocMaskSliceString != 0 {
		r.slicesString = a.slicesStringSlab[a.slicesStringIndex : a.slicesStringIndex+counts.slicesStrings]
	} else {
		r.slicesString = nil
	}
	if nonZeroMask&allocMaskSliceBool != 0 {
		r.slicesBool = a.slicesBoolSlab[a.slicesBoolIndex : a.slicesBoolIndex+counts.slicesBools]
	} else {
		r.slicesBool = nil
	}
	if nonZeroMask&allocMaskSliceUint != 0 {
		r.slicesUint = a.slicesUintSlab[a.slicesUintIndex : a.slicesUintIndex+counts.slicesUints]
	} else {
		r.slicesUint = nil
	}
	if nonZeroMask&allocMaskSliceByte != 0 {
		r.slicesByte = a.slicesByteSlab[a.slicesByteIndex : a.slicesByteIndex+counts.slicesBytes]
	} else {
		r.slicesByte = nil
	}
}

// assignScalarBanksFromCounts writes the optional scalar register banks onto r derived
// from counts.
//
// Banks with a zero count receive nil slices.
//
// Takes r (*Registers) which is the register set receiving the assignments.
// Takes counts (typedSlabCounts) which are the per-kind slot counts
// (float/string/bool/complex banks).
func (a *RegisterArena) assignScalarBanksFromCounts(r *Registers, counts typedSlabCounts) {
	if counts.floats > 0 {
		r.floats = a.floatSlab[a.floatIndex : a.floatIndex+counts.floats]
	} else {
		r.floats = nil
	}
	if counts.strings > 0 {
		r.strings = a.stringSlab[a.stringIndex : a.stringIndex+counts.strings]
	} else {
		r.strings = nil
	}
	if counts.bools > 0 {
		r.bools = a.boolSlab[a.boolIndex : a.boolIndex+counts.bools]
	} else {
		r.bools = nil
	}
	if counts.complexes > 0 {
		r.complex = a.complexSlab[a.complexIndex : a.complexIndex+counts.complexes]
	} else {
		r.complex = nil
	}
}

// assignSliceBanksFromCounts writes the typed-slice register banks onto r derived from
// counts.
//
// Banks with a zero count receive nil slices.
//
// Takes r (*Registers) which is the register set receiving the assignments.
// Takes counts (typedSlabCounts) which are the per-kind slice-header counts
// (int/float/string/bool/uint/byte banks).
func (a *RegisterArena) assignSliceBanksFromCounts(r *Registers, counts typedSlabCounts) {
	if counts.slicesInts > 0 {
		r.slicesInt = a.slicesIntSlab[a.slicesIntIndex : a.slicesIntIndex+counts.slicesInts]
	} else {
		r.slicesInt = nil
	}
	if counts.slicesFloats > 0 {
		r.slicesFloat = a.slicesFloatSlab[a.slicesFloatIndex : a.slicesFloatIndex+counts.slicesFloats]
	} else {
		r.slicesFloat = nil
	}
	if counts.slicesStrings > 0 {
		r.slicesString = a.slicesStringSlab[a.slicesStringIndex : a.slicesStringIndex+counts.slicesStrings]
	} else {
		r.slicesString = nil
	}
	if counts.slicesBools > 0 {
		r.slicesBool = a.slicesBoolSlab[a.slicesBoolIndex : a.slicesBoolIndex+counts.slicesBools]
	} else {
		r.slicesBool = nil
	}
	if counts.slicesUints > 0 {
		r.slicesUint = a.slicesUintSlab[a.slicesUintIndex : a.slicesUintIndex+counts.slicesUints]
	} else {
		r.slicesUint = nil
	}
	if counts.slicesBytes > 0 {
		r.slicesByte = a.slicesByteSlab[a.slicesByteIndex : a.slicesByteIndex+counts.slicesBytes]
	} else {
		r.slicesByte = nil
	}
}

// needsGrow reports whether at least one slab is too small to satisfy the requested
// counts at its current allocation index.
//
// Takes counts (typedSlabCounts) which is the per-bank request size.
//
// Returns true when any slab requires growth before AllocRegisters can hand out a
// sub-slice.
func (a *RegisterArena) needsGrow(counts typedSlabCounts) bool {
	return a.intIndex+counts.ints > len(a.intSlab) ||
		a.floatIndex+counts.floats > len(a.floatSlab) ||
		a.stringIndex+counts.strings > len(a.stringSlab) ||
		a.generalIndex+counts.generals > len(a.generalSlab) ||
		a.boolIndex+counts.bools > len(a.boolSlab) ||
		a.uintIndex+counts.uints > len(a.uintSlab) ||
		a.complexIndex+counts.complexes > len(a.complexSlab) ||
		a.slicesIntIndex+counts.slicesInts > len(a.slicesIntSlab) ||
		a.slicesFloatIndex+counts.slicesFloats > len(a.slicesFloatSlab) ||
		a.slicesStringIndex+counts.slicesStrings > len(a.slicesStringSlab) ||
		a.slicesBoolIndex+counts.slicesBools > len(a.slicesBoolSlab) ||
		a.slicesUintIndex+counts.slicesUints > len(a.slicesUintSlab) ||
		a.slicesByteIndex+counts.slicesBytes > len(a.slicesByteSlab)
}

// bumpIndices advances the arena's per-bank allocation indices by the per-bank request
// size after sub-slices have been handed out.
//
// Takes counts (typedSlabCounts) which is the per-bank request size just satisfied.
func (a *RegisterArena) bumpIndices(counts typedSlabCounts) {
	a.intIndex += counts.ints
	a.floatIndex += counts.floats
	a.stringIndex += counts.strings
	a.generalIndex += counts.generals
	a.boolIndex += counts.bools
	a.uintIndex += counts.uints
	a.complexIndex += counts.complexes
	a.slicesIntIndex += counts.slicesInts
	a.slicesFloatIndex += counts.slicesFloats
	a.slicesStringIndex += counts.slicesStrings
	a.slicesBoolIndex += counts.slicesBools
	a.slicesUintIndex += counts.slicesUints
	a.slicesByteIndex += counts.slicesBytes
}

// shrinkOvergrownSlabs replaces any slab that has grown beyond the DoS protection
// threshold with a fresh default-sized allocation.
func (a *RegisterArena) shrinkOvergrownSlabs() {
	a.shrinkOvergrownScalarSlabs()
	a.shrinkOvergrownSliceSlabs()
	a.shrinkOvergrownFrameSlabs()
	a.shrinkOvergrownBackingSlabs()
}

// shrinkOvergrownScalarSlabs replaces scalar register banks that grew past the DoS
// protection threshold with fresh default-sized allocations.
func (a *RegisterArena) shrinkOvergrownScalarSlabs() {
	if len(a.intSlab) > initialIntSlabs*maxArenaMultiplier {
		a.intSlab = make([]int64, initialIntSlabs)
	}
	if len(a.floatSlab) > initialFloatSlabs*maxArenaMultiplier {
		a.floatSlab = make([]float64, initialFloatSlabs)
	}
	if len(a.stringSlab) > initialStringSlabs*maxArenaMultiplier {
		a.stringSlab = make([]string, initialStringSlabs)
	}
	if len(a.generalSlab) > initialGeneralSlabs*maxArenaMultiplier {
		a.generalSlab = make([]reflect.Value, initialGeneralSlabs)
	}
	if len(a.boolSlab) > initialBoolSlabs*maxArenaMultiplier {
		a.boolSlab = make([]bool, initialBoolSlabs)
	}
	if len(a.uintSlab) > initialUintSlabs*maxArenaMultiplier {
		a.uintSlab = make([]uint64, initialUintSlabs)
	}
	if len(a.complexSlab) > initialComplexSlabs*maxArenaMultiplier {
		a.complexSlab = make([]complex128, initialComplexSlabs)
	}
}

// shrinkOvergrownSliceSlabs replaces typed-slice banks (int/float/ string/bool/uint/byte)
// that grew past the threshold with fresh default-sized allocations.
func (a *RegisterArena) shrinkOvergrownSliceSlabs() {
	if len(a.slicesIntSlab) > initialSlicesIntSlabs*maxArenaMultiplier {
		a.slicesIntSlab = make([][]int64, initialSlicesIntSlabs)
	}
	if len(a.slicesFloatSlab) > initialSlicesFloatSlabs*maxArenaMultiplier {
		a.slicesFloatSlab = make([][]float64, initialSlicesFloatSlabs)
	}
	if len(a.slicesStringSlab) > initialSlicesStringSlabs*maxArenaMultiplier {
		a.slicesStringSlab = make([][]string, initialSlicesStringSlabs)
	}
	if len(a.slicesBoolSlab) > initialSlicesBoolSlabs*maxArenaMultiplier {
		a.slicesBoolSlab = make([][]bool, initialSlicesBoolSlabs)
	}
	if len(a.slicesUintSlab) > initialSlicesUintSlabs*maxArenaMultiplier {
		a.slicesUintSlab = make([][]uint64, initialSlicesUintSlabs)
	}
	if len(a.slicesByteSlab) > initialSlicesByteSlabs*maxArenaMultiplier {
		a.slicesByteSlab = make([][]byte, initialSlicesByteSlabs)
	}
}

// shrinkOvergrownFrameSlabs replaces frame-adjacent slabs (frames, call-info bases,
// dispatch saves, upvalue cells and refs) that grew past the threshold.
func (a *RegisterArena) shrinkOvergrownFrameSlabs() {
	if len(a.frameSlab) > initialFrameSlabs*maxArenaMultiplier {
		a.frameSlab = make([]callFrame, initialFrameSlabs)
	}
	if len(a.callInfoBasesSlab) > initialFrameSlabs*maxArenaMultiplier {
		a.callInfoBasesSlab = make([]uintptr, initialFrameSlabs)
	}
	if len(a.dispatchSavesSlab) > initialFrameSlabs*maxArenaMultiplier {
		a.dispatchSavesSlab = make([]asmDispatchSave, initialFrameSlabs)
	}
	if len(a.upvalueCellSlab) > initialUpvalueCellSlabs*maxArenaMultiplier {
		a.upvalueCellSlab = make([]upvalueCell, initialUpvalueCellSlabs)
	}
	if len(a.upvalueReferenceSlab) > initialUpvalueRefSlabs*maxArenaMultiplier {
		a.upvalueReferenceSlab = make([]upvalue, initialUpvalueRefSlabs)
	}
}

// shrinkOvergrownBackingSlabs replaces the byte slab and the typed backing slabs (used by
// arena-routed make() and append-grow) that grew past the threshold.
func (a *RegisterArena) shrinkOvergrownBackingSlabs() {
	if len(a.byteSlab) > initialByteSlabSize*maxArenaMultiplier {
		if a.byteSlabPeakIndexLastReset > len(a.byteSlab)/backingSlabLowUseRatioDenominator {
			a.byteSlabIdleResetCount = 0
		} else {
			a.byteSlabIdleResetCount++
			if a.byteSlabIdleResetCount >= backingSlabIdleResetsBeforeShrink {
				a.byteSlab = make([]byte, initialByteSlabSize)
				a.byteSlabIdleResetCount = 0
			}
		}
	} else {
		a.byteSlabIdleResetCount = 0
	}
	if len(a.intBackingSlab) > initialIntBackingSize*maxArenaMultiplier {
		a.intBackingSlab = make([]int64, initialIntBackingSize)
	}
	if len(a.floatBackingSlab) > initialFloatBackingSize*maxArenaMultiplier {
		a.floatBackingSlab = make([]float64, initialFloatBackingSize)
	}
	if len(a.stringBackingSlab) > initialStringBackingSize*maxArenaMultiplier {
		a.stringBackingSlab = make([]string, initialStringBackingSize)
	}
	if len(a.boolBackingSlab) > initialBoolBackingSize*maxArenaMultiplier {
		a.boolBackingSlab = make([]bool, initialBoolBackingSize)
	}
	if len(a.uintBackingSlab) > initialUintBackingSize*maxArenaMultiplier {
		a.uintBackingSlab = make([]uint64, initialUintBackingSize)
	}
}

// frameStack returns the arena's pre-allocated call frame slab.
//
// Returns the frameSlab slice.
func (a *RegisterArena) frameStack() []callFrame {
	return a.frameSlab
}

// dispatchSaves returns the arena's pre-allocated slab for ASM dispatch register saves,
// parallel to the frame stack.
//
// Returns the dispatchSavesSlab slice.
func (a *RegisterArena) dispatchSaves() []asmDispatchSave {
	return a.dispatchSavesSlab
}

// allocUpvalueCells bump-allocates n upvalueCell slots from the arena.
//
// Takes n (int) which is the number of upvalue cells to allocate.
//
// Returns a zeroed slice of upvalueCell of length n.
func (a *RegisterArena) allocUpvalueCells(n int) []upvalueCell {
	if a.upvalueCellIndex+n > len(a.upvalueCellSlab) {
		a.growUpvalueCellSlab(a.upvalueCellIndex + n)
	}
	start := a.upvalueCellIndex
	a.upvalueCellIndex += n
	cells := a.upvalueCellSlab[start:a.upvalueCellIndex]
	clear(cells)
	return cells
}

// allocUpvalueRefs bump-allocates n upvalue slots from the arena.
//
// Takes n (int) which is the number of upvalue references to allocate.
//
// Returns a zeroed slice of upvalue of length n.
func (a *RegisterArena) allocUpvalueRefs(n int) []upvalue {
	if a.upvalueReferenceIndex+n > len(a.upvalueReferenceSlab) {
		a.growUpvalueRefSlab(a.upvalueReferenceIndex + n)
	}
	start := a.upvalueReferenceIndex
	a.upvalueReferenceIndex += n
	refs := a.upvalueReferenceSlab[start:a.upvalueReferenceIndex]
	clear(refs)
	return refs
}

// resolveDefaultMaxArenaBytes returns the host-aware arena budget.
//
// When detectHostMemoryBytes cannot probe, it falls back to
// defaultMaxArenaBytesUndetectedFallback. Otherwise the conservative candidate is one
// quarter of host RAM (right-shift by defaultMaxArenaBytesNormalShift); if that already
// exceeds the small-host threshold it is clamped by defaultMaxArenaBytesCeiling and
// returned, and if below the threshold it is recomputed as hostBytes * 3 / 4 and clamped
// by the same ceiling.
//
// Returns the resolved per-Execute byte budget.
func resolveDefaultMaxArenaBytes() uint64 {
	hostBytes := detectHostMemoryBytes()
	if hostBytes == 0 {
		return defaultMaxArenaBytesUndetectedFallback
	}
	candidate := hostBytes >> defaultMaxArenaBytesNormalShift
	if candidate < defaultMaxArenaBytesSmallHostThreshold {
		candidate = hostBytes / defaultMaxArenaBytesGenerousDenominator * defaultMaxArenaBytesGenerousNumerator
	}
	if candidate > defaultMaxArenaBytesCeiling {
		return defaultMaxArenaBytesCeiling
	}
	return candidate
}

// initRegisterArenaInPlace populates an existing RegisterArena slot with fresh slabs.
//
// Takes a (*RegisterArena) which is the arena slot to initialise; must be non-nil.
func initRegisterArenaInPlace(a *RegisterArena) {
	a.intSlab = make([]int64, initialIntSlabs)
	a.floatSlab = make([]float64, initialFloatSlabs)
	a.stringSlab = make([]string, initialStringSlabs)
	a.generalSlab = make([]reflect.Value, initialGeneralSlabs)
	a.boolSlab = make([]bool, initialBoolSlabs)
	a.uintSlab = make([]uint64, initialUintSlabs)
	a.complexSlab = make([]complex128, initialComplexSlabs)
	a.slicesIntSlab = make([][]int64, initialSlicesIntSlabs)
	a.slicesFloatSlab = make([][]float64, initialSlicesFloatSlabs)
	a.slicesStringSlab = make([][]string, initialSlicesStringSlabs)
	a.slicesBoolSlab = make([][]bool, initialSlicesBoolSlabs)
	a.slicesUintSlab = make([][]uint64, initialSlicesUintSlabs)
	a.slicesByteSlab = make([][]byte, initialSlicesByteSlabs)
	a.frameSlab = make([]callFrame, initialFrameSlabs)
	a.callInfoBasesSlab = make([]uintptr, initialFrameSlabs)
	a.dispatchSavesSlab = make([]asmDispatchSave, initialFrameSlabs)
	a.byteSlab = make([]byte, initialByteSlabSize)
	a.sliceHeaderSlab = make([]arenaSliceHeader, initialSliceHeaderCapacity)
	a.intBackingSlab = make([]int64, initialIntBackingSize)
	a.floatBackingSlab = make([]float64, initialFloatBackingSize)
	a.stringBackingSlab = make([]string, initialStringBackingSize)
	a.boolBackingSlab = make([]bool, initialBoolBackingSize)
	a.uintBackingSlab = make([]uint64, initialUintBackingSize)
	a.upvalueCellSlab = make([]upvalueCell, initialUpvalueCellSlabs)
	a.upvalueReferenceSlab = make([]upvalue, initialUpvalueRefSlabs)
}

// arenaCountsFromNumRegs unpacks a per-kind register-count array into the typedSlabCounts
// struct used by the arena's allocation machinery. Centralising the unpack means a new
// bank kind only requires an additional struct field plus a single new line here, not a
// parallel rewrite of every alloc/grow site.
//
// Takes numRegs ([NumRegisterKinds]uint32) which is the per-kind register count for the
// frame being allocated.
//
// Returns the equivalent typedSlabCounts.
func arenaCountsFromNumRegs(numRegs [NumRegisterKinds]uint32) typedSlabCounts {
	return typedSlabCounts{
		ints:          int(numRegs[registerInt]),
		floats:        int(numRegs[registerFloat]),
		strings:       int(numRegs[registerString]),
		generals:      int(numRegs[registerGeneral]),
		bools:         int(numRegs[registerBool]),
		uints:         int(numRegs[registerUint]),
		complexes:     int(numRegs[registerComplex]),
		slicesInts:    int(numRegs[registerSliceInt]),
		slicesFloats:  int(numRegs[registerSliceFloat]),
		slicesStrings: int(numRegs[registerSliceString]),
		slicesBools:   int(numRegs[registerSliceBool]),
		slicesUints:   int(numRegs[registerSliceUint]),
		slicesBytes:   int(numRegs[registerSliceByte]),
	}
}

// computeNonEmptyConstantMask returns the bitmask of non-empty constant pools.
//
// Used by rebuildDispatchPointers to skip empty-pool base-pointer writes. The bits map to
// the constMask* constants; keep this and those in sync.
//
// Takes cf (*CompiledFunction) which carries the constant pools and type/layout tables
// being inspected.
//
// Returns the OR of every constMask* bit whose backing pool is non-empty.
func computeNonEmptyConstantMask(cf *CompiledFunction) uint16 {
	var mask uint16
	if len(cf.intConstants) > 0 {
		mask |= constMaskInt
	}
	if len(cf.floatConstants) > 0 {
		mask |= constMaskFloat
	}
	if len(cf.stringConstants) > 0 {
		mask |= constMaskString
	}
	if len(cf.boolConstants) > 0 {
		mask |= constMaskBool
	}
	if len(cf.structLayoutTable) > 0 {
		mask |= constMaskStructLayoutTable
	}
	if len(cf.typeTable) > 0 {
		mask |= constMaskTypeTable
	}
	if len(cf.complexConstants) > 0 {
		mask |= constMaskComplex
	}
	if len(cf.uintConstants) > 0 {
		mask |= constMaskUint
	}
	return mask
}

// computeNonZeroBankMask returns the bitmask of in-use register banks.
//
// Bit i is set when numRegisters[i] > 0. Used by AllocRegistersIntoCached's bank-mask
// short-circuit.
//
// Takes numRegs ([NumRegisterKinds]uint32) which is the per-kind register count for the
// callee.
//
// Returns the OR of allocMask* bits for every bank with a non-zero count.
func computeNonZeroBankMask(numRegs [NumRegisterKinds]uint32) uint16 {
	var mask uint16
	for k := range registerKind(NumRegisterKinds) {
		if numRegs[k] > 0 {
			mask |= uint16(1) << uint(k)
		}
	}
	return mask
}

// GetRegisterArena retrieves a RegisterArena from the pool.
//
// Returns a reset RegisterArena ready for use. Allocates a fresh arena via
// initRegisterArenaInPlace if the pool is exhausted past its growth ceiling.
func GetRegisterArena() *RegisterArena {
	return registerArenaPool.MustGet()
}

// PutRegisterArena returns a RegisterArena to the pool after resetting.
//
// Takes a (*RegisterArena) which is the arena to return to the pool.
func PutRegisterArena(a *RegisterArena) {
	if a == nil {
		return
	}
	a.Reset()
	registerArenaPool.Put(a)
}

// materialiseString returns a heap-backed copy of s if it points into the arena's byte
// slabs.
//
// Takes arena (*RegisterArena) which is the arena whose byte slabs are checked for
// ownership.
// Takes s (string) which is the string to materialise.
//
// Returns a cloned string if s points into the arena, or s unchanged if it is already
// heap-backed.
func materialiseString(arena *RegisterArena, s string) string {
	if arena == nil || !arena.isLeaf {
		return s
	}
	return materialiseStringUnconditional(arena, s)
}

// materialiseStringUnconditional always heap-clones s when it points into the arena,
// regardless of the arena's leaf status. Used by boundary paths (return values, panic
// captures) where the arena is about to Reset whether it is the main arena or a leaf
// child.
//
// Takes arena (*RegisterArena) which owns the candidate slabs.
// Takes s (string) which is the candidate string.
//
// Returns a heap-cloned copy when arena-backed, or s unchanged.
func materialiseStringUnconditional(arena *RegisterArena, s string) string {
	if arena.ownsString(s) {
		return strings.Clone(s)
	}
	return s
}

// newRegisterArena creates a fresh arena with default slab sizes.
//
// Returns a newly allocated RegisterArena with all slabs at initial capacity.
func newRegisterArena() *RegisterArena {
	return &RegisterArena{
		intSlab:              make([]int64, initialIntSlabs),
		floatSlab:            make([]float64, initialFloatSlabs),
		stringSlab:           make([]string, initialStringSlabs),
		generalSlab:          make([]reflect.Value, initialGeneralSlabs),
		boolSlab:             make([]bool, initialBoolSlabs),
		uintSlab:             make([]uint64, initialUintSlabs),
		complexSlab:          make([]complex128, initialComplexSlabs),
		slicesIntSlab:        make([][]int64, initialSlicesIntSlabs),
		slicesFloatSlab:      make([][]float64, initialSlicesFloatSlabs),
		slicesStringSlab:     make([][]string, initialSlicesStringSlabs),
		slicesBoolSlab:       make([][]bool, initialSlicesBoolSlabs),
		slicesUintSlab:       make([][]uint64, initialSlicesUintSlabs),
		slicesByteSlab:       make([][]byte, initialSlicesByteSlabs),
		frameSlab:            make([]callFrame, initialFrameSlabs),
		callInfoBasesSlab:    make([]uintptr, initialFrameSlabs),
		dispatchSavesSlab:    make([]asmDispatchSave, initialFrameSlabs),
		byteSlab:             make([]byte, initialByteSlabSize),
		intBackingSlab:       make([]int64, initialIntBackingSize),
		floatBackingSlab:     make([]float64, initialFloatBackingSize),
		stringBackingSlab:    make([]string, initialStringBackingSize),
		boolBackingSlab:      make([]bool, initialBoolBackingSize),
		uintBackingSlab:      make([]uint64, initialUintBackingSize),
		upvalueCellSlab:      make([]upvalueCell, initialUpvalueCellSlabs),
		upvalueReferenceSlab: make([]upvalue, initialUpvalueRefSlabs),
	}
}
