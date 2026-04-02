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

// Package generator_schema provides versioned serialisation for the
// generator hexagon's project manifest FlatBuffer representation.
//
// It embeds the manifest.fbs schema file and computes a SHA-256 hash
// at init time. This hash is prepended to every serialised manifest
// payload so that the cache is automatically invalidated whenever
// the schema evolves. The sub-package generator_schema_gen contains
// the generated FlatBuffer types that define the binary format for
// the project manifest -- the runtime "phone book" describing a
// compiled Piko application's structure, routing, and assets.
package generator_schema
