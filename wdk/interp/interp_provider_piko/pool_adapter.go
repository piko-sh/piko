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

package interp_provider_piko

import (
	"fmt"
	"sync"

	"piko.sh/piko/internal/interp/interp_domain"
	"piko.sh/piko/internal/templater/templater_domain"
)

// poolAdapter wraps *sync.Pool to implement InterpreterPoolPort.
type poolAdapter struct {
	// pool holds reusable interpreter service instances.
	pool *sync.Pool

	// bytecodeEmissionDirectory is the root directory for emitting
	// source and compiled bytecode to disk. Empty disables emission.
	bytecodeEmissionDirectory string
}

var _ templater_domain.InterpreterPoolPort = (*poolAdapter)(nil)

// Get retrieves an interpreter from the pool.
// The returned interpreter is ready for use with symbols pre-loaded.
//
// Returns templater_domain.InterpreterPort which is a ready-to-use
// interpreter.
// Returns error when the pool returns an invalid type.
func (p *poolAdapter) Get() (templater_domain.InterpreterPort, error) {
	item := p.pool.Get()
	service, ok := item.(*interp_domain.Service)
	if !ok {
		return nil, fmt.Errorf("interpreter pool returned invalid type: %T", item)
	}
	return &interpreterAdapter{
		service:                   service,
		bytecodeEmissionDirectory: p.bytecodeEmissionDirectory,
	}, nil
}

// Put returns an interpreter to the pool after resetting it.
// The interpreter's state is cleared before being returned to the pool.
//
// Takes i (InterpreterPort) which is the interpreter to return.
func (p *poolAdapter) Put(i templater_domain.InterpreterPort) {
	if adapter, ok := i.(*interpreterAdapter); ok {
		adapter.Reset()
		p.pool.Put(adapter.service)
	}
}

// newPoolAdapter creates a new pool adapter with a golden interpreter
// service. Each Get() call returns a clone of the golden service.
//
// Takes golden (*interp_domain.Service) which is the pre-warmed service
// with symbols loaded.
// Takes bytecodeEmissionDirectory (string) which is the root directory
// for emitting source and bytecode to disk. Empty disables emission.
//
// Returns *poolAdapter which provides pooled interpreters.
func newPoolAdapter(golden *interp_domain.Service, bytecodeEmissionDirectory string) *poolAdapter {
	return &poolAdapter{
		pool: &sync.Pool{
			New: func() any {
				return golden.Clone()
			},
		},
		bytecodeEmissionDirectory: bytecodeEmissionDirectory,
	}
}
