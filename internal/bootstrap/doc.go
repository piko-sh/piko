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

// Package bootstrap assembles the Piko application and acts as its
// composition root.
//
// It uses a lightweight, manual Dependency Injection container and the
// Strategy pattern to construct the correct set of services for a given
// run mode (production, development, or interpreted development). Every
// service is initialised lazily and used as a singleton.
//
// # Run modes
//
// The package supports three run modes, each with its own builder:
// prod (compiled templates with AST caching), dev (file watching
// with hot reload), and dev-i (development with Go interpretation).
//
// # Usage
//
// Bootstrap is typically invoked in two phases:
//
//	// Phase 1: Create container and load configuration
//	container, err := bootstrap.ConfigAndContainer(ctx, deps,
//	    bootstrap.WithMemoryRegistryCache(),
//	    bootstrap.WithFrontendModule(daemon_frontend.ModuleAnalytics, config),
//	)
//
//	// Phase 2: Assemble the daemon for the chosen run mode
//	daemon, err := bootstrap.Daemon(ctx, piko.RunModeProd, container, deps)
//
// # Global service access
//
// After initialisation, services can be accessed globally via
// convenience functions. These return errNotInitialised if
// called before the framework is initialised:
//
//	emailService, err := bootstrap.GetEmailService()
//	cacheService, err := bootstrap.GetCacheService()
//	storageService, err := bootstrap.GetStorageService()
//
// # Thread safety
//
// The Container uses sync.Once guards for lazy initialisation, so each
// service is created exactly once even under concurrent access. The global
// service getters are safe for concurrent use from multiple goroutines.
package bootstrap
