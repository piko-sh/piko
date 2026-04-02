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

package templater_domain

// SymbolProviderPort defines the interface for providing Go symbols to
// the interpreter. It decouples the templater domain from the concrete
// concrete SymbolProvider implementation, preventing import cycles.
type SymbolProviderPort interface {
	// Use registers the symbols with the given interpreter instance.
	//
	// Takes i (InterpreterPort) which is the interpreter to register symbols
	// with.
	//
	// Returns error when symbol registration fails.
	Use(i InterpreterPort) error
}
