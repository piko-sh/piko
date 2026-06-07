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
	"piko.sh/piko/internal/interp/interp_adapters"
	"piko.sh/piko/internal/interp/interp_domain"
)

var (
	// PackCompiledFileSetToBytes serialises a compiled file set into piko's schema-versioned
	// wire format. Hosts pass this as the `bytecodePacker` argument to
	// Interpreter.PackageModule when producing a [modules_domain.ModuleBundle].
	PackCompiledFileSetToBytes = interp_adapters.PackCompiledFileSetToBytes

	// LoadCompiledFromBytes deserialises a piko-packed bytecode payload back into a
	// CompiledFileSet. Hosts pass this as the `bytecodeUnpacker` argument to
	// Interpreter.LoadModule when loading a [modules_domain.ModuleBundle].
	LoadCompiledFromBytes = interp_adapters.LoadCompiledFromBytes
)

// SymbolRegistry is the symbol-and-types registry the interpreter builds during
// compilation. Exposed so hosts that implement the modules.ModuleProvider interface have
// a stable type to thread through their resolution code.
type SymbolRegistry = interp_domain.SymbolRegistry
