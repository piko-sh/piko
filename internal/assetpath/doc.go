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

// Package assetpath handles predicates and transformations for asset source
// paths used throughout the Piko asset pipeline.
//
// Source paths in templates and configuration files (e.g. piko:img src,
// favicon src) must be transformed into served URLs before rendering. This
// package centralises that logic so the compiler, render, and bootstrap
// layers do not duplicate it.
//
// # Thread safety
//
// All functions are pure and safe for concurrent use.
package assetpath
