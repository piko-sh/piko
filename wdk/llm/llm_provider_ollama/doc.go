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

// Package llm_provider_ollama provides an Ollama LLM and embedding
// provider adapter.
//
// The provider implements both completions and embeddings against
// a locally-hosted Ollama server. By default it will auto-start
// the Ollama binary if the server is unreachable, and auto-pull
// models before first use.
//
// # Supply chain verification
//
// Models can be pinned to a specific digest using [ModelWithDigest]:
//
//	provider, err := llm_provider_ollama.NewOllamaProvider(
//	    llm_provider_ollama.Config{
//	        DefaultModel: llm_provider_ollama.ModelWithDigest("llama3.2", "a8b0c5157701"),
//	    },
//	)
//
// When a digest is set, the provider verifies the locally installed
// model's digest matches before using it. A mismatch returns an
// error indicating a possible supply chain compromise.
//
// # Thread safety
//
// All methods are safe for concurrent use.
package llm_provider_ollama
