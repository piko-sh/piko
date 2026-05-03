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

// Package driver_asset_registrar implements the collection_domain.AssetRegistrar
// port backed by the artefact registry.
//
// Static collection providers that pull content from external module
// sandboxes (for example the markdown provider reading docs from a Go
// module) discover sibling asset references during build. This adapter
// reads those assets via the supplied sandbox, upserts them into the
// artefact registry, and returns the serve URL that rewritten src
// attributes should point at.
package driver_asset_registrar
