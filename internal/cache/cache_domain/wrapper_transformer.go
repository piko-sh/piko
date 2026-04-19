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
	"context"
	"fmt"
	"iter"
	"maps"
	"time"

	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/internal/logger/logger_domain"
)

const (
	// logKeyKey is the log field name for cache entry keys.
	logKeyKey = "key"

	// formatAnyValue is the format string for logging keys of any type.
	formatAnyValue = "%v"
)

// wrapperConfig holds the settings for the transformerWrapper.
type wrapperConfig[K comparable, V any] struct {
	// encoder converts values to and from their stored format.
	encoder EncoderPort[V]
}

var _ Cache[any, any] = (*transformerWrapper[any, any])(nil)

// wrapperOption is a functional option for setting up a transformerWrapper.
type wrapperOption[K comparable, V any] func(*wrapperConfig[K, V])

// transformerWrapper wraps a cache provider to add transparent value
// transformation such as compression or encryption. It implements io.Closer
// and intercepts Set and Get operations to apply transformations automatically.
type transformerWrapper[K comparable, V any] struct {
	// provider is the wrapped storage provider for byte slices.
	provider ProviderPort[K, []byte]

	// registry holds the transformers used to build transformation chains.
	registry *TransformerRegistry

	// config holds the settings used to transform values.
	config *cache_dto.TransformConfig

	// encoder provides value encoding; defaults to JSON.
	encoder EncoderPort[V]
}

// encodeValue converts a generic value V to bytes using the configured
// encoder. Falls back to JSON for backward compatibility if no encoder
// was configured via WithWrapperEncoder.
//
// Takes value (V) which is the value to encode.
//
// Returns []byte which contains the encoded representation of the value.
// Returns error when encoding fails.
func (w *transformerWrapper[K, V]) encodeValue(value V) ([]byte, error) {
	if w.encoder != nil {
		data, err := w.encoder.Marshal(value)
		if err != nil {
			return nil, fmt.Errorf("failed to encode cache value: %w", err)
		}
		return data, nil
	}

	data, err := CacheAPI.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("failed to encode cache value (JSON fallback): %w", err)
	}
	return data, nil
}

// decodeValue converts bytes back to the generic value V.
//
// If no encoder was set via WithWrapperEncoder, falls back to JSON for
// backward compatibility.
//
// Takes data ([]byte) which contains the encoded value to decode.
//
// Returns V which is the decoded value.
// Returns error when the data cannot be unmarshalled.
func (w *transformerWrapper[K, V]) decodeValue(data []byte) (V, error) {
	var value V

	if w.encoder != nil {
		if err := w.encoder.Unmarshal(data, &value); err != nil {
			return value, fmt.Errorf("failed to decode cache value: %w", err)
		}
		return value, nil
	}

	if err := CacheAPI.Unmarshal(data, &value); err != nil {
		return value, fmt.Errorf("failed to decode cache value (JSON fallback): %w", err)
	}
	return value, nil
}

// transformAndWrap applies transformations and wraps the result with metadata.
//
// Takes data ([]byte) which contains the raw data to transform.
//
// Returns []byte which contains the wrapped data with transformation metadata,
// or the original data if no transformers are configured.
// Returns error when the transformer chain cannot be created, transformation
// fails, or the wrapped value cannot be marshalled.
func (w *transformerWrapper[K, V]) transformAndWrap(ctx context.Context, data []byte) ([]byte, error) {
	if w.config == nil || len(w.config.EnabledTransformers) == 0 {
		return data, nil
	}

	ctx, l := logger_domain.From(ctx, log)

	chain, err := NewTransformerChain(w.registry, w.config)
	if err != nil {
		return nil, fmt.Errorf("failed to create transformer chain: %w", err)
	}

	if chain.IsEmpty() {
		return data, nil
	}

	transformed, err := chain.Transform(ctx, data)
	if err != nil {
		return nil, fmt.Errorf("transformation failed: %w", err)
	}

	wrapped := NewTransformedValue(transformed, chain.GetTransformerNames())
	wrappedBytes, err := wrapped.Marshal()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal transformed value: %w", err)
	}

	l.Trace("Transformed and wrapped cache value",
		logger_domain.Strings("transformers", chain.GetTransformerNames()),
		logger_domain.Int("original_size", len(data)),
		logger_domain.Int("transformed_size", len(wrappedBytes)))

	return wrappedBytes, nil
}

// unwrapAndReverse checks if data is transformed, unwraps it, and reverses
// the transformations to restore the original value.
//
// Takes data ([]byte) which contains the potentially transformed value.
//
// Returns []byte which is the original untransformed data.
// Returns error when unwrapping or reverse transformation fails.
func (w *transformerWrapper[K, V]) unwrapAndReverse(ctx context.Context, data []byte) ([]byte, error) {
	ctx, l := logger_domain.From(ctx, log)

	if !IsTransformedValue(data) {
		l.Trace("Retrieved untransformed cache value (backward compatibility)")
		return data, nil
	}

	wrapped, err := UnmarshalTransformedValue(data)
	if err != nil {
		return nil, fmt.Errorf("failed to unwrap transformed value: %w", err)
	}

	if len(wrapped.Transformers) == 0 {
		return wrapped.Data, nil
	}

	config := &cache_dto.TransformConfig{
		EnabledTransformers: wrapped.Transformers,
		TransformerOptions:  make(map[string]any),
	}

	if w.config != nil {
		maps.Copy(config.TransformerOptions, w.config.TransformerOptions)
	}

	chain, err := NewTransformerChain(w.registry, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create transformer chain for reversal: %w", err)
	}

	reversed, err := chain.Reverse(ctx, wrapped.Data)
	if err != nil {
		return nil, fmt.Errorf("reverse transformation failed: %w", err)
	}

	l.Trace("Unwrapped and reversed cache value transformations",
		logger_domain.Strings("transformers", wrapped.Transformers),
		logger_domain.Int("transformed_size", len(data)),
		logger_domain.Int("original_size", len(reversed)))

	return reversed, nil
}

