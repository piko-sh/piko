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

package driven_code_emitter_go_literal

import (
	"context"
	goast "go/ast"
	"sync"

	"piko.sh/piko/internal/logger/logger_domain"
)

var (
	astBuilderPool = sync.Pool{
		New: func() any {
			return &astBuilder{}
		},
	}

	staticEmitterPool = sync.Pool{
		New: func() any {
			return &staticEmitter{}
		},
	}

	nodeEmitterPool = sync.Pool{
		New: func() any {
			return &nodeEmitter{}
		},
	}

	expressionEmitterPool = sync.Pool{
		New: func() any {
			return &expressionEmitter{}
		},
	}

	binaryOpEmitterPool = sync.Pool{
		New: func() any {
			return &binaryOpEmitter{}
		},
	}

	attributeEmitterPool = sync.Pool{
		New: func() any {
			return &attributeEmitter{}
		},
	}

	ifEmitterPool = sync.Pool{
		New: func() any {
			return &ifEmitter{}
		},
	}

	forEmitterPool = sync.Pool{
		New: func() any {
			return &forEmitter{}
		},
	}
)

// getAstBuilder retrieves an astBuilder from the pool and sets up all its
// sub-emitters.
//
// Takes emitter (*emitter) which provides the code generation context.
//
// Returns *astBuilder which is fully set up and ready for use.
func getAstBuilder(ctx context.Context, emitter *emitter) *astBuilder {
	b := retrieveAstBuilderFromPool(ctx)
	b.emitter = emitter

	initialiseStaticEmitter(b, emitter)
	initialiseExpressionEmitter(b, emitter)
	initialiseControlFlowEmitters(b, emitter)
	initialiseNodeEmitter(b, emitter)
	wireEmitterReferences(emitter, b)

	return b
}

// retrieveAstBuilderFromPool gets an astBuilder from the pool with type safety.
//
// Takes ctx (context.Context) which provides the base context for logging.
//
// Returns *astBuilder which is ready for use.
func retrieveAstBuilderFromPool(ctx context.Context) *astBuilder {
	if b, ok := astBuilderPool.Get().(*astBuilder); ok {
		return b
	}
	_, l := logger_domain.From(context.WithoutCancel(ctx), log)
	l.Error("astBuilderPool returned unexpected type, allocating new instance")
	return &astBuilder{}
}

// initialiseStaticEmitter sets up a static emitter from the pool and assigns
// it to the builder.
//
// Takes b (*astBuilder) which receives the set up static emitter.
// Takes emitter (*emitter) which provides the base emitter for the static
// emitter.
func initialiseStaticEmitter(b *astBuilder, emitter *emitter) {
	staticEmit, ok := staticEmitterPool.Get().(*staticEmitter)
	if !ok {
		_, l := logger_domain.From(context.Background(), log)
		l.Error("staticEmitterPool returned unexpected type, allocating new instance")
		staticEmit = &staticEmitter{}
	}
	staticEmit.emitter = emitter
	if staticEmit.staticNodeCache == nil {
		staticEmit.staticNodeCache = make(map[string]string)
	} else {
		clear(staticEmit.staticNodeCache)
	}
	if staticEmit.staticAttrCache == nil {
		staticEmit.staticAttrCache = make(map[string]string)
	} else {
		clear(staticEmit.staticAttrCache)
	}
	if staticEmit.allStaticVarDecls == nil {
		staticEmit.allStaticVarDecls = make(map[string]*goast.ValueSpec)
	} else {
		clear(staticEmit.allStaticVarDecls)
	}
	if staticEmit.staticAttrVarDecls == nil {
		staticEmit.staticAttrVarDecls = make(map[string]*goast.ValueSpec)
	} else {
		clear(staticEmit.staticAttrVarDecls)
	}
	staticEmit.initFunctionStatements = staticEmit.initFunctionStatements[:0]
	b.staticEmitter = staticEmit
}

// initialiseExpressionEmitter sets up the expression emitter and its
// dependencies.
//
// Takes b (*astBuilder) which receives the configured expression emitter.
// Takes emitter (*emitter) which provides the base emitter for expression
// handling.
func initialiseExpressionEmitter(b *astBuilder, emitter *emitter) {
	expressionEmit, ok := expressionEmitterPool.Get().(*expressionEmitter)
	if !ok {
		_, l := logger_domain.From(context.Background(), log)
		l.Error("expressionEmitterPool returned unexpected type, allocating new instance")
		expressionEmit = &expressionEmitter{}
	}
	expressionEmit.emitter = emitter

	initialiseBinaryOpEmitter(expressionEmit, emitter)
	expressionEmit.stringConv = newStringConverter()

	b.expressionEmitter = expressionEmit
}

