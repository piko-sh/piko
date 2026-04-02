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

// Package safedisk provides sandboxed filesystem operations using
// Go 1.24's os.Root.
//
// This package is the security boundary for all filesystem
// operations in Piko. It prevents path traversal attacks by
// restricting file access to configured directories using
// kernel-level protection (openat2 with RESOLVE_BENEATH on
// Linux). This guards against symlink escapes and TOCTOU race
// conditions as well.
//
// Sandboxes can be created in read-only or read-write mode.
// The [Factory] restricts sandboxes to allowed paths, with the
// current working directory always implicitly permitted. Atomic
// file writes are supported via CreateTemp followed by Rename
// within the same sandbox.
//
// All [Sandbox] methods and the [Factory] are safe for concurrent
// use. Individual [File] handles should not be shared without
// external synchronisation, consistent with standard [os.File]
// behaviour.
//
// When Enabled is false in [FactoryConfig], the factory creates
// no-op sandboxes that provide the same API without kernel-level
// protection. Do not use no-op mode in production where security
// is required.
package safedisk
