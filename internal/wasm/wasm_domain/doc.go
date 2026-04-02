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

// Package wasm_domain orchestrates the WASM runtime for Piko.
//
// The [Orchestrator] coordinates the inspector, compiler, generator,
// interpreter, and render hexagons for browser-based REPL and playground
// scenarios. It exposes a simple API that JavaScript code can call to
// analyse Go source, provide code completions, display hover information,
// validate syntax, generate code, and render templates.
//
// # Usage
//
//	orchestrator := wasm_domain.NewOrchestrator(
//	    wasm_domain.WithStdlibLoader(loader),
//	    wasm_domain.WithJSInterop(interop),
//	    wasm_domain.WithConsole(console),
//	    wasm_domain.WithGenerator(gen),
//	    wasm_domain.WithRenderer(renderer),
//	)
//	if err := orchestrator.Initialise(ctx); err != nil {
//	    return err
//	}
//
//	// Analyse Go source code
//	response, err := orchestrator.Analyse(ctx, &wasm_dto.AnalyseRequest{
//	    Sources: map[string]string{"main.go": source},
//	})
//
// # Thread safety
//
// The [Orchestrator] is safe for concurrent use. All public methods
// acquire appropriate read or write locks when accessing shared state.
package wasm_domain
