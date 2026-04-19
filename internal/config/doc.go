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

// Package config holds piko's internal value-type definitions for the
// configuration domains the framework cares about (network, security,
// paths, storage, database, OTLP, build, and so on). These are plain Go
// structs consumed by With* options on piko.New and by the bootstrap
// pipeline; they are not part of piko's public API.
//
// Users never reference these types directly. Configure the framework via
// With* functional options passed to piko.New. The sole env-var carve-out
// is PIKO_LOG_LEVEL, which is read directly in piko.New and applied via
// WithLogLevel before user options are processed.
//
// # Precedence
//
// During bootstrap (highest to lowest):
//
//  1. Programmatic overrides (individual With* options)
//  2. Resolved placeholder strings (e.g. "aws-secret:my/key")
//  3. Programmatic defaults
//  4. Struct-tag defaults
//
// No file or env loading happens at the framework level. If you want
// file/env-driven configuration for your own application, use the
// user-facing utility at [piko.sh/piko/wdk/config] which exposes the
// underlying loader machinery.
package config
