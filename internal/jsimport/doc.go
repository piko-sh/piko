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

// Package jsimport provides shared JavaScript and TypeScript import path
// transformation utilities.
//
// It handles two concerns that apply to both PKC component compilation and
// PK page generation:
//
//   - Extension normalisation: extensionless imports receive a .js suffix,
//     and .ts extensions are rewritten to .js so the browser requests the
//     transpiled output.
//   - @/ alias resolution: imports starting with @/ are expanded to served
//     asset paths using the project's Go module name, for example
//     @/lib/utils becomes /_piko/assets/{moduleName}/lib/utils.js.
//
// The compiler domain uses [ResolveModuleAlias] to build [JSDependency]
// records during component compilation. The generator domain uses
// [RewriteImportRecords] to rewrite esbuild AST import records in place
// before printing transpiled JavaScript.
package jsimport