// initialiseBinaryOpEmitter sets up a binary operation emitter within an
// expression emitter.
//
// Takes expressionEmit (*expressionEmitter) which receives the binary emitter.
// Takes emitter (*emitter) which provides the code emission context.
func initialiseBinaryOpEmitter(expressionEmit *expressionEmitter, emitter *emitter) {
	binaryEmit, ok := binaryOpEmitterPool.Get().(*binaryOpEmitter)
	if !ok {
		_, l := logger_domain.From(context.Background(), log)
		l.Error("binaryOpEmitterPool returned unexpected type, allocating new instance")
		binaryEmit = &binaryOpEmitter{}
	}
	binaryEmit.emitter = emitter
	binaryEmit.expressionEmitter = expressionEmit
	expressionEmit.binaryEmitter = binaryEmit
}

// initialiseControlFlowEmitters sets up the if and for statement emitters.
//
// Takes b (*astBuilder) which provides the AST building context.
// Takes emitter (*emitter) which receives the control flow emitter
// registrations.
func initialiseControlFlowEmitters(b *astBuilder, emitter *emitter) {
	initialiseIfEmitter(b, emitter)
	initialiseForEmitter(b, emitter)
}

// initialiseIfEmitter sets up the if emitter from a sync pool.
//
// Takes b (*astBuilder) which provides the builder to configure.
// Takes emitter (*emitter) which provides the base emitter instance.
func initialiseIfEmitter(b *astBuilder, emitter *emitter) {
	ifEmit, ok := ifEmitterPool.Get().(*ifEmitter)
	if !ok {
		_, l := logger_domain.From(context.Background(), log)
		l.Error("ifEmitterPool returned unexpected type, allocating new instance")
		ifEmit = &ifEmitter{}
	}
	ifEmit.emitter = emitter
	ifEmit.expressionEmitter = b.expressionEmitter
	ifEmit.astBuilder = b
	b.ifEmitter = ifEmit
}

// initialiseForEmitter sets up a forEmitter from the pool for the builder.
//
// Takes b (*astBuilder) which is the builder to set up.
// Takes emitter (*emitter) which is the emitter to link.
func initialiseForEmitter(b *astBuilder, emitter *emitter) {
	forEmit, ok := forEmitterPool.Get().(*forEmitter)
	if !ok {
		_, l := logger_domain.From(context.Background(), log)
		l.Error("forEmitterPool returned unexpected type, allocating new instance")
		forEmit = &forEmitter{}
	}
	forEmit.emitter = emitter
	forEmit.expressionEmitter = b.expressionEmitter
	forEmit.astBuilder = b
	b.forEmitter = forEmit
}

// initialiseNodeEmitter sets up a node emitter by getting one from a pool and
// setting its fields, then assigns it to the AST builder.
//
// Takes b (*astBuilder) which receives the configured node emitter.
// Takes emitter (*emitter) which provides the underlying emitter reference.
func initialiseNodeEmitter(b *astBuilder, emitter *emitter) {
	nodeEmit, ok := nodeEmitterPool.Get().(*nodeEmitter)
	if !ok {
		_, l := logger_domain.From(context.Background(), log)
		l.Error("nodeEmitterPool returned unexpected type, allocating new instance")
		nodeEmit = &nodeEmitter{}
	}
	nodeEmit.emitter = emitter
	nodeEmit.expressionEmitter = b.expressionEmitter
	nodeEmit.astBuilder = b

	initialiseAttributeEmitter(nodeEmit, emitter, b.expressionEmitter)
	b.nodeEmitter = nodeEmit
}

// initialiseAttributeEmitter sets up the attribute emitter for a node emitter.
//
// Takes nodeEmit (*nodeEmitter) which receives the configured attribute
// emitter.
// Takes emitter (*emitter) which provides the base emitter for attribute
// output.
// Takes expressionEmit (ExpressionEmitter) which handles expression rendering.
func initialiseAttributeEmitter(nodeEmit *nodeEmitter, emitter *emitter, expressionEmit ExpressionEmitter) {
	attributeEmit, ok := attributeEmitterPool.Get().(*attributeEmitter)
	if !ok {
		_, l := logger_domain.From(context.Background(), log)
		l.Error("attributeEmitterPool returned unexpected type, allocating new instance")
		attributeEmit = &attributeEmitter{}
	}
	attributeEmit.emitter = emitter
	attributeEmit.expressionEmitter = expressionEmit
	nodeEmit.attributeEmitter = attributeEmit
}

// wireEmitterReferences sets up two-way references between an emitter and an
// astBuilder.
//
// Takes emitter (*emitter) which receives the builder and static emitter.
// Takes b (*astBuilder) which provides the static emitter to link.
func wireEmitterReferences(emitter *emitter, b *astBuilder) {
	emitter.astBuilder = b
	if staticEmit, ok := b.staticEmitter.(*staticEmitter); ok {
		emitter.staticEmitter = staticEmit
	}
}

// putAstBuilder resets the given astBuilder and returns it along with all
// its sub-emitters to their respective pools.
//
// Takes b (*astBuilder) which is the builder to reset and return.
func putAstBuilder(b *astBuilder) {
	if b == nil {
		return
	}

	returnNodeEmitterToPool(b)
	returnControlFlowEmittersToPool(b)
	returnExpressionEmitterToPool(b)
	returnStaticEmitterToPool(b)
	resetAndReturnAstBuilder(b)
}

