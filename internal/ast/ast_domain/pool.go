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

// PutTree returns an AST tree to the pool by releasing its arena.
//
// All nodes allocated from the arena are reset in a single operation when
// the arena is returned to its pool.
//
// Takes ast (*TemplateAST) which is the tree to return to the pool.
func PutTree(ast *TemplateAST) {
	if ast == nil {
		return
	}

	if ast.arena != nil {
		PutArena(ast.arena)
		ast.arena = nil
	}
}

// ResetAllPools clears all sync.Pool instances in the package.
//
// Use with t.Cleanup(ResetAllPools) in tests to ensure pool isolation.
func ResetAllPools() {
	ResetExpressionParserPool()
	resetIteratorFramePool()
	resetASTEventPool()
	ResetDirectWriterPool()
	ResetRuntimeAnnotationPool()
	ResetTemplateASTPool()
	ResetArenaPool()
	ResetByteBufPool()
}
