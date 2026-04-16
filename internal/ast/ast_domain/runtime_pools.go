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

import "sync"

//nolint:revive // pool thresholds

var (
	// runtimeAnnotationPool is a sync.Pool that provides RuntimeAnnotation values.
	// It eliminates allocations for CSRF annotations after warmup.
	runtimeAnnotationPool = sync.Pool{
		New: func() any { return new(RuntimeAnnotation) },
	}

	// templateASTPool stores reusable TemplateAST instances to eliminate root AST
	// allocations after warmup.
	templateASTPool = sync.Pool{
		New: func() any { return new(TemplateAST) },
	}

	// rootNodesPool1 reuses []*TemplateNode slices (cap 1) to reduce allocation pressure.
	rootNodesPool1 = sync.Pool{New: func() any { return make([]*TemplateNode, 0, 1) }}

	// rootNodesPool2 reuses []*TemplateNode slices (cap 2) to reduce allocation pressure.
	rootNodesPool2 = sync.Pool{New: func() any { return make([]*TemplateNode, 0, 2) }}

	// rootNodesPool4 reuses []*TemplateNode slices (cap 4) to reduce allocation pressure.
	rootNodesPool4 = sync.Pool{New: func() any { return make([]*TemplateNode, 0, 4) }}

	// rootNodesPool6 reuses []*TemplateNode slices (cap 6) to reduce allocation pressure.
	rootNodesPool6 = sync.Pool{New: func() any { return make([]*TemplateNode, 0, 6) }}

	// rootNodesPool8 reuses []*TemplateNode slices (cap 8) to reduce allocation pressure.
	rootNodesPool8 = sync.Pool{New: func() any { return make([]*TemplateNode, 0, 8) }}

	// rootNodesPool10 reuses []*TemplateNode slices (cap 10) to reduce allocation pressure.
	rootNodesPool10 = sync.Pool{New: func() any { return make([]*TemplateNode, 0, 10) }}

	// rootNodesPool12 reuses []*TemplateNode slices (cap 12) to reduce allocation pressure.
	rootNodesPool12 = sync.Pool{New: func() any { return make([]*TemplateNode, 0, 12) }}

	// rootNodesPool16 reuses []*TemplateNode slices (cap 16) to reduce allocation pressure.
	rootNodesPool16 = sync.Pool{New: func() any { return make([]*TemplateNode, 0, 16) }}

	// rootNodesPool24 reuses []*TemplateNode slices (cap 24) to reduce allocation pressure.
	rootNodesPool24 = sync.Pool{New: func() any { return make([]*TemplateNode, 0, 24) }}

	// rootNodesPool32 reuses []*TemplateNode slices (cap 32) to reduce allocation pressure.
	rootNodesPool32 = sync.Pool{New: func() any { return make([]*TemplateNode, 0, 32) }}

	// rootNodesPool48 reuses []*TemplateNode slices (cap 48) to reduce allocation pressure.
	rootNodesPool48 = sync.Pool{New: func() any { return make([]*TemplateNode, 0, 48) }}

	// rootNodesPool64 reuses []*TemplateNode slices (cap 64) to reduce allocation pressure.
	rootNodesPool64 = sync.Pool{New: func() any { return make([]*TemplateNode, 0, 64) }}

	// rootNodesPool96 reuses []*TemplateNode slices (cap 96) to reduce allocation pressure.
	rootNodesPool96 = sync.Pool{New: func() any { return make([]*TemplateNode, 0, 96) }}

	// rootNodesPool128 reuses []*TemplateNode slices (cap 128) to reduce allocation pressure.
	rootNodesPool128 = sync.Pool{New: func() any { return make([]*TemplateNode, 0, 128) }}
)

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
	ra, ok := runtimeAnnotationPool.Get().(*RuntimeAnnotation)
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
	runtimeAnnotationPool.Put(ra)
}

// ResetRuntimeAnnotationPool clears and resets the pool for test isolation.
func ResetRuntimeAnnotationPool() {
	runtimeAnnotationPool = sync.Pool{
		New: func() any { return new(RuntimeAnnotation) },
	}
}

