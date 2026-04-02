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

// Package lifecycle_domain orchestrates the bridge between build-time
// operations and runtime execution, defining port interfaces
// (FileSystemWatcher, RouterReloadNotifier, InterpretedBuildOrchestrator)
// and coordinating file watching, build notifications, asset pipeline
// processing, and router hot-reload.
//
// # Asset processing
//
// The package generates transformation profiles for various asset
// types:
//
//   - Images (piko:img): Responsive variants, format conversion,
//     placeholders
//   - Videos (piko:video): HLS transcoding at multiple quality levels
//   - Static assets: Minification and compression (gzip, Brotli)
//
// # Thread safety
//
// The lifecycleService is safe for concurrent use. Entry point access
// is protected by a read-write mutex, and background goroutines
// coordinate via channels and stop signals.
package lifecycle_domain
