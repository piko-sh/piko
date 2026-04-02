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

package cache_domain

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"

	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/logger/logger_domain"
)

// CacheTransformerPort defines the interface for cache value transformers.
// Unlike storage transformers which work with io.Reader streams, cache
// transformers work directly with byte slices for efficiency with in-memory
// cache values.
type CacheTransformerPort interface {
	// Name returns the unique name of this transformer (e.g. "zstd",
	// "crypto-service").
	Name() string

	// Type returns the category of this transformer, such as compression,
	// encryption, or custom.
	Type() cache_dto.TransformerType

	// Priority returns the order in which this processor runs.
	//
	// Returns int where lower numbers run first during Set operations.
	// Suggested values: Compression 100-199, Encryption 200-299, Custom 300+.
	Priority() int

	// Transform applies the forward transformation such as compress or encrypt.
	// It is used during Set operations.
	//
	// Takes input ([]byte) which is the data to transform.
	// Takes options (any) which is sourced from
	// TransformConfig.TransformerOptions[Name()].
	//
	// Returns []byte which is the transformed data.
	// Returns error when the transformation fails.
	Transform(ctx context.Context, input []byte, options any) ([]byte, error)

	// Reverse applies the reverse transformation such as decompress or decrypt.
	// It is used during Get operations.
	//
	// Takes input ([]byte) which is the data to transform.
	// Takes options (any) which is sourced from
	// TransformConfig.TransformerOptions[Name()].
	//
	// Returns []byte which is the transformed data.
	// Returns error when the reverse transformation fails.
	Reverse(ctx context.Context, input []byte, options any) ([]byte, error)
}

// TransformerRegistry holds a set of named cache transformers.
type TransformerRegistry struct {
	// transformers maps transformer names to their implementations.
	transformers map[string]CacheTransformerPort
}

// NewTransformerRegistry creates a new, empty transformer registry.
//
// Returns *TransformerRegistry which is ready for registering transformers.
func NewTransformerRegistry() *TransformerRegistry {
	return &TransformerRegistry{
		transformers: make(map[string]CacheTransformerPort),
	}
}

// Register adds a new transformer to the registry.
//
// Takes ctx (context.Context) which carries logging context for trace and
// request ID propagation.
// Takes transformer (CacheTransformerPort) which is the transformer to add.
//
// Returns error when transformer is nil, has an empty name, or a transformer
// with the same name is already registered.
func (r *TransformerRegistry) Register(ctx context.Context, transformer CacheTransformerPort) error {
	if transformer == nil {
		return errTransformerNil
	}
	name := transformer.Name()
	if name == "" {
		return errTransformerNameEmpty
	}
	if _, exists := r.transformers[name]; exists {
		return fmt.Errorf("transformer '%s' is already registered", name)
	}
	r.transformers[name] = transformer
	_, l := logger_domain.From(ctx, log)
	l.Internal("Registered cache transformer", logger_domain.String("transformer_name", name))
	return nil
}

// Get retrieves a transformer by its registered name.
//
// Takes name (string) which specifies the transformer to retrieve.
//
// Returns CacheTransformerPort which is the requested transformer.
// Returns error when the transformer name is not found in the registry.
func (r *TransformerRegistry) Get(name string) (CacheTransformerPort, error) {
	transformer, ok := r.transformers[name]
	if !ok {
		return nil, fmt.Errorf("transformer '%s' not found", name)
	}
	return transformer, nil
}

// Has checks if a transformer with the given name is registered.
//
// Takes name (string) which is the transformer name to look up.
//
// Returns bool which is true if the transformer exists, false otherwise.
func (r *TransformerRegistry) Has(name string) bool {
	_, ok := r.transformers[name]
	return ok
}

