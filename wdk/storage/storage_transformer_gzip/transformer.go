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

package storage_transformer_gzip

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/klauspost/compress/gzip"
	"piko.sh/piko/internal/contextaware"
	"piko.sh/piko/wdk/logger"
	"piko.sh/piko/wdk/storage"
)

const (
	// defaultPriorityCompression is the default priority for compression transformers.
	defaultPriorityCompression = 100

	// DefaultMaxDecompressedBytes is the default cap on bytes returned from a
	// decompression stream.
	//
	// Set high (256 MiB) so legitimate workloads are unaffected while still
	// preventing pathological decompression bombs from dominating memory in
	// callers that buffer the stream. Override with WithMaxDecompressedBytes
	// for stricter or more relaxed limits.
	DefaultMaxDecompressedBytes int64 = 256 * 1024 * 1024
)

// ErrDecompressedTooLarge is returned by the reader produced by Reverse when
// the decompressed payload exceeds the configured maximum decompressed size.
// Callers can use errors.Is to distinguish this from a normal io.EOF.
var ErrDecompressedTooLarge = errors.New("storage_transformer_gzip: decompressed payload exceeds maximum allowed size")

var _ io.ReadCloser = (*readerCloser)(nil)

// GzipTransformer implements StreamTransformerPort for gzip compression.
// Gzip offers good compression with wide compatibility, suited for storage
// when interoperability matters.
type GzipTransformer struct {
	// name is the unique identifier for this transformer.
	name string

	// priority is the order in which this transformer runs; lower values run first.
	priority int

	// level is the gzip compression level; 0 uses the default.
	level int

	// maxDecompressedBytes caps the bytes returned from Reverse, preventing
	// decompression bombs from exhausting memory in downstream consumers.
	maxDecompressedBytes int64
}

var _ storage.StreamTransformerPort = (*GzipTransformer)(nil)

// Config holds settings for the gzip transformer.
type Config struct {
	// Name is the identifier for this transformer instance. Defaults to "gzip" if
	// not set.
	Name string

	// Priority determines execution order; lower values run first on writes.
	// Recommended range is 100-199 for compression transformers; default is 100.
	Priority int

	// Level sets the gzip compression level.
	// Defaults to gzip.DefaultCompression when set to zero.
	Level int

	// MaxDecompressedBytes caps the decompressed output size in bytes.
	//
	// When zero, DefaultMaxDecompressedBytes is used. Negative values disable
	// the cap (not recommended for untrusted input).
	MaxDecompressedBytes int64
}

// Option configures a GzipTransformer at construction time.
type Option func(*GzipTransformer)

// WithMaxDecompressedBytes sets the maximum number of decompressed bytes that
// can flow through the reader returned by Reverse.
//
// Reads beyond this cap surface ErrDecompressedTooLarge. Pass a non-positive
// value to disable the cap (only safe for fully trusted input streams).
//
// Takes maxBytes (int64) which is the cap in bytes; non-positive disables.
//
// Returns Option which sets the cap on a transformer.
func WithMaxDecompressedBytes(maxBytes int64) Option {
	return func(t *GzipTransformer) {
		t.maxDecompressedBytes = maxBytes
	}
}

