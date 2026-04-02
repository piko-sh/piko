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

// Package typegen_domain handles TypeScript type definition management
// and LSP intellisense for the Piko frontend framework.
//
// It writes TypeScript definition files (.d.ts) to disk for IDE
// integration and supplies completion data for the LSP server's
// piko.* and action.* namespaces. Port interfaces define the
// contracts that adapters implement for type info and action manifest
// provisioning.
//
// # Type definition sources
//
// Type definitions come from two sources. piko-ide.d.ts holds
// hand-maintained type declarations for the piko namespace, used by
// IDE plugins for autocomplete. piko-actions.d.ts holds embedded stub
// definitions for server-side actions (these will be generated from
// Go action handlers in future).
//
// # Output location
//
// Type definitions are written to {project_root}/dist/ts/. The
// IntelliJ plugin reads them directly for IDE integration, and the
// daemon serves them at /_piko/dist/ts/ for remote access.
//
// # Thread safety
//
// Both [TypeDefinitionService] and [TypeInfoService] are safe for
// concurrent use.
package typegen_domain
