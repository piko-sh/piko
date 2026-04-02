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

// Package asset_resolver implements the email asset resolver port
// using the registry service. It fetches email assets (images, files)
// from the registry, applies transformation profiles (resizing, format
// conversion), and returns ready-to-embed attachments with Content-ID
// set for inline references.
//
// Variant selection uses a progressive fallback: exact match first,
// then relaxing density, dimensions, profile, and finally falling
// back to the source or any available variant.
//
// All methods are safe for concurrent use.
package asset_resolver