// NewGzipTransformer creates a new gzip compression transformer.
//
// Takes config (Config) which sets the transformer options including name,
// priority, and compression level. Missing or zero values use defaults.
// Takes options (...Option) which override settings on the constructed
// transformer (e.g. WithMaxDecompressedBytes).
//
// Returns *GzipTransformer which is the configured transformer ready for use.
// Returns error when the compression level is outside the valid range.
func NewGzipTransformer(config Config, options ...Option) (*GzipTransformer, error) {
	if config.Name == "" {
		config.Name = "gzip"
	}
	if config.Priority == 0 {
		config.Priority = defaultPriorityCompression
	}
	if config.Level == 0 {
		config.Level = gzip.DefaultCompression
	}
	if config.MaxDecompressedBytes == 0 {
		config.MaxDecompressedBytes = DefaultMaxDecompressedBytes
	}

	if config.Level < gzip.NoCompression || config.Level > gzip.BestCompression {
		if config.Level != gzip.DefaultCompression {
			return nil, fmt.Errorf("invalid gzip compression level %d: must be between %d and %d (or %d for default)",
				config.Level, gzip.NoCompression, gzip.BestCompression, gzip.DefaultCompression)
		}
	}

	t := &GzipTransformer{
		name:                 config.Name,
		priority:             config.Priority,
		level:                config.Level,
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
func (g *GzipTransformer) Name() string {
	return g.name
}

// Type returns the transformer type for this compressor.
//
// Returns storage.TransformerType which identifies this as a compression
// transformer.
func (*GzipTransformer) Type() storage.TransformerType {
	return storage.TransformerCompression
}

// Priority returns the execution priority for this transformer.
//
// Returns int which indicates when this transformer runs relative to others.
func (g *GzipTransformer) Priority() int {
	return g.priority
}

// Transform compresses the input stream using gzip.
// It returns a reader that provides compressed data as the input is read.
//
// Takes input (io.Reader) which provides the data to compress.
// Takes options (any) which can optionally override the default compression
// level as map[string]any{"level": int}.
//
// Returns io.Reader which yields compressed data as it is read.
// Returns error when the gzip writer cannot be created.
//
// Spawns a goroutine that performs compression in the background. The
// goroutine runs until the input is fully read, an error occurs, or the
// context is cancelled. Errors during compression are propagated through the
// returned reader.
func (g *GzipTransformer) Transform(ctx context.Context, input io.Reader, options any) (io.Reader, error) {
	ctx, l := logger.From(ctx, log)

	level := g.level

	if opts, ok := options.(map[string]any); ok {
		if lvl, exists := opts["level"]; exists {
			if levelInt, ok := lvl.(int); ok {
				level = levelInt
			}
		}
	}

	l.Trace("Applying gzip compression",
		logger.String("transformer", g.name),
		logger.Int("level", level))

	pr, pw := io.Pipe()

	go func() {
		defer func() { _ = pw.Close() }()

		writer, err := gzip.NewWriterLevel(pw, level)
		if err != nil {
			_ = pw.CloseWithError(fmt.Errorf("failed to create gzip writer: %w", err))
			return
		}
		defer func() { _ = writer.Close() }()

		if _, err := io.Copy(writer, contextaware.NewReader(ctx, input)); err != nil {
			_ = pw.CloseWithError(fmt.Errorf("gzip compression error: %w", err))
			return
		}

		if err := writer.Close(); err != nil {
			_ = pw.CloseWithError(fmt.Errorf("gzip writer close error: %w", err))
			return
		}
	}()

	return pr, nil
}

// Reverse decompresses the input stream using gzip.
//
// It returns a reader that provides decompressed data as the input is read.
// The returned reader caps the decompressed bytes at the configured maximum
// (see WithMaxDecompressedBytes); reading beyond the cap yields
// ErrDecompressedTooLarge.
//
// Takes input (io.Reader) which provides the compressed data to decompress.
//
// Returns io.Reader which provides decompressed data as the input is read.
// Returns error when the gzip reader cannot be created.
func (g *GzipTransformer) Reverse(ctx context.Context, input io.Reader, _ any) (io.Reader, error) {
	_, l := logger.From(ctx, log)

	l.Trace("Reversing gzip compression (decompressing)",
		logger.String("transformer", g.name),
		logger.Int64("maxDecompressedBytes", g.maxDecompressedBytes))

	reader, err := gzip.NewReader(input)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}

	return newCappedReader(reader, g.maxDecompressedBytes), nil
}

// newCappedReader wraps a gzip reader so that reads beyond maxBytes surface
// ErrDecompressedTooLarge instead of allowing unbounded decompressed output.
// When maxBytes is non-positive, the cap is disabled and the reader behaves
// transparently.
//
// Takes reader (*gzip.Reader) which produces decompressed bytes.
// Takes maxBytes (int64) which caps the byte count; non-positive disables the
// cap.
//
// Returns *readerCloser which wraps the reader with the configured cap.
func newCappedReader(reader *gzip.Reader, maxBytes int64) *readerCloser {
	rc := &readerCloser{
		reader:   reader,
		source:   reader,
		maxBytes: maxBytes,
	}
	if maxBytes > 0 {
		rc.source = io.LimitReader(reader, maxBytes+1)
	}
	return rc
}

// readerCloser wraps a gzip reader to ensure proper cleanup.
//
// It enforces a configurable cap on the total decompressed bytes returned to
// callers, so a malicious payload cannot inflate to terabytes via a small
// upload. It implements io.ReadCloser.
type readerCloser struct {
	// reader decompresses gzip-compressed data.
	reader *gzip.Reader

	// source is the bounded byte source actually read by callers; it is
	// either the gzip reader directly (when no cap is set) or a LimitReader
	// wrapping it.
	source io.Reader

	// readBytes is the running count of decompressed bytes returned. The
	// running count is used to detect when the cap has been hit so the
	// sentinel ErrDecompressedTooLarge can be surfaced to the caller.
	readBytes int64

	// maxBytes is the cap on decompressed bytes; non-positive disables the
	// cap entirely.
	maxBytes int64
}

// Read reads decompressed data from the gzip reader, enforcing the configured
// maximum decompressed byte limit. When the limit is reached, the read returns
// ErrDecompressedTooLarge.
//
// Takes p ([]byte) which is the buffer to read decompressed data into.
//
// Returns n (int) which is the number of bytes read.
// Returns err (error) when the read fails, the stream ends, or the cap is hit.
func (r *readerCloser) Read(p []byte) (n int, err error) {
	n, err = r.source.Read(p)
	r.readBytes += int64(n)
	if r.maxBytes > 0 && r.readBytes > r.maxBytes {
		return n, fmt.Errorf("%w: decompressed at least %d bytes, cap %d",
			ErrDecompressedTooLarge, r.readBytes, r.maxBytes)
	}
	return n, err
}

// Close closes the gzip reader and releases resources.
//
// Returns error when the underlying reader fails to close.
func (r *readerCloser) Close() error {
	return r.reader.Close()
}

// DefaultConfig returns sensible default settings for gzip compression.
//
// Returns Config which contains the default compression settings ready for use.
func DefaultConfig() Config {
	return Config{
		Name:                 "gzip",
		Priority:             defaultPriorityCompression,
		Level:                gzip.DefaultCompression,
		MaxDecompressedBytes: DefaultMaxDecompressedBytes,
	}
}
