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

package ast_domain

// Provides a pooled arena allocator that replaces per-object sync.Pool
// operations with a single arena allocation per request.

import "sync"

const (
	// initialNodeCount is the initial capacity for template nodes in a render arena.
	initialNodeCount = 512

	// initialDirectWriters is the initial capacity for the direct writers slice.
	initialDirectWriters = 512

	// initialByteBufs is the initial capacity for the byte buffer slice.
	initialByteBufs = 512

	// initialAnnotations is the starting capacity for the annotations slice.
	initialAnnotations = 32

	// defaultByteBufCap is the default capacity in bytes for byte buffers.
	defaultByteBufCap = 512

	// maxNodeCount is the upper limit for the node slice before it is reset.
	maxNodeCount = initialNodeCount * 8

	// maxDirectWriters is the upper limit for direct writer slice capacity.
	maxDirectWriters = initialDirectWriters * 8

	// maxByteBufs is the maximum number of byte buffers before the pool is reset.
	maxByteBufs = initialByteBufs * 8

	// maxAnnotations is the limit at which the annotations slice is reallocated.
	maxAnnotations = initialAnnotations * 8

	// maxBucketMultiplier is the growth threshold for slab reallocation.
	// When a bucket exceeds this multiple of its initial size, it is reset.
	maxBucketMultiplier = 8

	// maxByteBufCapacity is the maximum capacity in bytes before a buffer is reset.
	maxByteBufCapacity = 64 * 1024
)

var (
	// childBucketCaps defines the available bucket capacities for child slices.
	childBucketCaps = [13]int{2, 4, 6, 8, 10, 12, 16, 24, 32, 48, 64, 96, 128}

	// attributeBucketCapacities defines the bucket capacities for attribute slices.
	attributeBucketCapacities = [7]int{2, 4, 6, 8, 10, 12, 16}

	// attributeWriterBucketCapacities defines the slice bucket
	// capacities for the attribute writer, using sizes
	// 2, 4, 8, 10, 12, and 16.
	attributeWriterBucketCapacities = [6]int{2, 4, 8, 10, 12, 16}

	// rootNodesBucketCaps defines the bucket capacities for root nodes slices.
	// Values are: 1, 2, 4, 6, 8, 10, 12, 16, 24, 32, 48, 64, 96, 128.
	rootNodesBucketCaps = [14]int{1, 2, 4, 6, 8, 10, 12, 16, 24, 32, 48, 64, 96, 128}

	// initialChildCounts holds the initial bucket counts for high-water mark
	// comparisons during Reset. These values match those used in newRenderArena.
	initialChildCounts = [13]int{128, 96, 64, 48, 32, 32, 16, 16, 8, 8, 4, 4, 4}

	// initialAttrCounts holds the initial bucket counts for attribute slab allocation.
	initialAttrCounts = [7]int{192, 128, 64, 32, 16, 16, 8}

	// initialWriterCounts holds the initial bucket counts for attribute writer slab allocation.
	initialWriterCounts = [6]int{128, 64, 32, 16, 16, 8}

	// initialRootNodesCounts holds the initial bucket counts for root node slab allocation.
	initialRootNodesCounts = [14]int{8, 8, 8, 4, 4, 4, 4, 4, 2, 2, 2, 2, 2, 2}

	// arenaPool is the single sync.Pool for arena instances.
	arenaPool = sync.Pool{
		New: func() any {
			return newRenderArena()
		},
	}
)

// slabBucket holds pre-allocated backing arrays for slices of a specific capacity.
type slabBucket[T any] struct {
	// backing holds pre-allocated slices for reuse.
	backing []T

	// used tracks the number of allocated slices in this bucket.
	used int

	// cap is the initial capacity for slices in this bucket.
	cap int
}

