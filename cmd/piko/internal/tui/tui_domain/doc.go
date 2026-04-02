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

// Package tui_domain defines the core business logic, port interfaces,
// and UI components for the terminal-based monitoring tool.
//
// It defines provider port interfaces for data fetching, the [Panel]
// interface and [BasePanel] base type for UI sections, reusable widgets
// (sparklines, tables, search boxes, status bars), the generic
// [AssetViewer] for list-based panels, and the main [Service] that
// orchestrates the Bubble Tea program lifecycle. Provider ports are
// implemented by adapters in sibling packages (e.g. provider_grpc);
// [Service] accepts a [Providers] struct for dependency injection and
// manages background data refresh via an internal orchestrator.
package tui_domain
