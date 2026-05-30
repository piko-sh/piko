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

//go:build !js || !wasm

package interp_adapters

import (
	"context"
	"errors"
	"fmt"

	"piko.sh/piko/internal/fbs"
	"piko.sh/piko/internal/interp/interp_domain"
	"piko.sh/piko/internal/interp/interp_schema"
	"piko.sh/piko/internal/interp/interp_schema/interp_schema_gen"
)

// LoadCompiledFileSet reads and deserialises a previously saved compiled file set,
// reconstructing runtime types and values via the provided SymbolRegistry.
//
// Takes key (string) which identifies the cached bytecode to load.
// Takes registry (*interp_domain.SymbolRegistry) which provides symbol and type lookups
// for runtime reconstruction.
//
// Returns *interp_domain.CompiledFileSet which is the reconstructed compiled file set.
// Returns error when the sandbox is nil, the key is empty, the cache is missing, the
// schema version has changed, or reconstruction fails.
func (bytecodeStore *BytecodeStore) LoadCompiledFileSet(ctx context.Context, key string, registry *interp_domain.SymbolRegistry) (*interp_domain.CompiledFileSet, error) {
	if bytecodeStore.sandbox == nil || key == "" {
		return nil, errors.New("bytecode store requires a sandbox and key")
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("bytecode load cancelled before read: %w", err)
	}

	fileName := fmt.Sprintf("bytecode-%s.bin", key)
	data, err := bytecodeStore.sandbox.ReadFile(fileName)
	if err != nil {
		return nil, fmt.Errorf("bytecode cache miss or read error for key %s: %w", key, err)
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("bytecode load cancelled after read: %w", err)
	}

	payload, err := interp_schema.Unpack(data)
	if err != nil {
		removeError := bytecodeStore.sandbox.Remove(fileName)
		if errors.Is(err, fbs.ErrSchemaVersionMismatch) {
			return nil, errors.Join(fmt.Errorf("bytecode schema version mismatch for key %s, invalidated", key), removeError)
		}
		return nil, errors.Join(fmt.Errorf("failed to unpack versioned bytecode for key %s: %w", key, err), removeError)
	}

	fbFileSet := interp_schema_gen.GetRootAsCompiledFileSet(payload, 0)
	if fbFileSet == nil {
		removeError := bytecodeStore.sandbox.Remove(fileName)
		return nil, errors.Join(fmt.Errorf("failed to parse corrupt bytecode file for key %s", key), removeError)
	}

	fileSet, err := unpackCompiledFileSet(ctx, fbFileSet, registry, len(payload))
	if err != nil {
		removeError := bytecodeStore.sandbox.Remove(fileName)
		return nil, errors.Join(fmt.Errorf("failed to reconstruct bytecode for key %s: %w", key, err), removeError)
	}
	return fileSet, nil
}