// RenderArena is a pooled container holding pre-allocated slabs for all AST
// types used during a single render request. The entire arena is obtained and
// released as a single unit, rather than managing each allocation separately.
type RenderArena struct {
	// ast holds the reusable TemplateAST; typically one per render.
	ast TemplateAST

	// byteBufs holds byte buffers for DirectWriter operations.
	byteBufs [][]byte

	// nodes is a contiguous slab of TemplateNode values for arena allocation.
	nodes []TemplateNode

	// directWriters is a slab of DirectWriter instances for reuse.
	directWriters []DirectWriter

	// annotations is a slab of RuntimeAnnotation values for reuse.
	annotations []RuntimeAnnotation

	// rootNodesSlabs holds pre-allocated slices of template node pointers,
	// organised by capacity bucket for efficient reuse.
	rootNodesSlabs [14]slabBucket[[]*TemplateNode]

	// childSlabs holds the child slice slabs organised by capacity bucket.
	// Buckets: 2, 4, 6, 8, 10, 12, 16, 24, 32, 48, 64, 96, 128.
	childSlabs [13]slabBucket[[]*TemplateNode]

	// attributeSlabs holds attribute slice storage organised by capacity bucket.
	// Buckets have capacities: 2, 4, 6, 8, 10, 12, 16.
	attributeSlabs [7]slabBucket[[]HTMLAttribute]

	// attributeWriterSlabs holds buckets for attribute writer slices, sized at
	// capacities 2, 4, 8, 10, 12, and 16.
	attributeWriterSlabs [6]slabBucket[[]*DirectWriter]

	// byteBufferIndex is the next available index in byteBufs;
	// reset to 0 on arena reset.
	byteBufferIndex int

	// directWriterIndex is the next available index in the directWriters slab.
	directWriterIndex int

	// annotationIndex is the next available index in the annotations slab.
	annotationIndex int

	// nodeIndex is the next available index in the nodes slice.
	nodeIndex int

	// astUsed tracks whether the arena's AST has been allocated.
	astUsed bool
}

// GetNode returns a zeroed TemplateNode from the arena's slab.
//
// Returns *TemplateNode which is a pooled node ready for use.
func (a *RenderArena) GetNode() *TemplateNode {
	if a.nodeIndex >= len(a.nodes) {
		a.growNodes()
	}
	node := &a.nodes[a.nodeIndex]
	a.nodeIndex++
	node.IsPooled = true
	return node
}

// GetChildSlice returns a child slice of at least the requested capacity.
//
// Takes capacity (int) which specifies the minimum slice capacity required.
//
// Returns *[]*TemplateNode which is a pointer to the backing slice for API
// compatibility, or nil if capacity exceeds the maximum bucket size.
// Returns []*TemplateNode which is the slice ready for use, or nil if
// capacity is zero or negative.
func (a *RenderArena) GetChildSlice(capacity int) (*[]*TemplateNode, []*TemplateNode) {
	if capacity <= 0 {
		return nil, nil
	}

	bucketIndex := a.childBucketIndex(capacity)
	if bucketIndex < 0 {
		s := make([]*TemplateNode, 0, capacity)
		return nil, s
	}

	return getSliceFromBucket(&a.childSlabs[bucketIndex], a.growChildBucket, bucketIndex, func(s *[]*TemplateNode) { *s = (*s)[:0] })
}

// GetAttrSlice returns an attribute slice of at least the requested capacity.
//
// Takes capacity (int) which specifies the minimum slice capacity needed.
//
// Returns *[]HTMLAttribute which is the backing slice pointer for reuse, or nil
// if capacity is zero or negative, or if the capacity exceeds bucket sizes.
// Returns []HTMLAttribute which is the slice ready for use, or nil if capacity
// is zero or negative.
func (a *RenderArena) GetAttrSlice(capacity int) (*[]HTMLAttribute, []HTMLAttribute) {
	if capacity <= 0 {
		return nil, nil
	}

	bucketIndex := a.attributeBucketIndex(capacity)
	if bucketIndex < 0 {
		s := make([]HTMLAttribute, 0, capacity)
		return nil, s
	}

	return getSliceFromBucket(&a.attributeSlabs[bucketIndex], a.growAttrBucket, bucketIndex, func(s *[]HTMLAttribute) { *s = (*s)[:0] })
}

// GetAttrWriterSlice returns an attribute writer slice of at least the
// requested capacity.
//
// Takes capacity (int) which specifies the minimum size of the slice.
//
// Returns *[]*DirectWriter which is the backing array pointer for reuse, or
// nil if the slice cannot be pooled.
// Returns []*DirectWriter which is the slice ready for use.
func (a *RenderArena) GetAttrWriterSlice(capacity int) (*[]*DirectWriter, []*DirectWriter) {
	if capacity <= 0 {
		return nil, nil
	}

	bucketIndex := a.attributeWriterBucketIndex(capacity)
	if bucketIndex < 0 {
		s := make([]*DirectWriter, 0, capacity)
		return nil, s
	}

	return getSliceFromBucket(&a.attributeWriterSlabs[bucketIndex], a.growAttrWriterBucket, bucketIndex, func(s *[]*DirectWriter) { *s = (*s)[:0] })
}

