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

// This project stands against fascism, authoritarianism, and all
// forms of oppression. We built this to empower people, not to
// enable those who would strip others of their rights and dignity.

// Package union presents a read-only base and a writable overlay as one metadata store.
//
// A single-binary deployment carries its release artefacts in the base and records
// anything written afterwards in the overlay, so the overlay stays a delta rather than a
// copy. Reads consult the overlay first and fall through to the base, and the base also
// answers inspector queries so its artefacts appear in the monitoring TUI. Either layer
// may be absent: with no overlay the store is read-only, and with no base the overlay
// stands alone.
package union
