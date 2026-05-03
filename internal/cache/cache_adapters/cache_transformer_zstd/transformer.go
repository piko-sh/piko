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

package cache_transformer_zstd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/klauspost/compress/zstd"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/logger/logger_domain"
)

const (
	// defaultTransformerName is the name used for the zstd transformer when no
	// custom name is set.
	defaultTransformerName = "zstd"

	// defaultPriority is the default execution priority for compression transformers.
	// Recommended range: 100-199 for compression transformers.
	defaultPriority = 100

	// logKeyTransformer is the standard log key for transformer name.
	logKeyTransformer = "transformer"

	// DefaultMaxDecompressedCacheBytes is the default cap on bytes that may be
	// produced by a single Reverse decompression.
	//
	// Cache values are typically small in-memory blobs, so a 64 MiB ceiling
	// is a generous default that still prevents pathological decompression
	// bombs from dominating the heap in callers that buffer the result.
	// Override with WithMaxDecompressedCacheBytes for stricter or more
	// relaxed limits.
	DefaultMaxDecompressedCacheBytes int64 = 64 * 1024 * 1024
)

// ErrDecompressedCacheTooLarge is returned by Reverse when the decompressed
// payload exceeds the configured maximum decompressed size. Callers can use
// errors.Is to distinguish this from ordinary decompression failures.
var ErrDecompressedCacheTooLarge = errors.New("cache_transformer_zstd: decompressed cache value exceeds maximum allowed size")

// ZstdCacheTransformer implements CacheTransformerPort for Zstandard
// compression. Zstandard provides excellent compression ratios with fast
// decompression speeds, well suited for cache value compression.
type ZstdCacheTransformer struct {
	// encoder compresses data using the zstd algorithm.
	encoder *zstd.Encoder

	// decoder decompresses zstd-encoded data.
	decoder *zstd.Decoder

	// name is the unique identifier for this transformer.
	name string

	// level is the default compression level used when options are missing or invalid.
	level zstd.EncoderLevel

	// priority is the execution order for this transformer; lower values run first.
	priority int

	// maxDecompressedBytes caps the bytes returned from Reverse, preventing
	// decompression bombs from exhausting memory in downstream consumers. A
	// non-positive value disables the cap.
	maxDecompressedBytes int64

	// closeOnce guards the encoder/decoder release sequence so Close is safe
	// to invoke repeatedly.
	closeOnce sync.Once
}

var _ cache_domain.CacheTransformerPort = (*ZstdCacheTransformer)(nil)

// Config holds configuration for the zstd cache transformer.
type Config struct {
	// Name is the unique identifier for this transformer instance. Defaults to
	// "zstd" if not set.
	Name string

	// Level sets the compression level from SpeedFastest (1) through
	// SpeedDefault (3), SpeedBetterCompression (5), to
	// SpeedBestCompression (11), defaulting to SpeedDefault (3).
	Level zstd.EncoderLevel

	// Priority determines execution order; lower values run first on Set.
	// Recommended range is 100-199 for compression transformers; default is 100.
	Priority int

	// MaxDecompressedBytes caps the decompressed output size in bytes
	// produced by Reverse.
	//
	// When zero, DefaultMaxDecompressedCacheBytes is used. Negative values
	// disable the cap (not recommended for untrusted cache contents).
	MaxDecompressedBytes int64
}

// Option configures a ZstdCacheTransformer at construction time.
type Option func(*ZstdCacheTransformer)

// WithMaxDecompressedCacheBytes sets the maximum number of decompressed
// bytes produced by Reverse before ErrDecompressedCacheTooLarge is surfaced.
//
// Pass a non-positive value to disable the cap (only safe for fully trusted
// cache contents).
//
// Takes maxBytes (int64) which is the cap in bytes; non-positive disables.
//
// Returns Option which sets the cap on a transformer.
func WithMaxDecompressedCacheBytes(maxBytes int64) Option {
	return func(t *ZstdCacheTransformer) {
		t.maxDecompressedBytes = maxBytes
	}
}

