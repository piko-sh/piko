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

// Package daemon_domain orchestrates HTTP server lifecycle, graceful
// shutdown, build notification processing, and on-demand image variant
// generation. It defines port interfaces (DaemonService, ServerAdapter,
// RouterBuilder, SEOServicePort, SignalNotifier) and the action system
// that enables server-side operations called from PK templates,
// including SSE transport.
//
// # Server modes
//
// The daemon supports two operating modes:
//
//   - Development: Includes file watching, hot-reload, and build
//     notifications via the coordinator
//   - Production: Assumes pre-built artefacts, optimised for performance
//
// # Thread safety
//
// The DaemonService is safe for concurrent use. The on-demand variant
// generator uses per-variant locking to prevent duplicate generation of the
// same variant by concurrent requests.
package daemon_domain
