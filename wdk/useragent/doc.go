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

// Package useragent classifies and bounds HTTP User-Agent headers.
//
// [Classify] derives a coarse [Classification] from a raw header, identifying the browser
// family and major version, operating-system family, device form factor, and whether the
// agent identifies itself as a bot. It recognises common tokens only and leaves a field
// empty rather than guessing.
//
// [Clamp] and [ClampTo] shorten an over-long header without splitting a UTF-8 rune, so a
// value can be logged or stored within a fixed budget ([MaxLength] by default).
//
// All functions are pure and safe for concurrent use.
package useragent
