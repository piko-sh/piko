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

// Package config_provider_piko provides access to the Piko framework's
// own configuration types.
//
// Re-exports the framework's internal configuration structs so that downstream
// applications can reference them directly. Useful when you need to extend
// [ServerConfig], inspect framework settings for compatibility, or build
// tooling that interacts with Piko's configuration layer.
//
// For most applications that define their own configuration, the parent
// [config] package and its [config.Load] function are the recommended
// entry point.
//
// # Usage
//
//	import "piko.sh/piko/wdk/config/config_provider_piko"
//
//	provider := config_provider_piko.NewConfigProvider()
//
// # Integration
//
// Wraps [internal/config] as a stable public surface. Use the parent config
// package for loading your own application configuration; reach for
// config_provider_piko only when you need direct access to the framework's
// configuration types.
package config_provider_piko