// GetIfPresent retrieves a value if present, automatically reversing any
// transformations and decoding the value.
//
// Takes ctx (context.Context) for cancellation and timeout.
// Takes key (K) which identifies the entry to retrieve.
//
// Returns V which is the decoded value, or the zero value if not found.
// Returns bool which indicates whether the key was present.
// Returns error when the underlying provider or transformation fails.
func (w *transformerWrapper[K, V]) GetIfPresent(ctx context.Context, key K) (V, bool, error) {
	var zero V

	data, ok, err := goroutine.SafeCall2(ctx, "cache.GetIfPresent", func() ([]byte, bool, error) { return w.provider.GetIfPresent(ctx, key) })
	if err != nil {
		return zero, false, fmt.Errorf("getting value from provider: %w", err)
	}
	if !ok {
		return zero, false, nil
	}

	reversed, err := w.unwrapAndReverse(ctx, data)
	if err != nil {
		return zero, false, fmt.Errorf("failed to reverse transformations on cache get: %w", err)
	}

	value, err := w.decodeValue(reversed)
	if err != nil {
		return zero, false, fmt.Errorf("failed to decode cache value: %w", err)
	}

	return value, true, nil
}

// Get retrieves a value for a key, using the loader if the key is missing.
//
// The loader's result is automatically encoded and transformed before storage.
// Retrieved values are automatically reverse-transformed and decoded.
//
// Takes key (K) which identifies the cached value to retrieve.
// Takes loader (Loader) which provides the value when the key is not cached.
//
// Returns V which is the cached or newly loaded value.
// Returns error when loading, encoding, transforming, or decoding fails.
func (w *transformerWrapper[K, V]) Get(ctx context.Context, key K, loader cache_dto.Loader[K, V]) (V, error) {
	var zero V

	wrappedLoader := cache_dto.LoaderFunc[K, []byte](func(ctx context.Context, k K) ([]byte, error) {
		value, err := loader.Load(ctx, k)
		if err != nil {
			return nil, fmt.Errorf("loading value for cache key: %w", err)
		}

		data, err := w.encodeValue(value)
		if err != nil {
			return nil, fmt.Errorf("encoding loaded value: %w", err)
		}

		transformed, err := w.transformAndWrap(ctx, data)
		if err != nil {
			return nil, fmt.Errorf("transforming loaded value: %w", err)
		}

		return transformed, nil
	})

	data, err := goroutine.SafeCall1(ctx, "cache.Get", func() ([]byte, error) { return w.provider.Get(ctx, key, wrappedLoader) })
	if err != nil {
		return zero, fmt.Errorf("getting value from provider: %w", err)
	}

	reversed, err := w.unwrapAndReverse(ctx, data)
	if err != nil {
		return zero, fmt.Errorf("failed to reverse transformations: %w", err)
	}

	value, err := w.decodeValue(reversed)
	if err != nil {
		return zero, fmt.Errorf("failed to decode cache value: %w", err)
	}

	return value, nil
}

// Set stores a value for a key, automatically encoding and transforming it
// before storage.
//
// Takes ctx (context.Context) for cancellation and timeout.
// Takes key (K) which identifies the cache entry.
// Takes value (V) which is the data to store.
// Takes tags (...string) which allows for group-based invalidation.
//
// Returns error when encoding, transformation, or the underlying provider fails.
func (w *transformerWrapper[K, V]) Set(ctx context.Context, key K, value V, tags ...string) error {
	data, err := w.encodeValue(value)
	if err != nil {
		return fmt.Errorf("failed to encode cache value for Set: %w", err)
	}

	transformed, err := w.transformAndWrap(ctx, data)
	if err != nil {
		return fmt.Errorf("failed to transform cache value for Set: %w", err)
	}

	return goroutine.SafeCall(ctx, "cache.Set", func() error { return w.provider.Set(ctx, key, transformed, tags...) })
}

// SetWithTTL stores a value with a specific time-to-live, automatically
// encoding and transforming it before storage.
//
// Takes key (K) which identifies the cache entry.
// Takes value (V) which is the data to store.
// Takes ttl (time.Duration) which specifies how long the entry should live.
// Takes tags (...string) which allows for group-based invalidation.
//
// Returns error when encoding or transformation fails.
func (w *transformerWrapper[K, V]) SetWithTTL(ctx context.Context, key K, value V, ttl time.Duration, tags ...string) error {
	ctx, l := logger_domain.From(ctx, log)

	data, err := w.encodeValue(value)
	if err != nil {
		l.Error("Failed to encode cache value for SetWithTTL",
			logger_domain.Error(err))
		return fmt.Errorf("encoding failed: %w", err)
	}

	transformed, err := w.transformAndWrap(ctx, data)
	if err != nil {
		l.Error("Failed to transform cache value for SetWithTTL",
			logger_domain.Error(err))
		return fmt.Errorf("transformation failed: %w", err)
	}

	return goroutine.SafeCall(ctx, "cache.SetWithTTL", func() error { return w.provider.SetWithTTL(ctx, key, transformed, ttl, tags...) })
}

