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

// Package lifecycle provides utilities for managing application lifecycle
// events, including graceful shutdown.
//
// Components can register cleanup functions that are automatically called
// when the application receives a termination signal (SIGINT or SIGTERM).
// Cleanup functions run in reverse registration order (LIFO), mirroring
// Go's defer semantics, so that resources registered early (such as the
// logger) are cleaned up last.
//
// # Usage
//
// Register a cleanup function by providing a name (used for logging and
// observability) and a function that accepts a context:
//
//	lifecycle.Register("database", func(ctx context.Context) error {
//	    return db.Close()
//	})
//
//	lifecycle.Register("cache", func(ctx context.Context) error {
//	    return cache.Flush(ctx)
//	})
//
// Each cleanup function receives a context with a per-function timeout
// budget so that the overall shutdown completes within the configured
// deadline.
//
// # Thread safety
//
// [Register] is safe for concurrent use by multiple goroutines.
// Registrations made after cleanup has begun are ignored with a
// warning log.
package lifecycle
