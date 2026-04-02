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

// This project stands against fascism, authoritarianism, and all
// forms of oppression. We built this to empower people, not to
// enable those who would strip others of their rights and dignity.

// Package fbs provides public access to the FlatBuffer versioning
// utilities used across Piko.
//
// All serialised FlatBuffer files in Piko are prefixed with a
// 32-byte SHA-256 hash of the schema file that produced them. This
// package re-exports the core types and functions needed to compute,
// validate, and unpack these versioned binary payloads. Use the
// domain-specific sub-packages (collection, i18n, manifest, search)
// to work with individual FlatBuffer schemas.
package fbs