// BulkSet stores multiple key-value pairs, encoding and transforming each
// value before storage.
//
// Takes items (map[K]V) which contains the key-value pairs to store.
// Takes tags (...string) which apply to all items in the batch.
//
// Returns error when the underlying storage operation fails.
func (w *transformerWrapper[K, V]) BulkSet(ctx context.Context, items map[K]V, tags ...string) error {
	if len(items) == 0 {
		return nil
	}

	ctx, l := logger_domain.From(ctx, log)

	transformedItems := make(map[K][]byte, len(items))
	for key, value := range items {
		data, err := w.encodeValue(value)
		if err != nil {
			l.Warn("Failed to encode cache value for BulkSet, skipping",
				logger_domain.Error(err))
			continue
		}

		transformed, err := w.transformAndWrap(ctx, data)
		if err != nil {
			l.Warn("Failed to transform cache value for BulkSet, skipping",
				logger_domain.Error(err))
			continue
		}

		transformedItems[key] = transformed
	}

	return goroutine.SafeCall(ctx, "cache.BulkSet", func() error { return w.provider.BulkSet(ctx, transformedItems, tags...) })
}

// Invalidate removes an entry from the cache.
//
// Takes ctx (context.Context) for cancellation and timeout.
// Takes key (K) which identifies the entry to remove.
//
// Returns error when the underlying provider fails.
func (w *transformerWrapper[K, V]) Invalidate(ctx context.Context, key K) error {
	return goroutine.SafeCall(ctx, "cache.Invalidate", func() error { return w.provider.Invalidate(ctx, key) })
}

// InvalidateByTags passes through to the underlying provider.
//
// Takes ctx (context.Context) for cancellation and timeout.
// Takes tags (...string) which specifies the cache tags to invalidate.
//
// Returns int which is the number of entries invalidated.
// Returns error when the underlying provider fails.
func (w *transformerWrapper[K, V]) InvalidateByTags(ctx context.Context, tags ...string) (int, error) {
	return goroutine.SafeCall1(ctx, "cache.InvalidateByTags", func() (int, error) { return w.provider.InvalidateByTags(ctx, tags...) })
}

// InvalidateAll removes all entries from the underlying cache provider.
//
// Takes ctx (context.Context) for cancellation and timeout.
//
// Returns error when the underlying provider fails.
func (w *transformerWrapper[K, V]) InvalidateAll(ctx context.Context) error {
	return goroutine.SafeCall(ctx, "cache.InvalidateAll", func() error { return w.provider.InvalidateAll(ctx) })
}

// unwrapAndDecode unwraps transformations and decodes bytes to a
// value.
//
// Takes data ([]byte) which contains the wrapped and encoded data.
//
// Returns V which is the decoded value.
// Returns error when unwrapping or decoding fails.
func (w *transformerWrapper[K, V]) unwrapAndDecode(ctx context.Context, data []byte) (V, error) {
	var zero V
	reversed, err := w.unwrapAndReverse(ctx, data)
	if err != nil {
		return zero, fmt.Errorf("failed to reverse transform: %w", err)
	}
	value, err := w.decodeValue(reversed)
	if err != nil {
		return zero, fmt.Errorf("failed to decode: %w", err)
	}
	return value, nil
}

// encodeAndTransform converts a value to bytes and applies a transformation.
//
// Takes value (V) which is the value to convert.
//
// Returns []byte which contains the transformed bytes.
// Returns error when encoding or transformation fails.
func (w *transformerWrapper[K, V]) encodeAndTransform(ctx context.Context, value V) ([]byte, error) {
	encoded, err := w.encodeValue(value)
	if err != nil {
		return nil, fmt.Errorf("failed to encode: %w", err)
	}
	transformed, err := w.transformAndWrap(ctx, encoded)
	if err != nil {
		return nil, fmt.Errorf("failed to transform: %w", err)
	}
	return transformed, nil
}

// computeCallback handles the inner compute callback for the Compute method.
//
// Takes ctx (context.Context) which is propagated for cancellation and timeout.
// Takes computeFunction (func(...)) which transforms the old value
// into a new value with an action indicating what to do.
//
// Returns func([]byte, bool) ([]byte, cache_dto.ComputeAction) which wraps the
// compute function to handle encoding and decoding of byte values.
func (w *transformerWrapper[K, V]) computeCallback(
	ctx context.Context,
	computeFunction func(oldValue V, found bool) (newValue V, action cache_dto.ComputeAction),
) func([]byte, bool) ([]byte, cache_dto.ComputeAction) {
	ctx, l := logger_domain.From(ctx, log)

	return func(oldBytes []byte, found bool) ([]byte, cache_dto.ComputeAction) {
		var oldV V
		if found {
			var err error
			oldV, err = w.unwrapAndDecode(ctx, oldBytes)
			if err != nil {
				l.Error("Compute: failed to process old value", logger_domain.Error(err))
				return nil, cache_dto.ComputeActionNoop
			}
		}

		newV, action := computeFunction(oldV, found)

		if action == cache_dto.ComputeActionSet {
			transformed, err := w.encodeAndTransform(ctx, newV)
			if err != nil {
				l.Error("Compute: failed to process new value", logger_domain.Error(err))
				return nil, cache_dto.ComputeActionNoop
			}
			return transformed, cache_dto.ComputeActionSet
		}

		return nil, action
	}
}

