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
	"reflect"
	"slices"
	"sync"

	"piko.sh/piko/internal/logger/logger_domain"
)

// EncodingRegistry maps Go types to their corresponding encoders.
// It provides runtime type-driven encoder selection, enabling providers to
// automatically choose the correct encoding strategy based on a value's type.
//
// Thread-safe for concurrent access.
type EncodingRegistry struct {
	// encoders maps Go types to their registered encoders.
	encoders map[reflect.Type]AnyEncoder

	// defaultEncoder is the fallback encoder used when no type-specific
	// encoder matches; nil means no default is configured.
	defaultEncoder AnyEncoder

	// mu guards the encoders map for safe concurrent access.
	mu sync.RWMutex
}

// NewEncodingRegistry creates a new registry with an optional default
// encoder. The default encoder is used as a fallback when no
// type-specific encoder is registered.
//
// Takes defaultEncoder (AnyEncoder) which provides the fallback
// encoder for unregistered types, or nil for no default.
//
// Returns *EncodingRegistry which is the configured registry ready for
// use.
func NewEncodingRegistry(defaultEncoder AnyEncoder) *EncodingRegistry {
	return &EncodingRegistry{
		encoders:       make(map[reflect.Type]AnyEncoder),
		defaultEncoder: defaultEncoder,
	}
}

// Register adds an encoder for a specific type to the registry.
//
// Takes ctx (context.Context) which carries logging context for trace and
// request ID propagation.
// Takes encoder (AnyEncoder) which provides the type-specific encoding logic.
//
// Returns error when encoder is nil, handles no concrete type, or an encoder
// for the same type is already registered.
//
// Safe for concurrent use.
func (r *EncodingRegistry) Register(ctx context.Context, encoder AnyEncoder) error {
	if encoder == nil {
		return errEncoderNil
	}

	t := encoder.HandlesType()
	if t == nil {
		return errEncoderNoType
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.encoders[t]; exists {
		return fmt.Errorf("an encoder for type %s is already registered", t.String())
	}

	r.encoders[t] = encoder
	_, l := logger_domain.From(ctx, log)
	l.Internal("Registered encoder for type",
		logger_domain.String("type", t.String()),
		logger_domain.Int("encoder_count", len(r.encoders)))

	return nil
}

// Get finds the appropriate encoder for a given value. It first checks for
// a type-specific registration, then falls back to the default encoder.
//
// This is the primary method used by providers during Set operations.
//
// Takes value (any) which is the value whose type determines the encoder.
//
// Returns AnyEncoder which is the matched or default encoder.
// Returns error when value is nil, or when no type-specific match exists and
// no default encoder is configured.
//
// Safe for concurrent use; protected by a read lock.
func (r *EncodingRegistry) Get(value any) (AnyEncoder, error) {
	t := reflect.TypeOf(value)
	if t == nil {
		return nil, errors.New("cannot encode nil value")
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	if encoder, ok := r.encoders[t]; ok {
		return encoder, nil
	}

	if r.defaultEncoder != nil {
		return r.defaultEncoder, nil
	}

	return nil, fmt.Errorf("no encoder found for type %s and no default encoder configured", t.String())
}

// GetByType finds an encoder by its reflect.Type.
//
// This is used during Get operations when the type is known but the value
// has not been unmarshalled yet.
//
// Takes t (reflect.Type) which specifies the type to find an encoder for.
//
// Returns AnyEncoder which handles encoding for the given type.
// Returns error when t is nil, or no encoder is registered for the type
// and no default encoder is configured.
//
// Safe for concurrent use; protected by a read lock.
func (r *EncodingRegistry) GetByType(t reflect.Type) (AnyEncoder, error) {
	if t == nil {
		return nil, errors.New("type cannot be nil")
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	if encoder, ok := r.encoders[t]; ok {
		return encoder, nil
	}

	if r.defaultEncoder != nil {
		return r.defaultEncoder, nil
	}

	return nil, fmt.Errorf("no encoder found for type %s and no default encoder configured", t.String())
}

// Has checks if an encoder for the given type is explicitly registered,
// not counting the default encoder.
//
// Takes t (reflect.Type) which specifies the type to look up.
//
// Returns bool which is true if an encoder is registered for the type.
//
// Safe for concurrent use.
func (r *EncodingRegistry) Has(t reflect.Type) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.encoders[t]
	return ok
}

// RegisteredTypes returns a sorted list of all explicitly registered types.
// The default encoder's type is not included in this list.
//
// Returns []reflect.Type which contains the registered types sorted by name.
//
// Safe for concurrent use.
func (r *EncodingRegistry) RegisteredTypes() []reflect.Type {
	r.mu.RLock()
	defer r.mu.RUnlock()

	types := make([]reflect.Type, 0, len(r.encoders))
	for t := range r.encoders {
		types = append(types, t)
	}

	slices.SortFunc(types, func(a, b reflect.Type) int {
		return cmp.Compare(a.String(), b.String())
	})

	return types
}

// Count returns the number of explicitly registered encoders, not counting
// the default.
//
// Returns int which is the count of registered encoders.
//
// Safe for concurrent use.
func (r *EncodingRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.encoders)
}

// HasDefault returns true if a default encoder is configured.
//
// Returns bool which indicates whether a default encoder exists.
//
// Safe for concurrent use.
func (r *EncodingRegistry) HasDefault() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.defaultEncoder != nil
}
