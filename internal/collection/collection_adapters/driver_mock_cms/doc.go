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

// Package driver_mock_cms simulates a headless CMS collection provider
// for testing and demonstration.
//
// It supplies both build-time and runtime provider implementations,
// doubling as a reference for building dynamic collection providers
// and enabling testing without external CMS dependencies.
//
// The build-time provider generates Go AST for runtime fetcher functions,
// while the runtime provider serves in-memory mock data.
//
// # Thread safety
//
// Both provider types are safe for concurrent use.
package driver_mock_cms
