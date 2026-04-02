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

package llm_provider_voyage

import (
	"net/http"

	"piko.sh/piko/wdk/llm"
)

// VoyageProvider wraps the internal Voyage provider. It implements
// llm.EmbeddingProviderPort via embedding.
type VoyageProvider struct {
	*voyageProvider
}

var _ llm.EmbeddingProviderPort = (*VoyageProvider)(nil)

// NewVoyageProvider creates a new Voyage AI embedding provider with the given
// configuration.
//
// The returned provider implements llm.EmbeddingProviderPort only - it does
// not provide chat completions. Register it alongside a separate LLM provider
// using [piko.WithEmbeddingProvider]:
//
//	provider, err := llm_provider_voyage.NewVoyageProvider(
//	    llm_provider_voyage.Config{
//	        APIKey: os.Getenv("VOYAGE_API_KEY"),
//	    },
//	)
//
// Takes config (Config) which contains the provider configuration.
//
// Returns *VoyageProvider which can be registered with the LLM service.
// Returns error when the configuration is invalid.
func NewVoyageProvider(config Config) (*VoyageProvider, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	config = config.WithDefaults()

	p := &voyageProvider{
		client: &http.Client{
			Timeout: httpClientTimeout,
		},
		config:              config,
		defaultModel:        config.DefaultModel,
		embeddingDimensions: config.EmbeddingDimensions,
	}

	return &VoyageProvider{voyageProvider: p}, nil
}
