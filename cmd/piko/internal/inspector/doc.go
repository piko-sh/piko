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

// Package inspector contains shared dto types and protobuf-to-dto extractors.
//
// The dto types and extractors are used by both the CLI describe / get / info
// commands and the TUI panels. The package exists so the two front-ends agree
// on what the data shape is for each inspector domain (providers, DLQ, rate
// limiter, health, build/runtime/memory/process, watchdog) without each side
// re-implementing the gRPC to field-list mapping.
//
// Extractors return DetailBody values; renderers in the CLI (Printer)
// and TUI (RenderDetailBody) adapt those into stdout or pane output.
package inspector