// GetDirectWriter returns a zeroed DirectWriter from the slab.
//
// Returns *DirectWriter which is ready for use.
func (a *RenderArena) GetDirectWriter() *DirectWriter {
	if a.directWriterIndex >= len(a.directWriters) {
		a.growDirectWriters()
	}
	dw := &a.directWriters[a.directWriterIndex]
	a.directWriterIndex++
	return dw
}

// GetByteBuf returns a byte buffer from the arena.
//
// Returns *[]byte which is a reset buffer ready for use.
func (a *RenderArena) GetByteBuf() *[]byte {
	if a.byteBufferIndex >= len(a.byteBufs) {
		a.growByteBufs()
	}
	buffer := &a.byteBufs[a.byteBufferIndex]
	*buffer = (*buffer)[:0]
	a.byteBufferIndex++
	return buffer
}

// GetRuntimeAnnotation returns a RuntimeAnnotation from the slab.
//
// Returns *RuntimeAnnotation which is the next available annotation from the
// arena, growing the pool if needed.
func (a *RenderArena) GetRuntimeAnnotation() *RuntimeAnnotation {
	if a.annotationIndex >= len(a.annotations) {
		a.growAnnotations()
	}
	ra := &a.annotations[a.annotationIndex]
	a.annotationIndex++
	return ra
}

// GetTemplateAST returns the arena's single TemplateAST.
//
// Returns *TemplateAST which is the pooled template AST for this arena.
func (a *RenderArena) GetTemplateAST() *TemplateAST {
	if a.astUsed {
		return GetTemplateAST()
	}
	a.astUsed = true
	a.ast.isPooled = true
	return &a.ast
}

// GetRootNodesSlice returns a root nodes slice of at least the requested
// capacity.
//
// Takes capacity (int) which specifies the minimum slice capacity needed.
//
// Returns []*TemplateNode which is a slice with the requested capacity, or nil
// if capacity is zero or negative.
func (a *RenderArena) GetRootNodesSlice(capacity int) []*TemplateNode {
	if capacity <= 0 {
		return nil
	}

	bucketIndex := a.rootNodesBucketIndex(capacity)
	if bucketIndex < 0 {
		return make([]*TemplateNode, 0, capacity)
	}

	bucket := &a.rootNodesSlabs[bucketIndex]
	if bucket.used >= len(bucket.backing) {
		a.growRootNodesBucket(bucketIndex)
	}

	backing := bucket.backing[bucket.used]
	bucket.used++
	return backing[:0]
}

// Reset clears all indices and resets used objects, preparing the arena for
// reuse. This is called by PutArena before returning the arena to the pool.
//
// For DoS protection, slabs that have grown beyond high-water mark limits
// are shrunk back to initial size. This prevents memory bloat from malicious
// or pathological requests while allowing legitimate growth to be retained.
func (a *RenderArena) Reset() {
	a.resetNodes()
	a.resetChildSlabs()
	a.resetAttrSlabs()
	a.resetAttrWriterSlabs()
	a.resetDirectWriters()
	a.resetByteBufs()
	a.resetAnnotations()
	a.resetAST()
	a.resetRootNodesSlabs()
}

// growNodes doubles the capacity of the node slice.
func (a *RenderArena) growNodes() {
	newSize := len(a.nodes) * 2
	newNodes := make([]TemplateNode, newSize)
	copy(newNodes, a.nodes)
	a.nodes = newNodes
}

//revive:disable:add-constant

// childBucketIndex returns the bucket index for the given capacity.
//
// Takes capacity (int) which is the required slice capacity.
//
// Returns int which is the bucket index, or -1 if the capacity exceeds 128.
func (*RenderArena) childBucketIndex(capacity int) int {
	switch {
	case capacity <= 2:
		return 0
	case capacity <= 4:
		return 1
	case capacity <= 6:
		return 2
	case capacity <= 8:
		return 3
	case capacity <= 10:
		return 4
	case capacity <= 12:
		return 5
	case capacity <= 16:
		return 6
	case capacity <= 24:
		return 7
	case capacity <= 32:
		return 8
	case capacity <= 48:
		return 9
	case capacity <= 64:
		return 10
	case capacity <= 96:
		return 11
	case capacity <= 128:
		return 12
	default:
		return -1
	}
}

