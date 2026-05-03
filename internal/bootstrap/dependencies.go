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

package bootstrap

import (
	"github.com/go-chi/chi/v5"
	"piko.sh/piko/internal/templater/templater_domain"
)

// Dependencies holds all the external inputs required to bootstrap the Piko
// application. This creates a clean and stable API for the bootstrap functions.
type Dependencies struct {
	// AppRouter is the root Chi router for the application. The bootstrap
	// process mounts all generated routes onto this router, including pages,
	// partials, and actions.
	AppRouter *chi.Mux

	// SymbolProvider provides symbols for interpreted mode; nil for other modes.
	// The concrete type implements SymbolProviderPort.
	SymbolProvider any

	// InterpreterPool provides a pool of interpreters for JIT compilation in
	// interpreted mode; nil for other modes.
	InterpreterPool templater_domain.InterpreterPoolPort
}
