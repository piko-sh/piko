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

// Package driven_transform_linearise reorganises PDF object order as a
// post-processing transformer so viewers can render pages progressively.
//
// It places the first page's objects at the start of the file and adds a
// linearisation parameter dictionary per PDF spec Annex F. The
// implementation recursively walks the first page's dictionary to collect
// its dependencies (content streams, resources, fonts, images) and writes
// them before all remaining objects.
package driven_transform_linearise