// NewZstdCacheTransformer creates a new zstd cache compression transformer.
//
// Takes config (Config) which specifies the compression settings including
// name, priority, compression level, and decompression cap.
// Takes options (...Option) which override settings on the constructed
// transformer (e.g. WithMaxDecompressedCacheBytes).
//
// Returns *ZstdCacheTransformer which is ready to compress and decompress
// cached data.
// Returns error when the zstd encoder or decoder cannot be created.
func NewZstdCacheTransformer(config Config, options ...Option) (*ZstdCacheTransformer, error) {
	if config.Name == "" {
		config.Name = defaultTransformerName
	}
	if config.Priority == 0 {
		config.Priority = defaultPriority
	}
	if config.Level == 0 {
		config.Level = zstd.SpeedDefault
	}
	if config.MaxDecompressedBytes == 0 {
		config.MaxDecompressedBytes = DefaultMaxDecompressedCacheBytes
	}

	encoder, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(config.Level))
	if err != nil {
		return nil, fmt.Errorf("failed to create zstd encoder: %w", err)
	}

	decoder, err := zstd.NewReader(nil)
	if err != nil {
		_ = encoder.Close()
		return nil, fmt.Errorf("failed to create zstd decoder: %w", err)
	}

	t := &ZstdCacheTransformer{
		name:                 config.Name,
		priority:             config.Priority,
		level:                config.Level,
		encoder:              encoder,
		decoder:              decoder,
		maxDecompressedBytes: config.MaxDecompressedBytes,
	}

	for _, opt := range options {
		opt(t)
	}

	return t, nil
}

// Name returns the unique identifier for this transformer.
//
// Returns string which is the transformer's unique identifier.
func (z *ZstdCacheTransformer) Name() string {
	return z.name
}

// Type returns the transformer type for this cache transformer.
//
// Returns cache_dto.TransformerType which indicates compression.
func (*ZstdCacheTransformer) Type() cache_dto.TransformerType {
	return cache_dto.TransformerCompression
}

// Priority returns the execution priority for this transformer.
//
// Returns int which is the priority level for ordering transformer execution.
func (z *ZstdCacheTransformer) Priority() int {
	return z.priority
}

// Transform compresses the input bytes using zstd.
//
// Options can optionally override the default compression level
// via map[string]any with a "level" key.
//
// Takes input ([]byte) which contains the data to compress.
// Takes options (any) which may specify a custom compression
// level.
//
// Returns []byte which contains the zstd-compressed data.
// Returns error when compression fails.
func (z *ZstdCacheTransformer) Transform(ctx context.Context, input []byte, options any) ([]byte, error) {
	_, l := logger_domain.From(ctx, log)

	level := z.getCompressionLevel(options)

	l.Trace("Compressing cache value with zstd",
		logger_domain.String(logKeyTransformer, z.name),
		logger_domain.Int("level", int(level)),
		logger_domain.Int("input_size", len(input)))

	compressed := z.encoder.EncodeAll(input, make([]byte, 0, len(input)))

	l.Trace("Zstd compression complete",
		logger_domain.String(logKeyTransformer, z.name),
		logger_domain.Int("input_size", len(input)),
		logger_domain.Int("compressed_size", len(compressed)),
		logger_domain.Float64("compression_ratio", float64(len(input))/float64(len(compressed))))

	return compressed, nil
}

// Reverse decompresses the input bytes using zstd.
//
// The decompressed output is capped at the configured maximum (see
// WithMaxDecompressedCacheBytes); attempts to decompress beyond the cap
// surface ErrDecompressedCacheTooLarge instead of allowing unbounded
// allocations. A non-positive cap disables the limit.
//
// Options are not used for decompression.
//
// Takes input ([]byte) which contains the compressed data to
// decompress.
//
// Returns []byte which contains the decompressed data.
// Returns error when decompression fails or the configured cap is exceeded.
func (z *ZstdCacheTransformer) Reverse(ctx context.Context, input []byte, _ any) ([]byte, error) {
	_, l := logger_domain.From(ctx, log)

	l.Trace("Decompressing cache value with zstd",
		logger_domain.String(logKeyTransformer, z.name),
		logger_domain.Int("compressed_size", len(input)),
		logger_domain.Int64("max_decompressed_bytes", z.maxDecompressedBytes))

	decompressed, err := z.decompress(input)
	if err != nil {
		return nil, err
	}

	l.Trace("Zstd decompression complete",
		logger_domain.String(logKeyTransformer, z.name),
		logger_domain.Int("compressed_size", len(input)),
		logger_domain.Int("decompressed_size", len(decompressed)))

	return decompressed, nil
}

// Close releases the encoder and decoder held by the transformer. It is safe
// to call Close repeatedly; subsequent invocations are no-ops.
//
// Returns error which is always nil; included so the transformer can satisfy
// io.Closer for resource-aware callers.
func (z *ZstdCacheTransformer) Close() error {
	z.closeOnce.Do(func() {
		if z.encoder != nil {
			_ = z.encoder.Close()
		}
		if z.decoder != nil {
			z.decoder.Close()
		}
	})
	return nil
}