// GetNames returns a sorted list of all registered transformer names.
//
// Returns []string which contains the transformer names in alphabetical order.
func (r *TransformerRegistry) GetNames() []string {
	names := make([]string, 0, len(r.transformers))
	for name := range r.transformers {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

// TransformerChain represents an ordered sequence of transformers to be applied
// to cache values.
type TransformerChain struct {
	// config holds the options for each transformer in the chain.
	config *cache_dto.TransformConfig

	// transformers holds the ordered list of transformers to apply.
	transformers []CacheTransformerPort
}

// NewTransformerChain creates and sorts a new transformer chain based on a
// configuration.
//
// Takes registry (*TransformerRegistry) which provides the available
// transformers.
// Takes config (*cache_dto.TransformConfig) which specifies which transformers
// to enable.
//
// Returns *TransformerChain which contains the sorted transformers ready for
// use.
// Returns error when the registry is nil or a transformer cannot be found.
func NewTransformerChain(registry *TransformerRegistry, config *cache_dto.TransformConfig) (*TransformerChain, error) {
	if registry == nil {
		return nil, errors.New("transformer registry cannot be nil")
	}
	if config == nil {
		return &TransformerChain{}, nil
	}

	chain := &TransformerChain{
		transformers: make([]CacheTransformerPort, 0, len(config.EnabledTransformers)),
		config:       config,
	}

	for _, name := range config.EnabledTransformers {
		transformer, err := registry.Get(name)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve transformer '%s': %w", name, err)
		}
		chain.transformers = append(chain.transformers, transformer)
	}

	slices.SortFunc(chain.transformers, func(a, b CacheTransformerPort) int {
		return cmp.Compare(a.Priority(), b.Priority())
	})

	return chain, nil
}

// IsEmpty returns true if the chain contains no transformers.
//
// Returns bool which is true when the chain has no transformers.
func (c *TransformerChain) IsEmpty() bool {
	return len(c.transformers) == 0
}

// GetTransformerNames returns the ordered list of transformer names in this
// chain.
//
// Returns []string which contains the names in execution order.
func (c *TransformerChain) GetTransformerNames() []string {
	names := make([]string, len(c.transformers))
	for i, t := range c.transformers {
		names[i] = t.Name()
	}
	return names
}

// Transform applies all transformers in forward priority order for Set
// operations.
//
// Data flows: original -> transformer[0] -> transformer[1] -> ... -> storage.
// If the chain is empty, the input is returned unchanged.
//
// Takes input ([]byte) which is the data to transform.
//
// Returns []byte which is the transformed data after all transformers run.
// Returns error when any transformer in the chain fails.
func (c *TransformerChain) Transform(ctx context.Context, input []byte) ([]byte, error) {
	if c.IsEmpty() {
		return input, nil
	}

	ctx, l := logger_domain.From(ctx, log)

	current := input
	for _, transformer := range c.transformers {
		options := c.config.TransformerOptions[transformer.Name()]
		transformed, err := transformer.Transform(ctx, current, options)
		if err != nil {
			return nil, fmt.Errorf("transformer '%s' failed: %w", transformer.Name(), err)
		}
		current = transformed
		l.Trace("Applied cache transformer",
			logger_domain.String("transformer", transformer.Name()),
			logger_domain.Int("priority", transformer.Priority()),
			logger_domain.Int("input_size", len(input)),
			logger_domain.Int("output_size", len(current)))
	}
	return current, nil
}

// Reverse applies all transformers in reverse priority order for Get
// operations. Data flows from storage through each transformer back to the
// original format.
//
// Takes input ([]byte) which contains the data to reverse transform.
//
// Returns []byte which contains the data after all reverse transformations.
// Returns error when any transformer in the chain fails to reverse.
func (c *TransformerChain) Reverse(ctx context.Context, input []byte) ([]byte, error) {
	if c.IsEmpty() {
		return input, nil
	}

	ctx, l := logger_domain.From(ctx, log)

	current := input
	for i := len(c.transformers) - 1; i >= 0; i-- {
		transformer := c.transformers[i]
		options := c.config.TransformerOptions[transformer.Name()]
		reversed, err := transformer.Reverse(ctx, current, options)
		if err != nil {
			return nil, fmt.Errorf("transformer '%s' reverse failed: %w", transformer.Name(), err)
		}
		current = reversed
		l.Trace("Reversed cache transformer",
			logger_domain.String("transformer", transformer.Name()),
			logger_domain.Int("priority", transformer.Priority()),
			logger_domain.Int("input_size", len(input)),
			logger_domain.Int("output_size", len(current)))
	}
	return current, nil
}

// TransformedValue wraps a cache value with transformation metadata.
// This enables automatic reversal on Get operations even when the transform
// configuration changes between Set and Get.
type TransformedValue struct {
	// Data holds the transformed bytes after compression and encryption.
	Data []byte `json:"data"`

	// Transformers lists the names of transformers that were applied, in order.
	// This list is used to reverse the changes when data is retrieved.
	Transformers []string `json:"transformers"`

	// Version is the schema version; allows for future format changes.
	Version int `json:"version"`
}

// NewTransformedValue creates a wrapped value with transformation metadata.
//
// Takes data ([]byte) which contains the raw value to wrap.
// Takes transformers ([]string) which lists the transformations to apply.
//
// Returns *TransformedValue which holds the data with its transformation
// metadata and version set to 1.
func NewTransformedValue(data []byte, transformers []string) *TransformedValue {
	return &TransformedValue{
		Data:         data,
		Transformers: transformers,
		Version:      1,
	}
}

// Marshal serialises the TransformedValue to JSON bytes for storage.
//
// Returns []byte which contains the JSON-encoded representation.
// Returns error when JSON serialisation fails.
func (tv *TransformedValue) Marshal() ([]byte, error) {
	return CacheAPI.Marshal(tv)
}

// UnmarshalTransformedValue parses a TransformedValue from JSON bytes.
//
// Takes data ([]byte) which contains the JSON-encoded TransformedValue.
//
// Returns *TransformedValue which is the parsed value.
// Returns error when the JSON data is malformed or cannot be parsed.
func UnmarshalTransformedValue(data []byte) (*TransformedValue, error) {
	var tv TransformedValue
	if err := CacheAPI.Unmarshal(data, &tv); err != nil {
		return nil, fmt.Errorf("failed to unmarshal transformed value: %w", err)
	}
	return &tv, nil
}

// IsTransformedValue checks if the given bytes represent a TransformedValue
// by attempting to unmarshal it and checking for the presence of a positive
// version field.
//
// Takes data ([]byte) which contains the JSON data to check.
//
// Returns bool which is true if the data has a version field greater than zero.
func IsTransformedValue(data []byte) bool {
	var check struct {
		Version int `json:"version"`
	}
	if err := CacheAPI.Unmarshal(data, &check); err != nil {
		return false
	}
	return check.Version > 0
}
