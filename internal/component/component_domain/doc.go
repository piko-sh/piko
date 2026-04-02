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

// Package component_domain defines the [ComponentRegistry] port interface
// and validation logic for the PKC component registry system.
//
// It enforces tag name rules from the Web Components specification and
// Piko's reserved namespace constraints, and prevents shadowing of
// standard HTML element names.
//
// # Validation rules
//
// Custom element tag names must satisfy all of the following:
//
//   - Contain at least one hyphen (Web Components specification)
//   - Not use reserved prefixes (piko:, pml-)
//   - Not shadow any standard HTML element name
//
// All tag name lookups are case-insensitive to match HTML behaviour.
//
// # Integration
//
// Adapters implementing [ComponentRegistry] are populated at startup
// with locally discovered components and external components registered
// via the WithComponents() facade option. The registry is consumed by
// the rendering pipeline to resolve custom element tags to their
// component definitions (see [component_dto.ComponentDefinition]).
package component_domain
