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

// Package provider_mock supplies in-memory mock cache adapters for testing.
//
// The generic mock implementation of the cache provider and adapter
// interfaces supports call recording, error injection, and state
// simulation, allowing tests to exercise cache-dependent code without
// real cache backends.
//
// # Thread safety
//
// All methods are safe for concurrent use. Internal state is guarded
// by a sync.RWMutex.
package provider_mock
