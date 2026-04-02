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

// Package generator_domain defines the core code generation business logic
// for Piko.
//
// It orchestrates the compilation pipeline that transforms annotated PK
// templates into executable Go code. It coordinates between the annotator,
// which builds dependency graphs and semantic metadata, and various
// emitters that produce Go source files, manifests, and static assets. It
// also handles client-side concerns such as TypeScript transpilation and
// PK script source transformation.
//
// The service supports two compilation modes. Single-file mode is used by
// development servers and testing. Full project mode is used for
// production builds, running annotation and code generation in parallel
// with a worker pool, and optionally producing SEO artefacts, i18n
// binaries, and search indexes.
package generator_domain