// growChildBucket doubles the capacity of the child bucket at the given index.
//
// Takes index (int) which specifies the bucket index to grow.
func (a *RenderArena) growChildBucket(index int) {
	growSlabBucket(&a.childSlabs[index], func(c int) []*TemplateNode { return make([]*TemplateNode, 0, c) })
}

// attributeBucketIndex returns the bucket index for a given attribute capacity.
//
// Takes capacity (int) which is the number of attributes to store.
//
// Returns int which is the bucket index, or -1 if capacity exceeds 16.
func (*RenderArena) attributeBucketIndex(capacity int) int {
	switch {
	case capacity <= 2:
		return 0
	case capacity <= 4:
		return 1
	case capacity <= 6:
		return 2
	case capacity <= 8:
		return 3
	case capacity <= 10:
		return 4
	case capacity <= 12:
		return 5
	case capacity <= 16:
		return 6
	default:
		return -1
	}
}

// growAttrBucket doubles the capacity of the attribute slab at the given index.
//
// Takes index (int) which specifies the attribute bucket to grow.
func (a *RenderArena) growAttrBucket(index int) {
	growSlabBucket(&a.attributeSlabs[index], func(c int) []HTMLAttribute { return make([]HTMLAttribute, 0, c) })
}

// attributeWriterBucketIndex returns the bucket index for the given capacity.
//
// Takes capacity (int) which specifies the required writer capacity.
//
// Returns int which is the bucket index, or -1 if the capacity is too large.
func (*RenderArena) attributeWriterBucketIndex(capacity int) int {
	switch {
	case capacity <= 2:
		return 0
	case capacity <= 4:
		return 1
	case capacity <= 8:
		return 2
	case capacity <= 10:
		return 3
	case capacity <= 12:
		return 4
	case capacity <= 16:
		return 5
	default:
		return -1
	}
}

// growAttrWriterBucket doubles the capacity of the attribute writer bucket at
// the given index.
//
// Takes index (int) which specifies the bucket index to grow.
func (a *RenderArena) growAttrWriterBucket(index int) {
	growSlabBucket(&a.attributeWriterSlabs[index], func(c int) []*DirectWriter { return make([]*DirectWriter, 0, c) })
}

// growDirectWriters doubles the capacity of the direct writers slice.
func (a *RenderArena) growDirectWriters() {
	newSize := len(a.directWriters) * 2
	newDWs := make([]DirectWriter, newSize)
	copy(newDWs, a.directWriters)
	a.directWriters = newDWs
}

// growByteBufs doubles the capacity of the byte buffer pool.
func (a *RenderArena) growByteBufs() {
	newSize := len(a.byteBufs) * 2
	newBufs := make([][]byte, newSize)
	copy(newBufs, a.byteBufs)
	for i := len(a.byteBufs); i < newSize; i++ {
		newBufs[i] = make([]byte, 0, defaultByteBufCap)
	}
	a.byteBufs = newBufs
}

// growAnnotations doubles the capacity of the annotations slice.
func (a *RenderArena) growAnnotations() {
	newSize := len(a.annotations) * 2
	newAnnots := make([]RuntimeAnnotation, newSize)
	copy(newAnnots, a.annotations)
	a.annotations = newAnnots
}

// rootNodesBucketIndex returns the bucket index for a given capacity.
//
// Takes capacity (int) which specifies the required node capacity.
//
// Returns int which is the bucket index, or -1 if capacity exceeds 128.
func (*RenderArena) rootNodesBucketIndex(capacity int) int {
	switch {
	case capacity <= 1:
		return 0
	case capacity <= 2:
		return 1
	case capacity <= 4:
		return 2
	case capacity <= 6:
		return 3
	case capacity <= 8:
		return 4
	case capacity <= 10:
		return 5
	case capacity <= 12:
		return 6
	case capacity <= 16:
		return 7
	case capacity <= 24:
		return 8
	case capacity <= 32:
		return 9
	case capacity <= 48:
		return 10
	case capacity <= 64:
		return 11
	case capacity <= 96:
		return 12
	case capacity <= 128:
		return 13
	default:
		return -1
	}
}

//revive:enable:add-constant

// growRootNodesBucket doubles the capacity of the root nodes bucket at the
// given index.
//
// Takes index (int) which specifies the bucket index to grow.
func (a *RenderArena) growRootNodesBucket(index int) {
	growSlabBucket(&a.rootNodesSlabs[index], func(c int) []*TemplateNode { return make([]*TemplateNode, 0, c) })
}

