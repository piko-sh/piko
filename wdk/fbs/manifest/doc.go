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

// Package manifest provides public access to the manifest FlatBuffer
// schema for inspecting compiled manifest.bin files.
//
// It exposes the schema hash, an [Unpack] function to strip the
// version header, and [ConvertManifest] to decode the raw FlatBuffer
// payload into a JSON-serialisable [Manifest] struct. The manifest
// contains metadata for pages, partials, emails, and PDFs, including
// route patterns, style blocks, i18n strategies, and local
// translations.
package manifest
