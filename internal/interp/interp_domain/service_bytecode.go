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

package interp_domain

import (
	"context"
	"fmt"
)

// WithBytecodeStore configures a BytecodeStorePort for persisting
// compiled bytecode. When set, SaveCompiled and LoadCompiled become
// available on the service.
//
// Takes store (BytecodeStorePort) which provides bytecode
// serialisation and persistence.
//
// Returns Option which configures the service.
func WithBytecodeStore(store BytecodeStorePort) Option {
	return func(c *serviceConfig) {
		c.bytecodeStore = store
	}
}

// SaveCompiled persists a compiled file set under the given key
// using the configured BytecodeStorePort.
//
// Takes ctx (context.Context) for cancellation.
// Takes key (string) which identifies the compiled file set.
// Takes cfs (*CompiledFileSet) which is the compiled file set to
// persist.
//
// Returns error when no store is configured, or serialisation fails.
func (s *Service) SaveCompiled(ctx context.Context, key string, cfs *CompiledFileSet) error {
	if s.config.bytecodeStore == nil {
		return errNoBytecodeStore
	}
	return s.config.bytecodeStore.SaveCompiledFileSet(ctx, key, cfs)
}

// LoadCompiled loads a previously saved compiled file set by key,
// reconstructing runtime types and values via the service's
// SymbolRegistry.
//
// Takes ctx (context.Context) for cancellation.
// Takes key (string) which identifies the compiled file set to load.
//
// Returns *CompiledFileSet which is the reconstructed compiled file
// set.
// Returns error when no store is configured, the key is not found,
// the schema version has changed, or reconstruction fails.
func (s *Service) LoadCompiled(ctx context.Context, key string) (*CompiledFileSet, error) {
	if s.config.bytecodeStore == nil {
		return nil, errNoBytecodeStore
	}
	cfs, err := s.config.bytecodeStore.LoadCompiledFileSet(ctx, key, s.symbols)
	if err != nil {
		return nil, fmt.Errorf("loading compiled bytecode %q: %w", key, err)
	}
	return cfs, nil
}
