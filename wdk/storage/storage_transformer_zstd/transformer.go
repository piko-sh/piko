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

package storage_transformer_zstd

import (
	"context"
	"fmt"
	"io"

	"github.com/klauspost/compress/zstd"
	"piko.sh/piko/internal/contextaware"
	"piko.sh/piko/wdk/logger"
	"piko.sh/piko/wdk/storage"
)

const (
	// defaultPriorityCompression is the default priority
	// for compression transformers.
	defaultPriorityCompression = 100
)

var _ io.ReadCloser = (*decoderReadCloser)(nil)

// ZstdTransformer implements StreamTransformerPort for Zstandard compression.
// Zstandard offers good compression with fast decompression, well suited
// for storage.
type ZstdTransformer struct {
	// name is the unique identifier for this transformer.
	name string

	// priority is the execution order for this
	// transformer; lower values run first.
	priority int

	// level specifies the zstd compression level; 0 uses the default.
	level zstd.EncoderLevel
}

var _ storage.StreamTransformerPort = (*ZstdTransformer)(nil)

// Config holds settings for the zstd transformer.
type Config struct {
	// Name is the identifier for this transformer instance. Default: "zstd".
	Name string

	// Priority determines execution order; lower values run first on writes.
	// Recommended range is 100-199 for compression transformers; defaults to 100.
	Priority int

	// Level sets the compression level, ranging from
	// SpeedFastest (1) through SpeedDefault (3) and
	// SpeedBetterCompression (5) to SpeedBestCompression (11),
	// defaulting to SpeedDefault (3).
	Level zstd.EncoderLevel
}

// NewZstdTransformer creates a new zstd compression transformer.
//
// Takes config (Config) which sets the transformer name, priority, and
// compression level. Missing values use sensible defaults.
//
// Returns *ZstdTransformer which is the configured transformer ready for use.
// Returns error when the configuration is not valid.
func NewZstdTransformer(config Config) (*ZstdTransformer, error) {
	if config.Name == "" {
		config.Name = "zstd"
	}
	if config.Priority == 0 {
		config.Priority = defaultPriorityCompression
	}
	if config.Level == 0 {
		config.Level = zstd.SpeedDefault
	}

	return &ZstdTransformer{
		name:     config.Name,
		priority: config.Priority,
		level:    config.Level,
	}, nil
}

// Name returns the unique identifier for this transformer.
//
// Returns string which is the transformer's unique name.
func (z *ZstdTransformer) Name() string {
	return z.name
}

// Type returns the transformer type (compression).
//
// Returns storage.TransformerType which identifies this as a compression
// transformer.
func (*ZstdTransformer) Type() storage.TransformerType {
	return storage.TransformerCompression
}

// Priority returns the execution priority for this transformer.
//
// Returns int which is the priority value; lower values run first.
func (z *ZstdTransformer) Priority() int {
	return z.priority
}

// Transform compresses the input stream using zstd.
//
// Takes input (io.Reader) which provides the data to compress.
// Takes options (any) which can optionally override the default compression
// level as map[string]any{"level": int}.
//
// Returns io.Reader which provides compressed data as the input is read.
// Returns error when the transformer cannot be initialised.
//
// Safe for concurrent use. Spawns a goroutine that compresses data into
// a pipe; errors propagate through the returned reader.
func (z *ZstdTransformer) Transform(ctx context.Context, input io.Reader, options any) (io.Reader, error) {
	ctx, l := logger.From(ctx, log)

	level := z.level

	if opts, ok := options.(map[string]any); ok {
		if lvl, exists := opts["level"]; exists {
			if levelInt, ok := lvl.(int); ok {
				level = zstd.EncoderLevel(levelInt)
			}
		}
	}

	l.Trace("Applying zstd compression",
		logger.String("transformer", z.name),
		logger.Int("level", int(level)))

	pr, pw := io.Pipe()

	go func() {
		defer func() { _ = pw.Close() }()

		encoder, err := zstd.NewWriter(pw, zstd.WithEncoderLevel(level))
		if err != nil {
			_ = pw.CloseWithError(fmt.Errorf("failed to create zstd encoder: %w", err))
			return
		}
		defer func() { _ = encoder.Close() }()

		if _, err := io.Copy(encoder, contextaware.NewReader(ctx, input)); err != nil {
			_ = pw.CloseWithError(fmt.Errorf("zstd compression error: %w", err))
			return
		}

		if err := encoder.Close(); err != nil {
			_ = pw.CloseWithError(fmt.Errorf("zstd encoder close error: %w", err))
			return
		}
	}()

	return pr, nil
}

// Reverse decompresses the input stream using zstd.
//
// Takes input (io.Reader) which provides the compressed data to decompress.
//
// Returns io.Reader which provides decompressed data as the input is read.
// Returns error when the decoder cannot be initialised.
func (z *ZstdTransformer) Reverse(ctx context.Context, input io.Reader, _ any) (io.Reader, error) {
	_, l := logger.From(ctx, log)

	l.Trace("Reversing zstd compression (decompressing)",
		logger.String("transformer", z.name))

	decoder, err := zstd.NewReader(input)
	if err != nil {
		return nil, fmt.Errorf("failed to create zstd decoder: %w", err)
	}

	return &decoderReadCloser{
		reader:  decoder,
		decoder: decoder,
	}, nil
}

// decoderReadCloser wraps a zstd decoder to ensure proper cleanup when closed.
// It implements io.ReadCloser.
type decoderReadCloser struct {
	// reader holds the zstd decoder for reading decompressed data.
	reader io.Reader

	// decoder is the zstd decompressor for reading compressed data.
	decoder *zstd.Decoder
}

// Read reads decompressed data from the zstd decoder.
//
// Takes p ([]byte) which is the buffer to read decompressed data into.
//
// Returns n (int) which is the number of bytes read.
// Returns err (error) when the read fails or the stream ends.
func (d *decoderReadCloser) Read(p []byte) (n int, err error) {
	return d.reader.Read(p)
}

// Close closes the zstd decoder and releases resources.
//
// Returns error which is always nil.
func (d *decoderReadCloser) Close() error {
	d.decoder.Close()
	return nil
}

// DefaultConfig returns sensible defaults for zstd compression.
//
// Returns Config which contains the default zstd compression settings.
func DefaultConfig() Config {
	return Config{
		Name:     "zstd",
		Priority: defaultPriorityCompression,
		Level:    zstd.SpeedDefault,
	}
}