// decompress runs the bounded zstd decompression. When the cap is enabled, a
// streaming decoder is used so that decompression terminates as soon as the
// limit is exceeded; the LimitReader+1 idiom distinguishes "exactly at cap"
// from "over cap" without buffering the whole result first.
//
// Takes input ([]byte) which contains the compressed payload.
//
// Returns []byte which is the decompressed payload.
// Returns error when decompression fails or the cap is exceeded.
func (z *ZstdCacheTransformer) decompress(input []byte) ([]byte, error) {
	if z.maxDecompressedBytes <= 0 {
		decompressed, err := z.decoder.DecodeAll(input, nil)
		if err != nil {
			return nil, fmt.Errorf("zstd decompression failed: %w", err)
		}
		return decompressed, nil
	}

	reader, err := zstd.NewReader(bytes.NewReader(input))
	if err != nil {
		return nil, fmt.Errorf("failed to create zstd decoder: %w", err)
	}
	defer reader.Close()

	limited := io.LimitReader(reader, z.maxDecompressedBytes+1)
	decompressed, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("zstd decompression failed: %w", err)
	}

	if int64(len(decompressed)) > z.maxDecompressedBytes {
		return nil, fmt.Errorf("%w: decompressed at least %d bytes, cap %d",
			ErrDecompressedCacheTooLarge, len(decompressed), z.maxDecompressedBytes)
	}

	return decompressed, nil
}

// getCompressionLevel returns the compression level from the given options,
// falling back to the transformer's default level.
//
// Takes options (any) which may be a map containing a "level" key.
//
// Returns zstd.EncoderLevel which is the compression level to use.
func (z *ZstdCacheTransformer) getCompressionLevel(options any) zstd.EncoderLevel {
	opts, ok := options.(map[string]any)
	if !ok {
		return z.level
	}

	lvl, exists := opts["level"]
	if !exists {
		return z.level
	}

	levelInt, ok := lvl.(int)
	if !ok {
		return z.level
	}

	return zstd.EncoderLevel(levelInt)
}

// DefaultConfig returns sensible defaults for zstd compression.
//
// Returns Config which contains the default transformer name, priority,
// compression level, and decompression cap.
func DefaultConfig() Config {
	return Config{
		Name:                 defaultTransformerName,
		Priority:             defaultPriority,
		Level:                zstd.SpeedDefault,
		MaxDecompressedBytes: DefaultMaxDecompressedCacheBytes,
	}
}

// createTransformerFromConfig creates a zstd transformer from a config value.
//
// Takes config (any) which specifies the transformer settings, or nil for
// defaults.
//
// Returns cache_domain.CacheTransformerPort which is the configured zstd
// transformer.
// Returns error when the transformer cannot be created.
func createTransformerFromConfig(config any) (cache_domain.CacheTransformerPort, error) {
	zstdConfig := DefaultConfig()

	if config != nil {
		zstdConfig = parseConfig(config, zstdConfig)
	}

	return NewZstdCacheTransformer(zstdConfig)
}

// parseConfig parses a config value into a Config struct.
//
// Takes config (any) which is the raw configuration to parse.
// Takes defaults (Config) which provides fallback values when parsing fails.
//
// Returns Config which is the parsed configuration, or defaults if the config
// type is not recognised.
func parseConfig(config any, defaults Config) Config {
	switch c := config.(type) {
	case Config:
		return c
	case map[string]any:
		return parseMapConfig(c, defaults)
	default:
		return defaults
	}
}

// parseMapConfig extracts Config fields from a map.
//
// Takes c (map[string]any) which contains the configuration key-value pairs.
// Takes config (Config) which provides the base configuration to update.
//
// Returns Config which is the updated configuration with extracted values.
func parseMapConfig(c map[string]any, config Config) Config {
	if name, ok := c["name"].(string); ok {
		config.Name = name
	}
	if priority, ok := c["priority"].(int); ok {
		config.Priority = priority
	}
	if level, ok := c["level"].(int); ok {
		config.Level = zstd.EncoderLevel(level)
	}
	if maxBytes, ok := c["max_decompressed_bytes"].(int64); ok {
		config.MaxDecompressedBytes = maxBytes
	} else if maxBytes, ok := c["max_decompressed_bytes"].(int); ok {
		config.MaxDecompressedBytes = int64(maxBytes)
	}
	return config
}

func init() {
	cache_domain.RegisterTransformerBlueprint(defaultTransformerName, createTransformerFromConfig)
}