// Compute computes a new value atomically for the given key.
//
// Takes ctx (context.Context) for cancellation and timeout.
// Takes key (K) which identifies the cache entry to compute.
// Takes computeFunction (func(...)) which receives the old value and
// whether it was found, and returns the new value with an action
// indicating what to do.
//
// Returns V which is the final computed value, or zero value on failure.
// Returns bool which indicates whether the computation succeeded.
// Returns error when the underlying provider or transformation fails.
func (w *transformerWrapper[K, V]) Compute(ctx context.Context, key K, computeFunction func(oldValue V, found bool) (newValue V, action cache_dto.ComputeAction)) (V, bool, error) {
	var finalValue V

	byteValue, ok, err := goroutine.SafeCall2(ctx, "cache.Compute", func() ([]byte, bool, error) {
		return w.provider.Compute(ctx, key, w.computeCallback(ctx, computeFunction))
	})
	if err != nil {
		return finalValue, false, fmt.Errorf("computing value in provider: %w", err)
	}

	if !ok || len(byteValue) == 0 {
		return finalValue, ok, nil
	}

	finalValue, err = w.unwrapAndDecode(ctx, byteValue)
	if err != nil {
		return finalValue, false, fmt.Errorf("compute: failed to process final value: %w", err)
	}

	return finalValue, ok, nil
}

// ComputeIfAbsent atomically computes and stores a value if the key is absent.
//
// Takes ctx (context.Context) for cancellation and timeout.
// Takes key (K) which identifies the entry to look up or create.
// Takes computeFunction (func() V) which generates the value if the key is absent.
//
// Returns V which is the existing or newly computed value.
// Returns bool which indicates whether the value was computed (true) or already
// existed (false).
// Returns error when the underlying provider or transformation fails.
func (w *transformerWrapper[K, V]) ComputeIfAbsent(ctx context.Context, key K, computeFunction func() V) (V, bool, error) {
	var finalValue V
	ctx, l := logger_domain.From(ctx, log)

	byteValue, computed, err := goroutine.SafeCall2(ctx, "cache.ComputeIfAbsent", func() ([]byte, bool, error) {
		return w.provider.ComputeIfAbsent(ctx, key, func() []byte {
			newV := computeFunction()

			encoded, err := w.encodeValue(newV)
			if err != nil {
				l.Error("ComputeIfAbsent: failed to encode computed value", logger_domain.Error(err))
				return nil
			}
			transformed, err := w.transformAndWrap(ctx, encoded)
			if err != nil {
				l.Error("ComputeIfAbsent: failed to transform computed value", logger_domain.Error(err))
				return nil
			}
			return transformed
		})
	})
	if err != nil {
		return finalValue, false, fmt.Errorf("computing absent value in provider: %w", err)
	}

	if len(byteValue) == 0 {
		return finalValue, computed, nil
	}

	reversed, err := w.unwrapAndReverse(ctx, byteValue)
	if err != nil {
		return finalValue, false, fmt.Errorf("ComputeIfAbsent: failed to reverse final value: %w", err)
	}
	finalValue, err = w.decodeValue(reversed)
	if err != nil {
		return finalValue, false, fmt.Errorf("ComputeIfAbsent: failed to decode final value: %w", err)
	}

	return finalValue, computed, nil
}

// computeIfPresentCallback creates the inner callback for ComputeIfPresent.
//
// Takes ctx (context.Context) which is propagated for cancellation and timeout.
// Takes computeFunction (func(oldValue V) (newValue V, action
// ComputeAction)) which updates the existing value and returns the
// action to take.
//
// Returns func([]byte) ([]byte, ComputeAction) which wraps the compute
// function with encoding and transformation logic.
func (w *transformerWrapper[K, V]) computeIfPresentCallback(
	ctx context.Context,
	computeFunction func(oldValue V) (newValue V, action cache_dto.ComputeAction),
) func([]byte) ([]byte, cache_dto.ComputeAction) {
	ctx, l := logger_domain.From(ctx, log)

	return func(oldBytes []byte) ([]byte, cache_dto.ComputeAction) {
		oldV, err := w.unwrapAndDecode(ctx, oldBytes)
		if err != nil {
			l.Error("ComputeIfPresent: failed to process old value", logger_domain.Error(err))
			return nil, cache_dto.ComputeActionNoop
		}

		newV, action := computeFunction(oldV)

		if action == cache_dto.ComputeActionSet {
			transformed, err := w.encodeAndTransform(ctx, newV)
			if err != nil {
				l.Error("ComputeIfPresent: failed to process new value", logger_domain.Error(err))
				return nil, cache_dto.ComputeActionNoop
			}
			return transformed, cache_dto.ComputeActionSet
		}

		return nil, action
	}
}

// ComputeIfPresent atomically computes a new value for the key if it is
// present.
//
// Takes ctx (context.Context) for cancellation and timeout.
// Takes key (K) which is the key to look up and potentially update.
// Takes computeFunction (func(...)) which computes the new value
// from the old value.
//
// Returns V which is the final computed value, or the zero value if not found.
// Returns bool which indicates whether the key was present and the computation
// succeeded.
// Returns error when the underlying provider or transformation fails.
func (w *transformerWrapper[K, V]) ComputeIfPresent(ctx context.Context, key K, computeFunction func(oldValue V) (newValue V, action cache_dto.ComputeAction)) (V, bool, error) {
	var finalValue V

	byteValue, ok, err := goroutine.SafeCall2(ctx, "cache.ComputeIfPresent", func() ([]byte, bool, error) {
		return w.provider.ComputeIfPresent(ctx, key, w.computeIfPresentCallback(ctx, computeFunction))
	})
	if err != nil {
		return finalValue, false, fmt.Errorf("computing present value in provider: %w", err)
	}

	if !ok || len(byteValue) == 0 {
		return finalValue, ok, nil
	}

	finalValue, err = w.unwrapAndDecode(ctx, byteValue)
	if err != nil {
		return finalValue, false, fmt.Errorf("ComputeIfPresent: failed to process final value: %w", err)
	}

	return finalValue, ok, nil
}

