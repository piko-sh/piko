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

const (
	// defaultModel is the model name used when none is specified.
	defaultModel = "zoltai-1"

	// defaultEmbeddingModel is the embedding model name used when none is
	// specified.
	defaultEmbeddingModel = "zoltai-embed-1"

	// defaultEmbeddingDimension is the vector size for fake embeddings.
	defaultEmbeddingDimension = 384
)

// Config holds settings for the Zoltai fake provider.
type Config struct {
	// FormatFortune formats a selected fortune into the full
	// response text, receiving the raw fortune string and returning
	// the formatted output (defaults to wrapping with the Zoltai
	// preamble and postamble when nil).
	FormatFortune func(fortune string) string

	// DefaultModel is the model name reported in responses.
	// Defaults to "zoltai-1" if empty.
	DefaultModel string

	// DefaultEmbeddingModel is the model name reported in embedding responses.
	// Defaults to "zoltai-embed-1" if empty.
	DefaultEmbeddingModel string

	// Fortunes is the pool of responses that Zoltai randomly selects from.
	// Defaults to the built-in fortunes when nil or empty.
	Fortunes []string

	// EmbeddingDimensions is the vector size for fake embeddings.
	// Defaults to 384 if zero.
	EmbeddingDimensions int

	// Seed sets a fixed seed for reproducible fortune selection.
	// Zero means a random seed is chosen at construction time.
	Seed int64
}

// Validate checks that the configuration is valid.
// Zoltai requires no external credentials, so this always returns nil.
//
// Returns error which is always nil.
func (*Config) Validate() error {
	return nil
}

// WithDefaults returns a copy of the config with default values applied.
//
// Returns Config with any missing fields set to their default values.
func (c Config) WithDefaults() Config {
	if c.DefaultModel == "" {
		c.DefaultModel = defaultModel
	}
	if c.DefaultEmbeddingModel == "" {
		c.DefaultEmbeddingModel = defaultEmbeddingModel
	}
	if c.EmbeddingDimensions <= 0 {
		c.EmbeddingDimensions = defaultEmbeddingDimension
	}
	if len(c.Fortunes) == 0 {
		c.Fortunes = fortunes
	}
	if c.FormatFortune == nil {
		c.FormatFortune = formatFortune
	}
	return c
}
