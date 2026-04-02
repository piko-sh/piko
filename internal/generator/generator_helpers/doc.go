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

// Package generator_helpers provides runtime utility functions called
// by generated Go code from Piko templates.
//
// The code generator emits calls to these functions for runtime
// concerns that cannot be resolved at generation time: dynamic value
// evaluation, class/style merging, type coercion, action payload
// encoding, and truthiness checks. The package also includes AST
// traversal utilities used by the generator itself.
//
// All exported functions are safe for concurrent use. Internal pooled
// resources use sync.Pool for thread-safe reuse. Arena-aware variants
// use per-request arenas that must not be shared between goroutines.
package generator_helpers
