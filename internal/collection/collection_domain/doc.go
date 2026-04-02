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

// Package collection_domain defines the core domain logic for the
// collection hexagon.
//
// It contains port interfaces, domain services, and registries for
// managing content from various sources. The collection system supports
// static (build-time), dynamic (runtime), and hybrid (incremental static
// regeneration) fetching strategies.
//
// # Context handling
//
// All provider operations accept a context.Context for cancellation and
// timeout control. Dynamic and hybrid providers propagate context to
// upstream API calls, honouring deadlines set by the caller.
//
// # Thread safety
//
// All registry types and global functions are safe for concurrent use.
package collection_domain