// resetNodes clears all nodes in the arena and resets the index to zero.
func (a *RenderArena) resetNodes() {
	for i := range a.nodeIndex {
		a.nodes[i] = TemplateNode{}
	}
	a.nodeIndex = 0

	if len(a.nodes) > maxNodeCount {
		a.nodes = make([]TemplateNode, initialNodeCount)
	}
}

// resetChildSlabs clears and resets all child slab buckets in the arena.
func (a *RenderArena) resetChildSlabs() {
	for i := range a.childSlabs {
		resetSlabBucket(&a.childSlabs[i], initialChildCounts[i], func(c int) []*TemplateNode { return make([]*TemplateNode, 0, c) })
	}
}

// resetAttrSlabs clears all attribute slabs and returns them to initial state.
func (a *RenderArena) resetAttrSlabs() {
	for i := range a.attributeSlabs {
		resetSlabBucket(&a.attributeSlabs[i], initialAttrCounts[i], func(c int) []HTMLAttribute { return make([]HTMLAttribute, 0, c) })
	}
}

// resetAttrWriterSlabs clears all attribute writer slabs and shrinks oversized
// buckets back to their initial capacity.
func (a *RenderArena) resetAttrWriterSlabs() {
	for i := range a.attributeWriterSlabs {
		resetSlabBucket(&a.attributeWriterSlabs[i], initialWriterCounts[i], func(c int) []*DirectWriter { return make([]*DirectWriter, 0, c) })
	}
}

// resetDirectWriters clears all direct writers and prepares them for reuse.
func (a *RenderArena) resetDirectWriters() {
	for i := range a.directWriterIndex {
		a.directWriters[i].resetForArena()
	}
	a.directWriterIndex = 0

	if len(a.directWriters) > maxDirectWriters {
		a.directWriters = make([]DirectWriter, initialDirectWriters)
	}
}

// resetByteBufs clears all byte buffers and resets the index for reuse.
func (a *RenderArena) resetByteBufs() {
	for i := range a.byteBufferIndex {
		if cap(a.byteBufs[i]) > maxByteBufCapacity {
			a.byteBufs[i] = make([]byte, 0, defaultByteBufCap)
		} else {
			a.byteBufs[i] = a.byteBufs[i][:0]
		}
	}
	a.byteBufferIndex = 0

	if len(a.byteBufs) > maxByteBufs {
		a.byteBufs = make([][]byte, initialByteBufs)
		for i := range a.byteBufs {
			a.byteBufs[i] = make([]byte, 0, defaultByteBufCap)
		}
	}
}

// resetAnnotations clears all annotations and resets the index to zero.
func (a *RenderArena) resetAnnotations() {
	for i := range a.annotationIndex {
		a.annotations[i] = RuntimeAnnotation{}
	}
	a.annotationIndex = 0

	if len(a.annotations) > maxAnnotations {
		a.annotations = make([]RuntimeAnnotation, initialAnnotations)
	}
}

// resetAST clears the AST state and marks it as unused.
func (a *RenderArena) resetAST() {
	a.ast = TemplateAST{}
	a.astUsed = false
}

// resetRootNodesSlabs clears all root node slabs and shrinks oversized buckets.
func (a *RenderArena) resetRootNodesSlabs() {
	for i := range a.rootNodesSlabs {
		resetSlabBucket(&a.rootNodesSlabs[i], initialRootNodesCounts[i], func(c int) []*TemplateNode { return make([]*TemplateNode, 0, c) })
	}
}

// GetArena retrieves a RenderArena from the pool.
//
// Returns *RenderArena which is a reusable arena, either from the pool or
// newly created.
func GetArena() *RenderArena {
	arena, ok := arenaPool.Get().(*RenderArena)
	if !ok {
		return newRenderArena()
	}
	return arena
}

// PutArena returns a RenderArena to the pool after resetting.
//
// Takes arena (*RenderArena) which is the arena to return to the pool.
func PutArena(arena *RenderArena) {
	if arena == nil {
		return
	}
	arena.Reset()
	arenaPool.Put(arena)
}

// ResetArenaPool clears the arena pool for test isolation.
func ResetArenaPool() {
	arenaPool = sync.Pool{
		New: func() any {
			return newRenderArena()
		},
	}
}

