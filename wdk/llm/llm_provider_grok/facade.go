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

package llm_provider_grok

import "piko.sh/piko/wdk/llm"

// NewGrokProvider creates a new Grok LLM provider with the given
// configuration.
//
// Takes config (Config) which contains the provider configuration including API
// key.
//
// Returns llm.ProviderPort which can be registered with the LLM service.
// Returns error when the configuration is invalid.
func NewGrokProvider(config Config) (llm.ProviderPort, error) {
	return New(config)
}
