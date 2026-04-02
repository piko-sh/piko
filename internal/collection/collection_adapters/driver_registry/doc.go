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

// Package driver_registry implements the provider registry port for
// collections.
//
// It stores registered collection providers in memory, allowing the
// collection service to look them up by name at runtime.
//
// # Usage
//
//	registry := driver_registry.NewMemoryRegistry()
//	registry.Register(myProvider)
//
//	provider, ok := registry.Get("markdown")
//
// # Thread safety
//
// All methods are safe for concurrent use. The registry uses a
// read-write mutex to allow concurrent reads whilst serialising writes.
package driver_registry