// returnNodeEmitterToPool returns the nodeEmitter and its attributeEmitter to
// their pools.
//
// Takes b (*astBuilder) which holds the nodeEmitter to return.
func returnNodeEmitterToPool(b *astBuilder) {
	if b.nodeEmitter == nil {
		return
	}

	if ne, ok := b.nodeEmitter.(*nodeEmitter); ok {
		returnAttributeEmitterToPool(ne)
		resetAndReturnNodeEmitter(ne)
	}
	b.nodeEmitter = nil
}

// returnAttributeEmitterToPool returns the attributeEmitter to its sync.Pool.
//
// Takes ne (*nodeEmitter) which holds the attributeEmitter to return.
func returnAttributeEmitterToPool(ne *nodeEmitter) {
	if ne.attributeEmitter == nil {
		return
	}

	if ae, ok := ne.attributeEmitter.(*attributeEmitter); ok {
		ae.emitter = nil
		ae.expressionEmitter = nil
		attributeEmitterPool.Put(ae)
	}
}

// resetAndReturnNodeEmitter clears a node emitter and returns it to the pool.
//
// Takes ne (*nodeEmitter) which is the emitter to reset and return.
func resetAndReturnNodeEmitter(ne *nodeEmitter) {
	ne.emitter = nil
	ne.expressionEmitter = nil
	ne.attributeEmitter = nil
	ne.astBuilder = nil
	nodeEmitterPool.Put(ne)
}

// returnControlFlowEmittersToPool returns for and if emitters to their pools.
//
// Takes b (*astBuilder) which holds the emitters to return.
func returnControlFlowEmittersToPool(b *astBuilder) {
	returnForEmitterToPool(b)
	returnIfEmitterToPool(b)
}

// returnForEmitterToPool returns a forEmitter to the sync pool.
//
// Takes b (*astBuilder) which holds the forEmitter to return.
func returnForEmitterToPool(b *astBuilder) {
	if b.forEmitter == nil {
		return
	}

	if fe, ok := b.forEmitter.(*forEmitter); ok {
		fe.emitter = nil
		fe.expressionEmitter = nil
		fe.astBuilder = nil
		forEmitterPool.Put(fe)
	}
	b.forEmitter = nil
}

// returnIfEmitterToPool returns the ifEmitter to its sync pool.
//
// Takes b (*astBuilder) which owns the ifEmitter to return.
func returnIfEmitterToPool(b *astBuilder) {
	if b.ifEmitter == nil {
		return
	}

	if ie, ok := b.ifEmitter.(*ifEmitter); ok {
		ie.emitter = nil
		ie.expressionEmitter = nil
		ie.astBuilder = nil
		ifEmitterPool.Put(ie)
	}
	b.ifEmitter = nil
}

// returnExpressionEmitterToPool returns the expression emitter and its binary
// operation emitter to their object pools for reuse.
//
// Takes b (*astBuilder) which holds the expression emitter to return.
func returnExpressionEmitterToPool(b *astBuilder) {
	if b.expressionEmitter == nil {
		return
	}

	if ee, ok := b.expressionEmitter.(*expressionEmitter); ok {
		returnBinaryOpEmitterToPool(ee)
		resetAndReturnExpressionEmitter(ee)
	}
	b.expressionEmitter = nil
}

// returnBinaryOpEmitterToPool returns a binary operation emitter to its pool.
//
// Takes ee (*expressionEmitter) which holds the binary emitter to return.
func returnBinaryOpEmitterToPool(ee *expressionEmitter) {
	if ee.binaryEmitter == nil {
		return
	}

	if be, ok := ee.binaryEmitter.(*binaryOpEmitter); ok {
		be.emitter = nil
		be.expressionEmitter = nil
		binaryOpEmitterPool.Put(be)
	}
}

// resetAndReturnExpressionEmitter clears and returns an expressionEmitter to
// its pool.
//
// Takes ee (*expressionEmitter) which is the emitter to clear and return.
func resetAndReturnExpressionEmitter(ee *expressionEmitter) {
	ee.emitter = nil
	ee.binaryEmitter = nil
	ee.stringConv = nil
	expressionEmitterPool.Put(ee)
}

// returnStaticEmitterToPool returns a static emitter to its pool after
// clearing its state.
//
// Takes b (*astBuilder) which holds the static emitter to return.
func returnStaticEmitterToPool(b *astBuilder) {
	if b.staticEmitter == nil {
		return
	}

	if se, ok := b.staticEmitter.(*staticEmitter); ok {
		se.emitter = nil

		staticEmitterPool.Put(se)
	}
	b.staticEmitter = nil
}

// resetAndReturnAstBuilder clears all fields of the builder and returns it to
// the pool for reuse.
//
// Takes b (*astBuilder) which is the builder to reset and return.
func resetAndReturnAstBuilder(b *astBuilder) {
	b.emitter = nil
	b.nodeEmitter = nil
	b.ifEmitter = nil
	b.forEmitter = nil
	b.staticEmitter = nil
	b.expressionEmitter = nil
	astBuilderPool.Put(b)
}
