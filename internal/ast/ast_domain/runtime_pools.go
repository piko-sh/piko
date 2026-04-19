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

//revive:disable:add-constant

package ast_domain

// Provides object pooling for runtime annotation instances to reduce
// allocations during template rendering. Implements sync.Pool-based recycling
// for RuntimeAnnotation objects used in request handling and CSRF protection.
//
// Each pool is held behind an atomic.Pointer[sync.Pool] so the Reset* helpers
// can swap in a fresh pool without racing concurrent Get/Put callers. A direct
// `pool = sync.Pool{...}` reassignment performs a non-atomic struct copy that
// tears reads of the (local, localSize) pair under the race detector and can
// crash with a checkptr violation in sync.indexLocal.

import (
	"sync"
	"sync/atomic"
)

//nolint:revive // pool thresholds

var (
	// runtimeAnnotationPool provides RuntimeAnnotation values, eliminating
	// allocations for CSRF annotations after warmup.
	runtimeAnnotationPool atomic.Pointer[sync.Pool]

	// templateASTPool stores reusable TemplateAST instances to eliminate root
	// AST allocations after warmup.
	templateASTPool atomic.Pointer[sync.Pool]

	// rootNodesPool1 reuses []*TemplateNode slices (cap 1).
	rootNodesPool1 atomic.Pointer[sync.Pool]

	// rootNodesPool2 reuses []*TemplateNode slices (cap 2).
	rootNodesPool2 atomic.Pointer[sync.Pool]

	// rootNodesPool4 reuses []*TemplateNode slices (cap 4).
	rootNodesPool4 atomic.Pointer[sync.Pool]

	// rootNodesPool6 reuses []*TemplateNode slices (cap 6).
	rootNodesPool6 atomic.Pointer[sync.Pool]

	// rootNodesPool8 reuses []*TemplateNode slices (cap 8).
	rootNodesPool8 atomic.Pointer[sync.Pool]

	// rootNodesPool10 reuses []*TemplateNode slices (cap 10).
	rootNodesPool10 atomic.Pointer[sync.Pool]

	// rootNodesPool12 reuses []*TemplateNode slices (cap 12).
	rootNodesPool12 atomic.Pointer[sync.Pool]

	// rootNodesPool16 reuses []*TemplateNode slices (cap 16).
	rootNodesPool16 atomic.Pointer[sync.Pool]

	// rootNodesPool24 reuses []*TemplateNode slices (cap 24).
	rootNodesPool24 atomic.Pointer[sync.Pool]

	// rootNodesPool32 reuses []*TemplateNode slices (cap 32).
	rootNodesPool32 atomic.Pointer[sync.Pool]

	// rootNodesPool48 reuses []*TemplateNode slices (cap 48).
	rootNodesPool48 atomic.Pointer[sync.Pool]

	// rootNodesPool64 reuses []*TemplateNode slices (cap 64).
	rootNodesPool64 atomic.Pointer[sync.Pool]

	// rootNodesPool96 reuses []*TemplateNode slices (cap 96).
	rootNodesPool96 atomic.Pointer[sync.Pool]

	// rootNodesPool128 reuses []*TemplateNode slices (cap 128).
	rootNodesPool128 atomic.Pointer[sync.Pool]
)

func init() {
	runtimeAnnotationPool.Store(newRuntimeAnnotationPool())
	templateASTPool.Store(newTemplateASTPool())
	rootNodesPool1.Store(newRootNodesPool(1))
	rootNodesPool2.Store(newRootNodesPool(2))
	rootNodesPool4.Store(newRootNodesPool(4))
	rootNodesPool6.Store(newRootNodesPool(6))
	rootNodesPool8.Store(newRootNodesPool(8))
	rootNodesPool10.Store(newRootNodesPool(10))
	rootNodesPool12.Store(newRootNodesPool(12))
	rootNodesPool16.Store(newRootNodesPool(16))
	rootNodesPool24.Store(newRootNodesPool(24))
	rootNodesPool32.Store(newRootNodesPool(32))
	rootNodesPool48.Store(newRootNodesPool(48))
	rootNodesPool64.Store(newRootNodesPool(64))
	rootNodesPool96.Store(newRootNodesPool(96))
	rootNodesPool128.Store(newRootNodesPool(128))
}