// ComputeWithTTL atomically computes a new value with per-call TTL control.
//
// Takes ctx (context.Context) for cancellation and timeout.
// Takes key (K) which identifies the cache entry to update.
// Takes computeFunction (func(...)) which receives the old value and found flag,
// returning a ComputeResult containing the new value, action, and optional TTL.
//
// Returns V which is the resulting value after the operation.
// Returns bool which indicates whether a value is now present.
// Returns error when the underlying provider or transformation fails.
func (w *transformerWrapper[K, V]) ComputeWithTTL(ctx context.Context, key K, computeFunction func(oldValue V, found bool) cache_dto.ComputeResult[V]) (V, bool, error) {
	var finalValue V
	ctx, l := logger_domain.From(ctx, log)

	wrappedComputeFunction := func(oldBytes []byte, found bool) cache_dto.ComputeResult[[]byte] {
		var oldValue V
		if found && len(oldBytes) > 0 {
			var err error
			oldValue, err = w.unwrapAndDecode(ctx, oldBytes)
			if err != nil {
				l.Error("ComputeWithTTL: failed to decode old value", logger_domain.Error(err))
				return cache_dto.ComputeResult[[]byte]{Action: cache_dto.ComputeActionNoop}
			}
		}

		result := computeFunction(oldValue, found)

		if result.Action != cache_dto.ComputeActionSet {
			return cache_dto.ComputeResult[[]byte]{
				Action: result.Action,
				TTL:    result.TTL,
			}
		}

		newBytes, err := w.encodeAndTransform(ctx, result.Value)
		if err != nil {
			l.Error("ComputeWithTTL: failed to encode new value", logger_domain.Error(err))
			return cache_dto.ComputeResult[[]byte]{Action: cache_dto.ComputeActionNoop}
		}

		return cache_dto.ComputeResult[[]byte]{
			Value:  newBytes,
			Action: cache_dto.ComputeActionSet,
			TTL:    result.TTL,
		}
	}

	byteValue, ok, err := goroutine.SafeCall2(ctx, "cache.ComputeWithTTL", func() ([]byte, bool, error) { return w.provider.ComputeWithTTL(ctx, key, wrappedComputeFunction) })
	if err != nil {
		return finalValue, false, fmt.Errorf("computing value with TTL in provider: %w", err)
	}

	if !ok || len(byteValue) == 0 {
		return finalValue, ok, nil
	}

	finalValue, err = w.unwrapAndDecode(ctx, byteValue)
	if err != nil {
		return finalValue, false, fmt.Errorf("ComputeWithTTL: failed to process final value: %w", err)
	}

	return finalValue, ok, nil
}

// encodeLoadedValues encodes a map of loaded values to bytes.
//
// Takes loadedValues (map[K]V) which contains the values to encode.
// Takes opName (string) which identifies the operation for log messages.
//
// Returns map[K][]byte which contains the successfully encoded values.
func (w *transformerWrapper[K, V]) encodeLoadedValues(ctx context.Context, loadedValues map[K]V, opName string) map[K][]byte {
	ctx, l := logger_domain.From(ctx, log)

	processed := make(map[K][]byte, len(loadedValues))
	for k, v := range loadedValues {
		transformed, err := w.encodeAndTransform(ctx, v)
		if err != nil {
			l.Warn(opName+": failed to process loaded value",
				logger_domain.String(logKeyKey, fmt.Sprintf(formatAnyValue, k)),
				logger_domain.Error(err))
			continue
		}
		processed[k] = transformed
	}
	return processed
}

// decodeResults converts a map of byte slices into typed values.
//
// Takes byteResults (map[K][]byte) which contains the raw byte data keyed by
// identifier.
// Takes opName (string) which identifies the operation for log messages.
//
// Returns map[K]V which contains the converted values, excluding any that
// failed to process.
func (w *transformerWrapper[K, V]) decodeResults(ctx context.Context, byteResults map[K][]byte, opName string) map[K]V {
	ctx, l := logger_domain.From(ctx, log)

	finalResults := make(map[K]V, len(byteResults))
	for k, byteValue := range byteResults {
		value, err := w.unwrapAndDecode(ctx, byteValue)
		if err != nil {
			l.Warn(opName+": failed to process value",
				logger_domain.String(logKeyKey, fmt.Sprintf(formatAnyValue, k)),
				logger_domain.Error(err))
			continue
		}
		finalResults[k] = value
	}
	return finalResults
}

