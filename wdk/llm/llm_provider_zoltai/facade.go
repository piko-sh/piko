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

package llm_provider_zoltai

import "piko.sh/piko/wdk/llm"

// ZoltaiProvider wraps the internal Zoltai provider and exposes both
// llm.ProviderPort and llm.EmbeddingProviderPort.
type ZoltaiProvider struct {
	*zoltaiProvider
}

var (
	_ llm.ProviderPort = (*ZoltaiProvider)(nil)

	_ llm.EmbeddingProviderPort = (*ZoltaiProvider)(nil)
)

// NewZoltaiProvider creates a new Zoltai fake LLM provider.
//
// The returned provider implements both llm.ProviderPort (completions) and
// llm.EmbeddingProviderPort (embeddings). Register it with both
// [llm.Service.RegisterProvider] and [llm.Service.RegisterEmbeddingProvider]
// to use it for all LLM operations.
//
// Takes config (Config) which contains the provider configuration.
//
// Returns *ZoltaiProvider which can be registered with the LLM service.
// Returns error when the configuration is invalid (currently never).
func NewZoltaiProvider(config Config) (*ZoltaiProvider, error) {
	p, err := newProvider(config)
	if err != nil {
		return nil, err
	}
	return &ZoltaiProvider{zoltaiProvider: p}, nil
}
