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

// Package component_adapters implements [component_domain.ComponentRegistry]
// with an in-memory, thread-safe store.
//
// Components are registered at application startup from local
// auto-discovery or external library registration, and looked up by
// tag name during template compilation and rendering.
//
// # Thread safety
//
// All methods on the in-memory registry are safe for concurrent use.
// Reads acquire a shared lock; writes acquire an exclusive lock.
package component_adapters
