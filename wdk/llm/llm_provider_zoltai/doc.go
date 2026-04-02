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

// Package llm_provider_zoltai provides a fake LLM provider that
// returns random predefined fortunes. It is a drop-in replacement
// for real providers when developing locally without calling
// external APIs or running a local model server.
//
// Zoltai implements both completions and embeddings, so it can be
// used anywhere a real provider would be wired in. It ignores all
// request parameters and always returns a single fortune.
// Embeddings are not semantically meaningful but are deterministic
// -- the same input text always produces the same vector.
//
// # Thread safety
//
// All methods are safe for concurrent use.
package llm_provider_zoltai