// newRuntimeAnnotationPool builds a fresh sync.Pool whose New func returns
// a zero-valued RuntimeAnnotation. Used by init and ResetRuntimeAnnotationPool.
//
// Returns *sync.Pool which is the freshly constructed pool.
func newRuntimeAnnotationPool() *sync.Pool {
	return &sync.Pool{New: func() any { return new(RuntimeAnnotation) }}
}

// newTemplateASTPool builds a fresh sync.Pool whose New func returns a
// zero-valued TemplateAST. Used by init and ResetTemplateASTPool.
//
// Returns *sync.Pool which is the freshly constructed pool.
func newTemplateASTPool() *sync.Pool {
	return &sync.Pool{New: func() any { return new(TemplateAST) }}
}

// newRootNodesPool builds a fresh sync.Pool whose New func returns an empty
// []*TemplateNode of the given capacity. Used by init and
// ResetTemplateASTPool to populate one bucket of the size-class ladder.
//
// Takes capacity (int) which is the slice capacity for the bucket.
//
// Returns *sync.Pool which is the freshly constructed pool.
func newRootNodesPool(capacity int) *sync.Pool {
	return &sync.Pool{New: func() any { return make([]*TemplateNode, 0, capacity) }}
}

// Reset clears all fields and returns pooled slices for reuse.
func (ast *TemplateAST) Reset() {
	if ast == nil {
		return
	}

	if ast.isPooled && cap(ast.RootNodes) > 0 {
		PutRootNodesSlice(ast.RootNodes)
	}

	ast.SourcePath = nil
	ast.ExpiresAtUnixNano = nil
	ast.Metadata = nil
	ast.RootNodes = nil
	ast.Diagnostics = nil
	ast.queryContext = nil
	ast.SourceSize = 0
	ast.Tidied = false
	ast.isPooled = false
}

// GetRuntimeAnnotation fetches a RuntimeAnnotation from the pool.
// The returned annotation has all fields set to their zero values.
//
// Returns *RuntimeAnnotation which is ready for use.
func GetRuntimeAnnotation() *RuntimeAnnotation {
	ra, ok := runtimeAnnotationPool.Load().Get().(*RuntimeAnnotation)
	if !ok {
		return new(RuntimeAnnotation)
	}
	return ra
}

// PutRuntimeAnnotation returns a RuntimeAnnotation to the pool.
// Resets all fields to zero values before returning.
//
// Takes ra (*RuntimeAnnotation) which is the annotation to return to the pool.
func PutRuntimeAnnotation(ra *RuntimeAnnotation) {
	if ra == nil {
		return
	}
	*ra = RuntimeAnnotation{}
	runtimeAnnotationPool.Load().Put(ra)
}

// ResetRuntimeAnnotationPool atomically swaps in a fresh RuntimeAnnotation
// pool for test isolation. Safe to call concurrently with Get/Put.
func ResetRuntimeAnnotationPool() {
	runtimeAnnotationPool.Store(newRuntimeAnnotationPool())
}

// GetTemplateAST retrieves a TemplateAST from the pool.
// The returned AST has all fields set to zero and is marked as pooled.
//
// Returns *TemplateAST which is a ready-to-use AST from the pool.
func GetTemplateAST() *TemplateAST {
	ast, ok := templateASTPool.Load().Get().(*TemplateAST)
	if !ok {
		ast = new(TemplateAST)
	}
	ast.isPooled = true
	return ast
}

// PutTemplateAST returns a TemplateAST to the pool for reuse.
//
// When ast is nil, returns straight away without action.
//
// Resets all fields and returns nested slices to their pools before storing.
//
// Takes ast (*TemplateAST) which is the template AST to return to the pool.
func PutTemplateAST(ast *TemplateAST) {
	if ast == nil {
		return
	}
	ast.Reset()
	templateASTPool.Load().Put(ast)
}

