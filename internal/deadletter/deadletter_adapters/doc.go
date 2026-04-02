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

// Package deadletter_adapters implements [deadletter_domain.DeadLetterPort]
// with in-memory and disk-based (JSON lines) storage backends.
//
// Both adapters are generic, accepting any entry type via Go generics,
// and support adding, retrieving, counting, clearing, and age-based
// querying of failed items.
//
// # Thread safety
//
// All methods on both adapters are safe for concurrent use. Each
// adapter guards its internal state with a mutex.
package deadletter_adapters
