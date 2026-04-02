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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPool_GetAstBuilder_Initialisation(t *testing.T) {
	t.Parallel()

	em := requireEmitter(t)
	em.resetState(context.Background())

	builder := getAstBuilder(context.Background(), em)

	require.NotNil(t, builder, "Should return non-nil builder")
	assert.Same(t, em, builder.emitter, "Should reference the emitter")

	assert.NotNil(t, builder.nodeEmitter, "NodeEmitter should be initialised")
	assert.NotNil(t, builder.expressionEmitter, "ExpressionEmitter should be initialised")
	assert.NotNil(t, builder.staticEmitter, "StaticEmitter should be initialised")
	assert.NotNil(t, builder.ifEmitter, "IfEmitter should be initialised")
	assert.NotNil(t, builder.forEmitter, "ForEmitter should be initialised")

	expressionEmitter, ok := builder.expressionEmitter.(*expressionEmitter)
	require.True(t, ok, "Should be concrete expressionEmitter type")
	assert.NotNil(t, expressionEmitter.binaryEmitter, "BinaryOpEmitter should be initialised")
	assert.Same(t, em, expressionEmitter.emitter, "ExpressionEmitter should reference emitter")

	binaryEmitter, ok := expressionEmitter.binaryEmitter.(*binaryOpEmitter)
	require.True(t, ok, "Should be concrete binaryOpEmitter type")
	assert.Same(t, expressionEmitter, binaryEmitter.expressionEmitter, "BinaryEmitter should reference ExpressionEmitter")

	nodeEmitter, ok := builder.nodeEmitter.(*nodeEmitter)
	require.True(t, ok, "Should be concrete nodeEmitter type")
	assert.NotNil(t, nodeEmitter.attributeEmitter, "AttributeEmitter should be initialised")

	concreteAttributeEmitter, ok := nodeEmitter.attributeEmitter.(*attributeEmitter)
	require.True(t, ok, "Should be concrete attributeEmitter type")
	assert.Same(t, em, concreteAttributeEmitter.emitter, "AttributeEmitter should reference emitter")
	assert.Same(t, expressionEmitter, concreteAttributeEmitter.expressionEmitter, "AttributeEmitter should reference ExpressionEmitter")

	ifEmitter, ok := builder.ifEmitter.(*ifEmitter)
	require.True(t, ok, "Should be concrete ifEmitter type")
	assert.Same(t, builder, ifEmitter.astBuilder, "IfEmitter should reference AstBuilder")
	assert.Same(t, em, ifEmitter.emitter, "IfEmitter should reference emitter")

	forEmitter, ok := builder.forEmitter.(*forEmitter)
	require.True(t, ok, "Should be concrete forEmitter type")
	assert.Same(t, builder, forEmitter.astBuilder, "ForEmitter should reference AstBuilder")
	assert.Same(t, em, forEmitter.emitter, "ForEmitter should reference emitter")
}

func TestPool_PutAstBuilder_Cleanup(t *testing.T) {

	em := requireEmitter(t)
	em.resetState(context.Background())

	builder := getAstBuilder(context.Background(), em)
	require.NotNil(t, builder.emitter, "Should be initialised before cleanup")

	putAstBuilder(builder)

	assert.Nil(t, builder.emitter, "Emitter reference should be nil")
	assert.Nil(t, builder.nodeEmitter, "NodeEmitter should be nil")
	assert.Nil(t, builder.expressionEmitter, "ExpressionEmitter should be nil")
	assert.Nil(t, builder.staticEmitter, "StaticEmitter should be nil")
	assert.Nil(t, builder.ifEmitter, "IfEmitter should be nil")
	assert.Nil(t, builder.forEmitter, "ForEmitter should be nil")
}

func TestPool_PutAstBuilder_NilSafety(t *testing.T) {
	t.Parallel()

	assert.NotPanics(t, func() {
		putAstBuilder(nil)
	}, "Should handle nil builder gracefully")
}

func TestPool_RetrieveAstBuilderFromPool(t *testing.T) {
	t.Parallel()

	builder := retrieveAstBuilderFromPool(context.Background())

	require.NotNil(t, builder, "Should return non-nil builder")

	astBuilderPool.Put(builder)
}

func TestPool_MultipleGetPut_NoStateLeak(t *testing.T) {
	t.Parallel()

	em1 := requireEmitter(t)
	em1.resetState(context.Background())
	em1.config = EmitterConfig{PackageName: "test1"}

	builder1 := getAstBuilder(context.Background(), em1)
	assert.Same(t, em1, builder1.emitter)
	putAstBuilder(builder1)

	em2 := requireEmitter(t)
	em2.resetState(context.Background())
	em2.config = EmitterConfig{PackageName: "test2"}

	builder2 := getAstBuilder(context.Background(), em2)

	assert.Same(t, em2, builder2.emitter, "Should reference new emitter")
	assert.NotSame(t, em1, builder2.emitter, "Should not reference old emitter")

	if builder1 == builder2 {

		assert.Equal(t, "test2", builder2.emitter.config.PackageName, "Should have new config")
	}

	putAstBuilder(builder2)
}

func TestPool_ExpressionEmitterCleanup(t *testing.T) {

	em := requireEmitter(t)
	em.resetState(context.Background())

	builder := getAstBuilder(context.Background(), em)

	expressionEmitter, ok := builder.expressionEmitter.(*expressionEmitter)
	require.True(t, ok)
	require.NotNil(t, expressionEmitter.binaryEmitter)

	putAstBuilder(builder)

	assert.Nil(t, builder.expressionEmitter)
}

func TestPool_StaticEmitterCleanup(t *testing.T) {

	em := requireEmitter(t)
	em.resetState(context.Background())

	builder := getAstBuilder(context.Background(), em)

	staticEmitter, ok := builder.staticEmitter.(*staticEmitter)
	require.True(t, ok)

	require.NotNil(t, staticEmitter.emitter)

	putAstBuilder(builder)

	assert.Nil(t, builder.staticEmitter)
}
