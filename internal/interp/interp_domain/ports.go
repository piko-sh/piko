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
	"go/types"
)

// SymbolProviderPort provides pre-registered native Go symbols to the
// interpreter. Implementations may load from vendored stdlib tables,
// user-generated extraction files, or programmatic registrations.
type SymbolProviderPort interface {
	// Exports returns the complete symbol export table keyed by Go
	// import path (e.g. "fmt", "encoding/json").
	Exports() SymbolExports
}

// TypesPackageProviderPort is an optional extension of [SymbolProviderPort].
// Providers implementing it supply pre-built [types.Package] objects for packages
// whose type information cannot be synthesised from reflect values alone.
type TypesPackageProviderPort interface {
	// TypesPackages returns a map of import paths to pre-built
	// types.Package objects, such as those for generic packages
	// (e.g. slices, maps) that cannot be synthesised from reflect.
	//
	// Returns map[string]*types.Package which maps each import path
	// to its pre-built package.
	TypesPackages() map[string]*types.Package
}

// BytecodeStorePort provides persistence for compiled bytecode.
// Implementations serialise and deserialise CompiledFileSet using
// FlatBuffers with schema-versioned payloads.
type BytecodeStorePort interface {
	// SaveCompiledFileSet serialises and persists a compiled file set
	// under the given key.
	//
	// Takes ctx (context.Context) for cancellation.
	// Takes key (string) which identifies the compiled file set.
	// Takes cfs (*CompiledFileSet) which is the compiled file set to
	// persist.
	//
	// Returns error when serialisation or storage fails.
	SaveCompiledFileSet(ctx context.Context, key string, cfs *CompiledFileSet) error

	// LoadCompiledFileSet reads and deserialises a previously saved
	// compiled file set, reconstructing runtime types and values via
	// the provided SymbolRegistry.
	//
	// Takes ctx (context.Context) for cancellation.
	// Takes key (string) which identifies the compiled file set to
	// load.
	// Takes registry (*SymbolRegistry) which provides access to
	// registered package symbols for reconstruction.
	//
	// Returns *CompiledFileSet which is the reconstructed compiled
	// file set.
	// Returns error when the key is not found, the schema version
	// has changed, or reconstruction fails.
	LoadCompiledFileSet(ctx context.Context, key string, registry *SymbolRegistry) (*CompiledFileSet, error)
}