// GetRootNodesSlice gets a slice from a pool for storing root nodes.
//
// The capacity is rounded up to the nearest bucket size (1, 2, 4, 6, 8, 10,
// 12, 16, 24, 32, 48, 64, 96, 128). For capacity over 128, a new slice is
// created instead.
//
// Takes capacity (int) which is the minimum slice capacity needed.
//
// Returns []*TemplateNode which is a pooled slice with zero length, or nil if
// capacity is zero or less.
func GetRootNodesSlice(capacity int) []*TemplateNode {
	if capacity <= 0 {
		return nil
	}
	switch {
	case capacity <= 1:
		return rootNodesPool1.Load().Get().([]*TemplateNode)[:0]
	case capacity <= 2:
		return rootNodesPool2.Load().Get().([]*TemplateNode)[:0]
	case capacity <= 4:
		return rootNodesPool4.Load().Get().([]*TemplateNode)[:0]
	case capacity <= 6:
		return rootNodesPool6.Load().Get().([]*TemplateNode)[:0]
	case capacity <= 8:
		return rootNodesPool8.Load().Get().([]*TemplateNode)[:0]
	case capacity <= 10:
		return rootNodesPool10.Load().Get().([]*TemplateNode)[:0]
	case capacity <= 12:
		return rootNodesPool12.Load().Get().([]*TemplateNode)[:0]
	case capacity <= 16:
		return rootNodesPool16.Load().Get().([]*TemplateNode)[:0]
	case capacity <= 24:
		return rootNodesPool24.Load().Get().([]*TemplateNode)[:0]
	case capacity <= 32:
		return rootNodesPool32.Load().Get().([]*TemplateNode)[:0]
	case capacity <= 48:
		return rootNodesPool48.Load().Get().([]*TemplateNode)[:0]
	case capacity <= 64:
		return rootNodesPool64.Load().Get().([]*TemplateNode)[:0]
	case capacity <= 96:
		return rootNodesPool96.Load().Get().([]*TemplateNode)[:0]
	case capacity <= 128:
		return rootNodesPool128.Load().Get().([]*TemplateNode)[:0]
	default:
		return make([]*TemplateNode, 0, capacity)
	}
}

// PutRootNodesSlice returns a RootNodes slice to its pool for reuse.
//
// When s is nil, returns without doing anything.
//
// Takes s ([]*TemplateNode) which is the slice to return to the pool.
func PutRootNodesSlice(s []*TemplateNode) {
	if s == nil {
		return
	}
	clear(s)
	c := cap(s)
	switch c {
	case 1:
		rootNodesPool1.Load().Put(s[:0])
	case 2:
		rootNodesPool2.Load().Put(s[:0])
	case 4:
		rootNodesPool4.Load().Put(s[:0])
	case 6:
		rootNodesPool6.Load().Put(s[:0])
	case 8:
		rootNodesPool8.Load().Put(s[:0])
	case 10:
		rootNodesPool10.Load().Put(s[:0])
	case 12:
		rootNodesPool12.Load().Put(s[:0])
	case 16:
		rootNodesPool16.Load().Put(s[:0])
	case 24:
		rootNodesPool24.Load().Put(s[:0])
	case 32:
		rootNodesPool32.Load().Put(s[:0])
	case 48:
		rootNodesPool48.Load().Put(s[:0])
	case 64:
		rootNodesPool64.Load().Put(s[:0])
	case 96:
		rootNodesPool96.Load().Put(s[:0])
	case 128:
		rootNodesPool128.Load().Put(s[:0])
	}
}

// ResetTemplateASTPool atomically swaps in fresh TemplateAST and rootNodes
// pools for test isolation. Safe to call concurrently with Get/Put.
func ResetTemplateASTPool() {
	templateASTPool.Store(newTemplateASTPool())
	rootNodesPool1.Store(newRootNodesPool(1))
	rootNodesPool2.Store(newRootNodesPool(2))
	rootNodesPool4.Store(newRootNodesPool(4))
	rootNodesPool6.Store(newRootNodesPool(6))
	rootNodesPool8.Store(newRootNodesPool(8))
	rootNodesPool10.Store(newRootNodesPool(10))
	rootNodesPool12.Store(newRootNodesPool(12))
	rootNodesPool16.Store(newRootNodesPool(16))
	rootNodesPool24.Store(newRootNodesPool(24))
	rootNodesPool32.Store(newRootNodesPool(32))
	rootNodesPool48.Store(newRootNodesPool(48))
	rootNodesPool64.Store(newRootNodesPool(64))
	rootNodesPool96.Store(newRootNodesPool(96))
	rootNodesPool128.Store(newRootNodesPool(128))
}