// BulkGet retrieves multiple values by their keys in a single operation.
//
// Takes keys ([]K) which specifies the keys to look up.
// Takes bulkLoader (BulkLoader[K, V]) which fetches missing keys from the
// source.
//
// Returns map[K]V which contains the values found, mapped by their keys.
// Returns error when the bulk load or value conversion fails.
func (w *transformerWrapper[K, V]) BulkGet(ctx context.Context, keys []K, bulkLoader cache_dto.BulkLoader[K, V]) (map[K]V, error) {
	wrappedBulkLoader := cache_dto.BulkLoaderFunc[K, []byte](func(ctx context.Context, missKeys []K) (map[K][]byte, error) {
		loadedValues, err := bulkLoader.BulkLoad(ctx, missKeys)
		if err != nil {
			return nil, fmt.Errorf("bulk loading values: %w", err)
		}
		return w.encodeLoadedValues(ctx, loadedValues, "BulkGet"), nil
	})

	byteResults, err := goroutine.SafeCall1(ctx, "cache.BulkGet", func() (map[K][]byte, error) { return w.provider.BulkGet(ctx, keys, wrappedBulkLoader) })
	if err != nil {
		return nil, fmt.Errorf("bulk getting from provider: %w", err)
	}

	return w.decodeResults(ctx, byteResults, "BulkGet"), nil
}

// BulkRefresh reloads values for multiple keys in the background.
//
// Takes keys ([]K) which specifies the cache keys to refresh.
// Takes bulkLoader (BulkLoader) which loads values for the given keys.
func (w *transformerWrapper[K, V]) BulkRefresh(ctx context.Context, keys []K, bulkLoader cache_dto.BulkLoader[K, V]) {
	wrappedBulkLoader := cache_dto.BulkLoaderFunc[K, []byte](func(ctx context.Context, missKeys []K) (map[K][]byte, error) {
		loadedValues, err := bulkLoader.BulkLoad(ctx, missKeys)
		if err != nil {
			return nil, fmt.Errorf("bulk loading values for refresh: %w", err)
		}
		return w.encodeLoadedValues(ctx, loadedValues, "BulkRefresh"), nil
	})

	w.provider.BulkRefresh(ctx, keys, wrappedBulkLoader)
}

// processRefreshResult processes a single byte result from Refresh and sends
// the result to a channel.
//
// Takes byteResult (cache_dto.LoadResult[[]byte]) which contains the raw bytes
// to process.
// Takes resultChan (chan<- cache_dto.LoadResult[V]) which receives the
// transformed result.
func (w *transformerWrapper[K, V]) processRefreshResult(ctx context.Context, byteResult cache_dto.LoadResult[[]byte], resultChan chan<- cache_dto.LoadResult[V]) {
	if byteResult.Err != nil {
		resultChan <- cache_dto.LoadResult[V]{Err: byteResult.Err}
		return
	}

	value, err := w.unwrapAndDecode(ctx, byteResult.Value)
	if err != nil {
		resultChan <- cache_dto.LoadResult[V]{Err: err}
		return
	}
	resultChan <- cache_dto.LoadResult[V]{Value: value}
}

// Refresh asynchronously reloads the value for a key.
//
// Takes ctx (context.Context) which controls cancellation and deadlines.
// Takes key (K) which identifies the cache entry to refresh.
// Takes loader (cache_dto.Loader[K, V]) which provides the refreshed value.
//
// Returns <-chan cache_dto.LoadResult[V] which delivers the refresh result
// asynchronously.
//
// Safe for concurrent use. Spawns a goroutine that reads from
// the provider's result channel and writes the decoded result.
func (w *transformerWrapper[K, V]) Refresh(ctx context.Context, key K, loader cache_dto.Loader[K, V]) <-chan cache_dto.LoadResult[V] {
	wrappedLoader := cache_dto.LoaderFunc[K, []byte](func(ctx context.Context, key K) ([]byte, error) {
		value, err := loader.Load(ctx, key)
		if err != nil {
			return nil, fmt.Errorf("loading value for refresh: %w", err)
		}
		return w.encodeAndTransform(ctx, value)
	})

	byteChan := w.provider.Refresh(ctx, key, wrappedLoader)
	resultChan := make(chan cache_dto.LoadResult[V], 1)

	go func() {
		defer close(resultChan)
		defer goroutine.RecoverPanic(ctx, "cache.transformerRefresh")
		for byteResult := range byteChan {
			w.processRefreshResult(ctx, byteResult, resultChan)
			return
		}
	}()

	return resultChan
}

// All returns an iterator over all key-value pairs.
//
// Returns iter.Seq2[K, V] which yields each key-value pair after reversing
// transformations and decoding.
func (w *transformerWrapper[K, V]) All() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		ctx, l := logger_domain.From(context.Background(), log)
		for k, byteValue := range w.provider.All() {
			reversed, err := w.unwrapAndReverse(ctx, byteValue)
			if err != nil {
				l.Warn("All: failed to reverse transform value",
					logger_domain.String(logKeyKey, fmt.Sprintf(formatAnyValue, k)),
					logger_domain.Error(err))
				continue
			}
			value, err := w.decodeValue(reversed)
			if err != nil {
				l.Warn("All: failed to decode value",
					logger_domain.String(logKeyKey, fmt.Sprintf(formatAnyValue, k)),
					logger_domain.Error(err))
				continue
			}
			if !yield(k, value) {
				return
			}
		}
	}
}

// Keys returns an iterator over all keys (pass-through safe).
//
// Returns iter.Seq[K] which yields each key in the cache.
func (w *transformerWrapper[K, V]) Keys() iter.Seq[K] {
	return w.provider.Keys()
}

// Values returns an iterator over all values.
//
// Returns iter.Seq[V] which yields each decoded value in the cache.
func (w *transformerWrapper[K, V]) Values() iter.Seq[V] {
	return func(yield func(V) bool) {
		ctx, l := logger_domain.From(context.Background(), log)
		for byteValue := range w.provider.Values() {
			reversed, err := w.unwrapAndReverse(ctx, byteValue)
			if err != nil {
				l.Warn("Values: failed to reverse transform value", logger_domain.Error(err))
				continue
			}
			value, err := w.decodeValue(reversed)
			if err != nil {
				l.Warn("Values: failed to decode value", logger_domain.Error(err))
				continue
			}
			if !yield(value) {
				return
			}
		}
	}
}

