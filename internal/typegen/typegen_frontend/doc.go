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

// Package typegen_frontend bundles embedded TypeScript type definitions
// for the Piko frontend framework.
//
// The definitions are generated from TypeScript source by vite-plugin-dts
// during the frontend build (npm run build in frontend/core) and embedded
// into the Go binary via [embed.FS] so the daemon can copy them to a
// project's dist/ts/ directory for IDE integration.
//
// [typegen_adapters.NewTypeDefinitionService] reads from
// [EmbeddedTypeDefinitions] to supply Piko framework types alongside
// action stub types.
package typegen_frontend
