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

// Package llm_provider_grok provides an LLM provider adapter for
// xAI's Grok API.
//
// Grok uses an OpenAI-compatible wire protocol, so this package
// delegates to the OpenAI provider internally while adding
// Grok-specific model filtering, error wrapping, and
// observability. Grok does not offer embedding models, so only
// completions are supported.
//
// # Thread safety
//
// All methods are safe for concurrent use.
package llm_provider_grok
