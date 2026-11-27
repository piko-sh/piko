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

// Package pipitpending skips tests that need a pipit release newer than the one Piko
// pins.
//
// Call [Skip] at the narrowest level, a subtest or case, with the pipit fix the test
// needs. Every skip message starts with [SkipPrefix], so the pending tests can be found
// with one search when the pin moves. Setting [LocalEnv] to 1 runs them against a local
// Pipit checkout instead; [Local] reports that mode to code that must adapt to it.
package pipitpending