// newRenderArena creates a new arena with default initial sizes.
//
// Returns *RenderArena which is a fully initialised arena ready for use.
func newRenderArena() *RenderArena {
	a := &RenderArena{
		nodes:         make([]TemplateNode, initialNodeCount),
		directWriters: make([]DirectWriter, initialDirectWriters),
		byteBufs:      make([][]byte, initialByteBufs),
		annotations:   make([]RuntimeAnnotation, initialAnnotations),
	}

	for i := range a.byteBufs {
		a.byteBufs[i] = make([]byte, 0, defaultByteBufCap)
	}

	for i, bucketCap := range childBucketCaps {
		a.childSlabs[i] = newSlabBucket(initialChildCounts[i], bucketCap, func(c int) []*TemplateNode { return make([]*TemplateNode, 0, c) })
	}

	for i, bucketCap := range attributeBucketCapacities {
		a.attributeSlabs[i] = newSlabBucket(initialAttrCounts[i], bucketCap, func(c int) []HTMLAttribute { return make([]HTMLAttribute, 0, c) })
	}

	for i, bucketCap := range attributeWriterBucketCapacities {
		a.attributeWriterSlabs[i] = newSlabBucket(initialWriterCounts[i], bucketCap, func(c int) []*DirectWriter { return make([]*DirectWriter, 0, c) })
	}

	for i, bucketCap := range rootNodesBucketCaps {
		a.rootNodesSlabs[i] = newSlabBucket(initialRootNodesCounts[i], bucketCap, func(c int) []*TemplateNode { return make([]*TemplateNode, 0, c) })
	}

	return a
}

// newSlabBucket creates a new slab bucket with pre-allocated slices.
//
// Takes count (int) which specifies the number of slices to create.
// Takes sliceCap (int) which sets the initial capacity of each slice.
// Takes makeSlice (func(int) T) which creates a new slice of the
// given capacity.
//
// Returns slabBucket[T] which contains the allocated backing slices.
func newSlabBucket[T any](count, sliceCap int, makeSlice func(int) T) slabBucket[T] {
	b := slabBucket[T]{
		backing: make([]T, count),
		cap:     sliceCap,
	}
	for i := range b.backing {
		b.backing[i] = makeSlice(sliceCap)
	}
	return b
}

// getSliceFromBucket retrieves the next available slot from a bucket.
//
// Takes bucket (*slabBucket[T]) which holds the backing storage.
// Takes grow (func(int)) which expands the bucket when full.
// Takes bucketIndex (int) which identifies the bucket to grow.
// Takes reset (func(*T)) which clears the slot before returning.
//
// Returns *T which is a pointer to the allocated slot.
// Returns T which is a copy of the reset value.
func getSliceFromBucket[T any](
	bucket *slabBucket[T],
	grow func(int),
	bucketIndex int,
	reset func(*T),
) (*T, T) {
	if bucket.used >= len(bucket.backing) {
		grow(bucketIndex)
	}

	backing := &bucket.backing[bucket.used]
	bucket.used++
	reset(backing)
	return backing, *backing
}

// growSlabBucket doubles the capacity of a slab bucket's backing store.
//
// Takes bucket (*slabBucket[T]) which is the bucket to grow.
// Takes makeSlice (func(int) T) which creates new slices with the given
// capacity.
func growSlabBucket[T any](bucket *slabBucket[T], makeSlice func(int) T) {
	newCount := len(bucket.backing) * 2
	if newCount == 0 {
		newCount = 2
	}
	newBacking := make([]T, newCount)
	copy(newBacking, bucket.backing)
	for i := len(bucket.backing); i < newCount; i++ {
		newBacking[i] = makeSlice(bucket.cap)
	}
	bucket.backing = newBacking
}

// resetSlabBucket clears all used slices in the bucket and resets it for reuse.
//
// When the backing array has grown beyond the initial count multiplied by the
// maximum bucket multiplier, it reallocates the backing array to the initial
// size.
//
// Takes bucket (*slabBucket[T]) which is the bucket to reset.
// Takes initialCount (int) which is the target size for the backing array.
// Takes makeSlice (func(int) T) which creates new slices of the given capacity.
func resetSlabBucket[T ~[]E, E any](bucket *slabBucket[T], initialCount int, makeSlice func(int) T) {
	for j := range bucket.used {
		clear(bucket.backing[j])
	}
	bucket.used = 0

	if len(bucket.backing) > initialCount*maxBucketMultiplier {
		bucket.backing = make([]T, initialCount)
		for j := range bucket.backing {
			bucket.backing[j] = makeSlice(bucket.cap)
		}
	}
}
