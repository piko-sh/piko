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

// Package dalcore holds the dialect-agnostic registry DAL implementation.
//
// [Core] owns all FlatBuffer serialisation, transaction lifecycle, and domain mapping
// that is identical across SQL dialects. It delegates every touch of a generated query to
// a per-dialect [Driver], whose concrete implementation lives in each querier_<dialect>
// package. This keeps the shared body in one place while each dialect's generated db
// package is imported by exactly one thin driver.
package dalcore
