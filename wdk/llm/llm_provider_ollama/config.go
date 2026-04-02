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

package llm_provider_ollama

import (
	"fmt"
	"strings"
	"time"
)

const (
	// defaultHost is the default host URL for the Ollama service.
	defaultHost = "http://localhost:11434"

	// defaultModel is the default LLM model used when none is specified.
	defaultModel = "llama3.2"

	// defaultEmbeddingModel is the default model used for generating embeddings.
	defaultEmbeddingModel = "all-minilm"

	// defaultHTTPTimeout is the default timeout for non-streaming HTTP
	// requests to the Ollama API.
	defaultHTTPTimeout = 10 * time.Minute

	// defaultImageFetchMaxBytes is the maximum allowed image size when
	// fetching URL-referenced images (20 MiB).
	defaultImageFetchMaxBytes = 20 << 20

	// defaultImageFetchTimeout is the per-image fetch timeout.
	defaultImageFetchTimeout = 30 * time.Second
)

// ModelRef identifies an Ollama model by name with an optional SHA256 digest
// for supply chain verification. When a Digest is set, the provider verifies
// that the local model's digest matches before using it, preventing
// substitution attacks.
//
// Use [Model] for a plain name or [ModelWithDigest] to pin to a specific
// version.
type ModelRef struct {
	// Name is the Ollama model name (e.g. "llama3.2", "nomic-embed-text").
	Name string

	// Digest is an optional SHA256 digest to verify against the
	// locally installed model, refusing a mismatch to prevent
	// substitution attacks.
	//
	// Obtain the digest from `ollama list` or the Ollama registry.
	Digest string
}

// String returns the model name.
//
// Returns string which is the name of the model.
func (m ModelRef) String() string {
	return m.Name
}

// IsZero reports whether the reference is empty (no name set).
//
// Returns bool which is true when the reference has no name set.
func (m ModelRef) IsZero() bool {
	return m.Name == ""
}

// verifyDigest checks that gotDigest matches the expected digest in the ref.
// Matching is prefix-based, so truncated digests work (as shown by
// `ollama list`).
//
// Takes gotDigest (string) which is the actual digest to compare.
//
// Returns error when the digests do not match, indicating a possible supply
// chain compromise.
func (m ModelRef) verifyDigest(gotDigest string) error {
	if m.Digest == "" {
		return nil
	}

	want := strings.TrimPrefix(m.Digest, "sha256:")
	got := strings.TrimPrefix(gotDigest, "sha256:")

	if !strings.HasPrefix(got, want) && !strings.HasPrefix(want, got) {
		return fmt.Errorf(
			"model %q digest mismatch: expected %s, got %s (possible supply chain compromise)",
			m.Name, m.Digest, gotDigest,
		)
	}

	return nil
}

// ImageFetchConfig controls how the Ollama provider fetches URL-referenced
// images. This must be explicitly enabled because it causes the provider to
// make outbound HTTP requests to arbitrary URLs supplied in messages.
type ImageFetchConfig struct {
	// MaxBytes is the maximum allowed image size in bytes.
	// Defaults to 20 MiB when zero.
	MaxBytes int64

	// Timeout is the per-image fetch timeout.
	// Defaults to 30 seconds when zero.
	Timeout time.Duration
}

// withDefaults returns a copy with default values applied to zero fields.
//
// Returns ImageFetchConfig which has zero-valued fields replaced with defaults.
func (c ImageFetchConfig) withDefaults() ImageFetchConfig {
	if c.MaxBytes == 0 {
		c.MaxBytes = defaultImageFetchMaxBytes
	}
	if c.Timeout == 0 {
		c.Timeout = defaultImageFetchTimeout
	}
	return c
}

// Config holds settings for the Ollama provider.
type Config struct {
	// AutoStart spawns `ollama serve` as a managed subprocess if the
	// server is not reachable on startup, defaulting to true.
	AutoStart *bool

	// AutoPull downloads models via Ollama's Pull API if they are not found
	// locally. Defaults to true.
	AutoPull *bool

	// ImageFetch configures optional downloading of URL-referenced images,
	// disabled by default for security so that nil causes image URL content
	// parts to be silently skipped.
	ImageFetch *ImageFetchConfig

	// DefaultModel is the model to use for completions when not given in
	// requests. Defaults to "llama3.2".
	DefaultModel ModelRef

	// DefaultEmbeddingModel is the model to use for embeddings when not given
	// in requests. Defaults to "all-minilm".
	DefaultEmbeddingModel ModelRef

	// Host is the Ollama API endpoint. Defaults to "http://localhost:11434".
	Host string

	// BinaryPath is the path to the ollama binary. If empty, the binary is
	// auto-detected from $PATH.
	BinaryPath string

	// HTTPTimeout is the timeout for non-streaming HTTP requests to the
	// Ollama API, defaulting to 10 minutes. Streaming calls are
	// unaffected as they use per-request context cancellation.
	HTTPTimeout time.Duration
}

// Validate checks that the configuration is valid.
//
// Returns error when required fields are missing.
func (*Config) Validate() error {
	return nil
}

// WithDefaults returns a copy of the config with default values applied.
//
// Returns Config with any missing fields set to their default values.
func (c Config) WithDefaults() Config {
	if c.Host == "" {
		c.Host = defaultHost
	}
	if c.DefaultModel.IsZero() {
		c.DefaultModel = Model(defaultModel)
	}
	if c.DefaultEmbeddingModel.IsZero() {
		c.DefaultEmbeddingModel = Model(defaultEmbeddingModel)
	}
	if c.AutoStart == nil {
		c.AutoStart = new(true)
	}
	if c.AutoPull == nil {
		c.AutoPull = new(true)
	}
	if c.HTTPTimeout == 0 {
		c.HTTPTimeout = defaultHTTPTimeout
	}
	if c.ImageFetch != nil {
		c.ImageFetch = new(c.ImageFetch.withDefaults())
	}
	return c
}

// Model creates a [ModelRef] from a plain model name.
//
// Takes name (string) which specifies the model name.
//
// Returns ModelRef which contains the specified model name.
func Model(name string) ModelRef {
	return ModelRef{Name: name}
}

// ModelWithDigest creates a [ModelRef] pinned to a specific SHA256 digest.
//
// Takes name (string) which specifies the model name.
// Takes digest (string) which is the SHA256 digest to pin the model to.
//
// Returns ModelRef which is the model reference pinned to the specified digest.
//
// The digest should be the value shown in `ollama list` (e.g.
// "1b226e2802db").
func ModelWithDigest(name, digest string) ModelRef {
	return ModelRef{Name: name, Digest: digest}
}
