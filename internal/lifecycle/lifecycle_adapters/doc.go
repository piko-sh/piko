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

// Package lifecycle_adapters implements the lifecycle_domain ports for
// file watching (fsNotifyWatcher), production builds (buildService),
// and interpreted code execution (InterpretedBuildOrchestrator).
//
// # Hot reload architecture
//
// The package implements a two-phase hot-reload strategy:
//
//  1. MarkDirty (fast path): On file save, components are marked dirty without
//     recompilation. This gives sub-second feedback (~10-50ms).
//  2. JITCompile (on request): Actual compilation occurs only when a
//     dirty page is requested, compiling just the necessary component and its
//     dependencies
//
// # Thread safety
//
// All exported types are safe for concurrent use. The InterpretedBuildOrchestrator
// uses a combination of sync.RWMutex for state access, singleflight for
// deduplicating concurrent compilations, and a semaphore for limiting
// interpreter concurrency.
package lifecycle_adapters
