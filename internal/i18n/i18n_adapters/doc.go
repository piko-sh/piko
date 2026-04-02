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

// Package i18n_adapters implements [i18n_domain.Service] via
// filesystem-backed loaders and emitters for internationalisation
// data.
//
// It supports JSON (for development) and FlatBuffer (for production,
// zero-allocation parsing) storage formats. All file operations are
// sandboxed via [safedisk.Sandbox] to prevent path traversal attacks.
//
// # Usage
//
// For production, use the auto-detecting service constructor:
//
//	service, err := i18n_adapters.NewService(ctx, sandbox, config)
//
// This checks for a pre-compiled FlatBuffer (dist/i18n.bin) first,
// falling back to JSON if not found. For explicit format selection:
//
//	// FlatBuffer mode (recommended for production)
//	loader := i18n_adapters.NewLoader(i18n_adapters.LoaderConfig{
//	    Sandbox:        sandbox,
//	    Mode:           i18n_adapters.LoaderModeFlatBuffer,
//	    FlatBufferPath: "dist/i18n.bin",
//	})
//
//	// JSON mode (for development)
//	loader := i18n_adapters.NewLoader(i18n_adapters.LoaderConfig{
//	    Sandbox:       sandbox,
//	    Mode:          i18n_adapters.LoaderModeJSON,
//	    JSONDirectory: "src/i18n",
//	})
//
// # Thread safety
//
// The service returned by [NewService] and [NewFlatBufferService]
// is safe for concurrent use. Providers and emitters should be used
// from a single goroutine.
package i18n_adapters