// GetEntry returns a snapshot of the entry.
//
// Takes ctx (context.Context) for cancellation and timeout.
// Takes key (K) which identifies the cache entry to retrieve.
//
// Returns cache_dto.Entry[K, V] which is the decoded entry snapshot.
// Returns bool which indicates whether the key was found.
// Returns error when the underlying provider or transformation fails.
func (w *transformerWrapper[K, V]) GetEntry(ctx context.Context, key K) (cache_dto.Entry[K, V], bool, error) {
	byteEntry, ok, err := goroutine.SafeCall2(ctx, "cache.GetEntry", func() (cache_dto.Entry[K, []byte], bool, error) { return w.provider.GetEntry(ctx, key) })
	if err != nil {
		return cache_dto.Entry[K, V]{}, false, fmt.Errorf("getting entry from provider: %w", err)
	}
	if !ok {
		return cache_dto.Entry[K, V]{}, false, nil
	}
	entry, ok, err := w.processEntry(ctx, byteEntry, "GetEntry")
	return entry, ok, err
}

// ProbeEntry returns a snapshot without affecting access patterns.
//
// Takes ctx (context.Context) for cancellation and timeout.
// Takes key (K) which identifies the cache entry to probe.
//
// Returns cache_dto.Entry[K, V] which is the decoded entry snapshot.
// Returns bool which indicates whether the key was found.
// Returns error when the underlying provider or transformation fails.
func (w *transformerWrapper[K, V]) ProbeEntry(ctx context.Context, key K) (cache_dto.Entry[K, V], bool, error) {
	byteEntry, ok, err := goroutine.SafeCall2(ctx, "cache.ProbeEntry", func() (cache_dto.Entry[K, []byte], bool, error) { return w.provider.ProbeEntry(ctx, key) })
	if err != nil {
		return cache_dto.Entry[K, V]{}, false, fmt.Errorf("probing entry from provider: %w", err)
	}
	if !ok {
		return cache_dto.Entry[K, V]{}, false, nil
	}
	entry, ok, err := w.processEntry(ctx, byteEntry, "ProbeEntry")
	return entry, ok, err
}

// processEntry unwraps transformations and decodes a byte-level cache
// entry back into a typed Entry[K, V]. Returns an error if the reverse
// transform or decode fails.
//
// Takes ctx (context.Context) which is propagated for cancellation and timeout.
// Takes byteEntry (cache_dto.Entry[K, []byte]) which is the raw
// byte-level entry to decode.
// Takes methodName (string) which identifies the calling method for
// error logging.
//
// Returns cache_dto.Entry[K, V] which is the decoded typed entry.
// Returns bool which is false if decoding fails.
// Returns error when unwrapping or decoding fails.
func (w *transformerWrapper[K, V]) processEntry(ctx context.Context, byteEntry cache_dto.Entry[K, []byte], methodName string) (cache_dto.Entry[K, V], bool, error) {
	reversed, err := w.unwrapAndReverse(ctx, byteEntry.Value)
	if err != nil {
		return cache_dto.Entry[K, V]{}, false, fmt.Errorf("%s: failed to reverse transform value: %w", methodName, err)
	}
	value, err := w.decodeValue(reversed)
	if err != nil {
		return cache_dto.Entry[K, V]{}, false, fmt.Errorf("%s: failed to decode value: %w", methodName, err)
	}

	return cache_dto.Entry[K, V]{
		Key:               byteEntry.Key,
		Value:             value,
		Weight:            byteEntry.Weight,
		ExpiresAtNano:     byteEntry.ExpiresAtNano,
		RefreshableAtNano: byteEntry.RefreshableAtNano,
		SnapshotAtNano:    byteEntry.SnapshotAtNano,
	}, true, nil
}

// EstimatedSize returns the approximate number of entries (pass-through safe).
//
// Returns int which is the estimated number of entries in the cache.
func (w *transformerWrapper[K, V]) EstimatedSize() int {
	return goroutine.SafeCallValue(context.Background(), "cache.EstimatedSize", func() int { return w.provider.EstimatedSize() })
}

// Stats returns cache performance statistics.
//
// Returns cache_dto.Stats which contains the current cache metrics.
func (w *transformerWrapper[K, V]) Stats() cache_dto.Stats {
	return goroutine.SafeCallValue(context.Background(), "cache.Stats", func() cache_dto.Stats { return w.provider.Stats() })
}

// Close releases all resources held by the transformer.
//
// Takes ctx (context.Context) for cancellation and timeout.
//
// Returns error when the underlying provider fails to close.
func (w *transformerWrapper[K, V]) Close(ctx context.Context) error {
	return goroutine.SafeCall(ctx, "cache.Close", func() error { return w.provider.Close(ctx) })
}

// GetMaximum returns the current maximum capacity (pass-through safe).
//
// Returns uint64 which is the maximum capacity from the underlying provider.
func (w *transformerWrapper[K, V]) GetMaximum() uint64 {
	return goroutine.SafeCallValue(context.Background(), "cache.GetMaximum", func() uint64 { return w.provider.GetMaximum() })
}

