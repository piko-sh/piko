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

// Package pikotest_dto defines pure data structures used by the pikotest
// testing framework. These types are shared across package boundaries and
// contain no business logic.
//
// Key types:
//
//   - [BuildASTFunc]: The function signature that compiled components expose
//     for testing
//   - [ComponentOption]: Functional option type for configuring ComponentTester
//   - [ComponentConfig]: Configuration data for ComponentTester setup
//   - [ActionResult]: Data container for server action invocation results
package pikotest_dto
