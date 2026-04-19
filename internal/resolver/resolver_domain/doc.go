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

// Package resolver_domain defines the port interfaces for import path
// resolution.
//
// It defines the [ResolverPort] interface for mapping Piko import paths
// (components, CSS, assets) to absolute filesystem paths, handling Go module
// detection, the @ alias for local module references, and external module
// resolution via GOMODCACHE.
//
// # Import path formats
//
// The resolver supports several path formats:
//
//   - Module-absolute: a GitHub-hosted module path such as
//     "example.com/myorg/myproject/partials/card.pk"
//   - @ alias: "@/partials/card.pk" (resolved relative to containing file)
//   - Relative: "./theme.css" or "../styles.css" (CSS imports only)
//
// # The @ alias
//
// The @ alias provides a concise syntax for referencing the local module:
//
//	@/partials/card.pk -> example.com/myorg/myproject/partials/card.pk
//
// Resolution is relative to the file containing the import. This guarantees
// correct behaviour when importing from external modules that also use @.
package resolver_domain