// SetMaximum changes the maximum capacity (pass-through safe).
//
// Takes size (uint64) which specifies the new maximum capacity.
func (w *transformerWrapper[K, V]) SetMaximum(size uint64) {
	goroutine.SafeCallValue(context.Background(), "cache.SetMaximum", func() struct{} { w.provider.SetMaximum(size); return struct{}{} })
}

// WeightedSize returns the current total weight.
//
// Returns uint64 which is the total weighted size of all cached entries.
func (w *transformerWrapper[K, V]) WeightedSize() uint64 {
	return goroutine.SafeCallValue(context.Background(), "cache.WeightedSize", func() uint64 { return w.provider.WeightedSize() })
}

// SetExpiresAfter manually sets the expiration for a key (pass-through safe).
//
// Takes ctx (context.Context) for cancellation and timeout.
// Takes key (K) which identifies the cache entry to update.
// Takes expiresAfter (time.Duration) which specifies the expiration duration.
//
// Returns error when the underlying provider fails.
func (w *transformerWrapper[K, V]) SetExpiresAfter(ctx context.Context, key K, expiresAfter time.Duration) error {
	return goroutine.SafeCall(ctx, "cache.SetExpiresAfter", func() error { return w.provider.SetExpiresAfter(ctx, key, expiresAfter) })
}

// SetRefreshableAfter manually sets the refresh time for a key.
//
// Pass-through safe.
//
// Takes ctx (context.Context) for cancellation and timeout.
// Takes key (K) which identifies the cache entry to update.
// Takes refreshableAfter (time.Duration) which specifies when the entry
// becomes eligible for refresh.
//
// Returns error when the underlying provider fails.
func (w *transformerWrapper[K, V]) SetRefreshableAfter(ctx context.Context, key K, refreshableAfter time.Duration) error {
	return goroutine.SafeCall(ctx, "cache.SetRefreshableAfter", func() error { return w.provider.SetRefreshableAfter(ctx, key, refreshableAfter) })
}

// Search returns ErrSearchNotSupported because the transformer wrapper
// stores values as encoded bytes, not searchable structures.
//
// Takes ctx (context.Context) for cancellation.
// Takes query (string) which is the search query (ignored).
// Takes opts (*cache_dto.SearchOptions) which are the search options (ignored).
//
// Returns empty SearchResult and ErrSearchNotSupported.
func (*transformerWrapper[K, V]) Search(_ context.Context, _ string, _ *cache_dto.SearchOptions) (cache_dto.SearchResult[K, V], error) {
	return cache_dto.SearchResult[K, V]{}, ErrSearchNotSupported
}

// Query returns ErrSearchNotSupported because the transformer wrapper
// stores values as encoded bytes, not queryable structures.
//
// Takes ctx (context.Context) for cancellation.
// Takes opts (*cache_dto.QueryOptions) which are the query options (ignored).
//
// Returns empty SearchResult and ErrSearchNotSupported.
func (*transformerWrapper[K, V]) Query(_ context.Context, _ *cache_dto.QueryOptions) (cache_dto.SearchResult[K, V], error) {
	return cache_dto.SearchResult[K, V]{}, ErrSearchNotSupported
}

// SupportsSearch returns false because transformer-wrapped caches cannot
// support search operations on encoded/transformed byte values.
//
// Returns bool which is always false for transformer wrappers.
func (*transformerWrapper[K, V]) SupportsSearch() bool {
	return false
}

// GetSchema returns nil because transformer-wrapped caches do not support
// search schemas.
//
// Returns *cache_dto.SearchSchema which is always nil for transformer wrappers.
func (*transformerWrapper[K, V]) GetSchema() *cache_dto.SearchSchema {
	return nil
}

// withWrapperEncoder provides a custom encoding implementation for the wrapper.
// If not used, the wrapper will default to JSON encoding for backward
// compatibility.
//
// Takes s (EncoderPort[V]) which is the encoder to use for value
// serialisation.
//
// Returns wrapperOption[K, V] which configures the wrapper's encoder.
func withWrapperEncoder[K comparable, V any](s EncoderPort[V]) wrapperOption[K, V] {
	return func(c *wrapperConfig[K, V]) {
		c.encoder = s
	}
}

// newTransformerWrapper creates a new transformer wrapper for a cache provider.
// The wrapper transforms the cache from Cache[K,V] to Cache[K,[]byte]
// internally, handling encoding and transformation transparently.
//
// By default, JSON encoding is used. To use a different encoder, pass the
// withWrapperEncoder option:
// wrapper := newTransformerWrapper(provider, registry, config,
//
//	withWrapperEncoder(myCustomEncoder))
//
// Takes provider (ProviderPort[K, []byte]) which is the underlying byte-level
// cache.
// Takes registry (*TransformerRegistry) which holds the available
// transformers.
// Takes config (*cache_dto.TransformConfig) which specifies which
// transformers to apply.
// Takes opts (...wrapperOption[K, V]) which provides optional configuration
// such as a custom encoder.
//
// Returns *transformerWrapper[K, V] which wraps the provider with encoding
// and transformation.
func newTransformerWrapper[K comparable, V any](
	provider ProviderPort[K, []byte],
	registry *TransformerRegistry,
	config *cache_dto.TransformConfig,
	opts ...wrapperOption[K, V],
) *transformerWrapper[K, V] {
	wrapperConfig := &wrapperConfig[K, V]{}

	for _, opt := range opts {
		opt(wrapperConfig)
	}

	return &transformerWrapper[K, V]{
		provider: provider,
		registry: registry,
		config:   config,
		encoder:  wrapperConfig.encoder,
	}
}
