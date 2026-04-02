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
	"piko.sh/piko/internal/generator/generator_domain"
)

func TestNewEmitterFactory(t *testing.T) {
	t.Parallel()

	factory := NewEmitterFactory(context.Background(), nil)

	require.NotNil(t, factory, "NewEmitterFactory should return non-nil")
}

func TestEmitterFactory_ImplementsCodeEmitterFactoryPort(t *testing.T) {
	t.Parallel()

	factory := NewEmitterFactory(context.Background(), nil)

	var _ generator_domain.CodeEmitterFactoryPort = factory
	assert.NotNil(t, factory)
}

func TestEmitterFactory_NewEmitter(t *testing.T) {
	t.Parallel()

	factory := NewEmitterFactory(context.Background(), nil)

	em := factory.NewEmitter()

	require.NotNil(t, em, "NewEmitter should return non-nil")

	var _ generator_domain.CodeEmitterPort = em
}

func TestEmitterFactory_NewEmitter_ReturnsNewInstances(t *testing.T) {
	t.Parallel()

	factory := NewEmitterFactory(context.Background(), nil)

	emitter1 := factory.NewEmitter()
	emitter2 := factory.NewEmitter()

	require.NotNil(t, emitter1)
	require.NotNil(t, emitter2)
	assert.NotSame(t, emitter1, emitter2, "Each call should return a new instance")
}
