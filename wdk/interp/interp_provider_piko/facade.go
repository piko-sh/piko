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
	"piko.sh/piko/internal/templater/templater_domain"
)

// InterpreterPort is the port a single interpreter instance implements. Re-exported from
// templater_domain so external callers can type their interpreter handles.
type InterpreterPort = templater_domain.InterpreterPort

// InterpreterPoolPort is the port a pool of interpreters implements. Re-exported from
// templater_domain so external callers can type pool handles without importing the
// internal package.
type InterpreterPoolPort = templater_domain.InterpreterPoolPort

// SymbolProviderPort is the port a symbol-provider implements. Re-exported so callers can
// register additional symbols without importing templater_domain.
type SymbolProviderPort = templater_domain.SymbolProviderPort

// SymbolExports is the package -> name -> reflect.Value map shape used to register
// additional symbols visible to interpreter scripts. Re-exported as an alias so external
// callers can build the map literally without an unnamable type in their signature.
type SymbolExports = templater_domain.SymbolExports

// BatchInterpreterPort is the port batched interpreters implement.
//
// Re-exported alias for templater_domain.BatchInterpreterPort; the piko bytecode
// interpreter implements it via the CompileAndExecute path. Provided for parity even
// though most consumers use only InterpreterPort.
type BatchInterpreterPort = templater_domain.BatchInterpreterPort