// GetTemplateAST retrieves a TemplateAST from the pool.
// The returned AST has all fields set to zero and is marked as pooled.
//
// Returns *TemplateAST which is a ready-to-use AST from the pool.
func GetTemplateAST() *TemplateAST {
	ast, ok := templateASTPool.Get().(*TemplateAST)
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
	templateASTPool.Put(ast)
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
		return rootNodesPool1.Get().([]*TemplateNode)[:0]
	case capacity <= 2:
		return rootNodesPool2.Get().([]*TemplateNode)[:0]
	case capacity <= 4:
		return rootNodesPool4.Get().([]*TemplateNode)[:0]
	case capacity <= 6:
		return rootNodesPool6.Get().([]*TemplateNode)[:0]
	case capacity <= 8:
		return rootNodesPool8.Get().([]*TemplateNode)[:0]
	case capacity <= 10:
		return rootNodesPool10.Get().([]*TemplateNode)[:0]
	case capacity <= 12:
		return rootNodesPool12.Get().([]*TemplateNode)[:0]
	case capacity <= 16:
		return rootNodesPool16.Get().([]*TemplateNode)[:0]
	case capacity <= 24:
		return rootNodesPool24.Get().([]*TemplateNode)[:0]
	case capacity <= 32:
		return rootNodesPool32.Get().([]*TemplateNode)[:0]
	case capacity <= 48:
		return rootNodesPool48.Get().([]*TemplateNode)[:0]
	case capacity <= 64:
		return rootNodesPool64.Get().([]*TemplateNode)[:0]
	case capacity <= 96:
		return rootNodesPool96.Get().([]*TemplateNode)[:0]
	case capacity <= 128:
		return rootNodesPool128.Get().([]*TemplateNode)[:0]
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
		rootNodesPool1.Put(s[:0])
	case 2:
		rootNodesPool2.Put(s[:0])
	case 4:
		rootNodesPool4.Put(s[:0])
	case 6:
		rootNodesPool6.Put(s[:0])
	case 8:
		rootNodesPool8.Put(s[:0])
	case 10:
		rootNodesPool10.Put(s[:0])
	case 12:
		rootNodesPool12.Put(s[:0])
	case 16:
		rootNodesPool16.Put(s[:0])
	case 24:
		rootNodesPool24.Put(s[:0])
	case 32:
		rootNodesPool32.Put(s[:0])
	case 48:
		rootNodesPool48.Put(s[:0])
	case 64:
		rootNodesPool64.Put(s[:0])
	case 96:
		rootNodesPool96.Put(s[:0])
	case 128:
		rootNodesPool128.Put(s[:0])
	}
}

// ResetTemplateASTPool clears all TemplateAST pools to ensure test isolation.
func ResetTemplateASTPool() {
	templateASTPool = sync.Pool{New: func() any { return new(TemplateAST) }}
	rootNodesPool1 = sync.Pool{New: func() any { return make([]*TemplateNode, 0, 1) }}
	rootNodesPool2 = sync.Pool{New: func() any { return make([]*TemplateNode, 0, 2) }}
	rootNodesPool4 = sync.Pool{New: func() any { return make([]*TemplateNode, 0, 4) }}
	rootNodesPool6 = sync.Pool{New: func() any { return make([]*TemplateNode, 0, 6) }}
	rootNodesPool8 = sync.Pool{New: func() any { return make([]*TemplateNode, 0, 8) }}
	rootNodesPool10 = sync.Pool{New: func() any { return make([]*TemplateNode, 0, 10) }}
	rootNodesPool12 = sync.Pool{New: func() any { return make([]*TemplateNode, 0, 12) }}
	rootNodesPool16 = sync.Pool{New: func() any { return make([]*TemplateNode, 0, 16) }}
	rootNodesPool24 = sync.Pool{New: func() any { return make([]*TemplateNode, 0, 24) }}
	rootNodesPool32 = sync.Pool{New: func() any { return make([]*TemplateNode, 0, 32) }}
	rootNodesPool48 = sync.Pool{New: func() any { return make([]*TemplateNode, 0, 48) }}
	rootNodesPool64 = sync.Pool{New: func() any { return make([]*TemplateNode, 0, 64) }}
	rootNodesPool96 = sync.Pool{New: func() any { return make([]*TemplateNode, 0, 96) }}
	rootNodesPool128 = sync.Pool{New: func() any { return make([]*TemplateNode, 0, 128) }}
}
