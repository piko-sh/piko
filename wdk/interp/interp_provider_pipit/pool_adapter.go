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

package interp_provider_pipit

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"piko.sh/piko/internal/templater/templater_domain"
	"pipit.sh/pipit"
)

var (
	// errModulesNotLoaded reports a Get on a pool whose queued modules have not been loaded
	// yet.
	errModulesNotLoaded = errors.New("interp_provider_pipit: queued modules are not loaded; call LoadModules before Get")
)

// poolAdapter hands out clones of a golden interpreter to implement InterpreterPoolPort.
//
// Every clone shares the golden interpreter's symbol registry. Packages compiled by one
// interpreter are registered there, which is what lets a later JIT recompile of a single
// component resolve the user packages the initial build compiled. Do not replace the
// clones with independent interpreters without accounting for that.
type poolAdapter struct {
	// golden is the pre-warmed interpreter every returned interpreter is cloned from.
	golden *pipit.Interpreter

	// moduleLoad holds the outcome of LoadModules; nil until it has run.
	moduleLoad atomic.Pointer[moduleLoadOutcome]

	// bytecodeEmissionDirectory is the root directory for emitting source and compiled
	// bytecode to disk. Empty disables emission.
	bytecodeEmissionDirectory string

	// pendingModules are the module bundles LoadModules loads into the golden interpreter.
	pendingModules []pendingModuleLoad

	// loadMutex serialises LoadModules so the golden interpreter is loaded exactly once.
	loadMutex sync.Mutex
}

// moduleLoadOutcome records the result of loading the queued modules.
type moduleLoadOutcome struct {
	// err is the first load failure, or nil when every module loaded.
	err error
}

var (
	_ templater_domain.InterpreterPoolPort = (*poolAdapter)(nil)
)

// newPoolAdapter creates a new pool adapter around a golden interpreter.
//
// Takes golden (*pipit.Interpreter) which is the pre-warmed interpreter with symbols
// loaded.
// Takes pendingModules ([]pendingModuleLoad) which are the bundles LoadModules loads.
// Takes bytecodeEmissionDirectory (string) which is the root directory for emitting
// source and bytecode to disk. Empty disables emission.
//
// Returns *poolAdapter which provides the interpreters.
func newPoolAdapter(golden *pipit.Interpreter, pendingModules []pendingModuleLoad, bytecodeEmissionDirectory string) *poolAdapter {
	return &poolAdapter{
		golden:                    golden,
		moduleLoad:                atomic.Pointer[moduleLoadOutcome]{},
		bytecodeEmissionDirectory: bytecodeEmissionDirectory,
		pendingModules:            pendingModules,
		loadMutex:                 sync.Mutex{},
	}
}

// LoadModules loads the queued module bundles into the golden interpreter, verifying each
// against its ref's pin, so every interpreter the pool returns can import them.
//
// The work runs once; later calls return the first call's result. A failure is also
// returned by every later Get, so a bad module fails loudly rather than silently serving
// interpreters without it.
//
// Returns error when a module fails to load or ctx is cancelled.
//
// Safe for concurrent use. The load mutex serialises callers, and the first load outcome
// is published atomically for subsequent calls and interpreter retrieval.
func (p *poolAdapter) LoadModules(ctx context.Context) error {
	p.loadMutex.Lock()
	defer p.loadMutex.Unlock()

	if outcome := p.moduleLoad.Load(); outcome != nil {
		return outcome.err
	}

	err := p.loadPendingModules(ctx)
	p.moduleLoad.Store(&moduleLoadOutcome{err: err})
	return err
}

// Get returns a clone of the golden interpreter, ready for use with symbols pre-loaded.
//
// Returns templater_domain.InterpreterPort which is a ready-to-use interpreter.
// Returns error when modules are queued but LoadModules has not run, or when loading them
// failed.
func (p *poolAdapter) Get() (templater_domain.InterpreterPort, error) {
	if len(p.pendingModules) > 0 {
		outcome := p.moduleLoad.Load()
		if outcome == nil {
			return nil, errModulesNotLoaded
		}
		if outcome.err != nil {
			return nil, outcome.err
		}
	}
	return &interpreterAdapter{
		service:                   p.golden.Clone(),
		bytecodeEmissionDirectory: p.bytecodeEmissionDirectory,
	}, nil
}

// loadPendingModules loads each queued bundle into the golden interpreter, walking its
// exports into the symbol registry so every pooled clone inherits them.
//
// Returns error which is the first load failure.
func (p *poolAdapter) loadPendingModules(ctx context.Context) error {
	for _, pending := range p.pendingModules {
		if ctx.Err() != nil {
			return fmt.Errorf("interp_provider_pipit: loading modules: %w", context.Cause(ctx))
		}
		if err := p.loadModule(ctx, pending); err != nil {
			return fmt.Errorf("interp_provider_pipit: loading module %q: %w", pending.ref.Path, err)
		}
	}
	return nil
}

// loadModule loads one bundle into the golden interpreter behind a panic boundary.
//
// Takes pending (pendingModuleLoad) which is the bundle and ref to load.
//
// Returns error when the interpreter rejects or fails to load the bundle.
func (p *poolAdapter) loadModule(ctx context.Context, pending pendingModuleLoad) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = interpreterPanicError(ctx, "loading a module", recovered)
		}
	}()
	_, err = p.golden.LoadModule(ctx, toPipitBundle(pending.bundle), toPipitRef(pending.ref), nil, pipit.LoadCompiledFromBytes)
	return translateModuleError(err)
}
