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

	"piko.sh/piko/internal/generator/generator_domain"
)

// EmitterFactory creates go_literal emitter instances.
// It implements CodeEmitterFactoryPort.
type EmitterFactory struct {
	// prerenderer renders static nodes to HTML bytes at generation time.
	// May be nil, in which case prerendering is disabled.
	prerenderer generator_domain.StaticPrerenderer
}

var _ generator_domain.CodeEmitterFactoryPort = (*EmitterFactory)(nil)

// NewEmitterFactory creates a new emitter factory.
//
// Takes prerenderer (generator_domain.StaticPrerenderer) which
// renders static nodes to HTML bytes at generation time. May be
// nil to disable prerendering.
//
// Returns *EmitterFactory which is the new factory instance.
func NewEmitterFactory(_ context.Context, prerenderer generator_domain.StaticPrerenderer) *EmitterFactory {
	return &EmitterFactory{
		prerenderer: prerenderer,
	}
}

// NewEmitter creates and returns a new emitter instance that satisfies the
// factory's contract.
//
// Returns generator_domain.CodeEmitterPort which is the emitter instance.
func (f *EmitterFactory) NewEmitter() generator_domain.CodeEmitterPort {
	return NewEmitterWithPrerenderer(context.Background(), f.prerenderer)
}
