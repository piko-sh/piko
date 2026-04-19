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

// Package route_pattern parses the trailing `{name[:regex]}` segment of a
// piko route pattern.
//
// Shared by the manifest builder (which widens bare `{name}` to `{name:.+}`
// for collection pages) and the HTTP router (which translates `{name:regex}`
// into chi's native `*` form), so both sides agree on the grammar.
package route_pattern
