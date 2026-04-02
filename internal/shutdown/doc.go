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

// Package shutdown provides graceful shutdown coordination for applications.
//
// It manages orderly cleanup when an application terminates, listening
// for OS signals (SIGINT, SIGTERM) and executing registered cleanup
// functions in LIFO order within a configurable timeout.
//
// # Usage
//
// Register cleanup functions during initialisation. They execute in reverse
// order (LIFO) during shutdown, similar to Go's defer semantics:
//
//	// Register cleanup for a database connection
//	shutdown.Register(ctx, "database", func(ctx context.Context) error {
//	    return db.Close()
//	})
//
//	// Register cleanup for cache (will run before database)
//	shutdown.Register(ctx, "cache", func(ctx context.Context) error {
//	    return cache.Flush(ctx)
//	})
//
//	// Block until shutdown signal, then run cleanup
//	shutdown.ListenAndShutdown(shutdown.DefaultTimeout)
//
// For testing, create isolated Manager instances:
//
//	manager := shutdown.NewManager()
//	manager.Register(ctx, "test-resource", cleanupFunction)
//
// # Thread safety
//
// All methods are safe for concurrent use. The Manager uses a mutex to protect
// registration and an atomic flag to prevent registration during cleanup.
package shutdown
